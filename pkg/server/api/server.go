package api

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ibrokethecloud/sim-cli/pkg/docker"
	"github.com/ibrokethecloud/sim-cli/pkg/kubeconfig"
	"github.com/ibrokethecloud/sim-cli/pkg/server/model"
	"github.com/ibrokethecloud/sim-cli/pkg/server/store"
	"k8s.io/client-go/tools/clientcmd"
)

type Server struct {
	store   store.Storage
	dataDir string
	docker  *docker.Client
}

func NewServer(store store.Storage, dataDir string) (*Server, error) {
	cli, err := docker.NewClient(context.Background())
	if err != nil {
		return nil, err
	}
	return &Server{
		store:   store,
		dataDir: dataDir,
		docker:  cli,
	}, nil
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/workspaces", s.handleListWorkspaces)
	mux.HandleFunc("POST /api/workspaces", s.handleCreateWorkspace)
	mux.HandleFunc("GET /api/workspaces/{name}", s.handleGetWorkspace)
	mux.HandleFunc("POST /api/workspaces/{name}/versions", s.handleUploadVersion)
	mux.HandleFunc("POST /api/workspaces/{name}/versions/{versionID}/start", s.handleStartSimulator)
	mux.HandleFunc("POST /api/workspaces/{name}/versions/{versionID}/stop", s.handleStopSimulator)
	mux.HandleFunc("GET /api/workspaces/{name}/versions/{versionID}/status", s.handleGetSimulatorStatus)
	mux.HandleFunc("GET /api/workspaces/{name}/versions/{versionID}/kubeconfig", s.handleGetKubeconfig)
	mux.HandleFunc("DELETE /api/workspaces/{name}/versions/{versionID}", s.handleDeleteVersion)
	mux.HandleFunc("POST /api/workspaces/{name}/resource-history", s.handleGetResourceHistory)
	mux.HandleFunc("GET /api/workspaces/{name}/namespaces", s.handleGetNamespaces)
	mux.HandleFunc("GET /api/workspaces/{name}/resource-types", s.handleGetResourceTypes)
	mux.HandleFunc("GET /api/workspaces/{name}/resources", s.handleGetResources)
}

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
		Name:      req.Name,
		CreatedAt: time.Now(),
		Versions:  []model.Version{},
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

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(ws)
}

