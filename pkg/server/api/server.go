package api

import (
	"context"
	"net/http"

	"github.com/ibrokethecloud/sim-cli/pkg/docker"
	"github.com/ibrokethecloud/sim-cli/pkg/server/store"
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
	mux.HandleFunc("PUT /api/workspaces/{name}", s.handleRenameWorkspace)
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

	mux.HandleFunc("POST /api/workspaces/{name}/versions/{versionID}/code-server", s.handleStartCodeServer)
}
