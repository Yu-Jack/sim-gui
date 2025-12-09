package docker

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

const (
	bundleNameKey     = "harvesterhci.io/bundle-name"
	simKubeConfigPath = "/root/.sim/admin.kubeconfig"
)

type Client struct {
	APIClient   client.APIClient
	Endpoint    docker.Endpoint
	ctx         context.Context
	buildWorker *ImageBuildWorker
}

// GetClient leverages dockerCli to handle interaction with the docker client
func GetClient() (*command.DockerCli, error) {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return nil, fmt.Errorf("failed to create new docker CLI with standard streams: %w", err)
	}

	newClientOpts := flags.NewClientOptions()
	newClientOpts.LogLevel = logrus.GetLevel().String()

	flagset := pflag.NewFlagSet("docker", pflag.ContinueOnError)
	newClientOpts.InstallFlags(flagset)
	newClientOpts.SetDefaultOptions(flagset)

	err = dockerCli.Initialize(newClientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker CLI: %v", err)
	}

	return dockerCli, nil
}

// NewClient initialises a new client for interacting with dockerd
func NewClient(ctx context.Context) (*Client, error) {
	dockerCli, err := GetClient()
	if err != nil {
		return nil, err
	}
	c := &Client{
		APIClient: dockerCli.Client(),
		Endpoint:  dockerCli.DockerEndpoint(),
		ctx:       ctx,
	}

	// Initialize and start the build worker
	c.buildWorker = NewImageBuildWorker(c)
	c.buildWorker.Start()

	return c, nil
}

// Close gracefully closes the client and shuts down the build worker
func (c *Client) Close() {
	if c.buildWorker != nil {
		c.buildWorker.Shutdown()
	}
}
