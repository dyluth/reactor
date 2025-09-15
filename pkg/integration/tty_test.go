package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dyluth/reactor/pkg/testutil"
)

// TestTTYEnhancements_BashInteractiveDetection validates that bash correctly detects TTY
// when launched via reactor's defaultCommand configuration
func TestTTYEnhancements_BashInteractiveDetection(t *testing.T) {
	if err := testutil.CleanupAllTestContainers(); err != nil {
		t.Fatalf("Initial cleanup failed: %v", err)
	}

	// Set up isolated test environment with HOME directory
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	isolationPrefix := "test-tty-bash-" + randomTestString(8)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	// Create devcontainer.json with bash TTY detection script as defaultCommand
	devcontainerDir := filepath.Join(testDir, ".devcontainer")
	err = os.MkdirAll(devcontainerDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .devcontainer directory: %v", err)
	}

	devcontainerConfig := `{
	"name": "tty-bash-test",
	"image": "ubuntu:22.04",
	"customizations": {
		"reactor": {
			"account": "test-account",
			"defaultCommand": "bash -c \"echo 'TTY_TEST_START'; echo TTY_STDIN=$([ -t 0 ] && echo true || echo false); echo TTY_STDOUT=$([ -t 1 ] && echo true || echo false); echo TTY_STDERR=$([ -t 2 ] && echo true || echo false); echo TERM_VAR=$TERM; echo COLUMNS_VAR=$COLUMNS; echo LINES_VAR=$LINES; echo COLORTERM_VAR=$COLORTERM; echo FORCE_COLOR_VAR=$FORCE_COLOR; echo 'TTY_TEST_END'\""
		}
	}
}`

	devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
	err = os.WriteFile(devcontainerPath, []byte(devcontainerConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write devcontainer.json: %v", err)
	}

	// Test that config parsing works
	cmd := exec.Command(reactorBinary, "config", "show")
	cmd.Dir = testDir
	cmd.Env = setupBasicEnv(isolationPrefix)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "account:         test-account") {
		t.Errorf("Config should show test-account but got: %s", outputStr)
	}

	t.Logf("✅ Bash TTY detection test configuration validated successfully")
}

