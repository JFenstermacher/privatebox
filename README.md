# Privatebox

Privatebox is a CLI tool for managing remote cloud instances with a focus on privacy and simplicity. It wraps the Pulumi Automation API to provide Infrastructure as Code (IaC) capabilities directly within a standalone binary, without requiring a separate Pulumi project setup or account.

## Features

*   **Cloud Agnostic Design**: Currently supports AWS, with architecture in place for GCP, Azure, and DigitalOcean.
*   **Private by Default**: Uses a local backend (`file://`) for state management, keeping your infrastructure data on your machine.
*   **Secure by Design**: Encrypts root volumes with a per-instance KMS key restricted to your user.
*   **Multi-Profile Support**: Manage multiple environments (e.g., dev, prod) with named configuration profiles.
*   **Simple Configuration**: YAML-based configuration located at `~/.config/privatebox/config.yaml`.
*   **Unified Interface**: Consistent `create`, `list`, `destroy`, `connect` commands.

## Installation

```bash
git clone https://github.com/JFenstermacher/privatebox.git
cd privatebox
go build -o privatebox cmd/privatebox/main.go
mv privatebox /usr/local/bin/ # Optional
```

## Quick Start

1.  **Create a profile**:
    ```bash
    privatebox config new default
    ```
2.  **Create an instance**:
    ```bash
    privatebox create my-box
    ```
3.  **Connect**:
    ```bash
    privatebox connect my-box
    ```

## Configuration

Configuration is stored in `~/.config/privatebox/config.yaml`. You can edit it manually or use:
```bash
privatebox config edit
```

### Configuration Structure

```yaml
current_profile: default
profiles:
  default:
    provider: aws
    region: us-east-1
    ssh_public_key_path: ~/.ssh/id_rsa.pub
    connect_command: ssh -i {key} {user}@{ip}
    aws:
      instance_type: t3.micro
      # Optional: Override AMI
      # ami: ami-12345678 
```

### Common Configurations

#### 1. Custom Ingress/Egress Rules
Restrict access to specific ports or IPs. By default, SSH (22) is open to the world.

```yaml
profiles:
  secure-dev:
    provider: aws
    region: us-east-1
    aws:
      instance_type: t3.micro
      ingress_rules:
        # Allow SSH only from a specific IP
        - protocol: tcp
          from_port: 22
          to_port: 22
          cidr_blocks: ["203.0.113.5/32"]
        # Allow HTTP
        - protocol: tcp
          from_port: 80
          to_port: 80
          cidr_blocks: ["0.0.0.0/0"]
      egress_rules:
        # Allow all outbound traffic (default behavior)
        - protocol: "-1"
          from_port: 0
          to_port: 0
          cidr_blocks: ["0.0.0.0/0"]
```

#### 2. Using a Specific AWS Profile
If you use `~/.aws/config` profiles to manage credentials (e.g., for different accounts).

```yaml
profiles:
  work-account:
    provider: aws
    region: us-west-2
    aws:
      profile: my-work-profile # Matches [profile my-work-profile] in ~/.aws/config
      instance_type: t3.medium
```

#### 3. Custom Connection Command (Mosh / SSM)
Change how `privatebox connect` connects to your instance.

**Using Mosh:**
```yaml
profiles:
  mosh-user:
    connect_command: mosh {user}@{ip}
```

**Using AWS SSM (Session Manager):**
```yaml
profiles:
  ssm-user:
    # Requires AWS CLI and Session Manager plugin installed
    connect_command: aws ssm start-session --target {id}
```
*Note: Instances are created with the `AmazonSSMManagedInstanceCore` IAM policy attached by default.*

#### 4. Default User Data
Define a startup script that runs automatically when you create an instance with this profile.

```yaml
profiles:
  docker-host:
    provider: aws
    aws:
      instance_type: t3.small
    user_data: |
      #!/bin/bash
      apt-get update
      apt-get install -y docker.io
      usermod -aG docker ubuntu
```

#### 5. Environment Variables
Inject environment variables into your shell when running `privatebox connect`.

```yaml
profiles:
  dev:
    env:
      EDITOR: vim
      # These variables are set in the local shell that executes ssh/mosh
      # They can be passed to the remote host if ssh config permits (SendEnv)
```

## Usage Commands

### Instance Management

*   **List all instances**:
    ```bash
    privatebox list
    # or
    privatebox ls
    ```

*   **Create**:
    ```bash
    # Use default profile configuration
    privatebox create my-vm

    # Use a specific profile
    privatebox create --profile work-account my-work-vm

    # Override instance type
    privatebox create --type c5.large compute-node

    # Provide a one-off user-data script
    privatebox create --user-data ./setup.sh custom-node
    ```

*   **Connect**:
    ```bash
    # Connects using the configured command (SSH default)
    privatebox connect my-vm
    
    # Fuzzy find instance if name omitted
    privatebox connect
    ```

*   **Power Management**:
    ```bash
    # Start a stopped instance
    privatebox up my-vm

    # Stop a running instance
    privatebox down my-vm
    ```

*   **Destroy**:
    ```bash
    # Permanently delete the instance and its storage
    privatebox destroy my-vm
    ```

### Profile Management

```bash
# List available profiles
privatebox config list

# Create a new profile
privatebox config new <name>

# Switch the active profile
privatebox config use <name>
```

## Architecture

*   **Language**: Go
*   **CLI Framework**: [urfave/cli/v3](https://github.com/urfave/cli)
*   **IaC Engine**: [Pulumi Automation API](https://www.pulumi.com/automation/)
*   **State**: Local file backend. Each instance stores its state in `~/.privatebox/state/<instance_name>/`.
*   **Security**:
    *   **KMS**: Creates a dedicated AWS KMS Key per instance.
    *   **EBS**: Encrypts root volume with that key.
    *   **Policy**: Key policy restricts access to the creating user only (plus root for deletion).

## Development

The project structure follows standard Go layout:

*   `cmd/privatebox`: Entry point.
*   `internal/orchestration`: Pulumi Automation API wrapper.
*   `internal/providers`: Cloud provider implementations.
*   `internal/config`: Configuration loading logic.
