package matcher

// Matcher defines the interface for path matching
type Matcher interface {
	// IsMalicious checks if a path is malicious
	IsMalicious(path string) bool

	// IsWhitelisted checks if an IP is in the whitelist
	IsWhitelisted(ip string) bool
}
