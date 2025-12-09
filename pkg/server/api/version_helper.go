package api

import (
	"fmt"

	"github.com/Yu-Jack/sim-gui/pkg/server/model"
)

// CleanVersionResult represents the result of cleaning a single version
type CleanVersionResult struct {
	VersionID string
	Error     error
}

// ResetVersionReadyState resets the ready state for a version
func (s *Server) ResetVersionReadyState(workspaceName, versionID string) error {
	ws, err := s.store.GetWorkspace(workspaceName)
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
		if err := s.store.UpdateWorkspace(*ws); err != nil {
			return err
		}
	}

	return nil
}

// MarkVersionReady marks a version as ready
func (s *Server) MarkVersionReady(workspaceName, versionID string) error {
	ws, err := s.store.GetWorkspace(workspaceName)
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
		if err := s.store.UpdateWorkspace(*ws); err != nil {
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
