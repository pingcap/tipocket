package util

import (
	"testing"

	"github.com/alecthomas/assert"
)

func TestRun(t *testing.T) {
	cli, err := NewDockerExecutor("tcp://172.16.4.182:2375")
	assert.NoError(t, err)
	stdout, stderr, err := cli.Run("alpine", "ping", "-c", "2", "8.8.8.8")
	assert.NoError(t, err)

	assert.NotEmpty(t, stdout)
	assert.Empty(t, stderr)
}
