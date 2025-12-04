package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/bndr/gotabulate"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-connections/nat"
)

// RunContainer runs an instance of support-bundle-kit simulator in a docker container image
func (c *Client) RunContainer(instanceName, bundlePath string) error {
	imageName := fmt.Sprintf("%s:%s", simCliPrefix, instanceName)
	resp, err := c.APIClient.ContainerCreate(c.ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"support-bundle-kit", "simulator", "reset", "--bundle-path", "/bundle"},
		ExposedPorts: map[nat.Port]struct{}{
			"6443/tcp": struct{}{},
		},
		Tty: false,
		Labels: map[string]string{
			bundleNameKey: bundlePath,
			simCliPrefix:  instanceName,
		},
	}, &container.HostConfig{
		AutoRemove:  false,
		NetworkMode: "bridge",
		PortBindings: map[nat.Port][]nat.PortBinding{
			"6443/tcp": {
				{
					HostIP: "0.0.0.0",
				},
			},
		},
	},
		nil, nil, instanceName)
	if err != nil {
		return fmt.Errorf("error creating container %s: %w", instanceName, err)
	}

	// start container
	if err := c.APIClient.ContainerStart(c.ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("error starting container %s: %w", instanceName, err)
	}
	return nil
}

// FindRunningContainer attempts to find instance of simulator associated with the instanceName
func (c *Client) FindRunningContainer(instanceName string) ([]types.Container, error) {
	filters := filters.NewArgs(filters.KeyValuePair{Key: "name", Value: instanceName})
	return c.APIClient.ContainerList(c.ctx, container.ListOptions{
		Filters: filters,
	})

}

// FindContainer attempts to find instance of simulator associated with the instanceName (running or stopped)
func (c *Client) FindContainer(instanceName string) ([]types.Container, error) {
	filters := filters.NewArgs(filters.KeyValuePair{Key: "name", Value: instanceName})
	return c.APIClient.ContainerList(c.ctx, container.ListOptions{
		Filters: filters,
		All:     true,
	})
}

// StopContainer attempts to find and stop a running instance of a container associated with given instanceName
func (c *Client) StopContainer(instanceName string) error {
	containers, err := c.FindRunningContainer(instanceName)
	if err != nil {
		return fmt.Errorf("error listing containers matching name %s: %w", instanceName, err)
	}

	for _, v := range containers {
		if err := c.APIClient.ContainerStop(c.ctx, v.ID, container.StopOptions{Signal: "SIGKILL"}); err != nil {
			return err
		}
	}
	return nil
}

// StartContainer starts an existing container
func (c *Client) StartContainer(containerID string) error {
	return c.APIClient.ContainerStart(c.ctx, containerID, container.StartOptions{})
}

// QueryExposedMapping attempts to find details of host/port needed for configuring the kubeconfig needed
// to access the instance running in associated container
func (c *Client) QueryExposedMapping(instanceName string) (string, string, error) {
	var endpoint, port string
	containers, err := c.FindRunningContainer(instanceName)
	if err != nil {
		return endpoint, port, fmt.Errorf("error listing containers matching name %s: %w", instanceName, err)
	}

	if len(containers) != 1 {
		return endpoint, port, fmt.Errorf("expected one container matching name %s, got %d", instanceName, len(containers))
	}

	port = fmt.Sprintf("%d", containers[0].Ports[0].PublicPort)
	netconfig, err := url.Parse(c.Endpoint.Host)
	if err != nil {
		return endpoint, port, fmt.Errorf("error parsing endpoint info: %w", err)
	}
	endpoint = netconfig.Host
	// when using local docker sock, this will be an empty string
	if endpoint == "" {
		endpoint = "localhost"
	}
	return endpoint, port, nil
}

// FindAllSimManagedInstances returns details of all sim-cli managed instances and presents them in a tabular form
func (c *Client) FindAllSimManagedInstances() error {
	filters := filters.NewArgs(filters.KeyValuePair{Key: "label", Value: simCliPrefix})
	containers, err := c.APIClient.ContainerList(c.ctx, container.ListOptions{
		Filters: filters,
		All:     true,
	})
	if err != nil {
		return fmt.Errorf("error listing containers: %w", err)
	}

	generateTable(containers)
	return nil
}

// generateTable is a helper method to return results in a tabular form
func generateTable(containers []types.Container) {
	var results [][]interface{}

	// gotabulate does no handle empty table and panics
	// so for now we send an empty row if there is nothing returned
	if len(containers) == 0 {
		results = append(results, []interface{}{"", "", "", "", ""})
	}

	for _, v := range containers {
		name := v.Labels[simCliPrefix]
		bundlePath := v.Labels[bundleNameKey]
		image := v.Image
		status := v.Status
		port := fmt.Sprintf("%d", v.Ports[0].PublicPort)
		results = append(results, []interface{}{name, bundlePath, image, status, port})
	}
	table := gotabulate.Create(results)
	table.SetHeaders([]string{"name", "bundlePath", "image", "status", "exposed port"})
	table.SetEmptyString("None")
	table.SetAlign("right")
	table.SetMaxCellSize(40)
	table.SetWrapStrings(true)
	fmt.Println(table.Render("grid"))
}

