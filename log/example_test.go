package log

import (
    "context"
    "fmt"
)

// Initialize log package
func ExampleInitialize() {
  c := Config{UseDevelopmentLogger: true}
  c.InitializeLogger()
}

// Create a logger
func ExampleCreate() {
  logger := Get(context.Background())
  fmt.Printf("%T", logger)
  // Output: *zap.Logger
}

// Get logging middleware function for HTTP Server
func ExampleMiddlewareHTTP() {
  f := HTTPServerMiddleware
  fmt.Printf("%T", f)
  // Output: func(http.Handler) http.Handler
}
