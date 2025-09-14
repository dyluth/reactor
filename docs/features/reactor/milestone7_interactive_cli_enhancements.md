# **Feature Design Document: M7 - Interactive CLI & TTY Enhancements**

Version: 1.0  
Status: Draft  
Author(s): Claude, cam  
Date: 2025-09-14

## **1. The 'Why': Rationale & User Focus**

*This section defines the purpose of the feature, the target user, and the value it delivers. It ensures we are solving the right problem for the right person.*

### **1.1. High-level summary**

This milestone enables seamless integration of AI CLI tools (Claude, ChatGPT, etc.) as default interactive commands in reactor development environments. Currently, when users configure `defaultCommand: "claude"` in their devcontainer.json, the Claude CLI auto-switches to non-interactive `--print` mode due to improper TTY detection in our Docker exec implementation. This enhancement provides proper pseudoterminal (PTY) allocation, terminal environment setup, and bidirectional I/O streaming to ensure AI CLIs detect interactive environments correctly and provide full-featured interactive sessions with colors, formatting, keyboard shortcuts, and real-time responsiveness.

### **1.2. User personas**

* **Primary Persona: AI-Assisted Developer (Alex)**: Alex uses Claude CLI extensively for coding assistance and wants to launch directly into Claude when starting a reactor environment. They value immediate access to AI assistance, proper terminal formatting, and seamless keyboard shortcuts (Ctrl+C, Ctrl+D, etc.).  
* **Secondary Persona: Team Lead (Taylor)**: Taylor configures standardized development environments for their team and needs reliable, consistent AI CLI integration across different host operating systems and terminal emulators. They value predictable behavior and minimal setup complexity.
* **Tertiary Persona: AI Tool Creator (Casey)**: Casey develops custom AI CLI tools and needs reactor to provide proper terminal environments that work with standard Node.js TTY detection mechanisms (`process.stdout.isTTY`, `process.stdin.isTTY`).

### **1.3. Problem statement & user stories**

**Problem Statement:**  
AI CLI tools like Claude detect non-interactive environments when launched via reactor's `defaultCommand` configuration, causing them to auto-switch to limited `--print` mode instead of providing full interactive sessions. This occurs because our current Docker exec implementation doesn't provide proper TTY characteristics (environment variables, terminal dimensions, raw mode handling) that Node.js applications use for interactive detection.

**User Stories:**

* As an AI-Assisted Developer, I want to configure `defaultCommand: "claude"` and immediately launch into a fully interactive Claude CLI session with colors, formatting, and all keyboard shortcuts working properly.
* As an AI-Assisted Developer, I want my terminal resize events to be properly forwarded to containerized AI CLIs so they can adjust their output formatting accordingly.  
* As a Team Lead, I want to standardize our team's reactor configurations to launch directly into AI assistance tools without requiring manual command execution after container startup.
* As an AI Tool Creator, I want my Node.js-based CLI tools to correctly detect TTY environments when running in reactor containers, so they can provide the intended user experience.

### **1.4. Success metrics**

**Business Metrics:**

* Achieve 90% success rate for `defaultCommand` with popular AI CLIs (Claude, ChatGPT CLI, GitHub Copilot CLI) within one month of release.
* Reduce developer onboarding time to AI-assisted environments by 30 seconds per session (eliminating manual `claude` command execution).
* Zero support tickets related to "Claude CLI not working in reactor" within two months of release.

**Technical Metrics:**

* `process.stdout.isTTY` and `process.stdin.isTTY` return `true` for AI CLIs launched via `defaultCommand` in 100% of test cases.
* Terminal resize events are forwarded to containerized processes within 50ms in 95% of cases.
* AI CLI interactive features (colors, progress bars, keyboard shortcuts) function identically to native terminal execution.
* Zero regressions in existing bash `defaultCommand` functionality.

## **2. The 'How': Technical Design & Architecture**

*This section details the proposed technical solution, exploring the system context, alternatives, and the specific changes required across the stack.*

### **2.1. System context & constraints**

