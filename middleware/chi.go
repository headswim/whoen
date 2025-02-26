package middleware

import (
	"net/http"
)

// ChiMiddleware is a middleware for the Chi router
type ChiMiddleware struct {
	middleware *Middleware
}

// Chi returns a ChiMiddleware for the given Middleware
func (m *Middleware) Chi() *ChiMiddleware {
	return &ChiMiddleware{
		middleware: m,
	}
}

// NewChi creates a new Chi middleware
func NewChi(options Options) (*ChiMiddleware, error) {
	middleware, err := New(options)
	if err != nil {
		return nil, err
	}

	return &ChiMiddleware{
		middleware: middleware,
	}, nil
}

// Middleware returns a Chi middleware function
func (m *ChiMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		clientIP, err := getClientIP(r)
		if err != nil {
			m.middleware.logger.Printf("Error getting client IP: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		// Check if the request is malicious
		blocked, err := m.middleware.HandleRequest(r)
		if err != nil {
			m.middleware.logger.Printf("Error handling request from %s: %v", clientIP, err)
			next.ServeHTTP(w, r)
			return
		}

		if blocked {
			m.middleware.logger.Printf("Blocked malicious request from %s to %s", clientIP, r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Forbidden: This request has been blocked for security reasons"))
			return
		}

		// Continue processing the request
		next.ServeHTTP(w, r)
	})
}

// CleanupExpired manually triggers cleanup of expired blocks
func (m *ChiMiddleware) CleanupExpired() error {
	return m.middleware.CleanupExpired()
}

// GetOptions returns the middleware options
func (m *ChiMiddleware) GetOptions() Options {
	return m.middleware.options
}
