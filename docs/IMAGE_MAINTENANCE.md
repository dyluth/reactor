# Container Image Maintenance Guide

This document outlines maintenance procedures for official Reactor container images.

## Image Overview

Reactor maintains four official images built in a layered hierarchy:

```
ghcr.io/dyluth/reactor/base     (foundation)
├── ghcr.io/dyluth/reactor/python
├── ghcr.io/dyluth/reactor/node  
└── ghcr.io/dyluth/reactor/go
```

## Local Development Commands

### Building Images Locally

```bash
# From repository root - build all images
docker build -t reactor/base:local images/base
docker build -t reactor/python:local images/python  
docker build -t reactor/node:local images/node
docker build -t reactor/go:local images/go

# Build single image
docker build -t reactor/base:local images/base

# Build with official tags (for local testing)
docker build -t ghcr.io/dyluth/reactor/base:latest images/base
```

### Testing Images Locally

```bash
# Test all images
./images/base/test.sh      # (if run inside base container)
./images/python/test.sh    # (if run inside python container)
./images/node/test.sh      # (if run inside node container)  
./images/go/test.sh        # (if run inside go container)

# Or run tests from outside containers
docker run --rm -v $(pwd)/images/base/test.sh:/test.sh reactor/base:local bash /test.sh
docker run --rm -v $(pwd)/images/python/test.sh:/test.sh reactor/python:local bash /test.sh
docker run --rm -v $(pwd)/images/node/test.sh:/test.sh reactor/node:local bash /test.sh
docker run --rm -v $(pwd)/images/go/test.sh:/test.sh reactor/go:local bash /test.sh
```

### Interactive Development

```bash
# Run interactive shell in any image for debugging
docker run --rm -it reactor/base:local bash
docker run --rm -it reactor/python:local bash
docker run --rm -it reactor/node:local bash
docker run --rm -it reactor/go:local bash

# Mount local workspace for testing
docker run --rm -it -v $(pwd):/workspace reactor/python:local bash
```

### Multi-Architecture Building (Advanced)

```bash
# Set up buildx for multi-architecture (one-time setup)
docker buildx create --name multiarch --use
docker buildx inspect --bootstrap

# Build for multiple architectures
docker buildx build --platform linux/amd64,linux/arm64 -t reactor/base:multiarch images/base

# Build and push to registry (requires authentication)
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/dyluth/reactor/base:test \
  --push images/base
```

## Automated Production Builds

### Official Images via GitHub Actions

**All official Reactor images are built automatically using GitHub Actions:**

- **Official Images**: `ghcr.io/dyluth/reactor/{base,python,node,go}:latest`
- **Workflow File**: `.github/workflows/build-images.yml`
- **Registry**: GitHub Container Registry (GHCR)
- **Platforms**: linux/amd64, linux/arm64

### Build Pipeline Details
- **Trigger**: Changes to `images/**` or workflow files, or manual dispatch
- **Process**: Build base image first, then language-specific images in parallel
- **Testing**: Each image runs its test suite during build
- **Security**: Trivy vulnerability scanning on every build
- **Publishing**: Automatic push to GHCR on successful builds
- **Artifacts**: SARIF security reports uploaded to GitHub Security tab

### Workflow Setup

The GitHub Actions workflow is **already configured** in the repository. No setup required for basic usage, but here's what it does:

1. **Multi-Architecture Builds**: Uses `docker/build-push-action` with buildx
2. **Dependency Management**: Base image built first, others depend on it  
3. **Security Integration**: Trivy scans with GitHub Security tab integration
4. **Size Monitoring**: Warns if images exceed size targets
5. **Comprehensive Testing**: Runs test scripts for each image

### GitHub Actions Configuration

The workflow uses these GitHub secrets and settings:
- **`GITHUB_TOKEN`**: Automatically provided, used for GHCR authentication
- **Permissions**: `contents: read`, `packages: write`, `security-events: write`
- **Registry**: `ghcr.io` (GitHub Container Registry)

**No additional setup required** - the workflow is ready to use once the repository has GitHub Actions enabled.

### Size Monitoring
- **Base image**: <200MB target
- **Language images**: <500MB target  
- **Alerts**: GitHub Actions will warn if limits exceeded

### Security Scanning
- **Tool**: Trivy vulnerability scanner
- **Frequency**: Every build
- **Severity**: Fails on HIGH or CRITICAL vulnerabilities
- **Results**: Uploaded to GitHub Security tab

## Manual Maintenance Tasks

### Weekly Review (15 minutes)
1. **Check Build Status**
   ```bash
   # View recent workflow runs
   gh run list --workflow=build-images.yml --limit=10
   ```

2. **Review Security Alerts**
   - Visit GitHub Security tab
   - Address any new vulnerability alerts
   - Update base packages if needed

