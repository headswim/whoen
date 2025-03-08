package matcher

// Patterns is a list of predefined malicious path patterns used to detect malicious requests
var Patterns = []string{
	"/.env",
	"/wp-admin",
	"/admin",
	"/config",
	"/backup",
	"/.git",
	"/wp-login.php",
	"/phpmyadmin",
	"/administrator",
	"/jenkins",
	"/.htaccess",
	"/.htpasswd",
	"/server-status",
	"/server-info",
	"/web.config",
	"/elmah.axd",
	"/trace.axd",
	"/install",
	"/setup",
	"/console",
	"/wp-content/debug.log",
	"/api/swagger",
	"/api/docs",
	"/actuator",
	"/actuator/health",
	"/actuator/info",
	"/v1/metrics",
	"/v2/metrics",
	"/metrics",
	"/debug/vars",
	"/debug/pprof",
}
