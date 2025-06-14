# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

sshx is a secure web-based collaborative terminal application with a hybrid
Rust/TypeScript architecture:

- **Backend**: Rust workspace in `/crates/` with server, client CLI, and core
  libraries
- **Frontend**: SvelteKit web application with TypeScript in `/src/`
- **Real-time Communication**: gRPC for server-client, WebSockets for web
  frontend
- **Encryption**: End-to-end encryption using Argon2 + AES

## Key Commands

### Development Environment Setup

```bash
# Start Redis (required for server)
docker compose up -d

# Install frontend dependencies
npm install

# Run everything in parallel (recommended)
mprocs
```

### Frontend Development

```bash
npm run dev      # Start Vite dev server (port 5173)
npm run build    # Build for production
npm run check    # Type check with svelte-check
npm run lint     # Lint TypeScript/JavaScript
npm run format   # Format with Prettier
```

### Rust Development

```bash
cargo build                    # Build all crates
cargo run --bin sshx-server   # Run server (port 8051 in dev)
cargo run --bin sshx          # Run client CLI
cargo test                    # Run all tests
cargo fmt                     # Format Rust code
```

### Testing

```bash
# Run specific Rust test
cargo test -p sshx-server test_name

# Frontend type checking
npm run check
```

## Architecture Overview

### Rust Workspace Structure (`/crates/`)

- **sshx-core**: Shared protobuf definitions, encryption utilities, and common
  types
- **sshx-server**: gRPC server, WebSocket handling, Redis state management
- **sshx**: CLI client application

### Frontend Structure (`/src/`)

- **routes/**: SvelteKit pages and API routes
- **lib/**: Components, stores, and utilities
- **lib/term/**: Terminal emulation with custom xterm.js fork
- **lib/encryption.ts**: Client-side encryption matching Rust implementation

### Key Architectural Patterns

1. **State Management**: Redis stores session state server-side
2. **Protocol**: Protobuf definitions in `crates/sshx-core/proto/` define all
   messages
3. **Terminal Multiplexing**: Multiple shells per session with window management
4. **Static URLs**: Sessions can be accessed via `/s/{session-id}#{secret}`

## Development Notes

- The server requires Redis to be running (use `docker compose up -d`)
- Frontend proxies API requests to `http://localhost:8051` in development
- Terminal functionality uses a custom fork: `@ekzhang/sshx-xterm`
- Build artifacts go to `target/` (Rust) and `build/` (SvelteKit)
- Release builds use `scripts/release.sh` for multi-platform compilation
