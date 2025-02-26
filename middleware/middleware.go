package middleware

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"whoen/blocker"
	"whoen/config"
	"whoen/matcher"
	"whoen/storage"
)

// Options represents the options for the middleware
type Options struct {
	Config          config.Config
	Storage         storage.Storage
	Matcher         matcher.Matcher
	Blocker         blocker.Blocker
	Logger          *log.Logger
	GracePeriod     int
	TimeoutEnabled  bool
	TimeoutDuration time.Duration
	TimeoutIncrease string // "linear" or "geometric"
}

// DefaultOptions returns the default options
func DefaultOptions() Options {
	cfg := config.DefaultConfig()
	return Options{
		Config:          cfg,
		GracePeriod:     cfg.GracePeriod,
		TimeoutEnabled:  cfg.TimeoutEnabled,
		TimeoutDuration: cfg.TimeoutDuration,
		TimeoutIncrease: cfg.TimeoutIncrease,
		Logger:          log.New(os.Stdout, "[whoen] ", log.LstdFlags),
	}
}

// Middleware represents the core middleware
type Middleware struct {
	options Options
	storage storage.Storage
	matcher matcher.Matcher
	blocker blocker.Blocker
	logger  *log.Logger
}

// New creates a new middleware
func New(options Options) (*Middleware, error) {
	m := &Middleware{
		options: options,
		logger:  options.Logger,
	}

	// Initialize storage if not provided
	if options.Storage == nil {
		storage, err := storage.NewJSONStorage(
			options.Config.BlockedIPsFile,
		)
		if err != nil {
			return nil, err
		}
		m.storage = storage
	} else {
		m.storage = options.Storage
	}

	// Initialize matcher if not provided
	if options.Matcher == nil {
		// Create a new matcher service with pre-defined patterns
		m.matcher = matcher.NewService()
	} else {
		m.matcher = options.Matcher
	}

	// Initialize blocker if not provided
	if options.Blocker == nil {
		m.blocker = blocker.NewServiceWithSystemType(options.Config.SystemType)
	} else {
		m.blocker = options.Blocker
	}

	/*
		Uncomment to enable periodic cleanup of expired blocks
		// This will run a background goroutine that periodically cleans up expired blocks
		cleanupTicker := time.NewTicker(1 * time.Hour)
		go func() {
		    for {
		        select {
		        case <-cleanupTicker.C:
		            if err := m.CleanupExpired(); err != nil {
		                m.logger.Printf("Error cleaning up expired blocks: %v", err)
		            }
		        }
		    }
		}()
	*/

	return m, nil
}

