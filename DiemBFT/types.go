package DiemBFT

import pb "github.com/nassultany/GoDiemBFT/protos"

type LedgerCommitInfo struct {
	commitStateId string
	voteInfoHash  string
}

type VoteInfo struct {
	id          string
	round       int
	parentId    string
	parentRound int
	execStateId string
}

type VoteMessage struct {
	voteInfo         VoteInfo
	ledgerCommitInfo LedgerCommitInfo
	highCommitQc     *QC
	sender           string
	signature        []byte
}

type QC struct {
	voteInfo         VoteInfo
	ledgerCommitInfo LedgerCommitInfo
	signatures       []Signature
	author           string
	authorSignature  []byte
}

type TC struct {
	round           int
	tmoHighQcRounds []int32
	tmoSignatures   [][]byte
}

type TimeoutInfo struct {
	round     int
	highQc    *QC
	sender    string
	signature []byte
}

type TimeoutMessage struct {
	tmoInfo      TimeoutInfo
	lastRoundTC  *TC
	highCommitQc *QC
}

type ProposalMessage struct {
	block        Block
	lastRoundTc  *TC
	highCommitQc *QC
	signature    []byte
}

type Block struct {
	author  string
	round   int
	payload string
	qc      *QC
	id      string
}

type PendingVote struct {
	sender    string
	signature []byte
}

type Signature struct {
	signer    string
	signature []byte
}

func (l *LedgerCommitInfo) toProto() *pb.LedgerCommitInfo {
	return &pb.LedgerCommitInfo{
		CommitStateId: l.commitStateId,
		VoteInfoHash:  l.voteInfoHash,
	}
}

func LedgerCommitInfoToStruct(lcm *pb.LedgerCommitInfo) LedgerCommitInfo {
	return LedgerCommitInfo{
		commitStateId: lcm.CommitStateId,
		voteInfoHash:  lcm.VoteInfoHash,
	}
}

func (v *VoteInfo) toProto() *pb.VoteInfo {
	return &pb.VoteInfo{
		Id:          v.id,
		Round:       int32(v.round),
		ParentId:    v.parentId,
		ParentRound: int32(v.parentRound),
		ExecStateId: v.execStateId,
	}
}

func VoteInfoToStruct(v *pb.VoteInfo) VoteInfo {
	return VoteInfo{
		id:          v.Id,
		round:       int(v.Round),
		parentId:    v.ParentId,
		parentRound: int(v.ParentRound),
		execStateId: v.ExecStateId,
	}
}

func (qc *QC) getSignaturesProto() []*pb.Signature {
	var signatures []*pb.Signature
	for _, signature := range qc.signatures {
		signatures = append(signatures, &pb.Signature{
			Signer:    signature.signer,
			Signature: signature.signature,
		})
	}
	return signatures
}

func (b *Block) toProto() *pb.Block {
	return &pb.Block{
		Author:  b.author,
		Round:   int32(b.round),
		Payload: b.payload,
		Qc:      b.qc.toProto(),
		Id:      b.id,
	}
}

func BlockToStruct(b *pb.Block) Block {
	return Block{
		id:      b.Id,
		round:   int(b.Round),
		payload: b.Payload,
		qc:      QcToStruct(b.Qc),
		author:  b.Author,
	}
}

func (qc *QC) toProto() *pb.QC {
	return &pb.QC{
		VoteInfo:         qc.voteInfo.toProto(),
		LedgerCommitInfo: qc.ledgerCommitInfo.toProto(),
		Signatures:       qc.getSignaturesProto(),
		Author:           qc.author,
		AuthorSignature:  qc.authorSignature,
	}
}

func SignaturesToStruct(s []*pb.Signature) []Signature {
	var result []Signature
	for _, sig := range s {
		result = append(result, Signature{signer: sig.Signer, signature: sig.Signature})
	}
	return result
}

