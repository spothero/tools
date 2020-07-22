package log

import (
	"context"
	"fmt"
	"net/http"
)

// Initialize log package
func ExampleConfig() {
	c := Config{UseDevelopmentLogger: true}
	err := c.InitializeLogger()
	fmt.Printf("%v", err)
	// Output: nil
}

// Create logger
func ExampleGet() {
	logger := Get(context.Background())
	fmt.Printf("%T", logger)
	// Output: *zap.Logger
}

// Get logging middleware function for HTTP Server
func ExampleHTTPServerMiddleware() {
	f := HTTPServerMiddleware
	fmt.Printf("%T", f)
	// Output: func(http.Handler) http.Handler
}

// Get logging http.RoundTripper
func ExampleRoundTripper() {
	rt := (http.RoundTripper)(RoundTripper{})
	fmt.Printf("%T", rt)
	// Output: log.RoundTripper
}
