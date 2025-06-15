// Package service provides systemd service management functionality.
package service

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	serviceName = "sshx"
	serviceFile = "/etc/systemd/system/sshx.service"
	binaryPath  = "/usr/local/bin/sshx"
)

// ServiceConfig holds configuration for the systemd service.
type ServiceConfig struct {
	Server        string
	Dashboard     bool
	EnableReaders bool
	Name          *string
	Shell         *string
}

// InstallWithConfig installs the sshx service with the provided configuration.
func InstallWithConfig(config ServiceConfig) error {
	// Check permissions
	if err := checkPermissions(); err != nil {
		return err
	}

	// Copy binary
	if err := copyBinary(); err != nil {
		return err
	}

	// Generate and write service file
	serviceContent := generateServiceFile(config)
	if err := writeServiceFile(serviceContent); err != nil {
		return err
	}

	// Reload systemd and enable/start service
	if err := reloadSystemd(); err != nil {
		return err
	}

	if err := enableService(); err != nil {
		return err
	}

	if err := startService(); err != nil {
		return err
	}

	fmt.Println("✓ SSHX service installed and started successfully")
	fmt.Println("  Use 'systemctl status sshx' to check status")
	fmt.Println("  Use 'journalctl -u sshx -f' to view logs")

	return nil
}

// Install installs the sshx service with default configuration.
func Install() error {
	return InstallWithConfig(ServiceConfig{
		Server: "https://sshx.io",
	})
}

// Uninstall removes the sshx service.
func Uninstall() error {
	// Check permissions
	if err := checkPermissions(); err != nil {
		return err
	}

	fmt.Println("Stopping sshx service...")
	_ = runCommand("systemctl", "stop", serviceName) // Ignore errors

	fmt.Println("Disabling sshx service...")
	_ = runCommand("systemctl", "disable", serviceName) // Ignore errors

	fmt.Println("Removing service file...")
	_ = os.Remove(serviceFile) // Ignore if file doesn't exist

	fmt.Println("Removing binary...")
	_ = os.Remove(binaryPath) // Ignore if file doesn't exist

	fmt.Println("Reloading systemd daemon...")
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	fmt.Println("✓ SSHX service uninstalled successfully")
	return nil
}

// Status checks the status of the sshx service.
func Status() error {
	return runCommand("systemctl", "status", serviceName)
}

// Start starts the sshx service.
func Start() error {
	return runCommand("systemctl", "start", serviceName)
}

// Stop stops the sshx service.
func Stop() error {
	return runCommand("systemctl", "stop", serviceName)
}

// checkPermissions verifies that we have the necessary permissions.
func checkPermissions() error {
	if !fileExists("/etc/systemd/system") {
		return fmt.Errorf("systemd directory not found. This system may not support systemd services")
	}

	// Try to create a test file to check permissions
	testFile := "/etc/systemd/system/.sshx-test"
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		return fmt.Errorf("service management requires root privileges. Please run with sudo")
	}
	os.Remove(testFile)

	return nil
}

// copyBinary copies the current executable to the system location.
func copyBinary() error {
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	fmt.Printf("Copying binary from %s to %s\n", currentExe, binaryPath)

	input, err := os.ReadFile(currentExe)
	if err != nil {
		return fmt.Errorf("failed to read current binary: %w", err)
	}

	if err := os.WriteFile(binaryPath, input, 0755); err != nil {
		return fmt.Errorf("failed to copy binary to %s: %w", binaryPath, err)
	}

	return nil
}

// generateServiceFile creates the systemd service file content.
func generateServiceFile(config ServiceConfig) string {
	execStart := binaryPath

	// Add server argument if not default
	if config.Server != "https://sshx.io" {
		execStart += fmt.Sprintf(" --server %s", config.Server)
	}

	// Add dashboard flag
	if config.Dashboard {
		execStart += " --dashboard"
	}

	// Add enable-readers flag
	if config.EnableReaders {
		execStart += " --enable-readers"
	}

	// Add name if specified
	if config.Name != nil {
		execStart += fmt.Sprintf(" --name '%s'", *config.Name)
	}

	// Add shell if specified
	if config.Shell != nil {
		execStart += fmt.Sprintf(" --shell '%s'", *config.Shell)
	}

	return fmt.Sprintf(`[Unit]
Description=SSHX Terminal Sharing Service
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5
User=root
Environment=HOME=/root
WorkingDirectory=/root

[Install]
WantedBy=multi-user.target`, execStart)
}

// writeServiceFile writes the service file content to the systemd directory.
func writeServiceFile(content string) error {
	fmt.Println("Installing systemd service...")
	if err := os.WriteFile(serviceFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}
	return nil
}

// reloadSystemd reloads the systemd daemon.
func reloadSystemd() error {
	fmt.Println("Reloading systemd daemon...")
	return runCommand("systemctl", "daemon-reload")
}

// enableService enables the systemd service.
func enableService() error {
	fmt.Println("Enabling sshx service...")
	return runCommand("systemctl", "enable", serviceName)
}

// startService starts the systemd service.
func startService() error {
	fmt.Println("Starting sshx service...")
	return runCommand("systemctl", "start", serviceName)
}

// runCommand executes a system command.
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// fileExists checks if a file or directory exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}