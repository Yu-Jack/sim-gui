package jsonstore

import (
	"encoding/json"

	"os"
	"path/filepath"
	"sync"

	"github.com/ibrokethecloud/sim-cli/pkg/server/model"
)

type JSONStore struct {
	filePath string
	mu       sync.RWMutex
	data     map[string]model.Workspace
}

func NewJSONStore(path string) (*JSONStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &JSONStore{
		filePath: path,
		data:     make(map[string]model.Workspace),
	}

	// Load existing data if file exists
	if _, err := os.Stat(path); err == nil {
		if err := s.load(); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *JSONStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(file, &s.data)
}

func (s *JSONStore) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *JSONStore) CreateWorkspace(ws model.Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[ws.Name]; exists {
		return os.ErrExist
	}
	s.data[ws.Name] = ws
	return s.save()
}

func (s *JSONStore) ListWorkspaces() ([]model.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]model.Workspace, 0, len(s.data))
	for _, ws := range s.data {
		list = append(list, ws)
	}
	return list, nil
}

func (s *JSONStore) GetWorkspace(name string) (*model.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ws, exists := s.data[name]
	if !exists {
		return nil, os.ErrNotExist
	}
	return &ws, nil
}

func (s *JSONStore) UpdateWorkspace(ws model.Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[ws.Name]; !exists {
		return os.ErrNotExist
	}
	s.data[ws.Name] = ws
	return s.save()
}

func (s *JSONStore) DeleteWorkspace(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[name]; !exists {
		return os.ErrNotExist
	}
	delete(s.data, name)
	return s.save()
}
