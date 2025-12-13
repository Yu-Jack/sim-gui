package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Yu-Jack/sim-gui/pkg/docker"
	"github.com/Yu-Jack/sim-gui/pkg/server/store"
	"github.com/Yu-Jack/sim-gui/pkg/updater"
)

type Server struct {
	store   store.Storage
	dataDir string
	docker  *docker.Client
	cleaner *docker.Cleaner
	updater *updater.Updater
}

func NewServer(store store.Storage, dataDir string, upd *updater.Updater) (*Server, error) {
	cli, err := docker.NewClient(context.Background())
	if err != nil {
		return nil, err
	}

	// Pull code-server image
	if err := cli.PullImage("codercom/code-server:latest"); err != nil {
		fmt.Printf("Failed to pull code-server image: %v\n", err)
	}

	if err := cli.PullImage("rancher/support-bundle-kit:master-head"); err != nil {
		fmt.Printf("Failed to pull support-bundle-kit image: %v\n", err)
	}

	cleaner := docker.NewCleaner(cli)

	return &Server{
		store:   store,
		dataDir: dataDir,
		docker:  cli,
		cleaner: cleaner,
		updater: upd,
	}, nil
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/workspaces", s.handleListWorkspaces)
	mux.HandleFunc("POST /api/workspaces", s.handleCreateWorkspace)
	mux.HandleFunc("GET /api/workspaces/{name}", s.handleGetWorkspace)
	mux.HandleFunc("DELETE /api/workspaces/{name}", s.handleDeleteWorkspace)
	mux.HandleFunc("PUT /api/workspaces/{name}", s.handleRenameWorkspace)
	mux.HandleFunc("GET /api/workspaces/{name}/kubeconfig", s.handleExportWorkspaceKubeconfig)
	mux.HandleFunc("POST /api/workspaces/{name}/clean-all", s.handleCleanAllWorkspaceImages)
	mux.HandleFunc("POST /api/clean-all", s.handleCleanAllImages)
	mux.HandleFunc("POST /api/workspaces/{name}/resource-history", s.handleGetResourceHistory)
	mux.HandleFunc("GET /api/workspaces/{name}/namespaces", s.handleGetNamespaces)
	mux.HandleFunc("GET /api/workspaces/{name}/resource-types", s.handleGetResourceTypes)
	mux.HandleFunc("GET /api/workspaces/{name}/resources", s.handleGetResources)

	mux.HandleFunc("POST /api/workspaces/{name}/versions", s.handleUploadVersion)
	mux.HandleFunc("POST /api/workspaces/{name}/versions/{versionID}/start", s.handleStartSimulator)
	mux.HandleFunc("POST /api/workspaces/{name}/versions/{versionID}/stop", s.handleStopSimulator)
	mux.HandleFunc("GET /api/workspaces/{name}/versions/{versionID}/status", s.handleGetSimulatorStatus)
	mux.HandleFunc("GET /api/workspaces/{name}/versions/{versionID}/kubeconfig", s.handleGetKubeconfig)
	mux.HandleFunc("DELETE /api/workspaces/{name}/versions/{versionID}", s.handleDeleteVersion)
	mux.HandleFunc("POST /api/workspaces/{name}/versions/{versionID}/clean-image", s.handleCleanVersionImage)

	mux.HandleFunc("POST /api/workspaces/{name}/versions/{versionID}/code-server", s.handleStartCodeServer)

	// Update check endpoint
	mux.HandleFunc("GET /api/update-status", s.handleGetUpdateStatus)
}
