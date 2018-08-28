package genesis

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_BuildGensisBlock(t *testing.T) {
	assert := assert.New(t)
	block, err := BuildGensisBlock()
	assert.NotNil(block)
	assert.Nil(err)
}
