package blockchain

import (
	"fmt"
	"github.com/DSiSc/blockchain/config"
	"github.com/DSiSc/blockchain/genesis"
	"github.com/DSiSc/blockstore"
	blkconf "github.com/DSiSc/blockstore/config"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/statedb-NG"
	"github.com/DSiSc/statedb-NG/ethdb"
	"math/big"
	"sync"
)

// global disk database instance
var (
	stateDiskDB      ethdb.Database         = nil
	globalBlockStore *blockstore.BlockStore = nil
)

// blockchain module constant value.
const (
	// DB plugin
	PLUGIN_LEVELDB = "leveldb"
	// memory plugin
	PLUGIN_MEMDB = "memorydb"
)

// InitBlockChain init blockchain module config.
func InitBlockChain(chainConfig config.BlockChainConfig) error {
	switch chainConfig.PluginName {
	case PLUGIN_LEVELDB:
		if chainConfig.StateDataPath == chainConfig.BlockDataPath {
			return fmt.Errorf("statedb path should be different with blockdb path")
		}
		// create statedb low level database.
		levelDB, err := ethdb.NewLDBDatabase(chainConfig.StateDataPath, 0, 0)
		if err != nil {
			return err
		}
		// create blockstore
		bstore, err := blockstore.NewBlockStore(&blkconf.BlockStoreConfig{
			PluginName: PLUGIN_LEVELDB,
			DataPath:   chainConfig.BlockDataPath,
		})
		if err != nil {
			return err
		}
		stateDiskDB = levelDB
		globalBlockStore = bstore
	case PLUGIN_MEMDB:
		stateDiskDB = ethdb.NewMemDatabase()
		// create blockstore
		bstore, err := blockstore.NewBlockStore(&blkconf.BlockStoreConfig{
			PluginName: PLUGIN_MEMDB,
			DataPath:   "",
		})
		if err != nil {
			return err
		}
		globalBlockStore = bstore
	default:
		return fmt.Errorf("Unsupported plugin type ")
	}

	// init genesis block
	currentHeight := globalBlockStore.GetCurrentBlockHeight()
	if blockstore.INIT_BLOCK_HEIGHT == currentHeight {
		return ResetBlockChain()
	}
	return nil
}

// reset blockchain to genesis state.
func ResetBlockChain() error {
	bc, err := NewBlockChainByHash(types.Hash{})
	if err != nil {
		return err
	}
	// build genesis block
	genesis, err := genesis.BuildGensisBlock()
	if err != nil {
		return err
	}

	//TODO init chain genesis account info from genesis block. e.g. ext := genesis.ExtraData; bc.CreateAccount()...
	genesisStateRoot := bc.IntermediateRoot(false)
	if err != nil {
		return err
	}

	// write block to chain.
	block := genesis.Block
	block.Header.StateRoot = genesisStateRoot
	return bc.WriteBlock(block)
}

// BlockChain is the chain manager.
//
// The BlockChain is used to write block to chain, get block from chain, and get the block state.
type BlockChain struct {
	blockStore   blockstore.BlockStoreAPI
	state        *statedb.StateDB
	mu           sync.RWMutex
	currentBlock *types.Block
}

// NewLatestStateBlockChain create a blockchain with latest state hash.
func NewLatestStateBlockChain() (*BlockChain, error) {
	if stateDiskDB == nil || globalBlockStore == nil {
		return nil, fmt.Errorf("BlockChain have not been initialized.")
	}
	currentBlock := globalBlockStore.GetCurrentBlock()
	if currentBlock == nil {
		return NewBlockChainByHash(types.Hash{})
	} else {
		return NewBlockChainByHash(currentBlock.Header.StateRoot)
	}
}

// NewLatestStateBlockChain create a blockchain with specified block hash.
func NewBlockChainByBlockHash(blockHash types.Hash) (*BlockChain, error) {
	if stateDiskDB == nil || globalBlockStore == nil {
		return nil, fmt.Errorf("BlockChain have not been initialized.")
	}
	block, err := globalBlockStore.GetBlockByHash(blockHash)
	if err != nil {
		return nil, fmt.Errorf("Can not find the block with specified hash: %s.", blockHash)
	}
	return NewBlockChainByHash(block.Header.StateRoot)
}

// NewBlockChain returns a blockchain instance with specified hash.
func NewBlockChainByHash(root types.Hash) (*BlockChain, error) {
	if stateDiskDB == nil || globalBlockStore == nil {
		return nil, fmt.Errorf("BlockChain have not been initialized.")
	}
	stateDB, err := statedb.New(root, statedb.NewDatabase(stateDiskDB))
	if err != nil {
		return nil, fmt.Errorf("Failed to create low level statedb, as %s", err)
	}
	return &BlockChain{
		state:      stateDB,
		blockStore: globalBlockStore,
	}, nil
}

// WriteBlockWithState writes the block to the database.
func (blockChain *BlockChain) WriteBlock(block *types.Block) error {
	// write state to database
	_, err := blockChain.commit(false)
	if err != nil {
		return err
	}

	// write block to block store
	return blockChain.blockStore.WriteBlock(block)
}

// GetBlockByHash retrieves a block from the local chain.
func (blockChain *BlockChain) GetBlockByHash(hash types.Hash) (*types.Block, error) {
	return blockChain.blockStore.GetBlockByHash(hash)
}

