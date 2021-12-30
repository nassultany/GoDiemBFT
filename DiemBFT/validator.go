package DiemBFT

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"sync"

	//"log"
	"strconv"
	"time"

	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/sign"
	pb "github.com/nassultany/GoDiemBFT/protos"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"
)

// maintains state for the replica, created during node setup
type Validator struct {
	// validator's unique identifier
	validatorId string

	// all validator Ids
	validatorsIds []string

	// number of faulty replicas tolerated
	f int

	// maps from peer validator ids to connection
	peerValidators map[string]pb.DiemRPCClient

	// RPC chan comes from the gRPC server, type is message type
	rpcCh <-chan interface{}

	// for round timer and timeouts
	timeoutCh   chan int
	stopTimerCh chan int
	wgRound     sync.WaitGroup

	// for logging
	logger *log.Logger

	/************ BLOCKTREE state variables **************/
	/*****************************************************/
	highCommitQc     *QC
	highQc           *QC
	pendingBlockTree map[string]BlockNode
	pendingVotes     map[string][]PendingVote
	genesisBlock     Block

	/********** LEADERELECTION state variables ***********/
	/*****************************************************/
	windowSize        int
	excludeSize       int
	reputationLeaders map[int]string

	/************ PACEMAKER state variables **************/
	/*****************************************************/
	currentRound    int
	lastRoundTc     *TC
	pendingTimeouts map[int][]TimeoutInfo
	delta           time.Duration // used for timeout calculation

	/*************** SAFETY state variables **************/
	/*****************************************************/
	privateKey       []byte
	publicKeys       map[string][]byte
	highestVoteRound int
	highestQcRound   int

	/*************** Ledger state variables **************/
	/*****************************************************/
	// Ledger is meant to represent the state machine your
	// program is meant to replicate. For this example, it is
	// a database represented simply as a text file
	ledger *os.File
	// channel where committed transactions are received and processed
	commitCh chan string
	// stores execStateId with index blockId
	stateId map[string]string
	// stores committed blocks (as of now stored in memory, but can be persisted in a file)
	commits map[string]Block
}

