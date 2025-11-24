# Introduction

Welcome to the documentation for **portctl**!

`portctl` is a modern, high-performance CLI tool designed to help developers manage network ports and processes with ease. It eliminates the need to remember complex `lsof` or `netstat` commands and provides a safe, user-friendly interface for identifying and killing processes.

## Key Features

- **🔍 Smart Process Discovery**: Instantly find what's running on any port.
- **⚡ Fast & Safe Killing**: Terminate processes by port or PID with safety checks.
- **🖥️ Interactive TUI**: A beautiful terminal UI for browsing and managing processes.
- **👀 Real-time Monitoring**: Watch ports for changes and get desktop notifications.
- **🛡️ Developer Focused**: "Quick" commands to kill dev servers, find free ports, and clean up zombies.
- **📊 System Stats**: Insight into system resource usage and port distribution.
- **🤖 AI Integration**: MCP server support with `.well-known` metadata (mcp-manifest.jsonld, llms.txt, skills.txt).
- **🔒 Secure**: SLSA attestations, dual SBOM generation (SPDX + CycloneDX), and comprehensive dependency tracking.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install ckodex-labs/tap/portctl
```

### Go Install

```bash
go install github.com/ckodex-labs/portctl@latest
```

### Manual Download

Download the latest binary from the [Releases Page](https://github.com/ckodex-labs/portctl/releases).

## Getting Started

Check out the [Usage Guide](usage.md) to learn how to use the CLI commands.

For architectural details and design decisions, see the [Architecture (arc42)](arc42/01_introduction.md) section.
