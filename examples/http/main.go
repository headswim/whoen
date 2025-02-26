package main

import (
	"fmt"
	"log"
	"net/http"

	"whoen/config"
	"whoen/middleware"
)

func main() {
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
