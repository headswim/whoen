package main

import (
	"fmt"
	"log"
	"net/http"

	"whoen/config"
	"whoen/middleware"

	"github.com/gin-gonic/gin"
)

// Note: In a real implementation, you would import the Gin framework:
// import "github.com/gin-gonic/gin"

func main() {
	// Restore OS-level blocks from previous runs (IMPORTANT)
	if err := middleware.RestoreBlocks("blocked_ips.json", "linux"); err != nil {
		log.Printf("Error restoring blocks: %v", err)
	}

	// Load configuration
	cfg := config.DefaultConfig()

	// Uncomment the following lines to enable periodic cleanup
	// cfg.CleanupEnabled = true
	// cfg.CleanupInterval = 1 * time.Hour

	// Create middleware
	options := middleware.DefaultOptions()
	options.Config = cfg

	// You can also enable cleanup directly in the options
	// options.CleanupEnabled = true
	// options.CleanupInterval = 1 * time.Hour

	// Create Gin middleware
	ginMiddleware, err := middleware.NewGin(options)
	if err != nil {
		log.Fatalf("Error creating middleware: %v", err)
	}

	// Create a Gin router
	r := gin.Default()

	// Use the middleware
	r.Use(ginMiddleware.Middleware())

	// Add a route
	r.GET("/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Hello from Gin example, %s!", name),
		})
	})

	// Add a route to manually trigger cleanup
	r.GET("/admin/cleanup", func(c *gin.Context) {
		if err := ginMiddleware.CleanupExpired(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Error cleaning up expired blocks: %v", err),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Cleanup completed successfully",
		})
	})

	// Start the server
	fmt.Println("Starting server on :8082...")
	log.Fatal(r.Run(":8082"))
}
