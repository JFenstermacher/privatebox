# Privatebox Specification

**Tool Name**: `privatebox`
**Goal**: Manage lifecycle of remote instances (AWS initially) via a unified CLI.
**Core Engine**: [Pulumi Automation API](https://www.pulumi.com/automation/).
**CLI Framework**: `urfave/cli/v3`.

## 1. Project Overview
The tool will act as a wrapper around the Pulumi Automation API, embedding Infrastructure as Code (IaC) directly into the Go binary. It is designed to be provider-agnostic, starting with AWS.

## 2. Architecture

1.  **CLI Layer (`urfave/cli`)**: Parses user input.
2.  **Config Manager**: Custom loader for `~/.config/privatebox/config.yaml`.
3.  **Orchestrator**: Sets up the Pulumi Stack (Local backend by default).
4.  **Provider Interface**:
    *   Injects provider-specific resources (e.g., `ec2.NewInstance`) into the Pulumi context.
    *   Translates generic inputs to provider-specific types.
    *   Fetches runtime status.

## 3. Directory Structure

```text
.
├── cmd
│   └── privatebox
│       └── main.go           # Entry point
├── internal
│   ├── cli
│   │   ├── commands.go       # Command definitions (create, list, etc.)
│   │   └── flags.go          # Global flags
│   ├── config
│   │   └── loader.go         # Custom YAML config logic
│   ├── orchestration
│   │   └── stack.go          # Pulumi Automation API wrapper
│   └── providers
│       ├── interface.go      # CloudProvider interface definition
│       └── aws
│           ├── provider.go   # Implements CloudProvider
│           └── program.go    # The actual Pulumi resource definition
├── go.mod
└── go.sum
```

## 4. Key Components & Interfaces

### A. Configuration (Custom)
Location: `~/.config/privatebox/config.yaml`

```go
package config

type Config struct {
    Provider      string `yaml:"provider"`       // "aws", "gcp", etc.
    PulumiBackend string `yaml:"pulumi_backend"` // "file://~/.privatebox/state" or s3/url
    Region        string `yaml:"region"`
    SSHPublicKey  string `yaml:"ssh_public_key_path"`
    // Provider specific generic map or structs
    AWS           AWSConfig `yaml:"aws,omitempty"`
}

type AWSConfig struct {
    Profile      string `yaml:"profile"`
    InstanceType string `yaml:"instance_type"` // default: t3.micro
    AMI          string `yaml:"ami"`           // optional override
}
```

### B. The Provider Interface
Location: `internal/providers/interface.go`

```go
package providers

import (
	"context"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// InstanceSpec defines what we want to build
type InstanceSpec struct {
    Name     string
    Type     string
    UserData string // The launch script content
    Tags     map[string]string
}

// RuntimeInfo contains status data (IP, State, Metrics)
type RuntimeInfo struct {
    ID        string
    PublicIP  string
    State     string
    CPUUsage  float64 
}

type CloudProvider interface {
    // Name returns the provider name (e.g., "aws")
    Name() string

    // GetPulumiProgram returns the function to run inside pulumi.Run
    GetPulumiProgram(spec InstanceSpec) pulumi.RunFunc

    // GetSSHUser returns the default user for this image (e.g., "ubuntu", "ec2-user")
    GetSSHUser() string
    
    // External Operations (Non-Pulumi)
    // Used for listing real-time status or metrics not stored in Pulumi state
    GetInstanceStatus(ctx context.Context, instanceID string) (*RuntimeInfo, error)
}
```

## 5. Implementation Steps

1.  **Scaffold & CLI Setup**: Init module, install deps, create `main.go`.
2.  **Custom Config**: Implement `loader.go` and `config` command.
3.  **Orchestration**: Implement `auto.Stack` wrapper with local backend default.
4.  **AWS Provider**: Implement `CloudProvider` for AWS (Security Group, KeyPair, EC2).
5.  **Core Commands**: `create`, `list`, `delete`.
6.  **SSH**: Implement `ssh` command using IP from state.
7.  **Monitoring**: Implement `GetInstanceStatus` with AWS SDK.
