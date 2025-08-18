use std::process::ExitCode;

use ansi_term::Color::{Cyan, Fixed, Green};
use anyhow::Result;
use clap::Parser;
use serde::{Deserialize, Serialize};
use sshx::{controller::Controller, runner::Runner, service, terminal::get_default_shell, connection::{connect_with_fallback, ConnectionConfig, verbose_config}};
use tokio::signal;
use tracing::{error, warn};

/// A secure web-based, collaborative terminal.
#[derive(Parser, Debug)]
#[clap(
    author,
    version,
    about = "
SSHX Terminal Sharing

Connection:
  Automatically tries gRPC first, then WebSocket fallback for compatibility
  with proxies and firewalls (e.g., Cloudflare tunnels).

Service Management:
  --service install    Install and enable systemd service with current configuration
  --service uninstall  Remove systemd service and binary
  --service status     Check service status
  --service start      Start service
  --service stop       Stop service

Examples:
  sshx --server https://your-server.com --dashboard --service install
  sshx --shell /bin/bash --name server1 --service install
  sshx --verbose       Show connection method and detailed debugging info
"
)]
struct Args {
    /// Address of the remote sshx server.
    #[clap(long, default_value = "https://sshx.io", env = "SSHX_SERVER")]
    server: String,

    /// Local shell command to run in the terminal.
    #[clap(long)]
    shell: Option<String>,

    /// Quiet mode, only prints the URL to stdout.
    #[clap(short, long)]
    quiet: bool,

    /// Session name displayed in the title (defaults to user@hostname).
    #[clap(long)]
    name: Option<String>,

    /// Enable read-only access mode - generates separate URLs for viewers and
    /// editors.
    #[clap(long)]
    enable_readers: bool,

    /// Enable verbose output showing connection details and fallback attempts.
    #[clap(short, long, env = "SSHX_VERBOSE")]
    verbose: bool,

    /// Service management (install|uninstall|status|start|stop)
    #[clap(long, value_parser = ["install", "uninstall", "status", "start", "stop"])]
    service: Option<String>,

    /// Register this session with a dashboard.
    /// If no key provided, generates a new dashboard.
    /// If key provided, joins existing dashboard.
    #[clap(long, value_name = "KEY")]
    dashboard: Option<Option<String>>,
}

/// Dashboard registration request payload
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct RegisterDashboardRequest {
    session_name: String,
    url: String,
    write_url: Option<String>,
    display_name: String,
    dashboard_key: Option<String>,
}

/// Dashboard registration response
#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct RegisterDashboardResponse {
    dashboard_url: String,
}

/// Extract relative URL from full URL (removes domain for reverse proxy compatibility)
fn make_relative_url(full_url: &str) -> String {
    if let Ok(url) = url::Url::parse(full_url) {
        // Return path + query + fragment for reverse proxy compatibility
        let mut relative = url.path().to_string();
        if let Some(query) = url.query() {
            relative.push('?');
            relative.push_str(query);
        }
        if let Some(fragment) = url.fragment() {
            relative.push('#');
            relative.push_str(fragment);
        }
        relative
    } else {
        // If parsing fails, assume it's already relative
        full_url.to_string()
    }
}

/// Register session with the dashboard
async fn register_with_dashboard(
    server_url: &str,
    controller: &Controller,
    display_name: &str,
    dashboard_key: Option<String>,
) -> Result<()> {
    let dashboard_url = format!("{}/api/dashboards/register", server_url);

    let request = RegisterDashboardRequest {
        session_name: controller.name().to_string(),
        url: make_relative_url(controller.url()),
        write_url: controller.write_url().map(make_relative_url),
        display_name: display_name.to_string(),
        dashboard_key,
    };

    let client = reqwest::Client::new();
    let response = client.post(&dashboard_url).json(&request).send().await?;

    if response.status().is_success() {
        let response_data: RegisterDashboardResponse = response.json().await?;
        println!("\n  {} Session registered to dashboard", Green.paint("✓"));
        println!(
            "  {} Dashboard URL: {}",
            Green.paint("➜"),
            Cyan.underline().paint(&response_data.dashboard_url)
        );
    } else {
        warn!("Failed to register with dashboard: {}", response.status());
    }

    Ok(())
}

