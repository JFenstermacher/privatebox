# Privatebox

Privatebox is a CLI tool for managing remote cloud instances with a focus on privacy and simplicity. It wraps the Pulumi Automation API to provide Infrastructure as Code (IaC) capabilities directly within a standalone binary, without requiring a separate Pulumi project setup or account.

## Features

*   **Cloud Agnostic Design**: Currently supports AWS, with architecture in place for GCP, Azure, and DigitalOcean.
*   **Private by Default**: Uses a local backend (`file://`) for state management, keeping your infrastructure data on your machine.
*   **Multi-Profile Support**: Manage multiple environments (e.g., dev, prod) with named configuration profiles.
*   **Simple Configuration**: YAML-based configuration located at `~/.config/privatebox/config.yaml`.
*   **Unified Interface**: Consistent `create`, `list`, `destroy`, `connect` commands regardless of the underlying provider.

## Installation

```bash
git clone https://github.com/JFenstermacher/privatebox.git
cd privatebox
go build -o privatebox cmd/privatebox/main.go
mv privatebox /usr/local/bin/ # Optional
```

## Usage

### 1. Configuration Management

Privatebox supports multiple named configuration profiles.

**Initialize:**
```bash
privatebox config init
```

**Manage Profiles:**
```bash
# List all profiles
privatebox config list

# Create a new profile (clones default defaults)
privatebox config new dev

# Switch default profile
privatebox config use dev

# Edit configuration (opens $EDITOR)
privatebox config edit
```

### 2. Configure AWS Credentials

Ensure you have standard AWS credentials set up (environment variables `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` or `~/.aws/credentials`).

### 3. Manage Instances

**Create an instance:**

```bash
# Create using the current default profile
privatebox create my-dev-box

# Create using a specific profile (e.g., prod)
privatebox create --profile prod --type t3.medium my-app-server

# Create with user-data script
privatebox create --user-data ./setup.sh my-worker
```

**List instance details:**

```bash
privatebox list my-dev-box
```

**SSH into the instance:**

```bash
privatebox connect my-dev-box
```

**Destroy the instance:**

```bash
privatebox destroy my-dev-box
```

## Architecture

*   **Language**: Go
*   **CLI Framework**: [urfave/cli/v3](https://github.com/urfave/cli)
*   **IaC Engine**: [Pulumi Automation API](https://www.pulumi.com/automation/)
*   **State**: Local file backend

## Development

The project structure follows standard Go layout:

*   `cmd/privatebox`: Entry point.
*   `internal/orchestration`: Pulumi Automation API wrapper.
*   `internal/providers`: Cloud provider implementations.
*   `internal/config`: Configuration loading logic.
