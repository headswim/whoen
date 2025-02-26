package main

import (
	"fmt"
	"log"
	"net/http"

	"whoen/config"
	"whoen/middleware"
)

// Note: In a real implementation, you would import the Gin framework:
// import "github.com/gin-gonic/gin"

func main() {
	// Load configuration
	cfg := config.DefaultConfig()

	// Create middleware
	options := middleware.DefaultOptions()
	options.Config = cfg

	// In a real implementation, you would use this:
	// ginMiddleware, err := middleware.NewGin(options)
	// if err != nil {
	//     log.Fatalf("Error creating middleware: %v", err)
	// }

	// In a real implementation, you would create a Gin router:
	// r := gin.Default()
	// r.Use(func(c *gin.Context) {
	//     ginContext := &middleware.GinContext{
	//         Request: c.Request,
	//         Writer:  c.Writer,
	//     }
	//     ginMiddleware.Middleware()(ginContext)
	//     if c.Writer.Written() {
	//         c.Abort()
	//         return
	//     }
	//     c.Next()
	// })

	// For this example, we'll use the standard HTTP server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Gin example, %s!", r.URL.Path[1:])
	})

	// Since we can't use the Gin middleware directly with the standard HTTP server,
	// we'll use the HTTP middleware instead for this example
	httpMiddleware, err := middleware.NewHTTP(options)
	if err != nil {
		log.Fatalf("Error creating HTTP middleware: %v", err)
	}

	http.Handle("/", httpMiddleware.Handler(handler))

	// Start the server
	fmt.Println("Starting server on :8082...")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
