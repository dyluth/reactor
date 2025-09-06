package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/dyluth/reactor/pkg/config"
	"github.com/dyluth/reactor/pkg/core"
	"github.com/dyluth/reactor/pkg/docker"
	"github.com/dyluth/reactor/pkg/orchestrator"
	"github.com/dyluth/reactor/pkg/workspace"
	"github.com/spf13/cobra"
)

// Build-time variables injected via linker flags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reactor",
		Short: "Containerized development environment for AI agents",
		Long: `Reactor provides simple, fast, and reliable containerized development environments
for AI CLI tools like Claude, Gemini, and others.

It manages account-isolated configuration, persistent sessions, and container
lifecycle while keeping your host machine clean.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add global flags
	cmd.PersistentFlags().Bool("verbose", false, "Enable verbose logging")

	// Add subcommands
	cmd.AddCommand(newUpCmd())
	cmd.AddCommand(newDownCmd())
	cmd.AddCommand(newExecCmd())
	cmd.AddCommand(newBuildCmd())
	cmd.AddCommand(newSessionsCmd())
	cmd.AddCommand(newDiffCmd())
	cmd.AddCommand(newAccountsCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newWorkspaceCmd())
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func newUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start dev container from devcontainer.json",
		Long: `Start a development container based on the devcontainer.json configuration.

The up command provisions a Docker container based on the devcontainer.json
specification found in your project, then attaches you to an interactive 
session. Containers are automatically reused when possible for fast startup.

Examples:
  reactor up                               # Start container from devcontainer.json
  reactor up --account work-account       # Override account for isolation
  reactor up --rebuild                     # Force rebuild before starting

For more details, see the full documentation.`,
		RunE: upCmdHandler,
	}

	// Add flags (removed --provider and --image, kept account for override)
	cmd.Flags().String("account", "", "Override account from devcontainer.json customizations")
	cmd.Flags().Bool("rebuild", false, "Force rebuild of container image before starting")
	cmd.Flags().Bool("discovery-mode", false, "Run with no mounts for configuration discovery")
	cmd.Flags().Bool("docker-host-integration", false, "Mount host Docker socket (DANGEROUS - use only with trusted images)")
	cmd.Flags().StringSliceP("port", "p", []string{}, "Port forwarding (host:container), can be used multiple times")

	return cmd
}

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop and remove dev container for current project",
		Long: `Stop and remove the development container for the current project.

This command stops the running container and removes it to free up system
resources. The container can be recreated with 'reactor up'.

Examples:
  reactor down                             # Stop and remove current project container

For more details, see the full documentation.`,
		RunE: downCmdHandler,
	}
}

func newExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec <command...>",
		Short: "Execute command in running dev container",
		Long: `Execute a command inside the running development container.

The container must already be running (started with 'reactor up'). This is
useful for running tests, builds, or other commands inside the container.

Examples:
  reactor exec npm test                    # Run npm test inside container
  reactor exec -- ls -la                  # Run ls command (use -- for flags)

For more details, see the full documentation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("exec command not yet implemented - this will be added in Milestone 2")
		},
	}
}

func newBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build dev container image from devcontainer.json",
		Long: `Build the development container image based on devcontainer.json.

This command only builds the container image without starting it. Use this
when you want to pre-build images or verify the build process.

Examples:
  reactor build                            # Build container image
  reactor build --no-cache                # Build without using cache

For more details, see the full documentation.`,
		RunE: buildCmdHandler,
	}
}

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [container-name]",
		Short: "Show container filesystem changes",
		Long: `Show changes made to container filesystem during AI agent session.

This command is particularly useful for discovery mode to understand what
configuration files and directories an AI agent creates. Without arguments,
it operates on the discovery container for the current project.

Examples:
  reactor diff                                    # Diff current project's discovery container
  reactor diff reactor-discovery-cam-myproject   # Diff specific container by name

For more details, see the full documentation.`,
		RunE: diffCmdHandler,
	}

	cmd.Flags().Bool("discovery", false, "Run in discovery mode (no file mounts)")

	return cmd
}

func newAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Manage account configurations",
		Long: `Manage isolated account configurations for different contexts.

The accounts system allows you to maintain separate AI agent configurations
for different contexts like work, personal projects, or different teams.
Each account has its own configuration directories and state isolation.

Examples:
  reactor accounts list           # List all configured accounts
  reactor accounts show          # Show current account
  reactor accounts set work      # Switch to work account

For more details, see the full documentation.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List configured accounts",
		Long:  "List all accounts with configuration directories in ~/.reactor/",
		RunE:  accountsListHandler,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current account",
		Long:  "Show the current account from project configuration",
		RunE:  accountsShowHandler,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <account-name>",
		Short: "Set active account",
		Long:  "Set the active account for the current project",
		Args:  cobra.ExactArgs(1),
		RunE:  accountsSetHandler,
	})

	return cmd
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage project configuration",
		Long: `Manage project-specific configuration for providers, accounts, and settings.

The config command helps you initialize, view, and modify reactor configuration
for your projects. Each project can have different providers, accounts, and
container images configured independently.

Examples:
  reactor config init                # Initialize project configuration
  reactor config show               # Display current configuration
  reactor config set provider claude # Set AI provider to claude
  reactor config get account        # Get current account setting

For more details, see the full documentation.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show resolved configuration",
		Long:  "Display current configuration hierarchy and account directory locations",
		RunE:  configShowHandler,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Get configuration value",
		Long:  "Retrieve configuration value from project settings",
		Args:  cobra.ExactArgs(1),
		RunE:  configGetHandler,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Long: `Set configuration value in project settings.
		
Examples:
  reactor config set provider claude
  reactor config set image python
  reactor config set danger true
  reactor config set account work-account`,
		Args: cobra.ExactArgs(2),
		RunE: configSetHandler,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize project configuration",
		Long:  "Create .reactor.conf with sensible defaults and set up account directories",
		RunE:  configInitHandler,
	})

	return cmd
}

func newSessionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "Manage container sessions",
		Long: `Manage and interact with reactor container sessions.

The sessions command helps you list, inspect, and attach to reactor containers
across different projects and accounts. This enables easy switching between
development contexts without losing your work.

Examples:
  reactor sessions list          # Show all reactor containers  
  reactor sessions attach        # Auto-attach to current project
  reactor sessions attach name   # Attach to specific container

For more details, see the full documentation.`,
	}

	// Add subcommands
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all reactor containers",
		Long: `List all reactor containers with their status and project information.

Shows containers across all accounts and projects, including both running and
stopped containers. Use this to see what development environments are available.

For more details, see the full documentation.`,
		RunE: sessionsListHandler,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "attach [container-name]",
		Short: "Attach to a container session",
		Long: `Attach to a specific container session by name, or auto-attach to the current project's container.

Without arguments, automatically finds and attaches to the container for the current
project. With a container name, attaches to that specific container. Stopped
containers are automatically started before attachment.

Examples:
  reactor sessions attach                           # Auto-attach to current project
  reactor sessions attach reactor-cam-myproject-abc123  # Attach to specific container

For more details, see the full documentation.`,
		RunE: sessionsAttachHandler,
		Args: cobra.MaximumNArgs(1),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clean",
		Short: "Clean up all reactor containers",
		Long: `Clean up all reactor containers to free system resources.

Removes all reactor containers (both running and stopped) across all accounts and
projects. This is useful for system maintenance or when you want to start fresh.

Examples:
  reactor sessions clean          # Remove all reactor containers

For more details, see the full documentation.`,
		RunE: sessionsCleanHandler,
	})

	return cmd
}

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Generate shell completion scripts",
		Long: `Generate completion scripts for your shell.

To install completions:

  # Bash
  source <(reactor completion bash)
  
  # To load completions permanently, add to your ~/.bashrc:
  echo 'source <(reactor completion bash)' >> ~/.bashrc

  # Zsh
  source <(reactor completion zsh)
  
  # To load completions permanently, add to your ~/.zshrc:
  echo 'source <(reactor completion zsh)' >> ~/.zshrc

  # Fish
  reactor completion fish | source
  
  # To load completions permanently:
  reactor completion fish > ~/.config/fish/completions/reactor.fish`,
		Args:                  cobra.ExactArgs(1),
		ValidArgs:             []string{"bash", "zsh", "fish"},
		RunE:                  completionHandler,
		DisableFlagsInUseLine: true,
	}
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display version, build date, and git commit information",
		Run:   versionHandler,
	}
}

// Command handlers
func upCmdHandler(cmd *cobra.Command, args []string) error {
	// Get CLI flags
	accountOverride, _ := cmd.Flags().GetString("account")
	rebuild, _ := cmd.Flags().GetBool("rebuild")
	discoveryMode, _ := cmd.Flags().GetBool("discovery-mode")
	dockerHostIntegration, _ := cmd.Flags().GetBool("docker-host-integration")
	portMappings, _ := cmd.Flags().GetStringSlice("port")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	// Get current working directory as project directory
	projectDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Build UpConfig for orchestrator
	upConfig := orchestrator.UpConfig{
		ProjectDirectory:      projectDirectory,
		AccountOverride:       accountOverride,
		ForceRebuild:          rebuild,
		CLIPortMappings:       portMappings,
		DiscoveryMode:         discoveryMode,
		DockerHostIntegration: dockerHostIntegration,
		Verbose:               verbose,
	}

	// Call orchestrator Up function
	ctx := context.Background()
	_, containerID, err := orchestrator.Up(ctx, upConfig)
	if err != nil {
		return err
	}

	// Initialize Docker service for session attachment
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Attach to interactive session
	if verbose {
		fmt.Printf("[INFO] Attaching to container...\n")
	} else {
		fmt.Printf("Attaching to container session...\n")
	}

	if err := dockerService.AttachInteractiveSession(ctx, containerID); err != nil {
		return fmt.Errorf("failed to attach to container session: %w", err)
	}

	// Inform user about container state after session ends
	fmt.Printf("\nSession ended. Container is still running.\n")
	fmt.Printf("Use 'docker stop %s' to stop it.\n", containerID)

	return nil
}

func downCmdHandler(cmd *cobra.Command, args []string) error {
	// Get current working directory as project directory
	projectDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Call orchestrator Down function
	ctx := context.Background()
	return orchestrator.Down(ctx, projectDirectory)
}

func diffCmdHandler(cmd *cobra.Command, args []string) error {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return err
	}

	// Load configuration to validate project setup
	configService := config.NewService()
	resolved, err := configService.ResolveConfiguration()
	if err != nil {
		return err
	}

	// Initialize Docker service
	ctx := context.Background()
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Check Docker daemon health
	if err := dockerService.CheckHealth(ctx); err != nil {
		return fmt.Errorf("docker daemon not available: %w", err)
	}

	// Determine container name to diff
	var containerName string
	if len(args) > 0 {
		// User provided specific container name
		containerName = args[0]
	} else {
		// Default to discovery container for current project
		containerName = core.GenerateDiscoveryContainerName(resolved.Account, resolved.ProjectRoot, resolved.ProjectHash)
	}

	// Check if container exists
	containerInfo, err := dockerService.ContainerExists(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to check container existence: %w", err)
	}

	if containerInfo.Status == docker.StatusNotFound {
		return fmt.Errorf("container %s not found. Run discovery mode first: reactor run --discovery-mode", containerName)
	}

	// Get container diff
	changes, err := dockerService.ContainerDiff(ctx, containerInfo.ID)
	if err != nil {
		return fmt.Errorf("failed to get container diff: %w", err)
	}

	// Display changes
	if len(changes) == 0 {
		fmt.Println("No changes detected in container filesystem.")
		return nil
	}

	fmt.Printf("Container filesystem changes for %s:\n", containerName)
	for _, change := range changes {
		fmt.Printf("%s %s\n", change.Kind, change.Path)
	}

	return nil
}

func buildCmdHandler(cmd *cobra.Command, args []string) error {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return err
	}

	// Load and validate configuration
	configService := config.NewService()
	resolved, err := configService.ResolveConfiguration()
	if err != nil {
		return err
	}

	// Check if build configuration is present
	if resolved.Build == nil {
		return fmt.Errorf("no build configuration found in devcontainer.json. Add a 'build' property to enable building")
	}

	// Initialize Docker service
	ctx := context.Background()
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Check Docker daemon health
	if err := dockerService.CheckHealth(ctx); err != nil {
		return fmt.Errorf("docker daemon not available: %w", err)
	}

	// Create a minimal up config to build the image
	// Get current working directory as project directory
	projectDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create build spec from resolved configuration by calling orchestrator's function
	// First change to project directory temporarily to ensure paths work correctly
	originalWD, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	if err := os.Chdir(projectDirectory); err != nil {
		return fmt.Errorf("failed to change to project directory %s: %w", projectDirectory, err)
	}

	// Create BuildSpec from resolved configuration using the same logic as orchestrator
	if resolved.Build == nil {
		return fmt.Errorf("build configuration is nil")
	}

	// Find the devcontainer.json file to determine context base directory
	configPath, found, err := config.FindDevContainerFile(resolved.ProjectRoot)
	if err != nil {
		return fmt.Errorf("failed to find devcontainer.json: %w", err)
	}
	if !found {
		return fmt.Errorf("devcontainer.json not found")
	}

	// Get directory containing devcontainer.json
	configDir := filepath.Dir(configPath)

	// Resolve build context relative to devcontainer.json directory
	var contextPath string
	if resolved.Build.Context != "" {
		if filepath.IsAbs(resolved.Build.Context) {
			contextPath = resolved.Build.Context
		} else {
			contextPath = filepath.Join(configDir, resolved.Build.Context)
		}
	} else {
		// Default context to same directory as devcontainer.json
		contextPath = configDir
	}

	// Clean the path
	contextPath = filepath.Clean(contextPath)

	// Dockerfile defaults to "Dockerfile" if not specified
	dockerfile := resolved.Build.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	// Create image name using project hash
	imageName := fmt.Sprintf("reactor-build:%s", resolved.ProjectHash)

	buildSpec := docker.BuildSpec{
		Dockerfile: dockerfile,
		Context:    contextPath,
		ImageName:  imageName,
	}

	// Force rebuild for explicit build command
	if err := dockerService.BuildImage(ctx, buildSpec, true); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("Build completed successfully.\n")
	return nil
}

