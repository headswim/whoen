package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/headswim/whoen"
)

// Note: In a real implementation, you would import the Gin framework:
// import "github.com/gin-gonic/gin"

func main() {
	// Step 1: Restore blocks from previous runs (IMPORTANT)
	// This ensures that IP blocks persist across application restarts
	if err := whoen.RestoreBlocks("blocked_ips.json"); err != nil {
		log.Printf("Error restoring blocks: %v", err)
	}

	// Step 2: Configure Whoen (optional)
	// You can use the default configuration or customize it
	cfg := whoen.Config{
		BlockedIPsFile:  "blocked_ips.json",
		GracePeriod:     3, // Block after 3 suspicious requests
		TimeoutEnabled:  true,
		TimeoutDuration: 1 * time.Hour, // Block for 1 hour
		TimeoutIncrease: "geometric",   // Increase timeout geometrically for repeat offenders
		CleanupEnabled:  true,
		CleanupInterval: 30 * time.Minute, // Clean up expired blocks every 30 minutes
	}

	// Step 3: Add custom IPs to the whitelist (optional)
	whoen.AddToWhitelist("192.168.1.100", "10.0.0.5")

	// Step 4: Create the middleware
	mw, err := whoen.NewWithConfig(cfg)
	if err != nil {
		log.Fatalf("Error creating Whoen middleware: %v", err)
	}

	// Step 5: Create a Gin router
	r := gin.Default()

	// Step 6: Use the middleware
	r.Use(mw.Gin().Middleware())

	// Step 7: Add your routes
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello, World!",
		})
	})

	// Add a route to manually trigger cleanup
	r.GET("/admin/cleanup", func(c *gin.Context) {
		if err := mw.CleanupExpired(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Error cleaning up expired blocks: %v", err),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Cleanup completed successfully",
		})
	})

	// Step 8: Start the server
	fmt.Println("Starting server on :8080...")
	log.Fatal(r.Run(":8080"))
}
