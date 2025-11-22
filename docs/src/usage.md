# Usage Guide

`portctl` provides a suite of commands to manage network ports and processes.

## Core Commands

### `list` - List Processes

List processes that are currently listening on ports.

```bash
# List all processes with open ports
portctl list

# List processes on a specific port
portctl list 8080

# List all processes (explicit)
portctl list --all
```

**Options:**
- `--json`, `-j`: Output in JSON format for scripting.
- `--all`, `-a`: List all processes.
- `--sort [field]`: Sort by `pid`, `port`, `cpu`, `memory`, `command`, `service`, or `user`.
- `--service [name]`: Filter by service name (e.g., `node`, `postgres`).
- `--user [name]`: Filter by user name.

### `kill` - Kill Processes

Kill processes by port or PID.

```bash
# Kill processes on port 8080
portctl kill 8080

# Kill process by PID
portctl kill --pid 12345

# Kill multiple ports
portctl kill 3000 8080 9000
```

**Options:**
- `--pid`, `-p`: Kill by PID instead of port.
- `--force`, `-f`: Force kill (SIGKILL on Unix, /F on Windows).
- `--yes`, `-y`: Skip confirmation prompt.
- `--service [name]`: Kill all processes matching a service name.
- `--user [name]`: Kill all processes owned by a user.
- `--older [duration]`: Kill processes older than a duration (e.g., `1h`).

### `interactive` - TUI Mode

Launch an interactive terminal user interface.

```bash
portctl interactive
# or alias
portctl tui
```

This mode allows you to:
- Browse processes with arrow keys.
- Filter list by typing `/`.
- View details by pressing `Enter`.
- Kill processes by pressing `k`.
- View system stats by pressing `s`.

### `watch` - Real-time Monitoring

Watch ports for changes in real-time.

```bash
# Watch all ports
portctl watch

# Watch a specific port
portctl watch 8080
```

**Options:**
- `--interval`, `-i`: Refresh interval (default `3s`).
- `--notify`, `-n`: Send desktop notifications on changes.
- `--changes-only`, `-c`: Only display output when changes occur.

### `scan` - Port Scanning

Scan local or remote hosts for open ports.

```bash
# Scan localhost for common ports
portctl scan localhost --common

# Scan a specific range
portctl scan 192.168.1.1 1-1000
```

**Options:**
- `--common`: Scan top 20 common ports.
- `--range`, `-r`: Specify port range (e.g., `80,443,3000-4000`).
- `--timeout`, `-t`: Connection timeout (default `3s`).
- `--concurrent`, `-c`: Number of concurrent scans (default `50`).

### `quick` - Developer Shortcuts

Quick actions for common developer tasks.

```bash
# Kill all development servers (ports 3000-9999)
portctl quick kill-dev

# Kill all Node.js processes
portctl quick kill-node

# Find next available port and export it
portctl quick next-port
```

### `available` - Find Free Ports

Find available ports for binding.

```bash
# Find 10 free ports starting from 3000
portctl available

# Find in specific range
portctl available --start 8000 --end 9000
```

### `stats` - System Statistics

View system resource usage and port statistics.

```bash
portctl stats
```

## Global Flags

- `--help`, `-h`: Show help for any command.
- `--version`, `-v`: Show version information.