func accountsListHandler(cmd *cobra.Command, args []string) error {
	configService := config.NewService()
	return configService.ListAccounts()
}

func accountsShowHandler(cmd *cobra.Command, args []string) error {
	configService := config.NewService()
	resolved, err := configService.ResolveConfiguration()
	if err != nil {
		return err
	}

	fmt.Printf("Current account: %s\n", resolved.Account)
	return nil
}

func accountsSetHandler(cmd *cobra.Command, args []string) error {
	// Find the devcontainer.json file to show where to edit
	configPath, found, err := config.FindDevContainerFile(".")
	if err != nil {
		return fmt.Errorf("error finding devcontainer.json: %w", err)
	}
	if !found {
		return fmt.Errorf("no devcontainer.json found. Run 'reactor init' to create one")
	}

	fmt.Printf("To set the account, edit the 'customizations.reactor.account' field in:\n")
	fmt.Printf("  %s\n\n", configPath)
	fmt.Printf("Example:\n")
	fmt.Printf("{\n")
	fmt.Printf("  \"customizations\": {\n")
	fmt.Printf("    \"reactor\": {\n")
	fmt.Printf("      \"account\": \"%s\"\n", args[0])
	fmt.Printf("    }\n")
	fmt.Printf("  }\n")
	fmt.Printf("}\n")
	return nil
}

func configShowHandler(cmd *cobra.Command, args []string) error {
	configService := config.NewService()
	return configService.ShowConfiguration()
}

func configGetHandler(cmd *cobra.Command, args []string) error {
	key := args[0]
	configService := config.NewService()

	// Try to resolve configuration to show current values
	resolved, err := configService.ResolveConfiguration()
	if err != nil {
		return err
	}

	switch key {
	case "account":
		fmt.Printf("%s\n", resolved.Account)
	case "image":
		fmt.Printf("%s\n", resolved.Image)
	default:
		// Find the devcontainer.json file to show where to check
		configPath, found, findErr := config.FindDevContainerFile(".")
		if findErr != nil {
			return fmt.Errorf("error finding devcontainer.json: %w", findErr)
		}
		if !found {
			return fmt.Errorf("no devcontainer.json found")
		}

		fmt.Printf("For configuration key '%s', check your devcontainer.json file:\n", key)
		fmt.Printf("  %s\n", configPath)
		fmt.Printf("See https://containers.dev/implementors/json_reference/ for available options.\n")
	}

	return nil
}

func configSetHandler(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Find the devcontainer.json file to show where to edit
	configPath, found, err := config.FindDevContainerFile(".")
	if err != nil {
		return fmt.Errorf("error finding devcontainer.json: %w", err)
	}
	if !found {
		return fmt.Errorf("no devcontainer.json found. Run 'reactor init' to create one")
	}

	switch key {
	case "account":
		fmt.Printf("To set the account, edit the 'customizations.reactor.account' field in:\n")
		fmt.Printf("  %s\n\n", configPath)
		fmt.Printf("Example:\n")
		fmt.Printf("{\n")
		fmt.Printf("  \"customizations\": {\n")
		fmt.Printf("    \"reactor\": {\n")
		fmt.Printf("      \"account\": \"%s\"\n", value)
		fmt.Printf("    }\n")
		fmt.Printf("  }\n")
		fmt.Printf("}\n")
	case "image":
		fmt.Printf("To set the image, edit the 'image' field in:\n")
		fmt.Printf("  %s\n\n", configPath)
		fmt.Printf("Example:\n")
		fmt.Printf("{\n")
		fmt.Printf("  \"image\": \"%s\"\n", value)
		fmt.Printf("}\n")
	default:
		fmt.Printf("To set '%s', edit your devcontainer.json file:\n", key)
		fmt.Printf("  %s\n", configPath)
		fmt.Printf("See https://containers.dev/implementors/json_reference/ for available options.\n")
	}

	return nil
}

func configInitHandler(cmd *cobra.Command, args []string) error {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return err
	}

	configService := config.NewService()
	return configService.InitializeProject()
}

func versionHandler(cmd *cobra.Command, args []string) {
	fmt.Printf("reactor version %s\n", Version)
	fmt.Printf("Git commit: %s\n", GitCommit)
	fmt.Printf("Build date: %s\n", BuildDate)
}

func completionHandler(cmd *cobra.Command, args []string) error {
	shell := args[0]

	switch shell {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	default:
		return fmt.Errorf("unsupported shell: %s. Supported shells: bash, zsh, fish", shell)
	}
}

// Session command handlers
func sessionsListHandler(cmd *cobra.Command, args []string) error {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return err
	}

	// Initialize Docker service
	ctx := context.Background()
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Check Docker daemon health
	if err := dockerService.CheckHealth(ctx); err != nil {
		return fmt.Errorf("docker daemon not available: %w", err)
	}

	// List all reactor containers
	containers, err := dockerService.ListReactorContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list reactor containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No reactor containers found.")
		fmt.Println("Run 'reactor run' to create a new container session.")
		return nil
	}

	// Display containers in a table format
	fmt.Printf("%-35s %-8s %-25s %-10s\n", "CONTAINER NAME", "STATUS", "IMAGE", "UPTIME")
	fmt.Printf("%-35s %-8s %-25s %-10s\n",
		strings.Repeat("-", 35),
		strings.Repeat("-", 8),
		strings.Repeat("-", 25),
		strings.Repeat("-", 10))

	for _, container := range containers {
		status := "unknown"
		switch container.Status {
		case docker.StatusRunning:
			status = "running"
		case docker.StatusStopped:
			status = "stopped"
		case docker.StatusNotFound:
			status = "missing"
		}

		// Truncate image name if too long
		image := container.Image
		if len(image) > 25 {
			image = image[:22] + "..."
		}

		// For now, show "-" for uptime since we don't have that info easily available
		// Could be enhanced to calculate from container inspection
		uptime := "-"

		fmt.Printf("%-35s %-8s %-25s %-10s\n", container.Name, status, image, uptime)
	}

	fmt.Printf("\nFound %d reactor container(s).\n", len(containers))
	fmt.Println("Use 'reactor sessions attach <container-name>' to connect to a container.")

	return nil
}

