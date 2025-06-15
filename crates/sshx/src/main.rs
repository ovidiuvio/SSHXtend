use std::process::ExitCode;

use ansi_term::Color::{Cyan, Fixed, Green};
use anyhow::Result;
use clap::Parser;
use serde::Serialize;
use sshx::{controller::Controller, runner::Runner, service, terminal::get_default_shell};
use tokio::signal;
use tracing::{error, warn};

/// A secure web-based, collaborative terminal.
#[derive(Parser, Debug)]
#[clap(author, version, about = "
SSHX Terminal Sharing

Service Management:
  --service install    Install and enable systemd service with current configuration
  --service uninstall  Remove systemd service and binary
  --service status     Check service status
  --service start      Start service
  --service stop       Stop service

Examples:
  sshx --server https://your-server.com --dashboard --service install
  sshx --shell /bin/bash --name server1 --service install
")]
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

    /// Optional custom session ID.
    #[clap(long)]
    session_id: Option<String>,

    /// Optional encryption key.
    #[clap(long)]
    secret: Option<String>,

    /// Service management (install|uninstall|status|start|stop)
    #[clap(long, value_parser = ["install", "uninstall", "status", "start", "stop"])]
    service: Option<String>,

    /// Register this session with the web dashboard for monitoring.
    #[clap(long)]
    dashboard: bool,
}

/// Dashboard registration request payload
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct RegisterDashboardRequest {
    session_name: String,
    url: String,
    write_url: Option<String>,
    display_name: String,
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
async fn register_with_dashboard(server_url: &str, controller: &Controller, display_name: &str) -> Result<()> {
    let dashboard_url = format!("{}/api/dashboard/register", server_url);
    
    let request = RegisterDashboardRequest {
        session_name: controller.name().to_string(),
        url: make_relative_url(controller.url()),
        write_url: controller.write_url().map(|u| make_relative_url(u)),
        display_name: display_name.to_string(),
    };

    let client = reqwest::Client::new();
    let response = client
        .post(&dashboard_url)
        .json(&request)
        .send()
        .await?;

    if response.status().is_success() {
        println!("✓ Session registered with dashboard");
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
                    args.dashboard,
                    args.enable_readers,
                    args.name.as_deref(),
                    args.shell.as_deref(),
                )
            },
            "uninstall" => service::uninstall(),
            "status" => service::status(),
            "start" => service::start(),
            "stop" => service::stop(),
            _ => Err(anyhow::anyhow!("Invalid service command"))
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
    let mut controller = Controller::new(
        &args.server,
        &name,
        runner,
        args.enable_readers,
        args.session_id,
        args.secret,
    )
    .await?;

    // Register with dashboard if requested
    if args.dashboard {
        if let Err(e) = register_with_dashboard(&args.server, &controller, &name).await {
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

    let default_level = if args.quiet { "error" } else { "info" };

    tracing_subscriber::fmt()
        .with_env_filter(std::env::var("RUST_LOG").unwrap_or(default_level.into()))
        .with_writer(std::io::stderr)
        .init();

    match start(args) {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            error!("{err:?}");
            ExitCode::FAILURE
        }
    }
}
