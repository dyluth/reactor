package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/reactor/pkg/config"
	"github.com/anthropics/reactor/pkg/core"
	"github.com/anthropics/reactor/pkg/docker"
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
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newSessionsCmd())
	cmd.AddCommand(newDiffCmd())
	cmd.AddCommand(newAccountsCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run AI agent in containerized environment",
		Long: `Run an AI agent in a containerized development environment with
persistent configuration and session management.

Examples:
  reactor run                              # Use project configuration
  reactor run --provider claude           # Override provider
  reactor run --image python --danger     # Use Python image with danger mode
  reactor run --account work-account      # Use specific account`,
		RunE: runCmdHandler,
	}

	// Add flags
	cmd.Flags().String("provider", "", "AI provider to use (claude, gemini, custom)")
	cmd.Flags().String("account", "", "Account for configuration isolation")
	cmd.Flags().String("image", "", "Container image (base, python, go, or custom URL)")
	cmd.Flags().Bool("danger", false, "Enable dangerous permissions for AI agent")

	return cmd
}

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show container filesystem changes",
		Long: `Show changes made to container filesystem during AI agent session.
Useful for discovery mode to understand what files an agent creates.`,
		RunE: diffCmdHandler,
	}

	cmd.Flags().Bool("discovery", false, "Run in discovery mode (no file mounts)")

	return cmd
}

func newAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Manage account configurations",
		Long:  `Manage isolated account configurations for different contexts (work, personal, etc.)`,
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
		Long:  `Manage project-specific configuration for providers, accounts, and settings.`,
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

List active and stopped containers, attach to running containers,
and manage your development sessions across different projects.`,
	}

	// Add subcommands
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all reactor containers",
		Long:  "List all reactor containers with their status and project information",
		RunE:  sessionsListHandler,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "attach [container-name]",
		Short: "Attach to a container session",
		Long: `Attach to a specific container session by name, or auto-attach to the current project's container.

Examples:
  reactor sessions attach                           # Auto-attach to current project
  reactor sessions attach reactor-cam-myproject-abc123  # Attach to specific container`,
		RunE: sessionsAttachHandler,
		Args: cobra.MaximumNArgs(1),
	})

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
	mounts := stateService.GetMounts()
	blueprint := core.NewContainerBlueprint(resolved, mounts)
	containerSpec := blueprint.ToContainerSpec()

	// Enhanced verbose output showing container naming and discovery
	if verbose {
		fmt.Printf("[INFO] Project: %s (%s)\n", filepath.Base(resolved.ProjectRoot), resolved.ProjectRoot)
		fmt.Printf("[INFO] Container name: %s\n", containerSpec.Name)
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

	// Provision container using recovery strategy
	containerInfo, err := dockerService.ProvisionContainer(ctx, containerSpec)
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
	_, err := configService.LoadConfiguration("", "", "", false)
	if err != nil {
		return err
	}

	return fmt.Errorf("container diff not implemented yet")
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