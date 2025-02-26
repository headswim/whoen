package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/headswim/whoen"
)

// Note: In a real implementation, you would import the Chi router:
// import "github.com/go-chi/chi/v5"

// This is a mock implementation of the Chi router for the example
type mockChiRouter struct {
	middlewares []func(http.Handler) http.Handler
	routes      map[string]http.HandlerFunc
}

func newMockRouter() *mockChiRouter {
	return &mockChiRouter{
		routes: make(map[string]http.HandlerFunc),
	}
}

func (r *mockChiRouter) Use(middleware func(http.Handler) http.Handler) {
	r.middlewares = append(r.middlewares, middleware)
}

func (r *mockChiRouter) Get(path string, handler http.HandlerFunc) {
	r.routes[path] = handler
}

func (r *mockChiRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler, ok := r.routes[req.URL.Path]
	if !ok {
		http.NotFound(w, req)
		return
	}

	// Apply middlewares in reverse order
	var h http.Handler = handler
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	h.ServeHTTP(w, req)
}

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

	// Step 5: Create a Chi router (using our mock implementation for the example)
	r := newMockRouter()

	// Step 6: Use the middleware
	r.Use(mw.Chi().Middleware)

	// Step 7: Add your routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// Add a route to manually trigger cleanup
	r.Get("/admin/cleanup", func(w http.ResponseWriter, r *http.Request) {
		if err := mw.CleanupExpired(); err != nil {
			http.Error(w, fmt.Sprintf("Error cleaning up expired blocks: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Cleanup completed successfully"))
	})

	// Step 8: Start the server
	fmt.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