func (s *Server) handleUploadVersion(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100 MB max memory
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}

	// Create version ID
	versionID := fmt.Sprintf("v%d", len(ws.Versions)+1)
	versionPath := filepath.Join(s.dataDir, "workspaces", name, versionID)

	if err := os.MkdirAll(versionPath, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var bundlePath string
	var bundleName string

	if len(files) == 1 {
		fileHeader := files[0]
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		bundleName = filepath.Base(fileHeader.Filename)
		bundlePath = filepath.Join(versionPath, bundleName)
		destFile, err := os.Create(bundlePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Sort files by filename to ensure correct order
		sort.Slice(files, func(i, j int) bool {
			return files[i].Filename < files[j].Filename
		})

		// Use a generic name for combined bundle
		bundleName = "bundle.zip"
		bundlePath = filepath.Join(versionPath, bundleName)

		destFile, err := os.Create(bundlePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer destFile.Close()

		for _, fileHeader := range files {
			f, err := fileHeader.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Copy content
			if _, err := io.Copy(destFile, f); err != nil {
				f.Close()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			f.Close()
		}
	}

	// Extract
	extractPath := filepath.Join(versionPath, "extracted")
	if err := os.MkdirAll(extractPath, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := unzip(bundlePath, extractPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to extract: %v", err), http.StatusInternalServerError)
		return
	}

	// Create Version
	version := model.Version{
		ID:                versionID,
		Name:              versionID, // Default name
		CreatedAt:         time.Now(),
		SupportBundleName: bundleName,
		BundlePath:        bundlePath,
	}

	ws.Versions = append(ws.Versions, version)
	if err := s.store.UpdateWorkspace(*ws); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleStartSimulator(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versionID := r.PathValue("versionID")

	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var version *model.Version
	for _, v := range ws.Versions {
		if v.ID == versionID {
			version = &v
			break
		}
	}

	if version == nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	instanceName := fmt.Sprintf("%s-%s", name, versionID)

	// Check if exists (running or stopped)
	containers, err := s.docker.FindContainer(instanceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(containers) > 0 {
		container := containers[0]
		if container.State == "running" {
			// Already running
			if !version.Ready {
				s.monitorReadyState(name, versionID, instanceName)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		// Stopped, try to start
		if err := s.docker.StartContainer(container.ID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to start existing container: %v", err), http.StatusInternalServerError)
			return
		}
		if !version.Ready {
			s.monitorReadyState(name, versionID, instanceName)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	// Create Image
	baseImage := "rancher/support-bundle-kit:master-head"
	if err := s.docker.CreateImage(instanceName, version.BundlePath, baseImage); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create image: %v", err), http.StatusInternalServerError)
		return
	}

	// Run Container
	if err := s.docker.RunContainer(instanceName, version.BundlePath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to run container: %v", err), http.StatusInternalServerError)
		return
	}

	// Monitor ready state
	if !version.Ready {
		s.monitorReadyState(name, versionID, instanceName)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStopSimulator(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versionID := r.PathValue("versionID")
	instanceName := fmt.Sprintf("%s-%s", name, versionID)

	if err := s.docker.StopContainer(instanceName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop container: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetSimulatorStatus(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versionID := r.PathValue("versionID")
	instanceName := fmt.Sprintf("%s-%s", name, versionID)

	containers, err := s.docker.FindRunningContainer(instanceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ready bool
	for _, v := range ws.Versions {
		if v.ID == versionID {
			ready = v.Ready
			break
		}
	}

	status := struct {
		Running bool `json:"running"`
		Ready   bool `json:"ready"`
	}{
		Running: len(containers) > 0,
		Ready:   ready,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleGetKubeconfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versionID := r.PathValue("versionID")
	instanceName := fmt.Sprintf("%s-%s", name, versionID)

	// Check if running
	containers, err := s.docker.FindRunningContainer(instanceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(containers) == 0 {
		http.Error(w, "Simulator not running", http.StatusConflict)
		return
	}

	// Read kubeconfig
	content, err := s.docker.ReadFile(instanceName, "/root/.sim/admin.kubeconfig")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read kubeconfig: %v", err), http.StatusInternalServerError)
		return
	}

	// Update endpoint
	endpoint, port, err := s.docker.QueryExposedMapping(instanceName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query exposed mapping: %v", err), http.StatusInternalServerError)
		return
	}

	config, err := kubeconfig.ConfigureKubeConfig(content, instanceName, endpoint, port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := clientcmd.Write(*config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.kubeconfig\"", instanceName))
	w.Write(data)
}

func (s *Server) handleDeleteVersion(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versionID := r.PathValue("versionID")

	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var versionIndex = -1
	for i, v := range ws.Versions {
		if v.ID == versionID {
			versionIndex = i
			break
		}
	}

	if versionIndex == -1 {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	// Remove files
	versionPath := filepath.Join(s.dataDir, "workspaces", name, versionID)
	if err := os.RemoveAll(versionPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove files: %v", err), http.StatusInternalServerError)
		return
	}

	// Remove container and image if exists
	instanceName := fmt.Sprintf("%s-%s", name, versionID)

	// Remove container first
	if err := s.docker.RemoveContainer(instanceName); err != nil {
		// Log error but continue to cleanup images and files
		fmt.Printf("Failed to remove container %s: %v\n", instanceName, err)
	}

	// Remove images
	_ = s.docker.RemoveImages(instanceName)

	// Update workspace
	ws.Versions = append(ws.Versions[:versionIndex], ws.Versions[versionIndex+1:]...)

	if err := s.store.UpdateWorkspace(*ws); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) markVersionReady(workspaceName, versionID string) {
	ws, err := s.store.GetWorkspace(workspaceName)
	if err != nil {
		fmt.Printf("Failed to get workspace to mark ready: %v\n", err)
		return
	}

	updated := false
	for i, v := range ws.Versions {
		if v.ID == versionID {
			if !v.Ready {
				ws.Versions[i].Ready = true
				updated = true
			}
			break
		}
	}

	if updated {
		if err := s.store.UpdateWorkspace(*ws); err != nil {
			fmt.Printf("Failed to update workspace ready status: %v\n", err)
		}
	}
}

func (s *Server) monitorReadyState(workspaceName, versionID, instanceName string) {
	go func() {
		if err := s.docker.WaitForLogMessage(instanceName, "All resources loaded successfully"); err == nil {
			s.markVersionReady(workspaceName, versionID)
		} else {
			fmt.Printf("Monitor ready state failed: %v\n", err)
		}
	}()
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

		stdout, stderr, err := s.execKubectl(instanceName, args...)

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

	instanceName, err := s.findLatestRunningInstance(name, ws)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	stdout, _, err := s.execKubectl(instanceName, "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}")
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

	instanceName, err := s.findLatestRunningInstance(name, ws)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	stdout, _, err := s.execKubectl(instanceName, "api-resources", "--verbs=list", "-o", "name")
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

	instanceName, err := s.findLatestRunningInstance(name, ws)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	stdout, _, err := s.execKubectl(instanceName, "get", resourceType, "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	allResources := strings.Split(strings.TrimSpace(stdout), " ")
	var filtered []string
	for _, res := range allResources {
		if res == "" {
			continue
		}
		if keyword == "" || strings.Contains(res, keyword) {
			filtered = append(filtered, res)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

func (s *Server) findLatestRunningInstance(name string, ws *model.Workspace) (string, error) {
	for i := len(ws.Versions) - 1; i >= 0; i-- {
		v := ws.Versions[i]
		iname := fmt.Sprintf("%s-%s", name, v.ID)
		containers, err := s.docker.FindRunningContainer(iname)
		if err == nil && len(containers) > 0 {
			return iname, nil
		}
	}
	return "", fmt.Errorf("no running simulator found")
}

func (s *Server) execKubectl(instanceName string, args ...string) (string, string, error) {
	cmd := append([]string{"kubectl"}, args...)
	env := []string{"KUBECONFIG=/root/.sim/admin.kubeconfig"}
	return s.docker.ExecContainer(instanceName, cmd, env)
}
