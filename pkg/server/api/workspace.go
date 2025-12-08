package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Yu-Jack/sim-gui/pkg/core"
	"github.com/Yu-Jack/sim-gui/pkg/server/model"
	"github.com/Yu-Jack/sim-gui/pkg/server/utils"
)

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := s.store.ListWorkspaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workspaces)
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Workspace name cannot be empty", http.StatusBadRequest)
		return
	}

	ws := model.Workspace{
		Name:        req.Name,
		DisplayName: req.Name,
		CreatedAt:   time.Now(),
		Versions:    []model.Version{},
	}

	if err := s.store.CreateWorkspace(ws); err != nil {
		if os.IsExist(err) {
			http.Error(w, "Workspace already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)
}

func (s *Server) handleRenameWorkspace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newDisplayName := req.Name
	if strings.TrimSpace(newDisplayName) == "" {
		http.Error(w, "New workspace name cannot be empty", http.StatusBadRequest)
		return
	}

	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ws.DisplayName = newDisplayName
	if err := s.store.UpdateWorkspace(*ws); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(ws)
}

func (s *Server) handleCleanAllWorkspaceImages(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	// Use cleaner to clean all versions and reset ready states
	results := s.cleaner.CleanAllVersionsInWorkspace(name)
	errors := core.FormatCleanResults(results)

	if len(errors) > 0 {
		http.Error(w, fmt.Sprintf("Some operations failed: %v", strings.Join(errors, "; ")), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCleanAllImages(w http.ResponseWriter, r *http.Request) {
	// Use cleaner to clean all workspaces and reset ready states
	results := s.cleaner.CleanAllWorkspaces()
	errors := core.FormatCleanResults(results)

	if len(errors) > 0 {
		http.Error(w, fmt.Sprintf("Some operations failed: %v", strings.Join(errors, "; ")), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetResourceHistory(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		Resource string `json:"resource"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	type VersionResult struct {
		VersionID string `json:"versionID"`
		Content   string `json:"content"`
		Error     string `json:"error,omitempty"`
		Status    string `json:"status"` // "found", "not_found", "stopped", "error"
	}

	var results []VersionResult

	for _, v := range ws.Versions {
		instanceName := fmt.Sprintf("%s-%s", name, v.ID)

		// Check if container is running
		containers, err := s.docker.FindRunningContainer(instanceName)
		if err != nil || len(containers) == 0 {
			results = append(results, VersionResult{
				VersionID: v.ID,
				Status:    "stopped",
				Error:     "Container not running",
			})
			continue
		}

		// Execute kubectl get <resource> -o yaml
		// Support format: namespace/type/name or type/name
		parts := strings.Split(req.Resource, "/")
		var args []string
		if len(parts) == 3 {
			namespace := parts[0]
			resourceType := parts[1]
			resourceName := parts[2]
			args = []string{"get", resourceType, resourceName, "-n", namespace, "-o", "yaml"}
		} else {
			args = []string{"get", req.Resource, "-o", "yaml"}
		}

		stdout, stderr, err := utils.ExecKubectl(s.docker, instanceName, args...)

		if err != nil {
			results = append(results, VersionResult{
				VersionID: v.ID,
				Status:    "error",
				Error:     err.Error(),
			})
			continue
		}

		if stderr != "" {
			results = append(results, VersionResult{
				VersionID: v.ID,
				Status:    "not_found",
				Error:     stderr,
			})
			continue
		}

		results = append(results, VersionResult{
			VersionID: v.ID,
			Status:    "found",
			Content:   stdout,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleGetNamespaces(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	instanceName, err := utils.FindLatestRunningInstance(name, ws, s.docker)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	stdout, _, err := utils.ExecKubectl(s.docker, instanceName, "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	namespaces := strings.Split(strings.TrimSpace(stdout), " ")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(namespaces)
}

func (s *Server) handleGetResourceTypes(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	instanceName, err := utils.FindLatestRunningInstance(name, ws, s.docker)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	stdout, _, err := utils.ExecKubectl(s.docker, instanceName, "api-resources", "--verbs=list", "-o", "name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resources := strings.Split(strings.TrimSpace(stdout), "\n")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

func (s *Server) handleGetResources(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	namespace := r.URL.Query().Get("namespace")
	resourceType := r.URL.Query().Get("resourceType")
	keyword := r.URL.Query().Get("keyword")

	if namespace == "" || resourceType == "" {
		http.Error(w, "namespace and resourceType are required", http.StatusBadRequest)
		return
	}

	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resourceMap := make(map[string]bool)

	for _, v := range ws.Versions {
		instanceName := fmt.Sprintf("%s-%s", name, v.ID)
		containers, err := s.docker.FindRunningContainer(instanceName)
		if err != nil || len(containers) == 0 {
			continue
		}

		stdout, _, err := utils.ExecKubectl(s.docker, instanceName, "get", resourceType, "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
		if err != nil {
			continue
		}

		resources := strings.Split(strings.TrimSpace(stdout), " ")
		for _, res := range resources {
			if res != "" {
				resourceMap[res] = true
			}
		}
	}

	var filtered []string
	for res := range resourceMap {
		if keyword == "" || strings.Contains(res, keyword) {
			filtered = append(filtered, res)
		}
	}
	sort.Strings(filtered)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Cleanup all versions
	for _, v := range ws.Versions {
		instanceName := fmt.Sprintf("%s-%s", name, v.ID)

		// Remove container
		if err := s.docker.RemoveContainer(instanceName); err != nil {
			fmt.Printf("Failed to remove container %s: %v\n", instanceName, err)
		}

		// Remove images
		_ = s.docker.RemoveImages(instanceName)

		// Cleanup code-server directory
		codeServerContainer := "sim-cli-code-server"
		targetDir := fmt.Sprintf("/home/coder/project/%s-%s", name, v.ID)
		if _, _, err := s.docker.ExecContainer(codeServerContainer, []string{"rm", "-rf", targetDir}, nil); err != nil {
			fmt.Printf("Failed to cleanup code-server directory: %v\n", err)
		}
	}

	// Remove workspace directory
	workspacePath := fmt.Sprintf("%s/workspaces/%s", s.dataDir, name)
	if err := os.RemoveAll(workspacePath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove workspace files: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete from store
	if err := s.store.DeleteWorkspace(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
