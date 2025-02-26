# Whoen

<p align="center">
  <strong>Malicious Request Detection Middleware for Go</strong>
</p>

<p align="center">
  <a href="#overview">Overview</a> •
  <a href="#installation">Installation</a> •
  <a href="#integration">Integration</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#key-features">Key Features</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#examples">Examples</a> •
  <a href="#license">License</a>
</p>

## Overview

Whoen is a lightweight, configurable middleware layer for Go web applications that detects and blocks malicious requests. It provides protection against common attack vectors by identifying suspicious request patterns and implementing configurable blocking strategies.

Designed to be framework-agnostic, Whoen integrates seamlessly with standard Go HTTP servers as well as popular frameworks like Gin and Chi.

## Installation

```bash
go get github.com/headswim/whoen
```

## Integration

Adding Whoen to your application is simple:

1. **Install the package** (in your project directory)
   ```bash
   go get github.com/headswim/whoen
   ```

2. **Initialize Whoen in your main function** (at the beginning, before server setup)
   ```go
   // In func main():
   if err := whoen.RestoreBlocks("blocked_ips.json"); err != nil {
       log.Printf("Error restoring blocks: %v", err)
   }
   ```

3. **Create the middleware** (in your server initialization code)
   ```go
   // In func main() or your server setup function:
   mw, err := whoen.New() // For default configuration
   if err != nil {
       log.Fatalf("Error creating middleware: %v", err)
   }
   ```

4. **Add the middleware to your HTTP stack** (where you define your routes)
   ```go
   // For standard HTTP (in your server setup):
   http.Handle("/", mw.HTTP().Handler(yourHandler))
   
   // For Gin (after creating your router):
   router.Use(mw.Gin().Middleware())
   
   // For Chi (after creating your router):
   router.Use(mw.Chi().Middleware)
   ```

5. **Set up storage directory** (one-time setup, before running your app)
   ```bash
   # Option 1: System-wide directory (requires sudo)
   sudo mkdir -p /var/lib/whoen
   sudo chown <your-app-user> /var/lib/whoen
   
   # Option 2: User directory
   mkdir -p ~/.whoen
   
   # Option 3: Current directory (no setup needed)
   # Whoen will use the current directory if options 1 and 2 fail
   ```

That's it! Your application is now protected against malicious requests.

## Quick Start

### Standard HTTP

```go
package main

import (
    "log"
    "net/http"
    "github.com/headswim/whoen"
)

func main() {
    // Initialize Whoen at the beginning of main
    if err := whoen.RestoreBlocks("blocked_ips.json"); err != nil {
        log.Printf("Error restoring blocks: %v", err)
    }

    // Create middleware with default configuration
    mw, err := whoen.New()
    if err != nil {
        log.Fatalf("Error creating middleware: %v", err)
    }

    // Create a simple handler
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })

    // Wrap the handler with the middleware
    http.Handle("/", mw.HTTP().Handler(handler))

    // Start the server
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Gin Framework

```go
package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/headswim/whoen"
)

func main() {
    // Initialize Whoen at the beginning of main
    if err := whoen.RestoreBlocks("blocked_ips.json"); err != nil {
        log.Printf("Error restoring blocks: %v", err)
    }

    // Create middleware with default configuration
    mw, err := whoen.New()
    if err != nil {
        log.Fatalf("Error creating middleware: %v", err)
    }

    // Create a Gin router
    r := gin.Default()

    // Use the middleware (before defining routes)
    r.Use(mw.Gin().Middleware())

    // Add your routes
    r.GET("/", func(c *gin.Context) {
        c.String(200, "Hello, World!")
    })

    // Start the server
    r.Run(":8080")
}
```

### Chi Router

```go
package main

import (
    "log"
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/headswim/whoen"
)

func main() {
    // Initialize Whoen at the beginning of main
    if err := whoen.RestoreBlocks("blocked_ips.json"); err != nil {
        log.Printf("Error restoring blocks: %v", err)
    }

    // Create middleware with default configuration
    mw, err := whoen.New()
    if err != nil {
        log.Fatalf("Error creating middleware: %v", err)
    }

    // Create a Chi router
    r := chi.NewRouter()

    // Use the middleware (before defining routes)
    r.Use(mw.Chi().Middleware)

    // Add your routes
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })

    // Start the server
    log.Fatal(http.ListenAndServe(":8080", r))
}
```

## Key Features

### 🔍 Intelligent Detection
- Automatically identifies suspicious request patterns
- Blocks common attack vectors (admin panels, config files, etc.)
- Zero configuration required for basic protection

### 🛡️ Flexible Protection
- Configurable grace period before blocking
- Temporary or permanent IP blocks
- Increasing timeout durations for repeat offenders

### 🔄 Seamless Integration
- Works with standard Go HTTP servers
- Native support for Gin and Chi frameworks
- Minimal code changes required

### 💾 Persistent Blocking
- Blocks persist across application restarts
- OS-level firewall integration for robust protection
- JSON-based storage for block information

## Configuration

Whoen works out of the box with sensible defaults, but you can customize it to suit your needs:

```go
// Create custom configuration
cfg := whoen.Config{
    // Where to store blocked IPs (defaults to ~/.whoen/blocked_ips.json)
    BlockedIPsFile: "blocked_ips.json",
    
    // How many suspicious requests to allow before blocking (default: 1)
    GracePeriod: 3,
    
    // Use temporary blocks instead of permanent bans (default: true)
    TimeoutEnabled: true,
    
    // How long to block for (default: 24 hours)
    TimeoutDuration: 1 * time.Hour,
    
    // How timeout increases for repeat offenders (default: "linear")
    // Options: "linear" or "geometric"
    TimeoutIncrease: "geometric",
    
    // Enable automatic cleanup of expired blocks (default: true)
    CleanupEnabled: true,
    
    // How often to clean up expired blocks (default: 1 hour)
    CleanupInterval: 30 * time.Minute,
}

// Create middleware with custom configuration
mw, err := whoen.NewWithConfig(cfg)
```

### Whitelisting IPs

To prevent certain IPs from being blocked:

```go
// Add IPs to the whitelist (in your main function)
whoen.AddToWhitelist(
    "192.168.1.10",  // Internal admin
    "10.0.0.5"       // Monitoring system
)
```

By default, localhost and common DNS servers are already whitelisted.

## Examples

Complete examples for each supported framework can be found in the `examples/` directory:

- `examples/http/` - Standard Go HTTP server example
- `examples/gin/` - Gin framework example
- `examples/chi/` - Chi router example

Each example demonstrates:
1. Initializing Whoen at application startup
2. Creating and configuring the middleware
3. Adding the middleware to your HTTP stack
4. Defining routes and starting the server

## License

This project is licensed under the MIT License - see the LICENSE file for details. 