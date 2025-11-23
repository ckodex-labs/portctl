# Architecture Documentation

This section provides architectural documentation for portctl following the arc42 template.

## Introduction and Goals

### Requirements Overview

**portctl** is a secure, cross-platform CLI tool designed to simplify port and process management for developers. It addresses the common pain point of identifying and terminating processes occupying specific ports during development.

**Key Requirements:**
- **Cross-platform compatibility**: Works on macOS, Linux, and Windows
- **Security**: Safe process termination with confirmation prompts
- **Developer experience**: Intuitive commands, interactive TUI, real-time monitoring
- **AI integration**: MCP server support for AI agent interaction
- **Automation**: JSON output for scripting and CI/CD integration

### Quality Goals

| Priority | Quality Goal | Scenario |
|----------|-------------|----------|
| 1 | **Security** | All process termination requires confirmation unless explicitly bypassed with `--yes` flag |
| 2 | **Reliability** | 100% accuracy in port-to-process mapping across all supported platforms |
| 3 | **Usability** | Commands are intuitive and self-documenting with `--help` |
| 4 | **Performance** | Process listing completes in <100ms for typical workloads |
| 5 | **Maintainability** | 80%+ test coverage, clear separation of concerns |

### Stakeholders

| Role | Expectations |
|------|-------------|
| **Developers** | Fast, reliable tool to kill processes on ports without memorizing `lsof`/`netstat` syntax |
| **DevOps Engineers** | Scriptable automation for CI/CD pipelines and deployment workflows |
| **AI Agents** | Programmatic access via MCP server for autonomous port management |
| **Contributors** | Well-documented codebase with clear architecture and testing guidelines |

## Architecture Constraints

### Technical Constraints

| Constraint | Description |
|------------|-------------|
| **Language** | Go 1.21+ for cross-platform compatibility and performance |
| **Dependencies** | Minimal external dependencies; prefer standard library |
| **Platforms** | macOS, Linux (various distros), Windows 10+ |
| **Distribution** | Single binary with no runtime dependencies |

### Organizational Constraints

| Constraint | Description |
|------------|-------------|
| **License** | MIT License for maximum adoption |
| **CI/CD** | GitHub Actions with Dagger for reproducible builds |
| **Release** | Automated releases via GoReleaser with SLSA attestations |
| **Documentation** | mdBook for user docs, godoc for API documentation |

## System Scope and Context

### Business Context

```
┌─────────────┐
│  Developer  │
└──────┬──────┘
       │ CLI commands
       ▼
┌─────────────────┐
│    portctl      │
└──────┬──────────┘
       │
       ├─► OS Process Manager (kill signals)
       ├─► Network Stack (lsof/netstat)
       └─► Terminal (TUI rendering)
```

### Technical Context

**External Interfaces:**

1. **Operating System APIs**
   - Process management (signals, PIDs)
   - Network information (`lsof`, `netstat`, `/proc`)
   - Terminal control (for TUI)

2. **User Interfaces**
   - CLI (Cobra framework)
   - TUI (Bubble Tea framework)
   - JSON output for scripting

3. **AI Integration**
   - MCP (Model Context Protocol) server
   - `.well-known` metadata files

## Solution Strategy

### Technology Decisions

| Decision | Rationale |
|----------|-----------|
| **Go** | Cross-platform, single binary, excellent stdlib, fast compilation |
| **Cobra** | Industry-standard CLI framework with excellent UX patterns |
| **Bubble Tea** | Modern TUI framework with reactive architecture |
| **Dagger** | Reproducible CI/CD pipelines, local-remote parity |
| **GoReleaser** | Automated multi-platform releases with SLSA compliance |

### Architectural Patterns

1. **Command Pattern**: Each CLI command is a separate handler with clear responsibilities
2. **Adapter Pattern**: Platform-specific process managers (macOS/Linux/Windows) implement common interface
3. **Observer Pattern**: Watch mode uses polling with notification callbacks
4. **Repository Pattern**: Process information is abstracted behind `ProcessManager` interface

## Building Block View

### Level 1: System Overview

```
┌────────────────────────────────────────────────┐
│                   portctl                      │
├────────────────────────────────────────────────┤
│  CLI Layer (Cobra)                             │
│  ├─ list, kill, watch, scan, stats, tui        │
├────────────────────────────────────────────────┤
│  Business Logic                                │
│  ├─ ProcessManager                             │
│  ├─ PortScanner                                │
│  ├─ Notifier                                   │
├────────────────────────────────────────────────┤
│  Platform Adapters                             │
│  ├─ macOS (lsof)                               │
│  ├─ Linux (lsof, /proc)                        │
│  ├─ Windows (netstat)                          │
└────────────────────────────────────────────────┘
```

### Level 2: Component Details

**ProcessManager**
- Responsibility: Abstract process listing and termination
- Interface:
  ```go
  type ProcessManager interface {
      GetAllProcesses() ([]Process, error)
      GetProcessesOnPort(port int) ([]Process, error)
      KillProcess(pid int, force bool) error
  }
  ```

**CLI Commands**
- `list`: Query and display processes
- `kill`: Terminate processes by port or PID
- `watch`: Real-time monitoring with notifications
- `scan`: Port scanning (local/remote)
- `tui`: Interactive terminal UI
- `stats`: System resource statistics

## Runtime View

### Scenario: Kill Process on Port

```
User                CLI              ProcessManager      OS
  │                  │                     │             │
  │─ portctl kill 8080 ──────────────────►│             │
  │                  │                     │             │
  │                  │──GetProcessesOnPort(8080)────────►│
  │                  │◄─────────────[Process{pid:1234}]──│
  │                  │                     │             │
  │◄─Confirm kill?───│                     │             │
  │─ yes ───────────►│                     │             │
  │                  │──KillProcess(1234)──────────────►│
  │                  │◄─────────────success──────────────│
  │◄─Success─────────│                     │             │
```

