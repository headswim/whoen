package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/headswim/whoen"
)

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

	// Step 5: Create a handler
	helloHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})

	// Step 6: Wrap the handler with the middleware
	http.Handle("/", mw.HTTP().Handler(helloHandler))

	// Add a route to manually trigger cleanup
	http.HandleFunc("/admin/cleanup", func(w http.ResponseWriter, r *http.Request) {
		if err := mw.CleanupExpired(); err != nil {
			http.Error(w, fmt.Sprintf("Error cleaning up expired blocks: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Cleanup completed successfully")
	})

	// Step 7: Start the server
	fmt.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
