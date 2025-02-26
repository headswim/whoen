package blocker

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Service implements the Blocker interface
type Service struct {
	blockedIPs map[string]time.Time // IP -> expiration time (zero for permanent)
	mutex      sync.RWMutex
	systemType string // "linux", "darwin" (mac), or "windows"
}

// NewService creates a new Service instance
func NewService() *Service {
	return &Service{
		blockedIPs: make(map[string]time.Time),
		systemType: "linux", // Default to linux
	}
}

// NewServiceWithSystemType creates a new Service instance with a specific system type
func NewServiceWithSystemType(systemType string) *Service {
	// Normalize system type
	normalizedType := strings.ToLower(systemType)
	if normalizedType == "mac" {
		normalizedType = "darwin"
	}

	return &Service{
		blockedIPs: make(map[string]time.Time),
		systemType: normalizedType,
	}
}

// SetSystemType sets the system type for the blocker
func (s *Service) SetSystemType(systemType string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Normalize system type
	if strings.ToLower(systemType) == "mac" {
		s.systemType = "darwin"
	} else {
		s.systemType = strings.ToLower(systemType)
	}
}

// Block blocks an IP
func (s *Service) Block(ip string, blockType BlockType, duration time.Duration) (*BlockResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := &BlockResult{
		IP:        ip,
		BlockType: blockType,
		Duration:  duration,
	}

	// Check if IP is already blocked
	if expiration, exists := s.blockedIPs[ip]; exists {
		// If it's a permanent block, or the existing block is longer, do nothing
		if expiration.IsZero() || (blockType == Timeout && time.Now().Add(duration).Before(expiration)) {
			return result, nil
		}
	}

	// Block the IP at the OS level
	var err error
	if s.systemType == "linux" {
		err = blockIPLinux(ip)
	} else if s.systemType == "darwin" {
		err = blockIPDarwin(ip)
	} else if s.systemType == "windows" {
		err = blockIPWindows(ip)
	} else {
		err = fmt.Errorf("unsupported system type: %s", s.systemType)
	}

	if err != nil {
		result.Error = err
		return result, err
	}

	// Update the blocked IPs map
	if blockType == Ban {
		s.blockedIPs[ip] = time.Time{} // Zero time for permanent blocks
	} else {
		s.blockedIPs[ip] = time.Now().Add(duration)
	}

	return result, nil
}

// Unblock unblocks an IP
func (s *Service) Unblock(ip string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if IP is blocked
	if _, exists := s.blockedIPs[ip]; !exists {
		return nil
	}

	// Unblock the IP at the OS level
	var err error
	if s.systemType == "linux" {
		err = unblockIPLinux(ip)
	} else if s.systemType == "darwin" {
		err = unblockIPDarwin(ip)
	} else if s.systemType == "windows" {
		err = unblockIPWindows(ip)
	} else {
		err = fmt.Errorf("unsupported system type: %s", s.systemType)
	}

	if err != nil {
		return err
	}

	// Remove from the blocked IPs map
	delete(s.blockedIPs, ip)

	return nil
}

// IsBlocked checks if an IP is blocked
func (s *Service) IsBlocked(ip string) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	expiration, exists := s.blockedIPs[ip]
	if !exists {
		return false, nil
	}

	// If it's a permanent block, or the block hasn't expired yet
	if expiration.IsZero() || time.Now().Before(expiration) {
		return true, nil
	}

	// Block has expired, remove it
	delete(s.blockedIPs, ip)
	return false, nil
}

// CleanupExpired removes expired blocks
func (s *Service) CleanupExpired() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for ip, expiration := range s.blockedIPs {
		if !expiration.IsZero() && now.After(expiration) {
			// Unblock the IP at the OS level
			var err error
			if s.systemType == "linux" {
				err = unblockIPLinux(ip)
			} else if s.systemType == "darwin" {
				err = unblockIPDarwin(ip)
			} else if s.systemType == "windows" {
				err = unblockIPWindows(ip)
			} else {
				continue // Skip unsupported system types
			}

			if err != nil {
				return err
			}

			// Remove from the blocked IPs map
			delete(s.blockedIPs, ip)
		}
	}

	return nil
}

// RestoreBlocks restores blocks from a list of IPs and expiration times
// This can be called from the main application to restore blocks after a restart
func (s *Service) RestoreBlocks(ips map[string]time.Time) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	restored := 0
	skipped := 0

	for ip, expiration := range ips {
		// Skip expired blocks
		if !expiration.IsZero() && now.After(expiration) {
			skipped++
			continue
		}

		// Apply the block at OS level
		var err error
		if s.systemType == "linux" {
			err = blockIPLinux(ip)
		} else if s.systemType == "darwin" {
			err = blockIPDarwin(ip)
		} else if s.systemType == "windows" {
			err = blockIPWindows(ip)
		} else {
			return fmt.Errorf("unsupported system type: %s", s.systemType)
		}

		if err != nil {
			return fmt.Errorf("failed to restore block for IP %s: %v", ip, err)
		}

		// Update the blocked IPs map
		s.blockedIPs[ip] = expiration
		restored++
	}

	fmt.Printf("Restored %d IP blocks, skipped %d expired blocks\n", restored, skipped)
	return nil
}

