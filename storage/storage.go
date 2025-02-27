package storage

import (
	"time"
)

// BlockStatus represents the status of a blocked IP
type BlockStatus struct {
	IP              string    `json:"ip"`
	BlockedAt       time.Time `json:"blocked_at"`
	BlockedUntil    time.Time `json:"blocked_until,omitempty"` // Empty for permanent blocks
	RequestCount    int       `json:"request_count"`
	TimeoutCount    int       `json:"timeout_count"`
	IsPermanent     bool      `json:"is_permanent"`
	LastRequestPath string    `json:"last_request_path"`
}

// RequestCounter represents the request count for an IP
type RequestCounter struct {
	IP           string    `json:"ip"`
	Count        int       `json:"count"`
	LastSeen     time.Time `json:"last_seen"`
	LastPath     string    `json:"last_path"`
	FirstSeen    time.Time `json:"first_seen"`
	TimeoutCount int       `json:"timeout_count"`
}

// Storage defines the interface for storing and retrieving blocked IPs
type Storage interface {
	// Blocked IPs management
	IsIPBlocked(ip string) (bool, *BlockStatus, error)
	BlockIP(ip string, until time.Time, isPermanent bool, path string) error
	UnblockIP(ip string) error
	GetBlockedIPs() ([]BlockStatus, error)
	IncrementRequestCount(ip string, path string) error
	IncrementTimeoutCount(ip string) error

	// Request counter management
	GetRequestCount(ip string) (int, error)
	SetRequestCount(ip string, count int, path string) error
	ResetRequestCount(ip string) error
	GetAllRequestCounts() (map[string]RequestCounter, error)

	// Cleanup expired blocks
	CleanupExpired() error

	// Storage management
	Save() error
	Load() error
	Close() error
}
