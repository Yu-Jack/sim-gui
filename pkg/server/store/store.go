package store

import "github.com/ibrokethecloud/sim-cli/pkg/server/model"

type Storage interface {
	CreateWorkspace(workspace model.Workspace) error
	ListWorkspaces() ([]model.Workspace, error)
	GetWorkspace(name string) (*model.Workspace, error)
	UpdateWorkspace(workspace model.Workspace) error
	DeleteWorkspace(name string) error
}