## Deployment View

### Distribution Channels

1. **Homebrew** (macOS/Linux)
   ```bash
   brew install ckodex-labs/tap/portctl
   ```

2. **Direct Download** (All platforms)
   - GitHub Releases with signed binaries
   - Multi-arch support (amd64, arm64)

3. **Docker** (Containerized environments)
   ```bash
   docker pull ghcr.io/ckodex-labs/portctl:latest
   ```

4. **Go Install** (Developers)
   ```bash
   go install github.com/ckodex-labs/portctl@latest
   ```

### Infrastructure

```
┌─────────────────────────────────────────┐
│          GitHub Repository              │
├─────────────────────────────────────────┤
│  - Source Code                          │
│  - CI/CD (GitHub Actions + Dagger)      │
│  - Releases (GoReleaser)                │
│  - Container Registry (GHCR)            │
│  - Documentation (GitHub Pages)         │
└─────────────────────────────────────────┘
```

## Cross-cutting Concepts

### Security

- **Process Isolation**: Only kills processes owned by current user (unless sudo)
- **Confirmation Prompts**: Prevents accidental termination
- **Input Validation**: All user inputs are validated and sanitized
- **SLSA Level 3 Compliance**: Signed build provenance and attestations for supply chain security

#### SBOM (Software Bill of Materials)

Each release includes comprehensive SBOMs in **both** industry-standard formats:
- **SPDX format**: `*.sbom.spdx.json` - Linux Foundation standard
- **CycloneDX format**: `*.sbom.cyclonedx.json` - OWASP standard
- Generated using **Syft** (via GoReleaser)
- Lists all dependencies and their versions
- Helps with vulnerability tracking and compliance
- Compatible with different security scanning tools

#### Signed Releases

Release signing with Cosign:
- **Cosign** integration enabled in `.goreleaser.yml`
- **Keyless signing** using GitHub OIDC (`COSIGN_EXPERIMENTAL=1`)
- Generates `.sig` signature files and `.cert` certificates
- Provides cryptographic proof of artifact integrity
- Verifiable with `cosign verify-blob` command

**Verification Example:**
```bash
cosign verify-blob \
  --certificate portctl_linux_amd64.tar.gz.cert \
  --signature portctl_linux_amd64.tar.gz.sig \
  portctl_linux_amd64.tar.gz
```

#### AI Integration Metadata (`.well-known/`)

The project includes AI-native metadata for agent integration:
- **`mcp-manifest.jsonld`**: Model Context Protocol server manifest (JSON-LD format)
- **`llms.txt`**: LLM guidance and context for AI agents
- **`skills.txt`**: Capability descriptions for MCP tools
- Published to GitHub Pages at `/.well-known/` path
- Included in release archives for offline access

### Error Handling

- **Graceful Degradation**: Falls back to alternative methods if primary fails
- **User-Friendly Messages**: Clear error messages with suggested actions
- **Logging**: Structured logging for debugging (optional verbose mode)

### Testing Strategy

- **Unit Tests**: Core business logic (80%+ coverage)
- **Integration Tests**: Platform-specific adapters
- **BDD Tests**: User scenarios with Godog
- **Snapshot Tests**: CLI output validation

## Design Decisions

### ADR-001: Use Go for Implementation

**Status**: Accepted

**Context**: Need cross-platform CLI tool with minimal dependencies

**Decision**: Use Go as primary language

**Consequences**:
- ✅ Single binary distribution
- ✅ Excellent cross-platform support
- ✅ Fast compilation and execution
- ❌ Larger binary size than scripting languages

### ADR-002: Dagger for CI/CD

**Status**: Accepted

**Context**: Need reproducible builds across local and CI environments

**Decision**: Use Dagger for all CI/CD pipelines

**Consequences**:
- ✅ Local-remote parity
- ✅ Portable pipeline definitions
- ✅ Container-based isolation
- ❌ Learning curve for contributors

### ADR-003: MCP Server Integration

**Status**: Accepted

**Context**: Enable AI agent interaction with port management

**Decision**: Implement MCP server alongside CLI

**Consequences**:
- ✅ AI-native tool design
- ✅ Future-proof for AI workflows
- ✅ Programmatic access
- ❌ Additional maintenance surface

## Quality Requirements

### Performance

- Process listing: <100ms for typical workloads
- TUI refresh rate: 60 FPS
- Memory footprint: <50MB

### Security

- SLSA Level 4 compliance
- Signed releases with provenance
- No privilege escalation vulnerabilities

### Maintainability

- 80%+ test coverage
- Clear module boundaries
- Comprehensive documentation

## Risks and Technical Debt

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Platform API changes | Medium | High | Automated testing on multiple OS versions |
| Dependency vulnerabilities | Low | Medium | Dependabot + regular updates |
| Performance degradation | Low | Medium | Benchmark tests in CI |

### Known Technical Debt

1. **Windows Support**: Less mature than macOS/Linux implementations
2. **Test Coverage**: Some edge cases not fully covered
3. **Documentation**: Some advanced features lack detailed examples

## Glossary

| Term | Definition |
|------|------------|
| **MCP** | Model Context Protocol - standard for AI agent tool integration |
| **SLSA** | Supply-chain Levels for Software Artifacts - security framework |
| **TUI** | Terminal User Interface - interactive console application |
| **Dagger** | CI/CD framework using containers for reproducible builds |
| **arc42** | Template for architecture documentation |
