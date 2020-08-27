package util

import (
	"testing"

	"github.com/alecthomas/assert"
	"github.com/docker/docker/client"
)

func TestRun(t *testing.T) {
	cli, err := NewDockerExecutor(client.DefaultDockerHost)
	assert.NoError(t, err)
	stdout, stderr, err := cli.Run("alpine", map[string]string{}, "ping", "-c", "2", "8.8.8.8")
	assert.NoError(t, err)

	assert.NotEmpty(t, stdout)
	assert.Empty(t, stderr)
}
