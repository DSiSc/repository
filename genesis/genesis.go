package genesis

import (
	"github.com/DSiSc/craft/types"
	"time"
)

type GensisBlock struct {
	block     *types.Block     // genesis to a block
	extraData []byte           // extra data of genesis
}

func BuildGensisBlock() (*GensisBlock, error) {
	genesisHeader := &types.Header{
		PrevBlockHash: types.Hash{},
		TxRoot:        types.Hash{},
		StateRoot:     types.Hash{},
		ReceipsRoot:   types.Hash{},
		Height:        uint64(0),
		Timestamp:     uint64(time.Date(2018, time.June, 30, 0, 0, 0, 0, time.UTC).Unix()),
		MixDigest:     types.Hash{},
		SigData:       nil,
	}

	genesisBlock := &GensisBlock{
		block: &types.Block{
			Header:       genesisHeader,
			Transactions: make([]*types.Transaction, 0),
		},
		extraData: nil,
	}

	return genesisBlock, nil
}
