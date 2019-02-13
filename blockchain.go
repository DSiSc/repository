package blockchain

import (
	"fmt"
	"github.com/DSiSc/blockchain/common"
	"github.com/DSiSc/blockchain/config"
	"github.com/DSiSc/blockchain/genesis"
	"github.com/DSiSc/blockstore"
	blkconf "github.com/DSiSc/blockstore/config"
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/monitor"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/statedb-NG"
	"github.com/DSiSc/statedb-NG/ethdb"
	"math/big"
	"sync"
)

// global disk database instance
var (
	lock              sync.Mutex
	initialized       bool
	stateDiskDB       ethdb.Database         = nil
	globalBlockStore  *blockstore.BlockStore = nil
	globalEventCenter types.EventCenter      = nil
)

// blockchain module constant value.
const (
	// DB plugin
	PLUGIN_LEVELDB = "leveldb"
	// memory plugin
	PLUGIN_MEMDB = "memorydb"
)

// InitBlockChain init blockchain module config.
func InitBlockChain(chainConfig config.BlockChainConfig, eventCenter types.EventCenter) error {
	lock.Lock()
	defer lock.Unlock()
	if initialized {
		log.Warn("BlockChain has been initialized")
	}
	log.Info("Start initializing block chain")
	switch chainConfig.PluginName {
	case PLUGIN_LEVELDB:
		if chainConfig.StateDataPath == chainConfig.BlockDataPath {
			log.Error("Statedb path should be different with Blockdb path")
			return fmt.Errorf("statedb path should be different with blockdb path")
		}
		// create statedb low level database.
		levelDB, err := ethdb.NewLDBDatabase(chainConfig.StateDataPath, 0, 0)
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
		stateDiskDB = ethdb.NewMemDatabase()
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
	// init genesis block
	currentHeight := globalBlockStore.GetCurrentBlockHeight()
	log.Info("Current block height %d", currentHeight)
	if blockstore.INIT_BLOCK_HEIGHT == currentHeight {
		log.Info("There are no blocks in block store, we will reset the chain to genesis state")
		return ResetBlockChain(chainConfig.GenesisFile)
	}
	initialized = true
	return nil
}

// reset blockchain to genesis state.
func ResetBlockChain(genesisPath string) error {
	log.Info("Start resetting chain with genesis block file: %s", genesisPath)
	bc, err := NewBlockChainByHash(types.Hash{})
	if err != nil {
		log.Error("Failed to create init-state block chain, as: %v", err)
		return err
	}
	// build genesis block
	genesis, err := genesis.BuildGensisBlock(genesisPath)
	if err != nil {
		log.Error("Failed to build genesis block, as: %v", err)
		return err
	}

	// record genesis account
	for _, account := range genesis.GenesisAccounts {
		if account.Balance.Cmp(big.NewInt(0)) == 1 {
			bc.CreateAccount(account.Addr)
			bc.SetBalance(account.Addr, account.Balance)
			bc.SetCode(account.Addr, account.Code)
		}
	}
	genesisStateRoot := bc.IntermediateRoot(false)

	// write block to chain.
	block := genesis.Block
	block.Header.StateRoot = genesisStateRoot
	block.HeaderHash = common.HeaderHash(block)
	err = bc.WriteBlock(block)
	if err != nil {
		log.Error("Failed to write genesis block to block store, as:%v", err)
		return err
	} else {
		return nil
	}
}

// BlockChain is the chain manager.
//
// The BlockChain is used to write block to chain, get block from chain, and get the block state.
type BlockChain struct {
	blockStore   blockstore.BlockStoreAPI
	state        *statedb.StateDB
	mu           sync.RWMutex
	eventCenter  types.EventCenter
	currentBlock *types.Block
}

// NewLatestStateBlockChain create a blockchain with latest state hash.
func NewLatestStateBlockChain() (*BlockChain, error) {
	log.Debug("Create block chain at the latest height")
	if stateDiskDB == nil || globalBlockStore == nil {
		log.Error("BlockChain have not been initialized")
		return nil, fmt.Errorf("BlockChain have not been initialized")
	}
	currentBlock := globalBlockStore.GetCurrentBlock()
	if currentBlock == nil {
		log.Info("There are no blocks in block store, create block chain with init state")
		return NewBlockChainByHash(types.Hash{})
	} else {
		return NewBlockChainByHash(currentBlock.Header.StateRoot)
	}
}

// NewLatestStateBlockChain create a blockchain with specified block hash.
func NewBlockChainByBlockHash(blockHash types.Hash) (*BlockChain, error) {
	log.Info("Create block chain with block hash: %x", blockHash)
	if stateDiskDB == nil || globalBlockStore == nil {
		log.Error("BlockChain have not been initialized")
		return nil, fmt.Errorf("BlockChain have not been initialized")
	}
	block, err := globalBlockStore.GetBlockByHash(blockHash)
	if err != nil {
		log.Error("Can not find block from block store with hash: %x", blockHash)
		return nil, fmt.Errorf("Can not find block from block store with hash: %x ", blockHash)
	}
	return NewBlockChainByHash(block.Header.StateRoot)
}

// NewBlockChain returns a blockchain instance with specified hash.
func NewBlockChainByHash(root types.Hash) (*BlockChain, error) {
	log.Debug("Create block chain with hash root: %x", root)
	if stateDiskDB == nil || globalBlockStore == nil {
		log.Error("BlockChain have not been initialized")
		return nil, fmt.Errorf("BlockChain have not been initialized")
	}
	stateDB, err := statedb.New(root, statedb.NewDatabase(stateDiskDB))
	if err != nil {
		log.Error("Failed to create statedb, as: %v ", err)
		return nil, fmt.Errorf("Failed to create statedb, as: %v ", err)
	}
	return &BlockChain{
		state:       stateDB,
		blockStore:  globalBlockStore,
		eventCenter: globalEventCenter,
	}, nil
}

// WriteBlockWithState writes the block to the database.
func (blockChain *BlockChain) WriteBlock(block *types.Block) error {
	return blockChain.WriteBlockWithReceipts(block, nil)
}

// GetTransactionByHash get transaction by hash
func (blockChain *BlockChain) GetTransactionByHash(hash types.Hash) (*types.Transaction, types.Hash, uint64, uint64, error) {
	return blockChain.blockStore.GetTransactionByHash(hash)
}

// WriteBlock write the block and relative receipts to database. return error if write failed.
func (blockChain *BlockChain) WriteBlockWithReceipts(block *types.Block, receipts []*types.Receipt) error {
	return blockChain.EventWriteBlockWithReceipts(block, receipts, true)
}

// WriteBlock write the block and relative receipts to database. return error if write failed.
func (blockChain *BlockChain) EventWriteBlockWithReceipts(block *types.Block, receipts []*types.Receipt, emitCommitEvent bool) error {
	log.Info("Start Writing block: %x, height: %d", block.HeaderHash, block.Header.Height)
	// write state to database
	stateRoot, err := blockChain.commit(false)
	if err != nil {
		log.Error("Failed to commit block %x's state root, block height %d, failed reason: %v", block.HeaderHash, block.Header.Height, err)
		if block.Header.Height == blockstore.INIT_BLOCK_HEIGHT || !emitCommitEvent {
			blockChain.eventCenter.Notify(types.EventBlockWriteFailed, err)
		} else {
			blockChain.eventCenter.Notify(types.EventBlockCommitFailed, err)
		}
		return err
	} else {
		log.Info("Commit state %x to database, current block height: %d", stateRoot, block.Header.Height)
	}

	// write block to block store
	if receipts == nil || len(receipts) == 0 {
		err = blockChain.blockStore.WriteBlock(block)
	} else {
		err = blockChain.blockStore.WriteBlockWithReceipts(block, receipts)
	}
	if err != nil {
		log.Error("Failed to write block to block store, as: %v", err)
		if block.Header.Height == blockstore.INIT_BLOCK_HEIGHT || !emitCommitEvent {
			blockChain.eventCenter.Notify(types.EventBlockWriteFailed, err)
		} else {
			blockChain.eventCenter.Notify(types.EventBlockCommitFailed, err)
		}
		return err
	}

	// send block commit event if it is not the genesis block.
	if block.Header.Height == blockstore.INIT_BLOCK_HEIGHT || !emitCommitEvent {
		blockChain.eventCenter.Notify(types.EventBlockWritten, block)
	} else {
		blockChain.eventCenter.Notify(types.EventBlockCommitted, block)
	}
	log.Info("Write block %x height %d successfully", block.HeaderHash, block.Header.Height)

	monitor.JTMetrics.BlockTxNum.Set(float64(len(block.Transactions)))
	monitor.JTMetrics.CommittedTx.Add(float64(len(block.Transactions)))
	monitor.JTMetrics.BlockHeight.Set(float64(block.Header.Height))
	return nil
}

// GetReceiptByHash get receipt by relative tx's hash
func (blockChain *BlockChain) GetReceiptByTxHash(txHash types.Hash) (*types.Receipt, types.Hash, uint64, uint64, error) {
	return blockChain.blockStore.GetReceiptByTxHash(txHash)
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
	log.Info("Start committing statedb state to database")
	//commit statedb
	stateRoot, err := blockChain.state.Commit(false)
	if err != nil {
		log.Info("failed to commit statedb to MPT tree, as: %v", err)
		return types.Hash{}, err
	}
	// commit statedb to low level disk database
	err = blockChain.state.Database().TrieDB().Commit(stateRoot, false)
	if err != nil {
		log.Info("failed to commit MPT tree database, as: %v", err)
		return types.Hash{}, err
	}
	return stateRoot, nil
}
