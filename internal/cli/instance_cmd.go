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
	"strings"
	"sync"

	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
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
			ArgsUsage: "[name]",
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
		{
			Name:      "up",
			Usage:     "Start an instance",
			ArgsUsage: "[name]",
			Flags:     []cli.Flag{profileFlag},
			Action:    upInstance,
		},
		{
			Name:      "down",
			Usage:     "Stop an instance",
			ArgsUsage: "[name]",
			Flags:     []cli.Flag{profileFlag},
			Action:    downInstance,
		},
	}
}

func loadProfile(cmd *cli.Command) (*config.Profile, string, error) {
	loader, err := config.NewLoader()
	if err != nil {
		return nil, "", err
	}

	appCfg, err := loader.Load()
	if err != nil {
		return nil, "", err
	}

	if len(appCfg.Profiles) == 0 {
		return nil, "", fmt.Errorf("no configuration profiles found. Run 'privatebox config new <name>' to start")
	}

	// Determine profile
	profileName := cmd.String("profile")
	if profileName == "" {
		profileName = appCfg.CurrentProfile
	}
	if profileName == "" {
		return nil, "", fmt.Errorf("no current profile set")
	}

	profile, ok := appCfg.Profiles[profileName]
	if !ok {
		return nil, "", fmt.Errorf("profile '%s' not found", profileName)
	}
	return &profile, profileName, nil
}

func getStackManager(cmd *cli.Command, instanceName string) (*orchestration.StackManager, *config.Profile, string, providers.CloudProvider, error) {
	profile, profileName, err := loadProfile(cmd)
	if err != nil {
		return nil, nil, "", nil, err
	}

	// Provider Factory (Switch based on cfg.Provider in future)
	var provider providers.CloudProvider
	if profile.Provider == "aws" {
		provider = aws.NewAWSProvider(*profile)
	} else {
		return nil, nil, "", nil, fmt.Errorf("unsupported provider: %s", profile.Provider)
	}

	// Pass pointer to profile
	mgr := orchestration.NewStackManager(profile, provider, instanceName)
	return mgr, profile, profileName, provider, nil
}

