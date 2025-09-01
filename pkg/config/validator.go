package config

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateProvider validates that the provider name is supported
func ValidateProvider(provider string) error {
	if provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}

	// Check if it's a built-in provider
	if _, exists := BuiltinProviders[provider]; exists {
		return nil
	}

	// Allow custom providers (any non-empty string that's not built-in)
	return nil
}

// ValidateAccount validates that the account name is valid for filesystem use
func ValidateAccount(account string) error {
	if account == "" {
		return fmt.Errorf("account cannot be empty")
	}

	// Check for valid filesystem directory name
	// Disallow paths and special characters that could cause issues
	if strings.Contains(account, "/") || strings.Contains(account, "\\") {
		return fmt.Errorf("account name cannot contain path separators")
	}

	if strings.Contains(account, "..") {
		return fmt.Errorf("account name cannot contain '..'")
	}

	// Disallow names that start with dots (hidden directories)
	if strings.HasPrefix(account, ".") {
		return fmt.Errorf("account name cannot start with '.'")
	}

	return nil
}

// ValidateImage validates that the image specification is valid
func ValidateImage(image string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}

	// Check if it's a built-in image alias first (these can be short like "go")
	if _, exists := BuiltinImages[image]; exists {
		return nil
	}

	// For non-builtin images, very short names are likely invalid
	if len(image) < 3 {
		return fmt.Errorf("image name too short (minimum 3 characters)")
	}

	// For custom images, do basic validation
	// Allow docker image names with optional registry/tag
	validImageName := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*[a-zA-Z0-9](?::[a-zA-Z0-9._-]+)?$`)
	if !validImageName.MatchString(image) {
		return fmt.Errorf("invalid image name format: %s", image)
	}

	return nil
}
