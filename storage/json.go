package storage

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// JSONStorage implements the Storage interface using JSON files
type JSONStorage struct {
	blockedIPsFile string
	blockedIPs     map[string]BlockStatus
	mutex          sync.RWMutex
	saveTicker     *time.Ticker
	done           chan bool
}

// NewJSONStorage creates a new JSONStorage instance
func NewJSONStorage(blockedIPsFile string) (*JSONStorage, error) {
	storage := &JSONStorage{
		blockedIPsFile: blockedIPsFile,
		blockedIPs:     make(map[string]BlockStatus),
		done:           make(chan bool),
	}

	// Load existing data
	err := storage.Load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Start periodic saving
	storage.saveTicker = time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-storage.saveTicker.C:
				_ = storage.Save()
			case <-storage.done:
				return
			}
		}
	}()

	return storage, nil
}

// IsIPBlocked checks if an IP is blocked
func (s *JSONStorage) IsIPBlocked(ip string) (bool, *BlockStatus, error) {
	s.mutex.RLock()

	status, exists := s.blockedIPs[ip]
	if !exists {
		s.mutex.RUnlock()
		return false, nil, nil
	}

	// Check if the block has expired
	if !status.IsPermanent && time.Now().After(status.BlockedUntil) {
		// Need to switch to a write lock to remove the expired block
		s.mutex.RUnlock()
		s.mutex.Lock()
		defer s.mutex.Unlock()

		// Check again after acquiring the write lock
		status, exists = s.blockedIPs[ip]
		if !exists {
			return false, nil, nil
		}

		// Check expiration again after acquiring the write lock
		if !status.IsPermanent && time.Now().After(status.BlockedUntil) {
			delete(s.blockedIPs, ip)
		}

		return false, &status, nil
	}

	s.mutex.RUnlock()
	return true, &status, nil
}

// BlockIP blocks an IP
func (s *JSONStorage) BlockIP(ip string, until time.Time, isPermanent bool, path string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status, exists := s.blockedIPs[ip]
	if exists {
		status.BlockedUntil = until
		status.IsPermanent = isPermanent
		status.LastRequestPath = path
	} else {
		status = BlockStatus{
			IP:              ip,
			BlockedAt:       time.Now(),
			BlockedUntil:    until,
			RequestCount:    1,
			TimeoutCount:    0,
			IsPermanent:     isPermanent,
			LastRequestPath: path,
		}
	}

	s.blockedIPs[ip] = status
	return s.Save()
}

// UnblockIP unblocks an IP
func (s *JSONStorage) UnblockIP(ip string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.blockedIPs, ip)
	return s.Save()
}

// GetBlockedIPs returns all blocked IPs
func (s *JSONStorage) GetBlockedIPs() ([]BlockStatus, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make([]BlockStatus, 0, len(s.blockedIPs))
	for _, status := range s.blockedIPs {
		result = append(result, status)
	}

	return result, nil
}

// IncrementRequestCount increments the request count for an IP
func (s *JSONStorage) IncrementRequestCount(ip string, path string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status, exists := s.blockedIPs[ip]
	if exists {
		status.RequestCount++
		status.LastRequestPath = path
		s.blockedIPs[ip] = status
	} else {
		s.blockedIPs[ip] = BlockStatus{
			IP:              ip,
			BlockedAt:       time.Now(),
			RequestCount:    1,
			LastRequestPath: path,
		}
	}

	return nil
}

// IncrementTimeoutCount increments the timeout count for an IP
func (s *JSONStorage) IncrementTimeoutCount(ip string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status, exists := s.blockedIPs[ip]
	if exists {
		status.TimeoutCount++
		s.blockedIPs[ip] = status
	}

	return nil
}

// Save saves the data to disk
func (s *JSONStorage) Save() error {
	// Save blocked IPs
	blockedIPsList := make([]BlockStatus, 0, len(s.blockedIPs))
	for _, status := range s.blockedIPs {
		blockedIPsList = append(blockedIPsList, status)
	}

	blockedIPsData, err := json.MarshalIndent(blockedIPsList, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.blockedIPsFile, blockedIPsData, 0644)
}

// Load loads the data from disk
func (s *JSONStorage) Load() error {
	// Load blocked IPs
	blockedIPsData, err := os.ReadFile(s.blockedIPsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, initialize with empty data
			s.blockedIPs = make(map[string]BlockStatus)
		} else {
			return err
		}
	} else {
		var blockedIPsList []BlockStatus
		err = json.Unmarshal(blockedIPsData, &blockedIPsList)
		if err != nil {
			return err
		}

		s.blockedIPs = make(map[string]BlockStatus, len(blockedIPsList))
		for _, status := range blockedIPsList {
			s.blockedIPs[status.IP] = status
		}
	}

	return nil
}

// Close closes the storage
func (s *JSONStorage) Close() error {
	s.saveTicker.Stop()
	s.done <- true
	return s.Save()
}

// CleanupExpired removes expired blocks from storage
func (s *JSONStorage) CleanupExpired() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for ip, status := range s.blockedIPs {
		if !status.IsPermanent && now.After(status.BlockedUntil) {
			delete(s.blockedIPs, ip)
		}
	}

	return s.Save()
}
