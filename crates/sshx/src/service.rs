//! Service handling functions

use anyhow::{Context, Result};
use std::env;
use std::fs;
use std::path::Path;
use std::process::Command;

/// Generate systemd service file content with configuration
fn generate_service_file(
    server: &str,
    dashboard: bool,
    enable_readers: bool,
    name: Option<&str>,
    shell: Option<&str>,
) -> String {
    let mut exec_start = "/usr/local/bin/sshx".to_string();

    // Add server argument if not default
    if server != "https://sshx.io" {
        exec_start.push_str(&format!(" --server {}", server));
    }

    // Add dashboard flag
    if dashboard {
        exec_start.push_str(" --dashboard");
    }

    // Add enable-readers flag
    if enable_readers {
        exec_start.push_str(" --enable-readers");
    }

    // Add name if specified
    if let Some(name) = name {
        exec_start.push_str(&format!(" --name '{}'", name));
    }

    // Add shell if specified
    if let Some(shell) = shell {
        exec_start.push_str(&format!(" --shell '{}'", shell));
    }

    format!(
        r#"[Unit]
Description=SSHX Terminal Sharing Service
After=network.target

[Service]
Type=simple
ExecStart={}
Restart=on-failure
RestartSec=5
User=root
Environment=HOME=/root
WorkingDirectory=/root

[Install]
WantedBy=multi-user.target"#,
        exec_start
    )
}

/// Install the sshx service with configuration.
pub fn install_with_config(
    server: &str,
    dashboard: bool,
    enable_readers: bool,
    name: Option<&str>,
    shell: Option<&str>,
) -> Result<()> {
    // Check if we're running as root by checking if we can write to /etc
    if !Path::new("/etc/systemd/system").exists() {
        return Err(anyhow::anyhow!(
            "systemd directory not found. This system may not support systemd services."
        ));
    }

    // Try to create a test file to check permissions
    if fs::write("/etc/systemd/system/.sshx-test", "").is_err() {
        return Err(anyhow::anyhow!(
            "Service installation requires root privileges. Please run with sudo."
        ));
    }
    let _ = fs::remove_file("/etc/systemd/system/.sshx-test");

    // Copy the current binary to /usr/local/bin/sshx
    let current_exe = env::current_exe().context("Failed to get current executable path")?;

    let target_path = "/usr/local/bin/sshx";

    println!(
        "Copying binary from {} to {}",
        current_exe.display(),
        target_path
    );
    fs::copy(&current_exe, target_path).context("Failed to copy binary to /usr/local/bin/sshx")?;

    // Set executable permissions
    Command::new("chmod")
        .args(["+x", target_path])
        .status()
        .context("Failed to set executable permissions")?;

    // Generate and write service file
    let service_content = generate_service_file(server, dashboard, enable_readers, name, shell);

    println!("Installing systemd service...");
    fs::write("/etc/systemd/system/sshx.service", service_content)
        .context("Failed to write service file")?;

    // Reload systemd daemon
    println!("Reloading systemd daemon...");
    Command::new("systemctl")
        .args(["daemon-reload"])
        .status()
        .context("Failed to reload systemd daemon")?;

    // Enable service
    println!("Enabling sshx service...");
    Command::new("systemctl")
        .args(["enable", "sshx"])
        .status()
        .context("Failed to enable sshx service")?;

    // Start service
    println!("Starting sshx service...");
    Command::new("systemctl")
        .args(["start", "sshx"])
        .status()
        .context("Failed to start sshx service")?;

    println!("✓ SSHX service installed and started successfully");
    println!("  Use 'systemctl status sshx' to check status");
    println!("  Use 'journalctl -u sshx -f' to view logs");

    Ok(())
}

/// Install the sshx service with default configuration.
pub fn install() -> Result<()> {
    install_with_config("https://sshx.io", false, false, None, None)
}

/// Uninstall the sshx service.
pub fn uninstall() -> Result<()> {
    // Check if we can write to systemd directory
    if fs::write("/etc/systemd/system/.sshx-test", "").is_err() {
        return Err(anyhow::anyhow!(
            "Service uninstallation requires root privileges. Please run with sudo."
        ));
    }
    let _ = fs::remove_file("/etc/systemd/system/.sshx-test");

    println!("Stopping sshx service...");
    let _ = Command::new("systemctl").args(["stop", "sshx"]).status(); // Ignore errors in case service is already stopped

    println!("Disabling sshx service...");
    let _ = Command::new("systemctl").args(["disable", "sshx"]).status(); // Ignore errors in case service is already disabled

    println!("Removing service file...");
    let _ = fs::remove_file("/etc/systemd/system/sshx.service"); // Ignore if file doesn't exist

    println!("Removing binary...");
    let _ = fs::remove_file("/usr/local/bin/sshx"); // Ignore if file doesn't exist

    println!("Reloading systemd daemon...");
    Command::new("systemctl")
        .args(["daemon-reload"])
        .status()
        .context("Failed to reload systemd daemon")?;

    println!("✓ SSHX service uninstalled successfully");

    Ok(())
}

/// Check the status of the sshx service.
pub fn status() -> Result<()> {
    Command::new("systemctl")
        .args(["status", "sshx"])
        .status()?;
    Ok(())
}

/// Start the sshx service.
pub fn start() -> Result<()> {
    Command::new("systemctl").args(["start", "sshx"]).status()?;
    Ok(())
}

/// Stop the sshx service.
pub fn stop() -> Result<()> {
    Command::new("systemctl").args(["stop", "sshx"]).status()?;
    Ok(())
}
