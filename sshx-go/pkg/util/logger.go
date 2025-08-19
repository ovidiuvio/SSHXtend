package util

import (
	"log"
	"os"
)

var (
	// DebugEnabled controls whether debug logs are printed
	DebugEnabled bool
)

// SetDebugMode enables or disables debug logging
func SetDebugMode(enabled bool) {
	DebugEnabled = enabled
}

// InitLogger initializes the logger based on environment and flags
func InitLogger(verbose bool) {
	// Check environment variable as fallback
	if os.Getenv("SSHX_DEBUG") != "" {
		DebugEnabled = true
	}
	
	// Command line flag takes precedence
	if verbose {
		DebugEnabled = true
	}
}

// DebugLog prints a debug message only if debug mode is enabled
func DebugLog(format string, args ...interface{}) {
	if DebugEnabled {
		log.Printf("[DEBUG] "+format, args...)
	}
}