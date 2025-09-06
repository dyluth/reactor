# 🚀 reactor

**A high-performance, developer-focused command-line interface (CLI) for the Dev Container standard.**

`reactor` provides the fastest and most ergonomic terminal-based experience for creating, managing, and connecting to standardized development environments.

---

## ⚡ Quickstart

Get from an empty directory to a running, containerized Go development environment in 3 commands.

**Prerequisites:** `docker`, `git`, and `go` must be installed.

```bash
# 1. Create a new project directory
mkdir my-go-app && cd my-go-app

# 2. Initialize a sample Go project and dev container
# This creates a devcontainer.json, a Dockerfile, and a "Hello World" main.go
reactor init --template go

# 3. Build and start your new dev environment!
reactor up
```

After `reactor up` completes, you will be inside a containerized shell. Your project directory is mounted at `/workspace`, and you can start coding.

## ✨ Key Features

*   **Dev Container Native:** Full support for the `devcontainer.json` specification. Works with any existing Dev Container project.
*   **Multi-Container Workspaces:** Define and manage a full stack of microservices with a single `reactor-workspace.yml` file and the `reactor workspace` commands.
*   **Account-Based Credentials:** Automatically and securely mount the correct credentials for different AI tools and cloud providers using `reactor`'s account management features.
*   **High-Performance:** Built in Go with a focus on speed and efficiency.

## 📖 Usage

### Single Container Commands

These commands operate on the `devcontainer.json` in the current directory.

| Command | Description |
| :--- | :--- |
| `reactor up` | Build (if needed) and start your dev container. |
| `reactor down` | Stop and remove your dev container. |
| `reactor build` | Build or rebuild the dev container image without starting it. |
| `reactor exec -- <cmd>` | Execute a command inside the running dev container. |
| `reactor list` | List all `reactor`-managed dev containers on your system. |
| `reactor init` | Create a new dev container configuration in the current directory. |

### Workspace Commands

These commands operate on a `reactor-workspace.yml` file in the current directory.

| Command | Description |
| :--- | :--- |
| `reactor workspace up` | Start all services defined in your workspace. |
| `reactor workspace down` | Stop and remove all services in your workspace. |
| `reactor workspace list` | List the status of all services in your workspace. |
| `reactor workspace exec <svc> -- <cmd>` | Execute a command in a specific service container. |

---

## 💻 Development

This repository is a monorepo containing the source code for the `reactor` CLI and the `reactor-fabric` orchestration engine.

### Getting Started

```bash
# Clone and set up
git clone https://github.com/dyluth/reactor.git
cd reactor

# See all available targets and usage examples
make

# Quick validation (fmt + lint + test)
make check

# Full CI validation (recommended before commits)
make ci

# Build the reactor binary
make build
```

### Build System

The `Makefile` provides a comprehensive set of targets for development and testing.

*   `make ci`: 🎯 Run the full CI pipeline.
*   `make check`: ⚡ Run a quick validation suite.
*   `make build`: 🔨 Build the `reactor` binary.
*   `make test-isolated`: 🧪 Run all Go tests with isolation.
*   `make docker-images`: 🐳 Build all official container images.

## License

This project is licensed under the MIT License.