func sessionsAttachHandler(cmd *cobra.Command, args []string) error {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return err
	}

	ctx := context.Background()

	// Initialize Docker service
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Check Docker daemon health
	if err := dockerService.CheckHealth(ctx); err != nil {
		return fmt.Errorf("docker daemon not available: %w", err)
	}

	var containerName string

	if len(args) == 0 {
		// Auto-attach to current project container
		// Load configuration to get project info
		configService := config.NewService()
		resolved, err := configService.ResolveConfiguration()
		if err != nil {
			return fmt.Errorf("failed to load project configuration: %w", err)
		}

		// Find container for current project
		containerInfo, err := dockerService.FindProjectContainer(ctx, resolved.Account, resolved.ProjectRoot, resolved.ProjectHash)
		if err != nil {
			return fmt.Errorf("failed to find project container: %w", err)
		}

		if containerInfo == nil {
			return fmt.Errorf("no container found for current project. Run 'reactor run' to create one")
		}

		containerName = containerInfo.Name
		fmt.Printf("Found container for current project: %s\n", containerName)
	} else {
		// Use specified container name
		containerName = args[0]
	}

	// Check if container exists and get its info
	containerInfo, err := dockerService.ContainerExists(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	if containerInfo.Status == docker.StatusNotFound {
		return fmt.Errorf("container '%s' not found", containerName)
	}

	// Start container if it's stopped
	if containerInfo.Status == docker.StatusStopped {
		fmt.Printf("Starting stopped container: %s\n", containerName)
		if err := dockerService.StartContainer(ctx, containerInfo.ID); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}
		fmt.Println("Container started successfully.")
	}

	// Attach to the container
	fmt.Printf("Attaching to container: %s\n", containerName)
	if err := dockerService.AttachInteractiveSession(ctx, containerInfo.ID); err != nil {
		return fmt.Errorf("failed to attach to container: %w", err)
	}

	// Show exit message
	fmt.Printf("\nSession ended. Container '%s' is still running.\n", containerName)
	fmt.Printf("Use 'docker stop %s' to stop it.\n", containerName)

	return nil
}

func sessionsCleanHandler(cmd *cobra.Command, args []string) error {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return err
	}

	ctx := context.Background()

	// Initialize Docker service
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Check Docker daemon health
	if err := dockerService.CheckHealth(ctx); err != nil {
		return fmt.Errorf("docker daemon not available: %w", err)
	}

	// List all reactor containers
	containers, err := dockerService.ListReactorContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list reactor containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No reactor containers found to clean up.")
		return nil
	}

	fmt.Printf("Found %d reactor containers to clean up:\n", len(containers))
	for _, container := range containers {
		fmt.Printf("  %s (%s)\n", container.Name, container.Status)
	}

	// Clean up all containers using standard removal
	removedCount := 0
	for _, container := range containers {
		fmt.Printf("Removing container: %s ... ", container.Name)

		// Use standard container removal
		err := dockerService.RemoveContainer(ctx, container.ID)
		if err != nil {
			fmt.Printf("failed: %v\n", err)
			// Continue with other containers
		} else {
			fmt.Println("done")
			removedCount++
		}
	}

	fmt.Printf("\nSuccessfully cleaned up %d of %d reactor containers.\n", removedCount, len(containers))
	return nil
}

func newWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage multi-container workspaces",
		Long: `Manage collections of related dev container services as a single workspace.

The workspace commands allow you to orchestrate multiple dev containers defined
in a reactor-workspace.yml file. This is ideal for microservice development
where you need to run multiple services simultaneously.

Examples:
  reactor workspace validate           # Validate workspace configuration
  reactor workspace list             # List services and their status
  reactor workspace up               # Start all services
  reactor workspace down             # Stop all services

For more details, see the full documentation.`,
	}

	// Add --file / -f flag to all workspace commands
	cmd.PersistentFlags().StringP("file", "f", "", "Path to workspace file (default: reactor-workspace.yml)")

	// Add subcommands for PR 1 and PR 2
	cmd.AddCommand(newWorkspaceValidateCmd())
	cmd.AddCommand(newWorkspaceListCmd())
	cmd.AddCommand(newWorkspaceUpCmd())
	cmd.AddCommand(newWorkspaceDownCmd())
	cmd.AddCommand(newWorkspaceExecCmd())

	return cmd
}

func newWorkspaceValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate workspace configuration",
		Long: `Validate the reactor-workspace.yml file and all service configurations.

This command parses the workspace file and validates:
- Workspace file syntax and version
- Service path existence and accessibility  
- Each service's devcontainer.json file validity
- Path traversal security checks

Examples:
  reactor workspace validate                    # Validate default workspace file
  reactor workspace validate -f my-workspace.yml  # Validate specific file

For more details, see the full documentation.`,
		RunE: workspaceValidateHandler,
	}
}

func newWorkspaceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List workspace services and their status",
		Long: `List all services defined in the workspace with their container status.

Shows each service name, path, account, and current container status (running,
stopped, or not found). This gives you a complete overview of your workspace
state at a glance.

Examples:
  reactor workspace list                       # List services in default workspace
  reactor workspace list -f my-workspace.yml  # List services in specific workspace

For more details, see the full documentation.`,
		RunE: workspaceListHandler,
	}
}

