# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

SSHXtend is an enhanced fork of sshx - a secure web-based collaborative terminal with enterprise features and AI integration. The project uses a sophisticated hybrid Rust/TypeScript architecture:

- **Backend**: Rust workspace in `/crates/` with 4 crates: server, client CLI, terminal client, and core libraries
- **Frontend**: SvelteKit web application with TypeScript, AI integration, and advanced theming
- **Go Client**: Alternative client in `/sshx-go/` for extended architecture support
- **Real-time Communication**: Dual protocol support - gRPC for native clients, WebSockets for web/fallback
- **Encryption**: End-to-end encryption using Argon2id + AES-128-CTR with public salt strategy
- **Enterprise Features**: Multi-tenant dashboards, service integration, AI assistant

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

- **sshx-core** (v0.4.1): Shared protobuf definitions, encryption utilities, ID management, and common types
- **sshx-server** (v0.4.1): Hybrid gRPC/HTTP server with WebSocket handling, Redis state management, and dashboard system
- **sshx** (v0.4.1): CLI client with service integration, dashboard registration, and intelligent connection fallback
- **sshx-term** (v0.1.0): SSH-like terminal client for direct session access with TUI selector

### Go Client Structure (`/sshx-go/`)

- **Alternative client implementation** with extended architecture support (MIPS, RISC-V, s390x)
- **Systemd service integration** with automated installation and management
- **Static binary builds** with CGO disabled for easy deployment
- **Protocol compatibility** with automatic gRPC → WebSocket fallback

### Frontend Structure (`/src/`)

