package matcher

import (
	"strings"
	"sync"
)

// Service implements the Matcher interface
type Service struct {
	mutex          sync.RWMutex
	whitelistedIPs map[string]bool // Map for O(1) lookup
}

// NewService creates a new Service instance
func NewService() *Service {
	service := &Service{
		whitelistedIPs: make(map[string]bool),
	}

	// Initialize whitelisted IPs map for faster lookups
	for _, ip := range Whitelist {
		service.whitelistedIPs[ip] = true
	}

	return service
}

// IsMalicious checks if a path is malicious
func (s *Service) IsMalicious(path string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Normalize path
	normalizedPath := strings.ToLower(path)

	// Check for exact matches and prefix matches
	for _, pattern := range Patterns {
		if normalizedPath == pattern || strings.HasPrefix(normalizedPath, pattern) {
			return true
		}
	}

	return false
}

// IsWhitelisted checks if an IP is in the whitelist
func (s *Service) IsWhitelisted(ip string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s.whitelistedIPs[ip]
	return exists
}