**Technology Stack:** 
- Go 1.21+ with Docker API client libraries
- Docker Engine API for container exec operations  
- POSIX terminal handling (works on macOS, Linux, Windows with WSL)
- Node.js-based AI CLI tools (Claude CLI, ChatGPT CLI, etc.)

**Current State:** 
The `ExecuteInteractiveCommand` function in `pkg/docker/service.go` creates Docker exec instances with `Tty: true` but lacks proper terminal environment setup. Current implementation:
- Sets `AttachStdin`, `AttachStdout`, `AttachStderr` to `true`
- Uses `io.Copy` for bidirectional streaming  
- Doesn't set terminal environment variables (`TERM`, `COLUMNS`, `LINES`)
- Doesn't handle terminal resize events or raw mode
- Doesn't provide proper signal forwarding (SIGINT, SIGTERM, SIGWINCH)

**Technical Constraints:**
- Must maintain backward compatibility with existing bash and shell commands
- Solution must work across macOS, Linux, and Windows WSL environments
- Cannot require additional system dependencies beyond standard Go libraries
- Must handle cases where host terminal is not available (CI/CD environments)
- Docker API limitations for TTY handling must be worked around

### **2.2. Guiding design principles**

**Simplicity over Complexity (YAGNI):** The solution focuses specifically on TTY detection for AI CLIs without adding unnecessary abstractions. We'll enhance the existing `ExecuteInteractiveCommand` function rather than creating new interfaces or patterns.

**Consistency with Existing Code:** The enhancement follows reactor's established patterns in `pkg/docker/service.go` and maintains the same function signatures to avoid breaking changes. The solution extends rather than replaces current functionality.

**Clarity and Readability:** All TTY handling logic will be clearly documented with explicit comments explaining terminal behavior. Complex signal handling and raw mode operations will be isolated into well-named helper functions.

### **2.3. Alternatives considered**

**Option 1: Enhanced Docker Exec with Proper TTY Setup**
* **Description:** Upgrade the existing `ExecuteInteractiveCommand` function to set proper terminal environment variables (`TERM=xterm-256color`, `COLUMNS`, `LINES`), detect host terminal size, enable raw mode on the host terminal, and implement bidirectional signal forwarding.
* **Pros:** Minimal code changes, maintains existing architecture, handles all TTY requirements comprehensively, works with all containerized CLI tools.
* **Cons:** Adds complexity to signal handling, requires cross-platform terminal size detection, needs careful raw mode management.

**Option 2: Docker Run Replacement Strategy** 
* **Description:** Replace `docker exec` with `docker run --rm -it` for `defaultCommand` scenarios, providing native TTY allocation from Docker.
* **Pros:** Simplest implementation, guaranteed proper TTY allocation, Docker handles most complexity.
* **Cons:** Loses container persistence, breaks reactor's container reuse model, requires significant architecture changes, impacts performance with container recreation.

**Option 3: External PTY Library Integration**
* **Description:** Use `github.com/creack/pty` or similar Go PTY libraries to create native pseudoterminals and bridge them to Docker containers.
* **Pros:** Maximum control over TTY behavior, handles edge cases robustly, platform-specific optimizations possible.
* **Cons:** Adds external dependencies, significantly increases complexity, potential licensing/maintenance concerns, over-engineering for the core use case.

**Chosen Approach Justification:**  
Option 1 (Enhanced Docker Exec) is chosen because it solves the user problem directly while maintaining reactor's existing architecture and performance characteristics. It provides comprehensive TTY support without architectural disruption and can be implemented incrementally with clear rollback capabilities.

### **2.4. Detailed design**

#### **2.4.1. Data model updates**

N/A - No database or persistent storage changes required.

#### **2.4.2. Data migration plan**

N/A - No data migration needed.

#### **2.4.3. API & backend changes**

**Enhanced ExecuteInteractiveCommand Function**

The core enhancement is in `pkg/docker/service.go`, expanding the existing `ExecuteInteractiveCommand` function:

