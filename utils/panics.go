package utils

import (
	"context"
	"fmt"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
)

// CatchPanic prevents a panic from escaping and allows us to log an error for it
func CatchPanic(ctx context.Context) {
	if err := recover(); err != nil {
		log.Get(ctx).Error("caught panic", zap.String("error", fmt.Sprint(err)))
	}
}

