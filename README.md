# 🚀 reactor

**A high-performance, developer-focused command-line interface (CLI) for the Dev Container standard.**

`reactor` provides the fastest and most ergonomic terminal-based experience for creating, managing, and connecting to standardized development environments.

---

## 📦 Installation

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform:

```bash
# Linux AMD64
curl -L https://github.com/dyluth/reactor/releases/latest/download/reactor-linux-amd64 -o reactor
chmod +x reactor
sudo mv reactor /usr/local/bin/

# Linux ARM64
curl -L https://github.com/dyluth/reactor/releases/latest/download/reactor-linux-arm64 -o reactor
chmod +x reactor
sudo mv reactor /usr/local/bin/

# macOS Intel
curl -L https://github.com/dyluth/reactor/releases/latest/download/reactor-darwin-amd64 -o reactor
chmod +x reactor
sudo mv reactor /usr/local/bin/

# macOS Apple Silicon
curl -L https://github.com/dyluth/reactor/releases/latest/download/reactor-darwin-arm64 -o reactor
chmod +x reactor
sudo mv reactor /usr/local/bin/
```

### Option 2: Build from Source

**Prerequisites:** `go` 1.22+ must be installed.

```bash
git clone https://github.com/dyluth/reactor.git
cd reactor
make build
sudo cp ./build/reactor /usr/local/bin/
```

## ⚡ Quickstart

**Prerequisites:** `docker` must be installed.

### Option 1: Create a New Go Project

Get from an empty directory to a running, containerized Go development environment in 4 commands.

```bash
# 1. Create a new project directory
mkdir my-go-app && cd my-go-app

# 2. Initialize a sample Go project and dev container
# This creates a devcontainer.json and sample code files
reactor config init --template go

# 3. Build and start your new dev environment!
reactor up

# 4. You're now inside the container! Try the sample web server:
go run main.go
# Visit http://localhost:8080 to see "Hello, World from your Reactor Go environment!"
```

### Option 2: Add Reactor to an Existing Project

Add containerized development to your existing Go project in 3 commands.

```bash
# 1. Navigate to your existing project directory
cd my-existing-go-project

# 2. Initialize reactor dev container configuration
# This creates .devcontainer/devcontainer.json only (preserves your existing code)
reactor config init

# 3. Build and start your containerized dev environment!
reactor up

# 4. You're now inside the container with your existing project mounted at /workspace
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
| `reactor sessions list` | List all `reactor`-managed dev containers on your system. |
| `reactor config init` | Create a new dev container configuration in the current directory. |
| `reactor config init --template <name>` | Generate a complete project from template (go, python, node). |

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