package middleware

import (
	"net/http"
)

// HTTPMiddleware is a middleware for standard net/http
type HTTPMiddleware struct {
	middleware *Middleware
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

// Handler returns a http.Handler that wraps the provided handler
func (m *HTTPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		blocked, err := m.middleware.HandleRequest(r)
		if err != nil {
			m.middleware.logger.Printf("Error handling request: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if blocked {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// HandlerFunc returns a http.HandlerFunc that wraps the provided handler
func (m *HTTPMiddleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blocked, err := m.middleware.HandleRequest(r)
		if err != nil {
			m.middleware.logger.Printf("Error handling request: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if blocked {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
