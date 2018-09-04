package blockchain

import (
	"fmt"
	"github.com/DSiSc/blockchain/config"
	"github.com/DSiSc/craft/types"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	chainConfig := config.BlockChainConfig{
		PluginName:    PLUGIN_MEMDB,
		StateDataPath: "/tmp/state",
		BlockDataPath: "/tmp/block",
	}
	err := InitBlockChain(chainConfig)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	m.Run()
}

// mock block struct
func mockBlock() *types.Block {
	header := &types.Header{}
	block := &types.Block{
		Header: header,
	}
	return block
}

// test init blockchain with memory database.
func TestInitBlockChain_WithMemDB(t *testing.T) {
	assert := assert.New(t)
	chainConfig := config.BlockChainConfig{
		PluginName:    PLUGIN_MEMDB,
		StateDataPath: "",
		BlockDataPath: "",
	}
	err := InitBlockChain(chainConfig)
	assert.Nil(err)
}

// test init blockchain with file database.
func TestInitBlockChain_WithFileDB(t *testing.T) {
	assert := assert.New(t)
	chainConfig := config.BlockChainConfig{
		PluginName:    PLUGIN_LEVELDB,
		StateDataPath: "/tmp/state",
		BlockDataPath: "/tmp/block",
	}
	err := InitBlockChain(chainConfig)
	assert.Nil(err)
}

// test reset chain
func TestResetBlockChain(t *testing.T) {
	assert := assert.New(t)
	err := ResetBlockChain()
	assert.Nil(err)
}

// test new latest blockchain
func TestNewLatestStateBlockChain(t *testing.T) {
	assert := assert.New(t)
	bc, err := NewLatestStateBlockChain()
	assert.Nil(err)
	assert.NotNil(bc)
}

// test new blockchain by block hash
func TestNewBlockChainByBlockHash(t *testing.T) {
	assert := assert.New(t)
	bc, err := NewLatestStateBlockChain()
	assert.Nil(err)
	assert.NotNil(bc)
	currentBlock := bc.GetCurrentBlock()
	blockHash := currentBlock.HeaderHash
	bc, err = NewBlockChainByBlockHash(blockHash)
	assert.Nil(err)
	assert.NotNil(bc)
	assert.Equal(currentBlock.Header.StateRoot, bc.IntermediateRoot(false))
}

// test new blockchain by hash
func TestNewBlockChainByHash(t *testing.T) {
	assert := assert.New(t)
	bc, err := NewLatestStateBlockChain()
	assert.Nil(err)
	assert.NotNil(bc)
	currentBlock := bc.GetCurrentBlock()
	currentHash := currentBlock.Header.StateRoot
	bc, err = NewBlockChainByHash(currentHash)
	assert.Nil(err)
	assert.NotNil(bc)
}

// test write block
func TestBlockChain_WriteBlock(t *testing.T) {
	assert := assert.New(t)
	bc, err := NewLatestStateBlockChain()
	assert.Nil(err)
	assert.NotNil(bc)

	block := mockBlock()
	block.Header.Height = bc.GetCurrentBlockHeight() + 1
	block.Header.StateRoot = bc.IntermediateRoot(false)

	err = bc.WriteBlock(block)
	assert.Nil(err)
	assert.Equal(block.Header.Height, bc.GetCurrentBlockHeight())
}
