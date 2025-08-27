# Reactor Core Concepts

Understanding how Reactor manages isolated AI tool environments.

## Architecture Overview

Reactor creates isolated, containerized environments for AI CLI tools with three key components:

1. **Account-based Isolation**: Separate configurations per user account
2. **Project-based State**: Each project gets its own persistent storage
3. **Container Recovery**: Reuse existing containers for fast startup

## Account & Project Isolation

### Directory Structure

```
~/.reactor/
├── <account>/              # e.g., personal, work, team
│   └── <project-hash>/     # first 8 chars of project path hash
│       ├── claude/         # mounted to /home/claude/.claude
│       ├── gemini/         # mounted to /home/claude/.gemini
│       └── openai/         # mounted to /home/claude/.openai
```

### How It Works

1. **Account Separation**: Different accounts (personal, work) get completely separate directories
2. **Project Hashing**: Each project directory gets a unique hash to prevent conflicts
3. **Provider Mounting**: AI tools see their expected config locations inside containers

## Container Lifecycle

### Container Naming

Containers use deterministic names based on account and project:
```
reactor-<account>-<folder>-<hash>
# Example: reactor-cam-myproject-abc12345
```

### Recovery Strategy

1. **Check Running**: Look for existing running container
2. **Check Stopped**: If stopped container exists, restart it  
3. **Create New**: Only create new container if none exists

This means your AI tool session persists across `reactor run` commands.

## Configuration Resolution

### Project Config (`.reactor.conf`)
```yaml
provider: claude    # which AI tool
account: default    # isolation account  
image: python       # container image
danger: false       # security settings
```

### Built-in Providers

| Provider | Default Image | Mounts |
|----------|---------------|--------|
| `claude` | `base` | `claude/` → `/home/claude/.claude` |
| `gemini` | `base` | `gemini/` → `/home/claude/.gemini` |
| `custom` | user-defined | user-defined |

## Security Model

### Default Security
- Containers run as non-root `claude` user
- No host filesystem access except mounted state directories
- Network isolation via Docker bridge networking

### Optional Risk Features
- `--docker-host-integration`: Mounts Docker socket (full Docker access)
- `danger: true`: Future feature for additional permissions

## Discovery Mode

Special mode for evaluating new AI tools safely:

1. **No Mounts**: Clean environment with no persistent state
2. **Unique Names**: `reactor-discovery-*` naming prevents conflicts
3. **Diff Capability**: Use `reactor diff` to see what files were created
4. **Cleanup**: Discovery containers are temporary and can be cleaned up