package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dyluth/reactor/pkg/config"
	"github.com/dyluth/reactor/pkg/core"
	"github.com/dyluth/reactor/pkg/docker"
	"github.com/spf13/cobra"
)

// Build-time variables injected via linker flags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// PortMapping represents a port forwarding configuration
type PortMapping struct {
	HostPort      int
	ContainerPort int
}

// parsePortMappings parses and validates port mapping strings in the format "host:container"
func parsePortMappings(portStrings []string) ([]PortMapping, error) {
	var mappings []PortMapping

	for _, portStr := range portStrings {
		parts := strings.Split(portStr, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid port mapping format '%s': expected 'host:container'", portStr)
		}

		hostPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid host port '%s': must be a number", parts[0])
		}

		containerPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid container port '%s': must be a number", parts[1])
		}

		// Validate port ranges
		if hostPort < 1 || hostPort > 65535 {
			return nil, fmt.Errorf("host port %d is out of valid range (1-65535)", hostPort)
		}
		if containerPort < 1 || containerPort > 65535 {
			return nil, fmt.Errorf("container port %d is out of valid range (1-65535)", containerPort)
		}

		mappings = append(mappings, PortMapping{
			HostPort:      hostPort,
			ContainerPort: containerPort,
		})
	}

	return mappings, nil
}

// checkPortConflicts checks if any of the host ports are already in use
func checkPortConflicts(mappings []PortMapping) []int {
	var conflictPorts []int

	for _, pm := range mappings {
		// Try to listen on the host port briefly to check if it's available
		addr := fmt.Sprintf(":%d", pm.HostPort)
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			// Port is open (something is listening)
			_ = conn.Close()
			conflictPorts = append(conflictPorts, pm.HostPort)
		} else {
			// Try to bind to the port to see if it's available
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				// Port is likely in use or restricted
				conflictPorts = append(conflictPorts, pm.HostPort)
			} else {
				_ = listener.Close()
			}
		}
	}

	return conflictPorts
}

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
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newSessionsCmd())
	cmd.AddCommand(newDiffCmd())
	cmd.AddCommand(newAccountsCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run AI agent in containerized environment",
		Long: `Run an AI agent in a containerized development environment with
persistent configuration and session management.

The run command provisions a Docker container with your AI agent and project
files, then attaches you to an interactive session. Containers are automatically
reused when possible for fast startup times.

Examples:
  reactor run                              # Use project configuration
  reactor run --provider claude           # Override provider to claude
  reactor run --image python --danger     # Use Python image with danger mode
  reactor run --account work-account      # Use specific account for isolation

For more details, see the full documentation.`,
		RunE: runCmdHandler,
	}

	// Add flags
	cmd.Flags().String("provider", "", "AI provider to use (claude, gemini, custom)")
	cmd.Flags().String("account", "", "Account for configuration isolation")
	cmd.Flags().String("image", "", "Container image (base, python, go, or custom URL)")
	cmd.Flags().Bool("danger", false, "Enable dangerous permissions for AI agent")
	cmd.Flags().Bool("discovery-mode", false, "Run with no mounts for configuration discovery")
	cmd.Flags().Bool("docker-host-integration", false, "Mount host Docker socket (DANGEROUS - use only with trusted images)")
	cmd.Flags().StringSliceP("port", "p", []string{}, "Port forwarding (host:container), can be used multiple times")

	return cmd
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
func runCmdHandler(cmd *cobra.Command, args []string) error {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return err
	}

	// Get CLI flags
	provider, _ := cmd.Flags().GetString("provider")
	account, _ := cmd.Flags().GetString("account")
	image, _ := cmd.Flags().GetString("image")
	danger, _ := cmd.Flags().GetBool("danger")
	discoveryMode, _ := cmd.Flags().GetBool("discovery-mode")
	dockerHostIntegration, _ := cmd.Flags().GetBool("docker-host-integration")
	portMappings, _ := cmd.Flags().GetStringSlice("port")

	// Validate flag combinations
	if discoveryMode {
		if len(portMappings) > 0 {
			return fmt.Errorf("--discovery-mode cannot be used with port forwarding")
		}
		if dockerHostIntegration {
			return fmt.Errorf("--discovery-mode cannot be used with --docker-host-integration")
		}
	}

	// Parse and validate port mappings
	parsedPorts, err := parsePortMappings(portMappings)
	if err != nil {
		return fmt.Errorf("port mapping error: %w", err)
	}

	// Check for port conflicts
	if len(parsedPorts) > 0 {
		conflictPorts := checkPortConflicts(parsedPorts)
		if len(conflictPorts) > 0 {
			fmt.Printf("⚠️  WARNING: The following host ports may already be in use:\n")
			for _, port := range conflictPorts {
				fmt.Printf("   Port %d - containers may fail to start or port forwarding may not work\n", port)
			}
			fmt.Printf("   Consider using different host ports or stopping conflicting services.\n\n")
		}
	}

	// Security warning for Docker host integration
	if dockerHostIntegration {
		fmt.Printf("⚠️  WARNING: Docker host integration enabled!\n")
		fmt.Printf("   This gives the container full access to your host Docker daemon.\n")
		fmt.Printf("   Only use this flag with trusted images and AI agents.\n")
		fmt.Printf("   The container can create, modify, and delete other containers.\n\n")
	}

	// Load and validate configuration
	configService := config.NewService()
	resolved, err := configService.LoadConfiguration(provider, account, image, danger)
	if err != nil {
		return err
	}

	// Display resolved configuration for debugging
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	if verbose {
		fmt.Printf("Resolved configuration:\n")
		fmt.Printf("  Provider: %s\n", resolved.Provider.Name)
		fmt.Printf("  Account: %s\n", resolved.Account)
		fmt.Printf("  Image: %s\n", resolved.Image)
		fmt.Printf("  Danger: %t\n", resolved.Danger)
		fmt.Printf("  Project: %s\n", resolved.ProjectRoot)
		fmt.Printf("  Config Dir: %s\n", resolved.ProjectConfigDir)
	}

	// Initialize state service for directory validation
	stateService := core.NewStateService(resolved)

	// Validate that required directories exist
	if err := stateService.ValidateDirectories(); err != nil {
		return fmt.Errorf("state validation failed: %w\nHint: Run 'reactor config init' to create required directories", err)
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

	// Generate mount specifications and create container blueprint
	var mounts []core.MountSpec
	if !discoveryMode {
		mounts = stateService.GetMounts()
	}
	// Convert port mappings to core format
	corePortMappings := make([]core.PortMapping, len(parsedPorts))
	for i, pm := range parsedPorts {
		corePortMappings[i] = core.PortMapping{
			HostPort:      pm.HostPort,
			ContainerPort: pm.ContainerPort,
		}
	}

	blueprint := core.NewContainerBlueprint(resolved, mounts, discoveryMode, dockerHostIntegration, corePortMappings)
	containerSpec := blueprint.ToContainerSpec()

	// Enhanced verbose output showing container naming and discovery
	if verbose {
		fmt.Printf("[INFO] Project: %s (%s)\n", filepath.Base(resolved.ProjectRoot), resolved.ProjectRoot)
		fmt.Printf("[INFO] Container name: %s\n", containerSpec.Name)
		if discoveryMode {
			fmt.Printf("[INFO] Discovery mode: no mounts will be created\n")
		}
		if dockerHostIntegration {
			fmt.Printf("[INFO] Docker host integration: Docker socket will be mounted\n")
		}
		if len(parsedPorts) > 0 {
			fmt.Printf("[INFO] Port forwarding: ")
			for i, pm := range parsedPorts {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%d->%d", pm.HostPort, pm.ContainerPort)
			}
			fmt.Printf("\n")
		}
	}

	// Check for existing container first for enhanced verbose feedback
	if verbose {
		existingContainer, err := dockerService.ContainerExists(ctx, containerSpec.Name)
		if err == nil {
			switch existingContainer.Status {
			case docker.StatusRunning:
				fmt.Printf("[INFO] Found existing container: running\n")
			case docker.StatusStopped:
				fmt.Printf("[INFO] Found existing container: stopped (will be restarted)\n")
			case docker.StatusNotFound:
				fmt.Printf("[INFO] No existing container found (will create new one)\n")
			}
		}
	}

	// Provision container using recovery strategy (with cleanup for discovery mode)
	var containerInfo docker.ContainerInfo
	if discoveryMode {
		// In discovery mode, check if we need to clean up existing container
		existingContainer, checkErr := dockerService.ContainerExists(ctx, containerSpec.Name)
		if checkErr == nil && existingContainer.Status != docker.StatusNotFound {
			fmt.Printf("Discovery mode: removing existing container for clean environment\n")
		}
		containerInfo, err = dockerService.ProvisionContainerWithCleanup(ctx, containerSpec, true)
	} else {
		containerInfo, err = dockerService.ProvisionContainer(ctx, containerSpec)
	}
	if err != nil {
		return fmt.Errorf("failed to provision container: %w", err)
	}

	fmt.Printf("Container provisioned: %s\n", containerInfo.Name)
	if verbose {
		fmt.Printf("Container ID: %s\n", containerInfo.ID)
		fmt.Printf("Status: %s\n", containerInfo.Status)
	}

	// Attach to interactive session
	if verbose {
		fmt.Printf("[INFO] Attaching to container...\n")
	} else {
		fmt.Printf("Attaching to container session...\n")
	}

	if err := dockerService.AttachInteractiveSession(ctx, containerInfo.ID); err != nil {
		return fmt.Errorf("failed to attach to container session: %w", err)
	}

	// Inform user about container state after session ends
	fmt.Printf("\nSession ended. Container '%s' is still running.\n", containerInfo.Name)
	fmt.Printf("Use 'docker stop %s' to stop it.\n", containerInfo.Name)

	return nil
}

