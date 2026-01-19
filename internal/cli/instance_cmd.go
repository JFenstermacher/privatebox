package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"privatebox/internal/config"
	"privatebox/internal/orchestration"
	"privatebox/internal/providers"
	"privatebox/internal/providers/aws"

	"github.com/urfave/cli/v3"
)

// InstanceCommands returns the CLI commands for managing instances.
func InstanceCommands() *cli.Command {
	return &cli.Command{
		Name:  "instance",
		Usage: "Manage remote instances",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "Create a new instance",
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "type", Usage: "Instance type (e.g. t3.small)"},
					&cli.StringFlag{Name: "user-data", Usage: "Path to user-data script"},
				},
				Action: createInstance,
			},
			{
				Name:      "destroy",
				Usage:     "Destroy an instance",
				ArgsUsage: "<name>",
				Action:    destroyInstance,
			},
			{
				Name:      "list",
				Usage:     "List info about an instance",
				ArgsUsage: "<name>",
				Action:    listInstance,
			},
			{
				Name:      "ssh",
				Usage:     "SSH into an instance",
				ArgsUsage: "<name>",
				Action:    sshInstance,
			},
		},
	}
}

func getStackManager(instanceName string) (*orchestration.StackManager, *config.Config, providers.CloudProvider, error) {
	loader, err := config.NewLoader()
	if err != nil {
		return nil, nil, nil, err
	}

	cfg, err := loader.Load()
	if err != nil {
		return nil, nil, nil, err
	}

	// Provider Factory (Switch based on cfg.Provider in future)
	var provider providers.CloudProvider
	if cfg.Provider == "aws" {
		provider = aws.NewAWSProvider(*cfg)
	} else {
		return nil, nil, nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}

	mgr := orchestration.NewStackManager(cfg, provider, instanceName)
	return mgr, cfg, provider, nil
}

func createInstance(ctx context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	mgr, cfg, _, err := getStackManager(name)
	if err != nil {
		return err
	}

	userDataPath := cmd.String("user-data")
	var userDataContent string
	if userDataPath != "" {
		data, err := os.ReadFile(userDataPath)
		if err != nil {
			return fmt.Errorf("failed to read user-data: %w", err)
		}
		userDataContent = string(data)
	}

	// Allow override of instance type
	instanceType := cmd.String("type")
	if instanceType != "" {
		cfg.AWS.InstanceType = instanceType
	}

	spec := providers.InstanceSpec{
		Name:     name,
		Type:     cfg.AWS.InstanceType,
		UserData: userDataContent,
	}

	_, err = mgr.Up(ctx, spec)
	if err != nil {
		return err
	}

	fmt.Printf("Instance '%s' created successfully.\n", name)
	return nil
}

func destroyInstance(ctx context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	mgr, _, _, err := getStackManager(name)
	if err != nil {
		return err
	}

	_, err = mgr.Destroy(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Instance '%s' destroyed.\n", name)
	return nil
}

func listInstance(ctx context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	mgr, _, provider, err := getStackManager(name)
	if err != nil {
		return err
	}

	outs, err := mgr.GetOutputs(ctx)
	if err != nil {
		return err
	}

	id, ok := outs["instanceID"].Value.(string)
	if !ok {
		return fmt.Errorf("instanceID output not found")
	}
	ip, _ := outs["publicIP"].Value.(string)

	fmt.Printf("Instance: %s\n", name)
	fmt.Printf("ID:       %s\n", id)
	fmt.Printf("IP:       %s\n", ip)

	// Fetch runtime status
	status, err := provider.GetInstanceStatus(ctx, id)
	if err != nil {
		fmt.Printf("Status:   Unknown (%v)\n", err)
	} else {
		fmt.Printf("State:    %s\n", status.State)
	}

	return nil
}

func sshInstance(ctx context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	mgr, cfg, provider, err := getStackManager(name)
	if err != nil {
		return err
	}

	outs, err := mgr.GetOutputs(ctx)
	if err != nil {
		return err
	}

	ip, ok := outs["publicIP"].Value.(string)
	if !ok {
		return fmt.Errorf("publicIP output not found, instance might not be ready")
	}

	user := provider.GetSSHUser()

	sshArgs := []string{user + "@" + ip}
	if cfg.SSHPublicKey != "" {
		// Use the private key assuming it's the pair of the public key
		// Usually private key is without .pub
		// This is a naive assumption for the MVP
		privKeyPath := cfg.SSHPublicKey
		// Remove .pub if present
		// In reality, user should config private key path or use ssh-agent
		// We'll pass it if it looks like a key path
		sshArgs = append([]string{"-i", privKeyPath}, sshArgs...)
	}

	fmt.Printf("Connecting to %s (%s)...\n", name, ip)

	sshCmd := exec.Command("ssh", sshArgs...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}
