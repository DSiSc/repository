package blockchain

import (
	"github.com/DSiSc/evm-NG/common"
	"github.com/DSiSc/evm-NG/statedb/state"
	pcommon "github.com/DSiSc/producer/common"
)

// BlockChainAPI is the blockchain module api, provide the chain manage API.
type BlockChainAPI interface {

	// GetBlockByHash retrieves a block from the local chain.
	GetBlockByHash(hash common.Hash) *pcommon.Block

	// CurrentBlock retrieves the head block from the local chain.
	CurrentBlock() *pcommon.Block

	// StateAt returns a new mutable state based on a particular point in time.
	StateAt(root common.Hash) (*state.CommonStateDB, error)

	// WriteBlockWithState writes the block and all associated state to the database.
	WriteBlockWithState(block *pcommon.Block, state *state.CommonStateDB) (err error)
}