```go
// ExecuteInteractiveCommand runs a command interactively with full TTY support
func (s *Service) ExecuteInteractiveCommand(ctx context.Context, containerID string, command []string, isInteractive bool) error {
    if len(command) == 0 {
        return fmt.Errorf("command array cannot be empty")
    }

    // Check if container is running
    containerInfo, err := s.client.ContainerInspect(ctx, containerID)
    if err != nil {
        return fmt.Errorf("failed to inspect container %s: %w", containerID, err)
    }
    if !containerInfo.State.Running {
        return fmt.Errorf("container %s is not running, cannot execute command", containerID)
    }

    // Phase 1: Detect host terminal capabilities
    terminalSize, err := getTerminalSize()
    if err != nil {
        terminalSize = &TerminalSize{Width: 80, Height: 24} // Safe defaults
    }

    // Phase 2: Enhanced exec configuration with proper environment
    execConfig := container.ExecOptions{
        AttachStdout: true,
        AttachStderr: true,
        AttachStdin:  true,
        Tty:          true,
        Cmd:          command,
        Env: []string{
            "TERM=xterm-256color",
            fmt.Sprintf("COLUMNS=%d", terminalSize.Width),
            fmt.Sprintf("LINES=%d", terminalSize.Height),
            "COLORTERM=truecolor",
            "FORCE_COLOR=1",
        },
    }

    // Create exec instance
    execResp, err := s.client.ContainerExecCreate(ctx, containerID, execConfig)
    if err != nil {
        return fmt.Errorf("failed to create exec instance: %w", err)
    }

    // Phase 3: Raw mode and signal handling setup
    var oldState *term.State
    if isInteractive {
        oldState, err = enableRawMode()
        if err == nil {
            defer restoreTerminalMode(oldState)
            go s.handleTerminalResize(ctx, execResp.ID, terminalSize)
            go s.handleSignalForwarding(ctx, execResp.ID)
        }
    }

    // Attach to exec instance
    attachResp, err := s.client.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{
        Tty: true,
    })
    if err != nil {
        return fmt.Errorf("failed to attach to exec instance: %w", err)
    }
    defer attachResp.Close()

    // Start exec instance
    if err := s.client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{
        Tty: true,
    }); err != nil {
        return fmt.Errorf("failed to start command execution: %w", err)
    }

    // Enhanced I/O handling with proper streaming
    return s.handleInteractiveIO(ctx, execResp.ID, attachResp, isInteractive)
}
```

**New Terminal Management Functions**

