package DiemBFT

import (
	"bytes"
	"encoding/hex"
	"strconv"

	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/sign"
)

type BlockNode struct {
	block    Block
	children []string // children block ids
}

func maxQc(qc, highCommitQc *QC) *QC {
	if highCommitQc == nil {
		return qc
	}
	if qc.voteInfo.round > highCommitQc.voteInfo.round {
		return qc
	} else {
		return highCommitQc
	}
}

func (v *Validator) processQc(qc *QC) {
	if qc == nil {
		return
	}
	v.logger.Trace("In processQC")
	_, committed := v.commits[qc.voteInfo.parentId]
	if qc.ledgerCommitInfo.commitStateId != "" && !committed {
		v.commit(qc.voteInfo.parentId)
		// TODO: pendingBlocktreePrune(qc.voteInfo.parentId)
		v.highCommitQc = maxQc(qc, v.highCommitQc)
	}
	v.highQc = maxQc(qc, v.highQc)
}

func (v *Validator) executeAndInsert(b Block) {
	v.logger.Trace("In executeAndInsert")
	v.speculate(b.qc.voteInfo.id, b.id, b.payload)
	v.pendingBlockTree[b.id] = BlockNode{block: b}
	if node, ok := v.pendingBlockTree[b.qc.voteInfo.id]; ok {
		node.children = append(node.children, b.id)
		v.pendingBlockTree[b.qc.voteInfo.id] = node
	}
}

func (v *Validator) processVote(vm VoteMessage) *QC {
	v.logger.Trace("In processVote")
	v.processQc(vm.highCommitQc)
	voteIdxHash := nacl.Hash([]byte(vm.ledgerCommitInfo.commitStateId + "-" + vm.ledgerCommitInfo.voteInfoHash))
	voteIdx := hex.EncodeToString((*voteIdxHash)[:])
	v.pendingVotes[voteIdx] = append(v.pendingVotes[voteIdx], PendingVote{vm.sender, vm.signature})
	if len(v.pendingVotes[voteIdx]) == 2*v.f+1 {
		v.logger.Trace("Making QC")
		var signaturebytes [][]byte
		var signatures []Signature
		for _, vote := range v.pendingVotes[voteIdx] {
			signaturebytes = append(signaturebytes, vote.signature)
			signatures = append(signatures, Signature{signer: vote.sender, signature: vote.signature})
		}
		return &QC{
			voteInfo:         vm.voteInfo,
			ledgerCommitInfo: vm.ledgerCommitInfo,
			signatures:       signatures,
			author:           v.validatorId,
			authorSignature:  sign.Sign(bytes.Join(signaturebytes, []byte("-")), v.privateKey),
		}
	}
	return nil
}

func (v *Validator) generateBlock(txns string, currentRound int) Block {
	v.logger.Trace("In generateBlock")
	var toHash [][]byte
	// no signatures in genesis QC
	if v.highQc.voteInfo.round != 0 {
		for _, vote := range v.highQc.signatures {
			toHash = append(toHash, vote.signature)
		}
	}
	toHash = append(toHash, []byte(v.validatorId+"-"+strconv.Itoa(v.currentRound)+"-"+txns+"-"+v.highQc.voteInfo.id+"-"))

	blockIdHash := nacl.Hash(bytes.Join(toHash, []byte("-")))
	blockId := hex.EncodeToString((*blockIdHash)[:])
	return Block{
		author:  v.validatorId,
		round:   v.currentRound,
		payload: txns,
		qc:      v.highQc,
		id:      blockId,
	}
}
