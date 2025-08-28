# Reactor

A command-line tool that provides developers with isolated, containerized environments for AI CLI tools like Claude and Gemini.

## Why Reactor?

- **Environment Isolation**: Run AI tools in clean containers without polluting your host system
- **Account Separation**: Keep personal and work AI configurations completely separate
- **Fast Recovery**: Reuse existing containers for sub-3-second startup times
- **Discovery Mode**: Safely evaluate new AI tools to see exactly what they create

## Quick Start

### Prerequisites
- Docker installed and running
- macOS or Linux (amd64/arm64)

### Installation
```bash
# Download and install (replace with actual release URL when available)
curl -L https://github.com/anthropics/reactor/releases/latest/download/reactor-$(uname -s)-$(uname -m) -o reactor
chmod +x reactor
sudo mv reactor /usr/local/bin/
```

### Basic Usage
```bash
# Initialize a project
cd your-project
reactor config init

# Start an AI session (creates container on first run)
reactor run

# Use with port forwarding for web development
reactor run -p 8000:8000

# Evaluate a new AI tool safely
reactor run --discovery-mode
reactor diff  # See what files it created
```

## Core Features

### Account-Based Isolation
Each account gets separate configuration storage:
```yaml
# .reactor.conf
provider: claude    # or gemini, custom
account: personal   # or work, team, etc.
image: python       # base, python, node, go, or custom
```

### Built-in Images
- **base**: Core tools + Claude & Gemini CLI (`ghcr.io/dyluth/reactor/base`)
- **python**: Base + Python development environment (`ghcr.io/dyluth/reactor/python`)
- **node**: Base + Node.js development environment (`ghcr.io/dyluth/reactor/node`)
- **go**: Base + Go development environment (`ghcr.io/dyluth/reactor/go`)
- **custom**: Use any Docker image

### Container Recovery
- **First run**: ~60-90 seconds (pulls image, creates container)
- **Subsequent runs**: ~3 seconds (recovers existing container)

## Documentation

- **[Getting Started](docs/guides/)**: Task-oriented guides for each command
- **[Core Concepts](docs/CORE_CONCEPTS.md)**: Understanding isolation and architecture
- **[Recipes](docs/RECIPES.md)**: Common workflows and practical examples  
- **[Troubleshooting](docs/TROUBLESHOOTING.md)**: Solutions for common issues

### Command Guides
- [`reactor run`](docs/guides/reactor-run.md) - Start AI tool sessions
- [`reactor config`](docs/guides/reactor-config.md) - Manage project configuration
- [`reactor sessions`](docs/guides/reactor-sessions.md) - List and manage containers
- [`reactor diff`](docs/guides/reactor-diff.md) - Discovery mode filesystem changes
- [`reactor accounts`](docs/guides/reactor-accounts.md) - Account management
- [`reactor completion`](docs/guides/reactor-completion.md) - Shell completions

## Common Workflows

### Development Setup
```bash
# Python web development
reactor run --image python -p 8000:8000

# Multi-service development  
reactor run -p 8000:8000 -p 3000:3000 -p 5432:5432
```

### Account Management
```bash
# Personal projects
reactor run --account personal

# Work projects
reactor run --account work
```

### Tool Evaluation
```bash
# Test new AI tool safely
reactor run --discovery-mode
# Use the tool to trigger its setup
exit
reactor diff                # See what was created
reactor sessions clean      # Clean up
```

## Security

- Containers run as non-root user
- No host filesystem access except mounted state directories
- Optional `--docker-host-integration` for Docker access (use with caution)

## Development

### Building Reactor CLI
```bash
# Build from source
git clone https://github.com/anthropics/reactor.git
cd reactor
make build

# Run tests
make test

# Run linter
make lint
```

### Building Container Images
```bash
# Build all images locally
./scripts/build-images.sh

# Build and test all images
./scripts/build-images.sh --test

# Build with official tags
./scripts/build-images.sh --official

# Manual builds (from repo root)
docker build -t reactor/base:local images/base
docker build -t reactor/python:local images/python
docker build -t reactor/node:local images/node
docker build -t reactor/go:local images/go
```

## Contributing

See [PROJECT_CHARTER.md](docs/PROJECT_CHARTER.md) for architecture and contribution guidelines.

## License

[License information to be added]