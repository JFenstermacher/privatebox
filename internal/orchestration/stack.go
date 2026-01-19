package orchestration

import (
	"context"
	"fmt"
	"os"
	"privatebox/internal/config"
	"privatebox/internal/providers"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
)

// StackManager handles the lifecycle of a Pulumi stack.
type StackManager struct {
	stackName string
	project   string
	cfg       *config.Profile
	provider  providers.CloudProvider
}

// NewStackManager creates a new stack manager.
func NewStackManager(cfg *config.Profile, provider providers.CloudProvider, instanceName string) *StackManager {
	return &StackManager{
		stackName: instanceName,
		project:   "privatebox",
		cfg:       cfg,
		provider:  provider,
	}
}

// getStack initializes the automation API stack.
func (s *StackManager) getStack(ctx context.Context, spec providers.InstanceSpec) (auto.Stack, error) {
	// Ensure the workdir exists for local state if needed
	// Pulumi automation API handles workspace setup, but we want to control the backend
	// The backend URL is set via environment variable PULUMI_BACKEND_URL or project settings.
	// For local backend, we usually set the environment variable.

	env := map[string]string{
		"PULUMI_CONFIG_PASSPHRASE": "", // No passphrase for local dev simplicity, or prompt user in real app
		"PULUMI_BACKEND_URL":       s.cfg.PulumiBackend,
	}

	// Set AWS specific env vars if present in config
	if s.cfg.AWS.Profile != "" {
		env["AWS_PROFILE"] = s.cfg.AWS.Profile
	}
	env["AWS_REGION"] = s.cfg.Region

	// Prepare the program
	program := s.provider.GetPulumiProgram(spec)

	// Create or select the stack
	// We use an inline program
	stack, err := auto.UpsertStackInlineSource(ctx, s.stackName, s.project, program, auto.EnvVars(env))
	if err != nil {
		return auto.Stack{}, fmt.Errorf("failed to upsert stack: %w", err)
	}

	// Set configuration on the stack if needed (e.g. region)
	// Usually provider configuration is handled via env vars or setConfig
	if err := stack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: s.cfg.Region}); err != nil {
		return auto.Stack{}, fmt.Errorf("failed to set region config: %w", err)
	}

	return stack, nil
}

// Up provisions the instance.
func (s *StackManager) Up(ctx context.Context, spec providers.InstanceSpec) (auto.UpResult, error) {
	stack, err := s.getStack(ctx, spec)
	if err != nil {
		return auto.UpResult{}, err
	}

	fmt.Printf("Provisioning instance '%s'...\n", s.stackName)

	// Run up
	// We stream stdout to the console so the user sees progress
	res, err := stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	if err != nil {
		return auto.UpResult{}, fmt.Errorf("failed to update stack: %w", err)
	}

	return res, nil
}

// Destroy tears down the instance.
func (s *StackManager) Destroy(ctx context.Context) (auto.DestroyResult, error) {
	// For destroy, we pass an empty spec because the program function isn't strictly needed
	// to tear down existing state, but UpsertStackInlineSource requires one.
	// We'll pass a dummy spec or the same one if available.
	// In a real CLI, we might not have the spec during destroy, so we might need `SelectStack` instead.

	// Better approach for destroy: Try SelectStack first.
	env := map[string]string{
		"PULUMI_CONFIG_PASSPHRASE": "",
		"PULUMI_BACKEND_URL":       s.cfg.PulumiBackend,
	}

	// We need a program even for SelectStackInlineSource usually, but let's try SelectStack
	// which assumes the project exists in the workspace.
	// However, with Automation API Inline, we usually need to re-supply the program.
	// Let's create a dummy spec for destroy purposes.
	dummySpec := providers.InstanceSpec{Name: s.stackName}
	program := s.provider.GetPulumiProgram(dummySpec)

	stack, err := auto.UpsertStackInlineSource(ctx, s.stackName, s.project, program, auto.EnvVars(env))
	if err != nil {
		return auto.DestroyResult{}, fmt.Errorf("failed to select stack: %w", err)
	}

	fmt.Printf("Destroying instance '%s'...\n", s.stackName)
	res, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout))
	if err != nil {
		return auto.DestroyResult{}, fmt.Errorf("failed to destroy stack: %w", err)
	}

	return res, nil
}

// GetOutputs returns the stack outputs.
func (s *StackManager) GetOutputs(ctx context.Context) (auto.OutputMap, error) {
	// Reconstruct stack
	env := map[string]string{
		"PULUMI_CONFIG_PASSPHRASE": "",
		"PULUMI_BACKEND_URL":       s.cfg.PulumiBackend,
	}
	dummySpec := providers.InstanceSpec{Name: s.stackName}
	program := s.provider.GetPulumiProgram(dummySpec)

	stack, err := auto.UpsertStackInlineSource(ctx, s.stackName, s.project, program, auto.EnvVars(env))
	if err != nil {
		return nil, fmt.Errorf("failed to select stack: %w", err)
	}

	outs, err := stack.Outputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get outputs: %w", err)
	}

	return outs, nil
}
