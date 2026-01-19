package userdata

import (
	"fmt"
	"os"
	"path/filepath"
)

const DirName = "userdata"

// Manager handles stored user-data scripts.
type Manager struct {
	basePath string
}

// NewManager creates a new Manager, ensuring the storage directory exists.
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}

	path := filepath.Join(home, ".config", "privatebox", DirName)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create userdata dir: %w", err)
	}

	return &Manager{basePath: path}, nil
}

func (m *Manager) getFilePath(name string) string {
	return filepath.Join(m.basePath, name)
}

// Create stores a script.
func (m *Manager) Create(name string, content []byte) error {
	path := m.getFilePath(name)
	// Check if exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("userdata script '%s' already exists", name)
	}

	return os.WriteFile(path, content, 0644)
}

// List returns the names of stored scripts.
func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Delete removes a stored script.
func (m *Manager) Delete(name string) error {
	path := m.getFilePath(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("userdata script '%s' not found", name)
	}
	return os.Remove(path)
}

// Get returns the content of a stored script.
func (m *Manager) Get(name string) ([]byte, error) {
	path := m.getFilePath(name)
	return os.ReadFile(path)
}

// Exists checks if a script exists.
func (m *Manager) Exists(name string) bool {
	_, err := os.Stat(m.getFilePath(name))
	return err == nil
}
