// Copyright 2020 SpotHero
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
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// duration to wait to flush events to sentry
const flushTimeout = 2 * time.Second

// Core Implements a zapcore.Core that sends logged errors to Sentry
type Core struct {
	zapcore.LevelEnabler
	withFields []zapcore.Field
}

// With adds structured context to the Sentry Core
func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	if len(fields) == 0 {
		return c
	}
	clonedLogger := *c
	clonedLogger.withFields = fields
	return &clonedLogger
}

// Check must be called before calling Write. This determines whether or not logs are sent to
// Sentry
func (c *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// send error logs and above to Sentry
	if ent.Level >= zapcore.ErrorLevel {
		return ce.AddCore(ent, c)
	}
	return ce
}

// filter out function calls from this module and from the logger in stack traces
// reported to sentry
var stacktraceModulesToIgnore = []*regexp.Regexp{
	regexp.MustCompile(`github\.com/spothero/tools/sentry*`),
	regexp.MustCompile(`github\.com/uber-go/zap*`),
	regexp.MustCompile(`go\.uber\.org/zap*`),
}

// Write logs the entry and fields supplied at the log site and writes them to their destination. If a
// Sentry Hub field is present in the fields, that Hub will be used for reporting to Sentry, otherrwise
// the default Sentry Hub will be used.
func (c *Core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	var severity sentry.Level
	switch ent.Level {
	case zapcore.DebugLevel:
		severity = sentry.LevelDebug
	case zapcore.InfoLevel:
		severity = sentry.LevelInfo
	case zapcore.WarnLevel:
		severity = sentry.LevelWarning
	case zapcore.ErrorLevel:
		severity = sentry.LevelError
	default:
		// captures Panic, DPanic, Fatal zapcore levels
		severity = sentry.LevelFatal
	}

	// Add extra logged fields to the Sentry packet
	// This block was adapted from the way zap encodes messages internally
	// See https://github.com/uber-go/zap/blob/v1.7.1/zapcore/field.go#L107
	sentryExtra := make(map[string]interface{})
	mergedFields := fields
	if len(c.withFields) > 0 {
		mergedFields = append(mergedFields, c.withFields...)
	}
	hub := sentry.CurrentHub()
	for _, field := range mergedFields {
		switch field.Type {
		case zapcore.ArrayMarshalerType:
			sentryExtra[field.Key] = field.Interface
		case zapcore.ObjectMarshalerType:
			sentryExtra[field.Key] = field.Interface
		case zapcore.BinaryType:
			sentryExtra[field.Key] = field.Interface
		case zapcore.BoolType:
			sentryExtra[field.Key] = field.Integer == 1
		case zapcore.ByteStringType:
			sentryExtra[field.Key] = field.Interface
		case zapcore.Complex128Type:
			sentryExtra[field.Key] = field.Interface
		case zapcore.Complex64Type:
			sentryExtra[field.Key] = field.Interface
		case zapcore.DurationType:
			sentryExtra[field.Key] = field.Integer
		case zapcore.Float64Type:
			sentryExtra[field.Key] = math.Float64frombits(uint64(field.Integer))
		case zapcore.Float32Type:
			sentryExtra[field.Key] = math.Float32frombits(uint32(field.Integer))
		case zapcore.Int64Type:
			sentryExtra[field.Key] = field.Integer
		case zapcore.Int32Type:
			sentryExtra[field.Key] = field.Integer
		case zapcore.Int16Type:
			sentryExtra[field.Key] = field.Integer
		case zapcore.Int8Type:
			sentryExtra[field.Key] = field.Integer
		case zapcore.StringType:
			sentryExtra[field.Key] = field.String
		case zapcore.TimeType:
			if field.Interface != nil {
				// Time has a timezone
				sentryExtra[field.Key] = time.Unix(0, field.Integer).In(field.Interface.(*time.Location))
			} else {
				sentryExtra[field.Key] = time.Unix(0, field.Integer)
			}
		case zapcore.Uint64Type:
			sentryExtra[field.Key] = uint64(field.Integer)
		case zapcore.Uint32Type:
			sentryExtra[field.Key] = uint32(field.Integer)
		case zapcore.Uint16Type:
			sentryExtra[field.Key] = uint16(field.Integer)
		case zapcore.Uint8Type:
			sentryExtra[field.Key] = uint8(field.Integer)
		case zapcore.UintptrType:
			sentryExtra[field.Key] = uintptr(field.Integer)
		case zapcore.ReflectType:
			sentryExtra[field.Key] = field.Interface
		case zapcore.NamespaceType:
			sentryExtra[field.Key] = field.Interface
		case zapcore.StringerType:
			sentryExtra[field.Key] = field.Interface.(fmt.Stringer).String()
		case zapcore.ErrorType:
			sentryExtra[field.Key] = field.Interface.(error).Error()
		case zapcore.SkipType:
			if field.Key == loggerFieldKey {
				if h, ok := field.Interface.(hubZapField); ok {
					hub = h.Hub
				}
			}
		default:
			sentryExtra[field.Key] = fmt.Sprintf("Unknown field type %v", field.Type)
		}
	}

	// Group logs with the same stack trace together unless there is no
	// stack trace, then group by message
	fingerprint := ent.Stack
	if ent.Stack == "" {
		fingerprint = ent.Message
	}
	event := sentry.NewEvent()
	event.Message = ent.Message
	event.Level = severity
	event.Logger = ent.LoggerName
	event.Timestamp = ent.Time.Unix()
	event.Extra = sentryExtra
	event.Fingerprint = []string{fingerprint}
	stackTrace := sentry.NewStacktrace()
	filteredFrames := make([]sentry.Frame, 0, len(stackTrace.Frames))
	for _, frame := range stackTrace.Frames {
		ignoreFrame := false
		for _, pattern := range stacktraceModulesToIgnore {
			if pattern.MatchString(frame.Module) {
				ignoreFrame = true
				break
			}
		}
		if !ignoreFrame {
			filteredFrames = append(filteredFrames, frame)
		}
	}
	event.Threads = []sentry.Thread{{
		Stacktrace: &sentry.Stacktrace{
			Frames: filteredFrames,
		},
		Current: true,
	}}

	hub.CaptureEvent(event)

	// level higher than error, (i.e. panic, fatal), the program might crash,
	// so block while sentry sends the event
	if ent.Level > zapcore.ErrorLevel {
		hub.Flush(flushTimeout)
	}
	return nil
}

