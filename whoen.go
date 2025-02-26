// Package whoen provides IP blocking middleware for Go web applications.
// It can be used with various web frameworks like Gin, Chi, and standard net/http.
package whoen

import (
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
