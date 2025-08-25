# LESSONS-LEARNED.md

## Project Overview

**Core Problem**: Create a containerized CLI tool that provides seamless multi-account isolation, persistent session management, and transparent Docker integration for development workflows. The system must handle container lifecycle management, cross-platform compatibility, and provide a zero-configuration user experience while maintaining complete isolation between different user accounts and projects.

**Success Criteria for Replacement System**:
- Container startup time: <3 seconds for existing containers, <10 seconds for new builds
- Support 10+ concurrent isolated accounts without conflicts
- 99%+ session recovery success rate after container stops/restarts
- Cross-platform compatibility (macOS ARM64/AMD64, Linux ARM64/AMD64)
- Zero-configuration experience for 90%+ of use cases
- Docker host access capabilities with full Docker API access (via socket mounting)

## Technical Architecture Foundations

### Dependency Injection Container Pattern

**Problem**: Managing complex interdependencies between Docker management, authentication, configuration, and lifecycle components while maintaining testability and modularity.

**Solution**: Implemented a centralized dependency injection container that initializes all components with their dependencies in correct order. This pattern proved essential for:
- Clean separation of concerns across components
- Easy mocking and testing of individual components
- Graceful error handling during initialization
- Consistent logging and configuration propagation

**Key Principle**: All external integrations (Docker API, filesystem, authentication systems) should be abstracted behind interfaces and injected rather than directly instantiated.

### Multi-Account Isolation Architecture

**Design Intention**: Complete isolation between different Claude accounts while sharing the underlying container infrastructure.

**Approach**:
- **Directory Structure**: Each account gets isolated directories (`~/.claude-reactor/.{account}-claude/`)
- **Container Naming**: Deterministic naming scheme: `claude-reactor-{variant}-{architecture}-{project-hash}-{account}`
- **Configuration Files**: Account-specific config files prevent cross-account authentication leakage
- **Session Directories**: Isolated session storage prevents conversation/project mixing

**Critical Insight**: The project hash in container names enables multiple projects to coexist safely while the account suffix ensures authentication isolation.

## Docker Integration Deep Dive

### Docker Host Access Pattern

**Problem**: Enable Docker API access from within containers while maintaining security and cross-platform compatibility.

**Solution**: Docker socket mounting with proper permission handling to grant containers host-level Docker access.

**Dockerfile Pattern**:
```dockerfile
# In Dockerfile - ensure docker group exists and has correct GID
ARG DOCKER_GID=999
RUN groupadd -g ${DOCKER_GID} docker || groupmod -g ${DOCKER_GID} docker
RUN usermod -aG docker claude

# Mount pattern in container runtime:
# -v /var/run/docker.sock:/var/run/docker.sock
# --group-add docker
```

**Permission Strategy**:
- Detect host Docker socket GID at runtime
- Pass as build argument to ensure container user has correct permissions
- Use `--group-add docker` flag during container execution
- Validate Docker API connectivity before proceeding with operations

**Security Considerations**:
- Docker socket access grants full Docker API privileges - equivalent to root access on the host
- Container effectively has host-level Docker daemon access, not isolated Docker-in-Docker
- Container must run with appropriate security context
- Consider using Docker API over TCP with TLS for production deployments
- Implement timeout contexts (30s default) for all Docker operations

### Multi-Stage Dockerfile Optimization

**Guiding Principles**:
- **Stage Purpose**: Each stage should serve a distinct purpose (build, runtime, language-specific)
- **Layer Caching**: Structure commands to maximize Docker layer cache hits
- **Size Optimization**: Use distroless or Alpine base images for final stages
- **Build Context**: Minimize build context size with strategic .dockerignore

**Pattern**:
```dockerfile
# Base stage - common dependencies
FROM ubuntu:22.04 as base
# ... install common tools

# Language-specific stages build on base
FROM base as go
# ... Go toolchain

FROM base as full  
# ... multiple language toolchains

# Final runtime selection via --target flag
```

## Container Lifecycle Management

### Container Naming & Hashing Strategy

**Problem**: Ensure deterministic container identification while supporting multi-project, multi-account scenarios.

**Strategy**:
- **Project Hash**: SHA-256 of absolute project path (first 8 chars)
- **Naming Pattern**: `claude-reactor-{variant}-{arch}-{project-hash}-{account}`
- **Deterministic**: Same project + account always generates same container name
- **Collision Avoidance**: Hash-based naming prevents accidental reuse across projects

