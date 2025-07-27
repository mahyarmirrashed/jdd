package config

import (
	"time"
)

// Config holds the YAML configuration for the daemon.
type Config struct {
	Root          string        // Root directory to watch
	LogLevel      string        // Logging level: debug, info, warn, error
	Exclude       []string      // Glob patterns to exclude
	DryRun        bool          // If true, don't move files
	Daemonize     bool          // If true, run as daemon; if false, run in foreground
	Delay         time.Duration // Time before before processing files
	Notifications bool          // If true, send desktop notifications
}
