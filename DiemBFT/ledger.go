package DiemBFT

import (
	"encoding/hex"

	"github.com/kevinburke/nacl"
)

func (v *Validator) commit(blockId string) {
	if blockNode, ok := v.pendingBlockTree[blockId]; ok {
		block := blockNode.block
		v.logger.Info("Committing block: ", blockId, " with payload: ", block.payload)
		v.commits[blockId] = block
		v.commitCh <- block.payload
	} else {
		v.logger.Error("Error in Ledger.commit: Block ID: ", blockId, " not in pendingBlockTree.")
		return
	}
}

func (v *Validator) speculate(prevBlockId, blockId, txns string) {
	execStateIdHash := nacl.Hash([]byte(v.stateId[prevBlockId] + txns))
	v.stateId[blockId] = hex.EncodeToString((*execStateIdHash)[:])
}

func (v *Validator) pendingState(blockId string) string {
	return v.stateId[blockId]
}

func (v *Validator) committedBlock(blockId string) *Block {
	if block, ok := v.commits[blockId]; ok {
		return &block
	}
	v.logger.Error("Referencing uncommitted block ", blockId)
	return nil
}
