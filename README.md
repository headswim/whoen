# Whoen

<p align="center">
  <img src="docs/logo.png" alt="Whoen Logo" width="200" />
</p>

<p align="center">
  <strong>Malicious Request Detection Middleware for Go</strong>
</p>

<p align="center">
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a> •
  <a href="#configuration">Configuration</a> •
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
    "net/http"
    "github.com/headswim/whoen/middleware"
)

func main() {
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
    "github.com/gin-gonic/gin"
    "github.com/headswim/whoen/middleware"
)

func main() {
    router := gin.Default()
    
    // Initialize and use Whoen middleware
    options := middleware.DefaultOptions()
    ginMiddleware, err := middleware.NewGin(options)
    if err != nil {
        log.Fatalf("Error creating middleware: %v", err)
    }
    
    router.Use(func(c *gin.Context) {
        ginContext := &middleware.GinContext{
            Request: c.Request,
            Writer:  c.Writer,
        }
        ginMiddleware.Middleware()(ginContext)
        if c.Writer.Written() {
            c.Abort()
            return
        }
        c.Next()
    })
    
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
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/headswim/whoen/middleware"
)

func main() {
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

## Features

### Malicious Request Detection

- Pattern-based detection of suspicious request paths
- Configurable matching strategies (exact, prefix, pattern)
- Extensible pattern library for common attack vectors

### Flexible Blocking Strategies

- Configurable grace period for first-time offenders
- Temporary timeouts with linear or geometric increase
- Permanent banning for persistent attackers
- IP-based blocking with persistence across server restarts

### Persistence and Monitoring

- JSON-based storage for blocked IPs and timeout tracking
- Detailed logging of detection and blocking events
- Thread-safe operations for high-concurrency environments

### Framework Support

- Standard Go HTTP server
- Gin framework
- Chi router

## Configuration

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

// Create middleware with custom options
httpMiddleware, err := middleware.NewHTTP(options)
if err != nil {
    log.Fatalf("Error creating middleware: %v", err)
}
```

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

Whitelisted IPs will bypass all blocking mechanisms and their requests will be allowed even if they match malicious patterns.

## JSON Data Files

Whoen uses JSON files for persistence:

### blocked_ips.json

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

## Project Structure

```
whoen/
├── config/
│   └── config.go       # Configuration handling (including defaults)
├── storage/
│   ├── storage.go      # Storage interface
│   └── json.go         # JSON implementation
├── middleware/
│   ├── middleware.go   # Core middleware logic + interface
│   ├── http.go         # Standard http adapter
│   ├── chi.go          # Chi adapter
│   └── gin.go          # Gin adapter
├── matcher/
│   ├── matcher.go      # Path matching interface
│   ├── patterns.go     # Pattern definitions to match
│   └── service.go      # Path matching implementation
├── blocker/
│   ├── blocker.go      # IP blocking interface
│   └── service.go      # Blocking implementation
├── examples/
│   ├── http/           # Standard http example
│   ├── gin/            # Gin example
│   └── chi/            # Chi example
├── go.mod              # Module definition
└── README.md           # Documentation
```

## Examples

Complete examples for each supported framework can be found in the `examples/` directory:

- `examples/http/` - Standard Go HTTP server example
- `examples/gin/` - Gin framework example
- `examples/chi/` - Chi router example

## Advanced Usage

### Custom Logger Integration

By default, Whoen uses its own internal logger. However, you can integrate with your application's logging system:

```go
// Create middleware with a custom logger
options := middleware.DefaultOptions()
options.Logger = yourLoggerImplementation

httpMiddleware, err := middleware.NewHTTP(options)
if err != nil {
    log.Fatalf("Error creating middleware: %v", err)
}
```

### Automatic Cleanup of Expired Blocks

By default, Whoen cleans up expired blocks when checking if an IP is blocked. However, this is a reactive approach and may leave some expired blocks in the system if they are not checked. For a more proactive approach, you can enable the periodic cleanup service by uncommenting the relevant code in the middleware:

```go
// In middleware/middleware.go, uncomment the following code in the New function:
cleanupTicker := time.NewTicker(1 * time.Hour)
go func() {
    for {
        select {
        case <-cleanupTicker.C:
            if err := m.CleanupExpired(); err != nil {
                m.logger.Printf("Error cleaning up expired blocks: %v", err)
            }
        }
    }
}()
```

This will run a background goroutine that periodically cleans up expired blocks, ensuring that both the storage and OS-level blocks are properly removed.

### Custom Malicious Path Patterns

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

## Contributing

Contributions are welcome, especially to `patterns.go`! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

