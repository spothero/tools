// Copyright 2019 SpotHero
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
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
)

// MockCore implements a zapcore.Core suitable for testing
type MockCore struct {
	zapcore.LevelEnabler
	mock.Mock
}

// With returns the mock core with the added fields
func (mc *MockCore) With(fields []zapcore.Field) zapcore.Core {
	return mc.Called(fields).Get(0).(zapcore.Core)
}

// Check is called before Write
func (mc *MockCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return mc.Called().Get(0).(*zapcore.CheckedEntry)
}

// Write logs the entry
func (mc *MockCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return mc.Called(ent, fields).Error(0)
}

// Sync syncs buffered logs
func (mc *MockCore) Sync() error {
	return mc.Called().Error(0)
}