// Sync flushes any buffered logs
func (c *Core) Sync() error {
	if !sentry.Flush(flushTimeout) {
		return fmt.Errorf("timed out waiting for Sentry flush")
	}
	return nil
}

const loggerFieldKey = "sentry"

type hubZapField struct {
	*sentry.Hub
}

// Hub attaches a Sentry hub to the logger such that if the logger ever logs an
// error, request context can be sent to Sentry.
func Hub(hub *sentry.Hub) zapcore.Field {
	// This is a hack in order to pass an arbitrary object (in this case a Sentry Hub) through the logger so
	// that it can be pulled out in the custom Zap core. The way this works is the sentry Hub is wrapped in a type
	// that implements Zap's ObjectMarshaler as a no-op. That object gets set on the zap.Field's Interface key
	// The Type key is set to SkipType and the Key field is set to some unique value. This makes it so the built-in
	// logger cores ignore the field, but in the custom Sentry core above we can check if the Key matches and try
	// to pull the Sentry hub out of the field.
	return zap.Field{
		Key:       loggerFieldKey,
		Type:      zapcore.SkipType,
		Interface: hubZapField{hub},
	}
}

// MarshalLogObject implements Zap's ObjectMarshaler interface but is a no-op
// since we don't actually want to add anything from the Sentry Hub to the log.
func (f hubZapField) MarshalLogObject(_ zapcore.ObjectEncoder) error {
	return nil
}
