# Managing Sessions with `reactor sessions`

List and manage active Reactor container sessions.

## Commands

```bash
# List all reactor containers
reactor sessions list

# List with detailed status
reactor sessions list --all

# Clean up stopped containers
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

# Clean up old discovery containers
reactor sessions clean

# Find containers for specific project
reactor sessions list | grep myproject
```

## Container Recovery

Reactor automatically recovers stopped containers when you run `reactor run`. The sessions command helps you:

- Track which projects have active containers
- Clean up discovery containers after evaluation
- Monitor resource usage across projects