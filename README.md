# SSHXtend 🚀

[![Version](https://img.shields.io/badge/version-v0.3.1--extended-blue)](https://github.com/ovidiuvio/sshx)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-available-brightgreen)](https://github.com/orgs/ovidiuvio/packages)
[![Enhanced with Claude](https://img.shields.io/badge/Enhanced_with-Claude_Sonnet-ff6b35?logo=anthropic&logoColor=white)](https://claude.ai)

**SSHXtend** is an enhanced fork of [sshx](https://github.com/ekzhang/sshx) - a secure, web-based collaborative terminal with powerful new features for teams and enterprises.

> ⚠️ **Important**: SSHXtend requires custom-built binaries and is **NOT compatible** with the original sshx installation. You must build from this repository or use our Docker images.

> 🤖 **Enhanced with Claude Sonnet**: All major features and improvements in this fork were developed collaboratively with Anthropic's Claude Sonnet AI assistant, demonstrating the power of AI-assisted software development.

![SSHXtend Terminal](https://i.imgur.com/Q3qKAHW.png)

## ✨ What's New in SSHXtend

This fork extends the original sshx with enterprise-grade features and enhanced usability:

### 🔗 **Static URLs & Custom Sessions**
Create predictable, bookmarkable session URLs with custom session IDs and secrets:
```bash
sshx --server http://localhost:8051 --secret "myteam-secret" --session-id "dev-env"
# ➜ Link: http://localhost:8051/s/dev-env#myteam-secret
```

### 📊 **Protected Session Dashboard**
Web-based dashboard to monitor and manage all active sessions:
- View all running sessions with real-time stats
- Pagination support for large deployments
- **🔐 Optional password protection** with dashboard key authentication
- Session metadata and user count
- Auto-refresh every 10 seconds with live updates

**🔑 Dashboard Authentication:**
```bash
# Protect dashboard with a key
sshx-server --dashboard-key "your-secret-key"

# Or via environment variable
export SSHX_DASHBOARD_KEY="your-secret-key"
sshx-server
```

<img src="/static/images/sshx-dashboard.png" alt="Session Dashboard" width="700">

*Real-time session monitoring with overview statistics, search functionality, and detailed session information*

### 🎨 **Advanced UI Settings & Customization**
Comprehensive settings panel with extensive customization options:

**🎨 Theme System:**
- 24 built-in color themes with light/dark variants (VS Code, Dracula, Gruvbox, Solarized, etc.)
- Auto UI theme detection (follows system light/dark preference)
- Manual light/dark/auto UI theme switching
- Real-time theme updates without terminal restart

**🔤 Typography & Display:**
- 8 professional monospace fonts (Fira Code, JetBrains Mono, Source Code Pro, etc.)
- Font size adjustment (8-32px)
- Configurable scrollback buffer (terminal history lines)
- Enhanced UI with modern design

**⚙️ User Experience:**
- Persistent settings via browser localStorage
- Real-time updates across all terminals
- Customizable display name for collaboration
- Settings accessible via toolbar gear icon

<img src="/static/images/sshx-settings.png" alt="Terminal Settings Panel" width="600">

*Comprehensive settings panel with theme selection, font customization, and terminal configuration options*

### 🖥️ **Enhanced Terminal Experience** 
- **14 concurrent terminals** (WebGL context limit: ~16 max, 2 safety margin)
- **Download terminal session logs** as text files with one-click export
- **Multi-window interface** with tabbed terminal management
- Customizable terminal fonts and display settings  
- WebGL-accelerated rendering for smooth performance
- Graceful error handling when terminal limits are reached

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
Additional Go-based client implementation in `sshx-go/` for platforms where the Rust client isn't fully supported or available:
- Broader platform compatibility
- Easier deployment in Go-based environments
- Alternative for systems with Rust compilation issues

### 🐳 **Ready-to-Use Docker Images**
Pre-built Docker images for instant deployment:
```bash
# Server image
docker pull ghcr.io/ovidiuvio/sshx-server:latest

# Client image  
docker pull ghcr.io/ovidiuvio/sshx:latest
```

## 🚀 Quick Start

> 🔴 **COMPATIBILITY WARNING**: The standard `curl -sSf https://sshx.io/get | sh` install will **NOT work** with this fork. Use the methods below instead.

### Using Docker (Recommended)

1. **Start the server:**
```bash
docker run -d -p 8051:8051 ghcr.io/ovidiuvio/sshx-server:latest
```

2. **Connect with the client:**
```bash
docker run -it ghcr.io/ovidiuvio/sshx:latest --server http://localhost:8051
```

3. **Access the dashboard:** Open `http://localhost:8051` in your browser
4. **Customize your experience:** Click the gear icon in the terminal toolbar to access settings

### Traditional Installation

**🔴 CRITICAL:** The original sshx binaries are **completely incompatible** with SSHXtend due to protocol and feature differences. You **MUST** build from this repository:

```bash
# ✅ REQUIRED: Build SSHXtend from source
git clone https://github.com/ovidiuvio/sshx
cd sshx
cargo install --path crates/sshx
cargo install --path crates/sshx-server

# ❌ DO NOT USE: Original sshx (will fail to connect)
# curl -sSf https://sshx.io/get | sh  # This will NOT work with SSHXtend servers
```

**Why incompatible?**
- Enhanced protocol features (dashboard registration, settings sync)
- Additional command-line arguments (`--dashboard`, `--session-id`, `--service`)
- Modified server endpoints and authentication
- Extended feature set not present in original client

## 🏗️ Core Features

All the powerful features from the original sshx, plus our enhancements:

- **🔐 End-to-end encryption** with Argon2 + AES
- **🌐 Collaborative terminals** with real-time cursor sharing  
- **📱 Responsive design** - resize, move, zoom on infinite canvas
- **🔄 Auto-reconnection** with real-time latency estimates
- **⚡ Predictive echo** for faster local editing (like Mosh)
- **🌍 Global mesh network** for optimal performance

## 📖 Usage Examples

### Basic Session
```bash
sshx
# Creates a session with random URL
# Access settings via gear icon to customize theme and appearance
```

### Custom Session for Teams
```bash
sshx --session-id "team-standup" --secret "standup-2024"
# Always creates the same shareable URL
# Perfect for recurring team collaboration
```

### Monitored Production Session
```bash
# Start server with dashboard monitoring
sshx-server --dashboard-key "production-monitor-2024"

# Connect client with dashboard registration
sshx --dashboard --session-id "prod-maintenance" --server http://your-server:8051
# Session appears in dashboard for real-time monitoring
```

### Advanced Configuration
```bash
# Full-featured production setup
export SSHX_DASHBOARD_KEY="$(openssl rand -base64 32)"
sshx-server --listen 0.0.0.0:8051

# Client with custom session and monitoring
sshx --dashboard \
     --session-id "dev-team-$(date +%Y%m%d)" \
     --secret "team-secret-key" \
     --server https://your-domain.com
# Combines static URLs + dashboard monitoring + security
```

### Service Installation
```bash
# Install as service
sudo sshx --service install

# Configure in /etc/systemd/system/sshx.service
sudo systemctl enable sshx
sudo systemctl start sshx
```

### Docker Compose
```yaml
version: '3.8'
services:
  sshx-server:
    image: ghcr.io/ovidiuvio/sshx-server:latest
    ports:
      - "8051:8051"
    environment:
      - REDIS_URL=redis://redis:6379
      - SSHX_DASHBOARD_KEY=your-strong-dashboard-key
    volumes:
      - ./data:/data  # Optional: persist data
  
  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
      
volumes:
  redis_data:
```

## 🎨 Visual Interface Features

SSHXtend provides a modern, highly customizable interface designed for professional terminal collaboration:

**🎭 Theme System:**
- Seamless light/dark mode switching with system preference detection
- 24 professionally crafted color palettes (VS Code, Dracula, Gruvbox, Solarized, etc.)
- Real-time theme switching without session interruption

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

**📱 Responsive Design:**
- Optimized for desktop collaboration and mobile monitoring
- Scalable interface components that adapt to screen size
- Touch-friendly controls for mobile dashboard access

## 🔧 Development

### Prerequisites
- Rust 1.70+
- Node.js 18+
- Redis (for server)
- Docker & Docker Compose

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
- **Client**: `ghcr.io/ovidiuvio/sshx:latest`

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
- **Strong Session Secrets**: Use long, random session secrets (>20 characters)
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
- Use strong, unique session secrets for sensitive work
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