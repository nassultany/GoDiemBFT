package DiemBFT

import (
	"math/rand"
)

func (v *Validator) electReputationLeader(qc QC) string {
	v.logger.Trace("In electReputationLeader")
	var activeValidators []string
	var lastAuthors []string // ordered set
	currentQc := qc
	for i := 0; i < v.windowSize || len(lastAuthors) < v.excludeSize; i++ {
		currentBlock := v.committedBlock(currentQc.voteInfo.parentId)
		blockAuthor := currentBlock.author
		if i < v.windowSize {
			for _, signature := range currentQc.signatures {
				activeValidators = append(activeValidators, signature.signer)
			}
		}
		if len(lastAuthors) < v.excludeSize {
			lastAuthors = append(lastAuthors, blockAuthor)
		}
		currentQc = *currentBlock.qc
	}
	for _, author := range lastAuthors {
		for i, activeValidator := range activeValidators {
			if author == activeValidator {
				activeValidators = append(activeValidators[:i], activeValidators[i+1:]...)
				break
			}
		}
	}
	rand.Seed(int64(qc.voteInfo.round))
	index := rand.Intn(len(activeValidators))
	return activeValidators[index]
}

func (v *Validator) updateLeaders(qc QC) {
	currentRound := v.currentRound
	// if round is 4 or less, don't call electReputationLeaders as you don't have enough block history
	// to generate a leader
	if currentRound <= 4 {
		return
	}
	extendedRound := qc.voteInfo.parentRound
	qcRound := qc.voteInfo.round
	if (extendedRound+1 == qcRound) && (qcRound+1 == currentRound) {
		v.reputationLeaders[currentRound+1] = v.electReputationLeader(qc)
	}
}

func (v *Validator) getLeader(round int) string {
	if leader, ok := v.reputationLeaders[round]; ok {
		v.logger.Trace("Getting leader from reputationLeaders")
		return leader
	}
	v.logger.Trace("Getting leader from Round Robin.")
	return v.validatorsIds[(round/2)%len(v.validatorsIds)]
}