// HandleRequest handles an HTTP request
func (m *Middleware) HandleRequest(r *http.Request) (bool, error) {
	// Get client IP
	ip, err := getClientIP(r)
	if err != nil {
		m.logger.Printf("Error getting client IP: %v", err)
		return false, err
	}

	// Check if IP is whitelisted
	if m.matcher.IsWhitelisted(ip) {
		m.logger.Printf("Allowing whitelisted IP: %s", ip)
		return false, nil
	}

	// Check if IP is already blocked
	isBlocked, err := m.blocker.IsBlocked(ip)
	if err != nil {
		m.logger.Printf("Error checking if IP is blocked: %v", err)
		return false, err
	}

	if isBlocked {
		m.logger.Printf("Blocked request from %s to %s", ip, r.URL.Path)
		return true, nil
	}

	// Check if path is malicious
	isMalicious := m.matcher.IsMalicious(r.URL.Path)
	if !isMalicious {
		return false, nil
	}

	// Path is malicious, increment request count
	err = m.storage.IncrementRequestCount(ip, r.URL.Path)
	if err != nil {
		m.logger.Printf("Error incrementing request count: %v", err)
		return false, err
	}

	// Check if IP should be blocked
	isBlocked, status, err := m.storage.IsIPBlocked(ip)
	if err != nil {
		m.logger.Printf("Error checking if IP should be blocked: %v", err)
		return false, err
	}

	if isBlocked {
		// IP is already blocked in storage, make sure it's blocked at OS level
		if status.IsPermanent {
			_, err = m.blocker.Block(ip, blocker.Ban, 0)
		} else {
			_, err = m.blocker.Block(ip, blocker.Timeout, time.Until(status.BlockedUntil))
		}
		if err != nil {
			m.logger.Printf("Error blocking IP: %v", err)
		}
		return true, nil
	}

	// Check if grace period is exceeded
	if status != nil && status.RequestCount > m.options.GracePeriod {
		// Grace period exceeded, block IP
		if m.options.TimeoutEnabled {
			// Calculate timeout duration
			duration := m.calculateTimeoutDuration(status.TimeoutCount)

			// Block IP with timeout
			_, err = m.blocker.Block(ip, blocker.Timeout, duration)
			if err != nil {
				m.logger.Printf("Error blocking IP: %v", err)
				return false, err
			}

			// Update storage
			err = m.storage.BlockIP(ip, time.Now().Add(duration), false, r.URL.Path)
			if err != nil {
				m.logger.Printf("Error updating storage: %v", err)
			}

			// Increment timeout count
			err = m.storage.IncrementTimeoutCount(ip)
			if err != nil {
				m.logger.Printf("Error incrementing timeout count: %v", err)
			}

			m.logger.Printf("Blocked IP %s for %s for accessing malicious path %s", ip, duration, r.URL.Path)
		} else {
			// Block IP permanently
			_, err = m.blocker.Block(ip, blocker.Ban, 0)
			if err != nil {
				m.logger.Printf("Error blocking IP: %v", err)
				return false, err
			}

			// Update storage
			err = m.storage.BlockIP(ip, time.Time{}, true, r.URL.Path)
			if err != nil {
				m.logger.Printf("Error updating storage: %v", err)
			}

			m.logger.Printf("Permanently blocked IP %s for accessing malicious path %s", ip, r.URL.Path)
		}

		return true, nil
	}

	m.logger.Printf("Malicious request from %s to %s (count: %d)", ip, r.URL.Path, status.RequestCount)
	return false, nil
}

// calculateTimeoutDuration calculates the timeout duration based on the timeout count
func (m *Middleware) calculateTimeoutDuration(timeoutCount int) time.Duration {
	baseDuration := m.options.TimeoutDuration

	if timeoutCount == 0 {
		return baseDuration
	}

	if m.options.TimeoutIncrease == "geometric" {
		// Geometric increase: duration * 2^timeoutCount
		multiplier := 1
		for i := 0; i < timeoutCount; i++ {
			multiplier *= 2
		}
		return baseDuration * time.Duration(multiplier)
	}

	// Linear increase: duration * (timeoutCount + 1)
	return baseDuration * time.Duration(timeoutCount+1)
}

// getClientIP gets the client IP from the request
func getClientIP(r *http.Request) (string, error) {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := splitAndTrim(xff)
		if len(ips) > 0 {
			return ips[0], nil
		}
	}

	// Check X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip, nil
	}

	// Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr, nil
	}

	return ip, nil
}

// splitAndTrim splits a string by comma and trims spaces
func splitAndTrim(s string) []string {
	var result []string
	for _, item := range split(s, ',') {
		item = trim(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

// split splits a string by a separator
func split(s string, sep byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// trim trims spaces from a string
func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for start < end && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}

// CleanupExpired removes expired blocks from both storage and blocker
func (m *Middleware) CleanupExpired() error {
	// Get all blocked IPs from storage
	blockedIPs, err := m.storage.GetBlockedIPs()
	if err != nil {
		return err
	}

	// Check each IP
	now := time.Now()
	for _, status := range blockedIPs {
		if !status.IsPermanent && now.After(status.BlockedUntil) {
			// Unblock at OS level
			if err := m.blocker.Unblock(status.IP); err != nil {
				m.logger.Printf("Error unblocking IP %s: %v", status.IP, err)
			}
		}
	}

	// Clean up expired blocks in storage
	if err := m.storage.CleanupExpired(); err != nil {
		return err
	}

	// Clean up expired blocks in blocker
	if err := m.blocker.CleanupExpired(); err != nil {
		return err
	}

	return nil
}
