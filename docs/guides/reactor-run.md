# Running AI Tools with `reactor run`

Start an interactive AI tool session in an isolated container.

## Basic Usage

```bash
# Run with default settings (uses project config)
reactor run

# Run with specific image
reactor run --image python

# Run with port forwarding (recommended: use -p shorthand)
reactor run -p 8080:8080 -p 3000:3000

# Run in discovery mode (no mounts, for testing new tools)
reactor run --discovery-mode

# Run with Docker host access (security risk)
reactor run --docker-host-integration
```

## Key Features

- **Container Recovery**: Automatically reuses existing containers
- **Account Isolation**: Separate configs per account/project
- **Port Forwarding**: Forward multiple ports with `-p HOST:CONTAINER` (or `--port HOST:CONTAINER`)
- **Discovery Mode**: Clean environment for evaluating new AI tools

## Security Notes

- `--docker-host-integration` grants full Docker daemon access
- Only use with trusted images and understand the security implications
- Discovery mode creates temporary containers with no state persistence

## Common Patterns

```bash
# Development workflow
reactor run --image python -p 8000:8000     # Web dev with port forwarding
reactor run --account work                   # Use work account config

# Tool evaluation
reactor run --discovery-mode             # Test new tool safely
reactor diff                            # See what files were created
```