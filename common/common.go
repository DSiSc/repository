package common

import (
	"encoding/hex"
	gconf "github.com/DSiSc/craft/config"
	"github.com/DSiSc/craft/rlp"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/crypto-suite/crypto/sha3"
	"hash"
)

// Lengths of hashes and addresses in bytes.
const (
	HashLength    = 32
	AddressLength = 20
)

// get hash algorithm by global config
func HashAlg() hash.Hash {
	var alg string
	if value, ok := gconf.GlobalConfig.Load(gconf.HashAlgName); ok {
		alg = value.(string)
	} else {
		alg = "SHA256"
	}
	return sha3.NewHashByAlgName(alg)
}

// calculate the hash value of the rlp encoded byte of x
func rlpHash(x interface{}) (h types.Hash) {
	hw := HashAlg()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

// TxHash calculate tx's hash
func TxHash(tx *types.Transaction) (hash types.Hash) {
	if hash := tx.Hash.Load(); hash != nil {
		return hash.(types.Hash)
	}
	v := rlpHash(tx)
	tx.Hash.Store(v)
	return v
}

// HeaderHash calculate block's hash
func HeaderHash(block *types.Block) (hash types.Hash) {
	//var defaultHash types.Hash
	if !(block.HeaderHash == types.Hash{}) {
		var hash types.Hash
		copy(hash[:], block.HeaderHash[:])
		return hash
	}
	return rlpHash(block.Header)
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
