package blocker

import (
	"time"
)

// BlockType represents the type of block
type BlockType int

const (
	// Timeout represents a temporary block
	// Ban represents a permanent block
	Timeout BlockType = iota
	Ban
)

// BlockResult represents the result of a block operation
type BlockResult struct {
	IP        string
	BlockType BlockType
	Duration  time.Duration
	Error     error
}

// Blocker defines the interface for IP blocking
type Blocker interface {
	// Block blocks an IP
	Block(ip string, blockType BlockType, duration time.Duration) (*BlockResult, error)

	// Unblock unblocks an IP
	Unblock(ip string) error

	// IsBlocked checks if an IP is blocked
	IsBlocked(ip string) (bool, error)

	// CleanupExpired removes expired blocks
	CleanupExpired() error
}
