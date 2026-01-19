package config

// AppConfig represents the top-level configuration file structure.
type AppConfig struct {
	CurrentProfile string             `json:"current_profile" yaml:"current_profile"`
	Profiles       map[string]Profile `json:"profiles" yaml:"profiles"`
	UserData       map[string]string  `json:"user_data" yaml:"user_data"`
}

// Profile represents a specific configuration set.
type Profile struct {
	Provider       string            `json:"provider" yaml:"provider"`                       // "aws", "gcp", etc.
	PulumiBackend  string            `json:"pulumi_backend" yaml:"pulumi_backend"`           // "file://~/.privatebox/state" or s3/url
	Region         string            `json:"region" yaml:"region"`                           // Global default region
	SSHPublicKey   string            `json:"ssh_public_key_path" yaml:"ssh_public_key_path"` // Path to public key for instances
	ConnectCommand string            `json:"connect_command" yaml:"connect_command"`         // Command template to connect (e.g. "ssh {user}@{ip}", "mosh ...")
	Env            map[string]string `json:"env,omitempty" yaml:"env,omitempty"`             // Extra environment variables
	AWS            AWSConfig         `json:"aws,omitempty" yaml:"aws,omitempty"`             // AWS specific config
}

// AWSConfig holds AWS-specific settings.
type AWSConfig struct {
	Profile      string `json:"profile" yaml:"profile"`
	InstanceType string `json:"instance_type" yaml:"instance_type"` // default: t3.micro
	AMI          string `json:"ami" yaml:"ami"`                     // optional override
}

// DefaultProfile returns a profile with sensible defaults.
func DefaultProfile() Profile {
	return Profile{
		Provider:       "aws",
		PulumiBackend:  "file://~/.privatebox/state",
		Region:         "us-east-1",
		ConnectCommand: "ssh -i {key} {user}@{ip}",
		AWS: AWSConfig{
			InstanceType: "t3.micro",
		},
	}
}

// NewAppConfig creates a new AppConfig with no profiles.
func NewAppConfig() AppConfig {
	return AppConfig{
		CurrentProfile: "",
		Profiles:       make(map[string]Profile),
		UserData:       make(map[string]string),
	}
}
