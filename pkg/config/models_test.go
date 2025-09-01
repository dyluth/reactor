package config

import (
	"testing"
)

func TestValidateProvider(t *testing.T) {
	// Test valid built-in providers
	if err := ValidateProvider("claude"); err != nil {
		t.Errorf("Expected claude to be valid, got error: %v", err)
	}

	if err := ValidateProvider("gemini"); err != nil {
		t.Errorf("Expected gemini to be valid, got error: %v", err)
	}

	// Test custom provider
	if err := ValidateProvider("custom-ai"); err != nil {
		t.Errorf("Expected custom provider to be valid, got error: %v", err)
	}

	// Test empty provider
	if err := ValidateProvider(""); err == nil {
		t.Error("Expected empty provider to be invalid")
	}
}

func TestValidateImage(t *testing.T) {
	// Test valid built-in images
	if err := ValidateImage("base"); err != nil {
		t.Errorf("Expected base image to be valid, got error: %v", err)
	}

	if err := ValidateImage("python"); err != nil {
		t.Errorf("Expected python image to be valid, got error: %v", err)
	}

	// Test custom image
	if err := ValidateImage("my-custom-image:latest"); err != nil {
		t.Errorf("Expected custom image to be valid, got error: %v", err)
	}

	// Test empty image
	if err := ValidateImage(""); err == nil {
		t.Error("Expected empty image to be invalid")
	}

	// Test too short image
	if err := ValidateImage("ab"); err == nil {
		t.Error("Expected too short image to be invalid")
	}
}

func TestGenerateProjectHash(t *testing.T) {
	hash1 := GenerateProjectHash("/path/to/project")
	hash2 := GenerateProjectHash("/path/to/project")
	hash3 := GenerateProjectHash("/different/path")

	// Same path should generate same hash
	if hash1 != hash2 {
		t.Error("Same path should generate same hash")
	}

	// Different paths should generate different hashes
	if hash1 == hash3 {
		t.Error("Different paths should generate different hashes")
	}

	// Hash should be 8 characters
	if len(hash1) != 8 {
		t.Errorf("Expected hash length 8, got %d", len(hash1))
	}

	// Empty path should still generate hash
	emptyHash := GenerateProjectHash("")
	if len(emptyHash) != 8 {
		t.Errorf("Expected empty path hash length 8, got %d", len(emptyHash))
	}

	// Very long path should still generate valid hash
	longPath := "/very/long/path/with/many/segments/that/exceeds/normal/length/project/directory/name"
	longHash := GenerateProjectHash(longPath)
	if len(longHash) != 8 {
		t.Errorf("Expected long path hash length 8, got %d", len(longHash))
	}

	// Hash should only contain valid characters (alphanumeric)
	for _, char := range hash1 {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			t.Errorf("Hash contains invalid character: %c", char)
		}
	}
}

func TestResolveImage(t *testing.T) {
	// CLI override should take precedence
	result := ResolveImage("python", "base", "go")
	expected := BuiltinImages["go"]
	if result != expected {
		t.Errorf("Expected CLI override to take precedence: got %s, want %s", result, expected)
	}

	// Config should be used if no CLI override
	result = ResolveImage("python", "base", "")
	expected = BuiltinImages["python"]
	if result != expected {
		t.Errorf("Expected config image to be used: got %s, want %s", result, expected)
	}

	// Provider default should be fallback
	result = ResolveImage("", "base", "")
	expected = BuiltinImages["base"]
	if result != expected {
		t.Errorf("Expected provider default to be used: got %s, want %s", result, expected)
	}

	// Custom CLI image should pass through unchanged
	customImage := "custom/myimage:latest"
	result = ResolveImage("", "base", customImage)
	if result != customImage {
		t.Errorf("Expected custom CLI image to pass through: got %s, want %s", result, customImage)
	}

	// Custom config image should pass through unchanged
	customConfigImage := "registry.io/project:v1.0"
	result = ResolveImage(customConfigImage, "base", "")
	if result != customConfigImage {
		t.Errorf("Expected custom config image to pass through: got %s, want %s", result, customConfigImage)
	}

	// Custom provider default should pass through unchanged
	customDefault := "myregistry/default:latest"
	result = ResolveImage("", customDefault, "")
	if result != customDefault {
		t.Errorf("Expected custom provider default to pass through: got %s, want %s", result, customDefault)
	}
}

func TestBuiltinProviders(t *testing.T) {
	// Test that built-in providers are properly configured
	claude, exists := BuiltinProviders["claude"]
	if !exists {
		t.Error("Claude provider should exist in built-in providers")
	}

	if claude.Name != "claude" {
		t.Errorf("Expected claude name to be 'claude', got '%s'", claude.Name)
	}

	if len(claude.Mounts) == 0 {
		t.Error("Claude provider should have at least one mount point")
	}

	gemini, exists := BuiltinProviders["gemini"]
	if !exists {
		t.Error("Gemini provider should exist in built-in providers")
	}

	if gemini.Name != "gemini" {
		t.Errorf("Expected gemini name to be 'gemini', got '%s'", gemini.Name)
	}
}
