# Troubleshooting Guide

Common issues and solutions for Reactor.

## Container Issues

### "Docker daemon is not accessible"

**Problem**: `reactor run` fails with Docker connection error.

**Solutions**:
```bash
# Check Docker is running
docker ps

# Check Docker permissions (Linux)
sudo usermod -aG docker $USER
# Log out and back in

# Check Docker Desktop (Mac/Windows)
# Ensure Docker Desktop is running
```

### Container Won't Start

**Problem**: Container creation succeeds but won't start.

**Diagnosis**:
```bash
# Check container logs
docker logs reactor-<account>-<project>-<hash>

# Check Docker daemon logs
docker system events
```

**Common Causes**:
- Image doesn't exist or is corrupted
- Port conflicts (use different ports)
- Insufficient resources (disk space, memory)

### Port Conflicts

**Problem**: Port forwarding fails with "port already in use".

**Solutions**:
```bash
# Find what's using the port
lsof -i :8080
netstat -tulpn | grep 8080

# Use different ports
reactor run -p 8081:8080  # Use 8081 instead of 8080

# Stop conflicting service
docker stop <container-using-port>
```

## Configuration Issues

### "No configuration found"

**Problem**: `reactor run` fails to find `.reactor.conf`.

**Solutions**:
```bash
# Initialize config
reactor config init

# Check current directory
pwd
ls -la .reactor.conf

# Create minimal config
echo "provider: claude" > .reactor.conf
```

### AI Tool Not Working

**Problem**: AI tool command not found in container.

**Diagnosis**:
```bash
# Check what's installed
reactor run --image base
which claude  # or which gemini

# Check image contents
docker run --rm -it ghcr.io/dyluth/reactor/base:latest bash
```

**Solutions**:
- Use correct built-in image (`base`, `python`, `node`, `go`)
- For custom images, ensure AI tools are pre-installed
- Check tool installation in container

## State and Mount Issues

### Lost Configuration

**Problem**: AI tool asks to re-authenticate every time.

**Diagnosis**:
```bash
# Check mount points
reactor run
mount | grep reactor

# Check state directory
ls -la ~/.reactor/<account>/<project-hash>/
```

**Solutions**:
- Ensure `.reactor.conf` has correct provider
- Check account/project isolation isn't causing separation
- Verify state directories exist and are writable

### Permission Errors

**Problem**: "Permission denied" errors in container.

**Solutions**:
```bash
# Check state directory ownership
ls -la ~/.reactor/
sudo chown -R $USER:$USER ~/.reactor/

# Check Docker socket permissions (for --docker-host-integration)
ls -la /var/run/docker.sock
sudo chmod 666 /var/run/docker.sock  # Temporary fix
```

## Performance Issues

### Slow Container Startup

**Problem**: `reactor run` takes very long.

**Diagnosis**:
```bash
# Check if pulling image
docker images | grep reactor

# Check container status
reactor sessions list
```

**Solutions**:
- First run always slower (image pull)
- Use `--verbose` flag to see what's happening
- Pre-pull images: `docker pull ghcr.io/dyluth/reactor/base:latest`

### High Resource Usage

**Problem**: Container using too much CPU/memory.

**Diagnosis**:
```bash
# Monitor container resources
docker stats

# Check running processes in container
docker exec -it reactor-<name> ps aux
```

**Solutions**:
- Limit container resources in Docker
- Choose minimal image (`base` vs `python`)
- Clean up unused containers with Docker commands

## Discovery Mode Issues

### `reactor diff` Shows No Changes

**Problem**: Discovery mode container shows no filesystem changes.

**Diagnosis**:
```bash
# Check discovery container exists
reactor sessions list | grep discovery

# Manually check container
docker exec -it reactor-discovery-<name> bash
```

**Solutions**:
- Ensure you used `--discovery-mode` flag
- Actually use the AI tool to trigger setup
- Some tools only create files on first real use

## Network Issues

### Cannot Access Forwarded Ports

**Problem**: Port forwarding configured but can't access service.

**Diagnosis**:
```bash
# Check if port is listening in container
docker exec -it <container> netstat -tlnp | grep 8080

# Check if port is bound on host
netstat -tlnp | grep 8080
```

**Solutions**:
- Ensure service is bound to `0.0.0.0` not `127.0.0.1`
- Check firewall settings
- Verify port mapping syntax: `-p 8080:8080`

## Emergency Recovery

### Complete Reset

If everything breaks:
```bash
# Clean up all reactor containers (recommended)
reactor sessions clean

# Alternative: Manual cleanup with Docker commands
docker ps --filter name=reactor --format "table {{.Names}}" | tail -n +2 | xargs docker stop
docker ps -a --filter name=reactor --format "table {{.Names}}" | tail -n +2 | xargs docker rm

# Remove all state (WARNING: loses all AI tool configs)
rm -rf ~/.reactor/

# Start fresh
cd your-project
reactor config init
reactor run
```

### Get Help

```bash
# Enable verbose logging
reactor --verbose run

# Check system info
reactor config show
docker version
docker system info
```