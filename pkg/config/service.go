package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Service handles configuration operations
type Service struct {
	projectRoot string
}

// NewService creates a new configuration service
func NewService() *Service {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "." // fallback
	}

	return &Service{
		projectRoot: cwd,
	}
}

// NewServiceWithRoot creates a new configuration service with a specific project root
func NewServiceWithRoot(projectRoot string) *Service {
	return &Service{
		projectRoot: projectRoot,
	}
}

// ResolveConfiguration loads and resolves configuration using the new devcontainer.json workflow
func (s *Service) ResolveConfiguration() (*ResolvedConfig, error) {
	// 1. Find devcontainer.json
	configPath, found, err := FindDevContainerFile(s.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("error searching for devcontainer.json: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("no devcontainer.json found in %s or %s. Run 'reactor init' to create one",
			filepath.Join(s.projectRoot, ".devcontainer", "devcontainer.json"),
			filepath.Join(s.projectRoot, ".devcontainer.json"))
	}

	// 2. Parse devcontainer.json
	devConfig, err := LoadDevContainerConfig(configPath)
	if err != nil {
		return nil, err
	}

	// 3. Map DevContainerConfig to ResolvedConfig
	return s.mapToResolvedConfig(devConfig)
}

// mapToResolvedConfig transforms DevContainerConfig into ResolvedConfig
func (s *Service) mapToResolvedConfig(devConfig *DevContainerConfig) (*ResolvedConfig, error) {
	// Extract account from customizations or use system default
	account := ""
	defaultCommand := ""
	if devConfig.Customizations != nil && devConfig.Customizations.Reactor != nil {
		account = devConfig.Customizations.Reactor.Account
		defaultCommand = devConfig.Customizations.Reactor.DefaultCommand
	}
	if account == "" {
		systemUser, err := GetSystemUsername()
		if err != nil {
			return nil, fmt.Errorf("failed to get system username for default account: %w", err)
		}
		account = systemUser
	}

	// For now, use claude as default provider until we implement provider-agnostic design
	providerInfo := BuiltinProviders["claude"]

	// Use image from devcontainer.json or default
	image := devConfig.Image
	if image == "" {
		image = providerInfo.DefaultImage
	}

	// Parse and validate forwardPorts from devcontainer.json
	forwardPorts, err := parseForwardPorts(devConfig.ForwardPorts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse forwardPorts from devcontainer.json: %w", err)
	}

	// Extract remoteUser from devcontainer.json (will be defaulted in core layer if empty)
	remoteUser := devConfig.RemoteUser

	// Generate project hash and paths
	projectHash := GenerateProjectHash(s.projectRoot)
	reactorHome, err := GetReactorHomeDir()
	if err != nil {
		return nil, err
	}

	accountConfigDir := filepath.Join(reactorHome, account)
	projectConfigDir := filepath.Join(accountConfigDir, projectHash)

	return &ResolvedConfig{
		Provider:          providerInfo,
		Account:           account,
		Image:             image,
		ProjectRoot:       s.projectRoot,
		ProjectHash:       projectHash,
		AccountConfigDir:  accountConfigDir,
		ProjectConfigDir:  projectConfigDir,
		ForwardPorts:      forwardPorts,
		RemoteUser:        remoteUser,
		Build:             devConfig.Build,
		PostCreateCommand: devConfig.PostCreateCommand,
		DefaultCommand:    defaultCommand,
		Danger:            false, // Default to safe mode for now
	}, nil
}

