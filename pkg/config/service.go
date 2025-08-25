package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Service handles configuration operations
type Service struct {
	projectRoot string
	configPath  string
}

// NewService creates a new configuration service
func NewService() *Service {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "." // fallback
	}

	return &Service{
		projectRoot: cwd,
		configPath:  GetProjectConfigPath(),
	}
}

// LoadConfiguration loads and resolves the complete configuration
func (s *Service) LoadConfiguration(cliProvider, cliAccount, cliImage string, cliDanger bool) (*ResolvedConfig, error) {
	// 1. Load project configuration
	projectConfig, err := LoadProjectConfig(s.configPath)
	if err != nil {
		return nil, err
	}

	// 2. Apply CLI flag overrides (permanent updates to config file)
	updated := false
	if cliProvider != "" && cliProvider != projectConfig.Provider {
		projectConfig.Provider = cliProvider
		updated = true
	}
	if cliAccount != "" && cliAccount != projectConfig.Account {
		projectConfig.Account = cliAccount
		updated = true
	}
	if cliImage != "" && cliImage != projectConfig.Image {
		projectConfig.Image = cliImage
		updated = true
	}
	if cliDanger != projectConfig.Danger {
		projectConfig.Danger = cliDanger
		updated = true
	}

	// Save config if it was updated by CLI flags
	if updated {
		if err := SaveProjectConfig(projectConfig, s.configPath); err != nil {
			return nil, fmt.Errorf("failed to save updated configuration: %w", err)
		}
	}

	// 3. Resolve provider info
	providerInfo, exists := BuiltinProviders[projectConfig.Provider]
	if !exists {
		return nil, fmt.Errorf("unknown provider: %s. Available providers: claude, gemini", projectConfig.Provider)
	}

	// 4. Resolve final image
	finalImage := ResolveImage(projectConfig.Image, providerInfo.DefaultImage, cliImage)

	// 5. Generate project hash
	projectHash := GenerateProjectHash(s.projectRoot)

	// 6. Resolve directory paths
	reactorHome, err := GetReactorHomeDir()
	if err != nil {
		return nil, err
	}

	accountConfigDir := filepath.Join(reactorHome, projectConfig.Account)
	projectConfigDir := filepath.Join(accountConfigDir, projectHash)

	return &ResolvedConfig{
		Provider:         providerInfo,
		Account:          projectConfig.Account,
		Image:            finalImage,
		ProjectRoot:      s.projectRoot,
		ProjectHash:      projectHash,
		AccountConfigDir: accountConfigDir,
		ProjectConfigDir: projectConfigDir,
		Danger:           projectConfig.Danger,
	}, nil
}

// InitializeProject creates a new .reactor.conf with defaults and sets up directories
func (s *Service) InitializeProject() error {
	// Check if config already exists
	if _, err := os.Stat(s.configPath); err == nil {
		return fmt.Errorf("project already initialized. Configuration exists at %s", s.configPath)
	}

	// Create default configuration
	config, err := CreateDefaultProjectConfig()
	if err != nil {
		return err
	}

	// Save the configuration
	if err := SaveProjectConfig(config, s.configPath); err != nil {
		return err
	}

	// Create necessary directories
	if err := s.createProjectDirectories(config); err != nil {
		return err
	}

	// Print configuration info
	fmt.Printf("Initialized project configuration at: %s\n\n", s.configPath)
	fmt.Printf("Default configuration:\n")
	fmt.Printf("  provider: %s\n", config.Provider)
	fmt.Printf("  account:  %s\n", config.Account)
	fmt.Printf("  image:    %s\n", config.Image)
	fmt.Printf("  danger:   %t\n\n", config.Danger)
	
	fmt.Printf("To change these settings, run:\n")
	fmt.Printf("  reactor config set provider <claude|gemini>\n")
	fmt.Printf("  reactor config set account <account-name>\n")
	fmt.Printf("  reactor config set image <base|python|go>\n")
	fmt.Printf("  reactor config set danger <true|false>\n\n")

	return nil
}