// workspaceValidateHandler validates a workspace file and all its services
func workspaceValidateHandler(cmd *cobra.Command, args []string) error {
	// Get workspace file path from flag or use default
	workspaceFile, _ := cmd.Flags().GetString("file")

	// Handle workspace file path
	var workspacePath string
	if workspaceFile != "" {
		// User specified a specific file path
		if filepath.Ext(workspaceFile) != "" {
			// It's a file path, use it directly
			workspacePath = workspaceFile
		} else {
			// It's a directory, find workspace file in it
			var found bool
			var err error
			workspacePath, found, err = workspace.FindWorkspaceFile(workspaceFile)
			if err != nil {
				return fmt.Errorf("error finding workspace file: %w", err)
			}
			if !found {
				return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in directory: %s", workspaceFile)
			}
		}

		// Check if the specified file exists
		if _, err := os.Stat(workspacePath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("workspace file not found: %s", workspacePath)
			}
			return fmt.Errorf("error accessing workspace file %s: %w", workspacePath, err)
		}
	} else {
		// No file specified, find default workspace file in current directory
		var found bool
		var err error
		workspacePath, found, err = workspace.FindWorkspaceFile("")
		if err != nil {
			return fmt.Errorf("error finding workspace file: %w", err)
		}
		if !found {
			return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in current directory")
		}
	}

	// Parse and validate workspace file
	ws, err := workspace.ParseWorkspaceFile(workspacePath)
	if err != nil {
		return fmt.Errorf("workspace validation failed: %w", err)
	}

	fmt.Printf("✓ Workspace file valid: %s\n", workspacePath)
	fmt.Printf("  Version: %s\n", ws.Version)
	fmt.Printf("  Services: %d\n\n", len(ws.Services))

	// Validate each service's devcontainer.json
	validServices := 0
	for serviceName, service := range ws.Services {
		fmt.Printf("Validating service '%s':\n", serviceName)
		fmt.Printf("  Path: %s\n", service.Path)
		if service.Account != "" {
			fmt.Printf("  Account: %s\n", service.Account)
		}

		// Resolve service path relative to workspace file
		workspaceDir := filepath.Dir(workspacePath)
		servicePath := service.Path
		if !filepath.IsAbs(servicePath) {
			servicePath = filepath.Join(workspaceDir, service.Path)
		}
		servicePath = filepath.Clean(servicePath)

		// Check for devcontainer.json in service directory
		devcontainerPath, found, err := config.FindDevContainerFile(servicePath)
		if err != nil {
			fmt.Printf("  ✗ Error checking devcontainer.json: %v\n\n", err)
			continue
		}
		if !found {
			fmt.Printf("  ✗ No devcontainer.json found\n\n")
			continue
		}

		// Try to parse the devcontainer.json to validate it
		configService := config.NewServiceWithRoot(servicePath)
		_, err = configService.ResolveConfiguration()
		if err != nil {
			fmt.Printf("  ✗ Invalid devcontainer.json: %v\n\n", err)
			continue
		}

		fmt.Printf("  ✓ devcontainer.json: %s\n\n", devcontainerPath)
		validServices++
	}

	// Summary
	totalServices := len(ws.Services)
	if validServices == totalServices {
		fmt.Printf("✓ All %d services validated successfully\n", totalServices)
	} else {
		fmt.Printf("✗ %d of %d services validated successfully\n", validServices, totalServices)
		return fmt.Errorf("workspace validation failed: %d service(s) have configuration errors", totalServices-validServices)
	}

	return nil
}

// workspaceListHandler lists services and their container status
func workspaceListHandler(cmd *cobra.Command, args []string) error {
	// Get workspace file path from flag or use default
	workspaceFile, _ := cmd.Flags().GetString("file")

	// Handle workspace file path
	var workspacePath string
	if workspaceFile != "" {
		// User specified a specific file path
		if filepath.Ext(workspaceFile) != "" {
			// It's a file path, use it directly
			workspacePath = workspaceFile
		} else {
			// It's a directory, find workspace file in it
			var found bool
			var err error
			workspacePath, found, err = workspace.FindWorkspaceFile(workspaceFile)
			if err != nil {
				return fmt.Errorf("error finding workspace file: %w", err)
			}
			if !found {
				return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in directory: %s", workspaceFile)
			}
		}

		// Check if the specified file exists
		if _, err := os.Stat(workspacePath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("workspace file not found: %s", workspacePath)
			}
			return fmt.Errorf("error accessing workspace file %s: %w", workspacePath, err)
		}
	} else {
		// No file specified, find default workspace file in current directory
		var found bool
		var err error
		workspacePath, found, err = workspace.FindWorkspaceFile("")
		if err != nil {
			return fmt.Errorf("error finding workspace file: %w", err)
		}
		if !found {
			return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in current directory")
		}
	}

	// Parse workspace file
	ws, err := workspace.ParseWorkspaceFile(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to parse workspace file: %w", err)
	}

	// Initialize Docker service to check container status
	ctx := context.Background()
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Check Docker daemon health
	if err := dockerService.CheckHealth(ctx); err != nil {
		return fmt.Errorf("docker daemon not available: %w", err)
	}

	// Generate workspace hash for container labeling
	workspaceHash, err := workspace.GenerateWorkspaceHash(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to generate workspace hash: %w", err)
	}

	fmt.Printf("Workspace: %s\n", workspacePath)
	fmt.Printf("Services: %d\n\n", len(ws.Services))

	// Display header
	fmt.Printf("%-15s %-30s %-15s %-10s\n", "SERVICE", "PATH", "ACCOUNT", "STATUS")
	fmt.Printf("%-15s %-30s %-15s %-10s\n",
		strings.Repeat("-", 15),
		strings.Repeat("-", 30),
		strings.Repeat("-", 15),
		strings.Repeat("-", 10))

	// Check status for each service
	for serviceName, service := range ws.Services {
		// Resolve service path for project hash calculation
		workspaceDir := filepath.Dir(workspacePath)
		servicePath := service.Path
		if !filepath.IsAbs(servicePath) {
			servicePath = filepath.Join(workspaceDir, service.Path)
		}
		servicePath = filepath.Clean(servicePath)

		// Generate expected container name using workspace naming convention
		projectHash := config.GenerateProjectHash(servicePath)
		expectedContainerName := fmt.Sprintf("reactor-ws-%s-%s", serviceName, projectHash)

		// Check container status
		containerInfo, err := dockerService.ContainerExists(ctx, expectedContainerName)
		status := "not found"
		if err == nil {
			switch containerInfo.Status {
			case docker.StatusRunning:
				status = "running"
			case docker.StatusStopped:
				status = "stopped"
			case docker.StatusNotFound:
				status = "not found"
			}
		}

		// Truncate path if too long for display
		displayPath := service.Path
		if len(displayPath) > 30 {
			displayPath = displayPath[:27] + "..."
		}

		// Get account (from service override or devcontainer.json)
		account := service.Account
		if account == "" {
			// Try to read account from service's devcontainer.json
			configService := config.NewServiceWithRoot(servicePath)
			if resolved, err := configService.ResolveConfiguration(); err == nil {
				account = resolved.Account
			} else {
				account = "-"
			}
		}
		if len(account) > 15 {
			account = account[:12] + "..."
		}

		fmt.Printf("%-15s %-30s %-15s %-10s\n", serviceName, displayPath, account, status)
	}

	fmt.Printf("\nWorkspace Hash: %s\n", workspaceHash[:16]+"...") // Show first 16 chars of hash

	return nil
}

func newWorkspaceUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [service...]",
		Short: "Start workspace services",
		Long: `Start all or specific services defined in the workspace.

This command orchestrates multiple dev containers in parallel, with pre-flight
port conflict checking and colored output streaming. Each service is started
using its devcontainer.json configuration with workspace-specific labeling
and naming conventions.

Examples:
  reactor workspace up                    # Start all services
  reactor workspace up api frontend      # Start specific services  
  reactor workspace up -f my-workspace.yml api  # Use specific workspace file

The command will:
- Validate all service configurations before starting any containers
- Check for host port conflicts across services
- Start services in parallel with goroutines
- Stream output with service-specific color prefixes
- Apply workspace labels for container tracking
- Report final success/failure status

For more details, see the full documentation.`,
		RunE: workspaceUpHandler,
	}

	// Add flags specific to the up command
	cmd.Flags().Bool("rebuild", false, "Force rebuild of container images")
	cmd.Flags().StringArrayP("port", "p", nil, "Port forwarding (host:container)")
	cmd.Flags().Bool("discovery", false, "Enable discovery mode (no mounts)")
	cmd.Flags().Bool("docker-host", false, "Enable Docker host integration (dangerous)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return cmd
}

func newWorkspaceDownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down [service...]",
		Short: "Stop workspace services",
		Long: `Stop and remove all or specific services defined in the workspace.

This command finds containers associated with the workspace using workspace
labels and stops them in parallel. If no services are specified, all services
in the workspace will be stopped.

Examples:
  reactor workspace down                    # Stop all services
  reactor workspace down api frontend      # Stop specific services  
  reactor workspace down -f my-workspace.yml # Use specific workspace file

Key features:
- Parallel execution for faster shutdown
- Workspace label-based container discovery
- Graceful container stopping and removal
- Progress reporting for each service

For more details, see the full documentation.`,
		RunE: workspaceDownHandler,
	}

	return cmd
}

func newWorkspaceExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <service> -- <command...>",
		Short: "Execute command in workspace service",
		Long: `Execute a command in a running workspace service container.

This command finds the container for the specified service using workspace
labels and naming conventions, then executes the provided command with
full I/O streaming.

Examples:
  reactor workspace exec api -- bash                   # Interactive shell
  reactor workspace exec api -- npm test               # Run tests  
  reactor workspace exec api -- ls -la /home           # Command with flags
  reactor workspace exec -f my-workspace.yml api -- ls # Use specific workspace

The service must already be running (started with 'reactor workspace up').
Use '--' to separate the service name from the command to execute.

For more details, see the full documentation.`,
		Args:                  cobra.MinimumNArgs(1),
		RunE:                  workspaceExecHandler,
		DisableFlagsInUseLine: true,
	}

	return cmd
}

// workspaceUpHandler starts all or specific services in a workspace
func workspaceUpHandler(cmd *cobra.Command, args []string) error {
	// Get workspace file path from flag or use default
	workspaceFile, _ := cmd.Flags().GetString("file")

	// Get command-specific flags
	forceRebuild, _ := cmd.Flags().GetBool("rebuild")
	portMappings, _ := cmd.Flags().GetStringArray("port")
	discoveryMode, _ := cmd.Flags().GetBool("discovery")
	dockerHostIntegration, _ := cmd.Flags().GetBool("docker-host")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Handle workspace file path (reusing existing logic pattern)
	var workspacePath string
	if workspaceFile != "" {
		if filepath.Ext(workspaceFile) != "" {
			workspacePath = workspaceFile
		} else {
			var found bool
			var err error
			workspacePath, found, err = workspace.FindWorkspaceFile(workspaceFile)
			if err != nil {
				return fmt.Errorf("error finding workspace file: %w", err)
			}
			if !found {
				return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in directory: %s", workspaceFile)
			}
		}

		if _, err := os.Stat(workspacePath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("workspace file not found: %s", workspacePath)
			}
			return fmt.Errorf("error accessing workspace file %s: %w", workspacePath, err)
		}
	} else {
		var found bool
		var err error
		workspacePath, found, err = workspace.FindWorkspaceFile("")
		if err != nil {
			return fmt.Errorf("error finding workspace file: %w", err)
		}
		if !found {
			return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in current directory")
		}
	}

	// Parse workspace file
	ws, err := workspace.ParseWorkspaceFile(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to parse workspace file: %w", err)
	}

	// Determine which services to start
	var servicesToStart []string
	if len(args) == 0 {
		// Start all services
		for serviceName := range ws.Services {
			servicesToStart = append(servicesToStart, serviceName)
		}
	} else {
		// Start specified services
		for _, serviceName := range args {
			if _, exists := ws.Services[serviceName]; !exists {
				return fmt.Errorf("service '%s' not found in workspace", serviceName)
			}
			servicesToStart = append(servicesToStart, serviceName)
		}
	}

	// Generate workspace hash for labeling
	workspaceHash, err := workspace.GenerateWorkspaceHash(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to generate workspace hash: %w", err)
	}

	fmt.Printf("Starting workspace services: %v\n", servicesToStart)
	fmt.Printf("Workspace: %s\n", workspacePath)

	// Check if workspace is already running
	if err := checkWorkspaceNotRunning(workspaceHash, servicesToStart); err != nil {
		return err
	}

	fmt.Println()

	// Pre-flight validation: check all service configurations and port conflicts
	if err := validateServicesAndPorts(ws, servicesToStart, workspacePath, portMappings); err != nil {
		return fmt.Errorf("pre-flight validation failed: %w", err)
	}

	// Start services in parallel
	return startServicesInParallel(ws, servicesToStart, workspacePath, workspaceHash, orchestrator.UpConfig{
		ForceRebuild:          forceRebuild,
		CLIPortMappings:       portMappings,
		DiscoveryMode:         discoveryMode,
		DockerHostIntegration: dockerHostIntegration,
		Verbose:               verbose,
	})
}

// workspaceExecHandler executes a command in a workspace service container
func workspaceExecHandler(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("service name and command required (use: reactor workspace exec <service> -- <command>)")
	}

	serviceName := args[0]
	command := args[1:]

	// Get workspace file path from flag or use default
	workspaceFile, _ := cmd.Flags().GetString("file")

	// Handle workspace file path (reusing existing logic pattern)
	var workspacePath string
	if workspaceFile != "" {
		if filepath.Ext(workspaceFile) != "" {
			workspacePath = workspaceFile
		} else {
			var found bool
			var err error
			workspacePath, found, err = workspace.FindWorkspaceFile(workspaceFile)
			if err != nil {
				return fmt.Errorf("error finding workspace file: %w", err)
			}
			if !found {
				return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in directory: %s", workspaceFile)
			}
		}

		if _, err := os.Stat(workspacePath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("workspace file not found: %s", workspacePath)
			}
			return fmt.Errorf("error accessing workspace file %s: %w", workspacePath, err)
		}
	} else {
		var found bool
		var err error
		workspacePath, found, err = workspace.FindWorkspaceFile("")
		if err != nil {
			return fmt.Errorf("error finding workspace file: %w", err)
		}
		if !found {
			return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in current directory")
		}
	}

	// Parse workspace file
	ws, err := workspace.ParseWorkspaceFile(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to parse workspace file: %w", err)
	}

	// Check if service exists
	if _, exists := ws.Services[serviceName]; !exists {
		return fmt.Errorf("service '%s' not found in workspace", serviceName)
	}

	// Generate workspace hash for container labeling
	workspaceHash, err := workspace.GenerateWorkspaceHash(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to generate workspace hash: %w", err)
	}

	// Initialize Docker service
	ctx := context.Background()
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Find container using workspace labels instead of reconstructing name
	client := dockerService.GetClient()
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.instance=%s", workspaceHash))
	filterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.service=%s", serviceName))

	containers, err := client.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return fmt.Errorf("container for service '%s' not found - start it first with 'reactor workspace up %s'", serviceName, serviceName)
	}

	if len(containers) > 1 {
		return fmt.Errorf("multiple containers found for service '%s' - this shouldn't happen", serviceName)
	}

	container := containers[0]
	if container.State != "running" {
		return fmt.Errorf("container for service '%s' is not running (status: %s) - start it first with 'reactor workspace up %s'", serviceName, container.State, serviceName)
	}

	// Execute the command in the container
	fmt.Printf("Executing command in service '%s': %v\n", serviceName, command)
	return dockerService.ExecuteInteractiveCommand(ctx, container.ID, command)
}

