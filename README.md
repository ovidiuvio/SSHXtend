# SSHXtend 🚀

[![Version](https://img.shields.io/badge/version-v0.4.0--extended-blue)](https://github.com/ovidiuvio/sshx)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-available-brightgreen)](https://github.com/orgs/ovidiuvio/packages)
[![Enhanced with Claude](https://img.shields.io/badge/Enhanced_with-Claude_Sonnet-ff6b35?logo=anthropic&logoColor=white)](https://claude.ai)

**SSHXtend** is an enhanced fork of [sshx](https://github.com/ekzhang/sshx) - a secure, web-based collaborative terminal with powerful new features for teams and enterprises.

> ⚠️ **Important**: SSHXtend requires custom-built binaries and is **NOT compatible** with the original sshx installation. You must build from this repository or use our Docker images.

> 🤖 **Enhanced with Claude Sonnet**: All major features and improvements in this fork were developed collaboratively with Anthropic's Claude Sonnet AI assistant, demonstrating the power of AI-assisted software development.

![SSHXtend Terminal](https://i.imgur.com/Q3qKAHW.png)

## ✨ What's New in SSHXtend

This fork extends the original sshx with enterprise-grade features and enhanced usability:

### 🔗 **Secure Random Sessions**
Automatically generates cryptographically secure session URLs for maximum security:
```bash
sshx --server http://localhost:8051
# ➜ Link: http://localhost:8051/s/kM9pL2nQ7v#R4sT6wXyZ1aB3cD5e
```

### 📊 **Multi-Dashboard System**
Web-based dashboards to monitor and manage sessions by groups:
- Multiple isolated dashboards at `/d/<key>` URLs
- Each dashboard tracks its own set of sessions
- Search and filter within each dashboard
- Pagination support for large deployments
- Session metadata and user count
- Auto-refresh every 10 seconds with live updates
- Automatic cleanup of empty dashboards after 24 hours

**🔑 Dashboard Usage:**
```bash
# Create a new dashboard when connecting
sshx --dashboard
# Output: Dashboard URL: https://sshx.stream/d/xK9mP2nQ7vR4sT6w

# Join an existing dashboard
sshx --dashboard xK9mP2nQ7vR4sT6w
```

<img src="/static/images/sshx-dashboard.png" alt="Session Dashboard" width="700">

*Real-time session monitoring with overview statistics, search functionality, and detailed session information*

### 🤖 **AI-Powered Terminal Assistant** (NEW)
Integrated AI assistant for intelligent terminal help and troubleshooting:

**✨ AI Features:**
- **Dual AI Provider Support**: Choose between Google Gemini or OpenRouter (supports 100+ models)
- **Smart Context Management**: Automatically manages conversation context with token tracking
- **Persistent Conversations**: Continue conversations across terminal sessions
- **Intelligent Terminal Analysis**: Select any terminal output for instant AI analysis
- **Natural Conversation Flow**: Maintains chronological context of all interactions

**🎯 How It Works:**
1. **Select text** in terminal → Click the sparkle button for instant AI help
2. **Continue conversations**: Blue sparkle button resumes previous discussion
3. **Start fresh**: Orange sparkle button begins new conversation
4. **Keyboard shortcuts**: `Ctrl+Shift+A` (or `Cmd+Shift+A` on Mac) for quick access

**⚙️ Configuration:**
```javascript
// In Settings panel, configure your AI provider:
{
  "aiProvider": "gemini",           // or "openrouter"
  "geminiApiKey": "your-api-key",
  "openRouterApiKey": "your-key",
  "openRouterModel": "gpt-4",       // 100+ models available
  "aiContextLength": 128000,        // Custom context window
  "aiAutoCompress": true            // Auto-compress long conversations
}
```

**💬 AI Mode Commands:**
- `/exit` - Exit AI mode (conversation preserved)
- `/new` - Start a fresh conversation
- `Enter` with empty input - Quick exit

**🔄 Context Management:**
- Visual indicators show context usage percentage
- Automatic compression when nearing token limits
- Preserves all terminal selections and conversation history
- Smart context ordering maintains natural conversation flow

### 🎨 **Advanced UI Settings & Customization**
Comprehensive settings panel with extensive customization options:

**🎨 Theme System:**
- **30+ built-in color themes** with light/dark variants (VS Code, Dracula, Gruvbox, Solarized, Tokyo Night, Catppuccin, and more)
- Auto UI theme detection (follows system light/dark preference)
- Manual light/dark/auto UI theme switching
- Real-time theme updates without terminal restart

**🔤 Typography & Display:**
- 8 professional monospace fonts (Fira Code, JetBrains Mono, Source Code Pro, etc.)
- Font size adjustment (8-32px)
- **Font weight customization** for regular and bold text
- **Zoom functionality** with `Ctrl+/Cmd+` plus/minus for quick size adjustments
- Persistent font preferences across sessions

<img src="/static/images/sshx-settings.png" alt="Terminal Settings Panel" width="600">

*Comprehensive settings panel with theme selection, font customization, and terminal configuration options*

### 🖥️ **Enhanced Terminal Experience** 
- **14 concurrent terminals** (WebGL context limit: ~16 max, 2 safety margin)
- **Download terminal session logs** as text files with one-click export
- Customizable terminal fonts and display settings
- **Copy-on-select**: Automatically copy selected text to clipboard
- **Middle-click paste**: Quick paste functionality for Linux/Unix users
- **Auto-arrange layout**: Organize multiple terminals with one click
- **Smart toolbar positioning**: Top or bottom placement options
- **Connection status indicators**: Visual feedback for terminal state  

<img src="/static/images/sshx-light.png" alt="Light Theme Terminal" width="800">

*Multiple terminal windows in light theme showing system monitoring and process management*

**📥 Session Log Export:**
Export your terminal session content for documentation, debugging, or sharing:

<img src="/static/images/sshx-download-log.png" alt="Download Terminal Logs" width="600">

*One-click download functionality for terminal session logs and command history*

### 🐧 **Linux Service Integration**
Easy service management for production deployments:
```bash
# Install as system service
sshx --service install

# Manage the service
sshx --service start|stop|uninstall
```

### 🐹 **Go Client Alternative**
Additional Go-based client implementation in `sshx-go/` with enhanced capabilities:
- **Extended Architecture Support**: MIPS, MIPS64, RISC-V, s390x architectures
- **Systemd Service Integration**: Built-in service installation and management
- **Transport Compatibility**: Automatic gRPC → WebSocket fallback matching Rust client
- **Static Binaries**: CGO-disabled builds for easy deployment
- **Alternative for Exotic Platforms**: Supports architectures not covered by Rust toolchain

### 🖥️ **SSH-Like Terminal Client** (NEW)
Introducing `sshx-term` - a dedicated terminal client for direct session access:
- **SSH-Like Experience**: Connect directly to existing sessions from command line
- **Interactive Terminal Selection**: TUI selector for multi-terminal sessions
- **Smart Connection Logic**: Automatic terminal detection and creation
- **Raw Terminal Mode**: Full terminal capabilities with proper signal handling
- **WebSocket Protocol**: Uses same secure protocol as web frontend

### 🐳 **Ready-to-Use Docker Images**
Pre-built Docker images for instant deployment:
```bash
# Server image (multi-stage build: Rust + SvelteKit frontend)
docker pull ghcr.io/ovidiuvio/sshxtend-server:latest

# Production deployment with Redis
docker compose up -d  # Includes Redis 7.2 on port 12601
```

## 🚀 Quick Start

> 🔴 **COMPATIBILITY WARNING**: The standard `curl -sSf https://sshx.io/get | sh` install will **NOT work** with this fork. Use the methods below instead.

### Using Docker (Recommended)

1. **Start the server:**
```bash
docker run -d -p 8051:8051 ghcr.io/ovidiuvio/sshxtend-server:latest
```

2. **Connect with the client:**
TODO
```

3. **Access the dashboard:** Open `http://localhost:8051` in your browser
4. **Customize your experience:** Click the gear icon in the terminal toolbar to access settings
5. **Enable AI assistant:** Add your API key in Settings and start using AI help with `Ctrl+Shift+A`

### Installation Options

#### **Option 1: Automated Installation (Recommended)**
```bash
# Multi-platform installer (Linux, macOS, FreeBSD)
curl -sSf https://raw.githubusercontent.com/ovidiuvio/sshx/main/static/get | sh

# Installation modes:
curl -sSf ... | sh -s install   # System-wide installation (/usr/local/bin)
curl -sSf ... | sh -s download  # Download to current directory
curl -sSf ... | sh -s run       # Download and run temporarily
```

#### **Option 2: Build from Source**
```bash
# ✅ Build SSHXtend from source
git clone https://github.com/ovidiuvio/sshx
cd sshx
cargo install --path crates/sshx          # CLI client
cargo install --path crates/sshx-server   # Server
cargo install --path crates/sshx-term     # SSH-like terminal client
```

#### **Option 3: Go Client (Extended Architecture Support)**
```bash
# For MIPS, RISC-V, s390x, and other exotic architectures
git clone https://github.com/ovidiuvio/sshx
cd sshx/sshx-go
go build -ldflags="-s -w" -o sshxtend-go .
```

#### **Option 4: GitHub Releases**
Download pre-built binaries from [GitHub Releases](https://github.com/ovidiuvio/sshx/releases) for:
- **Linux**: x86_64, aarch64, armv6l, armv7l (musl-based)
- **macOS**: Intel (x86_64) and Apple Silicon (aarch64)
- **Windows**: x86_64, i686, aarch64
- **FreeBSD**: x86_64
- **Exotic**: MIPS, MIPS64, RISC-V, s390x (Go client)

## 📖 Usage Examples

### AI-Assisted Troubleshooting
```bash
# When you encounter an error:
# 1. Select the error text in terminal
# 2. Press Ctrl+Shift+A (or click sparkle button)
# 3. AI analyzes and provides solutions

# Continue conversation with more context:
# 1. Run additional commands
# 2. Select new output
# 3. Click blue sparkle to add to existing conversation
# 4. AI maintains full context of the debugging session
```

### Secure Team Collaboration
```bash
sshx --server http://localhost:8051
# Generates unique secure URL for team sharing
# Enhanced security with random session IDs and encryption keys
```

### Monitored Production Session
```bash
# Start server with dashboard monitoring
sshx-server --dashboard-key "production-monitor-2024"

# Connect client with dashboard registration
sshx --dashboard --server http://your-server:8051
# Session appears in dashboard for real-time monitoring
```

### Advanced Configuration
```bash
# Full-featured production setup
export SSHX_DASHBOARD_KEY="$(openssl rand -base64 32)"
sshx-server --listen 0.0.0.0:8051

# Client with dashboard monitoring
sshx --dashboard --server https://your-domain.com
# Automatically generates secure URLs with dashboard monitoring
```

### SSH-Like Terminal Access
```bash
# Connect to existing session with sshx-term
sshx-term https://your-domain.com/s/kM9pL2nQ7v#R4sT6wXyZ1aB3cD5e

# Interactive terminal selection for multi-terminal sessions
sshx-term --new https://your-domain.com/s/session-id#secret
# Creates new terminal if --new flag provided

# Direct terminal connection (single terminal sessions)
sshx-term wss://your-domain.com/s/session-id#secret
# Supports multiple URL schemes: http, https, ws, wss
```

## ⌨️ Keyboard Shortcuts

### Terminal Controls
- **`Ctrl/Cmd + Plus (+)`**: Zoom in (increase font size)
- **`Ctrl/Cmd + Minus (-)`**: Zoom out (decrease font size)
- **`Ctrl/Cmd + 0`**: Reset zoom to default
- **`Ctrl/Cmd + Shift + A`**: Activate AI assistant with selected text
- **`Middle Click`**: Paste from clipboard (when enabled)

### AI Mode Commands
- **`Enter`** (empty input): Exit AI mode
- **`/exit`**: Exit AI mode (conversation preserved)
- **`/new`**: Start new conversation
- **`Ctrl + C`**: Exit AI mode immediately

### Service Installation
```bash
# Install as service
sudo sshx --service install

# Configure in /etc/systemd/system/sshx.service
sudo systemctl enable sshx
sudo systemctl start sshx
```
## 🎨 Visual Interface Features

**🎭 Theme System:**
- Seamless light/dark mode switching with system preference detection

**📊 Dashboard Interface:**
- Clean, responsive design with real-time statistics cards
- Searchable session table with pagination for large deployments
- Live refresh every 10 seconds with visual indicators
- Authentication prompt with branded interface

**⚙️ Settings Experience:**
- Comprehensive settings panel with intuitive organization
- Live preview of font and theme changes
- Persistent configuration across browser sessions
- Professional typography with 8 curated monospace fonts
- **AI configuration section** for API keys and model selection
- **Terminal behavior settings**: Copy-on-select, middle-click paste
- **Toolbar customization**: Position and layout preferences
- **Advanced display options**: Font weights, zoom controls

## 🏗️ Architecture Overview

SSHXtend uses a sophisticated hybrid architecture with enterprise-grade features:

### **Rust Workspace Structure**
- **`sshx-core`** (v0.4.1): Shared protobuf definitions, encryption utilities, ID management
- **`sshx-server`** (v0.4.1): Hybrid gRPC/HTTP server with WebSocket and dashboard management
- **`sshx`** (v0.4.1): CLI client with service integration and dashboard registration
- **`sshx-term`** (v0.1.0): SSH-like terminal client for direct session access

### **Frontend Architecture**
- **SvelteKit**: TypeScript SPA with static site generation
- **AI Integration**: Dual provider support (Google Gemini, OpenRouter)
- **Theme System**: 100+ terminal themes with light/dark UI modes
- **Real-time Communication**: WebSocket with JSON protocol for web clients

### **Protocol Design**
- **Dual Protocol Support**: Native gRPC for CLI, WebSocket for web/fallback
- **Intelligent Fallback**: Automatic gRPC → WebSocket with connectivity testing
- **End-to-End Encryption**: Argon2id + AES-128-CTR with public salt strategy
- **Session Persistence**: Redis-backed state with CBOR serialization

## 🔧 Development

### Prerequisites
- **Rust**: 1.70+ with cross-compilation targets
- **Node.js**: 18+ with npm/pnpm for frontend
- **Redis**: 7.2+ for session state management
- **Docker**: For containerized Redis and deployment testing
- **Protobuf Compiler**: v29.2+ for gRPC code generation

### Setup
```bash
# Clone the SSHXtend repository (required for extended features)
git clone https://github.com/ovidiuvio/sshx
cd sshx

# Start Redis (required for server)
docker compose up -d

# Install frontend dependencies  
npm install

# Build Rust components
cargo build

# Run everything in parallel (recommended)
mprocs
```

### Available Commands
```bash
# Frontend
npm run dev        # Development server (port 5173)
npm run build      # Production build
npm run check      # Type checking
npm run lint       # Linting
npm run format     # Code formatting

# Backend
cargo build                    # Build all crates
cargo run --bin sshx-server   # Run server (port 8051)
cargo run --bin sshx          # Run client
cargo test                    # Run tests
cargo fmt                     # Format code
```

## 🐳 Docker Images

Pre-built images are available on GitHub Container Registry:

- **Server**: `ghcr.io/ovidiuvio/sshx-server:latest`

These images are built from the `dev` branch and include all the extended features.

## 📚 API & Dashboard

Access the web dashboard at `http://your-server:8051/` to:
- View all active sessions with real-time stats
- Monitor user activity and session metadata  
- Manage session lifecycle with pagination
- **Download session logs** - Export terminal content with the download button (see screenshots above)

### 🔐 Dashboard Security

**Enable Password Protection:**
```bash
# Via command line
sshx-server --dashboard-key "your-strong-secret-key"

# Via environment variable (recommended for production)
export SSHX_DASHBOARD_KEY="your-strong-secret-key"
sshx-server
```

**Client Registration with Dashboard:**
```bash
# Register session with dashboard for monitoring
sshx --dashboard --server http://your-server:8051
```

**Authentication Behavior:**
- If no dashboard key is configured: **Open access** (no authentication required)
- If dashboard key is set: **Protected access** (authentication required)
- Dashboard key is stored in browser localStorage after successful login
- API endpoints return 401 Unauthorized for invalid/missing keys

## ⚠️ Security Considerations

### 🚨 **CRITICAL: Public Deployment Warning**

**⚠️ DANGER: If you run SSHXtend publicly without proper security measures, you are potentially exposing terminal access to the entire internet!**

**Default Behavior Risks:**
- **Dashboard**: If no `--dashboard-key` is set, the dashboard at `/` is **publicly accessible**
- **Session Access**: Anyone with a session URL can join and execute commands
- **Terminal Sharing**: All session participants have full terminal access
- **No User Authentication**: The application has no built-in user management system

### 🛡️ **Production Security Best Practices**

**Essential Security Measures:**
```bash
# 1. ALWAYS set a strong dashboard key for public deployments
export SSHX_DASHBOARD_KEY="$(openssl rand -base64 32)"
sshx-server

# 2. Use HTTPS/TLS (required for secure communication)
# Deploy behind a reverse proxy like nginx with SSL

# 3. Use firewall rules to restrict access
ufw allow from 192.168.1.0/24 to any port 8051  # Only allow internal network
```

**Additional Recommendations:**
- **Network Isolation**: Deploy on internal networks only when possible
- **VPN Access**: Require VPN connection for external access
- **Secure Session Management**: All sessions use cryptographically secure random IDs and encryption keys
- **Regular Key Rotation**: Change dashboard keys periodically
- **Access Logging**: Monitor server logs for unauthorized access attempts
- **Principle of Least Privilege**: Only share session URLs with trusted users

### 🔍 **Security Limitations**

**Current Authentication Model:**
- **Single Dashboard Key**: No individual user accounts or role-based access
- **Shared Terminal Access**: All session participants have equal access
- **Client-Side Storage**: Dashboard key stored in browser localStorage (vulnerable to XSS)
- **No Session Timeouts**: Dashboard authentication persists until manually cleared
- **No Rate Limiting**: No built-in protection against brute force attacks

**Mitigation Strategies:**
- All sessions automatically use cryptographically secure random IDs and encryption keys
- Regularly rotate dashboard keys in high-security environments
- Consider implementing additional authentication layers (reverse proxy auth)
- Monitor dashboard access patterns for suspicious activity
- Use dedicated instances for different security contexts

## 🤖 AI-Assisted Development

This project showcases **AI-assisted software development** using Claude Sonnet:

- **🧠 Feature Design**: Claude Sonnet helped architect and design new features
- **💻 Code Implementation**: Major functionality developed collaboratively with AI
- **📝 Documentation**: This README and technical documentation written with Claude
- **🔍 Code Analysis**: Deep codebase analysis and understanding enhanced by AI
- **🚀 Rapid Prototyping**: Faster development cycles through AI assistance

**Result**: Enhanced productivity and code quality through human-AI collaboration.

## 🤝 Contributing

This is a community-driven fork! Contributions welcome:

1. Fork the repository
2. Create a feature branch
3. Make your changes  
4. Add tests if applicable
5. Submit a pull request

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- **🤖 [Claude Sonnet](https://claude.ai)** by Anthropic - AI assistant that collaboratively developed all major enhancements in this fork
- Original [sshx](https://github.com/ekzhang/sshx) by Eric Zhang
- Built with Rust, SvelteKit, and WebRTC
- Inspired by tmux, screen, and Mosh

## 🔗 Links

- **Original Project**: [github.com/ekzhang/sshx](https://github.com/ekzhang/sshx)
- **Docker Images**: [GitHub Packages](https://github.com/orgs/ovidiuvio/packages)
- **Issues**: Report bugs or request features in our Issues tab

---

**Ready to supercharge your terminal collaboration?** 🚀 Get started with SSHXtend today!
