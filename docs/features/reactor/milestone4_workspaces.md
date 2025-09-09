# **Feature Design Document: M4 - Multi-Container Workspaces**

Version: 1.3
Status: Approved
Author(s): Gemini, cam
Date: 2025-09-02

## **1. The 'Why': Rationale & User Focus**
*(This section remains unchanged)*

### **1.1. High-level summary**
This feature introduces "Workspaces," a powerful new capability in `reactor` for defining and managing a collection of related but separate Dev Container projects as a single, cohesive unit. By creating a simple `reactor-workspace.yml` file, developers can orchestrate a complete stack of microservices (e.g., frontend, backend API, database) with a single command, dramatically simplifying complex development workflows.

### **1.2. User personas**
*   **Primary Persona: The Microservice Developer ("Stack Dev")**: A developer working on a modern application composed of multiple services, often in different repositories or directories. They need to run, test, and manage the entire application stack simultaneously on their local machine.

### **1.3. Problem statement & user stories**
**Problem Statement:**
As applications grow in complexity, developers often need to run more than one service at a time. Managing multiple, independent `reactor up` sessions in different terminal windows is manual, cumbersome, and error-prone. There is no way to start, stop, or see the status of an entire application stack with a single, unified command.

**User Stories:**
*   As a **Stack Dev**, I want to define all my project's services in a single `reactor-workspace.yml` file, so I have a single source of truth for my local development stack.
*   As a **Stack Dev**, I want to run `reactor workspace up` to start my entire stack (API, frontend, etc.) with one command.
*   As a **Stack Dev**, I want to run `reactor workspace up api frontend` to start only specific services from my workspace.
*   As a **Stack Dev**, I want to run `reactor workspace exec api -- <command>` to execute a command in a specific running service.
*   As a **Stack Dev**, I want to run `reactor workspace down` to stop and clean up all containers in my stack with one command.
*   As a **Stack Dev**, I want to run `reactor workspace list` to see the status of all services defined in my workspace at a glance.

### **1.4. Success metrics**
**Technical Metrics:**
*   The `reactor workspace` commands (`up`, `down`, `list`, `exec`, `validate`) are fully functional.
*   The tool can successfully parse a `reactor-workspace.yml` file and start all defined services in parallel.
*   Host port conflicts between services are detected, and the `up` command fails with a clear error.
*   The implementation is covered by comprehensive unit and integration tests.

## **2. The 'How': Technical Design & Architecture**

### **2.1. Core Architectural Decision: Direct Function Invocation via Orchestrator**

The workspace feature will be implemented by refactoring the existing single-container logic into a reusable Go package, which will then be invoked directly. **We will NOT use `exec.Command()` to call the `reactor` binary itself.**

This requires a foundational refactoring (**PR 0**):
*   A new `pkg/orchestrator` package will be created.
*   The core logic from `cmd/reactor/up.go` and `down.go` will be moved into public functions within this new package.
*   The `up` and `down` command handlers will be simplified to be thin wrappers around these new public functions.
*   The new `workspace` service will also call these functions directly, running each call in a separate goroutine.

### **2.2. Detailed design**

#### **2.2.1. The `reactor-workspace.yml` file**

This new file is the single source of truth for a workspace. `reactor` will automatically look for a file named `reactor-workspace.yml` or `reactor-workspace.yaml` in the current directory.

**Schema Definition:**
```yaml
# reactor-workspace.yml

# The version of the workspace file format. Must be "1".
version: "1"

# A map of services that make up the workspace.
# The key for each service (e.g., "api", "frontend") is its logical name.
services:
  api:
    # The relative path from this workspace file to the directory
    # containing the service's devcontainer.json file.
    # Type: string, Required
    path: ./services/backend-api

  frontend:
    path: ./services/frontend-app
    # (Optional) Override the account specified in the service's devcontainer.json.
    # This is useful for running the entire stack with a specific set of credentials.
    # Type: string, Optional
    account: work-account

  database:
    path: ./services/database
```

**Design Rationale:** The `services` map structure (using service names as keys) was chosen intentionally to provide a familiar user experience for developers already accustomed to the `docker-compose.yml` format. This aligns with the project's ethos of leveraging existing conventions to make the tool intuitive.

#### **2.2.2. New Package: `pkg/workspace`**

A new package will be created to handle the logic for this feature.

*   **`pkg/workspace/models.go`**: This file will define the Go structs that map to the YAML file schema.

    ```go
    package workspace

    // Workspace defines the structure of the reactor-workspace.yml file.
    type Workspace struct {
        Version  string             `yaml:"version"`
        Services map[string]Service `yaml:"services"`
    }

    // Service defines the configuration for a single service within the workspace.
    type Service struct {
        Path    string `yaml:"path"`
        Account string `yaml:"account,omitempty"`
    }
    ```

*   **`pkg/workspace/parser.go`**: This file will contain the logic to find and parse the workspace file.
    *   It must implement `FindWorkspaceFile() (string, bool, error)` which looks for `reactor-workspace.yml` and then `reactor-workspace.yaml`.
    *   It must implement `ParseWorkspaceFile(path string) (*Workspace, error)` which reads the file and unmarshals it into the `Workspace` struct. It must also validate that `Version` is "1" and that the `services` map is not empty.

*   **`pkg/workspace/service.go`**: This file will contain the core orchestration logic. It will accept a parsed `Workspace` object and will be responsible for calling the existing `reactor` logic for each service. It should use goroutines and waitgroups to manage parallel execution.

#### **2.2.3. New Package: `pkg/orchestrator`**

This new package will contain the core, reusable logic for bringing a single dev container environment up or down.

