package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoader_Save(t *testing.T) {
	type testCase struct {
		name    string
		cfg     *AppConfig
		wantErr bool
	}

	tests := []testCase{
		{
			name: "Basic Config",
			cfg: &AppConfig{
				CurrentProfile: "default",
				Profiles: map[string]Profile{
					"default": {
						Provider: "aws",
						Region:   "us-east-1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Config With Env",
			cfg: &AppConfig{
				CurrentProfile: "dev",
				Profiles: map[string]Profile{
					"dev": {
						Provider: "aws",
						Region:   "us-west-1",
						Env: map[string]string{
							"FOO": "bar",
							"BAZ": "qux",
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			loader := &Loader{configPath: configPath}

			err := loader.Save(tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				// Verify file exists and is valid YAML
				content, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("Failed to read saved file: %v", err)
				}

				var loadedCfg AppConfig
				if err := yaml.Unmarshal(content, &loadedCfg); err != nil {
					t.Errorf("Saved content is not valid YAML: %v", err)
				}

				if loadedCfg.CurrentProfile != tc.cfg.CurrentProfile {
					t.Errorf("Config mismatch. Got %s, want %s", loadedCfg.CurrentProfile, tc.cfg.CurrentProfile)
				}

				// Check profiles
				if len(loadedCfg.Profiles) != len(tc.cfg.Profiles) {
					t.Errorf("Profile count mismatch. Got %d, want %d", len(loadedCfg.Profiles), len(tc.cfg.Profiles))
				}

				for name, p := range tc.cfg.Profiles {
					loadedP, ok := loadedCfg.Profiles[name]
					if !ok {
						t.Errorf("Profile %s missing", name)
						continue
					}
					if len(p.Env) != len(loadedP.Env) {
						t.Errorf("Profile %s Env count mismatch. Got %d, want %d", name, len(loadedP.Env), len(p.Env))
					}
					for k, v := range p.Env {
						if loadedP.Env[k] != v {
							t.Errorf("Profile %s Env mismatch for key %s. Got %s, want %s", name, k, loadedP.Env[k], v)
						}
					}
				}
			}
		})
	}
}

func TestLoader_Load(t *testing.T) {
	type fileState struct {
		name    string
		content []byte
	}

	type testCase struct {
		name        string
		files       []fileState // Files to create before test
		wantProfile string
		wantErr     bool
	}

	validYAML, _ := yaml.Marshal(AppConfig{
		CurrentProfile: "yaml-profile",
		Profiles:       map[string]Profile{"default": {Provider: "aws"}},
	})

	tests := []testCase{
		{
			name:        "No Config Files",
			files:       []fileState{},
			wantProfile: "", // Default empty config
			wantErr:     false,
		},
		{
			name: "YAML Only",
			files: []fileState{
				{name: "config.yaml", content: validYAML},
			},
			wantProfile: "yaml-profile",
			wantErr:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			loader := &Loader{configPath: configPath}

			for _, f := range tc.files {
				path := filepath.Join(tmpDir, f.name)
				if err := os.WriteFile(path, f.content, 0644); err != nil {
					t.Fatalf("Failed to write setup file %s: %v", f.name, err)
				}
			}

			cfg, err := loader.Load()
			if (err != nil) != tc.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr && cfg.CurrentProfile != tc.wantProfile {
				t.Errorf("Expected profile %s, got %s", tc.wantProfile, cfg.CurrentProfile)
			}
		})
	}
}
