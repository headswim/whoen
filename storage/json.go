package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JSONStorage implements the Storage interface using JSON files
type JSONStorage struct {
	blockedIPsFile    string
	requestCountsFile string
	blockedIPs        map[string]BlockStatus
	requestCounts     map[string]RequestCounter
	mutex             sync.RWMutex
	saveTicker        *time.Ticker
	done              chan bool
}

// NewJSONStorage creates a new JSONStorage instance
func NewJSONStorage(blockedIPsFile string) (*JSONStorage, error) {
	// Create the request counts file in the same directory as the blocked IPs file
	dir := filepath.Dir(blockedIPsFile)
	requestCountsFile := filepath.Join(dir, "request_counts.json")

	storage := &JSONStorage{
		blockedIPsFile:    blockedIPsFile,
		requestCountsFile: requestCountsFile,
		blockedIPs:        make(map[string]BlockStatus),
		requestCounts:     make(map[string]RequestCounter),
		done:              make(chan bool),
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
				log.Printf("[whoen-debug] Running periodic save")
				if err := storage.Save(); err != nil {
					log.Printf("[whoen-error] Error in periodic save: %v", err)
				}
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

	// Update the request counter
	counter, exists := s.requestCounts[ip]
	if exists {
		counter.Count++
		counter.LastSeen = time.Now()
		counter.LastPath = path
	} else {
		counter = RequestCounter{
			IP:        ip,
			Count:     1,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			LastPath:  path,
		}
	}
	s.requestCounts[ip] = counter

	// Also update the blocked IP status if it exists
	status, exists := s.blockedIPs[ip]
	if exists {
		status.RequestCount++
		status.LastRequestPath = path
		s.blockedIPs[ip] = status
	}

	// Save changes to disk immediately
	return s.Save()
}

// IncrementTimeoutCount increments the timeout count for an IP
func (s *JSONStorage) IncrementTimeoutCount(ip string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update the request counter
	counter, exists := s.requestCounts[ip]
	if exists {
		counter.TimeoutCount++
		s.requestCounts[ip] = counter
	}

	// Also update the blocked IP status if it exists
	status, exists := s.blockedIPs[ip]
	if exists {
		status.TimeoutCount++
		s.blockedIPs[ip] = status
	}

	// Save changes to disk immediately
	return s.Save()
}

// GetRequestCount gets the request count for an IP
func (s *JSONStorage) GetRequestCount(ip string) (int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	counter, exists := s.requestCounts[ip]
	if !exists {
		return 0, nil
	}

	return counter.Count, nil
}

// SetRequestCount sets the request count for an IP
func (s *JSONStorage) SetRequestCount(ip string, count int, path string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	counter, exists := s.requestCounts[ip]
	if exists {
		counter.Count = count
		counter.LastSeen = time.Now()
		counter.LastPath = path
	} else {
		counter = RequestCounter{
			IP:        ip,
			Count:     count,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			LastPath:  path,
		}
	}
	s.requestCounts[ip] = counter

	return s.Save()
}

// ResetRequestCount resets the request count for an IP
func (s *JSONStorage) ResetRequestCount(ip string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.requestCounts, ip)
	return s.Save()
}

// GetAllRequestCounts returns all request counts
func (s *JSONStorage) GetAllRequestCounts() (map[string]RequestCounter, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]RequestCounter, len(s.requestCounts))
	for ip, counter := range s.requestCounts {
		result[ip] = counter
	}

	return result, nil
}

