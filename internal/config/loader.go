package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName  = ".config/privatebox"
	configFileName = "config.json"
)

// Loader handles reading and writing configuration.
type Loader struct {
	configPath string
}

// NewLoader creates a new configuration loader.
func NewLoader() (*Loader, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(home, configDirName)
	configPath := filepath.Join(configDir, configFileName)

	return &Loader{
		configPath: configPath,
	}, nil
}

// Load reads the configuration from disk.
// If the file does not exist, it returns the default configuration.
func (l *Loader) Load() (*AppConfig, error) {
	if _, err := os.Stat(l.configPath); os.IsNotExist(err) {
		cfg := NewAppConfig()
		return &cfg, nil
	}

	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Try to unmarshal as AppConfig (new format)
	var appCfg AppConfig
	if err := json.Unmarshal(data, &appCfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Migration: If Profiles is empty, it might be the old format
	if appCfg.Profiles == nil {
		var oldProfile Profile
		// Try unmarshaling as a single Profile
		if err := json.Unmarshal(data, &oldProfile); err == nil {
			// If successful and looks valid (e.g. has Provider), wrap it
			if oldProfile.Provider != "" {
				appCfg = NewAppConfig()
				appCfg.Profiles["default"] = oldProfile
				return &appCfg, nil
			}
		}
	}

	// Ensure defaults if empty
	if appCfg.Profiles == nil {
		appCfg = NewAppConfig()
	}

	return &appCfg, nil
}

// Save writes the configuration to disk.
func (l *Loader) Save(cfg *AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(l.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(l.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the absolute path to the configuration file.
func (l *Loader) GetConfigPath() string {
	return l.configPath
}
