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
	"privatebox/internal/userdata"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v3"
)

// GetRootCommands returns the root-level CLI commands for managing instances.
func GetRootCommands() []*cli.Command {
	profileFlag := &cli.StringFlag{Name: "profile", Usage: "Configuration profile to use"}

	return []*cli.Command{
		{
			Name:      "create",
			Usage:     "Create a new instance",
			ArgsUsage: "<name>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "type", Usage: "Instance type (e.g. t3.small)"},
				&cli.StringFlag{Name: "user-data", Usage: "Path to user-data script"},
				profileFlag,
			},
			Action: createInstance,
		},
		{
			Name:      "destroy",
			Usage:     "Destroy an instance",
			ArgsUsage: "<name>",
			Flags:     []cli.Flag{profileFlag},
			Action:    destroyInstance,
		},
		{
			Name:      "list",
			Aliases:   []string{"ls"},
			Usage:     "List info about an instance",
			ArgsUsage: "<name>",
			Flags:     []cli.Flag{profileFlag},
			Action:    listInstance,
		},
		{
			Name:      "connect",
			Usage:     "Connect (SSH) to an instance",
			ArgsUsage: "[name]",
			Flags: []cli.Flag{
				profileFlag,
			},
			Action: connectInstance,
		},
	}
}

func loadProfile(cmd *cli.Command) (*config.Profile, error) {
	loader, err := config.NewLoader()
	if err != nil {
		return nil, err
	}

	appCfg, err := loader.Load()
	if err != nil {
		return nil, err
	}

	if len(appCfg.Profiles) == 0 {
		return nil, fmt.Errorf("no configuration profiles found. Run 'privatebox config new <name>' to start")
	}

	// Determine profile
	profileName := cmd.String("profile")
	if profileName == "" {
		profileName = appCfg.CurrentProfile
	}
	if profileName == "" {
		return nil, fmt.Errorf("no current profile set")
	}

	profile, ok := appCfg.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", profileName)
	}
	return &profile, nil
}

func getStackManager(cmd *cli.Command, instanceName string) (*orchestration.StackManager, *config.Profile, providers.CloudProvider, error) {
	profile, err := loadProfile(cmd)
	if err != nil {
		return nil, nil, nil, err
	}

	// Provider Factory (Switch based on cfg.Provider in future)
	var provider providers.CloudProvider
	if profile.Provider == "aws" {
		provider = aws.NewAWSProvider(*profile)
	} else {
		return nil, nil, nil, fmt.Errorf("unsupported provider: %s", profile.Provider)
	}

	// Pass pointer to profile
	mgr := orchestration.NewStackManager(profile, provider, instanceName)
	return mgr, profile, provider, nil
}

func createInstance(ctx context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	mgr, cfg, _, err := getStackManager(cmd, name)
	if err != nil {
		return err
	}

	userDataArg := cmd.String("user-data")
	var userDataContent string
	var userDataName string

	if userDataArg != "" {
		// Check if it's a stored script
		um, err := userdata.NewManager()
		if err != nil {
			return err
		}

		if um.Exists(userDataArg) {
			content, err := um.Get(userDataArg)
			if err != nil {
				return fmt.Errorf("failed to get userdata script: %w", err)
			}
			userDataContent = string(content)
			userDataName = userDataArg
		} else {
			// Assume file path
			data, err := os.ReadFile(userDataArg)
			if err != nil {
				return fmt.Errorf("user-data argument is neither a stored script nor a valid file: %w", err)
			}
			userDataContent = string(data)
		}
	}

	// Allow override of instance type
	instanceType := cmd.String("type")
	if instanceType != "" {
		cfg.AWS.InstanceType = instanceType
	}

	spec := providers.InstanceSpec{
		Name:         name,
		Type:         cfg.AWS.InstanceType,
		UserData:     userDataContent,
		UserDataName: userDataName,
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

	mgr, _, _, err := getStackManager(cmd, name)
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

	mgr, _, provider, err := getStackManager(cmd, name)
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

func connectInstance(ctx context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		// No name provided, try to find available instances
		profile, err := loadProfile(cmd)
		if err != nil {
			return err
		}

		stacks, err := orchestration.ListStacks(profile)
		if err != nil {
			return fmt.Errorf("failed to list instances: %w", err)
		}

		if len(stacks) == 0 {
			return fmt.Errorf("no instances found")
		} else if len(stacks) == 1 {
			name = stacks[0]
			fmt.Printf("Only one instance found, connecting to '%s'...\n", name)
		} else {
			// Interactive selection
			prompt := promptui.Select{
				Label: "Select instance to connect to",
				Items: stacks,
				Searcher: func(input string, index int) bool {
					item := stacks[index]
					name := strings.ToLower(item)
					input = strings.ToLower(input)
					return strings.Contains(name, input)
				},
				StartInSearchMode: true,
			}

			_, result, err := prompt.Run()
			if err != nil {
				return fmt.Errorf("prompt failed: %w", err)
			}
			name = result
		}
	}

	mgr, cfg, provider, err := getStackManager(cmd, name)
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

	instanceID, _ := outs["instanceID"].Value.(string)

	user := provider.GetSSHUser()
	host := fmt.Sprintf("%s@%s", user, ip)

	// Determine Private Key Path
	privKeyPath := ""
	if cfg.SSHPublicKey != "" {
		privKeyPath = cfg.SSHPublicKey
		if strings.HasSuffix(privKeyPath, ".pub") {
			privKeyPath = strings.TrimSuffix(privKeyPath, ".pub")
		}
	}

	// Determine Command Template
	cmdTemplate := cfg.ConnectCommand
	if cmdTemplate == "" {
		if privKeyPath != "" {
			cmdTemplate = "ssh -i {key} {host}"
		} else {
			cmdTemplate = "ssh {host}"
		}
	}

	// Replace Variables
	commandStr := cmdTemplate
	commandStr = strings.ReplaceAll(commandStr, "{user}", user)
	commandStr = strings.ReplaceAll(commandStr, "{ip}", ip)
	commandStr = strings.ReplaceAll(commandStr, "{id}", instanceID)
	commandStr = strings.ReplaceAll(commandStr, "{key}", privKeyPath)
	commandStr = strings.ReplaceAll(commandStr, "{host}", host)

	fmt.Printf("Connecting to %s (%s)...\n", name, ip)
	fmt.Printf("Command: %s\n", commandStr)

	// Use sh -c to allow for complex commands (pipes, etc) and correct argument parsing by shell
	sshCmd := exec.Command("sh", "-c", commandStr)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	// Prepare environment variables
	env := os.Environ()

	// 1. AWS Profile from config
	if cfg.AWS.Profile != "" {
		env = append(env, fmt.Sprintf("AWS_PROFILE=%s", cfg.AWS.Profile))
	}

	// 2. Custom environment variables from profile
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	sshCmd.Env = env

	return sshCmd.Run()
}
