package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Yu-Jack/sim-gui/pkg/kubeconfig"
	"github.com/Yu-Jack/sim-gui/pkg/server/model"
	"github.com/Yu-Jack/sim-gui/pkg/server/utils"
	"k8s.io/client-go/tools/clientcmd"
)

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

	if err := utils.Unzip(bundlePath, extractPath); err != nil {
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

func (s *Server) handleCleanVersionImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versionID := r.PathValue("versionID")
	instanceName := fmt.Sprintf("%s-%s", name, versionID)

	// Check if container is running
	containers, err := s.docker.FindRunningContainer(instanceName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check container status: %v", err), http.StatusInternalServerError)
		return
	}

	if len(containers) > 0 {
		http.Error(w, "Cannot clean image while simulator is running. Please stop the simulator first.", http.StatusBadRequest)
		return
	}

	// Remove stopped containers first to avoid conflicts
	if err := s.docker.RemoveContainer(instanceName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove stopped containers: %v", err), http.StatusInternalServerError)
		return
	}

	// Remove the Docker image
	if err := s.docker.RemoveImages(instanceName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove image: %v", err), http.StatusInternalServerError)
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

	// Cleanup code-server directory
	codeServerContainer := "sim-cli-code-server"
	targetDir := fmt.Sprintf("/home/coder/project/%s-%s", name, versionID)
	if _, _, err := s.docker.ExecContainer(codeServerContainer, []string{"rm", "-rf", targetDir}, nil); err != nil {
		fmt.Printf("Failed to cleanup code-server directory: %v\n", err)
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