func createInstance(ctx context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	mgr, cfg, profileName, _, err := getStackManager(cmd, name)
	if err != nil {
		return err
	}

	userDataArg := cmd.String("user-data")
	var userDataContent string
	var userDataName string

	if userDataArg != "" {
		// Load config to check for user-data alias
		loader, err := config.NewLoader()
		if err != nil {
			return err
		}
		appCfg, err := loader.Load()
		if err != nil {
			return err
		}

		if content, ok := appCfg.UserData[userDataArg]; ok {
			userDataContent = content
			userDataName = userDataArg
		} else {
			// Assume file path
			data, err := os.ReadFile(userDataArg)
			if err != nil {
				return fmt.Errorf("user-data argument is neither a stored alias nor a valid file: %w", err)
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
		ProfileName:  profileName,
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

	mgr, _, _, _, err := getStackManager(cmd, name)
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
	// Determine profile first, as we need it to list stacks
	profile, _, err := loadProfile(cmd)
	if err != nil {
		return err
	}

	var instances []string
	name := cmd.Args().First()
	if name != "" {
		instances = []string{name}
	} else {
		// List all stacks
		stacks, err := orchestration.ListStacks(profile)
		if err != nil {
			return fmt.Errorf("failed to list instances: %w", err)
		}
		instances = stacks
	}

	if len(instances) == 0 {
		fmt.Println("No instances found.")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "PROFILE", "PRIVATE IP", "PUBLIC IP", "STATE"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)

	for _, instName := range instances {
		// Create provider
		var provider providers.CloudProvider
		if profile.Provider == "aws" {
			provider = aws.NewAWSProvider(*profile)
		} else {
			fmt.Fprintf(os.Stderr, "Skipping %s: unsupported provider %s\n", instName, profile.Provider)
			continue
		}

		mgr := orchestration.NewStackManager(profile, provider, instName)

		outs, err := mgr.GetOutputs(ctx)
		if err != nil {
			// If we can't get outputs (e.g. stack broken), just show empty or error
			table.Append([]string{instName, "", "", "", "Error: " + err.Error()})
			continue
		}

		id, _ := outs["instanceID"].Value.(string)
		publicIP, _ := outs["publicIP"].Value.(string)
		privateIP, _ := outs["privateIP"].Value.(string)
		profileName, _ := outs["profileName"].Value.(string)

		if profileName == "" {
			profileName = "Unknown"
		}

		state := "Unknown"
		if id != "" {
			status, err := provider.GetInstanceStatus(ctx, id)
			if err == nil {
				state = status.State
			} else {
				state = fmt.Sprintf("Error: %v", err)
			}
		} else {
			state = "Provisioning/Error"
		}

		table.Append([]string{instName, profileName, privateIP, publicIP, state})
	}

	table.Render()
	return nil
}

func connectInstance(ctx context.Context, cmd *cli.Command) error {
	name, err := selectInstance(ctx, cmd, "")
	if err != nil {
		return err
	}

	mgr, cfg, _, provider, err := getStackManager(cmd, name)
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

func upInstance(ctx context.Context, cmd *cli.Command) error {
	name, err := selectInstance(ctx, cmd, "stopped")
	if err != nil {
		return err
	}

	mgr, _, _, provider, err := getStackManager(cmd, name)
	if err != nil {
		return err
	}

	outs, err := mgr.GetOutputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	instanceID, ok := outs["instanceID"].Value.(string)
	if !ok || instanceID == "" {
		return fmt.Errorf("instance ID not found in stack outputs")
	}

	fmt.Printf("Starting instance '%s' (%s)...\n", name, instanceID)
	if err := provider.StartInstance(ctx, instanceID); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	fmt.Println("Instance start requested.")
	return nil
}

func downInstance(ctx context.Context, cmd *cli.Command) error {
	name, err := selectInstance(ctx, cmd, "running")
	if err != nil {
		return err
	}

	mgr, _, _, provider, err := getStackManager(cmd, name)
	if err != nil {
		return err
	}

	outs, err := mgr.GetOutputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	instanceID, ok := outs["instanceID"].Value.(string)
	if !ok || instanceID == "" {
		return fmt.Errorf("instance ID not found in stack outputs")
	}

	fmt.Printf("Stopping instance '%s' (%s)...\n", name, instanceID)
	if err := provider.StopInstance(ctx, instanceID); err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	fmt.Println("Instance stop requested.")
	return nil
}

func selectInstance(ctx context.Context, cmd *cli.Command, filterState string) (string, error) {
	name := cmd.Args().First()
	if name != "" {
		return name, nil
	}

	candidates, err := getInstancesWithState(ctx, cmd, filterState)
	if err != nil {
		return "", err
	}

	if len(candidates) == 0 {
		msg := "no instances found"
		if filterState != "" {
			msg += fmt.Sprintf(" with state '%s'", filterState)
		}
		return "", fmt.Errorf(msg)
	} else if len(candidates) == 1 {
		name = candidates[0]
		fmt.Printf("Selected '%s'\n", name)
		return name, nil
	}

	// Interactive selection
	prompt := promptui.Select{
		Label: "Select instance",
		Items: candidates,
		Searcher: func(input string, index int) bool {
			item := candidates[index]
			name := strings.ToLower(item)
			input = strings.ToLower(input)
			return strings.Contains(name, input)
		},
		StartInSearchMode: true,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed: %w", err)
	}
	return result, nil
}

func getInstancesWithState(ctx context.Context, cmd *cli.Command, desiredState string) ([]string, error) {
	profile, _, err := loadProfile(cmd)
	if err != nil {
		return nil, err
	}

	stacks, err := orchestration.ListStacks(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	if desiredState == "" {
		return stacks, nil
	}

	// Filter by state
	var (
		mu         sync.Mutex
		candidates []string
		wg         sync.WaitGroup
	)

	fmt.Printf("Filtering instances by state '%s'...\n", desiredState)

	for _, stackName := range stacks {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			// We need a provider for each stack
			var provider providers.CloudProvider
			if profile.Provider == "aws" {
				provider = aws.NewAWSProvider(*profile)
			} else {
				return
			}

			mgr := orchestration.NewStackManager(profile, provider, name)
			outs, err := mgr.GetOutputs(ctx)
			if err != nil {
				return
			}

			id, ok := outs["instanceID"].Value.(string)
			if !ok || id == "" {
				return
			}

			status, err := provider.GetInstanceStatus(ctx, id)
			if err != nil {
				return
			}

			// Check match
			match := false
			if desiredState == "running" && status.State == "running" {
				match = true
			} else if desiredState == "stopped" && status.State == "stopped" {
				match = true
			}

			if match {
				mu.Lock()
				candidates = append(candidates, name)
				mu.Unlock()
			}
		}(stackName)
	}

	wg.Wait()
	return candidates, nil
}
