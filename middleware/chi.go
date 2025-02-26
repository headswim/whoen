package middleware

import (
	"net/http"
)

// ChiMiddleware is a middleware for the Chi router
type ChiMiddleware struct {
	middleware *Middleware
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
