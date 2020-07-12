package log

import (
    "fmt"
    "context"
)

// This example demonstrates how to initialize and create a logger.
func ExampleCreateLogger() {
  c := Config{UseDevelopmentLogger: true}
  c.InitializeLogger()

  logger := Get(context.Background())
  fmt.Printf("%T", logger)
  // Output: *zap.Logger
}
