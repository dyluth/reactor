# Project Configuration with `reactor config`

Initialize and manage project-specific Reactor configurations.

## Commands

```bash
# Initialize config for current project
reactor config init

# Show current configuration
reactor config show

# Show effective configuration (with defaults)
reactor config show --resolved
```

## Configuration File

Creates `.reactor.conf` in your project directory:

```yaml
provider: claude          # claude, gemini, or custom
account: default          # account name for isolation
image: python            # base, python, go, or custom image URL
danger: false            # enable dangerous permissions
```

## Built-in Providers

- **claude**: Mounts `~/.reactor/<account>/<project>/claude` to `/home/claude/.claude`
- **gemini**: Mounts `~/.reactor/<account>/<project>/gemini` to `/home/claude/.gemini`
- **custom**: Define your own mount strategy

## Built-in Images

- **base**: Core tools + AI agents (Claude, Gemini)
- **python**: Base + Python development environment  
- **go**: Base + Go development environment
- **Custom URL**: Any Docker image (must have `claude` user)

## Account Isolation

Accounts separate configurations:
```bash
# Personal projects
provider: claude
account: personal

# Work projects  
provider: claude
account: work
```

Each account gets isolated directories under `~/.reactor/<account>/`.