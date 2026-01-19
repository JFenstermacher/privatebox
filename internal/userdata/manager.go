package userdata

import (
	"fmt"
	"os"
	"path/filepath"
	"privatebox/internal/config"
)

const LegacyDirName = "userdata"

// Manager handles stored user-data scripts in the config.
type Manager struct {
	loader *config.Loader
}

// NewManager creates a new Manager and migrates legacy files if present.
func NewManager() (*Manager, error) {
	loader, err := config.NewLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to create config loader: %w", err)
	}

	m := &Manager{loader: loader}

	// Attempt migration
	if err := m.migrateLegacy(); err != nil {
		// Log error but don't fail? Or fail?
		// Failing is safer so user knows something is up.
		return nil, fmt.Errorf("failed to migrate legacy userdata: %w", err)
	}

	return m, nil
}

func (m *Manager) migrateLegacy() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	legacyPath := filepath.Join(home, ".config", "privatebox", LegacyDirName)

	info, err := os.Stat(legacyPath)
	if os.IsNotExist(err) {
		return nil // No legacy dir
	}
	if !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(legacyPath)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		_ = os.Remove(legacyPath)
		return nil
	}

	// Load config to merge
	cfg, err := m.loader.Load()
	if err != nil {
		return err
	}
	if cfg.UserData == nil {
		cfg.UserData = make(map[string]string)
	}

	migrated := false
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(legacyPath, e.Name()))
		if err != nil {
			return err
		}
		// Only add if not already in config (prefer config?)
		// Or prefer legacy? Let's assume config is source of truth, but if we are migrating, we put them in.
		if _, exists := cfg.UserData[e.Name()]; !exists {
			cfg.UserData[e.Name()] = string(content)
			migrated = true
		}
	}

	if migrated {
		if err := m.loader.Save(cfg); err != nil {
			return err
		}
	}

	// Rename legacy dir to indicate migration done
	// We won't delete it to be safe, just rename
	backupPath := legacyPath + ".migrated"
	_ = os.Rename(legacyPath, backupPath)

	return nil
}

// Create stores a script in the config.
func (m *Manager) Create(name string, content []byte) error {
	cfg, err := m.loader.Load()
	if err != nil {
		return err
	}

	if cfg.UserData == nil {
		cfg.UserData = make(map[string]string)
	}

	if _, exists := cfg.UserData[name]; exists {
		return fmt.Errorf("userdata script '%s' already exists", name)
	}

	cfg.UserData[name] = string(content)
	return m.loader.Save(cfg)
}

// List returns the names of stored scripts.
func (m *Manager) List() ([]string, error) {
	cfg, err := m.loader.Load()
	if err != nil {
		return nil, err
	}

	var names []string
	for k := range cfg.UserData {
		names = append(names, k)
	}
	return names, nil
}

// Delete removes a stored script.
func (m *Manager) Delete(name string) error {
	cfg, err := m.loader.Load()
	if err != nil {
		return err
	}

	if _, exists := cfg.UserData[name]; !exists {
		return fmt.Errorf("userdata script '%s' not found", name)
	}

	delete(cfg.UserData, name)
	return m.loader.Save(cfg)
}

// Get returns the content of a stored script.
func (m *Manager) Get(name string) ([]byte, error) {
	cfg, err := m.loader.Load()
	if err != nil {
		return nil, err
	}

	content, exists := cfg.UserData[name]
	if !exists {
		return nil, fmt.Errorf("userdata script '%s' not found", name)
	}
	return []byte(content), nil
}

// Exists checks if a script exists.
func (m *Manager) Exists(name string) bool {
	cfg, err := m.loader.Load()
	if err != nil {
		return false
	}
	_, exists := cfg.UserData[name]
	return exists
}
