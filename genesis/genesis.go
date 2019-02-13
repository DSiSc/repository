package genesis

import (
	"encoding/json"
	"fmt"
	"github.com/DSiSc/blockchain/common"
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/types"
	"math"
	"math/big"
	"os"
	"time"
)

// GenesisAccount is the account in genesis block.
type GenesisAccount struct {
	Addr    types.Address `json:"addr"     gencodec:"required"`
	Balance *big.Int      `json:"balance"`
	Code    []byte        `json:"code"`
}

// GensisBlock is the genesis block struct of the chain.
type GensisBlock struct {
	Block           *types.Block
	GenesisAccounts []GenesisAccount
	ExtraData       []byte `json:"extra_data"`
}

// BuildGensisBlock build genesis block from the specified config file.
// if the genesis config file is not specified, build default genesis block
func BuildGensisBlock(genesisPath string) (*GensisBlock, error) {
	if len(genesisPath) != 0 {
		log.Info("Build genesis block from genesis file: %s", genesisPath)
		return buildGenesisFromConfig(genesisPath)
	} else {
		log.Info("Start building default genesis block")
		return buildDefaultGenesis()
	}
}

// parse genesis block from config file.
func buildGenesisFromConfig(genesisPath string) (*GensisBlock, error) {
	file, err := os.Open(genesisPath)
	if err != nil {
		log.Error("Failed to open genesis file, as: %v", err)
		return nil, fmt.Errorf("Failed to open genesis file, as: %v ", err)
	}
	defer file.Close()

	genesis := new(GensisBlock)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		log.Error("Failed to parse genesis file, as: %v", err)
		return nil, fmt.Errorf("Failed to parse genesis file, as: %v ", err)
	}
	return genesis, nil
}

// build default genesis block.
func buildDefaultGenesis() (*GensisBlock, error) {
	genesisHeader := &types.Header{
		PrevBlockHash: types.Hash{},
		TxRoot:        types.Hash{},
		ReceiptsRoot:  types.Hash{},
		Height:        uint64(0),
		Timestamp:     uint64(time.Date(2018, time.August, 28, 0, 0, 0, 0, time.UTC).Unix()),
	}

	// genesis block
	genesisBlock := &GensisBlock{
		Block: &types.Block{
			Header:       genesisHeader,
			Transactions: make([]*types.Transaction, 0),
		},
		ExtraData: nil,
		GenesisAccounts: []GenesisAccount{
			{
				Addr:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
				Balance: new(big.Int).SetInt64(math.MaxInt64),
			},
			{
				Addr:    common.HexToAddress("0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b"),
				Balance: new(big.Int).SetInt64(math.MaxInt64),
			},
		},
	}
	return genesisBlock, nil
}