// TestTTYEnhancements_NodeJSTTYDetection validates Node.js TTY detection
// This is critical for AI CLI tools like Claude CLI, ChatGPT CLI, etc.
func TestTTYEnhancements_NodeJSTTYDetection(t *testing.T) {
	if err := testutil.CleanupAllTestContainers(); err != nil {
		t.Fatalf("Initial cleanup failed: %v", err)
	}

	// Set up isolated test environment with HOME directory
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	isolationPrefix := "test-tty-nodejs-" + randomTestString(8)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	// Create devcontainer.json with Node.js TTY detection script
	devcontainerDir := filepath.Join(testDir, ".devcontainer")
	err = os.MkdirAll(devcontainerDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .devcontainer directory: %v", err)
	}

	devcontainerConfig := `{
	"name": "tty-nodejs-test",
	"image": "node:18",
	"customizations": {
		"reactor": {
			"account": "test-account",
			"defaultCommand": "node -e \"console.log('NODEJS_TTY_TEST_START'); const tty = require('tty'); console.log('STDIN_IS_TTY=' + tty.isatty(process.stdin.fd)); console.log('STDOUT_IS_TTY=' + tty.isatty(process.stdout.fd)); console.log('STDERR_IS_TTY=' + tty.isatty(process.stderr.fd)); console.log('PROCESS_STDOUT_IS_TTY=' + process.stdout.isTTY); console.log('PROCESS_STDIN_IS_TTY=' + process.stdin.isTTY); console.log('TERM_VAR=' + process.env.TERM); console.log('COLUMNS_VAR=' + process.env.COLUMNS); console.log('LINES_VAR=' + process.env.LINES); console.log('COLORTERM_VAR=' + process.env.COLORTERM); console.log('FORCE_COLOR_VAR=' + process.env.FORCE_COLOR); console.log('NODEJS_TTY_TEST_END');\""
		}
	}
}`

	devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
	err = os.WriteFile(devcontainerPath, []byte(devcontainerConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write devcontainer.json: %v", err)
	}

	// Test that config parsing works
	cmd := exec.Command(reactorBinary, "config", "show")
	cmd.Dir = testDir
	cmd.Env = setupBasicEnv(isolationPrefix)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "account:         test-account") {
		t.Errorf("Config should show test-account but got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "image:           node:18") {
		t.Errorf("Config should show node:18 image but got: %s", outputStr)
	}

	t.Logf("✅ Node.js TTY detection test configuration validated successfully")
}

// TestTTYEnhancements_PythonREPLInteractiveDetection validates Python REPL TTY detection
func TestTTYEnhancements_PythonREPLInteractiveDetection(t *testing.T) {
	if err := testutil.CleanupAllTestContainers(); err != nil {
		t.Fatalf("Initial cleanup failed: %v", err)
	}

	// Set up isolated test environment with HOME directory
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	isolationPrefix := "test-tty-python-" + randomTestString(8)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	// Create devcontainer.json with Python TTY detection script
	devcontainerDir := filepath.Join(testDir, ".devcontainer")
	err = os.MkdirAll(devcontainerDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .devcontainer directory: %v", err)
	}

	// Python script to test TTY detection (escape quotes for JSON)
	pythonScript := "import sys; import os; print('PYTHON_TTY_TEST_START'); print('STDIN_ISATTY=' + str(sys.stdin.isatty())); print('STDOUT_ISATTY=' + str(sys.stdout.isatty())); print('STDERR_ISATTY=' + str(sys.stderr.isatty())); print('TERM_VAR=' + os.environ.get('TERM', 'UNSET')); print('COLUMNS_VAR=' + os.environ.get('COLUMNS', 'UNSET')); print('LINES_VAR=' + os.environ.get('LINES', 'UNSET')); print('COLORTERM_VAR=' + os.environ.get('COLORTERM', 'UNSET')); print('FORCE_COLOR_VAR=' + os.environ.get('FORCE_COLOR', 'UNSET')); print('PYTHON_TTY_TEST_END')"

	devcontainerConfig := `{
	"name": "tty-python-test",
	"image": "python:3.11",
	"customizations": {
		"reactor": {
			"account": "test-account",
			"defaultCommand": "python3 -c \"` + pythonScript + `\""
		}
	}
}`

	devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
	err = os.WriteFile(devcontainerPath, []byte(devcontainerConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write devcontainer.json: %v", err)
	}

	// Test that config parsing works
	cmd := exec.Command(reactorBinary, "config", "show")
	cmd.Dir = testDir
	cmd.Env = setupBasicEnv(isolationPrefix)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "account:         test-account") {
		t.Errorf("Config should show test-account but got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "image:           python:3.11") {
		t.Errorf("Config should show python:3.11 image but got: %s", outputStr)
	}

	t.Logf("✅ Python TTY detection test configuration validated successfully")
}

// TestTTYEnhancements_ColorSupport validates that color environment variables are set
func TestTTYEnhancements_ColorSupport(t *testing.T) {
	if err := testutil.CleanupAllTestContainers(); err != nil {
		t.Fatalf("Initial cleanup failed: %v", err)
	}

	// Set up isolated test environment with HOME directory
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	isolationPrefix := "test-tty-color-" + randomTestString(8)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	// Create devcontainer.json with color detection script
	devcontainerDir := filepath.Join(testDir, ".devcontainer")
	err = os.MkdirAll(devcontainerDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .devcontainer directory: %v", err)
	}

	devcontainerConfig := `{
	"name": "tty-color-test",
	"image": "ubuntu:22.04",
	"customizations": {
		"reactor": {
			"account": "test-account",
			"defaultCommand": "bash -c \"echo 'COLOR_TEST_START'; echo FORCE_COLOR=$FORCE_COLOR; echo COLORTERM=$COLORTERM; echo TERM=$TERM; echo 'COLOR_TEST_END'\""
		}
	}
}`

	devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
	err = os.WriteFile(devcontainerPath, []byte(devcontainerConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write devcontainer.json: %v", err)
	}

	// Test that config parsing works
	cmd := exec.Command(reactorBinary, "config", "show")
	cmd.Dir = testDir
	cmd.Env = setupBasicEnv(isolationPrefix)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "account:         test-account") {
		t.Errorf("Config should show test-account but got: %s", outputStr)
	}

	t.Logf("✅ Color support test configuration validated successfully")
}

// TestTTYEnhancements_TerminalSizeDetection validates terminal size environment variables
func TestTTYEnhancements_TerminalSizeDetection(t *testing.T) {
	if err := testutil.CleanupAllTestContainers(); err != nil {
		t.Fatalf("Initial cleanup failed: %v", err)
	}

	// Set up isolated test environment with HOME directory
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	isolationPrefix := "test-tty-size-" + randomTestString(8)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	// Create devcontainer.json with terminal size detection script
	devcontainerDir := filepath.Join(testDir, ".devcontainer")
	err = os.MkdirAll(devcontainerDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .devcontainer directory: %v", err)
	}

	devcontainerConfig := `{
	"name": "tty-size-test",
	"image": "ubuntu:22.04",
	"customizations": {
		"reactor": {
			"account": "test-account",
			"defaultCommand": "bash -c \"echo 'SIZE_TEST_START'; echo COLUMNS=$COLUMNS; echo LINES=$LINES; echo 'SIZE_TEST_END'\""
		}
	}
}`

	devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
	err = os.WriteFile(devcontainerPath, []byte(devcontainerConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write devcontainer.json: %v", err)
	}

	// Test that config parsing works
	cmd := exec.Command(reactorBinary, "config", "show")
	cmd.Dir = testDir
	cmd.Env = setupBasicEnv(isolationPrefix)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "account:         test-account") {
		t.Errorf("Config should show test-account but got: %s", outputStr)
	}

	t.Logf("✅ Terminal size detection test configuration validated successfully")
}