3. **Monitor Image Sizes**
   - Check recent build summaries for size warnings
   - Investigate if any image exceeded targets

### Monthly Updates (1-2 hours)

1. **Update Package Versions**
   ```bash
   # Update Dockerfiles with latest patch versions
   # Focus on security patches first
   
   # Base image priorities:
   # - OS packages (debian:bullseye-slim updates)
   # - Docker CLI updates
   # - Node.js LTS updates
   
   # Language-specific priorities:
   # - Python: security patches only
   # - Node.js: TypeScript, ESLint updates  
   # - Go: patch releases only
   ```

2. **Test Image Functionality**
   ```bash
   # Build images locally for testing (from repository root)
   docker build -t reactor/base:test images/base
   docker build -t reactor/python:test images/python  
   docker build -t reactor/node:test images/node
   docker build -t reactor/go:test images/go
   
   # Run test scripts
   docker run --rm -v $(pwd)/images/base/test.sh:/test.sh reactor/base:test bash /test.sh
   docker run --rm -v $(pwd)/images/python/test.sh:/test.sh reactor/python:test bash /test.sh
   docker run --rm -v $(pwd)/images/node/test.sh:/test.sh reactor/node:test bash /test.sh
   docker run --rm -v $(pwd)/images/go/test.sh:/test.sh reactor/go:test bash /test.sh
   ```

3. **Update Documentation**
   - Review and update this maintenance guide
   - Update version references in README.md
   - Check Dockerfile comments are accurate

### Quarterly Major Updates (4-8 hours)

1. **Language Version Updates**
   - **Python**: Consider minor version updates (3.11 → 3.12)
   - **Node.js**: Update to latest LTS
   - **Go**: Update to latest stable release

2. **Base Image Refresh**
   - Consider updating Debian base image
   - Evaluate new essential tools for inclusion
   - Review and optimize Dockerfile layers

3. **Security Audit**
   - Run comprehensive security scan with multiple tools
   - Review all installed packages for necessity
   - Update to latest secure versions

## Emergency Procedures

### Critical Vulnerability Response
1. **Assessment** (within 2 hours)
   - Determine if vulnerability affects Reactor images
   - Assess severity and exploitability
   - Check if automated builds caught it

2. **Immediate Action** (within 4 hours)  
   - Create hotfix branch
   - Update affected packages
   - Test minimal fix functionality
   - Push fix to trigger rebuild

3. **Communication**
   - Create GitHub issue describing vulnerability
   - Update README with security notice if needed
   - Consider GitHub Security Advisory for severe issues

### Build Failure Recovery
1. **Investigate root cause**
   ```bash
   # Check workflow logs
   gh run view [run-id] --log
   
   # Common causes:
   # - Package version conflicts
   # - Network timeouts during downloads
   # - Registry rate limits
   # - Test failures
   ```

2. **Fix and retry**
   - Address underlying issue
   - Consider reverting to known-good versions temporarily
   - Re-run failed workflow

### Size Limit Exceeded
1. **Identify growth sources**
   ```bash
   # Analyze image layers
   docker history ghcr.io/dyluth/reactor/base:latest
   
   # Find large files
   docker run --rm ghcr.io/dyluth/reactor/base:latest \
     find / -type f -size +10M 2>/dev/null
   ```

2. **Optimization strategies**
   - Combine RUN commands to reduce layers
   - Remove unnecessary packages
   - Clean up caches in same RUN command
   - Use multi-stage builds if needed

## Version Management

### Tagging Strategy
- **latest**: Always points to most recent build from main
- **SHA tags**: Every build gets unique SHA-based tag
- **Branch tags**: Development builds get branch-prefixed tags

### Rollback Procedure
```bash
# If latest image has issues, promote previous good build
# 1. Find previous good SHA tag
gh run list --workflow=build-images.yml --status=success --limit=5

# 2. Retag previous build as latest (requires admin access)
# This should be automated in future iterations
```

## Monitoring and Alerts

### Key Metrics
- Build success rate (target: >99.9%)
- Image pull counts (growth indicator)
- Vulnerability count (target: 0 high/critical)
- Build duration (should remain stable)

### Alert Thresholds
- Build failures: Immediate notification
- Security vulnerabilities: Within 2 hours
- Size increases >20%: Weekly review
- Download spikes: Investigate for issues

## Contributing to Images

### Making Changes
1. Create feature branch from main
2. Modify relevant Dockerfile(s)
3. Test changes locally
4. Update test scripts if needed
5. Create PR - automated builds will test
6. Merge after review and successful builds

### Best Practices
- Pin all package versions
- Minimize layer count
- Clean up in same RUN command as installation
- Add comments explaining non-obvious choices
- Update corresponding test scripts
- Consider backward compatibility

For questions or assistance, create an issue in the repository.