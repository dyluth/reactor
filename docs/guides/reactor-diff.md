# Discovering File Changes with `reactor diff`

Show filesystem changes made by AI tools in discovery mode containers.

## Usage

```bash
# Show changes from the most recent discovery container
reactor diff

# Show changes from a specific container
reactor diff --container reactor-discovery-cam-myproject-abc123
```

## Output Format

Changes are shown in git-style format:
- `A` - Added files/directories
- `C` - Changed files
- `D` - Deleted files

```
A /home/claude/.claude/config.json
A /home/claude/.claude/sessions/
C /home/claude/.bashrc
```

## Workflow

1. Run tool in discovery mode: `reactor run --discovery-mode`
2. Use the AI tool to trigger its setup/configuration
3. Exit the container
4. Run `reactor diff` to see what was created
5. Use this information to configure proper mounts

## Common Use Cases

- **New Tool Evaluation**: See exactly what config files a new AI tool creates
- **Mount Strategy**: Determine which directories need persistent storage
- **Security Audit**: Verify tools only create expected files
- **Debugging**: Understand why a tool isn't working in normal mode