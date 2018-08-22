package blockchain

import (
	"github.com/DSiSc/evm-NG/common"
	"github.com/DSiSc/evm-NG/statedb/ethdb"
	"github.com/DSiSc/evm-NG/statedb/state"
	pcommon "github.com/DSiSc/producer/common"
	"sync"
)

// BlockChain is the chain manager.
//
// The BlockChain is used to write block to chain, get block from chain, and get the block state.
type BlockChain struct {
	db ethdb.Database
	mu sync.RWMutex
}

// NewBlockChain returns a fully initialised block chain using information
// available in the database.
func NewBlockChain(db ethdb.Database) (*BlockChain, error) {
	bc := &BlockChain{
		db: db,
	}
	return bc, nil
}

// GetBlockByHash retrieves a block from the local chain.
func (blockChain *BlockChain) GetBlockByHash(hash common.Hash) *pcommon.Block {
	//TODO
	return &pcommon.Block{}
}

// CurrentBlock retrieves the head block from the local chain.
func (blockChain *BlockChain) CurrentBlock() *pcommon.Block {
	//TODO
	return &pcommon.Block{}
}

// StateAt returns a new mutable state based on a particular point in time.
func (blockChain *BlockChain) StateAt(root common.Hash) (*state.CommonStateDB, error) {
	return state.New(root, state.NewDatabase(blockChain.db))
}

// WriteBlockWithState writes the block to the database.
func (blockChain *BlockChain) WriteBlock(block *pcommon.Block) (err error) {
	//TODO
	return nil
}
