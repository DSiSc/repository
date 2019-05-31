package repository

import (
	"fmt"
	"github.com/DSiSc/blockstore"
	blkconf "github.com/DSiSc/blockstore/config"
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/monitor"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/repository/config"
	"github.com/DSiSc/statedb-NG"
	"github.com/DSiSc/statedb-NG/ethdb"
	"github.com/DSiSc/statedb-NG/ethdb/leveldb"
	"github.com/DSiSc/statedb-NG/ethdb/memorydb"
	"math/big"
	"sync"
)

// global disk database instance
var (
	lock              sync.RWMutex
	initialized       bool
	stateDiskDB       ethdb.Database         = nil
	globalBlockStore  *blockstore.BlockStore = nil
	globalEventCenter types.EventCenter      = nil
)

// repository module constant value.
const (
	// DB plugin
	PLUGIN_LEVELDB = "leveldb"
	// memory plugin
	PLUGIN_MEMDB = "memorydb"
)

// InitRepository init repository module config.
func InitRepository(chainConfig config.RepositoryConfig, eventCenter types.EventCenter) error {
	lock.Lock()
	defer lock.Unlock()
	if initialized {
		log.Warn("Chain repository has been initialized")
	}
	log.Info("Start initializing block chain")
	switch chainConfig.PluginName {
	case PLUGIN_LEVELDB:
		if chainConfig.StateDataPath == chainConfig.BlockDataPath {
			log.Error("Statedb path should be different with Blockdb path")
			return fmt.Errorf("statedb path should be different with blockdb path")
		}
		// create statedb low level database.
		levelDB, err := leveldb.New(chainConfig.StateDataPath, 0, 0, "")
		if err != nil {
			log.Error("Failed to create levelDB for Statedb, as: %v", err)
			return err
		}
		// create blockstore
		bstore, err := blockstore.NewBlockStore(&blkconf.BlockStoreConfig{
			PluginName: PLUGIN_LEVELDB,
			DataPath:   chainConfig.BlockDataPath,
		})
		if err != nil {
			log.Error("Failed to create file-based block store, as: %v", err)
			return err
		}
		stateDiskDB = levelDB
		globalBlockStore = bstore
	case PLUGIN_MEMDB:
		stateDiskDB = memorydb.New()
		// create blockstore
		bstore, err := blockstore.NewBlockStore(&blkconf.BlockStoreConfig{
			PluginName: PLUGIN_MEMDB,
			DataPath:   "",
		})
		if err != nil {
			log.Error("Failed to create memory-based block store, as: %v", err)
			return err
		}
		globalBlockStore = bstore
	default:
		log.Error("Unsupported plugin type: %s ", chainConfig.PluginName)
		return fmt.Errorf("Unsupported plugin type: %s ", chainConfig.PluginName)
	}

	globalEventCenter = eventCenter
	initialized = true
	return nil
}

// Repository is the chain manager.
//
// The Repository is used to write block to chain, get block from chain, and get the block state.
type Repository struct {
	blockStore   blockstore.BlockStoreAPI
	state        *statedb.StateDB
	mu           sync.RWMutex
	eventCenter  types.EventCenter
	currentBlock *types.Block
}

// NewLatestStateRespository create a repository with latest state hash.
func NewLatestStateRespository() (*Repository, error) {
	log.Debug("Create block chain at the latest height")
	if stateDiskDB == nil || globalBlockStore == nil {
		log.Error("Repository have not been initialized")
		return nil, fmt.Errorf("Repository have not been initialized")
	}
	currentBlock := globalBlockStore.GetCurrentBlock()
	if currentBlock == nil {
		log.Info("There are no blocks in block store, create block chain with init state")
		return NewRepositoryByHash(types.Hash{})
	} else {
		return NewRepositoryByHash(currentBlock.Header.StateRoot)
	}
}

// NewRepositoryByBlockHash create a repository with specified block hash.
func NewRepositoryByBlockHash(blockHash types.Hash) (*Repository, error) {
	log.Info("Create block chain with block hash: %x", blockHash)
	if stateDiskDB == nil || globalBlockStore == nil {
		log.Error("Repository have not been initialized")
		return nil, fmt.Errorf("Repository have not been initialized")
	}
	block, err := globalBlockStore.GetBlockByHash(blockHash)
	if err != nil {
		log.Error("Can not find block from block store with hash: %x", blockHash)
		return nil, fmt.Errorf("Can not find block from block store with hash: %x ", blockHash)
	}
	return NewRepositoryByHash(block.Header.StateRoot)
}

// NewRepositoryByHash returns a repository instance with specified hash.
func NewRepositoryByHash(root types.Hash) (*Repository, error) {
	log.Debug("Create block chain with hash root: %x", root)
	if stateDiskDB == nil || globalBlockStore == nil {
		log.Error("Repository have not been initialized")
		return nil, fmt.Errorf("Repository have not been initialized")
	}
	stateDB, err := statedb.New(root, statedb.NewDatabase(stateDiskDB))
	if err != nil {
		log.Error("Failed to create statedb, as: %v ", err)
		return nil, fmt.Errorf("Failed to create statedb, as: %v ", err)
	}
	lock.RLock()
	defer lock.RUnlock()
	return &Repository{
		state:       stateDB,
		blockStore:  globalBlockStore,
		eventCenter: globalEventCenter,
	}, nil
}

