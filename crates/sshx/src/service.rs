//! Service handling functions

use anyhow::Result;
use std::process::Command;
use std::fs;

const SERVICE_FILE: &str = r#"[Unit]
Description=SSHX Terminal Sharing Service 
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/sshx
Restart=on-failure

[Install]
WantedBy=multi-user.target"#;

/// Install the sshx service.
pub fn install() -> Result<()> {
    fs::write("/etc/systemd/system/sshx.service", SERVICE_FILE)?;
    
    Command::new("systemctl")
        .args(["daemon-reload"])
        .status()?;
        
    Command::new("systemctl")
        .args(["enable", "sshx"]) 
        .status()?;

    Command::new("systemctl")
        .args(["start", "sshx"])
        .status()?;
        
    Ok(())
}

/// Uninstall the sshx service.
pub fn uninstall() -> Result<()> {
    Command::new("systemctl")
        .args(["disable", "sshx"])
        .status()?;
        
    fs::remove_file("/etc/systemd/system/sshx.service")?;
    
    Command::new("systemctl")
        .args(["stop", "sshx"])
        .status()?;

    Command::new("systemctl")
        .args(["daemon-reload"])
        .status()?;
        
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
    Command::new("systemctl")
        .args(["start", "sshx"])
        .status()?;
    Ok(())
}

/// Stop the sshx service.
pub fn stop() -> Result<()> {
    Command::new("systemctl")
        .args(["stop", "sshx"])
        .status()?;
    Ok(())
}