func diffCmdHandler(cmd *cobra.Command, args []string) error {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return err
	}

	// Load configuration to validate project setup
	configService := config.NewService()
	resolved, err := configService.LoadConfiguration("", "", "", false)
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

func accountsListHandler(cmd *cobra.Command, args []string) error {
	configService := config.NewService()
	return configService.ListAccounts()
}

func accountsShowHandler(cmd *cobra.Command, args []string) error {
	configService := config.NewService()
	value, err := configService.GetConfigValue("account")
	if err != nil {
		return err
	}

	fmt.Printf("Current account: %s\n", value)
	return nil
}

func accountsSetHandler(cmd *cobra.Command, args []string) error {
	account := args[0]
	configService := config.NewService()

	if err := configService.SetConfigValue("account", account); err != nil {
		return err
	}

	fmt.Printf("Account set to: %s\n", account)
	return nil
}

func configShowHandler(cmd *cobra.Command, args []string) error {
	configService := config.NewService()
	return configService.ShowConfiguration()
}

func configGetHandler(cmd *cobra.Command, args []string) error {
	key := args[0]
	configService := config.NewService()

	value, err := configService.GetConfigValue(key)
	if err != nil {
		return err
	}

	fmt.Printf("%v\n", value)
	return nil
}

func configSetHandler(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	configService := config.NewService()
	if err := configService.SetConfigValue(key, value); err != nil {
		return err
	}

	fmt.Printf("Set %s = %s\n", key, value)
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
		resolved, err := configService.LoadConfiguration("", "", "", false)
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