// InitializeProject creates a basic devcontainer.json template
func (s *Service) InitializeProject() error {
	// Check if devcontainer.json already exists
	configPath, found, err := FindDevContainerFile(s.projectRoot)
	if err != nil {
		return fmt.Errorf("error checking for existing devcontainer.json: %w", err)
	}
	if found {
		return fmt.Errorf("project already initialized. Configuration exists at %s", configPath)
	}

	// Create .devcontainer directory
	devcontainerDir := filepath.Join(s.projectRoot, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devcontainer directory: %w", err)
	}

	// Get system username for default account
	username, err := GetSystemUsername()
	if err != nil {
		return fmt.Errorf("failed to get system username: %w", err)
	}

	// Create basic devcontainer.json template
	configPath = filepath.Join(devcontainerDir, "devcontainer.json")
	template := fmt.Sprintf(`{
	"name": "%s",
	"image": "ghcr.io/dyluth/reactor/base:latest",
	
	"customizations": {
		"reactor": {
			"account": "%s"
		}
	}
}`, filepath.Base(s.projectRoot), username)

	if err := os.WriteFile(configPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write devcontainer.json: %w", err)
	}

	fmt.Printf("Initialized devcontainer.json at: %s\n\n", configPath)
	fmt.Printf("Default configuration:\n")
	fmt.Printf("  name: %s\n", filepath.Base(s.projectRoot))
	fmt.Printf("  image: ghcr.io/dyluth/reactor/base:latest\n")
	fmt.Printf("  account: %s\n\n", username)
	fmt.Printf("Edit %s to customize your development environment.\n", configPath)

	return nil
}

// ShowConfiguration displays the current devcontainer configuration
func (s *Service) ShowConfiguration() error {
	// Try to resolve current configuration
	resolved, err := s.ResolveConfiguration()
	if err != nil {
		return err
	}

	// Find the devcontainer.json file to show its path
	configPath, found, err := FindDevContainerFile(s.projectRoot)
	if err != nil {
		return fmt.Errorf("error finding devcontainer.json: %w", err)
	}
	if !found {
		return fmt.Errorf("no devcontainer.json found")
	}

	fmt.Printf("DevContainer Configuration (%s):\n", configPath)
	fmt.Printf("  account:         %s\n", resolved.Account)
	fmt.Printf("  image:           %s\n", resolved.Image)
	fmt.Printf("  project root:    %s\n", resolved.ProjectRoot)
	fmt.Printf("  project hash:    %s\n", resolved.ProjectHash)
	fmt.Printf("  account dir:     %s\n", resolved.AccountConfigDir)
	fmt.Printf("  project config:  %s\n\n", resolved.ProjectConfigDir)

	fmt.Printf("Edit %s to customize your development environment.\n", configPath)
	fmt.Printf("See https://containers.dev/implementors/json_reference/ for full specification.\n")

	return nil
}

// ListAccounts scans ~/.reactor/ for existing accounts
func (s *Service) ListAccounts() error {
	reactorHome, err := GetReactorHomeDir()
	if err != nil {
		return err
	}

	// Check if reactor home exists
	if _, err := os.Stat(reactorHome); os.IsNotExist(err) {
		fmt.Printf("No accounts found. Reactor home directory does not exist: %s\n", reactorHome)
		return nil
	}

	// Read directory contents
	entries, err := os.ReadDir(reactorHome)
	if err != nil {
		return fmt.Errorf("failed to read reactor home directory: %w", err)
	}

	accounts := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			accounts = append(accounts, entry.Name())
		}
	}

	if len(accounts) == 0 {
		fmt.Printf("No accounts found in %s\n", reactorHome)
		return nil
	}

	fmt.Printf("Configured accounts:\n")
	for _, account := range accounts {
		fmt.Printf("  %s\n", account)

		// Show projects for this account
		accountDir := filepath.Join(reactorHome, account)
		projectEntries, err := os.ReadDir(accountDir)
		if err != nil {
			continue
		}

		for _, project := range projectEntries {
			if project.IsDir() {
				// Try to read project-path.txt to get the human-readable path
				projectPathFile := filepath.Join(accountDir, project.Name(), "project-path.txt")
				if projectPathData, err := os.ReadFile(projectPathFile); err == nil {
					projectPath := strings.TrimSpace(string(projectPathData))
					fmt.Printf("    - %s (%s)\n", projectPath, project.Name())
				} else {
					// Fallback to hash-only display if project-path.txt doesn't exist
					fmt.Printf("    project: %s\n", project.Name())
				}
			}
		}
	}

	return nil
}

