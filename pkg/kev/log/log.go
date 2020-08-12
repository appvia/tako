/**
 * Copyright 2020 Appvia Ltd <info@appvia.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

const (
	// DebugPrefix debug log level prefix
	DebugPrefix = "🔎"

	// InfoPrefix info log level prefix
	InfoPrefix = "💡"

	// WarnPrefix warn log level prefix
	WarnPrefix = "⚠️ "

	// ErrorPrefix error log level prefix
	ErrorPrefix = "✋"

	// FatalPrefix fatal log level prefix
	FatalPrefix = "😱"
)

var logger = &logrus.Logger{
	Out: os.Stdout,
	Formatter: &prefixed.TextFormatter{
		DisableTimestamp: true,
	},
	Hooks: make(logrus.LevelHooks),
	Level: logrus.InfoLevel,
}

var enableFileInfo = false

// Fields wraps logrus.Fields
type Fields logrus.Fields

// GetLogger returns underlying logrus logger
func GetLogger() *logrus.Logger {
	return logger
}

// SetLogLevel sets logging level
func SetLogLevel(level logrus.Level) {
	logger.Level = level
}

// SetLogFormatter sets logging formatter
func SetLogFormatter(formatter logrus.Formatter) {
	logger.Formatter = formatter
}

// SetOutput sets logger output
func SetOutput(out io.Writer) {
	logger.SetOutput(out)
}

// EnableFileInfo enables file information in log entry
func EnableFileInfo() {
	enableFileInfo = true
}

// DisableFileInfo disables file information in log entry
func DisableFileInfo() {
	enableFileInfo = false
}

// Debug logs a Debug message
func Debug(args ...interface{}) {
	logger.WithFields(decorate("debug")).Debug(args...)
}

// DebugWithFields logs a Debug message with fields
func DebugWithFields(f Fields, args ...interface{}) {
	logger.WithFields(decorate("debug", f)).Debug(args...)
}

// Debugf logs a Debug message
func Debugf(m string, args ...interface{}) {
	logger.WithFields(decorate("debug")).Debugf(m, args...)
}

// DebugfWithFields logs a Debug message with fields
func DebugfWithFields(f Fields, m string, args ...interface{}) {
	logger.WithFields(decorate("debug", f)).Debugf(m, args...)
}

// Info logs a Info message
func Info(args ...interface{}) {
	logger.WithFields(decorate("info")).Info(args...)
}

// InfoWithFields logs an Info message with fields
func InfoWithFields(f Fields, args ...interface{}) {
	logger.WithFields(decorate("info", f)).Info(args...)
}

// Infof logs a Info message
func Infof(m string, args ...interface{}) {
	logger.WithFields(decorate("info")).Infof(m, args...)
}

// InfofWithFields logs an Info message with fields
func InfofWithFields(f Fields, m string, args ...interface{}) {
	logger.WithFields(decorate("info", f)).Infof(m, args...)
}

// Warn logs a Warning message
func Warn(args ...interface{}) {
	logger.WithFields(decorate("warn")).Warn(args...)
}

// WarnWithFields logs a Warn message with fields
func WarnWithFields(f Fields, args ...interface{}) {
	logger.WithFields(decorate("warn", f)).Warn(args...)
}

// Warnf logs a Warning message
func Warnf(m string, args ...interface{}) {
	logger.WithFields(decorate("warn")).Warnf(m, args...)
}

// WarnfWithFields logs a Warn message with fields
func WarnfWithFields(f Fields, m string, args ...interface{}) {
	logger.WithFields(decorate("warn", f)).Warnf(m, args...)
}

// Error logs an Error message
func Error(args ...interface{}) {
	logger.WithFields(decorate("error")).Error(args...)
}

// ErrorWithFields logs an Error message with fields
func ErrorWithFields(f Fields, args ...interface{}) {
	logger.WithFields(decorate("error", f)).Error(args...)
}

// Errorf logs an Error message
func Errorf(m string, args ...interface{}) {
	logger.WithFields(decorate("error")).Errorf(m, args...)
}

// ErrorfWithFields logs an Error message with fields
func ErrorfWithFields(f Fields, m string, args ...interface{}) {
	logger.WithFields(decorate("error", f)).Errorf(m, args...)
}

// Fatal logs a fatal error
func Fatal(args ...interface{}) {
	logger.WithFields(decorate("fatal")).Fatal(args...)
}

// FatalWithFields logs a Fatal error with fields
func FatalWithFields(f Fields, args ...interface{}) {
	logger.WithFields(decorate("fatal", f)).Fatal(args...)
}

// Fatalf logs a fatal error
func Fatalf(m string, args ...interface{}) {
	logger.WithFields(decorate("fatal")).Fatalf(m, args...)
}

// FatalfWithFields logs a Fatal error with fields
func FatalfWithFields(f Fields, m string, args ...interface{}) {
	logger.WithFields(decorate("fatal", f)).Fatalf(m, args...)
}

// decorate adds extra fields based on the entry log level
func decorate(level string, f ...Fields) logrus.Fields {
	fields := Fields{}
	if len(f) > 0 {
		fields = f[0]
	}

	if fields["prefix"] == nil || fields["prefix"] == "" {
		switch level {
		case "debug":
			fields["prefix"] = DebugPrefix
		case "info":
			fields["prefix"] = InfoPrefix
		case "warn":
			fields["prefix"] = WarnPrefix
		case "error":
			fields["prefix"] = ErrorPrefix
		case "fatal":
			fields["prefix"] = FatalPrefix
		default:
		}
	}

	if enableFileInfo {
		fields["file"] = fileInfo(2)
	}

	return logrus.Fields(fields)
}

func fileInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		if slash >= 0 {
			file = file[slash+1:]
		}
	}
	return fmt.Sprintf("%s:%d", file, line)
}
