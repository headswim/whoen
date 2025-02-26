package config

import (
	"time"
)

// Config holds the configuration for the whoen middleware
type Config struct {
	BlockedIPsFile  string        `json:"blocked_ips_file"`
	GracePeriod     int           `json:"grace_period"`
	TimeoutEnabled  bool          `json:"timeout_enabled"`
	TimeoutDuration time.Duration `json:"timeout_duration"`
	TimeoutIncrease string        `json:"timeout_increase"`
	LogFile         string        `json:"log_file"`
	SystemType      string        `json:"system_type"`
	CleanupEnabled  bool          `json:"cleanup_enabled"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		BlockedIPsFile:  "blocked_ips.json",
		GracePeriod:     1,              // 0 will block on first attempt
		TimeoutEnabled:  true,           // Enable timeout
		TimeoutDuration: 24 * time.Hour, // Timeout duration must be set if timeout is enabled
		TimeoutIncrease: "linear",       // Timeout increase type (linear / geometric)
		LogFile:         "whoen.log",    // where the log file is located
		SystemType:      "linux",        // System type (mac / linux / windows)
		CleanupEnabled:  false,          // Disabled by default
		CleanupInterval: 1 * time.Hour,  // Run cleanup every hour when enabled
	}
}
