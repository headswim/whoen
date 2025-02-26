// Package whoen provides IP blocking middleware for Go web applications.
// It can be used with various web frameworks like Gin, Chi, and standard net/http.
package whoen

import (
	"log"
	"os"
	"runtime"

	"github.com/headswim/whoen/blocker"
	"github.com/headswim/whoen/config"
	"github.com/headswim/whoen/matcher"
	"github.com/headswim/whoen/middleware"
	"github.com/headswim/whoen/storage"
)

// New creates a new instance of the whoen middleware with default configuration
func New() (*middleware.Middleware, error) {
	return NewWithConfig(config.DefaultConfig())
}

// NewWithConfig creates a new instance of the whoen middleware with custom configuration
func NewWithConfig(cfg config.Config) (*middleware.Middleware, error) {
	// Auto-detect system type if not specified
	if cfg.SystemType == "" {
		cfg.SystemType = getSystemType()
	}

	// Create storage
	store, err := storage.NewJSONStorage(cfg.BlockedIPsFile)
	if err != nil {
		return nil, err
	}

	// Create blocker service
	blockSvc := blocker.NewServiceWithSystemType(cfg.SystemType)

	// Create matcher service
	matchSvc := matcher.NewService()

	// Create middleware options
	opts := middleware.Options{
		Config:          cfg,
		Storage:         store,
		Matcher:         matchSvc,
		Blocker:         blockSvc,
		Logger:          log.New(os.Stdout, "[whoen] ", log.LstdFlags),
		GracePeriod:     cfg.GracePeriod,
		TimeoutEnabled:  cfg.TimeoutEnabled,
		TimeoutDuration: cfg.TimeoutDuration,
		TimeoutIncrease: cfg.TimeoutIncrease,
		CleanupEnabled:  cfg.CleanupEnabled,
		CleanupInterval: cfg.CleanupInterval,
	}

	// Create middleware
	return middleware.New(opts)
}

// getSystemType returns the appropriate system type based on runtime.GOOS
func getSystemType() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

// RestoreBlocks restores OS-level blocks from previous runs
// This should be called at application startup to ensure blocks persist across restarts
func RestoreBlocks(blockedIPsFile string) error {
	systemType := getSystemType()
	return middleware.RestoreBlocks(blockedIPsFile, systemType)
}

// SetWhitelist allows setting a custom whitelist of IPs that should never be blocked
func SetWhitelist(ips []string) {
	matcher.Whitelist = ips
}

// AddToWhitelist adds IPs to the whitelist
func AddToWhitelist(ips ...string) {
	matcher.Whitelist = append(matcher.Whitelist, ips...)
}

// SetPatterns allows setting custom patterns for detecting malicious requests
func SetPatterns(patterns []string) {
	matcher.Patterns = patterns
}

// AddPatterns adds patterns to the existing list
func AddPatterns(patterns ...string) {
	matcher.Patterns = append(matcher.Patterns, patterns...)
}

// Expose important types from subpackages
type (
	// Config represents the configuration for whoen
	Config = config.Config

	// BlockType represents the type of block (Ban or Timeout)
	BlockType = blocker.BlockType

	// BlockResult represents the result of a block operation
	BlockResult = blocker.BlockResult
)

// Constants for block types
const (
	Timeout = blocker.Timeout
	Ban     = blocker.Ban
)