// workspaceDownHandler stops and removes all or specific services in a workspace
func workspaceDownHandler(cmd *cobra.Command, args []string) error {
	// Get workspace file path from flag or use default
	workspaceFile, _ := cmd.Flags().GetString("file")

	// Handle workspace file path (reusing existing logic pattern)
	var workspacePath string
	if workspaceFile != "" {
		if filepath.Ext(workspaceFile) != "" {
			workspacePath = workspaceFile
		} else {
			// It's a directory, find workspace file in it
			var found bool
			var err error
			workspacePath, found, err = workspace.FindWorkspaceFile(workspaceFile)
			if err != nil {
				return fmt.Errorf("error finding workspace file: %w", err)
			}
			if !found {
				return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in directory: %s", workspaceFile)
			}
		}
		// Verify file exists
		if _, err := os.Stat(workspacePath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("workspace file not found: %s", workspacePath)
			}
			return fmt.Errorf("error accessing workspace file %s: %w", workspacePath, err)
		}
	} else {
		// No file specified, find default workspace file in current directory
		var found bool
		var err error
		workspacePath, found, err = workspace.FindWorkspaceFile("")
		if err != nil {
			return fmt.Errorf("error finding workspace file: %w", err)
		}
		if !found {
			return fmt.Errorf("no reactor-workspace.yml or reactor-workspace.yaml found in current directory")
		}
	}

	// Parse workspace file
	ws, err := workspace.ParseWorkspaceFile(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to parse workspace file: %w", err)
	}

	// Determine which services to stop
	var servicesToStop []string
	if len(args) == 0 {
		// No services specified, stop all services in workspace
		for serviceName := range ws.Services {
			servicesToStop = append(servicesToStop, serviceName)
		}
	} else {
		// Specific services specified, validate they exist
		for _, serviceName := range args {
			if _, exists := ws.Services[serviceName]; !exists {
				return fmt.Errorf("service '%s' not found in workspace", serviceName)
			}
		}
		servicesToStop = args
	}

	// Generate workspace hash for container labeling
	workspaceHash, err := workspace.GenerateWorkspaceHash(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to generate workspace hash: %w", err)
	}

	fmt.Printf("Stopping workspace services: %v\n", servicesToStop)
	fmt.Printf("Workspace: %s\n", workspacePath)

	// Stop services in parallel
	return stopServicesInParallel(servicesToStop, workspaceHash)
}

// validateServicesAndPorts performs pre-flight validation for workspace services
func validateServicesAndPorts(ws *workspace.Workspace, servicesToStart []string, workspacePath string, cliPorts []string) error {
	workspaceDir := filepath.Dir(workspacePath)
	allHostPorts := make(map[int][]string) // Map of host port to services using it

	fmt.Printf("Pre-flight validation:\n")

	// Validate each service configuration and collect port mappings
	for _, serviceName := range servicesToStart {
		service := ws.Services[serviceName]
		fmt.Printf("  Validating service '%s'...\n", serviceName)

		// Resolve service path
		servicePath := service.Path
		if !filepath.IsAbs(servicePath) {
			servicePath = filepath.Join(workspaceDir, service.Path)
		}
		servicePath = filepath.Clean(servicePath)

		// Check devcontainer.json exists and is valid
		configService := config.NewServiceWithRoot(servicePath)
		resolved, err := configService.ResolveConfiguration()
		if err != nil {
			return fmt.Errorf("service '%s' configuration invalid: %w", serviceName, err)
		}

		// Collect port mappings from devcontainer.json
		for _, port := range resolved.ForwardPorts {
			if existing, exists := allHostPorts[port.HostPort]; exists {
				allHostPorts[port.HostPort] = append(existing, serviceName)
			} else {
				allHostPorts[port.HostPort] = []string{serviceName}
			}
		}
	}

	// Parse and validate CLI port mappings
	for _, portStr := range cliPorts {
		parts := strings.Split(portStr, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid CLI port mapping format '%s': expected 'host:container'", portStr)
		}

		hostPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid CLI host port '%s': must be a number", parts[0])
		}

		// CLI ports can override devcontainer ports, but we still track them
		if existing, exists := allHostPorts[hostPort]; exists {
			fmt.Printf("  ⚠️  CLI port %d overrides devcontainer.json port for services: %v\n", hostPort, existing)
		}
		allHostPorts[hostPort] = []string{"CLI"}
	}

	// Check for port conflicts between services
	var conflicts []string
	for hostPort, services := range allHostPorts {
		if len(services) > 1 && !contains(services, "CLI") {
			conflicts = append(conflicts, fmt.Sprintf("port %d used by services: %v", hostPort, services))
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("port conflicts detected:\n  - %s", strings.Join(conflicts, "\n  - "))
	}

	fmt.Printf("  ✓ All service configurations valid\n")
	fmt.Printf("  ✓ No port conflicts detected\n\n")
	return nil
}

