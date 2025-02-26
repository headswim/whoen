package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GinMiddleware is a middleware for the Gin framework
type GinMiddleware struct {
	middleware *Middleware
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
		blocked, err := m.middleware.HandleRequest(c.Request)
		if err != nil {
			m.middleware.logger.Printf("Error handling request: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Internal Server Error",
			})
			return
		}

		if blocked {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Forbidden",
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
