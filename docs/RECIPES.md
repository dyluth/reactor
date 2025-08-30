# Reactor Recipes

Common workflows and practical examples for using Reactor effectively.

## Quick Start Recipes

### New Project Setup
```bash
cd myproject
reactor config init         # Create .reactor.conf
# Edit .reactor.conf as needed
reactor run                 # Start AI session
```

### Evaluating a New AI Tool
```bash
reactor run --discovery-mode    # Clean environment
# Use the AI tool, trigger setup
exit                           # Exit container
reactor diff                   # See what was created
reactor sessions clean         # Clean up test containers
```

## Development Workflows

### Web Development with Port Forwarding
```bash
# Python web server
reactor run --image python -p 8000:8000

# Node.js development
reactor run --image base -p 3000:3000 -p 8080:8080

# Multiple services
reactor run -p 8000:8000 -p 8080:8080 -p 5432:5432
```

### Multi-Account Setup
```bash
# Personal projects
echo "provider: claude" > .reactor.conf
echo "account: personal" >> .reactor.conf

# Work projects  
echo "provider: claude" > .reactor.conf
echo "account: work" >> .reactor.conf
```

## Advanced Use Cases

### Host Docker Access
```bash
# WARNING: Security risk - only use with trusted images
reactor run --docker-host-integration

# Inside container, you now have access to host Docker daemon
docker ps                     # See host containers
docker run hello-world        # Create containers on host
```

### Custom Image with Specific Tools
```yaml
# .reactor.conf
provider: custom
account: default
image: myregistry/my-dev-image:latest
```

### Provider Switching
```bash
# Switch from Claude to Gemini for same project
sed -i 's/provider: claude/provider: gemini/' .reactor.conf
reactor run  # Now uses Gemini with separate state
```

## Container Management

### Container Lifecycle
```bash
# See all your containers
reactor sessions list

# Clean up all reactor containers (recommended)
reactor sessions clean

# Alternative: Clean up discovery containers with Docker
docker ps -a --filter name=reactor-discovery --format "table {{.Names}}" | tail -n +2 | xargs docker rm -f

# Nuclear option: Force remove all reactor containers manually
docker ps -a --filter name=reactor --format "table {{.Names}}" | tail -n +2 | xargs docker rm -f
```

### State Management
```bash
# Backup account state
tar -czf backup.tar.gz ~/.reactor/personal/

# Move project to different account
mv ~/.reactor/personal/abc12345 ~/.reactor/work/

# Clean start (remove all state)
rm -rf ~/.reactor/
```

## Team Collaboration

### Shared Project Configuration
```yaml
# .reactor.conf (commit to git)
provider: claude
account: team
image: python
```

### Environment Consistency
```bash
# Team member A
reactor run --account team

# Team member B (same config)
reactor run --account team
```

## Performance Tips

### Fast Container Recovery
- Reactor reuses existing containers automatically
- First run: ~60-90 seconds (image pull + creation)
- Subsequent runs: ~3 seconds (container recovery)

### Image Selection
- `base`: Smallest, fastest startup
- `python`: Use only if you need Python tools
- `go`: Use only if you need Go tools
- Custom: Pre-install only what you need