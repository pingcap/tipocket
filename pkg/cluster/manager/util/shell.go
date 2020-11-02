package util

import (
	"fmt"
	"net/url"
	"os/exec"
	"path"
	"strings"

	"go.uber.org/zap"

	"github.com/juju/errors"
)

// CommandError saves command execution error
type CommandError struct {
	WorkDir string
	Cmd     string
	Args    []string
	Err     error
	Output  string
}

func (c *CommandError) Error() string {
	return fmt.Sprintf("run cmd(on %s): %s, args: %v failed, err: %s\noutput:\n%s", c.WorkDir, c.Cmd, c.Args, c.Err, c.Output)
}

// Command ...
func Command(workDir, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		err = &CommandError{WorkDir: workDir, Cmd: command, Args: args, Err: err, Output: string(out)}
		zap.L().Error("run cmd failed", zap.String("cmd", command), zap.Strings("args", args), zap.Error(err))
		return string(out), err
	}
	zap.L().Debug("run cmd succeed", zap.String("cmd", command), zap.Strings("args", args), zap.String("output", string(out)))
	return string(out), nil
}

// Wget ...
func Wget(rawURL string, dest string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if len(dest) == 0 {
		dest = "."
	}
	fileName := path.Base(u.Path)
	filePath := path.Join(dest, fileName)
	if _, err := Command(dest, "mkdir", "-p", dest); err != nil {
		return "", errors.Trace(err)
	}
	if _, err := Command(dest, "wget", "--tries", "20", "--waitretry", "60",
		"--retry-connrefused", "--dns-timeout", "60", "--connect-timeout", "60",
		"--read-timeout", "60", "--no-clobber", "--no-verbose", "--directory-prefix", dest, rawURL); err != nil {
		return "", errors.Trace(err)
	}
	return filePath, err
}

// MkDir ...
func MkDir(dir string) error {
	_, err := Command("", "mkdir", "-p", dir)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Unzip unzips a archive file
func Unzip(filePath, destDir string) (err error) {
	if strings.HasSuffix(filePath, ".zip") {
		_, err = Command("", "unzip", "-d", destDir, filePath)
	} else if strings.HasSuffix(filePath, ".tar.gz") {
		_, err = Command("", "tar", "-xzf", filePath, "-C", destDir)

	} else {
		_, err = Command("", "tar", "-xf", filePath, "-C", destDir)
	}
	return errors.Trace(err)
}
