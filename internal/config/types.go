package config

// Config represents the global application configuration.
type Config struct {
	Provider      string    `json:"provider"`            // "aws", "gcp", etc.
	PulumiBackend string    `json:"pulumi_backend"`      // "file://~/.privatebox/state" or s3/url
	Region        string    `json:"region"`              // Global default region
	SSHPublicKey  string    `json:"ssh_public_key_path"` // Path to public key for instances
	AWS           AWSConfig `json:"aws,omitempty"`       // AWS specific config
}

// AWSConfig holds AWS-specific settings.
type AWSConfig struct {
	Profile      string `json:"profile"`
	InstanceType string `json:"instance_type"` // default: t3.micro
	AMI          string `json:"ami"`           // optional override
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Provider:      "aws",
		PulumiBackend: "file://~/.privatebox/state",
		Region:        "us-east-1",
		AWS: AWSConfig{
			InstanceType: "t3.micro",
		},
	}
}