// CleanAccounts scans ~/.reactor/ for orphaned account configurations
// and prompts the user to remove project directories for non-existent paths
func (s *Service) CleanAccounts() error {
	reactorHome, err := GetReactorHomeDir()
	if err != nil {
		return err
	}

	// Check if reactor home exists
	if _, err := os.Stat(reactorHome); os.IsNotExist(err) {
		fmt.Printf("No accounts found. Reactor home directory does not exist: %s\n", reactorHome)
		return nil
	}

	// Read directory contents
	entries, err := os.ReadDir(reactorHome)
	if err != nil {
		return fmt.Errorf("failed to read reactor home directory: %w", err)
	}

	var orphanedDirs []string

	// Scan all accounts and their project directories
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		accountDir := filepath.Join(reactorHome, entry.Name())
		projectEntries, err := os.ReadDir(accountDir)
		if err != nil {
			continue // Skip accounts we can't read
		}

		for _, project := range projectEntries {
			if !project.IsDir() {
				continue
			}

			projectConfigDir := filepath.Join(accountDir, project.Name())
			projectPathFile := filepath.Join(projectConfigDir, "project-path.txt")

			// Try to read the project path
			if projectPathData, err := os.ReadFile(projectPathFile); err == nil {
				projectPath := strings.TrimSpace(string(projectPathData))

				// Check if the project path still exists
				if _, err := os.Stat(projectPath); os.IsNotExist(err) {
					orphanedDirs = append(orphanedDirs, projectConfigDir)
				}
			}
			// If we can't read project-path.txt, we can't determine if it's orphaned
		}
	}

	if len(orphanedDirs) == 0 {
		fmt.Printf("No orphaned account configurations found.\n")
		return nil
	}

	// Display orphaned directories
	fmt.Printf("Found %d orphaned account configuration(s):\n", len(orphanedDirs))
	for _, dir := range orphanedDirs {
		// Extract account and project hash from path
		relPath, _ := filepath.Rel(reactorHome, dir)
		fmt.Printf("  %s\n", relPath)
	}

	// Prompt for confirmation
	fmt.Printf("\nAre you sure you want to delete these %d configuration directories? [y/N]: ", len(orphanedDirs))
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		// On any error (like EOF), treat it as a "no"
		response = "n"
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		fmt.Printf("Operation cancelled.\n")
		return nil
	}

	// Remove orphaned directories
	removedCount := 0
	for _, dir := range orphanedDirs {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", dir, err)
		} else {
			removedCount++
		}
	}

	fmt.Printf("Successfully removed %d orphaned configuration directories.\n", removedCount)
	return nil
}

// parseForwardPorts parses the forwardPorts array from devcontainer.json
// Handles both int (8080 -> 8080:8080) and string ("8080:3000") formats
func parseForwardPorts(forwardPorts []interface{}) ([]PortMapping, error) {
	var result []PortMapping

	for i, port := range forwardPorts {
		var hostPort, containerPort int
		var err error

		switch v := port.(type) {
		case float64:
			// JSON numbers are unmarshalled as float64 in Go
			hostPort = int(v)
			containerPort = int(v)

			// Validate port range
			if hostPort < 1 || hostPort > 65535 {
				return nil, fmt.Errorf("forwardPorts[%d]: port %d is out of valid range (1-65535)", i, hostPort)
			}

		case string:
			// Parse "host:container" format
			parts := strings.Split(v, ":")
			if len(parts) != 2 {
				return nil, fmt.Errorf("forwardPorts[%d]: invalid string format '%s', expected 'host:container'", i, v)
			}

			hostPort, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("forwardPorts[%d]: invalid host port '%s', must be a number", i, parts[0])
			}

			containerPort, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("forwardPorts[%d]: invalid container port '%s', must be a number", i, parts[1])
			}

			// Validate port ranges
			if hostPort < 1 || hostPort > 65535 {
				return nil, fmt.Errorf("forwardPorts[%d]: host port %d is out of valid range (1-65535)", i, hostPort)
			}
			if containerPort < 1 || containerPort > 65535 {
				return nil, fmt.Errorf("forwardPorts[%d]: container port %d is out of valid range (1-65535)", i, containerPort)
			}

		default:
			return nil, fmt.Errorf("forwardPorts[%d]: invalid type %T, expected number or string", i, v)
		}

		result = append(result, PortMapping{
			HostPort:      hostPort,
			ContainerPort: containerPort,
		})
	}

	return result, nil
}