*   **`pkg/orchestrator/orchestrator.go`**: This file will define the public interface.

    ```go
    package orchestrator

    import (
        "context"
        "github.com/dyluth/reactor/pkg/config"
    )

    // UpConfig contains all necessary, pre-resolved parameters for an 'up' operation.
    type UpConfig struct {
        // The absolute path to the service's project directory (the one containing .devcontainer).
        ProjectDirectory string

        // An optional account override from the workspace file. If empty, the account
        // from the devcontainer.json file will be used.
        AccountOverride string

        // A flag to force a rebuild of the container image.
        ForceRebuild bool

        // An optional map of labels to apply to the container (for workspace tracking).
        Labels map[string]string

        // An optional name prefix for the container (e.g., "reactor-ws-api-").
        NamePrefix string
    }

    // Up orchestrates the entire 'reactor up' logic for a single service.
    // It returns the final resolved config and container ID on success.
    func Up(ctx context.Context, config UpConfig) (*config.ResolvedConfig, string, error) {
        // ... implementation ...
    }

    // Down orchestrates the 'reactor down' logic for a single service.
    func Down(ctx context.Context, projectDirectory string) error {
        // ... implementation ...
    }
    ```

#### **2.2.3. Container Identity and State Management**

To reliably track which containers belong to which workspace, we will use both labels and a distinct naming convention.

*   **Label:** Every container started by a workspace command will have a Docker label applied: `com.reactor.workspace.instance=<hash>`, where `<hash>` is the **SHA256 hash** of the **canonical, absolute path** of the `reactor-workspace.yml` file.
*   **Naming Convention:** Workspace containers will be named with a `reactor-ws-` prefix, followed by the service name from the workspace file (e.g., `reactor-ws-api-<project-hash>`). The `<project-hash>` is the existing hash derived from the service's directory path.
*   **Docker Service Update:** The `docker.ContainerSpec` struct in `pkg/docker` must be updated to include a `Labels map[string]string` field.

#### **2.2.4. New CLI Commands (`cmd/reactor/workspace*`)**

A new command group, `reactor workspace`, will be created. All commands will accept an optional `--file` / `-f` flag to specify the path to the workspace file.

1.  **`reactor workspace up [service...]`**
    *   **Logic:**
        1.  Find and parse the workspace file.
        2.  **Pre-flight Check:** Before starting any containers, parse the `devcontainer.json` for *all* services to be started and check for host port conflicts. If any conflicts exist, the command must fail immediately with a clear error.
        3.  For each service to be started (all services, or those specified as arguments):
            a. Launch a goroutine to call the refactored `orchestrator.Up()` function.
            b. The `orchestrator.Up()` call must be executed with its `ProjectDirectory` set to the service's absolute path. All sub-paths within the service's `devcontainer.json` (e.g., for `build.context`) remain relative to that service's directory.
            c. The `orchestrator.Up` function is responsible for applying the `AccountOverride`.
            d. Stream `stdout`/`stderr` to the console, prefixed with the service name and a distinct color.
        4.  Wait for all goroutines to complete.
        5.  **Failure Definition:** A service "fails" if the `orchestrator.Up()` function returns an error (e.g., build error, non-zero `postCreateCommand` exit code).
        6.  **No Retry:** There will be no automatic retry logic in this milestone.
        7.  **Final Report:** Print a summary of which services started successfully and which failed.

2.  **`reactor workspace exec <service> -- <command...>`**
    *   This will find the container corresponding to the specified service (by name prefix and workspace label) and execute the given command within it, streaming I/O.

3.  **`reactor workspace down [service...]`**
    *   If no services are specified, it will find all containers with the workspace label and run `orchestrator.Down()` for them in parallel.
    *   If services are specified, it will act only on those.

4.  **`reactor workspace list`** and **`validate`**
    *   `validate` will parse the workspace file and, for each service, verify that the specified `path` exists and contains a valid `devcontainer.json` file.

### **2.3. Phased implementation plan**

*   **PR 0: Orchestrator Refactoring**
    *   [ ] Create the new `pkg/orchestrator` package.
    *   [ ] Define the `UpConfig` struct and the `Up()` and `Down()` function signatures as specified above.
    *   [ ] Move the core logic from the `up` and `down` command handlers into these new functions. The logic must be modified to use the `UpConfig.ProjectDirectory` instead of the process's CWD.
    *   [ ] Update the `up` and `down` commands to be thin wrappers that build the `UpConfig` and call the new orchestrator functions.
    *   [ ] **Goal:** All existing single-container tests must pass after this refactoring.

*   **PR 1: Parser, `validate`, and `list` commands**
    *   [ ] Implement `pkg/workspace` and the `validate` and `list` commands.

*   **PR 2: `up` and `exec` commands**
    *   [ ] Implement the `reactor workspace up` and `exec` commands, including parallel execution, output streaming, and the pre-flight port conflict check.

*   **PR 3: `down` command and Integration Testing**
    *   [ ] Implement the `reactor workspace down` command.
    *   [ ] Add comprehensive integration tests for the full workspace lifecycle.

### **2.4. Testing strategy**
*(Unchanged from v1.1, with the addition of the `validate` and `exec` tests)*

### **3. The 'What Ifs': Risks & Mitigation**
*(Unchanged from v1.1, with the addition of the following)*
*   **Risk:** Multiple services define the same host port.
*   **Mitigation:** The `up` command will perform a pre-flight check and fail with a clear error before starting any containers.
*   **Risk:** Service paths use path traversal (`../`) to escape the workspace directory.
*   **Mitigation:** All service paths will be resolved to an absolute path and validated to ensure they are still located within the parent directory of the workspace file.