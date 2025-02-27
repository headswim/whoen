package config

import (
	"os"
	"path/filepath"
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
	StorageDir      string        `json:"storage_dir"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	// Try to use a system-wide directory for storage if possible
	storageDir := getDefaultStorageDir()

	return Config{
		BlockedIPsFile:  filepath.Join(storageDir, "blocked_ips.json"),
		GracePeriod:     3,                                      // Default to 3 requests before blocking
		TimeoutEnabled:  true,                                   // Enable timeout
		TimeoutDuration: 24 * time.Hour,                         // Timeout duration must be set if timeout is enabled
		TimeoutIncrease: "linear",                               // Timeout increase type (linear / geometric)
		LogFile:         filepath.Join(storageDir, "whoen.log"), // where the log file is located
		SystemType:      "",                                     // Auto-detected in whoen.go
		CleanupEnabled:  true,                                   // Enable cleanup by default
		CleanupInterval: 1 * time.Hour,                          // Run cleanup every hour
		StorageDir:      storageDir,                             // Store the directory for future reference
	}
}

// ValidateConfig validates the configuration and sets defaults for missing values
func ValidateConfig(cfg *Config) {
	// Set default values for empty fields
	if cfg.BlockedIPsFile == "" {
		cfg.BlockedIPsFile = "blocked_ips.json"
	}

	if cfg.GracePeriod < 0 {
		cfg.GracePeriod = 3 // Default to 3 requests before blocking
	}

	if cfg.TimeoutDuration <= 0 {
		cfg.TimeoutDuration = 24 * time.Hour
	}

	// Ensure TimeoutIncrease is valid
	if cfg.TimeoutIncrease != "linear" && cfg.TimeoutIncrease != "geometric" {
		cfg.TimeoutIncrease = "linear" // Default to linear
	}

	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 1 * time.Hour
	}

	// Ensure storage directory exists
	if cfg.StorageDir == "" {
		cfg.StorageDir = "."
	}
}

// getDefaultStorageDir returns the default directory for storing Whoen data
func getDefaultStorageDir() string {
	// Try to use a system-wide directory if possible
	var baseDir string

	// Check if we can write to /var/lib/whoen (Linux/Mac)
	if _, err := os.Stat("/var/lib"); err == nil {
		baseDir = "/var/lib/whoen"
		// Try to create the directory
		if err := os.MkdirAll(baseDir, 0755); err == nil {
			// Test if we can write to it
			testFile := filepath.Join(baseDir, ".test")
			if f, err := os.Create(testFile); err == nil {
				f.Close()
				os.Remove(testFile)
				return baseDir
			}
		}
	}

	// Check if we can use the user's home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		baseDir = filepath.Join(homeDir, ".whoen")
		if err := os.MkdirAll(baseDir, 0755); err == nil {
			return baseDir
		}
	}

	// Fall back to the current directory
	return "."
}

// WithStorageDir sets a custom storage directory and updates file paths
func (c Config) WithStorageDir(dir string) Config {
	c.StorageDir = dir
	c.BlockedIPsFile = filepath.Join(dir, filepath.Base(c.BlockedIPsFile))
	c.LogFile = filepath.Join(dir, filepath.Base(c.LogFile))
	return c
}