func NewValidator(validatorId string, f int, validators map[string]string, rpcCh chan interface{}, privateKey []byte, publicKeys map[string][]byte, logLevel log.Level) Validator {
	peerValidators := make(map[string]pb.DiemRPCClient)
	for validator, addr := range validators {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			log.Fatalln(err)
		}
		client := pb.NewDiemRPCClient(conn)
		peerValidators[validator] = client
	}

	validatorIds := make([]string, len(publicKeys))
	i := 0
	for id := range publicKeys {
		validatorIds[i] = id
		i++
	}
	// sort them alphabetically so we can deterministically choose leader in round robin scenario
	sort.Slice(validatorIds, func(i, j int) bool {
		return validatorIds[i] < validatorIds[j]
	})

	genesisBlockId := hex.EncodeToString((*nacl.Hash([]byte("genesis")))[:])
	genesisBlock := Block{id: genesisBlockId}
	pendingBlockTree := make(map[string]BlockNode)
	pendingBlockTree[genesisBlockId] = BlockNode{block: genesisBlock}

	// setup ledger for SMR
	ledger, err := os.Create(validatorId + ".ledger")
	if err != nil {
		log.Fatalln(err)
	}
	ledgerFile, err := os.OpenFile(ledger.Name(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend)
	if err != nil {
		log.Fatalln(err)
	}

	// for logging
	logger := log.New()
	logger.SetLevel(logLevel)
	logFile, err := os.Create(validatorId + ".log")
	if err != nil {
		log.Fatalln(err)
	}
	file, err := os.OpenFile(logFile.Name(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend)
	if err != nil {
		log.Fatalln(err)
	}
	logger.SetOutput(file)

	return Validator{
		validatorId:    validatorId,
		validatorsIds:  validatorIds,
		f:              f,
		peerValidators: peerValidators,
		rpcCh:          rpcCh,
		stopTimerCh:    make(chan int, 1),
		wgRound:        sync.WaitGroup{},
		logger:         logger,
		// BLOCKTREE
		genesisBlock:     genesisBlock,
		pendingBlockTree: pendingBlockTree,
		//highQc:       &genesisQc,
		pendingVotes: make(map[string][]PendingVote),
		// LEADERELECTION
		windowSize:        f,
		excludeSize:       f + 1,
		reputationLeaders: make(map[int]string),
		// PACEMAKER
		currentRound: 0,
		delta:        time.Second,
		// SAFETY
		privateKey:       privateKey,
		publicKeys:       publicKeys,
		highestVoteRound: 0,
		highestQcRound:   0,
		// Ledger
		ledger:   ledgerFile,
		commitCh: make(chan string, 1),
		stateId:  make(map[string]string),
		commits:  make(map[string]Block),
	}
}

func (v *Validator) ProcessCertificateQc(qc *QC) {
	if qc == nil {
		v.logger.Trace("qc is nil")
		return
	}
	v.processQc(qc)
	v.updateLeaders(*qc)
	v.advanceRoundQc(qc)
}

func (v *Validator) ProcessProposalMsg(P ProposalMessage) {
	v.logger.Info(fmt.Sprintf("Processing proposal message: %+v", P))
	v.ProcessCertificateQc(P.block.qc)
	v.ProcessCertificateQc(P.highCommitQc)
	v.advanceRoundTc(P.lastRoundTc)
	currentRound := v.currentRound
	leader := v.getLeader(currentRound)
	if (P.block.round != currentRound) || (P.block.author != leader) {
		v.logger.Info("Dropping proposal message due to inconsistent metadata")
		v.logger.Info("		P.block.round: ", P.block.round, "Current Round: ", currentRound, "P.block.author: ", P.block.author, "Leader: ", leader)
		return
	}
	v.executeAndInsert(P.block)
	voteMsg := v.makeVote(P.block, P.lastRoundTc)
	if voteMsg != nil {
		nextLeader := v.getLeader(currentRound + 1)
		//v.logger.Trace(fmt.Sprintf("Sending vote message for round %d to %s: %+v", currentRound+1, nextLeader, voteMsg))
		v.logger.Trace(fmt.Sprintf("Sending vote message to %s: %+v", nextLeader, voteMsg))
		go func(vm *VoteMessage, nxtLeader string) {
			_, err := v.peerValidators[nxtLeader].VoteMsg(context.Background(), vm.toProto())
			if err != nil {
				fmt.Println(err)
			}
		}(voteMsg, nextLeader)
	}
}

func (v *Validator) ProcessTimeoutMsg(M TimeoutMessage) {
	v.logger.Trace("Processing timeout message: ", M)
	v.ProcessCertificateQc(M.tmoInfo.highQc)
	v.processQc(M.highCommitQc)
	v.advanceRoundTc(M.lastRoundTC)
	tc := v.processRemoteTimeout(M)
	if tc != nil {
		v.advanceRoundTc(tc)
		v.processNewRoundEvent(tc)
	}
}

func (v *Validator) ProcessVoteMsg(M VoteMessage) {
	qc := v.processVote(M)
	if qc != nil {
		v.ProcessCertificateQc(qc)
		v.processNewRoundEvent(nil)
	}
}

func (v *Validator) processNewRoundEvent(lastTc *TC) {
	v.logger.Trace("In process new round event")
	leader := v.getLeader(v.currentRound)
	v.logger.Trace("The leader of round ", v.currentRound, " is ", leader)
	if leader == v.validatorId {
		v.logger.Trace(fmt.Sprintf("I, %s, am the leader", v.validatorId))

		// simulate that the leader has made the QC for round 0 and is now sending message for round 1
		if v.currentRound == 0 {
			genesisQc := QC{voteInfo: VoteInfo{id: v.genesisBlock.id, round: 0}}
			v.highQc = &genesisQc
			//v.currentRound = 1
			v.startTimer(1)
		}

		b := v.generateBlock(v.validatorId+" "+strconv.Itoa(v.currentRound), v.currentRound)
		// broadcast proposalMessage
		proposalMessage := ProposalMessage{block: b, lastRoundTc: lastTc, highCommitQc: v.highCommitQc, signature: sign.Sign([]byte(b.id), v.privateKey)}
		proposalMessageProto := proposalMessage.toProto()

		// sleep just so output doesn't get out of control
		time.Sleep(500 * time.Millisecond)

		v.logger.Info(fmt.Sprintf("Sending proposal message: %+v", proposalMessage))
		for _, conn := range v.peerValidators {
			go func(pm *pb.ProposalMessage, connection pb.DiemRPCClient) {
				_, err := connection.ProposalMsg(context.Background(), pm)
				if err != nil {
					fmt.Println(err)
				}
			}(proposalMessageProto, conn)
		}
	}
}

func (v *Validator) Run() {
	v.logger.Info("STARTING LOG")
	v.logger.Info("My id: ", v.validatorId)
	v.logger.Info("Other validators: ", v.peerValidators)
	v.processNewRoundEvent(nil)
	go func(commitCh <-chan string) {
		for {
			txn := <-commitCh
			// apply transaction/payload to your state machine
			v.ledger.Write([]byte(txn + "\n"))
		}
	}(v.commitCh)
	for {
		select {
		case rpc := <-v.rpcCh:
			v.logger.Trace("Got rpc")
			switch cmd := rpc.(type) {
			case ProposalMessage:
				v.ProcessProposalMsg(cmd)
			case TimeoutMessage:
				v.ProcessTimeoutMsg(cmd)
			case VoteMessage:
				v.ProcessVoteMsg(cmd)
			}
		case tmo := <-v.timeoutCh:
			if tmo == v.currentRound {
				fmt.Printf("Timing out in round %d\n", tmo)
				v.localTimeoutRound()
			}
		}
	}
}
