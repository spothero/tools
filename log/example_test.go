package log

import (
    "fmt"
    "context"
)

func ExampleCreateLogger() {
  c := Config{UseDevelopmentLogger: true}
  c.InitializeLogger()
  logger := Get(context.Background())
  fmt.Printf("%v", logger != nil)
  // Output: true
}

