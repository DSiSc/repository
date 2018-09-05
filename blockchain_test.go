package blockchain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/DSiSc/blockchain/common"
	"github.com/DSiSc/blockchain/config"
	"github.com/DSiSc/craft/types"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// init event center
	types.GlobalEventCenter = &eventCenter{}

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
	err := ResetBlockChain("")
	assert.Nil(err)
	bc, err := NewLatestStateBlockChain()
	assert.Nil(err)
	assert.NotNil(bc)
	balance := bc.GetBalance(common.HexToAddress("0x0000000000000000000000000000000000000000"))
	assert.Equal(0, balance.Cmp(big.NewInt(100000000)))
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
	blockHash := blockHash(currentBlock)
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

// BlockHash calculate block's hash
func blockHash(block *types.Block) (hash types.Hash) {
	jsonByte, _ := json.Marshal(block)
	sumByte := sum(jsonByte)
	copy(hash[:], sumByte)
	return
}

// Sum returns the first 20 bytes of SHA256 of the bz.
func sum(bz []byte) []byte {
	hash := sha256.Sum256(bz)
	return hash[:types.HashLength]
}

type eventCenter struct {
}

// subscriber subscribe specified eventType with eventFunc
func (*eventCenter) Subscribe(eventType types.EventType, eventFunc types.EventFunc) types.Subscriber {
	return nil
}

// subscriber unsubscribe specified eventType
func (*eventCenter) UnSubscribe(eventType types.EventType, subscriber types.Subscriber) (err error) {
	return nil
}

// notify subscriber of eventType
func (*eventCenter) Notify(eventType types.EventType, value interface{}) (err error) {
	return nil
}

// notify specified eventFunc
func (*eventCenter) NotifySubscriber(eventFunc types.EventFunc, value interface{}) {

}

// notify subscriber traversing all events
func (*eventCenter) NotifyAll() (errs []error) {
	return nil
}

// unsubscrible all event
func (*eventCenter) UnSubscribeAll() {
}
