# sshx-term

A terminal client for connecting to sshx sessions that works and feels like SSH.

## Features

- **SSH-like experience**: Connect directly to sshx sessions from your terminal
- **Smart terminal selection**: Automatically handles multiple terminals in a session
- **Terminal UI selector**: When multiple terminals exist, shows a clean TUI to choose
- **Raw terminal mode**: Full terminal capabilities with proper signal handling
- **Clean exit handling**: Proper terminal cleanup on exit, crash, or Ctrl+C

## Installation

```bash
cargo build --bin sshx-term
```

## Usage

### Basic Connection

```bash
# Connect to a session (various URL formats supported)
sshx-term "https://sshx.io/s/abc123#encryption_key"
sshx-term "sshx.io/s/abc123#encryption_key"
sshx-term "abc123#encryption_key"  # assumes sshx.io
sshx-term "abc123#encryption_key@custom.server"

# With write password (for read-write access)
sshx-term "abc123#encryption_key,write_password"
```

### Connection Behavior

The client automatically handles terminal selection:

- **No terminals**: Creates a new terminal automatically
- **One terminal**: Connects directly without showing selector
- **Multiple terminals**: Shows a terminal selector UI

### Terminal Selector

When multiple terminals exist, you'll see a clean table interface:

```
┌─ Select Terminal ──────────────────────────────────────────────────────┐
│ #  │ ID  │ Title/Process              │ Size    │ Activity  │ Status   │
│ 1  │ 1   │ bash                       │ 80×24   │ 5s        │ Active   │
│ 2  │ 2   │ vim:README.md              │ 120×40  │ 1m        │ Focused  │
│ 3  │ 3   │ htop                       │ 80×24   │ 30s       │ Active   │
│ 4  │ 4   │ ssh:server.com             │ 80×24   │ 2h        │ Idle     │
│ n  │ NEW │ Create new terminal        │ -       │ -         │ Ready    │
└────────────────────────────────────────────────────────────────────────┘
┌──────────────────────────────────────────────────────────────────────┐
│  Use ↑↓ to navigate, ENTER to select, 'n' for new terminal, 'q' to quit │ 4 terminals available │
└──────────────────────────────────────────────────────────────────────┘
```

**Smart Terminal Titles:**
The client automatically detects and displays meaningful terminal titles by:
- **Process Detection**: Recognizes common programs (vim, htop, ssh, git, etc.)
- **File Context**: Shows filenames being edited (`vim:README.md`)
- **Path Context**: Displays current directory (`bash:project`)
- **Remote Sessions**: Shows SSH targets (`ssh:server.com`)
- **Intelligent Truncation**: Keeps important information visible

**Status Indicators:**
- **Active** (green) - Recently active terminal
- **Busy** (yellow) - High activity
- **Idle** (gray) - No activity for 5+ minutes  
- **Focused** (cyan) - Currently focused by other users

**Navigation:**
- **↑↓ Arrow keys** - Navigate between terminals
- **Number keys** - Jump to specific terminal
- **Enter** - Connect to selected terminal
- **n** - Create new terminal
- **q/Esc** - Quit

### Command Line Options

```bash
sshx-term [OPTIONS] <URL>

Options:
  -n, --new                  Always create a new terminal (skip selector)
  -t, --terminal <ID>        Connect to specific terminal ID
  -l, --list                 List terminals and exit (don't connect)
  -r, --readonly             Connect in read-only mode
  -v, --verbose              Enable verbose logging
  -h, --help                 Show help
```

### Examples

```bash
# Always create new terminal
sshx-term -n "abc123#key"

# Connect to specific terminal ID
sshx-term -t 2 "abc123#key"

# List available terminals
sshx-term -l "abc123#key"

# Read-only connection
sshx-term -r "abc123#key"

# Verbose logging
sshx-term -v "abc123#key"
```

## Exiting

The client provides multiple ways to exit safely:

- **Ctrl+D** - Send EOF to exit the client (recommended)
- **Ctrl+] q** - Emergency exit sequence (like telnet)
- **Ctrl+C** - Interrupt and exit cleanly

⚠️ **Important**: Do NOT type `exit` if you're running sshx-term from within an sshx web terminal, as this will close the underlying shell and break the web session. Use **Ctrl+D** or **Ctrl+] q** instead.

The terminal will always be restored to its original state on exit.

## URL Formats

The client supports various sshx URL formats:

- Full URL: `https://sshx.io/s/session_id#encryption_key`
- Domain only: `sshx.io/s/session_id#encryption_key`
- Short form: `session_id#encryption_key` (assumes sshx.io)
- Custom server: `session_id#encryption_key@custom.server`
- With write password: `session_id#encryption_key,write_password`

## Architecture

- **WebSocket connection**: Uses the same protocol as the web frontend
- **End-to-end encryption**: Compatible with sshx's encryption system
- **Raw terminal mode**: Direct keyboard input and terminal output
- **Signal handling**: Proper handling of resize, Ctrl+C, etc.
- **Clean state management**: Ensures terminal is always restored properly

## Development

The client is built with:

- **tokio**: Async runtime
- **tokio-tungstenite**: WebSocket client
- **crossterm**: Terminal control
- **ratatui**: Terminal UI for selector
- **ciborium**: CBOR message encoding
- **sshx encryption**: Shared encryption module