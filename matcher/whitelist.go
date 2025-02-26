package matcher

// Whitelist is a list of IP addresses that should never be blocked
var Whitelist = []string{
	// Google DNS
	"8.8.8.8",
	"8.8.4.4",

	// Cloudflare DNS
	"1.1.1.1",
	"1.0.0.1",

	// Localhost
	"127.0.0.1",
	"::1",

	// Common private network ranges (examples)
	// "192.168.1.100", // Example: Your admin IP
	// "10.0.0.5",      // Example: Your monitoring system
}
