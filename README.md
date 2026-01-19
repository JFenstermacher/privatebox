# Privatebox

Privatebox is a CLI tool for managing remote cloud instances with a focus on privacy and simplicity. It wraps the Pulumi Automation API to provide Infrastructure as Code (IaC) capabilities directly within a standalone binary, without requiring a separate Pulumi project setup or account.

## Features

*   **Cloud Agnostic Design**: Currently supports AWS, with architecture in place for GCP, Azure, and DigitalOcean.
*   **Private by Default**: Uses a local backend (`file://`) for state management, keeping your infrastructure data on your machine.
*   **Simple Configuration**: JSON-based configuration located at `~/.config/privatebox/config.json`.
*   **Unified Interface**: Consistent `create`, `list`, `destroy`, `ssh` commands regardless of the underlying provider.

## Installation

```bash
git clone https://github.com/JFenstermacher/privatebox.git
cd privatebox
go build -o privatebox cmd/privatebox/main.go
mv privatebox /usr/local/bin/ # Optional
```

## Usage

### 1. Initialize Configuration

Initialize the default configuration file:

```bash
privatebox config init
```

This creates `~/.config/privatebox/config.json`. You can edit this file to set your default region, instance type, and SSH key path.

### 2. Configure AWS Credentials

Ensure you have standard AWS credentials set up (environment variables `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` or `~/.aws/credentials`).

### 3. Manage Instances

**Create an instance:**

```bash
# Create a default instance (t3.micro, Ubuntu 22.04)
privatebox instance create my-dev-box

# Create with specific type and user-data script
privatebox instance create --type t3.medium --user-data ./setup.sh my-app-server
```

**List instance details:**

```bash
privatebox instance list my-dev-box
```

**SSH into the instance:**

```bash
privatebox instance ssh my-dev-box
```

**Destroy the instance:**

```bash
privatebox instance destroy my-dev-box
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
