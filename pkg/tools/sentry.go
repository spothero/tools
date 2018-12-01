package core

import (
	"fmt"
	"math"
	"time"

	"github.com/getsentry/raven-go"
	"go.uber.org/zap/zapcore"
)

// SentryConfig defines the necessary configuration for instantiating a Sentry Reporter
type SentryConfig struct {
	DSN         string
	AppPackage  string
	Environment string
	AppVersion  string
}

var appPackage string

// InitializeRaven Initializes the Raven client. This function should be called as soon as
// possible after the application configuration is loaded so that raven
// is setup.
func (sc *SentryConfig) InitializeRaven() {
	raven.SetDSN(sc.DSN)
	raven.SetEnvironment(sc.Environment)
	raven.SetRelease(sc.AppVersion)
	appPackage = sc.AppPackage
}

// SentryCore Implements a zapcore.Core that sends logged errors to Sentry
type SentryCore struct {
	zapcore.LevelEnabler
}

// With adds structured context to the Sentry Core
func (c *SentryCore) With(fields []zapcore.Field) zapcore.Core {
	return c.With(fields)
}

// Check must be called before calling Write. This determines whether or not logs are sent to
// Sentry
func (c *SentryCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// send error logs and above to Sentry
	if ent.Level >= zapcore.ErrorLevel {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write logs the entry and fields supplied at the log site and writes them to their destination
func (c *SentryCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	var severity raven.Severity
	switch ent.Level {
	case zapcore.DebugLevel:
		severity = raven.DEBUG
	case zapcore.InfoLevel:
		severity = raven.INFO
	case zapcore.WarnLevel:
		severity = raven.WARNING
	case zapcore.ErrorLevel:
		severity = raven.ERROR
	default:
		// captures Panic, DPanic, Fatal zapcore levels
		severity = raven.FATAL
	}

	// Add extra logged fields to the Sentry packet
	// This block was adapted from the way zap encodes messages internally
	// See https://github.com/uber-go/zap/blob/v1.7.1/zapcore/field.go#L107
	ravenExtra := make(map[string]interface{})
	for _, field := range fields {
		switch field.Type {
		case zapcore.ArrayMarshalerType:
			ravenExtra[field.Key] = field.Interface
		case zapcore.ObjectMarshalerType:
			ravenExtra[field.Key] = field.Interface
		case zapcore.BinaryType:
			ravenExtra[field.Key] = field.Interface
		case zapcore.BoolType:
			ravenExtra[field.Key] = field.Integer == 1
		case zapcore.ByteStringType:
			ravenExtra[field.Key] = field.Interface
		case zapcore.Complex128Type:
			ravenExtra[field.Key] = field.Interface
		case zapcore.Complex64Type:
			ravenExtra[field.Key] = field.Interface
		case zapcore.DurationType:
			ravenExtra[field.Key] = field.Integer
		case zapcore.Float64Type:
			ravenExtra[field.Key] = math.Float64frombits(uint64(field.Integer))
		case zapcore.Float32Type:
			ravenExtra[field.Key] = math.Float32frombits(uint32(field.Integer))
		case zapcore.Int64Type:
			ravenExtra[field.Key] = field.Integer
		case zapcore.Int32Type:
			ravenExtra[field.Key] = field.Integer
		case zapcore.Int16Type:
			ravenExtra[field.Key] = field.Integer
		case zapcore.Int8Type:
			ravenExtra[field.Key] = field.Integer
		case zapcore.StringType:
			ravenExtra[field.Key] = field.String
		case zapcore.TimeType:
			if field.Interface != nil {
				// Time has a timezone
				ravenExtra[field.Key] = time.Unix(0, field.Integer).In(field.Interface.(*time.Location))
			} else {
				ravenExtra[field.Key] = time.Unix(0, field.Integer)
			}
		case zapcore.Uint64Type:
			ravenExtra[field.Key] = uint64(field.Integer)
		case zapcore.Uint32Type:
			ravenExtra[field.Key] = uint32(field.Integer)
		case zapcore.Uint16Type:
			ravenExtra[field.Key] = uint16(field.Integer)
		case zapcore.Uint8Type:
			ravenExtra[field.Key] = uint8(field.Integer)
		case zapcore.UintptrType:
			ravenExtra[field.Key] = uintptr(field.Integer)
		case zapcore.ReflectType:
			ravenExtra[field.Key] = field.Interface
		case zapcore.NamespaceType:
			ravenExtra[field.Key] = field.Interface
		case zapcore.StringerType:
			ravenExtra[field.Key] = field.Interface.(fmt.Stringer).String()
		case zapcore.ErrorType:
			ravenExtra[field.Key] = field.Interface.(error).Error()
		case zapcore.SkipType:
			break
		default:
			ravenExtra[field.Key] = fmt.Sprintf("Unknown field type %v", field.Type)
		}
	}

	// Group logs with the same stack trace together unless there is no
	// stack trace, then group by message
	fingerprint := ent.Stack
	if ent.Stack == "" {
		fingerprint = ent.Message
	}
	packet := &raven.Packet{
		Message:     ent.Message,
		Timestamp:   raven.Timestamp(ent.Time),
		Level:       severity,
		Logger:      ent.LoggerName,
		Extra:       ravenExtra,
		Fingerprint: []string{fingerprint},
	}

	// get stack trace but omit the internal logging calls from the trace
	trace := raven.NewStacktrace(2, 5, []string{appPackage})
	packet.Interfaces = append(packet.Interfaces, trace)

	_, _ = raven.Capture(packet, nil)

	// level higher than error, (i.e. panic, fatal), the program might crash,
	// so block while raven sends the event
	if ent.Level > zapcore.ErrorLevel {
		raven.Wait()
	}
	return nil
}

// Sync flushes any buffered logs
func (c *SentryCore) Sync() error {
	raven.Wait()
	return nil
}