**Benefits**:
- Enables automatic container discovery
- Supports multiple projects simultaneously
- Account isolation without complex lookup mechanisms

### Container Recovery Logic Evolution

**Problem**: Reliably reconnect to existing containers across different scenarios (config deletion, directory changes, container restarts).

**Evolution of Approaches**:
1. **ID-based Recovery**: Store container ID in config → fails when config deleted
2. **Name-based Recovery**: Always check for existing containers by name → robust recovery
3. **Hybrid Approach**: Try ID first, fallback to name-based discovery

**Final Pattern**:
```
1. Check saved ContainerID (if exists and healthy) → reconnect
2. Check for existing container by name:
   - If running → reconnect and update config
   - If stopped → restart and reconnect
   - If corrupted → remove and create new
3. Create new container only if none found
```

### Session Persistence Implementation

**Key Insight**: Always create and mount session directories, don't conditionally mount based on existence.

**Approach**:
- **Named Accounts**: Mount `~/.claude-reactor/.{account}-claude/` → `/home/claude/.claude` 
the default account is `default`
- **Directory Creation**: Always ensure directories exist before mounting
- **Mount Strategy**: Bind mounts (not volumes) for direct file system access

## Build System & Cross-Platform Considerations

### Cross-Compilation Strategy

**CGO Considerations**:
- Docker SDK requires CGO for some operations
- Set `CGO_ENABLED=0` for pure Go binaries when possible
- Use build constraints to handle platform-specific code
- Consider static linking for Linux distributions

**Platform Matrix**:
- Primary: darwin/arm64 (M1 Macs)
- Secondary: darwin/amd64, linux/arm64, linux/amd64
- Build all platforms in CI, distribute via GitHub releases

### Version Injection & Metadata Handling

**Approach**: Inject build-time metadata using Go linker flags.

**Pattern**:
```makefile
LDFLAGS = -X main.Version=$(VERSION) \
          -X main.GitCommit=$(GIT_COMMIT) \
          -X main.BuildDate=$(BUILD_DATE)

go build -ldflags "$(LDFLAGS)" ...
```

**Metadata Strategy**:
- Version from Git tags or "dev" for development builds
- Git commit for troubleshooting and support
- Build date for cache invalidation and debugging
- Embed in debug commands and help output

### Build Caching Principles

**Go Module Caching**:
- Download dependencies in separate Docker stage
- Copy go.mod/go.sum before source code
- Use Go build cache (`GOCACHE`) in CI environments

**Docker Layer Optimization**:
- Order Dockerfile commands by change frequency
- Combine related RUN commands to reduce layers
- Use .dockerignore to exclude unnecessary files from build context

## Platform-Specific Challenges

### macOS Binary Signing & Distribution

**Key Challenges**:
- Unsigned binaries trigger Gatekeeper warnings
- "Killed: 9" errors indicate code signing issues
- Notarization required for seamless distribution

**Workaround Strategies**:
- Remove quarantine attributes: `xattr -d com.apple.quarantine`
- User education about security warnings
- Consider code signing certificates for production distribution
- Provide installation scripts that handle common macOS issues

**Lesson**: macOS security model requires either proper code signing or user education about security overrides.

### Architecture Detection & Compatibility

**Detection Strategy**:
- Use `runtime.GOARCH` and `runtime.GOOS` for build-time detection
- Implement runtime architecture detection for Docker platform selection
- Support both native and emulated architectures (Rosetta 2)

**Docker Platform Selection**:
- Default to native architecture when available
- Fall back to `linux/amd64` for maximum compatibility
- Use `--platform` flag for explicit platform targeting

### File System Permission Handling

**Cross-Platform Considerations**:
- Linux: UID/GID mapping for bind mounts
- macOS: Different permission model in Docker Desktop
- Windows: Path separator and volume mount challenges

**Strategy**:
- Detect host OS and adjust mount strategies accordingly
- Use consistent UID/GID in containers (1000:1000)
- Handle path normalization across platforms

## Testing Strategies

### Docker Integration Testing

**Approach**:
- **Unit Tests**: Mock Docker SDK for business logic testing
- **Integration Tests**: Test against real Docker daemon with cleanup
- **Container Lifecycle Tests**: Full container creation, start, stop, removal cycles
- **Mount Validation**: Verify proper file system mounting and permissions

