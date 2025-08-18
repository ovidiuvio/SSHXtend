package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"

	"sshx-go/pkg/client"
	"sshx-go/pkg/service"
	"sshx-go/pkg/terminal"
	"sshx-go/pkg/transport"
)

func main() {
	var (
		server        = flag.String("server", "https://sshx.io", "Address of the remote sshx server")
		shell         = flag.String("shell", "", "Local shell command to run in the terminal")
		quiet         = flag.Bool("quiet", false, "Quiet mode, only prints the URL to stdout")
		name          = flag.String("name", "", "Session name displayed in the title (defaults to user@hostname)")
		enableReaders = flag.Bool("enable-readers", false, "Enable read-only access mode")
		serviceCmd    = flag.String("service", "", "Service management (install|uninstall|status|start|stop)")
		dashboard     = flag.Bool("dashboard", false, "Register with a new dashboard")
		dashboardKey  = flag.String("dashboard-key", "", "Join existing dashboard with specified key")
		useTransport  = flag.Bool("use-transport", false, "Use new transport abstraction with gRPC->WebSocket fallback")
		verbose       = flag.Bool("verbose", false, "Enable verbose connection reporting")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `SSHX Terminal Sharing

Service Management:
  --service install    Install and enable systemd service with current configuration
  --service uninstall  Remove systemd service and binary
  --service status     Check service status
  --service start      Start service
  --service stop       Stop service

Examples:
  sshx --dashboard                            # Create new dashboard
  sshx --dashboard-key abc123                  # Join existing dashboard
  sshx --server https://your-server.com --dashboard --service install
  sshx --shell /bin/bash --name server1 --service install

Usage:
`)
		flag.PrintDefaults()
	}

	flag.Parse()

	if err := runSshx(*server, *shell, *quiet, *name, *enableReaders, *serviceCmd, *dashboard, *dashboardKey, *useTransport, *verbose); err != nil {
		log.Fatal(err)
	}
}

func runSshx(server, shell string, quiet bool, name string, enableReaders bool, serviceCmd string, dashboard bool, dashboardKey string, useTransport bool, verbose bool) error {
	// Handle service commands if present
	if serviceCmd != "" {
		return handleServiceCommand(serviceCmd, server, dashboard || dashboardKey != "", enableReaders, name, shell)
	}

	// Get shell command
	shellCmd := shell
	if shellCmd == "" {
		shellCmd = terminal.GetDefaultShell()
	}

	// Get session name
	sessionName := name
	if sessionName == "" {
		sessionName = getDefaultSessionName()
	}

	// Create runner
	runner := &client.ShellRunner{Shell: shellCmd}

	// Create controller config
	config := client.ControllerConfig{
		Origin:        server,
		Name:          sessionName,
		Runner:        runner,
		EnableReaders: enableReaders,
	}

	var controller interface {
		Name() string
		URL() string
		WriteURL() *string
		Run() error
		Close() error
	}

	// Create controller - use new transport abstraction if requested
	if useTransport {
		connConfig := transport.DefaultConnectionConfig()
		if verbose {
			connConfig = transport.VerboseConfig()
		}
		v2Controller, err := client.NewControllerV2WithConnection(config, connConfig)
		if err != nil {
			return fmt.Errorf("failed to create controller with transport: %w", err)
		}
		controller = v2Controller
		if verbose {
			log.Printf("Using %s transport", v2Controller.ConnectionMethod())
		}
	} else {
		v1Controller, err := client.NewController(config)
		if err != nil {
			return fmt.Errorf("failed to create controller: %w", err)
		}
		controller = v1Controller
	}

	// Register with dashboard if requested
	if dashboard || dashboardKey != "" {
		var key *string
		if dashboardKey != "" {
			// Join existing dashboard
			key = &dashboardKey
		}
		// else key is nil, which creates a new dashboard
		if err := registerWithDashboard(server, controller, sessionName, key); err != nil {
			log.Printf("Dashboard registration failed: %v", err)
		}
	}

	// Print greeting or URL
	if quiet {
		if writeURL := controller.WriteURL(); writeURL != nil {
			fmt.Println(*writeURL)
		} else {
			fmt.Println(controller.URL())
		}
	} else {
		printGreeting(shellCmd, controller)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run controller in background
	done := make(chan error, 1)
	go func() {
		done <- controller.Run()
	}()

	// Wait for completion or signal
	select {
	case <-sigChan:
		log.Println("Received interrupt, shutting down...")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("controller error: %w", err)
		}
	}

	// Graceful shutdown
	return controller.Close()
}

