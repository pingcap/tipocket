package util

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/appleboy/easyssh-proxy"
	"go.uber.org/zap"
)

// SSHConfig ...
type SSHConfig struct {
	Host       string // hostname of the SSH server
	Port       uint   // port of the SSH server
	User       string // username to login to the SSH server
	Password   string // password of the user
	KeyFile    string // path to the private key file
	Passphrase string // passphrase of the private key file
	// Timeout is the maximum wait time for the TCP connection to establish
	Timeout time.Duration
}

// SSHExecutor ...
type SSHExecutor struct {
	Config *easyssh.MakeConfig
}

// Execute executes a command on a remote host
func (ssh *SSHExecutor) Execute(cmd string, timeout ...time.Duration) ([]byte, []byte, error) {
	cmd = fmt.Sprintf("export LANG=C; PATH=$PATH:/usr/bin:/usr/sbin %s", cmd)
	if len(timeout) == 0 {
		timeout = append(timeout, 60*time.Second)
	}

	stdout, stderr, done, err := ssh.Config.Run(cmd, timeout...)
	zap.L().Debug("SSHCommand",
		zap.String("user", ssh.Config.User),
		zap.String("host", ssh.Config.Server),
		zap.String("port", ssh.Config.Port),
		zap.String("cmd", cmd),
		zap.String("stdout", stdout),
		zap.String("stderr", stderr),
		zap.Bool("done", done),
		zap.Error(err))

	if err != nil {
		errMsg := fmt.Sprintf("SSHCommand \"%s\" on host %s failed: %v\n", cmd, ssh.Config.Server, err)
		if len(stdout) != 0 {
			errMsg = fmt.Sprintf("%sstdout:\n%s\n", errMsg, stdout)
		}
		if len(stderr) != 0 {
			errMsg = fmt.Sprintf("%sstderr:\n%s\n", errMsg, stderr)
		}
		return []byte(stdout), []byte(stderr), fmt.Errorf(errMsg)
	}
	if !done {
		return []byte(stdout), []byte(stderr), fmt.Errorf("SSHCommand \"%s\" on host %s timeout",
			cmd, ssh.Config.Server,
		)
	}
	return []byte(stdout), []byte(stderr), nil
}

// NewSSHExecutor creates a ssh executor
func NewSSHExecutor(c SSHConfig) SSHExecutor {
	if c.Port == 0 {
		c.Port = 22
	}
	if c.Timeout == 0 {
		c.Timeout = 5 * time.Second
	}
	e := SSHExecutor{Config: &easyssh.MakeConfig{
		User:    c.User,
		Server:  c.Host,
		Port:    strconv.Itoa(int(c.Port)),
		Timeout: c.Timeout,
	}}
	if c.Passphrase != "" {
		e.Config.Passphrase = c.Passphrase
	} else if c.KeyFile != "" {
		e.Config.KeyPath = c.KeyFile
		e.Config.Passphrase = c.Passphrase
	}
	return e
}

// SSHKeyPath ...
func SSHKeyPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/.ssh/id_rsa", homeDir)
}