// blockIPLinux blocks an IP on Linux using iptables
func blockIPLinux(ip string) error {
	// Use -I INPUT 1 to insert at the beginning of the chain for highest priority
	cmd := exec.Command("sudo", "iptables", "-I", "INPUT", "1", "-s", ip, "-j", "DROP")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to block IP %s with iptables: %v (output: %s)", ip, err, string(output))
	}

	// Also block outgoing connections to this IP for complete isolation
	outCmd := exec.Command("sudo", "iptables", "-I", "OUTPUT", "1", "-d", ip, "-j", "DROP")
	outOutput, outErr := outCmd.CombinedOutput()
	if outErr != nil {
		return fmt.Errorf("failed to block outgoing connections to IP %s with iptables: %v (output: %s)", ip, outErr, string(outOutput))
	}
	return nil
}

// unblockIPLinux unblocks an IP on Linux using iptables
func unblockIPLinux(ip string) error {
	// Remove both INPUT and OUTPUT rules
	inCmd := exec.Command("sudo", "iptables", "-D", "INPUT", "-s", ip, "-j", "DROP")
	inOutput, inErr := inCmd.CombinedOutput()

	outCmd := exec.Command("sudo", "iptables", "-D", "OUTPUT", "-d", ip, "-j", "DROP")
	outOutput, outErr := outCmd.CombinedOutput()

	// Return an error if either command failed
	if inErr != nil {
		return fmt.Errorf("failed to unblock IP %s with iptables (INPUT): %v (output: %s)", ip, inErr, string(inOutput))
	}
	if outErr != nil {
		return fmt.Errorf("failed to unblock IP %s with iptables (OUTPUT): %v (output: %s)", ip, outErr, string(outOutput))
	}
	return nil
}

// blockIPDarwin blocks an IP on macOS using pfctl
func blockIPDarwin(ip string) error {
	// Check if the rule already exists
	checkCmd := exec.Command("sudo", "pfctl", "-t", "blocklist", "-T", "show")
	output, err := checkCmd.CombinedOutput()
	if err != nil {
		// If the table doesn't exist, create it
		createCmd := exec.Command("sudo", "pfctl", "-t", "blocklist", "-T", "create")
		createOutput, createErr := createCmd.CombinedOutput()
		if createErr != nil {
			return fmt.Errorf("failed to create blocklist table with pfctl: %v (output: %s)", createErr, string(createOutput))
		}
	}

	if !strings.Contains(string(output), ip) {
		// Add the IP to the blocklist table
		addCmd := exec.Command("sudo", "pfctl", "-t", "blocklist", "-T", "add", ip)
		addOutput, addErr := addCmd.CombinedOutput()
		if addErr != nil {
			return fmt.Errorf("failed to add IP %s to blocklist with pfctl: %v (output: %s)", ip, addErr, string(addOutput))
		}
	}

	// Make sure pf is enabled
	enableCmd := exec.Command("sudo", "pfctl", "-e")
	enableOutput, enableErr := enableCmd.CombinedOutput()

	// Ensure the blocklist table is referenced in the pf rules
	// This adds a rule to block all traffic to/from the IPs in the blocklist table
	ruleCmd := exec.Command("sudo", "sh", "-c",
		`echo "block drop in quick from <blocklist> to any" | sudo pfctl -f - -a blocklist`)
	ruleOutput, ruleErr := ruleCmd.CombinedOutput()

	if enableErr != nil {
		return fmt.Errorf("failed to enable pf: %v (output: %s)", enableErr, string(enableOutput))
	}
	if ruleErr != nil {
		return fmt.Errorf("failed to add blocklist rule with pfctl: %v (output: %s)", ruleErr, string(ruleOutput))
	}
	return nil
}

// unblockIPDarwin unblocks an IP on macOS using pfctl
func unblockIPDarwin(ip string) error {
	cmd := exec.Command("sudo", "pfctl", "-t", "blocklist", "-T", "delete", ip)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unblock IP %s with pfctl: %v (output: %s)", ip, err, string(output))
	}
	return nil
}

// blockIPWindows blocks an IP on Windows using netsh
func blockIPWindows(ip string) error {
	// Block inbound connections
	inCmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		"name=BlockIP_In_"+ip,
		"dir=in",
		"action=block",
		"remoteip="+ip,
		"enable=yes",
		"profile=any")
	inOutput, inErr := inCmd.CombinedOutput()
	if inErr != nil {
		return fmt.Errorf("failed to block inbound connections from IP %s with netsh: %v (output: %s)", ip, inErr, string(inOutput))
	}

	// Block outbound connections
	outCmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		"name=BlockIP_Out_"+ip,
		"dir=out",
		"action=block",
		"remoteip="+ip,
		"enable=yes",
		"profile=any")
	outOutput, outErr := outCmd.CombinedOutput()
	if outErr != nil {
		return fmt.Errorf("failed to block outbound connections to IP %s with netsh: %v (output: %s)", ip, outErr, string(outOutput))
	}
	return nil
}

// unblockIPWindows unblocks an IP on Windows using netsh
func unblockIPWindows(ip string) error {
	// Remove inbound rule
	inCmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule",
		"name=BlockIP_In_"+ip)
	inOutput, inErr := inCmd.CombinedOutput()

	// Remove outbound rule
	outCmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule",
		"name=BlockIP_Out_"+ip)
	outOutput, outErr := outCmd.CombinedOutput()

	// Return an error if either command failed
	if inErr != nil {
		return fmt.Errorf("failed to unblock inbound connections from IP %s with netsh: %v (output: %s)", ip, inErr, string(inOutput))
	}
	if outErr != nil {
		return fmt.Errorf("failed to unblock outbound connections to IP %s with netsh: %v (output: %s)", ip, outErr, string(outOutput))
	}
	return nil
}