func QcToStruct(qc *pb.QC) *QC {
	if qc == nil {
		return nil
	}
	return &QC{
		voteInfo:         VoteInfoToStruct(qc.VoteInfo),
		ledgerCommitInfo: LedgerCommitInfoToStruct(qc.LedgerCommitInfo),
		signatures:       SignaturesToStruct(qc.Signatures),
		author:           qc.Author,
		authorSignature:  qc.AuthorSignature,
	}
}

func (tc *TC) toProto() *pb.TC {
	return &pb.TC{
		Round:           int32(tc.round),
		TmoHighQcRounds: tc.tmoHighQcRounds,
		TmoSignatures:   tc.tmoSignatures,
	}
}

func TcToStruct(tc *pb.TC) *TC {
	if tc == nil {
		return nil
	}
	return &TC{
		round:           int(tc.Round),
		tmoHighQcRounds: tc.TmoHighQcRounds,
		tmoSignatures:   tc.TmoSignatures,
	}
}

func (tmo *TimeoutInfo) toProto() *pb.TimeoutInfo {
	return &pb.TimeoutInfo{
		Round:     int32(tmo.round),
		HighQc:    tmo.highQc.toProto(),
		Sender:    tmo.sender,
		Signature: tmo.signature,
	}
}

func TmoInfoToStruct(tmoInfo *pb.TimeoutInfo) TimeoutInfo {
	return TimeoutInfo{
		round:     int(tmoInfo.Round),
		highQc:    QcToStruct(tmoInfo.HighQc),
		sender:    tmoInfo.Sender,
		signature: tmoInfo.Signature,
	}
}

func (tm *TimeoutMessage) toProto() *pb.TimeoutMessage {
	return &pb.TimeoutMessage{
		TmoInfo:      tm.tmoInfo.toProto(),
		LastRoundTc:  tm.lastRoundTC.toProto(),
		HighCommitQc: tm.highCommitQc.toProto(),
	}
}

func TimeoutMessageToStruct(tm *pb.TimeoutMessage) TimeoutMessage {
	return TimeoutMessage{
		tmoInfo:      TmoInfoToStruct(tm.TmoInfo),
		lastRoundTC:  TcToStruct(tm.LastRoundTc),
		highCommitQc: QcToStruct(tm.HighCommitQc),
	}
}

func (vm *VoteMessage) toProto() *pb.VoteMessage {
	var messageHighCommitQc *pb.QC = nil
	if vm.highCommitQc != nil {
		messageHighCommitQc = vm.highCommitQc.toProto()
	}

	return &pb.VoteMessage{
		VoteInfo:         vm.voteInfo.toProto(),
		LedgerCommitInfo: vm.ledgerCommitInfo.toProto(),
		HighCommitQc:     messageHighCommitQc,
		Sender:           vm.sender,
		Signature:        vm.signature,
	}
}

func VoteMessageToStruct(vm *pb.VoteMessage) VoteMessage {
	highCommitQc := QcToStruct(vm.HighCommitQc)
	return VoteMessage{
		voteInfo:         VoteInfoToStruct(vm.VoteInfo),
		ledgerCommitInfo: LedgerCommitInfoToStruct(vm.LedgerCommitInfo),
		highCommitQc:     highCommitQc,
		sender:           vm.Sender,
		signature:        vm.Signature,
	}
}

func (pm *ProposalMessage) toProto() *pb.ProposalMessage {
	var lastRoundTc *pb.TC = nil
	var highCommitQc *pb.QC = nil
	if pm.lastRoundTc != nil {
		lastRoundTc = pm.lastRoundTc.toProto()
	}
	if pm.highCommitQc != nil {
		highCommitQc = pm.highCommitQc.toProto()
	}

	return &pb.ProposalMessage{
		Block:        pm.block.toProto(),
		LastRoundTc:  lastRoundTc,
		HighCommitQc: highCommitQc,
		Signature:    pm.signature,
	}
}

func ProposalMessageToStruct(pm *pb.ProposalMessage) ProposalMessage {
	return ProposalMessage{
		block:        BlockToStruct(pm.Block),
		lastRoundTc:  TcToStruct(pm.LastRoundTc),
		highCommitQc: QcToStruct(pm.HighCommitQc),
		signature:    pm.Signature,
	}
}
