package executor

import "github.com/Yu-Jack/sim-gui/pkg/docker"

type ContainerExecutor struct {
	client        *docker.Client
	containerName string
}

func NewContainerExecutor(client *docker.Client, containerName string) *ContainerExecutor {
	return &ContainerExecutor{
		client:        client,
		containerName: containerName,
	}
}

func (e *ContainerExecutor) Exec(command []string, env []string) (string, string, error) {
	return e.client.ExecContainer(e.containerName, command, env)
}