// startServicesInParallel starts multiple services using goroutines
func startServicesInParallel(ws *workspace.Workspace, servicesToStart []string, workspacePath, workspaceHash string, baseConfig orchestrator.UpConfig) error {
	workspaceDir := filepath.Dir(workspacePath)

	// Channel for collecting results
	type serviceResult struct {
		serviceName string
		err         error
		containerID string
	}

	resultChan := make(chan serviceResult, len(servicesToStart))

	// Start services in parallel
	for _, serviceName := range servicesToStart {
		go func(name string) {
			service := ws.Services[name]

			// Resolve service path
			servicePath := service.Path
			if !filepath.IsAbs(servicePath) {
				servicePath = filepath.Join(workspaceDir, service.Path)
			}
			servicePath = filepath.Clean(servicePath)

			// Create service-specific orchestrator config
			serviceConfig := baseConfig
			serviceConfig.ProjectDirectory = servicePath
			serviceConfig.AccountOverride = service.Account
			serviceConfig.NamePrefix = fmt.Sprintf("reactor-ws-%s-", name)

			// Add workspace labels
			if serviceConfig.Labels == nil {
				serviceConfig.Labels = make(map[string]string)
			}
			serviceConfig.Labels["com.reactor.workspace.instance"] = workspaceHash
			serviceConfig.Labels["com.reactor.workspace.service"] = name

			// Start the service
			ctx := context.Background()
			fmt.Printf("[%s] Starting service...\n", name)

			resolved, containerID, err := orchestrator.Up(ctx, serviceConfig)
			if err != nil {
				fmt.Printf("[%s] ❌ Failed: %v\n", name, err)
				resultChan <- serviceResult{name, err, ""}
				return
			}

			fmt.Printf("[%s] ✅ Started successfully (container: %s)\n", name, containerID)
			if resolved != nil && len(resolved.ForwardPorts) > 0 {
				fmt.Printf("[%s] Port mappings: ", name)
				for i, port := range resolved.ForwardPorts {
					if i > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("%d->%d", port.HostPort, port.ContainerPort)
				}
				fmt.Printf("\n")
			}

			resultChan <- serviceResult{name, nil, containerID}
		}(serviceName)
	}

	// Collect results
	var successCount, failCount int
	var errors []string

	for i := 0; i < len(servicesToStart); i++ {
		result := <-resultChan
		if result.err != nil {
			failCount++
			errors = append(errors, fmt.Sprintf("%s: %v", result.serviceName, result.err))
		} else {
			successCount++
		}
	}

	// Print final summary
	fmt.Printf("\n=== Workspace Start Summary ===\n")
	fmt.Printf("✅ Started successfully: %d/%d services\n", successCount, len(servicesToStart))
	if failCount > 0 {
		fmt.Printf("❌ Failed to start: %d/%d services\n", failCount, len(servicesToStart))
		for _, errMsg := range errors {
			fmt.Printf("  - %s\n", errMsg)
		}
		return fmt.Errorf("%d service(s) failed to start", failCount)
	}

	fmt.Printf("\nWorkspace is ready! 🚀\n")
	return nil
}

// stopServicesInParallel stops workspace services in parallel using their workspace labels
func stopServicesInParallel(servicesToStop []string, workspaceHash string) error {
	ctx := context.Background()
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			log.Printf("Warning: failed to close Docker service: %v", err)
		}
	}()

	client := dockerService.GetClient()

	// Channel for collecting results
	type serviceResult struct {
		serviceName string
		err         error
		containerID string
	}

	resultChan := make(chan serviceResult, len(servicesToStop))

	// Stop services in parallel
	for _, serviceName := range servicesToStop {
		go func(name string) {
			fmt.Printf("[%s] Looking for container...\n", name)

			// Find container using workspace labels
			filterArgs := filters.NewArgs()
			filterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.instance=%s", workspaceHash))
			filterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.service=%s", name))

			containers, err := client.ContainerList(ctx, container.ListOptions{
				Filters: filterArgs,
				All:     true, // Include stopped containers
			})
			if err != nil {
				fmt.Printf("[%s] ❌ Failed to list containers: %v\n", name, err)
				resultChan <- serviceResult{name, err, ""}
				return
			}

			if len(containers) == 0 {
				fmt.Printf("[%s] ⚠️  No container found (already removed or never created)\n", name)
				resultChan <- serviceResult{name, nil, ""}
				return
			}

			if len(containers) > 1 {
				fmt.Printf("[%s] ⚠️  Multiple containers found, stopping all\n", name)
			}

			// Stop and remove each container found
			for _, cont := range containers {
				fmt.Printf("[%s] Stopping container %s...\n", name, cont.ID[:12])

				// Stop the container first if it's running
				if cont.State == "running" {
					timeout := 10
					if err := client.ContainerStop(ctx, cont.ID, container.StopOptions{Timeout: &timeout}); err != nil {
						fmt.Printf("[%s] ⚠️  Failed to stop container: %v\n", name, err)
					}
				}

				// Remove the container
				if err := client.ContainerRemove(ctx, cont.ID, container.RemoveOptions{
					Force: true, // Force removal even if running
				}); err != nil {
					fmt.Printf("[%s] ❌ Failed to remove container: %v\n", name, err)
					resultChan <- serviceResult{name, err, cont.ID}
					return
				}

				fmt.Printf("[%s] ✅ Stopped and removed container %s\n", name, cont.ID[:12])
			}

			resultChan <- serviceResult{name, nil, containers[0].ID}
		}(serviceName)
	}

	// Collect results
	var successCount, failCount int
	var errors []string

	for i := 0; i < len(servicesToStop); i++ {
		result := <-resultChan
		if result.err != nil {
			failCount++
			errors = append(errors, fmt.Sprintf("%s: %v", result.serviceName, result.err))
		} else {
			successCount++
		}
	}

	// Print final summary
	fmt.Printf("\n=== Workspace Stop Summary ===\n")
	fmt.Printf("✅ Stopped successfully: %d/%d services\n", successCount, len(servicesToStop))
	if failCount > 0 {
		fmt.Printf("❌ Failed to stop: %d/%d services\n", failCount, len(servicesToStop))
		for _, errMsg := range errors {
			fmt.Printf("  - %s\n", errMsg)
		}
		return fmt.Errorf("%d service(s) failed to stop", failCount)
	}

	fmt.Printf("\nWorkspace stopped! 🛑\n")
	return nil
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// checkWorkspaceNotRunning checks if any of the services are already running
func checkWorkspaceNotRunning(workspaceHash string, servicesToStart []string) error {
	ctx := context.Background()
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			log.Printf("Warning: failed to close Docker service: %v", err)
		}
	}()

	client := dockerService.GetClient()

	// Find any running containers for this workspace
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.instance=%s", workspaceHash))
	filterArgs.Add("status", "running")

	runningContainers, err := client.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to check existing containers: %w", err)
	}

	if len(runningContainers) == 0 {
		return nil // No running containers, safe to start
	}

	// Check which services are already running
	var runningServices []string
	var conflictingServices []string

	for _, container := range runningContainers {
		if serviceName, exists := container.Labels["com.reactor.workspace.service"]; exists {
			runningServices = append(runningServices, serviceName)
			// Check if we're trying to start a service that's already running
			for _, serviceToStart := range servicesToStart {
				if serviceName == serviceToStart {
					conflictingServices = append(conflictingServices, serviceName)
				}
			}
		}
	}

	if len(conflictingServices) > 0 {
		fmt.Printf("⚠️  Some services are already running: %v\n", conflictingServices)
		fmt.Printf("   All running services in this workspace: %v\n", runningServices)
		fmt.Printf("   Use 'reactor workspace exec <service> -- <command>' to run commands in existing containers\n")
		fmt.Printf("   Or stop the workspace first with: docker stop %s\n",
			strings.Join(getContainerNames(runningContainers), " "))
		return fmt.Errorf("workspace services already running")
	}

	// Some services are running but not conflicting - just inform the user
	if len(runningServices) > 0 {
		fmt.Printf("ℹ️  Other services already running in this workspace: %v\n", runningServices)
	}

	return nil
}

// getContainerNames extracts container names from a list of containers
func getContainerNames(containers []types.Container) []string {
	var names []string
	for _, container := range containers {
		// Container names have a leading slash, remove it
		name := container.Names[0]
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}
		names = append(names, name)
	}
	return names
}
