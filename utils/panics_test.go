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

func TestLogPanic(t *testing.T) {
	t.Run("panic is not caught", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()
		defer LogPanic(context.Background())
		panic("this passes through")
	})
}