fn print_greeting(shell: &str, controller: &Controller) {
    let version_str = match option_env!("CARGO_PKG_VERSION") {
        Some(version) => format!("v{version}"),
        None => String::from("[dev]"),
    };
    if let Some(write_url) = controller.write_url() {
        println!(
            r#"
  {sshx} {version}

  {arr}  Read-only link: {link_v}
  {arr}  Writable link:  {link_e}
  {arr}  Shell:          {shell_v}
"#,
            sshx = Green.bold().paint("sshx"),
            version = Green.paint(&version_str),
            arr = Green.paint("➜"),
            link_v = Cyan.underline().paint(controller.url()),
            link_e = Cyan.underline().paint(write_url),
            shell_v = Fixed(8).paint(shell),
        );
    } else {
        println!(
            r#"
  {sshx} {version}

  {arr}  Link:  {link_v}
  {arr}  Shell: {shell_v}
"#,
            sshx = Green.bold().paint("sshx"),
            version = Green.paint(&version_str),
            arr = Green.paint("➜"),
            link_v = Cyan.underline().paint(controller.url()),
            shell_v = Fixed(8).paint(shell),
        );
    }
}

#[tokio::main]
async fn start(args: Args) -> Result<()> {
    // Handle service commands if present
    if let Some(cmd) = args.service {
        return match cmd.as_str() {
            "install" => {
                // Use current arguments for service configuration
                service::install_with_config(
                    &args.server,
                    args.dashboard.is_some(),
                    args.enable_readers,
                    args.name.as_deref(),
                    args.shell.as_deref(),
                )
            }
            "uninstall" => service::uninstall(),
            "status" => service::status(),
            "start" => service::start(),
            "stop" => service::stop(),
            _ => Err(anyhow::anyhow!("Invalid service command")),
        };
    }

    let shell = match args.shell {
        Some(shell) => shell,
        None => get_default_shell().await,
    };

    let name = args.name.unwrap_or_else(|| {
        let mut name = whoami::username();
        if let Ok(host) = whoami::fallible::hostname() {
            // Trim domain information like .lan or .local
            let host = host.split('.').next().unwrap_or(&host);
            name += "@";
            name += host;
        }
        name
    });

    let runner = Runner::Shell(shell.clone());
    
    // Create connection configuration based on verbose flag
    let connection_config = if args.verbose {
        verbose_config()
    } else {
        ConnectionConfig::default()
    };
    
    // Establish connection with automatic fallback
    let connection_result = connect_with_fallback(&args.server, &name, connection_config).await?;
    
    // Report connection method if verbose
    if args.verbose {
        match connection_result.method {
            sshx::connection::ConnectionMethod::Grpc => {
                eprintln!("  {} Connected via gRPC", Green.paint("✓"));
            }
            sshx::connection::ConnectionMethod::WebSocketFallback => {
                eprintln!("  {} Connected via WebSocket fallback", Green.paint("✓"));
            }
        }
    }
    
    let mut controller = Controller::with_transport(&args.server, &name, runner, args.enable_readers, connection_result.transport).await?;

    // Register with dashboard if requested
    if let Some(dashboard_option) = args.dashboard {
        // dashboard_option is Some(key) if key provided, None if just --dashboard
        let dashboard_key = dashboard_option;
        if let Err(e) =
            register_with_dashboard(&args.server, &controller, &name, dashboard_key).await
        {
            warn!("Dashboard registration failed: {}", e);
        }
    }

    if args.quiet {
        if let Some(write_url) = controller.write_url() {
            println!("{}", write_url);
        } else {
            println!("{}", controller.url());
        }
    } else {
        print_greeting(&shell, &controller);
    }

    let exit_signal = signal::ctrl_c();
    tokio::pin!(exit_signal);
    tokio::select! {
        _ = controller.run() => unreachable!(),
        Ok(()) = &mut exit_signal => (),
    };
    controller.close().await?;

    Ok(())
}

fn main() -> ExitCode {
    let args = Args::parse();

    let default_level = if args.quiet { 
        "error" 
    } else if args.verbose {
        "debug"
    } else { 
        "info" 
    };

    tracing_subscriber::fmt()
        .with_env_filter(std::env::var("RUST_LOG").unwrap_or(default_level.into()))
        .with_writer(std::io::stderr)
        .init();

    match start(args) {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            // Provide user-friendly error messages
            let error_msg = format!("{}", err);
            if error_msg.contains("Both gRPC and WebSocket connections failed") {
                error!("❌ Unable to connect to the sshx server.");
                error!("   Please check:");
                error!("   • Server URL is correct: {}", std::env::var("SSHX_SERVER").unwrap_or_else(|_| "https://sshx.io".to_string()));
                error!("   • Network connectivity is available");
                error!("   • Server is running and accessible");
                error!("   Use --verbose for detailed connection diagnostics");
            } else if error_msg.contains("gRPC") && error_msg.contains("WebSocket") {
                error!("❌ Connection failed: {}", err);
                error!("   Try again with --verbose for detailed diagnostics");
            } else {
                error!("❌ {}", err);
            }
            ExitCode::FAILURE
        }
    }
}
