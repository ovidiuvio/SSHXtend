use anyhow::{Context, Result};
use crossterm::{
    terminal::{disable_raw_mode, enable_raw_mode, size},
};
use sshx_core::Sid;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::signal;
use tracing::{debug, error};

use crate::client::SshxClient;

pub async fn run_terminal_session(client: &mut SshxClient, shell_id: Sid) -> Result<()> {
    // Subscribe to the shell
    client.subscribe_to_shell(shell_id).await?;

    // Get initial terminal size and resize the shell
    let (cols, rows) = size()?;
    client
        .resize_shell(shell_id, rows, cols)
        .await
        .context("Failed to resize shell")?;


    // Enable raw mode for direct terminal control
    enable_raw_mode()?;

    // Setup signal handling for terminal resize
    let mut sigwinch = signal::unix::signal(signal::unix::SignalKind::window_change())?;

    // Setup stdin for reading
    let mut stdin = tokio::io::stdin();
    let mut stdout = tokio::io::stdout();

    let result = run_session_loop(client, shell_id, &mut stdin, &mut stdout, &mut sigwinch).await;

    // Always clean up raw mode, even on error
    let cleanup_result = disable_raw_mode();

    // Return cleanup error if that failed, otherwise original result
    cleanup_result.context("Failed to restore terminal")?;
    result
}

async fn run_session_loop(
    client: &mut SshxClient,
    shell_id: Sid,
    stdin: &mut tokio::io::Stdin,
    stdout: &mut tokio::io::Stdout,
    sigwinch: &mut signal::unix::Signal,
) -> Result<()> {
    let mut input_buffer = [0u8; 1024];

    // Setup Ctrl+C handler
    let mut sigint = signal::unix::signal(signal::unix::SignalKind::interrupt())?;

    loop {
        tokio::select! {
            // Handle Ctrl+C (SIGINT)
            _ = sigint.recv() => {
                debug!("Received SIGINT, exiting cleanly");
                break;
            }

            // Handle terminal resize
            _ = sigwinch.recv() => {
                if let Ok((cols, rows)) = size() {
                    debug!("Terminal resized to {}x{}", cols, rows);
                    if let Err(e) = client.resize_shell(shell_id, rows, cols).await {
                        error!("Failed to resize shell: {}", e);
                    }
                }
            }

            // Handle stdin input
            result = stdin.read(&mut input_buffer) => {
                match result {
                    Ok(0) => {
                        // EOF - connection closed
                        debug!("Stdin closed (EOF)");
                        break;
                    }
                    Ok(n) => {
                        let data = &input_buffer[..n];
                        
                        // Check for Ctrl+D (EOF)
                        if data.len() == 1 && data[0] == 0x04 {
                            debug!("Ctrl+D detected, exiting");
                            break;
                        }

                        // Check for escape sequence to exit client: Ctrl+] followed by q
                        if should_exit_on_input(data) {
                            debug!("Exit escape sequence detected, exiting client");
                            break;
                        }

                        // Send input to remote shell
                        if let Err(e) = client.send_input(shell_id, data).await {
                            error!("Failed to send input: {}", e);
                            break;
                        }
                    }
                    Err(e) => {
                        error!("Failed to read from stdin: {}", e);
                        break;
                    }
                }
            }

            // Handle output from remote shell
            result = client.receive_terminal_data(Some(shell_id)) => {
                match result {
                    Ok(Some((received_shell_id, data))) => {
                        if received_shell_id == shell_id {
                            // Write data directly to stdout
                            if let Err(e) = stdout.write_all(&data).await {
                                error!("Failed to write to stdout: {}", e);
                                break;
                            }
                            if let Err(e) = stdout.flush().await {
                                error!("Failed to flush stdout: {}", e);
                                break;
                            }
                        }
                    }
                    Ok(None) => {
                        // Non-terminal data (like shell updates), continue
                    }
                    Err(e) => {
                        let error_msg = e.to_string();
                        if error_msg.contains("has been closed") {
                            debug!("Remote shell closed, exiting cleanly");
                            // Just break - don't print anything, like SSH
                        } else {
                            error!("Failed to receive terminal data: {}", e);
                        }
                        break;
                    }
                }
            }
        }
    }

    debug!("Exiting session loop");
    Ok(())
}

// Track escape sequence state
static mut ESCAPE_STATE: EscapeState = EscapeState::Normal;

#[derive(Debug, PartialEq)]
enum EscapeState {
    Normal,
    GotCtrlRightBracket, // Got Ctrl+] (0x1D)
}

fn should_exit_on_input(data: &[u8]) -> bool {
    unsafe {
        for &byte in data {
            match ESCAPE_STATE {
                EscapeState::Normal => {
                    if byte == 0x1D { // Ctrl+] 
                        ESCAPE_STATE = EscapeState::GotCtrlRightBracket;
                    }
                }
                EscapeState::GotCtrlRightBracket => {
                    if byte == b'q' || byte == b'Q' {
                        ESCAPE_STATE = EscapeState::Normal;
                        return true; // Exit sequence detected
                    } else {
                        ESCAPE_STATE = EscapeState::Normal;
                    }
                }
            }
        }
    }
    false
}