**Test Environment Considerations**:
- Ensure Docker daemon is available in CI
- Clean up test containers/images to prevent resource leaks
- Use deterministic names for test containers
- Test both positive and negative scenarios (network failures, permission issues)

### Multi-Account Isolation Testing

**Test Scenarios**:
- Concurrent account usage
- Account switching within same project
- Configuration isolation verification
- Session data separation validation

**Recommended Test Structure**:
- Create isolated temp directories for each test
- Generate unique account names to prevent test interference
- Verify no cross-contamination between account configurations
- Test recovery scenarios (config deletion, container removal)

## Anti-Patterns & Design Failures

### Failed Approaches

**Container ID Storage Dependency**:
- **What Failed**: Relying solely on stored container IDs for recovery
- **Why**: Config files get deleted, corrupted, or lost
- **Lesson**: Always implement name-based container discovery as fallback

**Conditional Directory Mounting**:
- **What Failed**: Only mounting directories if they exist on host
- **Why**: Session data gets created inside container and lost on removal
- **Lesson**: Always create and mount session directories proactively

**Single Recovery Strategy**:
- **What Failed**: Having only one container recovery approach
- **Why**: Different failure modes require different recovery strategies
- **Lesson**: Implement layered recovery with graceful degradation

### Performance Issues Discovered

**Docker API Timeouts**:
- **Issue**: Default Docker API calls could hang indefinitely
- **Solution**: Always use context.WithTimeout for Docker operations
- **Recommendation**: 30s for container operations, 60s for image builds

**Container Name Conflicts**:
- **Issue**: Simple naming schemes caused container conflicts
- **Solution**: Include project hash in container names
- **Lesson**: Design naming schemes for collision avoidance from the start

### Architectural Decisions Reversed

**Monolithic Configuration**:
- **Original**: Single configuration file for all settings
- **Problem**: Account isolation and concurrent usage issues
- **Revised**: Account-specific configuration files with inheritance
- **Lesson**: Design for multi-tenancy from the beginning

**Volume vs Bind Mounts**:
- **Original**: Docker volumes for session persistence
- **Problem**: Difficult to access/debug session data from host
- **Revised**: Bind mounts for direct filesystem access
- **Lesson**: Choose mount strategy based on operational requirements

## External Dependencies & Library Choices

### Docker SDK Selection

**Choice**: Official Docker Go SDK (`github.com/docker/docker`)
**Rationale**:
- Full API coverage and official support
- Active maintenance and security updates
- Comprehensive examples and documentation
- Better than shelling out to docker CLI for complex operations

### CLI Framework

**Choice**: Cobra CLI framework
**Rationale**:
- Industry standard for Go CLI applications
- Excellent subcommand support and help generation
- Built-in shell completion capabilities
- Extensive ecosystem and community support

### Configuration Management

**Approach**: Custom solution rather than Viper
**Rationale**:
- Simple YAML structure sufficient for use case
- Avoid dependency bloat for basic configuration needs
- Custom validation logic easier to implement
- Direct control over configuration persistence behavior

## Replacement System Recommendations

### Architecture Improvements

1. **Event-Driven Architecture**: Consider using channels/event system for container lifecycle events
2. **Plugin System**: Design extensible architecture for custom container variants
3. **Configuration Validation**: Implement comprehensive config validation with user-friendly error messages
4. **Observability**: Built-in metrics, tracing, and structured logging from day one

### Security Enhancements

1. **Principle of Least Privilege**: Investigate alternatives to full Docker socket access
2. **Container Security**: Implement security profiles and resource limits
3. **Input Validation**: Comprehensive validation of user inputs and file paths
4. **Audit Logging**: Log all container operations for security monitoring

### Operational Excellence

1. **Health Checks**: Implement comprehensive health checking for all components
2. **Resource Management**: Monitor and limit resource usage per account/project
3. **Graceful Shutdown**: Proper cleanup of resources on application termination
4. **Configuration Migration**: Version configuration files and provide migration paths

### Development Experience

1. **Error Messages**: Invest heavily in clear, actionable error messages
2. **Debug Mode**: Comprehensive debug output for troubleshooting
3. **Documentation**: Auto-generated CLI documentation and examples
4. **Testing**: Achieve >90% test coverage with focus on integration tests

This document captures the essential technical lessons for rebuilding this system with improved design while avoiding the pitfalls and anti-patterns discovered during the original development.