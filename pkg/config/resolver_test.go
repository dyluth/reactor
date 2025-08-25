package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetReactorHomeDir(t *testing.T) {
	// Save original environment
	originalPrefix := os.Getenv("REACTOR_ISOLATION_PREFIX")
	defer os.Setenv("REACTOR_ISOLATION_PREFIX", originalPrefix)

	// Test without isolation prefix
	os.Unsetenv("REACTOR_ISOLATION_PREFIX")
	homeDir, err := GetReactorHomeDir()
	if err != nil {
		t.Errorf("GetReactorHomeDir failed: %v", err)
	}

	expectedSuffix := ".reactor"
	if filepath.Base(homeDir) != expectedSuffix {
		t.Errorf("Expected directory to end with %s, got %s", expectedSuffix, filepath.Base(homeDir))
	}

	// Test with isolation prefix
	testPrefix := "test-12345"
	os.Setenv("REACTOR_ISOLATION_PREFIX", testPrefix)
	
	isolatedHomeDir, err := GetReactorHomeDir()
	if err != nil {
		t.Errorf("GetReactorHomeDir with isolation failed: %v", err)
	}

	expectedIsolatedSuffix := ".reactor-" + testPrefix
	if filepath.Base(isolatedHomeDir) != expectedIsolatedSuffix {
		t.Errorf("Expected isolated directory to end with %s, got %s", expectedIsolatedSuffix, filepath.Base(isolatedHomeDir))
	}

	// Verify they are different
	if homeDir == isolatedHomeDir {
		t.Error("Isolated and non-isolated home directories should be different")
	}
}

func TestGetProjectConfigPath(t *testing.T) {
	// Save original environment
	originalPrefix := os.Getenv("REACTOR_ISOLATION_PREFIX")
	defer os.Setenv("REACTOR_ISOLATION_PREFIX", originalPrefix)

	// Test without isolation prefix
	os.Unsetenv("REACTOR_ISOLATION_PREFIX")
	configPath := GetProjectConfigPath()
	expectedPath := ".reactor.conf"
	if configPath != expectedPath {
		t.Errorf("Expected config path %s, got %s", expectedPath, configPath)
	}

	// Test with isolation prefix
	testPrefix := "test-67890"
	os.Setenv("REACTOR_ISOLATION_PREFIX", testPrefix)
	
	isolatedConfigPath := GetProjectConfigPath()
	expectedIsolatedPath := "." + testPrefix + ".conf"
	if isolatedConfigPath != expectedIsolatedPath {
		t.Errorf("Expected isolated config path %s, got %s", expectedIsolatedPath, isolatedConfigPath)
	}

	// Verify they are different
	if configPath == isolatedConfigPath {
		t.Error("Isolated and non-isolated config paths should be different")
	}
}

func TestIsolationPrefixEmpty(t *testing.T) {
	// Save original environment
	originalPrefix := os.Getenv("REACTOR_ISOLATION_PREFIX")
	defer os.Setenv("REACTOR_ISOLATION_PREFIX", originalPrefix)

	// Test with empty isolation prefix (should behave like no prefix)
	os.Setenv("REACTOR_ISOLATION_PREFIX", "")
	
	homeDir, err := GetReactorHomeDir()
	if err != nil {
		t.Errorf("GetReactorHomeDir with empty prefix failed: %v", err)
	}
	
	configPath := GetProjectConfigPath()
	
	// Should be same as default behavior
	expectedHomeSuffix := ".reactor"
	expectedConfigPath := ".reactor.conf"
	
	if filepath.Base(homeDir) != expectedHomeSuffix {
		t.Errorf("Expected default home directory suffix %s, got %s", expectedHomeSuffix, filepath.Base(homeDir))
	}
	
	if configPath != expectedConfigPath {
		t.Errorf("Expected default config path %s, got %s", expectedConfigPath, configPath)
	}
}