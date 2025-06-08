[![OpenSSF Best Practices](https://bestpractices.coreinfrastructure.org/projects/0/badge)](https://bestpractices.coreinfrastructure.org/projects/0)

# portctl 🚀

A fast, cross-platform CLI tool for managing processes on specific ports. Perfect for developers who need to quickly identify and kill processes occupying ports during development.

## Features

- 🔍 **List processes** on specific ports or all ports
- ⚡ **Kill processes** by port or PID
- 🖥️ **Cross-platform** support (macOS, Linux, Windows)
- 🎨 **Beautiful output** with colored tables
- 🛡️ **Safety features** with confirmation prompts
- 📊 **JSON output** for scripting and automation
- 🚀 **Zero dependencies** - single binary

---

## ⚙️ SDLC Automation & Meta-Architecture Compliance

This project uses a **modular, MCStack-compliant Dagger pipeline** for full SDLC, compliance, and meta-architecture automation:

### Dagger Pipeline Usage

- **Full pipeline:**
  ```bash
  go run .dagger/main.go --step all
  ```
- **Individual steps:**
  ```bash
  go run .dagger/main.go --step lint       # Lint code
  go run .dagger/main.go --step test       # Run tests
  go run .dagger/main.go --step build      # Build binaries
  go run .dagger/main.go --step snapshot   # CLI snapshot regression tests
  go run .dagger/main.go --step release    # GoReleaser: build, sign, SBOM, Docker, Homebrew
  go run .dagger/main.go --step docs       # Build mdBook docs
  go run .dagger/main.go --step mcp        # Run MCP server
  go run .dagger/main.go --step wellknown  # Validate .well-known/ metadata
  ```

### Snapshot Regression Testing

- CLI output is regression-tested with [Cupaloy](https://github.com/bradleyjkemp/cupaloy):
  ```bash
  go test ./internal/snapshots
  ```
  This ensures anti-fragility and prevents unintentional CLI changes.

### Documentation (mdBook)

- Docs are in `docs/` and built with [mdBook](https://rust-lang.github.io/mdBook/):
  ```bash
  go run .dagger/main.go --step docs
  mdbook serve docs  # for local preview
  ```
  Docs are ready for CI/CD and GitHub Pages publishing.

### Meta-Architecture & Compliance

- **SLSA 4, SBOM, Keyless Signing:** Release pipeline uses GoReleaser for multi-platform, signed, and SBOM-compliant builds.
- **MCP Server:** `/mcp` endpoint serves a machine-readable manifest for AI/LLM and tool ecosystem integration.
- **.well-known/**: All compliance, AI, and SBOM metadata is published for discoverability and audit.
- **MCStack Principles:** Modular, auditable, and anti-fragile by design.

---

## 🧠 MCP gRPC API for portctl

The MCP server exposes the real capabilities of the `portctl` CLI via a gRPC API for automation, LLMs, and agentic workflows.

### Endpoints

- `ListProcesses(ListProcessesRequest) → ListProcessesResponse`
  - List processes by port, user, or all.
- `KillProcess(KillProcessRequest) → KillProcessResponse`
  - Kill a process by PID or port.
- `GetStatus(StatusRequest) → StatusResponse`
  - Returns the current portctl version and server uptime.

### How to Use

1. **Start the server:**
   ```sh
   go run ./cmd/mcp-server
   ```
2. **Call from a gRPC client:**
   - Use Go, Python, or `grpcurl`:
     ```sh
     grpcurl -plaintext localhost:50051 mcp.PortctlService/GetStatus
     ```
3. **Integration Test:**
   - Ensure server is running, then:
     ```sh
     go test ./internal/tests/ -run TestGetStatus
     ```

### Proto File Location
- `proto/mcp.proto` (see for full message definitions)

### Security
- Only exposes safe, real portctl features.
- All actions are logged for auditability.

---

## Installation

### Option 1: Build from source
```bash
git clone https://github.com/mchorfa/portctl.git
cd portctl
go build -o portctl
sudo mv portctl /usr/local/bin/  # Optional: add to PATH
```

### Option 2: Go install
```bash
go install github.com/mchorfa/portctl@latest
```

### Option 3: Download binary
Download the latest binary from the [releases page](https://github.com/mchorfa/portctl/releases).

## Quick Start

```bash
# List all processes with open ports
portctl list

# List processes on a specific port
portctl list 8080

# Kill processes on port 8080
portctl kill 8080

# Kill a specific process by PID
portctl kill --pid 12345

# Force kill without confirmation
portctl kill 8080 --force --yes
```

## Usage

### List Command

```bash
# List all processes with open ports
portctl list

# List processes on port 8080
portctl list 8080

# Output in JSON format
portctl list 8080 --json

# List all processes (explicit)
portctl list --all
```

**Example output:**
```
| PID   | PORT | PROTOCOL | STATE  | COMMAND  |
| ----- | ---- | -------- | ------ | -------- |
| 12345 | 8080 | tcp      | LISTEN | node     |
| 12346 | 3000 | tcp      | LISTEN | python3  |
| 12347 | 5432 | tcp      | LISTEN | postgres |

Found 3 process(es)
```

### Kill Command

```bash
# Kill processes on port 8080 (with confirmation)
portctl kill 8080

# Kill process by PID
portctl kill --pid 12345

# Force kill (SIGKILL/taskkill /F)
portctl kill 8080 --force

# Skip confirmation prompt
portctl kill 8080 --yes

# Combine flags
portctl kill 8080 --force --yes
```

**Example interaction:**
```bash
$ portctl kill 8080
Found 1 process(es) on port 8080:
  PID 12345: node (tcp)

Are you sure you want to kill 1 process(es) on port 8080? [y/N]: y
Killing process 12345 (node)...
Successfully killed process 12345
Successfully killed all processes on port 8080
```

## Advanced Usage

### JSON Output for Scripting

```bash
# Get JSON output for automation
portctl list 8080 --json | jq '.[0].pid'

# Kill all Node.js processes on various ports
for port in 3000 8080 8081; do
  portctl kill $port --yes 2>/dev/null || true
done
```

### Integration with Other Tools

```bash
# Find and kill all Node.js processes
portctl list --json | jq -r '.[] | select(.command | contains("node")) | .pid' | \
  xargs -I {} portctl kill --pid {} --yes

# Monitor port usage
watch -n 2 'portctl list'
```

## Command Reference

### Global Flags
- `--help, -h`: Show help
- `--version, -v`: Show version

### `portctl list [port]`
List processes on ports.

**Arguments:**
- `port` (optional): Specific port number to check

**Flags:**
- `--json, -j`: Output in JSON format
- `--all, -a`: List all processes (same as omitting port)

### `portctl kill [port]`
Kill processes on ports.

**Arguments:**
- `port`: Port number (required unless --pid is used)

**Flags:**
- `--pid, -p INT`: Kill specific process by PID
- `--force, -f`: Force kill (SIGKILL on Unix, /F on Windows)
- `--yes, -y`: Skip confirmation prompt

## Platform Support

### macOS/Linux
- Uses `lsof` when available (more accurate)
- Falls back to `netstat` if `lsof` is not installed
- Supports `SIGTERM` (graceful) and `SIGKILL` (force) signals

### Windows
- Uses `netstat` and `tasklist` for process discovery
- Uses `taskkill` for termination
- Supports normal and force (`/F`) termination

## Safety Features

1. **Confirmation prompts**: Always asks before killing processes (unless `--yes`)
2. **Process listing**: Shows exactly what will be killed before doing it
3. **Graceful termination**: Uses SIGTERM by default, SIGKILL only with `--force`
4. **Error handling**: Clear error messages and non-zero exit codes on failure
5. **PID validation**: Verifies processes exist before attempting to kill them

## Common Use Cases

### Development Workflow
```bash
# Check what's running on your dev ports
portctl list

# Kill that stuck dev server
portctl kill 3000

# Clean up after testing
portctl kill 8080 8081 8082 --yes
```

### Port Conflicts
```bash
# Find what's using port 80
portctl list 80

# Kill it if it's safe to do so
portctl kill 80
```

### Automation Scripts
```bash
#!/bin/bash
# cleanup-dev-ports.sh
PORTS=(3000 8080 8081 8082 5000)

echo "Cleaning up development ports..."
for port in "${PORTS[@]}"; do
  if portctl list "$port" &>/dev/null; then
    echo "Killing processes on port $port"
    portctl kill "$port" --yes
  fi
done
echo "Cleanup complete!"
```

## Error Handling

portctl provides clear error messages and appropriate exit codes:

- `0`: Success
- `1`: General error (invalid arguments, process not found, etc.)
- `2`: Permission denied (may need sudo/admin privileges)

## Performance

- **Fast startup**: Minimal dependencies and efficient process discovery
- **Low memory**: Typically uses <10MB RAM
- **Cross-platform**: Single codebase works on all major platforms

## Building from Source

Requirements:
- Go 1.21 or later

```bash
# Clone the repository
git clone https://github.com/mchorfa/portctl.git
cd portctl

# Download dependencies
go mod download

# Build for current platform
go build -o portctl

# Build for all platforms
make build-all  # If Makefile is available

# Run tests
go test ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Troubleshooting

### Permission Denied
Some processes may require elevated privileges to kill:
```bash
# On Unix systems
sudo portctl kill 80

# On Windows (run as Administrator)
portctl kill 80
```

### Command Not Found
Make sure the binary is in your PATH:
```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH=$PATH:/path/to/portctl

# Or install globally
sudo mv portctl /usr/local/bin/
```

### Process Still Running
Some processes may ignore SIGTERM. Use force kill:
```bash
portctl kill 8080 --force
```

## Similar Tools

- `lsof`: More powerful but complex syntax
- `netstat`: Basic but requires manual PID lookup
- `fuser`: Unix-only, limited output formatting
- `ss`: Modern netstat replacement, but no kill functionality

portctl combines the best of these tools with a developer-friendly interface! 🎉

## 🛡️ OpenSSF Best Practices

This project aims to comply with the [OpenSSF Best Practices](https://bestpractices.coreinfrastructure.org/) for open source security and quality. See the [OpenSSF Badge](https://bestpractices.coreinfrastructure.org/) for more info.

- Automated CI/CD with GitHub Actions
- Security and vulnerability scanning
- Static analysis and code quality checks
- Documentation and artifact publishing

## 🧪 Quality Automation

The Makefile provides a `quality` target that runs all major static analysis and code quality checks:

```sh
make quality
```

This runs:
- Linting (`lint`)
- Vetting (`vet`)
- Staticcheck (`staticcheck`)
- Ineffassign (`ineffassign`)
- Misspell (`misspell`)
- Deadcode (`deadcode`)
- Go mod tidy check (`mod-tidy-check`)
- Go fmt check (`fmt-check`)

## 📈 Coverage

![Coverage](coverage.svg) <!-- Add a real badge if using a service -->

## 🏗️ CI/CD

All pushes and pull requests are checked by GitHub Actions for build, test, lint, security, and documentation. See `.github/workflows/ci.yml` for details.