// ReadFile will read a specific file from a running container and return the results
func (c *Client) ReadFile(name string, path string) ([]byte, error) {
	containers, err := c.FindRunningContainer(name)
	if err != nil {
		return nil, fmt.Errorf("error listing containers matching name %s: %w", name, err)
	}

	if len(containers) != 1 {
		return nil, fmt.Errorf("expected one container matching name %s, got %d", name, len(containers))
	}
	contents, _, err := c.APIClient.CopyFromContainer(c.ctx, containers[0].ID, path)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", path, err)
	}
	tr := tar.NewReader(contents)
	buf := new(bytes.Buffer)
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("error reading from tar archive: %w", err)
		}

		buf.ReadFrom(tr)
		return buf.Bytes(), nil
	}
	return nil, nil
}

// RemoveContainer attempts to find and remove a container associated with given instanceName
func (c *Client) RemoveContainer(instanceName string) error {
	// Also check for stopped containers
	filters := filters.NewArgs(filters.KeyValuePair{Key: "name", Value: instanceName})
	allContainers, err := c.APIClient.ContainerList(c.ctx, container.ListOptions{
		Filters: filters,
		All:     true,
	})
	if err != nil {
		return fmt.Errorf("error listing all containers matching name %s: %w", instanceName, err)
	}

	// Merge lists or just use allContainers
	for _, v := range allContainers {
		// Stop if running
		if v.State == "running" {
			if err := c.APIClient.ContainerStop(c.ctx, v.ID, container.StopOptions{Signal: "SIGKILL"}); err != nil {
				return fmt.Errorf("error stopping container %s: %w", v.ID, err)
			}
		}
		// Remove
		if err := c.APIClient.ContainerRemove(c.ctx, v.ID, container.RemoveOptions{Force: true}); err != nil {
			return fmt.Errorf("error removing container %s: %w", v.ID, err)
		}
	}
	return nil
}

// WaitForLogMessage tails the container logs and waits for a specific message.
func (c *Client) WaitForLogMessage(instanceName, message string) error {
	containers, err := c.FindRunningContainer(instanceName)
	if err != nil {
		return fmt.Errorf("error listing containers matching name %s: %w", instanceName, err)
	}
	if len(containers) == 0 {
		return fmt.Errorf("container %s not found", instanceName)
	}

	options := container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true}
	out, err := c.APIClient.ContainerLogs(c.ctx, containers[0].ID, options)
	if err != nil {
		return fmt.Errorf("error getting container logs: %w", err)
	}
	defer out.Close()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, message) {
			return nil
		}
	}
	return scanner.Err()
}

// RunCodeServer starts a code-server container
func (c *Client) RunCodeServer(instanceName string) (string, string, error) {
	// Check if container exists (running or stopped)
	containers, err := c.FindContainer(instanceName)
	if err != nil {
		return "", "", fmt.Errorf("error finding container: %w", err)
	}

	if len(containers) == 0 {
		// Create container
		imageName := "codercom/code-server:latest"
		// We don't explicitly pull here, assuming Docker daemon handles it or it's present.

		resp, err := c.APIClient.ContainerCreate(c.ctx, &container.Config{
			Image: imageName,
			Cmd:   []string{"--auth", "none", "--bind-addr", "0.0.0.0:8080", "/home/coder/project"},
			ExposedPorts: map[nat.Port]struct{}{
				"8080/tcp": {},
			},
			Tty: false,
			Labels: map[string]string{
				simCliPrefix: instanceName,
			},
		}, &container.HostConfig{
			AutoRemove: true,
			PortBindings: map[nat.Port][]nat.PortBinding{
				"8080/tcp": {
					{
						HostIP:   "0.0.0.0",
						HostPort: "0", // Random port
					},
				},
			},
		}, nil, nil, instanceName)
		if err != nil {
			return "", "", fmt.Errorf("error creating code-server container: %w", err)
		}

		if err := c.APIClient.ContainerStart(c.ctx, resp.ID, container.StartOptions{}); err != nil {
			return "", "", fmt.Errorf("error starting code-server container: %w", err)
		}
	} else {
		// Container exists
		containerID := containers[0].ID
		if containers[0].State != "running" {
			if err := c.StartContainer(containerID); err != nil {
				return "", "", fmt.Errorf("error starting existing code-server container: %w", err)
			}
		}
	}

	// Inspect to get the port
	inspect, err := c.APIClient.ContainerInspect(c.ctx, instanceName)
	if err != nil {
		return "", "", fmt.Errorf("error inspecting container: %w", err)
	}

	bindings := inspect.NetworkSettings.Ports["8080/tcp"]
	if len(bindings) > 0 {
		return fmt.Sprintf("http://localhost:%s", bindings[0].HostPort), inspect.ID, nil
	}

	return "", "", fmt.Errorf("failed to get exposed port for code-server")
}