// createProjectDirectories creates the necessary account and provider directories
func (s *Service) createProjectDirectories(config *ProjectConfig) error {
	projectHash := GenerateProjectHash(s.projectRoot)
	
	reactorHome, err := GetReactorHomeDir()
	if err != nil {
		return err
	}

	// Create ~/.reactor/<account>/<project-hash>/<provider>/ directories
	providerInfo := BuiltinProviders[config.Provider]
	for _, mount := range providerInfo.Mounts {
		dirPath := filepath.Join(reactorHome, config.Account, projectHash, mount.Source)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
		fmt.Printf("Created directory: %s\n", dirPath)
	}

	return nil
}

// GetConfigValue retrieves a configuration value by key
func (s *Service) GetConfigValue(key string) (interface{}, error) {
	config, err := LoadProjectConfig(s.configPath)
	if err != nil {
		return nil, err
	}

	switch key {
	case "provider":
		return config.Provider, nil
	case "account":
		return config.Account, nil
	case "image":
		return config.Image, nil
	case "danger":
		return config.Danger, nil
	default:
		return nil, fmt.Errorf("unknown configuration key: %s. Valid keys: provider, account, image, danger", key)
	}
}

// SetConfigValue sets a configuration value by key
func (s *Service) SetConfigValue(key, value string) error {
	config, err := LoadProjectConfig(s.configPath)
	if err != nil {
		return err
	}

	switch key {
	case "provider":
		if err := ValidateProvider(value); err != nil {
			return err
		}
		config.Provider = value
	case "account":
		if err := ValidateAccount(value); err != nil {
			return err
		}
		config.Account = value
	case "image":
		if err := ValidateImage(value); err != nil {
			return err
		}
		config.Image = value
	case "danger":
		switch value {
		case "true", "1", "yes", "on":
			config.Danger = true
		case "false", "0", "no", "off":
			config.Danger = false
		default:
			return fmt.Errorf("invalid boolean value for danger: %s. Use true/false", value)
		}
	default:
		return fmt.Errorf("unknown configuration key: %s. Valid keys: provider, account, image, danger", key)
	}

	// Save the updated configuration
	if err := SaveProjectConfig(config, s.configPath); err != nil {
		return err
	}

	// If account or provider changed, create directories as needed
	if key == "account" || key == "provider" {
		if err := s.createProjectDirectories(config); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create directories: %v\n", err)
		}
	}

	return nil
}

// ShowConfiguration displays the current configuration with directory paths
func (s *Service) ShowConfiguration() error {
	config, err := LoadProjectConfig(s.configPath)
	if err != nil {
		return err
	}

	resolved, err := s.LoadConfiguration("", "", "", false)
	if err != nil {
		return err
	}

	fmt.Printf("Project Configuration (%s):\n", s.configPath)
	fmt.Printf("  provider: %s\n", config.Provider)
	fmt.Printf("  account:  %s\n", config.Account)
	fmt.Printf("  image:    %s\n", config.Image)
	fmt.Printf("  danger:   %t\n\n", config.Danger)

	fmt.Printf("Resolved Configuration:\n")
	fmt.Printf("  final image:     %s\n", resolved.Image)
	fmt.Printf("  project root:    %s\n", resolved.ProjectRoot)
	fmt.Printf("  project hash:    %s\n", resolved.ProjectHash)
	fmt.Printf("  account dir:     %s\n", resolved.AccountConfigDir)
	fmt.Printf("  project config:  %s\n\n", resolved.ProjectConfigDir)

	fmt.Printf("Available Providers:\n")
	for name, info := range BuiltinProviders {
		fmt.Printf("  %s (default image: %s)\n", name, info.DefaultImage)
		for _, mount := range info.Mounts {
			fmt.Printf("    - mounts %s -> %s\n", mount.Source, mount.Target)
		}
	}

	fmt.Printf("\nAvailable Images:\n")
	for alias, image := range BuiltinImages {
		fmt.Printf("  %s -> %s\n", alias, image)
	}

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
				fmt.Printf("    project: %s\n", project.Name())
			}
		}
	}

	return nil
}