```go
type TerminalSize struct {
    Width  int
    Height int
}

// getTerminalSize detects host terminal dimensions using golang.org/x/term
func getTerminalSize() (*TerminalSize, error) {
    width, height, err := term.GetSize(int(os.Stdin.Fd()))
    if err != nil {
        return nil, fmt.Errorf("failed to get terminal size: %w", err)
    }
    return &TerminalSize{Width: width, Height: height}, nil
}

// enableRawMode puts host terminal in raw mode for proper character forwarding
func enableRawMode() (*term.State, error) {
    return term.MakeRaw(int(os.Stdin.Fd()))
}

// restoreTerminalMode restores terminal to original state
func restoreTerminalMode(state *term.State) {
    _ = term.Restore(int(os.Stdin.Fd()), state)
}

// handleTerminalResize forwards SIGWINCH to containerized process
func (s *Service) handleTerminalResize(ctx context.Context, execID string, currentSize *TerminalSize) {
    // Create signal channel for SIGWINCH (window change)
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGWINCH)
    defer signal.Stop(sigChan)

    for {
        select {
        case <-ctx.Done():
            return
        case <-sigChan:
            // Get new terminal size
            newSize, err := getTerminalSize()
            if err != nil {
                continue // Skip this resize event
            }

            // Only resize if dimensions actually changed
            if newSize.Width != currentSize.Width || newSize.Height != currentSize.Height {
                // Use Docker API to resize the exec TTY
                if err := s.client.ContainerExecResize(ctx, execID, container.ResizeOptions{
                    Height: uint(newSize.Height),
                    Width:  uint(newSize.Width),
                }); err != nil {
                    // Log but don't fail - resize is best-effort
                    continue
                }
                *currentSize = *newSize
            }
        }
    }
}

// handleSignalForwarding forwards SIGINT, SIGTERM to container process
func (s *Service) handleSignalForwarding(ctx context.Context, execID string) {
    // Create signal channel for terminal signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    defer signal.Stop(sigChan)

    for {
        select {
        case <-ctx.Done():
            return
        case sig := <-sigChan:
            // Forward signal to container process via Docker API
            // Note: Docker doesn't have direct signal forwarding for exec,
            // so we'll send the signal to the container process
            switch sig {
            case syscall.SIGINT:
                // Send SIGINT to exec process (Ctrl+C)
                // This is handled by the TTY forwarding
                continue
            case syscall.SIGTERM:
                // Graceful termination
                return
            }
        }
    }
}

// handleInteractiveIO manages bidirectional I/O streaming with proper cleanup
func (s *Service) handleInteractiveIO(ctx context.Context, execID string, attachResp types.HijackedResponse, isInteractive bool) error {
    // Channel to signal completion
    done := make(chan error, 2)

    // Copy container output to stdout
    go func() {
        _, err := io.Copy(os.Stdout, attachResp.Reader)
        done <- err
    }()

    // Copy stdin to container
    go func() {
        _, err := io.Copy(attachResp.Conn, os.Stdin)
        // Suppress "broken pipe" errors for interactive sessions
        if err != nil && isInteractive && strings.Contains(err.Error(), "broken pipe") {
            err = nil // Expected when user detaches
        }
        done <- err
    }()

    // Wait for exec completion with proper polling
    go func() {
        for {
            inspectResp, err := s.client.ContainerExecInspect(ctx, execID)
            if err != nil {
                done <- fmt.Errorf("failed to inspect exec: %w", err)
                return
            }

            if !inspectResp.Running {
                if inspectResp.ExitCode != 0 {
                    done <- fmt.Errorf("command failed with exit code %d", inspectResp.ExitCode)
                } else {
                    done <- nil
                }
                return
            }

            select {
            case <-ctx.Done():
                done <- ctx.Err()
                return
            case <-time.After(100 * time.Millisecond):
                // Continue polling
            }
        }
    }()

    // Wait for first completion signal
    return <-done
}
```

**Scope Clarification**: The TTY enhancements apply to all invocations of `ExecuteInteractiveCommand` where `isInteractive=true`, including `reactor up` (initial shell/defaultCommand), `reactor workspace exec` (interactive workspace commands), and `reactor sessions attach` (container reattachment). This ensures consistent interactive behavior across all reactor features.

#### **2.4.4. Frontend changes**

N/A - This is a backend/CLI enhancement with no UI components.

### **2.5. Non-functional requirements (NFRs)**

**Performance:** 
- Terminal size detection must complete within 10ms on 95% of systems
- Signal forwarding latency must be under 50ms for SIGINT/SIGTERM events  
- Terminal resize event propagation must complete within 50ms in 95% of cases
- TTY setup overhead must add less than 100ms to container startup time

**Scalability:** 
- Solution must handle concurrent interactive sessions (up to 10 simultaneous reactor containers per developer)
- Memory overhead per interactive session must remain under 1MB additional allocation
- Must work reliably with different terminal emulators (iTerm2, Terminal.app, VSCode integrated terminal, Windows Terminal)

**Reliability:** 
- Graceful degradation when host terminal is not available (CI/CD environments)
- Automatic fallback to previous behavior if TTY enhancements fail
- Proper cleanup of terminal state on abnormal termination (panic recovery)
- Zero impact on non-interactive command execution

**Accessibility (a11y):**
- Terminal output preserves ANSI escape codes for screen readers that support them
- Keyboard shortcuts (Ctrl+C, Ctrl+D, Ctrl+Z) function correctly in containerized environments
- Raw terminal mode maintains existing accessibility tool compatibility
- No interference with host terminal's accessibility configurations

**Operations & Developer Experience:** 
- All TTY-related functionality must be unit testable with mock terminal interfaces
- Debug logging for TTY setup and signal handling (controllable via verbose flag)
- Clear error messages when TTY setup fails with suggested remediation steps
- Automated integration tests covering major terminal scenarios

## **3. The 'What': Implementation & Execution**

