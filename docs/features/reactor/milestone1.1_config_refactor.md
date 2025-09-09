# **Implementation Plan: Milestone 1.1 - `devcontainer.json` Configuration**

Version: 1.1
Status: Approved
Date: 2025-08-28

## **1. Objective**

This document outlines the technical plan to refactor the `pkg/config` package to natively support the `devcontainer.json` specification. This is the first and most critical task in making `reactor` a first-class CLI for Dev Containers.

The goal is to replace the legacy `.reactor.conf` YAML parsing with a robust system for finding and parsing `devcontainer.json` files. This file will now be the **single source of truth** for environment configuration, including `reactor`-specific extensions.

## **2. Implementation Details**

### **Task 2.1: Add New Dependencies**

*   **Action:** Add a new Go module dependency for a JSONC (JSON with Comments) parser.
*   **Recommendation:** `github.com/tailscale/hujson` is a lightweight, well-tested, and suitable choice.

### **Task 2.2: Define `devcontainer.json` Go Structs**

*   **Action:** In `pkg/config/models.go`, create a new set of structs to represent the `devcontainer.json` file format. These structs will be the target for unmarshalling the parsed JSON.
*   **Implementation:**

    ```go
    // DevContainerConfig represents the structure of a devcontainer.json file.
    type DevContainerConfig struct {
        Name              string                   `json:"name"`
        Image             string                   `json:"image"`
        Build             *Build                   `json:"build"`
        ForwardPorts      []interface{}            `json:"forwardPorts"` // Can be int or string "host:container"
        RemoteUser        string                   `json:"remoteUser"`
        PostCreateCommand string                   `json:"postCreateCommand"`
        Customizations    *Customizations          `json:"customizations"`
    }

    // Build defines Docker build properties.
    type Build struct {
        Dockerfile string `json:"dockerfile"`
        Context    string `json:"context"`
    }

    // Customizations block for tool-specific settings.
    type Customizations struct {
        Reactor *ReactorCustomizations `json:"reactor"`
    }

    // ReactorCustomizations defines reactor-specific settings.
    type ReactorCustomizations struct {
        Account        string `json:"account"`
        DefaultCommand string `json:"defaultCommand"`
    }
    ```

### **Task 2.3: Implement File Discovery & Parsing**

*   **Action:** In `pkg/config/loader.go`, create the functions responsible for finding and parsing the configuration file.
*   **Implementation:**
    1.  **`FindDevContainerFile(dir string) (string, bool, error)`**: This function will search the given directory `dir` for the dev container configuration file in the order: `.devcontainer/devcontainer.json`, then `.devcontainer.json`.
    2.  **`LoadDevContainerConfig(filePath string) (*DevContainerConfig, error)`**: This function will take a file path, read it, use `hujson` to convert JSONC to standard JSON, and then `json.Unmarshal` it into the `DevContainerConfig` struct.

### **Task 2.4: Refactor `config.Service` and Define Resolution Logic**

*   **Action:** The `config.Service` will be refactored to use the new `devcontainer.json`-native workflow. The `ResolveConfiguration` method must follow this exact order of operations:
    1.  **Find `devcontainer.json`:** Search for the configuration file using `FindDevContainerFile`.
    2.  **Handle Not Found:** If no file is found, return an error. `reactor` cannot operate without it.
    3.  **Parse `devcontainer.json`:** If found, parse the file using `LoadDevContainerConfig`.
    4.  **Map to `ResolvedConfig`:** Transform the parsed `DevContainerConfig` into the canonical `ResolvedConfig` struct. This mapping includes all environment settings and reactor-specific extensions (like `account`) from the `customizations.reactor` block.

*   **Critical Architecture Note:** The data flow is: `devcontainer.json` → `DevContainerConfig` → `ResolvedConfig` → rest of application. The `ResolvedConfig` struct remains the canonical internal data model that decouples our core logic from the configuration source. The `pkg/core` package and `NewContainerBlueprint` function will continue to work with `ResolvedConfig` unchanged.

### **Task 2.5: Unit Testing**

*   **Action:** Create `pkg/config/devcontainer_test.go`.
*   **Required Tests:**
    *   Test `FindDevContainerFile` for all scenarios (finds in both locations, handles not found).
    *   Test `LoadDevContainerConfig` with valid files, files with comments, and malformed JSON.
    *   Test the refactored `config.Service` correctly maps `DevContainerConfig` to `ResolvedConfig`, including the `customizations.reactor.account` key and other reactor-specific extensions.
    *   Test the complete data flow: `devcontainer.json` → `DevContainerConfig` → `ResolvedConfig` transformation.

## **3. Definition of Done**

This task is complete when:
*   All new functions (`FindDevContainerFile`, `LoadDevContainerConfig`) are implemented as described.
*   The `config.Service` is fully refactored to use the `devcontainer.json` → `DevContainerConfig` → `ResolvedConfig` data flow.
*   All legacy `.reactor.conf` environment logic has been removed from the configuration layer.
*   The `ResolvedConfig` struct remains unchanged and continues to serve as the canonical internal data model.
*   All new logic is covered by comprehensive unit tests with >80% coverage.
*   The existing integration tests are updated to use `.devcontainer/devcontainer.json` fixtures and are all passing.
*   The `pkg/core` package continues to work unchanged with `ResolvedConfig` input.