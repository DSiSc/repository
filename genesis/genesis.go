package genesis

import (
	"github.com/DSiSc/blockchain/common"
	"github.com/DSiSc/craft/types"
	"time"
)

type GenesisBlockConf struct {
	PrevBlockHash types.Hash
	TxRoot        types.Hash
	StateRoot     types.Hash
	ReceiptsRoot  types.Hash
	Height        uint64
	Timestamp     uint64
	MixDigest     types.Hash
}

type GensisBlock struct {
	Block     *types.Block
	ExtraData []byte
}

// BuildGensisBlock build genesis block from genesis config file
func BuildGensisBlock() (*GensisBlock, error) {
	genesisHeader := &types.Header{
		PrevBlockHash: types.Hash{},
		TxRoot:        types.Hash{},
		StateRoot:     types.Hash{},
		ReceiptsRoot:  types.Hash{},
		Height:        uint64(0),
		Timestamp:     uint64(time.Date(2018, time.August, 28, 0, 0, 0, 0, time.UTC).Unix()),
		MixDigest:     types.Hash{},
	}

	genesisBlock := &GensisBlock{
		Block: &types.Block{
			Header:       genesisHeader,
			Transactions: make([]*types.Transaction, 0),
			SigData:      make([][]byte, 0),
		},
		ExtraData: nil,
	}

	genesisBlock.Block.HeaderHash = common.HeaderHash(genesisBlock.Block)

	return genesisBlock, nil
}
