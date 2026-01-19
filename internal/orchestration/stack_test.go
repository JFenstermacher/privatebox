package orchestration

import (
	"privatebox/internal/config"
	"testing"
)

func TestStackManager_getEnv(t *testing.T) {
	tests := []struct {
		name         string
		backend      string
		instanceName string
		wantBackend  string
	}{
		{
			name:         "File backend",
			backend:      "file://~/.privatebox/state",
			instanceName: "dev1",
			wantBackend:  "file://~/.privatebox/state/dev1",
		},
		{
			name:         "File backend with trailing slash",
			backend:      "file:///tmp/state/",
			instanceName: "dev2",
			wantBackend:  "file:///tmp/state/dev2",
		},
		{
			name:         "S3 backend",
			backend:      "s3://my-bucket",
			instanceName: "dev1",
			wantBackend:  "s3://my-bucket",
		},
		{
			name:         "Managed backend",
			backend:      "https://api.pulumi.com",
			instanceName: "dev1",
			wantBackend:  "https://api.pulumi.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Profile{
				PulumiBackend: tt.backend,
				Region:        "us-east-1",
			}
			s := &StackManager{
				cfg:       cfg,
				stackName: tt.instanceName,
			}

			got := s.getEnv()
			if got["PULUMI_BACKEND_URL"] != tt.wantBackend {
				t.Errorf("getEnv() backend = %v, want %v", got["PULUMI_BACKEND_URL"], tt.wantBackend)
			}

			// Verify standard envs
			if got["PULUMI_CONFIG_PASSPHRASE"] != "" {
				t.Error("PULUMI_CONFIG_PASSPHRASE should be empty")
			}
			if got["AWS_REGION"] != "us-east-1" {
				t.Errorf("AWS_REGION = %v, want us-east-1", got["AWS_REGION"])
			}
		})
	}
}
