package middleware

import (
	"net/http"
)

// HTTPMiddleware is a middleware for standard HTTP servers
type HTTPMiddleware struct {
	middleware *Middleware
}

// HTTP returns an HTTPMiddleware for the given Middleware
func (m *Middleware) HTTP() *HTTPMiddleware {
	return &HTTPMiddleware{
		middleware: m,
	}
}

// NewHTTP creates a new HTTP middleware
func NewHTTP(options Options) (*HTTPMiddleware, error) {
	middleware, err := New(options)
	if err != nil {
		return nil, err
	}

	return &HTTPMiddleware{
		middleware: middleware,
	}, nil
}

// Handler wraps an http.Handler with the middleware
func (m *HTTPMiddleware) Handler(next http.Handler) http.Handler {
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

// Middleware returns a function that can be used with http.HandleFunc
func (m *HTTPMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.Handler(next).ServeHTTP(w, r)
	}
}

// CleanupExpired manually triggers cleanup of expired blocks
func (m *HTTPMiddleware) CleanupExpired() error {
	return m.middleware.CleanupExpired()
}

// GetOptions returns the middleware options
func (m *HTTPMiddleware) GetOptions() Options {
	return m.middleware.options
}