// WriteBlockWithState writes the block to the database.
func (repo *Repository) WriteBlock(block *types.Block) error {
	return repo.WriteBlockWithReceipts(block, nil)
}

// GetTransactionByHash get transaction by hash
func (repo *Repository) GetTransactionByHash(hash types.Hash) (*types.Transaction, types.Hash, uint64, uint64, error) {
	return repo.blockStore.GetTransactionByHash(hash)
}

// WriteBlock write the block and relative receipts to database. return error if write failed.
func (repo *Repository) WriteBlockWithReceipts(block *types.Block, receipts []*types.Receipt) error {
	log.Debug("write block %x with receipt.", block.HeaderHash)
	return repo.EventWriteBlockWithReceipts(block, receipts, true)
}

// WriteBlock write the block and relative receipts to database. return error if write failed.
func (repo *Repository) EventWriteBlockWithReceipts(block *types.Block, receipts []*types.Receipt, emitCommitEvent bool) error {
	log.Info("Start Writing block: %x, height: %d", block.HeaderHash, block.Header.Height)
	// write state to database
	stateRoot, err := repo.Commit(false)
	if err != nil {
		log.Error("Failed to Commit block %x's state root, block height %d, failed reason: %v", block.HeaderHash, block.Header.Height, err)
		if block.Header.Height == blockstore.INIT_BLOCK_HEIGHT || !emitCommitEvent {
			repo.eventCenter.Notify(types.EventBlockWriteFailed, err)
		} else {
			repo.eventCenter.Notify(types.EventBlockCommitFailed, err)
		}
		return err
	} else {
		log.Info("Commit state %x to database, current block height: %d", stateRoot, block.Header.Height)
	}

	// write block to block store
	if receipts == nil || len(receipts) == 0 {
		err = repo.blockStore.WriteBlock(block)
	} else {
		err = repo.blockStore.WriteBlockWithReceipts(block, receipts)
	}
	if err != nil {
		log.Error("Failed to write block to block store, as: %v", err)
		if block.Header.Height == blockstore.INIT_BLOCK_HEIGHT || !emitCommitEvent {
			repo.eventCenter.Notify(types.EventBlockWriteFailed, err)
		} else {
			repo.eventCenter.Notify(types.EventBlockCommitFailed, err)
		}
		return err
	}

	// send block Commit event if it is not the genesis block.
	if block.Header.Height == blockstore.INIT_BLOCK_HEIGHT || !emitCommitEvent {
		repo.eventCenter.Notify(types.EventBlockWritten, block)
	} else {
		repo.eventCenter.Notify(types.EventBlockCommitted, block)
	}
	log.Info("Write block %x height %d successfully", block.HeaderHash, block.Header.Height)

	monitor.JTMetrics.BlockTxNum.Set(float64(len(block.Transactions)))
	monitor.JTMetrics.CommittedTx.Add(float64(len(block.Transactions)))
	monitor.JTMetrics.BlockHeight.Set(float64(block.Header.Height))
	return nil
}

// GetReceiptByHash get receipt by relative tx's hash
func (repo *Repository) GetReceiptByTxHash(txHash types.Hash) (*types.Receipt, types.Hash, uint64, uint64, error) {
	return repo.blockStore.GetReceiptByTxHash(txHash)
}

// GetReceiptByHash get receipt by relative block's hash
func (repo *Repository) GetReceiptByBlockHash(blockHash types.Hash) []*types.Receipt {
	return repo.blockStore.GetReceiptByBlockHash(blockHash)
}

// GetLogs get transaction's execution log
func (repo *Repository) GetLogs(txHash types.Hash) []*types.Log {
	log.Debug("Get Tx[%x]'s execution log", txHash)
	return repo.state.GetLogs(txHash)
}

// GetBlockByHash retrieves a block from the local chain.
func (repo *Repository) GetBlockByHash(hash types.Hash) (*types.Block, error) {
	log.Debug("Get block by hash %x.", hash)
	return repo.blockStore.GetBlockByHash(hash)
}

// GetBlockByHeight get block by height.
func (repo *Repository) GetBlockByHeight(height uint64) (*types.Block, error) {
	log.Debug("Get block by height %d.", height)
	return repo.blockStore.GetBlockByHeight(height)
}

// GetCurrentBlock get current block.
func (repo *Repository) GetCurrentBlock() *types.Block {
	log.Debug("get current block.")
	return repo.blockStore.GetCurrentBlock()
}

// GetCurrentBlockHeight get current block height.
func (repo *Repository) GetCurrentBlockHeight() uint64 {
	log.Debug("get current block height.")
	return repo.blockStore.GetCurrentBlockHeight()
}

func (repo *Repository) SetBalance(addr types.Address, amount *big.Int) {
	repo.state.SetBalance(addr, amount)
}

