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

package sentry

import (
	"errors"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type StringerMock struct{}

func (sm StringerMock) String() string {
	return "stringer-mock"
}

type TransportMock struct{}

func (t TransportMock) Configure(_ sentry.ClientOptions) {}
func (t TransportMock) SendEvent(_ *sentry.Event)        {}
func (t TransportMock) Events() []*sentry.Event          { return nil }
func (t TransportMock) Flush(_ time.Duration) bool {
	return true
}

func TestWith(t *testing.T) {
	tests := []struct {
		name        string
		fields      []zapcore.Field
		expectEqual bool
	}{
		{
			"nil fields returns an unmodified core",
			nil,
			true,
		},
		{
			"no fields returns an unmodified core",
			[]zapcore.Field{},
			true,
		},
		{
			"added fields returns a unmodified core",
			[]zapcore.Field{
				{
					Key:    "test",
					Type:   zapcore.SkipType,
					String: "test",
				},
			},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := Core{}
			newCorePtr := c.With(test.fields)
			newCore := *(newCorePtr.(*Core))
			if test.expectEqual {
				assert.Equal(t, c, newCore)
			} else {
				assert.NotEqual(t, c, newCore)
			}
		})
	}
}

func TestWithMultipleTimes(t *testing.T) {
	tests := []struct {
		name                 string
		fields, appendFields []zapcore.Field
		expectEqual          bool
	}{
		{
			"append fields if with is called multiple times",
			[]zapcore.Field{
				{
					Key:    "test",
					Type:   zapcore.StringType,
					String: "test",
				},
			},
			[]zapcore.Field{
				{
					Key:    "test1",
					Type:   zapcore.StringType,
					String: "test",
				},
			},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := Core{}
			newCorePtr := c.With(test.fields).With(test.fields)
			totalFields := len(test.fields) + len(test.appendFields)
			assert.Equal(t, totalFields, len(newCorePtr.(*Core).withFields))
		})
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name  string
		entry zapcore.Entry
	}{
		{
			"nonerrors are not sent to sentry (the core is not added)",
			zapcore.Entry{Level: zapcore.InfoLevel},
		},
		{
			"errors are sent to sentry",
			zapcore.Entry{Level: zapcore.ErrorLevel},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ce := &zapcore.CheckedEntry{}
			assert.Equal(t, ce, (&Core{}).Check(test.entry, ce))
		})
	}
}

func TestWrite(t *testing.T) {
	transportMock := &TransportMock{}
	originalHub := sentry.CurrentHub().Clone()
	tests := []struct {
		core   *Core
		hub    *sentry.Hub
		entry  zapcore.Entry
		name   string
		fields []zapcore.Field
	}{
		{
			name:  "sentry core does not write debug logs",
			core:  &Core{},
			hub:   originalHub,
			entry: zapcore.Entry{Level: zapcore.DebugLevel},
		},
		{
			name:  "sentry core does not write info logs",
			core:  &Core{},
			hub:   originalHub,
			entry: zapcore.Entry{Level: zapcore.InfoLevel},
		},
		{
			name:  "sentry core does not write warn logs",
			core:  &Core{},
			hub:   originalHub,
			entry: zapcore.Entry{Level: zapcore.WarnLevel},
		},
		{
			name:  "sentry core does not write error logs",
			core:  &Core{},
			hub:   originalHub,
			entry: zapcore.Entry{Level: zapcore.ErrorLevel},
		},
		{
			name: "sentry core writes logs higher than error",
			core: &Core{
				withFields: []zapcore.Field{
					{Key: "0", Type: zapcore.ArrayMarshalerType},
					{Key: "1", Type: zapcore.ObjectMarshalerType},
					{Key: "2", Type: zapcore.BinaryType},
					{Key: "3", Type: zapcore.BoolType},
					{Key: "4", Type: zapcore.ByteStringType},
					{Key: "5", Type: zapcore.Complex128Type},
					{Key: "6", Type: zapcore.Complex64Type},
					{Key: "7", Type: zapcore.DurationType},
					{Key: "8", Type: zapcore.Float64Type},
					{Key: "9", Type: zapcore.Float32Type},
					{Key: "10", Type: zapcore.Int64Type},
					{Key: "11", Type: zapcore.Int32Type},
					{Key: "12", Type: zapcore.Int16Type},
					{Key: "13", Type: zapcore.Int8Type},
					{Key: "14", Type: zapcore.StringType},
					{Key: "15", Type: zapcore.TimeType},
					{Key: "16", Type: zapcore.Uint64Type},
					{Key: "17", Type: zapcore.Uint32Type},
					{Key: "18", Type: zapcore.Uint16Type},
					{Key: "19", Type: zapcore.Uint8Type},
					{Key: "20", Type: zapcore.UintptrType},
					{Key: "21", Type: zapcore.ReflectType},
					{Key: "22", Type: zapcore.NamespaceType},
					{Key: "23", Type: zapcore.StringerType, Interface: StringerMock{}},
					{Key: "24", Type: zapcore.ErrorType, Interface: errors.New("err")},
					{Key: "25", Type: zapcore.SkipType + 1},
					{Key: "26", Type: zapcore.TimeType, Interface: time.FixedZone("UTC", 0)},
					{
						Type:      zapcore.SkipType,
						Key:       loggerFieldKey,
						Interface: hubZapField{Hub: originalHub},
					},
					Tag("test", "123"),
					Fingerprint("test", "123"),
				},
			},
			hub:   sentry.NewHub(&sentry.Client{Transport: transportMock}, &sentry.Scope{}),
			entry: zapcore.Entry{Level: zapcore.PanicLevel},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, test.core.Write(test.entry, test.fields))
		})
	}
}

func TestHub(t *testing.T) {
	assert.Equal(
		t,
		zap.Field{
			Key:       loggerFieldKey,
			Type:      zapcore.SkipType,
			Interface: hubZapField{sentry.CurrentHub()},
		},
		Hub(sentry.CurrentHub()),
	)
}

func TestMarshalLogObject(t *testing.T) {
	assert.Nil(t, hubZapField{}.MarshalLogObject(zapcore.NewMapObjectEncoder()))
}

func TestTag(t *testing.T) {
	assert.Equal(t,
		zap.Field{
			Key:       "tag",
			String:    "value",
			Type:      zapcore.SkipType,
			Interface: TagType,
		},
		Tag("tag", "value"),
	)
}

func TestFingerprint(t *testing.T) {
	assert.Equal(t,
		zap.Field{
			Key:       "tag",
			String:    "value",
			Type:      zapcore.SkipType,
			Interface: FingerprintType,
		},
		Fingerprint("tag", "value"),
	)
}
