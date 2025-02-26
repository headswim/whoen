package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GinMiddleware is a middleware for the Gin framework
type GinMiddleware struct {
	middleware *Middleware
}

// Gin returns a GinMiddleware for the given Middleware
func (m *Middleware) Gin() *GinMiddleware {
	return &GinMiddleware{
		middleware: m,
	}
}

// NewGin creates a new Gin middleware
func NewGin(options Options) (*GinMiddleware, error) {
	middleware, err := New(options)
	if err != nil {
		return nil, err
	}

	return &GinMiddleware{
		middleware: middleware,
	}, nil
}

// Middleware returns a Gin middleware function
func (m *GinMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := c.ClientIP()

		// Check if the request is malicious
		blocked, err := m.middleware.HandleRequest(c.Request)
		if err != nil {
			m.middleware.logger.Printf("Error handling request from %s: %v", clientIP, err)
			c.Next() // Continue processing the request even if there's an error
			return
		}

		if blocked {
			m.middleware.logger.Printf("Blocked malicious request from %s to %s", clientIP, c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "This request has been blocked for security reasons",
			})
			return
		}

		// Continue processing the request
		c.Next()
	}
}

// CleanupExpired manually triggers cleanup of expired blocks
func (m *GinMiddleware) CleanupExpired() error {
	return m.middleware.CleanupExpired()
}

// GetOptions returns the middleware options
func (m *GinMiddleware) GetOptions() Options {
	return m.middleware.options
}