// CreateAccount create an account
func (repo *Repository) CreateAccount(address types.Address) {
	repo.state.CreateAccount(address)
}

func (repo *Repository) SubBalance(address types.Address, value *big.Int) {
	repo.state.SubBalance(address, value)
}
func (repo *Repository) AddBalance(address types.Address, value *big.Int) {
	repo.state.AddBalance(address, value)
}
func (repo *Repository) GetBalance(address types.Address) *big.Int {
	return repo.state.GetBalance(address)
}

// GetNonce get account's nounce
func (repo *Repository) GetNonce(address types.Address) uint64 {
	return repo.state.GetNonce(address)
}
func (repo *Repository) SetNonce(address types.Address, value uint64) {
	repo.state.SetNonce(address, value)
}

func (repo *Repository) GetCodeHash(address types.Address) types.Hash {
	return repo.state.GetCodeHash(address)
}
func (repo *Repository) GetCode(address types.Address) []byte {
	log.Debug("get code by address %x.", address)
	return repo.state.GetCode(address)
}
func (repo *Repository) SetCode(address types.Address, code []byte) {
	repo.state.SetCode(address, code)
}
func (repo *Repository) GetCodeSize(address types.Address) int {
	return repo.state.GetCodeSize(address)
}

func (repo *Repository) AddRefund(value uint64) {
	repo.state.AddRefund(value)
}
func (repo *Repository) GetRefund() uint64 {
	return repo.state.GetRefund()
}

func (repo *Repository) GetState(address types.Address, bhash types.Hash) types.Hash {
	return repo.state.GetState(address, bhash)
}

func (repo *Repository) SetState(address types.Address, key, value types.Hash) {
	repo.state.SetState(address, key, value)
}

func (repo *Repository) Suicide(address types.Address) bool {
	return repo.state.Suicide(address)
}
func (repo *Repository) HasSuicided(address types.Address) bool {
	return repo.state.HasSuicided(address)
}

// Exist reports whether the given account exists in state.
// Notably this should also return true for suicided accounts.
func (repo *Repository) Exist(address types.Address) bool {
	return repo.state.Exist(address)
}

// Empty returns whether the given account is empty. Empty
// is defined according to EIP161 (balance = nonce = code = 0).
func (repo *Repository) Empty(address types.Address) bool {
	return repo.state.Empty(address)
}

func (repo *Repository) RevertToSnapshot(revid int) {
	repo.state.RevertToSnapshot(revid)
}
func (repo *Repository) Snapshot() int {
	return repo.state.Snapshot()
}

func (repo *Repository) AddPreimage(hash types.Hash, preimage []byte) {
	repo.state.AddPreimage(hash, preimage)
}

func (repo *Repository) ForEachStorage(addr types.Address, cb func(key, value types.Hash) bool) {
	repo.state.ForEachStorage(addr, cb)
}

// Reset clears out all ephemeral state objects from the state db, but keeps
// the underlying state trie to avoid reloading data for the next operations.
func (repo *Repository) Reset(root types.Hash) error {
	return repo.state.Reset(root)
}

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (repo *Repository) Finalise(deleteEmptyObjects bool) {
	repo.state.Finalise(deleteEmptyObjects)
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (repo *Repository) IntermediateRoot(deleteEmptyObjects bool) types.Hash {
	log.Debug("get intermediate root.")
	return repo.state.IntermediateRoot(deleteEmptyObjects)
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (repo *Repository) Prepare(thash, bhash types.Hash, ti int) {
	repo.state.Prepare(thash, bhash, ti)
}

// AddLog add contract execute log.
func (repo *Repository) AddLog(log *types.Log) {
	repo.state.AddLog(log)
}

// Commit writes the state to the underlying in-memory trie database.
func (repo *Repository) Commit(deleteEmptyObjects bool) (root types.Hash, err error) {
	log.Info("Start committing statedb state to database")
	//Commit statedb
	stateRoot, err := repo.state.Commit(false)
	if err != nil {
		log.Info("failed to Commit statedb to MPT tree, as: %v", err)
		return types.Hash{}, err
	}
	// Commit statedb to low level disk database
	err = repo.state.Database().TrieDB().Commit(stateRoot, false)
	if err != nil {
		log.Info("failed to Commit MPT tree database, as: %v", err)
		return types.Hash{}, err
	}
	return stateRoot, nil
}

// Put add a record to database
func (repo *Repository) Put(key []byte, value []byte) error {
	return repo.blockStore.Put(key, value)
}

// Get get a record by key
func (repo *Repository) Get(key []byte) ([]byte, error) {
	return repo.blockStore.Get(key)
}

// Delete removes the key from the key-value data store.
func (repo *Repository) Delete(key []byte) error {
	return repo.blockStore.Delete(key)
}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (repo *Repository) GetCommittedState(addr types.Address, hash types.Hash) types.Hash {
	return repo.state.GetCommittedState(addr, hash)
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (repo *Repository) SubRefund(gas uint64) {
	repo.state.SubRefund(gas)
}
