package util

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/juju/errors"
	"go.uber.org/zap"
)

type DockerExecutor struct {
	*client.Client
}

func NewDockerExecutor(host string) (*DockerExecutor, error) {
	cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Trace(err)
	}
	d := &DockerExecutor{cli}
	header, err := d.Ping(context.Background())
	if err != nil {
		return nil, err
	}
	zap.L().Debug("docker cli", zap.String("api version", header.APIVersion))
	return &DockerExecutor{cli}, nil
}

func (d *DockerExecutor) Run(dockerImage string, envs map[string]string, cmd string, args ...string) ([]byte, []byte, error) {
	ctx := context.Background()
	var env []string
	for key, value := range envs {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	reader, err := d.ImagePull(ctx, dockerImage, types.ImagePullOptions{})
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	defer reader.Close()

	resp, err := d.ContainerCreate(ctx, &container.Config{
		Image: dockerImage,
		Env:   env,
		Cmd:   append([]string{cmd}, args...),
	}, nil, nil, "")

	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	if err := d.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, nil, errors.Trace(err)
	}
	statusCh, errCh := d.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
	case <-statusCh:
	}

	out, err := d.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	var stdOutBuffer bytes.Buffer
	var stdErrBuffer bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdOutBuffer, &stdErrBuffer, out); err != nil {
		return nil, nil, errors.Trace(err)
	}

	return stdOutBuffer.Bytes(), stdErrBuffer.Bytes(), nil
}
