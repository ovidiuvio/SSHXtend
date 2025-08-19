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
	"sshx-go/pkg/util"
)

// ANSI color codes to match Rust ansi_term crate
const (
	Green       = "\033[32m"
	BoldGreen   = "\033[1;32m" 
	Cyan        = "\033[36m"
	UnderlineCyan = "\033[4;36m"
	Fixed8      = "\033[38;5;8m"  // Gray color for secondary info
	Reset       = "\033[0m"
)

func main() {
	// Get default values from environment variables - matches Rust implementation
	defaultServer := os.Getenv("SSHX_SERVER")
	if defaultServer == "" {
		defaultServer = "https://sshx.io"
	}
	
	defaultVerbose := os.Getenv("SSHX_VERBOSE") != ""

	var (
		server        = flag.String("server", defaultServer, "Address of the remote sshx server")
		shell         = flag.String("shell", "", "Local shell command to run in the terminal")
		quiet         = flag.Bool("quiet", false, "Quiet mode, only prints the URL to stdout")
		name          = flag.String("name", "", "Session name displayed in the title (defaults to user@hostname)")
		enableReaders = flag.Bool("enable-readers", false, "Enable read-only access mode - generates separate URLs for viewers and editors")
		serviceCmd    = flag.String("service", "", "Service management (install|uninstall|status|start|stop)")
		dashboard     = flag.Bool("dashboard", false, "Register with a new dashboard")
		dashboardKey  = flag.String("dashboard-key", "", "Join existing dashboard with specified key")
		verbose       = flag.Bool("verbose", defaultVerbose, "Enable verbose output showing connection details and fallback attempts")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `A secure web-based, collaborative terminal.

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

Usage:
`)
		flag.PrintDefaults()
	}

	flag.Parse()

	if err := runSshx(*server, *shell, *quiet, *name, *enableReaders, *serviceCmd, *dashboard, *dashboardKey, *verbose); err != nil {
		// Provide user-friendly error messages - matches Rust implementation
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "Both gRPC and WebSocket connections failed") {
			fmt.Fprintf(os.Stderr, "❌ Unable to connect to the sshx server.\n")
			fmt.Fprintf(os.Stderr, "   Please check:\n")
			fmt.Fprintf(os.Stderr, "   • Server URL is correct: %s\n", *server)
			fmt.Fprintf(os.Stderr, "   • Network connectivity is available\n")
			fmt.Fprintf(os.Stderr, "   • Server is running and accessible\n")
			if !*verbose {
				fmt.Fprintf(os.Stderr, "   Use --verbose for detailed connection diagnostics\n")
			}
		} else if strings.Contains(errorMsg, "gRPC") && strings.Contains(errorMsg, "WebSocket") {
			fmt.Fprintf(os.Stderr, "❌ Connection failed: %v\n", err)
			if !*verbose {
				fmt.Fprintf(os.Stderr, "   Try again with --verbose for detailed diagnostics\n")
			}
		} else {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		}
		os.Exit(1)
	}
}

