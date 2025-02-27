package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JSONStorage implements the Storage interface using JSON files
type JSONStorage struct {
	blockedIPsFile    string
	requestCountsFile string
	mutex             sync.RWMutex
}

// NewJSONStorage creates a new JSONStorage instance
func NewJSONStorage(blockedIPsFile string) (*JSONStorage, error) {
	// Create the request counts file in the same directory as the blocked IPs file
	dir := filepath.Dir(blockedIPsFile)
	requestCountsFile := filepath.Join(dir, "request_counts.json")

	storage := &JSONStorage{
		blockedIPsFile:    blockedIPsFile,
		requestCountsFile: requestCountsFile,
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// Create files if they don't exist
	for _, file := range []string{blockedIPsFile, requestCountsFile} {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if err := os.WriteFile(file, []byte("[]"), 0644); err != nil {
				return nil, fmt.Errorf("failed to create file %s: %v", file, err)
			}
		}
	}

	return storage, nil
}

// readBlockedIPs reads the blocked IPs from file
func (s *JSONStorage) readBlockedIPs() ([]BlockStatus, error) {
	data, err := os.ReadFile(s.blockedIPsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []BlockStatus{}, nil
		}
		return nil, err
	}

	var blockedIPs []BlockStatus
	if err := json.Unmarshal(data, &blockedIPs); err != nil {
		return nil, err
	}

	return blockedIPs, nil
}

// writeBlockedIPs writes the blocked IPs to file
func (s *JSONStorage) writeBlockedIPs(blockedIPs []BlockStatus) error {
	data, err := json.MarshalIndent(blockedIPs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.blockedIPsFile, data, 0644)
}

// readRequestCounts reads the request counts from file
func (s *JSONStorage) readRequestCounts() ([]RequestCounter, error) {
	data, err := os.ReadFile(s.requestCountsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []RequestCounter{}, nil
		}
		return nil, err
	}

	var requestCounts []RequestCounter
	if err := json.Unmarshal(data, &requestCounts); err != nil {
		return nil, err
	}

	return requestCounts, nil
}

// writeRequestCounts writes the request counts to file
func (s *JSONStorage) writeRequestCounts(requestCounts []RequestCounter) error {
	data, err := json.MarshalIndent(requestCounts, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.requestCountsFile, data, 0644)
}

// IsIPBlocked checks if an IP is blocked
func (s *JSONStorage) IsIPBlocked(ip string) (bool, *BlockStatus, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	blockedIPs, err := s.readBlockedIPs()
	if err != nil {
		return false, nil, err
	}

	now := time.Now()
	for _, status := range blockedIPs {
		if status.IP == ip {
			if !status.IsPermanent && now.After(status.BlockedUntil) {
				return false, &status, nil
			}
			return true, &status, nil
		}
	}

	return false, nil, nil
}

// BlockIP blocks an IP
func (s *JSONStorage) BlockIP(ip string, until time.Time, isPermanent bool, path string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	blockedIPs, err := s.readBlockedIPs()
	if err != nil {
		return err
	}

	// Update or add block status
	found := false
	for i, status := range blockedIPs {
		if status.IP == ip {
			blockedIPs[i].BlockedUntil = until
			blockedIPs[i].IsPermanent = isPermanent
			blockedIPs[i].LastRequestPath = path
			found = true
			break
		}
	}

	if !found {
		blockedIPs = append(blockedIPs, BlockStatus{
			IP:              ip,
			BlockedAt:       time.Now(),
			BlockedUntil:    until,
			RequestCount:    1,
			TimeoutCount:    0,
			IsPermanent:     isPermanent,
			LastRequestPath: path,
		})
	}

	return s.writeBlockedIPs(blockedIPs)
}

// UnblockIP unblocks an IP
func (s *JSONStorage) UnblockIP(ip string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	blockedIPs, err := s.readBlockedIPs()
	if err != nil {
		return err
	}

	// Remove IP from blocked list
	newBlockedIPs := make([]BlockStatus, 0, len(blockedIPs))
	for _, status := range blockedIPs {
		if status.IP != ip {
			newBlockedIPs = append(newBlockedIPs, status)
		}
	}

	return s.writeBlockedIPs(newBlockedIPs)
}

// GetBlockedIPs returns all blocked IPs
func (s *JSONStorage) GetBlockedIPs() ([]BlockStatus, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.readBlockedIPs()
}

