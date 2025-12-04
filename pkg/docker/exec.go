package docker

import (
	"bytes"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
)

func (c *Client) ExecContainer(containerName string, command []string, env []string) (string, string, error) {
	execConfig := types.ExecConfig{
		Cmd:          command,
		Env:          env,
		AttachStdout: true,
		AttachStderr: true,
	}

	execIDResp, err := c.APIClient.ContainerExecCreate(c.ctx, containerName, execConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to create exec configuration: %w", err)
	}

	resp, err := c.APIClient.ContainerExecAttach(c.ctx, execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", "", fmt.Errorf("failed to attach to exec process: %w", err)
	}
	defer resp.Close()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, resp.Reader); err != nil {
		return "", "", fmt.Errorf("failed to copy output: %w", err)
	}

	inspect, err := c.APIClient.ContainerExecInspect(c.ctx, execIDResp.ID)
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("failed to inspect exec process: %w", err)
	}

	if inspect.ExitCode != 0 {
		return stdout.String(), stderr.String(), fmt.Errorf("command failed with exit code %d: %s", inspect.ExitCode, stderr.String())
	}

	return stdout.String(), stderr.String(), nil
}
