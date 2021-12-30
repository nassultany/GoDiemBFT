package DiemBFT

import (
	"encoding/hex"
	"strconv"

	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/sign"
)

func max(l, r int) int {
	if l > r {
		return l
	} else {
		return r
	}
}

func maxTCRound(tc *TC) int {
	maxVal := -1
	for _, rnd := range tc.tmoHighQcRounds {
		maxVal = max(maxVal, int(rnd))
	}
	return maxVal
}

func (v *Validator) increaseHighestVoteRound(round int) {
	v.highestVoteRound = max(round, v.highestVoteRound)
}

func (v *Validator) updateHighestQcRound(qcRound int) {
	v.highestQcRound = max(qcRound, v.highestQcRound)
}

func consecutive(blockRound, round int) bool {
	return round+1 == blockRound
}

func safeToExtend(blockRound, qcRound int, tc *TC) bool {
	if tc == nil {
		return false
	}
	tcMaxRound := maxTCRound(tc)
	if tcMaxRound == -1 {
		return false
	}
	return consecutive(blockRound, tc.round) && qcRound >= tcMaxRound
}

func (v *Validator) safeToVote(blockRound, qcRound int, tc *TC) bool {
	if blockRound <= max(v.highestVoteRound, qcRound) {
		return false
	}
	return consecutive(blockRound, qcRound) || safeToExtend(blockRound, qcRound, tc)
}

func (v *Validator) safeToTimeout(round, qcRound int, tc TC) bool {
	if qcRound < v.highestQcRound || round <= max(v.highestVoteRound-1, qcRound) {
		return false
	}
	return consecutive(round, qcRound) || consecutive(round, tc.round)
}

func (v *Validator) commitStateIdCandidate(blockRound int, qc QC) string {
	if consecutive(blockRound, qc.voteInfo.round) {
		return v.pendingState(qc.voteInfo.id)
	}
	return ""
}

func (v *Validator) makeVote(b Block, lastTc *TC) *VoteMessage {
	v.logger.Trace("In makeVote")
	qcRound := b.qc.voteInfo.round
	// TODO: if valid_signatures(b, lastTc) && safeToVote(b.round, qcRound, lastTc)
	if v.safeToVote(b.round, qcRound, lastTc) {
		v.updateHighestQcRound(qcRound)
		v.increaseHighestVoteRound(b.round)
		voteInfo := VoteInfo{
			id:          b.id,
			round:       b.round,
			parentId:    b.qc.voteInfo.id,
			parentRound: b.qc.voteInfo.round,
			execStateId: v.pendingState(b.id),
		}
		voteInfoHashBytes := nacl.Hash([]byte(voteInfo.id + "-" + strconv.Itoa(voteInfo.round) + "-" + voteInfo.parentId + "-" + strconv.Itoa(voteInfo.parentRound) + "-" + voteInfo.execStateId))
		voteInfoHash := hex.EncodeToString((*voteInfoHashBytes)[:])
		ledgerCommitInfo := LedgerCommitInfo{
			commitStateId: v.commitStateIdCandidate(b.round, *b.qc),
			voteInfoHash:  voteInfoHash,
		}
		return &VoteMessage{
			voteInfo:         voteInfo,
			ledgerCommitInfo: ledgerCommitInfo,
			highCommitQc:     v.highCommitQc,
			sender:           v.validatorId,
			signature:        sign.Sign([]byte(ledgerCommitInfo.commitStateId+ledgerCommitInfo.voteInfoHash), v.privateKey),
		}
	}
	return nil
}

func (v *Validator) makeTimeout(round int, highQc QC, lastTc TC) *TimeoutInfo {
	qcRound := highQc.voteInfo.round
	// TODO: if valid_signatures(high_qc, last_tc) && safeToTimeout(round, qcRound, LastTc)
	if v.safeToTimeout(round, qcRound, lastTc) {
		v.increaseHighestVoteRound(round)
		return &TimeoutInfo{round: round, highQc: &highQc}
	}
	return nil
}