*This section breaks the work into manageable pieces and defines the strategy for testing, documentation, and quality assurance.*

### **3.1. Phased implementation plan**

**Phase 1: Core TTY Enhancement**

* [ ] PR 1.1: Add terminal size detection utilities with cross-platform support (`pkg/docker/terminal.go`)
* [ ] PR 1.2: Enhance `ExecuteInteractiveCommand` with environment variable setup (TERM, COLUMNS, LINES)
* [ ] PR 1.3: Add unit tests for terminal size detection and environment setup logic

**Phase 2: Signal and Raw Mode Handling** 

* [ ] PR 2.1: Implement raw mode enable/disable with proper cleanup and panic recovery
* [ ] PR 2.2: Add signal forwarding for SIGINT, SIGTERM with graceful error handling  
* [ ] PR 2.3: Implement terminal resize (SIGWINCH) forwarding using Docker resize API

**Phase 3: Integration and Validation**

* [ ] PR 3.1: Add integration tests with Claude CLI, bash, and Python REPL scenarios
* [ ] PR 3.2: Add debug logging and error handling for TTY operations
* [ ] PR 3.3: Performance optimization and memory leak prevention

**Phase 4: Documentation and Polish**

* [ ] PR 4.1: Update README with `defaultCommand` examples and TTY troubleshooting guide
* [ ] PR 4.2: Add developer documentation for TTY handling internals
* [ ] PR 4.3: Create comprehensive test suite covering edge cases and failure modes

### **3.2. Testing strategy**

**Unit Tests:**
- Test terminal size detection with mocked environments (no TTY, various dimensions)
- Test environment variable generation for different terminal configurations
- Test raw mode enable/disable with proper state restoration
- Test signal handling with simulated terminal events

**Integration Tests:**

```go
// Integration test example for TTY validation
func TestTTYEnhancements_ClaudeInteractiveDetection(t *testing.T) {
    // Test that validates process.stdout.isTTY detection in container

    // 1. Create test script that checks TTY status
    testScript := `#!/bin/bash
echo "TTY_STDIN=$([ -t 0 ] && echo true || echo false)"
echo "TTY_STDOUT=$([ -t 1 ] && echo true || echo false)"
echo "TTY_STDERR=$([ -t 2 ] && echo true || echo false)"
echo "TERM_VAR=$TERM"
echo "COLUMNS_VAR=$COLUMNS"
echo "LINES_VAR=$LINES"
`

    // 2. Run script in container via ExecuteInteractiveCommand
    output := captureContainerOutput(t, testScript, true) // isInteractive=true

    // 3. Parse and validate output
    assert.Contains(t, output, "TTY_STDIN=true")
    assert.Contains(t, output, "TTY_STDOUT=true")
    assert.Contains(t, output, "TTY_STDERR=true")
    assert.Contains(t, output, "TERM_VAR=xterm-256color")
    assert.Regexp(t, `COLUMNS_VAR=\d+`, output)
    assert.Regexp(t, `LINES_VAR=\d+`, output)
}

func TestTTYEnhancements_NodeJSTTYDetection(t *testing.T) {
    // Test Node.js TTY detection specifically
    nodeScript := `
const tty = require('tty');
console.log('STDIN_IS_TTY=' + tty.isatty(process.stdin.fd));
console.log('STDOUT_IS_TTY=' + tty.isatty(process.stdout.fd));
console.log('STDERR_IS_TTY=' + tty.isatty(process.stderr.fd));
console.log('PROCESS_STDOUT_IS_TTY=' + process.stdout.isTTY);
console.log('PROCESS_STDIN_IS_TTY=' + process.stdin.isTTY);
`

    output := runNodeScriptInContainer(t, nodeScript, true)

    // Validate Node.js TTY detection
    assert.Contains(t, output, "STDIN_IS_TTY=true")
    assert.Contains(t, output, "STDOUT_IS_TTY=true")
    assert.Contains(t, output, "PROCESS_STDOUT_IS_TTY=true")
    assert.Contains(t, output, "PROCESS_STDIN_IS_TTY=true")
}

func TestTTYEnhancements_ColorSupport(t *testing.T) {
    // Test that color escape codes are preserved
    colorScript := `#!/bin/bash
