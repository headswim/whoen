package matcher

// Whitelist is a list of IP addresses that should never be blocked
var Whitelist = []string{
	"8.8.8.8", // Google's DNS as a safe default example
}
