package docker

import (
	"fmt"
)

// Cleaner handles cleanup operations for containers and images
type Cleaner struct {
	docker *Client
}

// NewCleaner creates a new Cleaner instance
func NewCleaner(docker *Client) *Cleaner {
	return &Cleaner{
		docker: docker,
	}
}

// CleanInstance cleans containers and images for a specific instance
func (c *Cleaner) CleanInstance(instanceName string) error {
	// Stop container if running
	if err := c.docker.StopContainer(instanceName); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove all containers (including stopped ones)
	if err := c.docker.RemoveContainer(instanceName); err != nil {
		return fmt.Errorf("failed to remove containers: %w", err)
	}

	// Remove images
	if err := c.docker.RemoveImages(instanceName); err != nil {
		return fmt.Errorf("failed to remove images: %w", err)
	}

	return nil
}
