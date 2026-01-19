package providers

import (
	"context"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// InstanceSpec defines the desired state of an instance.
type InstanceSpec struct {
	Name         string
	Type         string            // e.g. "t3.micro"
	ProfileName  string            // Profile used to create the instance
	UserData     string            // Cloud-init script or similar
	UserDataName string            // Name of the managed userdata script (optional)
	Tags         map[string]string // Resource tags
}

// RuntimeInfo contains status data fetched from the cloud provider.
type RuntimeInfo struct {
	ID       string
	PublicIP string
	State    string
	CPUUsage float64
}

// CloudProvider defines the contract for any cloud backend (AWS, GCP, etc).
type CloudProvider interface {
	// Name returns the provider identifier (e.g. "aws").
	Name() string

	// GetPulumiProgram returns the logic to run inside the Pulumi engine.
	GetPulumiProgram(spec InstanceSpec) pulumi.RunFunc

	// GetSSHUser returns the default username for SSH connections (e.g. "ubuntu").
	GetSSHUser() string

	// GetInstanceStatus fetches real-time data from the cloud API (outside Pulumi state).
	GetInstanceStatus(ctx context.Context, instanceID string) (*RuntimeInfo, error)
}
