package genesis

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// test build default genesis block
func TestBuildDefaultGensisBlock(t *testing.T) {
	assert := assert.New(t)
	block, err := BuildGensisBlock("")
	assert.NotNil(block)
	assert.Nil(err)
}

// test build genesis block from config file
func TestBuildGensisBlockFromFile(t *testing.T) {
	assert := assert.New(t)
	block, err := BuildGensisBlock("./genesis.json")
	assert.NotNil(block)
	assert.Nil(err)
	assert.Equal(block.GenesisAccounts[1].Code, []byte{0, 1, 1, 0})
}