// GetBlockByHeight get block by height.
func (blockChain *BlockChain) GetBlockByHeight(height uint64) (*types.Block, error) {
	return blockChain.blockStore.GetBlockByHeight(height)
}

// GetCurrentBlock get current block.
func (blockChain *BlockChain) GetCurrentBlock() *types.Block {
	return blockChain.blockStore.GetCurrentBlock()
}

// GetCurrentBlockHeight get current block height.
func (blockChain *BlockChain) GetCurrentBlockHeight() uint64 {
	return blockChain.blockStore.GetCurrentBlockHeight()
}

func (blockChain *BlockChain) SetBalance(addr types.Address, amount *big.Int) {
	blockChain.state.SetBalance(addr, amount)
}

// CreateAccount create an account
func (blockChain *BlockChain) CreateAccount(address types.Address) {
	blockChain.state.CreateAccount(address)
}

func (blockChain *BlockChain) SubBalance(address types.Address, value *big.Int) {
	blockChain.state.SubBalance(address, value)
}
func (blockChain *BlockChain) AddBalance(address types.Address, value *big.Int) {
	blockChain.state.AddBalance(address, value)
}
func (blockChain *BlockChain) GetBalance(address types.Address) *big.Int {
	return blockChain.state.GetBalance(address)
}

// GetNonce get account's nounce
func (blockChain *BlockChain) GetNonce(address types.Address) uint64 {
	return blockChain.state.GetNonce(address)
}
func (blockChain *BlockChain) SetNonce(address types.Address, value uint64) {
	blockChain.state.SetNonce(address, value)
}

func (blockChain *BlockChain) GetCodeHash(address types.Address) types.Hash {
	return blockChain.state.GetCodeHash(address)
}
func (blockChain *BlockChain) GetCode(address types.Address) []byte {
	return blockChain.state.GetCode(address)
}
func (blockChain *BlockChain) SetCode(address types.Address, code []byte) {
	blockChain.state.SetCode(address, code)
}
func (blockChain *BlockChain) GetCodeSize(address types.Address) int {
	return blockChain.state.GetCodeSize(address)
}

func (blockChain *BlockChain) AddRefund(value uint64) {
	blockChain.state.AddRefund(value)
}
func (blockChain *BlockChain) GetRefund() uint64 {
	return blockChain.state.GetRefund()
}

func (blockChain *BlockChain) GetState(address types.Address, bhash types.Hash) types.Hash {
	return blockChain.state.GetState(address, bhash)
}

func (blockChain *BlockChain) SetState(address types.Address, key, value types.Hash) {
	blockChain.state.SetState(address, key, value)
}

func (blockChain *BlockChain) Suicide(address types.Address) bool {
	return blockChain.state.Suicide(address)
}
func (blockChain *BlockChain) HasSuicided(address types.Address) bool {
	return blockChain.state.HasSuicided(address)
}

// Exist reports whether the given account exists in state.
// Notably this should also return true for suicided accounts.
func (blockChain *BlockChain) Exist(address types.Address) bool {
	return blockChain.state.Exist(address)
}

// Empty returns whether the given account is empty. Empty
// is defined according to EIP161 (balance = nonce = code = 0).
func (blockChain *BlockChain) Empty(address types.Address) bool {
	return blockChain.state.Empty(address)
}

func (blockChain *BlockChain) RevertToSnapshot(revid int) {
	blockChain.state.RevertToSnapshot(revid)
}
func (blockChain *BlockChain) Snapshot() int {
	return blockChain.state.Snapshot()
}

func (blockChain *BlockChain) AddPreimage(hash types.Hash, preimage []byte) {
	blockChain.state.AddPreimage(hash, preimage)
}

func (blockChain *BlockChain) ForEachStorage(addr types.Address, cb func(key, value types.Hash) bool) {
	blockChain.state.ForEachStorage(addr, cb)
}

// Reset clears out all ephemeral state objects from the state db, but keeps
// the underlying state trie to avoid reloading data for the next operations.
func (blockChain *BlockChain) Reset(root types.Hash) error {
	return blockChain.state.Reset(root)
}

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (blockChain *BlockChain) Finalise(deleteEmptyObjects bool) {
	blockChain.state.Finalise(deleteEmptyObjects)
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (blockChain *BlockChain) IntermediateRoot(deleteEmptyObjects bool) types.Hash {
	return blockChain.state.IntermediateRoot(deleteEmptyObjects)
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (blockChain *BlockChain) Prepare(thash, bhash types.Hash, ti int) {
	blockChain.state.Prepare(thash, bhash, ti)
}

// TODO: add contract execute log.
func (blockChain *BlockChain) AddLog(interface{}) {
	//TODO
}

// Commit writes the state to the underlying in-memory trie database.
func (blockChain *BlockChain) commit(deleteEmptyObjects bool) (root types.Hash, err error) {
	//commit statedb
	stateRoot, err := blockChain.state.Commit(false)
	if err != nil {
		return types.Hash{}, err
	}
	// commit statedb to low level disk database
	err = blockChain.state.Database().TrieDB().Commit(stateRoot, false)
	if err != nil {
		return types.Hash{}, err
	}
	return stateRoot, nil
}
