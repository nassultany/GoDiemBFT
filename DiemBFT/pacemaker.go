package DiemBFT

import (
	"context"
	"fmt"
	"time"

	pb "github.com/nassultany/GoDiemBFT/protos"
)

func (v *Validator) getRoundTimer(round int) time.Duration {
	return 4 * v.delta
}

func (v *Validator) stopTimer(round int) {
	// round 0 doesn't have a timer
	if v.currentRound == 0 {
		return
	}
	v.stopTimerCh <- round
}

func (v *Validator) startTimer(newRound int) {
	v.stopTimer(v.currentRound)
	v.currentRound = newRound
	timerDuration := v.getRoundTimer(v.currentRound)
	v.wgRound.Wait()
	v.logger.Info("Starting round ", newRound)
	v.wgRound.Add(1)
	go func(round int, duration time.Duration, timeoutCh chan int, stopTimerCh chan int) {
		defer v.wgRound.Done()
		for {
			select {
			case <-time.After(duration):
				v.logger.Info("Timing out in round ", round)
				timeoutCh <- round
				return
			case stopRound := <-stopTimerCh:
				if stopRound == round {
					v.logger.Info("Stopping round ", round)
					return
				}
				v.logger.Error(fmt.Sprintf("Stopping round %d with stop round %d\n", round, stopRound))
			}
		}
	}(v.currentRound, timerDuration, v.timeoutCh, v.stopTimerCh)
}

func (v *Validator) localTimeoutRound() {
	tmoInfo := v.makeTimeout(v.currentRound, *v.highQc, *v.lastRoundTc)
	if tmoInfo == nil {
		v.logger.Error("Error making timeout message")
		return
	}
	timeoutMessage := TimeoutMessage{tmoInfo: *tmoInfo, lastRoundTC: v.lastRoundTc, highCommitQc: v.highCommitQc}
	timeoutMessageProto := timeoutMessage.toProto()

	// broadcast timeoutMessage
	for _, conn := range v.peerValidators {
		go func(timeoutMessage *pb.TimeoutMessage, connection pb.DiemRPCClient) {
			_, err := connection.TimeoutMsg(context.Background(), timeoutMessage)
			if err != nil {
				fmt.Println(err)
			}
		}(timeoutMessageProto, conn)
	}
}

func (v *Validator) senderInPendingTimeouts(round int, sender string) bool {
	if tmoInfos, ok := v.pendingTimeouts[round]; ok {
		for _, tmoInfo := range tmoInfos {
			if sender == tmoInfo.sender {
				return true
			}
		}
	}
	return false
}

func (v *Validator) processRemoteTimeout(tmo TimeoutMessage) *TC {
	tmoInfo := tmo.tmoInfo
	if tmoInfo.round < v.currentRound {
		return nil
	}

	if !v.senderInPendingTimeouts(tmoInfo.round, tmoInfo.sender) {
		v.pendingTimeouts[tmoInfo.round] = append(v.pendingTimeouts[tmoInfo.round], tmoInfo)
	}

	if len(v.pendingTimeouts[tmoInfo.round]) == v.f+1 {
		v.stopTimer(v.currentRound)
		v.localTimeoutRound()
	}

	if len(v.pendingTimeouts[tmoInfo.round]) == 2*v.f+1 {
		var highQcRounds []int32
		var signatures [][]byte
		for _, tmoInfo := range v.pendingTimeouts[tmoInfo.round] {
			highQcRounds = append(highQcRounds, int32(tmoInfo.round))
			signatures = append(signatures, tmoInfo.signature)
		}
		return &TC{
			round:           tmoInfo.round,
			tmoHighQcRounds: highQcRounds,
			tmoSignatures:   signatures,
		}
	}
	return nil
}

func (v *Validator) advanceRoundTc(tc *TC) bool {
	if tc == nil {
		return false
	}
	if tc.round < v.currentRound {
		return false
	}
	v.lastRoundTc = tc
	v.startTimer(tc.round + 1)
	return true
}

func (v *Validator) advanceRoundQc(qc *QC) bool {
	v.logger.Trace("In advanceRoundQc")
	if qc.voteInfo.round < v.currentRound {
		return false
	}
	v.lastRoundTc = nil
	v.startTimer(qc.voteInfo.round + 1)
	return true
}
