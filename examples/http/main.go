package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/headswim/whoen/config"
	"github.com/headswim/whoen/middleware"
)

func main() {
	// Restore OS-level blocks from previous runs (IMPORTANT)
	if err := middleware.RestoreBlocks("blocked_ips.json", "linux"); err != nil {
		log.Printf("Error restoring blocks: %v", err)
	}

	// Load configuration
	cfg := config.DefaultConfig()

	// Create middleware
	options := middleware.DefaultOptions()
	options.Config = cfg

	httpMiddleware, err := middleware.NewHTTP(options)
	if err != nil {
		log.Fatalf("Error creating middleware: %v", err)
	}

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
	})

	// Wrap the handler with the middleware
	http.Handle("/", httpMiddleware.Handler(handler))

	// Start the server
	fmt.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
