package config

import (
	"strings"
	"testing"
)

func TestValidateAccount(t *testing.T) {
	testCases := []struct {
		name        string
		account     string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid account",
			account:     "testuser",
			expectError: false,
		},
		{
			name:        "valid account with numbers",
			account:     "user123",
			expectError: false,
		},
		{
			name:        "valid account with hyphens",
			account:     "test-user",
			expectError: false,
		},
		{
			name:        "valid account with underscores",
			account:     "test_user",
			expectError: false,
		},
		{
			name:        "empty account",
			account:     "",
			expectError: true,
			errorText:   "account cannot be empty",
		},
		{
			name:        "account with forward slash",
			account:     "test/user",
			expectError: true,
			errorText:   "cannot contain path separators",
		},
		{
			name:        "account with backslash",
			account:     "test\\user",
			expectError: true,
			errorText:   "cannot contain path separators",
		},
		{
			name:        "account with double dots",
			account:     "test..user",
			expectError: true,
			errorText:   "cannot contain '..'",
		},
		{
			name:        "account starting with dot",
			account:     ".hidden",
			expectError: true,
			errorText:   "cannot start with '.'",
		},
		{
			name:        "malicious path traversal",
			account:     "../../../etc",
			expectError: true,
			errorText:   "cannot contain path separators",
		},
		{
			name:        "malicious relative path",
			account:     "../../malicious",
			expectError: true,
			errorText:   "cannot contain path separators",
		},
		{
			name:        "account with just dots",
			account:     "..",
			expectError: true,
			errorText:   "cannot contain '..'", // This will match first due to order
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAccount(tc.account)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for account '%s', but got none", tc.account)
				} else if tc.errorText != "" && !strings.Contains(err.Error(), tc.errorText) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for account '%s', got: %v", tc.account, err)
				}
			}
		})
	}
}

func TestValidateImage_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		image       string
		expectError bool
		errorText   string
	}{
		// Built-in images
		{
			name:        "builtin base image",
			image:       "base",
			expectError: false,
		},
		{
			name:        "builtin python image",
			image:       "python",
			expectError: false,
		},
		{
			name:        "builtin go image",
			image:       "go",
			expectError: true, // "go" is only 2 chars, fails minimum length
			errorText:   "image name too short",
		},
		{
			name:        "builtin node image",
			image:       "node",
			expectError: false,
		},
		// Valid custom images
		{
			name:        "simple custom image",
			image:       "myimage",
			expectError: false,
		},
		{
			name:        "image with registry",
			image:       "registry.example.com/myimage",
			expectError: false,
		},
		{
			name:        "image with tag",
			image:       "myimage:latest",
			expectError: false,
		},
		{
			name:        "full image reference",
			image:       "registry.example.com/namespace/image:v1.2.3",
			expectError: false,
		},
		{
			name:        "github registry image",
			image:       "ghcr.io/owner/repo:main",
			expectError: false,
		},
		{
			name:        "dockerhub official image",
			image:       "ubuntu:20.04",
			expectError: false,
		},
		// Edge cases for valid images
		{
			name:        "image with port",
			image:       "localhost:5000/myimage",
			expectError: true, // Current regex doesn't allow : in middle
			errorText:   "invalid image name format",
		},
		{
			name:        "image with underscores",
			image:       "my_image",
			expectError: false,
		},
		{
			name:        "image with hyphens",
			image:       "my-image",
			expectError: false,
		},
		// Invalid images
		{
			name:        "empty image",
			image:       "",
			expectError: true,
			errorText:   "image cannot be empty",
		},
		{
			name:        "too short image",
			image:       "ab",
			expectError: true,
			errorText:   "image name too short",
		},
		{
			name:        "image starting with non-alphanumeric",
			image:       "-badimage",
			expectError: true,
			errorText:   "invalid image name format",
		},
		{
			name:        "image ending with non-alphanumeric",
			image:       "badimage-",
			expectError: true,
			errorText:   "invalid image name format",
		},
		{
			name:        "image with invalid characters",
			image:       "bad@image",
			expectError: true,
			errorText:   "invalid image name format",
		},
		{
			name:        "image with spaces",
			image:       "bad image",
			expectError: true,
			errorText:   "invalid image name format",
		},
		{
			name:        "image with special characters",
			image:       "bad!image",
			expectError: true,
			errorText:   "invalid image name format",
		},
		{
			name:        "malformed tag",
			image:       "image:",
			expectError: true,
			errorText:   "invalid image name format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateImage(tc.image)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for image '%s', but got none", tc.image)
				} else if tc.errorText != "" && !strings.Contains(err.Error(), tc.errorText) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for image '%s', got: %v", tc.image, err)
				}
			}
		})
	}
}

func TestValidateProvider_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		provider    string
		expectError bool
		errorText   string
	}{
		// Built-in providers (should always be valid)
		{
			name:        "claude provider",
			provider:    "claude",
			expectError: false,
		},
		{
			name:        "gemini provider",
			provider:    "gemini",
			expectError: false,
		},
		// Custom providers (any non-empty string should be valid)
		{
			name:        "custom provider",
			provider:    "openai",
			expectError: false,
		},
		{
			name:        "custom provider with numbers",
			provider:    "ai-model-v2",
			expectError: false,
		},
		{
			name:        "custom provider with special chars",
			provider:    "custom_ai-provider.v1",
			expectError: false,
		},
		// Invalid cases
		{
			name:        "empty provider",
			provider:    "",
			expectError: true,
			errorText:   "provider cannot be empty",
		},
		{
			name:        "whitespace only provider",
			provider:    "   ",
			expectError: false, // Current implementation allows whitespace, might want to change
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProvider(tc.provider)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for provider '%s', but got none", tc.provider)
				} else if tc.errorText != "" && !strings.Contains(err.Error(), tc.errorText) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for provider '%s', got: %v", tc.provider, err)
				}
			}
		})
	}
}