// Save saves the data to disk
func (s *JSONStorage) Save() error {
	log.Printf("[whoen-debug] Starting save operation to %s and %s", s.blockedIPsFile, s.requestCountsFile)

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.blockedIPsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[whoen-error] Failed to create directory %s: %v", dir, err)
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// Save blocked IPs
	blockedIPsList := make([]BlockStatus, 0, len(s.blockedIPs))
	for _, status := range s.blockedIPs {
		blockedIPsList = append(blockedIPsList, status)
	}

	blockedIPsData, err := json.MarshalIndent(blockedIPsList, "", "  ")
	if err != nil {
		log.Printf("[whoen-error] Failed to marshal blocked IPs: %v", err)
		return fmt.Errorf("failed to marshal blocked IPs: %v", err)
	}
	log.Printf("[whoen-debug] Marshaled %d blocked IPs", len(blockedIPsList))

	if err := os.WriteFile(s.blockedIPsFile, blockedIPsData, 0644); err != nil {
		log.Printf("[whoen-error] Failed to write blocked IPs file: %v", err)
		return fmt.Errorf("failed to write blocked IPs file: %v", err)
	}
	log.Printf("[whoen-debug] Successfully wrote blocked IPs to %s", s.blockedIPsFile)

	// Save request counts
	requestCountsList := make([]RequestCounter, 0, len(s.requestCounts))
	for _, counter := range s.requestCounts {
		requestCountsList = append(requestCountsList, counter)
	}

	requestCountsData, err := json.MarshalIndent(requestCountsList, "", "  ")
	if err != nil {
		log.Printf("[whoen-error] Failed to marshal request counts: %v", err)
		return fmt.Errorf("failed to marshal request counts: %v", err)
	}
	log.Printf("[whoen-debug] Marshaled %d request counts", len(requestCountsList))

	if err := os.WriteFile(s.requestCountsFile, requestCountsData, 0644); err != nil {
		log.Printf("[whoen-error] Failed to write request counts file: %v", err)
		return fmt.Errorf("failed to write request counts file: %v", err)
	}
	log.Printf("[whoen-debug] Successfully wrote request counts to %s", s.requestCountsFile)

	log.Printf("[whoen-debug] Save operation completed successfully")
	return nil
}

// Load loads the data from disk
func (s *JSONStorage) Load() error {
	log.Printf("[whoen-debug] Starting load operation from %s and %s", s.blockedIPsFile, s.requestCountsFile)

	// Initialize maps if they don't exist
	if s.blockedIPs == nil {
		s.blockedIPs = make(map[string]BlockStatus)
	}
	if s.requestCounts == nil {
		s.requestCounts = make(map[string]RequestCounter)
	}

	// Load blocked IPs
	blockedIPsData, err := os.ReadFile(s.blockedIPsFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[whoen-debug] Blocked IPs file doesn't exist, starting with empty map")
		} else {
			log.Printf("[whoen-error] Error reading blocked IPs file: %v", err)
			return fmt.Errorf("error reading blocked IPs file: %v", err)
		}
	} else {
		var blockedIPsList []BlockStatus
		if err := json.Unmarshal(blockedIPsData, &blockedIPsList); err != nil {
			log.Printf("[whoen-error] Error unmarshaling blocked IPs: %v", err)
			return fmt.Errorf("error unmarshaling blocked IPs: %v", err)
		}
		log.Printf("[whoen-debug] Loaded %d blocked IPs from file", len(blockedIPsList))
		for _, status := range blockedIPsList {
			s.blockedIPs[status.IP] = status
		}
	}

	// Load request counts
	requestCountsData, err := os.ReadFile(s.requestCountsFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[whoen-debug] Request counts file doesn't exist, starting with empty map")
		} else {
			log.Printf("[whoen-error] Error reading request counts file: %v", err)
			return fmt.Errorf("error reading request counts file: %v", err)
		}
	} else {
		var requestCountsList []RequestCounter
		if err := json.Unmarshal(requestCountsData, &requestCountsList); err != nil {
			log.Printf("[whoen-error] Error unmarshaling request counts: %v", err)
			return fmt.Errorf("error unmarshaling request counts: %v", err)
		}
		log.Printf("[whoen-debug] Loaded %d request counts from file", len(requestCountsList))
		for _, counter := range requestCountsList {
			s.requestCounts[counter.IP] = counter
		}
	}

	log.Printf("[whoen-debug] Load operation completed successfully")
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
