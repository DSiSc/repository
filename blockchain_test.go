package blockchain

import (
	"crypto/sha256"
	"fmt"
	"github.com/DSiSc/blockchain/common"
	"github.com/DSiSc/blockchain/config"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/monkey"
	"github.com/stretchr/testify/assert"
	"math"
	"math/big"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {
	// init event center
	chainConfig := config.BlockChainConfig{
		PluginName:    PLUGIN_MEMDB,
		StateDataPath: "/tmp/state",
		BlockDataPath: "/tmp/block",
	}
	err := InitBlockChain(chainConfig, &eventCenter{})
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

// mock block with txs
func mockBlockWithTx() (*types.Block, types.Transaction) {
	block := mockBlock()
	address := common.HexToAddress("")
	txHash := common.HexToHash("")
	tx := types.Transaction{
		Data: types.TxData{
			AccountNonce: 1,
			Recipient:    &address,
			From:         &address,
			Payload:      nil,
			Amount:       big.NewInt(100),
			GasLimit:     0,
			Price:        big.NewInt(100),
			V:            big.NewInt(100),
			R:            big.NewInt(100),
			S:            big.NewInt(100),
			Hash:         &txHash,
		},
	}
	block.Transactions = []*types.Transaction{&tx}
	return block, tx
}

// mock receipts
func mockReceipts() []*types.Receipt {
	receipt := types.Receipt{
		Status: 1,
	}
	return []*types.Receipt{&receipt}
}

// test init blockchain with memory database.
func TestInitBlockChain_WithMemDB(t *testing.T) {
	assert := assert.New(t)
	chainConfig := config.BlockChainConfig{
		PluginName:    PLUGIN_MEMDB,
		StateDataPath: "",
		BlockDataPath: "",
	}
	err := InitBlockChain(chainConfig, &eventCenter{})
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
	err := InitBlockChain(chainConfig, &eventCenter{})
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
	assert.Equal(0, balance.Cmp(big.NewInt(math.MaxInt64)))
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
	blockHash := common.HeaderHash(currentBlock)
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
	block.HeaderHash = common.HeaderHash(block)

	err = bc.WriteBlock(block)
	assert.Nil(err)
	assert.Equal(block.Header.Height, bc.GetCurrentBlockHeight())
}

// test write block with receipts
func TestBlockChain_WriteBlockWithReceipts(t *testing.T) {
	assert := assert.New(t)
	bc, err := NewLatestStateBlockChain()
	assert.Nil(err)
	assert.NotNil(bc)

	block := mockBlock()
	block.Header.Height = bc.GetCurrentBlockHeight() + 1
	block.Header.StateRoot = bc.IntermediateRoot(false)
	block.HeaderHash = common.HeaderHash(block)

	receipts := mockReceipts()
	err = bc.WriteBlockWithReceipts(block, receipts)
	assert.Nil(err)
	assert.Equal(block.Header.Height, bc.GetCurrentBlockHeight())
}

// test write block with receipts
func TestBlockChain_EventWriteBlockWithReceipts(t *testing.T) {
	assert := assert.New(t)
	bc, err := NewLatestStateBlockChain()
	assert.Nil(err)
	assert.NotNil(bc)

	block := mockBlock()
	block.Header.Height = bc.GetCurrentBlockHeight() + 1
	block.Header.StateRoot = bc.IntermediateRoot(false)
	block.HeaderHash = common.HeaderHash(block)

	receipts := mockReceipts()
	monkey.PatchInstanceMethod(reflect.TypeOf(globalEventCenter), "Notify", func(this *eventCenter, eventType types.EventType, value interface{}) (err error) {
		assert.Equal(types.EventBlockWritten, eventType)
		return nil
	})
	err = bc.EventWriteBlockWithReceipts(block, receipts, false)
	assert.Nil(err)
	assert.Equal(block.Header.Height, bc.GetCurrentBlockHeight())
	monkey.UnpatchAll()
	monkey.PatchInstanceMethod(reflect.TypeOf(globalEventCenter), "Notify", func(this *eventCenter, eventType types.EventType, value interface{}) (err error) {
		assert.Equal(types.EventBlockCommitted, eventType)
		return nil
	})
	err = bc.EventWriteBlockWithReceipts(block, receipts, true)
	assert.Nil(err)
	assert.Equal(block.Header.Height, bc.GetCurrentBlockHeight())
	monkey.UnpatchAll()
}

// test get transaction by hash
func TestBlockChain_GetTransactionByHash(t *testing.T) {
	assert := assert.New(t)
	bc, err := NewLatestStateBlockChain()
	assert.Nil(err)
	assert.NotNil(bc)

	block, tx := mockBlockWithTx()
	block.Header.Height = bc.GetCurrentBlockHeight() + 1
	block.Header.StateRoot = bc.IntermediateRoot(false)
	block.HeaderHash = common.HeaderHash(block)
	err = bc.WriteBlock(block)
	assert.Nil(err)
	assert.Equal(block.Header.Height, bc.GetCurrentBlockHeight())

	savedTx, _, _, _, err := bc.GetTransactionByHash(common.TxHash(&tx))
	assert.Nil(err)
	common.TxHash(savedTx)
	assert.Equal(&tx, savedTx)
}

// test get receipt by tx hash
func TestBlockChain_GetReceiptByTxHash(t *testing.T) {
	assert := assert.New(t)
	bc, err := NewLatestStateBlockChain()
	assert.Nil(err)
	assert.NotNil(bc)

	block, tx := mockBlockWithTx()
	block.Header.Height = bc.GetCurrentBlockHeight() + 1
	block.Header.StateRoot = bc.IntermediateRoot(false)
	block.HeaderHash = common.HeaderHash(block)

	receipts := mockReceipts()
	err = bc.WriteBlockWithReceipts(block, receipts)
	assert.Nil(err)
	assert.Equal(block.Header.Height, bc.GetCurrentBlockHeight())

	savedReceipt, _, _, _, err := bc.GetReceiptByTxHash(common.TxHash(&tx))
	assert.Nil(err)
	assert.Equal(receipts[0], savedReceipt)
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
