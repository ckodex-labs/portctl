# arc42: 3. System Scope and Context

## 3.1 Business Context

`portctl` is used by developers/ops to manage processes on ports, automate cleanup, and resolve conflicts. It is distributed via Homebrew and GitHub Releases.

## 3.2 Technical Context

 
- CLI tool written in Go
- Integrates with OS process/network tools (lsof, netstat, tasklist)
- CI/CD via Dagger and GitHub Actions
- Release automation with GoReleaser
- Security via SLSA, SBOM, gosec, govulncheck
- Documentation via godoc/swag and arc42
- Homebrew formula for macOS/Linux users
