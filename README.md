# Whoen

<p align="center">
  <img src="public/logo.png" alt="Whoen Logo" width="200" />
</p>

<p align="center">
  <strong>Malicious Request Detection Middleware for Go</strong>
</p>

<p align="center">
  <a href="#overview">Overview</a> •
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#key-features">Key Features</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#advanced-usage">Advanced Usage</a> •
  <a href="#architecture">Architecture</a> •
  <a href="#examples">Examples</a> •
  <a href="#contributing">Contributing</a> •
  <a href="#license">License</a>
</p>

## Overview

Whoen is a lightweight, configurable middleware layer for Go web applications that detects and blocks malicious requests. It provides protection against common attack vectors by identifying suspicious request patterns and implementing configurable blocking strategies.

Designed to be framework-agnostic, Whoen integrates seamlessly with standard Go HTTP servers as well as popular frameworks like Gin and Chi.

## Installation

```bash
go get github.com/headswim/whoen
```

## Quick Start

### Standard HTTP

```go
package main

import (
    "log"
    "net/http"
    "github.com/headswim/whoen/middleware"
)

func main() {
    // Restore OS-level blocks from previous runs (IMPORTANT)
    if err := middleware.RestoreBlocks("blocked_ips.json", "linux"); err != nil {
        log.Printf("Error restoring blocks: %v", err)
    }
    
    // Initialize Whoen middleware with default configuration
    options := middleware.DefaultOptions()
    httpMiddleware, err := middleware.NewHTTP(options)
    if err != nil {
        log.Fatalf("Error creating middleware: %v", err)
    }
    
    // Wrap your existing handler with Whoen middleware
    http.Handle("/", httpMiddleware.Handler(yourHandler))
    
    http.ListenAndServe(":8080", nil)
}
```

### Gin Framework

```go
package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/headswim/whoen/middleware"
)

func main() {
    // Restore OS-level blocks from previous runs (IMPORTANT)
    if err := middleware.RestoreBlocks("blocked_ips.json", "linux"); err != nil {
        log.Printf("Error restoring blocks: %v", err)
    }
    
    router := gin.Default()
    
    // Initialize and use Whoen middleware
    options := middleware.DefaultOptions()
    ginMiddleware, err := middleware.NewGin(options)
    if err != nil {
        log.Fatalf("Error creating middleware: %v", err)
    }
    
    router.Use(ginMiddleware.Middleware())
    
    // Your routes
    router.GET("/", func(c *gin.Context) {
        c.String(200, "Hello World")
    })
    
    router.Run(":8080")
}
```

### Chi Router

```go
package main

import (
    "log"
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/headswim/whoen/middleware"
)

func main() {
    // Restore OS-level blocks from previous runs (IMPORTANT)
    if err := middleware.RestoreBlocks("blocked_ips.json", "linux"); err != nil {
        log.Printf("Error restoring blocks: %v", err)
    }
    
    router := chi.NewRouter()
    
    // Initialize and use Whoen middleware
    options := middleware.DefaultOptions()
    chiMiddleware, err := middleware.NewChi(options)
    if err != nil {
        log.Fatalf("Error creating middleware: %v", err)
    }
    
    router.Use(chiMiddleware.Middleware)
    
    // Your routes
    router.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello World"))
    })
    
    http.ListenAndServe(":8080", router)
}
```

## Key Features

### Malicious Request Detection

- Pattern-based detection of suspicious request paths
- Extensible pattern library for common attack vectors
- Intelligent matching against known malicious patterns

### Flexible Blocking Strategies

- Configurable grace period for first-time offenders
- Temporary timeouts with linear or geometric increase
- Permanent banning for persistent attackers
- IP-based blocking with OS-level firewall integration
- Persistence of blocked IPs across application restarts

### Persistence and Monitoring

- JSON-based storage for blocked IPs and timeout tracking
- OS-level firewall integration (iptables, pfctl, netsh)
- Detailed logging of detection and blocking events
- Thread-safe operations for high-concurrency environments

### Framework Support

- Standard Go HTTP server
- Gin framework
- Chi router

## Configuration

### Basic Configuration

Whoen can be configured programmatically:

