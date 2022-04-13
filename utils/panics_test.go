package utils

import (
	"context"
	"testing"
)

func TestCatchPanic(t *testing.T) {
	t.Run("panic is caught", func(t *testing.T) {
		defer CatchPanic(context.Background())
		panic("this gets caught")
	})
}
