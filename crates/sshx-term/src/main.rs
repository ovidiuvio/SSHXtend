use anyhow::Result;
use clap::Parser;

mod client;
mod selector;
mod session;
mod terminal;

use client::SshxClient;
use selector::show_terminal_selector;
use session::parse_sshx_url;
use terminal::run_terminal_session;

/// Terminal client for sshx sessions
#[derive(Parser, Debug)]
#[clap(author, version, about, long_about = None)]
struct Args {
    /// sshx session URL
    url: String,
    
    /// Always create a new terminal (don't show selector)
    #[clap(short, long)]
    new: bool,
    
    /// Connect to specific terminal ID
    #[clap(short, long)]
    terminal: Option<u32>,
    
    /// List terminals and exit
    #[clap(short, long)]
    list: bool,
    
    /// Read-only mode
    #[clap(short, long)]
    readonly: bool,
    
    /// Verbose output
    #[clap(short, long)]
    verbose: bool,
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();
    
    // Setup logging if verbose
    if args.verbose {
        tracing_subscriber::fmt::init();
    }

    // Setup global panic hook for terminal cleanup
    std::panic::set_hook(Box::new(|panic_info| {
        let _ = crossterm::terminal::disable_raw_mode();
        eprintln!("sshx-term panicked: {}", panic_info);
        std::process::exit(1);
    }));

    // Setup Ctrl+C handler for clean exit
    tokio::spawn(async {
        tokio::signal::ctrl_c().await.expect("Failed to listen for Ctrl+C");
        let _ = crossterm::terminal::disable_raw_mode();
        std::process::exit(0);
    });
    
    // Parse sshx URL to extract session info
    let (server, session_id, key, write_password) = parse_sshx_url(&args.url)?;
    
    // Connect to the session
    let mut client = SshxClient::connect(
        server, 
        session_id, 
        key,
        if args.readonly { None } else { write_password }
    ).await?;
    
    // Get current shells
    let shells = client.get_shells().await?;
    
    // Handle list mode
    if args.list {
        if shells.is_empty() {
            println!("No terminals in this session");
        } else {
            println!("Terminals in session:");
            for (i, shell) in shells.iter().enumerate() {
                println!("  [{}] Terminal {} ({}x{})", 
                    i + 1, shell.id, shell.winsize.cols, shell.winsize.rows);
            }
        }
        return Ok(());
    }
    
    // Determine which shell to connect to
    let shell_id = if args.new {
        // Always create new terminal
        client.create_shell(0, 0).await?
    } else if let Some(terminal_id) = args.terminal {
        // Connect to specific terminal
        let target_sid = sshx_core::Sid(terminal_id);
        if shells.iter().any(|s| s.id == target_sid) {
            target_sid
        } else {
            eprintln!("Terminal {} not found", terminal_id);
            std::process::exit(1);
        }
    } else if shells.is_empty() {
        // No terminals - create one automatically  
        client.create_shell(0, 0).await?
    } else if shells.len() == 1 {
        // Single terminal - connect directly
        shells[0].id
    } else {
        // Multiple terminals - show selector
        let selected = show_terminal_selector(&shells).await?;
        if selected.0 == u32::MAX {
            // User chose "Create new terminal"
            client.create_shell(0, 0).await?
        } else {
            selected
        }
    };
    
    // Enter terminal session
    run_terminal_session(&mut client, shell_id).await?;
    
    // Force immediate exit to return control to shell
    drop(client);
    std::process::exit(0)
}