package core

import (
	"fmt"

	"github.com/Yu-Jack/sim-gui/pkg/docker"
	"github.com/Yu-Jack/sim-gui/pkg/server/model"
	"github.com/Yu-Jack/sim-gui/pkg/server/store"
)

// Cleaner handles cleanup operations for containers and images
type Cleaner struct {
	docker *docker.Client
	store  store.Storage
}

// NewCleaner creates a new Cleaner instance
func NewCleaner(docker *docker.Client, store store.Storage) *Cleaner {
	return &Cleaner{
		docker: docker,
		store:  store,
	}
}

// CleanVersionResult represents the result of cleaning a single version
type CleanVersionResult struct {
	VersionID string
	Error     error
}

// CleanVersion cleans containers and images for a specific version and resets its ready state
func (c *Cleaner) CleanVersion(workspaceName, versionID string) error {
	instanceName := fmt.Sprintf("%s-%s", workspaceName, versionID)

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

	// Reset ready state
	if err := c.resetVersionReadyState(workspaceName, versionID); err != nil {
		return fmt.Errorf("failed to reset ready state: %w", err)
	}

	return nil
}

// CleanAllVersionsInWorkspace cleans all versions in a workspace
func (c *Cleaner) CleanAllVersionsInWorkspace(workspaceName string) []CleanVersionResult {
	ws, err := c.store.GetWorkspace(workspaceName)
	if err != nil {
		return []CleanVersionResult{{VersionID: "workspace", Error: err}}
	}

	results := make([]CleanVersionResult, 0, len(ws.Versions))
	for _, version := range ws.Versions {
		err := c.CleanVersion(workspaceName, version.ID)
		results = append(results, CleanVersionResult{
			VersionID: version.ID,
			Error:     err,
		})
	}

	return results
}

// CleanAllWorkspaces cleans all versions across all workspaces
func (c *Cleaner) CleanAllWorkspaces() []CleanVersionResult {
	workspaces, err := c.store.ListWorkspaces()
	if err != nil {
		return []CleanVersionResult{{VersionID: "all", Error: err}}
	}

	var results []CleanVersionResult
	for _, ws := range workspaces {
		wsResults := c.CleanAllVersionsInWorkspace(ws.Name)
		results = append(results, wsResults...)
	}

	return results
}

// resetVersionReadyState resets the ready state for a version
func (c *Cleaner) resetVersionReadyState(workspaceName, versionID string) error {
	ws, err := c.store.GetWorkspace(workspaceName)
	if err != nil {
		return err
	}

	updated := false
	for i, v := range ws.Versions {
		if v.ID == versionID && v.Ready {
			ws.Versions[i].Ready = false
			updated = true
			break
		}
	}

	if updated {
		if err := c.store.UpdateWorkspace(*ws); err != nil {
			return err
		}
	}

	return nil
}

// MarkVersionReady marks a version as ready
func (c *Cleaner) MarkVersionReady(workspaceName, versionID string) error {
	ws, err := c.store.GetWorkspace(workspaceName)
	if err != nil {
		return err
	}

	updated := false
	for i, v := range ws.Versions {
		if v.ID == versionID && !v.Ready {
			ws.Versions[i].Ready = true
			updated = true
			break
		}
	}

	if updated {
		if err := c.store.UpdateWorkspace(*ws); err != nil {
			return err
		}
	}

	return nil
}

// FormatCleanResults formats clean results into error messages
func FormatCleanResults(results []CleanVersionResult) []string {
	var errors []string
	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("Version %s: %v", result.VersionID, result.Error))
		}
	}
	return errors
}

// HasVersionInWorkspace checks if a version exists in a workspace
func HasVersionInWorkspace(ws *model.Workspace, versionID string) bool {
	for _, v := range ws.Versions {
		if v.ID == versionID {
			return true
		}
	}
	return false
}
