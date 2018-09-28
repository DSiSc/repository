package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/types"
)

// Lengths of hashes and addresses in bytes.
const (
	HashLength    = 32
	AddressLength = 20
)

func Sum(bz []byte) []byte {
	hash := sha256.Sum256(bz)
	return hash[:types.HashLength]
}

// TxHash calculate tx's hash
func TxHash(tx *types.Transaction) (hash types.Hash) {
	if hash := tx.Hash.Load(); hash != nil {
		return hash.(types.Hash)
	}
	hashData := types.TxData{
		AccountNonce: tx.Data.AccountNonce,
		Price:        tx.Data.Price,
		GasLimit:     tx.Data.GasLimit,
		Recipient:    tx.Data.Recipient,
		Amount:       tx.Data.Amount,
		Payload:      tx.Data.Payload,
		V:            tx.Data.V,
		R:            tx.Data.R,
		S:            tx.Data.S,
	}
	jsonByte, _ := json.Marshal(hashData)
	sumByte := Sum(jsonByte)
	var temp types.Hash
	copy(temp[:], sumByte)
	tx.Hash.Store(temp)
	return temp
}

func HeaderHash(block *types.Block) (hash types.Hash) {
	var defaultHash types.Hash
	if !bytes.Equal(block.HeaderHash[:], defaultHash[:]) {
		log.Info("block hash %v has exits.", block.HeaderHash)
		copy(hash[:], block.HeaderHash[:])
		return
	}
	jsonByte, _ := json.Marshal(block.Header)
	sumByte := Sum(jsonByte)
	copy(hash[:], sumByte)
	return
}

// BytesToAddress returns Address with value b.
// If b is larger than len(h), b will be cropped from the left.
func BytesToAddress(b []byte) types.Address {
	var a types.Address
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
	return a
}

// HexToAddress returns Address with byte values of s.
// If s is larger than len(h), s will be cropped from the left.
func HexToAddress(s string) types.Address { return BytesToAddress(FromHex(s)) }

// HexToHash sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) types.Hash { return BytesToHash(FromHex(s)) }

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) types.Hash {
	var h types.Hash
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
	return h
}

// FromHex returns the bytes represented by the hexadecimal string s.
// s may be prefixed with "0x".
func FromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return Hex2Bytes(s)
}

// Hex2Bytes returns the bytes represented by the hexadecimal string str.
func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}
