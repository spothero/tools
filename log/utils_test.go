// Copyright 2023 SpotHero
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// makeLoggerObservable makes the module logger observable for testing functions
func makeLoggerObservable(t *testing.T, level zapcore.Level) *observer.ObservedLogs {
	// Override the global logger with the observable
	core, recordedLogs := observer.New(level)
	c := &Config{Cores: []zapcore.Core{core}}
	err := c.InitializeLogger()
	require.NoError(t, err)
	loggerMutex.Lock()
	logger = zap.New(core)
	loggerMutex.Unlock()
	return recordedLogs
}

// verifyLogContext verifies that the zap logger is set on the given context
func verifyLogContext(t *testing.T, ctx context.Context) {
	_, ok := ctx.Value(logKey).(*zap.Logger)
	assert.True(t, ok)
}