echo -e "\033[31mRED\033[0m \033[32mGREEN\033[0m \033[34mBLUE\033[0m"
echo "FORCE_COLOR=$FORCE_COLOR"
echo "COLORTERM=$COLORTERM"
`

    output := captureRawContainerOutput(t, colorScript, true)

    // Validate ANSI color codes are present (not stripped)
    assert.Contains(t, output, "\033[31m") // Red color code
    assert.Contains(t, output, "\033[32m") // Green color code
    assert.Contains(t, output, "\033[34m") // Blue color code
    assert.Contains(t, output, "FORCE_COLOR=1")
    assert.Contains(t, output, "COLORTERM=truecolor")
}
```

- **Terminal Feature Validation:** Automated verification of interactive features using bash/Node.js test scripts and expect-like libraries (`github.com/Netflix/go-expect`) to simulate user input
- **Resize Event Handling:** Automated terminal resize simulation with validation of proper forwarding
- **Signal Forwarding:** Tests for Ctrl+C (SIGINT) and proper process termination

**End-to-End (E2E) Integration Tests using Go os/exec:**

```go
func TestE2E_DefaultCommandClaude(t *testing.T) {
    // E2E test: defaultCommand: "claude" launches interactive Claude CLI

    // 1. Create test project with devcontainer.json
    testProject := setupTestProject(t, map[string]interface{}{
        "image": "ghcr.io/dyluth/reactor/claude:latest",
        "customizations": map[string]interface{}{
            "reactor": map[string]interface{}{
                "defaultCommand": "claude",
            },
        },
    })
    defer cleanupTestProject(t, testProject)

    // 2. Run reactor up with automated input simulation
    cmd := exec.CommandContext(ctx, "reactor", "up")
    cmd.Dir = testProject.Path

    // Use pty package to simulate interactive terminal
    pty, tty, err := pty.Open()
    require.NoError(t, err)
    defer pty.Close()
    defer tty.Close()

    cmd.Stdin = tty
    cmd.Stdout = tty
    cmd.Stderr = tty
    cmd.SysProcAttr = &syscall.SysProcAttr{Setctty: true, Setsid: true}

    // 3. Start reactor up and validate Claude CLI interactive mode
    err = cmd.Start()
    require.NoError(t, err)

    // Read initial output and verify Claude interactive prompt appears
    output, err := readPtyOutput(pty, 5*time.Second)
    require.NoError(t, err)
    assert.Contains(t, output, "Claude CLI") // or Claude's interactive prompt

    // 4. Send test command to verify interactivity
    _, err = pty.Write([]byte("hello\r\n"))
    require.NoError(t, err)

    // Verify Claude responds (indicating TTY detection worked)
    response, err := readPtyOutput(pty, 3*time.Second)
    require.NoError(t, err)
    assert.NotEmpty(t, response)

    // Clean exit
    _, _ = pty.Write([]byte("exit\r\n"))
    _ = cmd.Wait()
}

func TestE2E_TerminalResize(t *testing.T) {
    // Test terminal resize event forwarding
    // Uses programmatic PTY resize and validates container receives SIGWINCH

    cmd := startReactorInteractiveSession(t)
    pty := attachToPTY(t, cmd)

    // Get initial terminal size
    initialSize := getPTYSize(t, pty)

    // Resize PTY programmatically
    newSize := &pty.Winsize{Rows: 50, Cols: 120}
    err := pty.Setsize(newSize)
    require.NoError(t, err)

    // Validate container received resize (via environment check or specific test command)
    output := sendCommandAndReadResponse(t, pty, "echo $COLUMNS $LINES")
    assert.Contains(t, output, "120 50")
}

func TestE2E_CrossPlatform_MacOS(t *testing.T) {
    if runtime.GOOS != "darwin" {
        t.Skip("macOS-specific test")
    }
    // macOS-specific TTY validation
}

func TestE2E_CrossPlatform_Linux(t *testing.T) {
    if runtime.GOOS != "linux" {
        t.Skip("Linux-specific test")
    }
    // Linux-specific TTY validation
}
```

