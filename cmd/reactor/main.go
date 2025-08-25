package main

import (
	"fmt"
	"os"

	"github.com/anthropics/reactor/pkg/config"
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

	return fmt.Errorf("Container provisioning not implemented yet")
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

	return fmt.Errorf("Container diff not implemented yet")
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