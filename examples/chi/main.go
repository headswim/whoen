package main

import (
	"fmt"
	"log"
	"net/http"

	"whoen/config"
	"whoen/middleware"
)

// Note: In a real implementation, you would import the Chi router:
// import "github.com/go-chi/chi/v5"

func main() {
	// Load configuration
	cfg := config.DefaultConfig()

	// Create middleware
	options := middleware.DefaultOptions()
	options.Config = cfg

	chiMiddleware, err := middleware.NewChi(options)
	if err != nil {
		log.Fatalf("Error creating middleware: %v", err)
	}

	// In a real implementation, you would create a Chi router:
	// r := chi.NewRouter()
	// r.Use(chiMiddleware.Middleware)

	// For this example, we'll use the standard HTTP server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Chi example, %s!", r.URL.Path[1:])
	})

	// Wrap the handler with the middleware
	http.Handle("/", chiMiddleware.Middleware(handler))

	// Start the server
	fmt.Println("Starting server on :8081...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
