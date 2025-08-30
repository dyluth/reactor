# Managing Sessions with `reactor sessions`

List and manage active Reactor container sessions.

## Commands

```bash
# List all reactor containers
reactor sessions list

# Attach to a container (auto-detect current project)
reactor sessions attach

# Attach to specific container by name
reactor sessions attach reactor-account-project-hash

# Clean up all reactor containers
reactor sessions clean
```

## Container States

- **Running**: Active container with live session
- **Stopped**: Container exists but not running (can be recovered)
- **Discovery**: Temporary containers from `--discovery-mode`

## Output Format

```
NAME                                    STATUS    IMAGE           CREATED
reactor-cam-myproject-abc123           running   ghcr.io/...     2 hours ago
reactor-discovery-cam-test-def456      stopped   ghcr.io/...     1 hour ago
```

## Management Tasks

```bash
# See all your containers across projects
reactor sessions list

# Attach to current project's container
reactor sessions attach

# Find containers for specific project
reactor sessions list | grep myproject

# Clean up all reactor containers (recommended)
reactor sessions clean

# Alternative: Clean up containers manually with Docker commands
docker stop <container-name>
docker rm <container-name>
```

## Container Recovery

Reactor automatically recovers stopped containers when you run `reactor run`. The sessions command helps you:

- Track which projects have active containers
- Attach to existing containers without running `reactor run`
- Monitor resource usage across projects