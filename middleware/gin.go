package middleware

import (
	"net/http"
)

// GinHandlerFunc is a type alias for Gin's handler function
type GinHandlerFunc func(*GinContext)

// GinContext is a type alias for Gin's context
type GinContext struct {
	Request *http.Request
	Writer  http.ResponseWriter
	// Other fields would be here in a real implementation
	// but we're keeping it simple for this example
}

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
func (m *GinMiddleware) Middleware() GinHandlerFunc {
	return func(c *GinContext) {
		blocked, err := m.middleware.HandleRequest(c.Request)
		if err != nil {
			m.middleware.logger.Printf("Error handling request: %v", err)
			c.Writer.WriteHeader(http.StatusInternalServerError)
			c.Writer.Write([]byte("Internal Server Error"))
			return
		}

		if blocked {
			c.Writer.WriteHeader(http.StatusForbidden)
			c.Writer.Write([]byte("Forbidden"))
			return
		}

		// Continue processing the request
		// In a real implementation, this would call c.Next()
	}
}