func runSshx(server, shell string, quiet bool, name string, enableReaders bool, serviceCmd string, dashboard bool, dashboardKey string, verbose bool) error {
	// Initialize logger with verbose mode
	util.InitLogger(verbose)
	
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

	// Create connection configuration
	connConfig := transport.DefaultConnectionConfig()
	if verbose {
		connConfig = transport.VerboseConfig()
	}
	
	// Create controller using transport abstraction with automatic fallback
	controller, err := client.NewControllerWithConnection(config, connConfig)
	if err != nil {
		return fmt.Errorf("failed to create controller with transport: %w", err)
	}

	// Report connection method if verbose
	if verbose {
		switch controller.ConnectionMethod() {
		case transport.MethodGrpc:
			log.Printf("✓ Connected via gRPC")
		case transport.MethodWebSocketFallback:
			log.Printf("✓ Connected via WebSocket fallback")
		}
	}

	// Register with dashboard if requested
	var dashboardInfo *DashboardInfo
	if dashboard || dashboardKey != "" {
		var key *string
		if dashboardKey != "" {
			// Join existing dashboard
			key = &dashboardKey
		}
		// else key is nil, which creates a new dashboard
		if info, err := registerWithDashboard(server, controller, sessionName, key); err != nil {
			log.Printf("Dashboard registration failed: %v", err)
		} else {
			dashboardInfo = info
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
		printGreeting(shellCmd, controller, controller.ConnectionMethod(), dashboardInfo)
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

// DashboardInfo for display
type DashboardInfo struct {
	Key string
	URL string
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
}, displayName string, dashboardKey *string) (*DashboardInfo, error) {
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
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Make HTTP POST request
	resp, err := http.Post(dashboardURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to post to dashboard: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response RegisterDashboardResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		fmt.Println("\n  ✓ Session registered to dashboard")
		
		return &DashboardInfo{
			Key: response.DashboardKey,
			URL: response.DashboardURL,
		}, nil
	} else {
		log.Printf("Failed to register with dashboard: %s", resp.Status)
		return nil, fmt.Errorf("Dashboard registration failed with status: %s", resp.Status)
	}
}

func printGreeting(shell string, controller interface {
	URL() string
	WriteURL() *string
}, connectionMethod transport.ConnectionMethod, dashboardInfo *DashboardInfo) {
	version := "v1.0.0" // You could make this dynamic
	transportStr := connectionMethod.String()
	
	if writeURL := controller.WriteURL(); writeURL != nil {
		if dashboardInfo != nil {
			fmt.Printf(`
  %s%ssshx%s %s%s%s

  %s➜%s  Read-only link: %s%s%s
  %s➜%s  Writable link:  %s%s%s
  %s➜%s  Dashboard:      %s%s%s
  %s➜%s  Dashboard ID:   %s%s%s
  %s➜%s  Shell:          %s%s%s
  %s➜%s  Transport:      %s%s%s

`, BoldGreen, Green, Reset, Green, version, Reset,
				Green, Reset, UnderlineCyan, controller.URL(), Reset,
				Green, Reset, UnderlineCyan, *writeURL, Reset,
				Green, Reset, UnderlineCyan, dashboardInfo.URL, Reset,
				Green, Reset, Fixed8, dashboardInfo.Key, Reset,
				Green, Reset, Fixed8, shell, Reset,
				Green, Reset, Fixed8, transportStr, Reset)
		} else {
			fmt.Printf(`
  %s%ssshx%s %s%s%s

  %s➜%s  Read-only link: %s%s%s
  %s➜%s  Writable link:  %s%s%s
  %s➜%s  Shell:          %s%s%s
  %s➜%s  Transport:      %s%s%s

`, BoldGreen, Green, Reset, Green, version, Reset,
				Green, Reset, UnderlineCyan, controller.URL(), Reset,
				Green, Reset, UnderlineCyan, *writeURL, Reset,
				Green, Reset, Fixed8, shell, Reset,
				Green, Reset, Fixed8, transportStr, Reset)
		}
	} else {
		if dashboardInfo != nil {
			fmt.Printf(`
  %s%ssshx%s %s%s%s

  %s➜%s  Link:         %s%s%s
  %s➜%s  Dashboard:    %s%s%s
  %s➜%s  Dashboard ID: %s%s%s
  %s➜%s  Shell:        %s%s%s
  %s➜%s  Transport:    %s%s%s

`, BoldGreen, Green, Reset, Green, version, Reset,
				Green, Reset, UnderlineCyan, controller.URL(), Reset,
				Green, Reset, UnderlineCyan, dashboardInfo.URL, Reset,
				Green, Reset, Fixed8, dashboardInfo.Key, Reset,
				Green, Reset, Fixed8, shell, Reset,
				Green, Reset, Fixed8, transportStr, Reset)
		} else {
			fmt.Printf(`
  %s%ssshx%s %s%s%s

  %s➜%s  Link:      %s%s%s
  %s➜%s  Shell:     %s%s%s
  %s➜%s  Transport: %s%s%s

`, BoldGreen, Green, Reset, Green, version, Reset,
				Green, Reset, UnderlineCyan, controller.URL(), Reset,
				Green, Reset, Fixed8, shell, Reset,
				Green, Reset, Fixed8, transportStr, Reset)
		}
	}
}