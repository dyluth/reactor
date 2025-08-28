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

## Automated Maintenance

### Build Pipeline
- **Trigger**: Changes to `images/**` or workflow files
- **Platforms**: linux/amd64, linux/arm64
- **Registry**: GitHub Container Registry (GHCR)
- **Security**: Trivy vulnerability scanning on every build

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
   # Build images locally for testing
   cd images/base && docker build -t test-base .
   cd ../python && docker build -t test-python .
   cd ../node && docker build -t test-node .  
   cd ../go && docker build -t test-go .
   
   # Run test scripts
   docker run --rm -v $(pwd)/images/base/test.sh:/test.sh test-base bash /test.sh
   # Repeat for other images
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