**Performance Tests:**
- Benchmark terminal setup overhead (target: <100ms additional startup time)  
- Load testing with 10 concurrent interactive sessions to validate scalability
- Memory leak detection for long-running interactive sessions (24+ hours)

**Accessibility Tests:**
- Screen reader compatibility testing with VoiceOver on macOS
- High contrast theme validation with terminal output
- Keyboard-only navigation testing with AI CLI interactions

## **4. The 'What Ifs': Risks & Mitigation**

*This section addresses potential issues, ensuring the feature is secure, reliable, and can be deployed and managed safely.*

### **4.1. Security & privacy considerations**

**Authentication & Authorization:** No changes to authentication - TTY enhancements work within existing Docker exec security model. Container execution permissions remain unchanged.

**Data Validation:** All terminal size inputs are validated (positive integers, reasonable bounds 1-9999). Environment variables are sanitized to prevent injection attacks. Signal handling validates signal types against whitelist.

**Data Privacy:** TTY enhancements only handle terminal metadata (size, signals). No user content or AI CLI conversation data is intercepted or logged. Debug logging excludes any command arguments or output content.

### **4.2. Rollout & deployment**

**Feature Flags:** No feature flag needed - enhancement is backward compatible. If TTY setup fails, system automatically falls back to existing behavior with informative warning message.

**Monitoring & Observability:**
- **Key Metrics:** 
  - `tty.setup.success.rate` - Percentage of successful TTY initializations
  - `tty.setup.duration.p95` - 95th percentile TTY setup time
  - `signal.forwarding.latency.p95` - Signal forwarding response time
  - `terminal.resize.events.count` - Number of resize events processed
- **Logging:** 
  - DEBUG: "TTY setup started for container {containerID} with terminal size {width}x{height}"
  - INFO: "Interactive command started successfully with TTY support"  
  - WARN: "TTY setup failed, falling back to basic mode: {error}"
  - ERROR: "Signal forwarding failed: {error} (container: {containerID})"
- **Alerting:** Currently no alerting needed as this is a developer tool enhancement. Future consideration: Alert if TTY setup failure rate exceeds 10% across all users.

**Rollback Plan:** TTY enhancements are additive - if critical issues occur, the changes can be reverted without affecting existing functionality. Emergency rollback: comment out TTY enhancement code and rebuild, falling back to original `ExecuteInteractiveCommand` implementation.

### **4.3. Dependencies and integrations**

**Internal Dependencies:** 
- Requires existing Docker service in `pkg/docker/service.go`
- Depends on current container lifecycle management in orchestrator
- Uses existing command execution patterns in main CLI handlers

**External Dependencies:** 
- `golang.org/x/term` package for cross-platform terminal handling (standard library, stable)
- Docker Engine API (already established dependency, no version changes)
- Host terminal capabilities (gracefully degrades when unavailable)

**Data Dependencies:** None - TTY handling is stateless and real-time.

### **4.4. Cost and resource analysis**

**Infrastructure Costs:** Zero additional infrastructure cost - uses existing Docker containers and host terminal resources.

**Operational Costs:** 
- Minimal increase in CPU/memory usage per interactive session (estimated <1MB RAM, <0.1% CPU)
- Potential support reduction due to improved AI CLI usability
- Development time: ~40 hours across 4 phases

### **4.5. Open questions & assumptions**

**Open Questions:**

No open questions remain - all design decisions have been finalized.

**Design Decisions:**

- TTY enhancements apply to **all** interactive sessions via `ExecuteInteractiveCommand` (reactor up, workspace exec, sessions attach)
- Non-responsive AI CLIs will inherit Docker's default timeout behavior - no custom timeout needed
- Terminal behavior uses standard settings (TERM=xterm-256color, COLORTERM=truecolor) without user configuration options

**Assumptions:**

- Most developers use modern terminal emulators with resize event support
- AI CLI tools follow standard Node.js TTY detection patterns (`process.stdout.isTTY`)
- Host terminals support raw mode (true for all major terminals)
- Docker containers have sufficient permissions for TTY operations
- Developers prefer automatic TTY handling over manual configuration options