```go
// Create options with custom configuration
options := middleware.DefaultOptions()
options.GracePeriod = 3
options.TimeoutEnabled = true
options.TimeoutDuration = time.Hour * 12
options.TimeoutIncrease = "geometric"
options.Config.BlockedIPsFile = "custom_blocked_ips.json"
options.Config.LogFile = "custom_whoen.log"
options.Config.SystemType = "linux"  // Options: "linux", "mac", "windows"

// Create middleware with custom options
httpMiddleware, err := middleware.NewHTTP(options)
if err != nil {
    log.Fatalf("Error creating middleware: %v", err)
}
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `GracePeriod` | Number of malicious requests allowed before blocking | 1 |
| `TimeoutEnabled` | Whether to use temporary blocks instead of permanent bans | true |
| `TimeoutDuration` | Base duration for temporary blocks | 24 hours |
| `TimeoutIncrease` | How timeout duration increases for repeat offenders ("linear" or "geometric") | "linear" |
| `Config.BlockedIPsFile` | Path to the JSON file for storing blocked IPs | "blocked_ips.json" |
| `Config.LogFile` | Path to the log file | "whoen.log" |
| `Config.SystemType` | Operating system type for firewall commands ("linux", "mac", "windows") | "linux" |
| `Config.CleanupEnabled` | Whether to enable periodic cleanup of expired blocks | false |
| `Config.CleanupInterval` | Interval for periodic cleanup | 1 hour |

### Whitelisting IPs

Whoen supports whitelisting specific IP addresses that should never be blocked, regardless of their request patterns. This is useful for trusted sources like:

- Internal monitoring systems
- Admin IPs
- Trusted API clients

By default, the configuration includes Google's DNS (8.8.8.8) as an example of a whitelisted IP. You can customize the whitelist by modifying the `matcher/whitelist.go` file:

```go
// In matcher/whitelist.go
var Whitelist = []string{
    "192.168.1.10",  // Internal admin
    "10.0.0.5",      // Monitoring system
    "203.0.113.42",  // Trusted API client
}
```

**Note**: Changes to the whitelist require recompiling and restarting your application to take effect. The whitelist is loaded once during application startup.

Whitelisted IPs will bypass all blocking mechanisms and their requests will be allowed even if they match malicious patterns.

## Advanced Usage

### OS-Level Block Persistence

Whoen uses OS-level firewall commands (iptables on Linux, pfctl on macOS, netsh on Windows) to block malicious IPs. These blocks are stored in the JSON file for persistence, but the OS-level firewall rules themselves do not persist across system restarts.

**IMPORTANT**: To ensure that blocked IPs remain blocked after your application restarts, you must call the `RestoreBlocks` function at the beginning of your `main` function:

```go
func main() {
    // Restore OS-level blocks from previous runs
    // Parameters: 
    // - JSON file path: Path to the file where blocked IPs are stored
    // - System type: "linux", "mac", or "windows" (must match your OS)
    if err := middleware.RestoreBlocks("blocked_ips.json", "linux"); err != nil {
        log.Printf("Error restoring blocks: %v", err)
    }
    
    // Rest of your application initialization...
}
```

This function:
- Reads the blocked IPs from your JSON storage file
- Reapplies the OS-level firewall rules for all non-expired blocks
- Skips any blocks that have already expired
- Logs the number of restored and skipped blocks

**Note**: The OS-level blocking commands require sudo/administrator privileges. Make sure your application has the necessary permissions to execute these commands.

### Custom Logger Integration

By default, Whoen uses its own internal logger. However, you can integrate with your application's logging system by providing a custom logger that implements the standard Go `*log.Logger` interface:

```go
// Create a custom logger
customLogger := log.New(os.Stdout, "[my-app] ", log.LstdFlags)

// Create middleware with the custom logger
options := middleware.DefaultOptions()
options.Logger = customLogger

httpMiddleware, err := middleware.NewHTTP(options)
if err != nil {
    log.Fatalf("Error creating middleware: %v", err)
}
```

### Automatic Cleanup of Expired Blocks

By default, Whoen cleans up expired blocks when checking if an IP is blocked. However, this is a reactive approach and may leave some expired blocks in the system if they are not checked.

For a more proactive approach, you can enable the periodic cleanup service by setting `CleanupEnabled` to true in your configuration:

```go
// Enable cleanup in the configuration
cfg := config.DefaultConfig()
cfg.CleanupEnabled = true
cfg.CleanupInterval = 1 * time.Hour

// Or directly in the options
options := middleware.DefaultOptions()
options.CleanupEnabled = true
options.CleanupInterval = 1 * time.Hour
```

This will run a background goroutine that periodically cleans up expired blocks, ensuring that both the storage and OS-level blocks are properly removed.

### Malicious Pattern Detection

Whoen comes with a predefined list of malicious patterns that it checks against request paths:

- Common admin panels (wp-admin, phpmyadmin, etc.)
- Configuration files (.env, .htaccess, etc.)
- Debug endpoints (actuator, metrics, debug/pprof, etc.)
- Installation and setup paths
- And many more

You can extend or replace the default malicious path patterns:

```go
// File: matcher/patterns.go
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
    // ... more patterns

    // Add your custom patterns here
    "/your-custom-pattern",
    "/another-pattern",
}
```

### OS-Level Blocking Mechanisms

Whoen blocks IPs at the operating system level using the following mechanisms:

- **Linux**: Uses iptables to block IPs
- **macOS**: Uses pfctl (Packet Filter) to block IPs
- **Windows**: Uses Windows Firewall (netsh) to block IPs

### JSON Data Files

Whoen uses JSON files for persistence:

#### blocked_ips.json

This file stores information about blocked IPs, including their request counts, timeout status, and more:

```json
[
  {
    "ip": "192.168.1.100",
    "blocked_at": "2023-05-01T12:34:56Z",
    "blocked_until": "2023-05-02T12:34:56Z",
    "request_count": 5,
    "timeout_count": 1,
    "is_permanent": false,
    "last_request_path": "/.env"
  },
  {
    "ip": "10.0.0.5",
    "blocked_at": "2023-05-01T10:11:12Z",
    "request_count": 10,
    "timeout_count": 0,
    "is_permanent": true,
    "last_request_path": "/wp-admin"
  }
]
```

## Architecture

Whoen consists of several core components working together to provide comprehensive protection:

### Blocker

The blocker component handles the actual blocking of IP addresses at the operating system level:

- Interface defining methods for blocking, unblocking, and checking IP status
- Service implementing the blocking logic using OS-specific commands
- Support for both temporary timeouts and permanent bans

### Matcher

The matcher component identifies malicious requests:

- Pattern matching against known malicious request paths
- Whitelist management for trusted IPs
- Efficient pattern matching with O(1) lookups

### Storage

The storage component persists blocked IP information:

- Interface for storing and retrieving blocked IP data
- JSON implementation with periodic auto-saving
- Detailed tracking of each blocked IP's status

### Middleware

The middleware component ties everything together:

- Core logic for request processing and decision making
- Framework adapters for different web frameworks
- Extensive configuration options

## Examples

Complete examples for each supported framework can be found in the `examples/` directory:

- `examples/http/` - Standard Go HTTP server example
- `examples/gin/` - Gin framework example
- `examples/chi/` - Chi router example

## Contributing

Contributions are welcome, especially to `patterns.go`! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