- **routes/**: SvelteKit pages (/, /s/[id], /d/[key]) and API routes
- **lib/**: Core components, stores, and utilities
- **lib/term/**: Terminal emulation with custom xterm.js fork (@ekzhang/sshx-xterm)
- **lib/encryption.ts**: Client-side encryption matching Rust implementation (Argon2id + AES-CTR)
- **lib/ai/**: AI integration services
  - **gemini.ts**: Google Gemini API integration with model selection
  - **openrouter.ts**: OpenRouter API for 100+ LLM models
  - **contextManager.ts**: Intelligent conversation context management with compression
- **lib/ui/dashboard/**: Multi-tenant dashboard components with real-time updates
- **lib/themes.ts**: 100+ terminal color themes with light/dark UI modes
- **lib/api.ts**: Dashboard API client with pagination and authentication

### Key Architectural Patterns

1. **Dual Protocol Architecture**: Native gRPC for CLI clients with automatic WebSocket fallback
2. **State Management**: Redis-backed session persistence with CBOR serialization and zstd compression
3. **Protocol**: Comprehensive protobuf definitions in `crates/sshx-core/proto/` with CLI WebSocket extensions
4. **Terminal Multiplexing**: Up to 14 concurrent terminals per session (WebGL context limitations)
5. **Multi-Tenant Dashboards**: Isolated monitoring namespaces at `/d/{key}` URLs with auto-cleanup
6. **End-to-End Encryption**: Argon2id key derivation + AES-128-CTR with public salt strategy
7. **Service Integration**: systemd service management with automated installation/configuration
8. **AI Integration**: Context-aware assistant with dual provider support and intelligent compression
9. **Static URLs**: Sessions accessible via `/s/{session-id}#{secret}` with cryptographically secure IDs

## Development Notes

### Core Requirements
- **Redis 7.2+**: Required for server (use `docker compose up -d` - runs on port 12601)
- **Rust 1.70+**: With cross-compilation targets for multi-platform builds
- **Node.js 18+**: With npm for frontend development
- **Protobuf Compiler**: v29.2+ for gRPC code generation
- **Docker**: For containerized development and deployment testing

### Development Workflow
- **Frontend**: Proxies API requests to `http://[::1]:8051` in development mode
- **Terminal**: Uses custom xterm.js fork: `@ekzhang/sshx-xterm` with WebGL acceleration
- **Build Artifacts**: `target/` (Rust), `build/` (SvelteKit), binary caching for faster builds
- **Process Management**: Use `mprocs` for parallel development (server, client, web frontend)
- **Release Builds**: `scripts/release.sh` for comprehensive cross-platform compilation

### Advanced Features
- **AI Integration**: Dual provider support (Gemini, OpenRouter) with context management
- **Dashboard System**: Multi-tenant monitoring with pagination, search, and real-time updates  
- **Theme System**: 100+ terminal color schemes with automatic light/dark UI detection
- **Service Integration**: systemd service installation and management (Linux)
- **Cross-Platform**: 15+ architectures via Rust + Go client implementations

### Terminal Limits

The application is limited to **14 concurrent terminals** due to browser WebGL context limitations:

- Each terminal uses xterm.js with WebGL addon for GPU-accelerated rendering
- Browsers limit WebGL contexts to ~16 concurrent contexts
- Beyond this limit, browsers destroy oldest contexts, causing terminals to become unrenderable
- Users see an "upset emoticon" in affected terminals when this occurs
- The 14-terminal limit (with 2-context safety margin) prevents this issue

To support more terminals, you would need to:
- Disable WebGL addon (reduces performance but removes limit)
- Implement hybrid rendering (WebGL for active terminals, canvas for others)
- Use context pooling/recycling for inactive terminals

- For the websocket transport, the server is expecting/sending JSON arrays, not base64 strings.

## Testing and Quality Assurance

### Rust Testing
```bash
# Run all tests
cargo test

# Test specific crate
cargo test -p sshx-server
cargo test -p sshx-core

# Run with output
cargo test -- --nocapture

# Format Rust code
cargo fmt
```

### Frontend Testing
```bash
# Type checking
npm run check

# Linting and formatting
npm run lint
npm run format

# Build verification
npm run build
```

### Integration Testing
```bash
# Start development environment
mprocs  # Runs server, client, and frontend in parallel

# Test connection fallback
# 1. Start server: cargo run --bin sshx-server
# 2. Test gRPC: cargo run --bin sshx -- --server http://localhost:8051
# 3. Test WebSocket fallback: Use browser at http://localhost:5173
```

## Common Development Patterns

### Adding New Features
1. **Protocol Changes**: Update `crates/sshx-core/proto/sshx.proto` first
2. **Server Implementation**: Add handlers in `crates/sshx-server/src/`
3. **Client Implementation**: Update both Rust and Go clients if needed
4. **Frontend Integration**: Add UI components in `src/lib/`
5. **Testing**: Add tests for both backend and frontend components

### AI Integration Development
- **Provider Integration**: Add new providers in `src/lib/ai/`
- **Context Management**: Extend `contextManager.ts` for new features
- **Model Configuration**: Update model lists and capabilities
- **Testing**: Test with actual API keys in development environment

### Dashboard Development
- **Backend**: Extend dashboard API in `sshx-server/src/dashboard.rs`
- **Frontend**: Update components in `src/lib/ui/dashboard/`
- **Real-time Updates**: Ensure WebSocket message handling supports new features
- **Pagination**: Consider server-side pagination for scalability

### Theme Development
- **Terminal Themes**: Add to `src/lib/themes.ts` (10,000+ line file)
- **UI Themes**: Update CSS variables in theme configuration
- **Color Consistency**: Ensure theme works across terminal and UI components

## Debugging Tips

### Common Issues
1. **Redis Connection**: Ensure Redis is running on port 12601
2. **Protocol Mismatch**: Ensure client and server versions match
3. **WebGL Contexts**: Monitor browser console for WebGL context warnings
4. **AI Rate Limits**: Check API key configuration and usage quotas
5. **Dashboard Auth**: Verify `SSHX_DASHBOARD_KEY` environment variable

### Development Debugging
```bash
# Enable debug logging
RUST_LOG=debug cargo run --bin sshx-server

# Frontend debugging
# Open browser dev tools, check Network tab for WebSocket messages

# Test WebSocket directly
# Use browser dev tools or WebSocket testing tools
```

## Security Considerations in Development

- **API Keys**: Never commit AI provider API keys to the repository
- **Dashboard Keys**: Use strong randomly generated keys for development
- **Encryption**: Test encryption/decryption with various key lengths
- **Network Security**: Test both HTTP and HTTPS deployments
- **Authentication**: Verify dashboard authentication works correctly

## Performance Considerations

- **Terminal Rendering**: Monitor WebGL context usage (max 14 terminals)
- **Memory Usage**: Watch for memory leaks in long-running sessions
- **Network Efficiency**: Optimize message frequency and size
- **Redis Performance**: Monitor session state storage efficiency
- **AI Context**: Manage conversation context to avoid token limits