# Account Management with `reactor accounts`

Manage and view account-based configuration isolation.

## Commands

```bash
# List all configured accounts
reactor accounts list

# Show current account
reactor accounts show

# Set active account
reactor accounts set work
```

## Account Directory Structure

Each account gets isolated state under `~/.reactor/`:

```
~/.reactor/
├── personal/
│   └── abc123def/          # project hash
│       ├── claude/         # mounted to /home/claude/.claude
│       └── gemini/         # mounted to /home/claude/.gemini
└── work/
    └── xyz789abc/
        ├── claude/
        └── openai/
```

## Use Cases

**Personal vs Work Separation**:
```yaml
# Personal projects (.reactor.conf)
provider: claude
account: personal

# Work projects (.reactor.conf)  
provider: claude
account: work
```

**Multiple AI Providers**:
```yaml
# Project A
provider: claude
account: main

# Project B
provider: gemini  
account: main
```

## Benefits

- **Credential Isolation**: Separate API keys and auth tokens
- **Project Isolation**: Different projects can't interfere
- **Provider Switching**: Easy switching between AI tools
- **Team Consistency**: Shared account configurations