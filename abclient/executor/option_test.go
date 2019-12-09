package executor


import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestOptionClone(t *testing.T) {
	option := &Option{}
	optionClone := option.Clone()

	option.Clear = false
	optionClone.Clear = true
	assert.NotEqual(t, *option, *optionClone)
}
