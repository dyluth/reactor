# **Feature Design: `reactor` - The Command-Line Interface for Dev Containers**

Version: 2.0
Status: Approved
Date: 2025-08-28

## **1. Vision & Strategy**

### **1.1. High-Level Summary**

`reactor` is a high-performance, developer-focused **command-line interface (CLI) for the Dev Container standard**.

The core mission is to provide the fastest and most ergonomic terminal-based experience for creating, managing, and connecting to the standardized development environments that are rapidly becoming the industry norm. It embraces the `devcontainer.json` specification to ensure broad ecosystem compatibility while providing powerful extensions for credential management and multi-container orchestration.

### **1.2. User Personas**

*   **Primary Persona: The Modern Developer ("Dev")**: A software engineer who uses Dev Containers (via VS Code, GitHub Codespaces, etc.) and wants a powerful, fast, and scriptable command-line tool to manage these environments for local development, testing, and automation.
*   **Secondary Persona: The Platform Engineer ("Ops")**: A DevOps or platform engineer responsible for creating and managing standardized development environments for their teams using `devcontainer.json`. They need a reliable tool to test, validate, and orchestrate these environments from the command line and in CI/CD pipelines.

### **1.3. Why `reactor`?**

The Dev Container ecosystem is heavily optimized for GUI-based workflows within IDEs. `reactor` fills a significant gap by providing a first-class, standalone CLI that is:
*   **Faster:** Uses intelligent caching and optimized workflows to build and start environments faster than standard tools.
*   **More Powerful:** Extends the Dev Container standard with unique features like account-based secret management and multi-container workspace orchestration.
*   **More Ergonomic:** Provides a clean, intuitive CLI designed for power users who live in the terminal.

### **1.4. The CLI: A User-Centric Overview**

**Key Technical Features & Concepts**

*   **Full Dev Container Specification Support:** `reactor` is a fully compliant execution engine for any project using a standard `.devcontainer/devcontainer.json` file. It provides a familiar, ergonomic CLI that emulates the official `devcontainer` command structure, making adoption seamless.
*   **Multi-Container Workspace Orchestration:** Go beyond a single container. A `reactor-workspace.yml` file allows you to define a collection of independent dev container projects and manage their entire lifecycle (`up`, `down`, `exec`) as a single unit—perfect for microservice development.
*   **Environment Isolation:** Uses Docker to create hermetic, containerized environments on demand.
*   **State Management:** Implements a robust, account-based isolation model for credentials and secrets that extends the dev container standard.
*   **Performance:** Uses intelligent caching and optimized workflows to build and start environments faster than standard tools.

The `reactor` CLI is designed to be powerful yet intuitive, aligning with the conventions of the Dev Container ecosystem while providing unique extensions.

**Core Lifecycle Commands**
*   `reactor up`: The primary command. Builds (if needed) and starts the dev container defined in the current directory, dropping the user into an interactive shell.
    *   `--account <name>`: Activates account-based secret mounting for the session.
    *   `--rebuild`: Forces a rebuild of the container image before starting.
*   `reactor down`: Stops and removes the dev container and any related resources for the current project.
*   `reactor exec <command...>`: Executes a command inside the running dev container for the current project.
*   `reactor build`: Builds or rebuilds the dev container image without starting it.

**Environment Management Commands**
*   `reactor list`: Lists all `reactor`-managed dev containers on your system, showing their status and project directory.
*   `reactor connect <name>`: Connects your terminal to an already-running dev container.
*   `reactor init`: Creates a starter `.devcontainer/devcontainer.json` in the current directory to get a new project started quickly.

**Workspace Commands (Milestone 4)**
*   `reactor workspace up`: Brings up an entire multi-container workspace defined in `reactor-workspace.yml`.
*   `reactor workspace down`: Shuts down an entire workspace.
*   `reactor workspace list`: Lists defined workspaces.

## **2. The Roadmap: How We Get There**

This new strategy represents a **clean break** from the previous `.reactor.conf`-based system. All environment configuration will now be defined in `devcontainer.json`. The `.reactor.conf` file will only be used for the optional, reactor-specific `account` setting for credential isolation.

The project will be implemented via a series of milestones to refactor `reactor` into a first-class Dev Container tool.

### **Milestone 1: Foundational `devcontainer.json` Support (The "Embrace" Milestone)**

This is the core refactoring effort to make `reactor` a native Dev Container tool.

*   **Task 1.1: New Configuration Parser**
    *   **Action:** Refactor `pkg/config` to find and parse `.devcontainer/devcontainer.json`.
    *   **Details:** This requires replacing the YAML parser with a JSONC-compliant parser and creating new Go structs that map to the Dev Container specification.
*   **Task 1.2: Implement Core Lifecycle CLI**
    *   **Action:** Implement the new, `devcontainer-cli`-aligned command structure: **`reactor up`**, **`reactor down`**, **`reactor exec`**, and **`reactor build`**. The old `run` command will be removed.
*   **Task 1.3: Basic Environment Provisioning**
    *   **Action:** Refactor the provisioning logic to run a container based on the `image` property from the parsed `devcontainer.json`.
*   **Task 1.4: Implement Core Dev Container Features**
    *   **Action:** Add support for the `forwardPorts` and `remoteUser` properties within the `up` command.

### **Milestone 2: Advanced Dev Container Features**

This milestone achieves feature parity with standard Dev Container tooling.

*   **Task 2.1: Build from Dockerfile**
    *   **Action:** Enhance the `build` and `up` commands to execute a `docker build` when the `build.dockerfile` property is present.
*   **Task 2.2: Lifecycle Hooks**
    *   **Action:** Implement support for the `postCreateCommand` lifecycle hook.

### **Milestone 3: `reactor`-Specific Extensions (The "Extend" Milestone)**

This milestone introduces `reactor`'s unique value-add on top of the standard, using the official `customizations` property in `devcontainer.json`.

*   **Task 3.1: Implement Account-Based Secret Management**
    *   **Action:** `reactor` will look for an `account` key within the `customizations.reactor` block of `devcontainer.json`.
    *   **Details:** If `account` is specified, `reactor` will automatically mount the corresponding credential sets from `~/.reactor/{account}/` into the container. This eliminates the need for a separate `.reactor.conf` file.
*   **Task 3.2: Implement Optional AI Entrypoint**
    *   **Action:** Support a `defaultCommand` key within `customizations.reactor` to specify a command (like `claude` or `gemini`) to execute on startup instead of a shell.

### **Milestone 4: Multi-Container Workspaces (The "Differentiate" Milestone)**

This milestone provides a powerful solution for a major gap in the current ecosystem.

*   **Task 4.1: Design `reactor-workspace.yml`**
    *   **Action:** Design and implement the `reactor-workspace.yml` file format, which will allow a user to define a collection of interdependent dev containers.
*   **Task 4.2: Implement Workspace Orchestration**
    *   **Action:** Create a new **`reactor workspace`** command group (`up`, `down`, `list`) to manage the entire collection of containers as a single logical unit.

### **Milestone 5: User Onboarding & Templates (The "Polish" Milestone)**

This milestone focuses on creating a seamless and intuitive first-run experience for new users.

*   **Task 5.1: Implement `init` Templates**
    *   **Action:** Enhance the `reactor init` command with a `--template` flag to generate complete, runnable "hello world" projects for different languages (Go, Python, Node.js).
    *   **Details:** This includes generating a `devcontainer.json`, a `Dockerfile`, and sample application code, allowing a user to go from an empty directory to a running application with just two commands.
