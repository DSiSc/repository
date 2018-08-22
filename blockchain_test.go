package blockchain

import (
	"github.com/DSiSc/evm-NG/common"
	"github.com/DSiSc/evm-NG/statedb/ethdb"
	common2 "github.com/DSiSc/producer/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

// mock database.
func mockDataBase() ethdb.Database {
	return ethdb.NewMemDatabase()
}

// Test create blockchain.
func TestNewBlockChain(t *testing.T) {
	assert := assert.New(t)
	db := mockDataBase()
	bc, err := NewBlockChain(db)
	assert.NotNil(bc, "FAILED: failed to create blockchain")
	assert.Nil(err, "FAILED: failed to create blockchain.")
}

// Test get block by hash.
func TestBlockChain_GetBlockByHash(t *testing.T) {
	assert := assert.New(t)
	db := mockDataBase()
	bc, _ := NewBlockChain(db)
	assert.NotNil(bc, "FAILED: failed to create blockchain")
	block := bc.GetBlockByHash(common.Hash{})
	assert.NotNil(block, "FAILED: failed to get block by hash")
}

// Test get current block of the chain.
func TestBlockChain_CurrentBlock(t *testing.T) {
	assert := assert.New(t)
	db := mockDataBase()
	bc, _ := NewBlockChain(db)
	assert.NotNil(bc, "FAILED: failed to create blockchain")
	block := bc.CurrentBlock()
	assert.NotNil(block, "FAILED: failed to get block by hash")
}

// Test get the statedb at specify height.
func TestBlockChain_StateAt(t *testing.T) {
	assert := assert.New(t)
	root := common.Hash{}
	db := mockDataBase()
	bc, _ := NewBlockChain(db)
	assert.NotNil(bc, "FAILED: failed to create blockchain")
	state, err := bc.StateAt(root)
	assert.NotNil(state, "FAILED: failed to get the statedb")
	assert.Nil(err, "FAILED: failed to get the statedb")
}

// create block
func mockBlock() *common2.Block {
	header := &common2.Header{}
	block := &common2.Block{
		Header: header,
	}
	return block
}

// Test write block to chain
func TestBlockChain_WriteBlockWithState(t *testing.T) {
	assert := assert.New(t)
	db := mockDataBase()
	bc, _ := NewBlockChain(db)
	assert.NotNil(bc, "FAILED: failed to create blockchain")

	block := mockBlock()
	err := bc.WriteBlock(block)
	assert.Nil(err, "FAILED: failed to write block and state")
}