func handleServiceCommand(serviceCmd, server string, dashboard, enableReaders bool, name, shell string) error {
	config := service.ServiceConfig{
		Server:        server,
		Dashboard:     dashboard,
		EnableReaders: enableReaders,
	}

	if name != "" {
		config.Name = &name
	}

	if shell != "" {
		config.Shell = &shell
	}

	switch serviceCmd {
	case "install":
		return service.InstallWithConfig(config)
	case "uninstall":
		return service.Uninstall()
	case "status":
		return service.Status()
	case "start":
		return service.Start()
	case "stop":
		return service.Stop()
	default:
		return fmt.Errorf("invalid service command: %s", serviceCmd)
	}
}

func getDefaultSessionName() string {
	sessionName := "unknown"
	
	if currentUser, err := user.Current(); err == nil {
		sessionName = currentUser.Username
	}
	
	if hostname, err := os.Hostname(); err == nil {
		// Trim domain information like .lan or .local
		if parts := strings.Split(hostname, "."); len(parts) > 0 {
			hostname = parts[0]
		}
		sessionName += "@" + hostname
	}
	
	return sessionName
}

// RegisterDashboardRequest matches the Rust implementation
type RegisterDashboardRequest struct {
	SessionName  string  `json:"sessionName"`
	URL          string  `json:"url"`
	WriteURL     *string `json:"writeUrl,omitempty"`
	DisplayName  string  `json:"displayName"`
	DashboardKey *string `json:"dashboardKey,omitempty"`
}

// RegisterDashboardResponse from the server
type RegisterDashboardResponse struct {
	DashboardKey string `json:"dashboardKey"`
	DashboardURL string `json:"dashboardUrl"`
}

// makeRelativeURL extracts relative URL from full URL for reverse proxy compatibility
func makeRelativeURL(fullURL string) string {
	if u, err := url.Parse(fullURL); err == nil {
		relative := u.Path
		if u.RawQuery != "" {
			relative += "?" + u.RawQuery
		}
		if u.Fragment != "" {
			relative += "#" + u.Fragment
		}
		return relative
	}
	// If parsing fails, assume it's already relative
	return fullURL
}

func registerWithDashboard(server string, controller interface {
	Name() string
	URL() string
	WriteURL() *string
}, displayName string, dashboardKey *string) error {
	dashboardURL := server + "/api/dashboards/register"
	
	// Prepare request payload - matches Rust RegisterDashboardRequest exactly
	request := RegisterDashboardRequest{
		SessionName:  controller.Name(),
		URL:          makeRelativeURL(controller.URL()),
		DisplayName:  displayName,
		DashboardKey: dashboardKey,
	}
	
	if writeURL := controller.WriteURL(); writeURL != nil {
		relativeWriteURL := makeRelativeURL(*writeURL)
		request.WriteURL = &relativeWriteURL
	}
	
	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Make HTTP POST request
	resp, err := http.Post(dashboardURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to post to dashboard: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response RegisterDashboardResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
		log.Println("✓ Session registered to dashboard")
		log.Printf("➜ Dashboard URL: %s", response.DashboardURL)
	} else {
		log.Printf("Failed to register with dashboard: %s", resp.Status)
	}
	
	return nil
}

func printGreeting(shell string, controller interface {
	URL() string
	WriteURL() *string
}) {
	version := "v1.0.0" // You could make this dynamic

	if writeURL := controller.WriteURL(); writeURL != nil {
		fmt.Printf(`
  sshx %s

  ➜  Read-only link: %s
  ➜  Writable link:  %s
  ➜  Shell:          %s

`, version, controller.URL(), *writeURL, shell)
	} else {
		fmt.Printf(`
  sshx %s

  ➜  Link:  %s
  ➜  Shell: %s

`, version, controller.URL(), shell)
	}
}