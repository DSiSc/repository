package common

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/DSiSc/craft/types"
)

func Sum(bz []byte) []byte {
	hash := sha256.Sum256(bz)
	return hash[:types.HashLength]
}

func HeaderHash(block *types.Block) (hash types.Hash) {
	header := block.Header
	jsonByte, _ := json.Marshal(header)
	sumByte := Sum(jsonByte)
	copy(hash[:], sumByte)
	block.HeaderHash = hash
	return
}