// IncrementRequestCount increments the request count for an IP
func (s *JSONStorage) IncrementRequestCount(ip string, path string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	requestCounts, err := s.readRequestCounts()
	if err != nil {
		return err
	}

	// Update request counts
	now := time.Now()
	found := false
	for i, counter := range requestCounts {
		if counter.IP == ip {
			requestCounts[i].Count++
			requestCounts[i].LastSeen = now
			requestCounts[i].LastPath = path
			found = true
			break
		}
	}

	if !found {
		requestCounts = append(requestCounts, RequestCounter{
			IP:        ip,
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
			LastPath:  path,
		})
	}

	// Also update blocked IP status if it exists
	blockedIPs, err := s.readBlockedIPs()
	if err != nil {
		return err
	}

	for i, status := range blockedIPs {
		if status.IP == ip {
			blockedIPs[i].RequestCount++
			blockedIPs[i].LastRequestPath = path
			if err := s.writeBlockedIPs(blockedIPs); err != nil {
				return err
			}
			break
		}
	}

	return s.writeRequestCounts(requestCounts)
}

// IncrementTimeoutCount increments the timeout count for an IP
func (s *JSONStorage) IncrementTimeoutCount(ip string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	requestCounts, err := s.readRequestCounts()
	if err != nil {
		return err
	}

	// Update request counts
	for i, counter := range requestCounts {
		if counter.IP == ip {
			requestCounts[i].TimeoutCount++
			if err := s.writeRequestCounts(requestCounts); err != nil {
				return err
			}
			break
		}
	}

	// Also update blocked IP status if it exists
	blockedIPs, err := s.readBlockedIPs()
	if err != nil {
		return err
	}

	for i, status := range blockedIPs {
		if status.IP == ip {
			blockedIPs[i].TimeoutCount++
			return s.writeBlockedIPs(blockedIPs)
		}
	}

	return nil
}

// GetRequestCount gets the request count for an IP
func (s *JSONStorage) GetRequestCount(ip string) (int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	requestCounts, err := s.readRequestCounts()
	if err != nil {
		return 0, err
	}

	for _, counter := range requestCounts {
		if counter.IP == ip {
			return counter.Count, nil
		}
	}

	return 0, nil
}

// SetRequestCount sets the request count for an IP
func (s *JSONStorage) SetRequestCount(ip string, count int, path string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	requestCounts, err := s.readRequestCounts()
	if err != nil {
		return err
	}

	now := time.Now()
	found := false
	for i, counter := range requestCounts {
		if counter.IP == ip {
			requestCounts[i].Count = count
			requestCounts[i].LastSeen = now
			requestCounts[i].LastPath = path
			found = true
			break
		}
	}

	if !found {
		requestCounts = append(requestCounts, RequestCounter{
			IP:        ip,
			Count:     count,
			FirstSeen: now,
			LastSeen:  now,
			LastPath:  path,
		})
	}

	return s.writeRequestCounts(requestCounts)
}

// ResetRequestCount resets the request count for an IP
func (s *JSONStorage) ResetRequestCount(ip string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	requestCounts, err := s.readRequestCounts()
	if err != nil {
		return err
	}

	newRequestCounts := make([]RequestCounter, 0, len(requestCounts))
	for _, counter := range requestCounts {
		if counter.IP != ip {
			newRequestCounts = append(newRequestCounts, counter)
		}
	}

	return s.writeRequestCounts(newRequestCounts)
}

// GetAllRequestCounts returns all request counts
func (s *JSONStorage) GetAllRequestCounts() (map[string]RequestCounter, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	requestCounts, err := s.readRequestCounts()
	if err != nil {
		return nil, err
	}

	result := make(map[string]RequestCounter, len(requestCounts))
	for _, counter := range requestCounts {
		result[counter.IP] = counter
	}

	return result, nil
}

// CleanupExpired removes expired blocks from storage
func (s *JSONStorage) CleanupExpired() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	blockedIPs, err := s.readBlockedIPs()
	if err != nil {
		return err
	}

	now := time.Now()
	staleThreshold := now.Add(-24 * time.Hour)

	// Clean up expired blocks
	newBlockedIPs := make([]BlockStatus, 0, len(blockedIPs))
	for _, status := range blockedIPs {
		if !status.IsPermanent && now.After(status.BlockedUntil) {
			continue
		}
		newBlockedIPs = append(newBlockedIPs, status)
	}

	if err := s.writeBlockedIPs(newBlockedIPs); err != nil {
		return err
	}

	// Clean up stale request counts
	requestCounts, err := s.readRequestCounts()
	if err != nil {
		return err
	}

	newRequestCounts := make([]RequestCounter, 0, len(requestCounts))
	for _, counter := range requestCounts {
		if !counter.LastSeen.Before(staleThreshold) {
			newRequestCounts = append(newRequestCounts, counter)
		}
	}

	return s.writeRequestCounts(newRequestCounts)
}

// Save is a no-op since we save immediately after each operation
func (s *JSONStorage) Save() error {
	return nil
}

// Load is a no-op since we load for each operation
func (s *JSONStorage) Load() error {
	return nil
}

// Close is a no-op since we don't maintain any in-memory state
func (s *JSONStorage) Close() error {
	return nil
}
