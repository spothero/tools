package utils

import (
	"context"
	"fmt"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
)

// CatchPanic prevents a panic from escaping and allows us to log an error for it. This is helpful to keep track of
// async goroutines that do not return anything to the main app thread. We want to catch the panic and log an error in
// these cases and want to prevent them from crashing the app. Ideally this is used in a debugging process that
// leads to the resolution of the panic case.
func CatchPanic(ctx context.Context) {
	if err := recover(); err != nil {
		log.Get(ctx).Error("caught panic", zap.String("error", fmt.Sprint(err)))
	}
}

// LogPanic produces an error log to record the issue and repanics to allow the app to crash. This is helpful to get a
// definitive error log before the uncaught panic crashes the app. Ideally this is used in a debugging process that
// leads to the resolution of the panic case.
func LogPanic(ctx context.Context) {
	if err := recover(); err != nil {
		log.Get(ctx).Error("uncaught panic", zap.String("error", fmt.Sprint(err)))
		panic(fmt.Sprint(err))
	}
}
