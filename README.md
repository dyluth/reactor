# The Reactor Ecosystem

This repository contains the source code for the Reactor ecosystem, a suite of tools designed for modern, AI-driven software development.

## Projects

This is a multi-project monorepo containing the following tools:

### 1. 🚀 `reactor` - A Command-Line Interface for Dev Containers

`reactor` is a high-performance, developer-focused CLI for the Dev Container standard. It provides the fastest and most ergonomic terminal-based experience for managing standardized development environments.

*   **[➡️ Full Design Document & Roadmap](./docs/features/reactor/README.md)**

### 2. 🤖 `reactor-fabric` - A Container-Native AI Agent Orchestrator

`reactor-fabric` is a standalone engine designed to manage a fleet of autonomous, container-native agents to automate complex software development and operational tasks.

*   **[➡️ Full Design Document](./docs/features/fabric/README.md)**

## Development

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
