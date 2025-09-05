# Reactor Project Context & Development Guide

## 🎯 **Project Vision & Ethos**

**Reactor** is a high-performance, developer-focused CLI for Dev Containers that creates the fastest and most ergonomic development experience possible. We build with **zero tolerance for test suite instability** and prioritize **architectural solutions over quick hacks**.

### Core Principles
- **Reliability First**: Test suite must always be stable - no test pollution allowed
- **Architectural Excellence**: Proper solutions, not workarounds - think hard before implementing
- **Milestone-Driven**: Development follows structured milestones with clear deliverables
- **Docker-First**: All environments run in containers with proper isolation
- **Speed & Ergonomics**: Fastest possible experience for developers

## 🏗️ **Architecture Overview**

Reactor is built as a modular Go application with clean separation of concerns:

```
reactor/
├── cmd/reactor/           # CLI entry points and command handlers
├── pkg/config/            # Configuration parsing and devcontainer.json handling  
├── pkg/core/              # Container blueprint creation and naming logic
├── pkg/docker/            # Docker API interactions and container management
├── pkg/orchestrator/      # Core up/down orchestration logic (Milestone 4)
├── pkg/integration/       # Comprehensive integration test suite
└── pkg/testutil/          # Test utilities and cleanup functions
```

## 📋 **Current Status: Milestone 3 Complete, Milestone 4 Ready**

### ✅ **Completed: Milestone 3 - Account-Based Credential Mounting**
- Account-based credential mounting with isolation prefixes
- Configurable container entrypoints via `devcontainer.json` defaultCommand
- Docker host integration with security controls  
- Robust path handling for directories with spaces/special characters
- Label-based Docker container cleanup (`com.reactor.test=true`)

### 🚧 **Next: Milestone 4 - Multi-Container Workspaces**
Implementation plan is defined in `/docs/features/reactor/milestone4_workspaces.md`:
- **PR 0**: Orchestrator refactoring (move up/down logic to pkg/orchestrator)
- **PR 1**: Workspace parser, validate, and list commands
- **PR 2**: Workspace up and exec commands with parallel execution
- **PR 3**: Workspace down command and integration testing

## 🗂️ **Key Files to Read (Priority Order)**

### 1. **Architecture & Design Documents**
- `/docs/features/reactor/milestone4_workspaces.md` - Next milestone specification
- `/docs/features/reactor/milestone3_reactor_extensions.md` - Recently completed features
- `/CLAUDE.md` - Project-specific development guidelines

### 2. **Core Implementation Files**
- `/pkg/orchestrator/orchestrator.go` - Core up/down orchestration logic
- `/pkg/core/blueprint.go` - Container blueprint creation and naming
- `/pkg/config/service.go` - Configuration resolution and devcontainer.json parsing
- `/pkg/docker/service.go` - Docker API interactions and container management

### 3. **Test Architecture (Critical to Understand)**
- `/pkg/integration/main_test.go` - Global test setup with Docker label cleanup
- `/pkg/integration/milestone3_test.go` - Credential mounting integration tests
- `/pkg/testutil/docker_cleanup.go` - Label-based cleanup functions
- `/pkg/integration/basic_test.go` - Core functionality integration tests

### 4. **Command Handlers**
- `/cmd/reactor/main.go` - CLI setup and command registration
- `/cmd/reactor/up.go` - Single container up command (to be refactored in M4)
- `/cmd/reactor/config.go` - Configuration management commands

## 🧪 **Testing Philosophy & Architecture**

### Test Isolation Strategy
- **Docker Labels**: All test containers use `com.reactor.test=true` label
- **Isolation Prefixes**: `REACTOR_ISOLATION_PREFIX` for parallel test execution
- **Global Cleanup**: `globalTestCleanup()` runs before/after test suites
- **No Test Pollution**: Zero tolerance - tests must be completely isolated

### Test Commands
```bash
make test-isolated    # Run all tests with proper isolation
make ci              # Full CI pipeline validation
go test ./pkg/integration -run TestAccountBasedCredentialMounting -v
```

### Test Patterns
- Use `testutil.SetupIsolatedTest(t)` for integration tests
- Always use `t.Cleanup()` for container cleanup
- Create temporary directories inside reactor project root only
- Use meaningful test names that describe the scenario

## 🛠️ **Development Workflow**

### 1. **Understanding Changes**
- Always read relevant design documents first
- Examine existing test coverage for similar features
- Understand the broader architectural context

### 2. **Making Changes**
- Follow existing code patterns and conventions
- Never assume libraries are available - check imports first
- Security first - never expose secrets or credentials
- Add comprehensive test coverage

### 3. **Quality Standards**
- Run `make check` for quick validation
- Run `make ci` before commits for full validation
- All tests must pass - no exceptions
- Code must be properly formatted (`make fmt`)

### 4. **Commit Standards**
```bash
git commit -m "Brief description of change

Detailed explanation of what was changed and why.

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

## 🎯 **Working with Claude Code**

### Communication Style
- **Direct & Concise**: Answer questions without preamble or postamble
- **Action-Oriented**: Show code, run tests, demonstrate solutions
- **Quality-Focused**: Architectural fixes over quick hacks
- **Test-Driven**: Always verify solutions with comprehensive testing

### Development Approach
1. **Analyze** the milestone specification and existing code
2. **Plan** the implementation with architectural considerations
3. **Implement** with proper error handling and test coverage
4. **Test** thoroughly with integration and unit tests
5. **Verify** no regressions with full test suite

### File Handling Rules
- Always create temporary files/folders inside the reactor directory
- Never modify files unless required for the specific task
- Prefer editing existing files over creating new ones
- Only create documentation if explicitly requested

## 🔍 **Key Technologies & Dependencies**

- **Go 1.21+**: Core language with comprehensive standard library usage
- **Cobra**: CLI framework for command handling
- **Docker Go SDK**: Container management and API interactions
- **HuJSON**: JSON-with-comments parsing for devcontainer.json
- **Testify**: Assertion library for tests (in some legacy tests)

## 📈 **Success Metrics**

### Technical Excellence
- Zero flaky tests - test suite must be 100% reliable
- No test pollution between test runs
- Comprehensive integration test coverage
- Docker container lifecycle properly managed

### User Experience
- Single-command workflows for complex operations
- Clear, actionable error messages
- Fast startup times and responsive operations
- Intuitive CLI interface following Unix conventions

---

**Ready for Milestone 4 Implementation** - The foundation is solid, the architecture is clean, and the test suite provides confidence for the next phase of development.