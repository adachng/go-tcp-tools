// MIT License
//
// Copyright (c) 2026-present adachng (github.com/adachng)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package logger

import (
	"fmt"
	"log"
	"sync"
)

type LogLevel int

const (
	LevelTraceL3 LogLevel = iota
	LevelTraceL2
	LevelTraceL1
	LevelDebug
	LevelInfo
	LevelNotice
	LevelError
	LevelPanic
	LevelFatal
)

var logLevelName = map[LogLevel]string{
	LevelTraceL3: "TRACEL3",
	LevelTraceL2: "TRACEL2",
	LevelTraceL1: "TRACEL1",
	LevelDebug:   "DEBUG",
	LevelInfo:    "INFO",
	LevelNotice:  "NOTICE",
	LevelError:   "ERROR",
	LevelPanic:   "PANIC",
	LevelFatal:   "FATAL",
}

var logLevelShortName = map[LogLevel]string{
	LevelTraceL3: "T3",
	LevelTraceL2: "T2",
	LevelTraceL1: "T1",
	LevelDebug:   "D",
	LevelInfo:    "I",
	LevelNotice:  "N",
	LevelError:   "E",
	LevelPanic:   "P",
	LevelFatal:   "F",
}

// Implements [fmt.Stringer] for [LogLevel].
//
// This should only be used by client code external to this package.
// Internal code reads concurrent-safe flag to determine
// whether to log with short form (e.g. "T3") or long form (e.g. "TRACEL3").
//
// See https://pkg.go.dev/fmt#Stringer for more details.
func (l *LogLevel) String() string {
	// Note that map keys are not automatically dereferenced.
	return logLevelName[*l]
}

// Additional settings that extend [log.Logger].
//
// The additional settings include enclosing prefix and suffix for the log level,
// as well as a flag to indicate preference for short form of the log level string.
//
// The default prefix and suffix are "{" and "}" respectively.
// Which means the log level is indicated in the message as "{INFO}" or "{I}"
// depending on the flag.
type Config struct {
	LogLevelPrefix string
	LogLevelSuffix string

	LogLevel LogLevel

	IsPreferShortName bool
}

// Concurrency-safe extended logger struct.
type Logger struct {
	mu sync.Mutex

	c Config
	u *log.Logger
}

// Singleton instance of Logger for convenience.
var singleton *Logger = &Logger{
	c: Config{
		LogLevelPrefix:    "{",
		LogLevelSuffix:    "}",
		LogLevel:          LevelDebug,
		IsPreferShortName: true,
	},
	u: log.Default(),
}

// Returns the singleton instance in this package.
func Default() *Logger {
	return singleton
}

// Configure the additional settings of the logger.
func (l *Logger) Configure(c Config) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.c = c
}

// Sets the current log LogLevel.
// Mutes log entries lower than current LogLevel.
//
// Differs from Configure() for convenience to not modify other settings.
func (l *Logger) SetLogLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.c.LogLevel = level
}

func (l *Logger) GetUnderlying() *log.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.u
}

// Sets the underlying log.Logger.
func (l *Logger) SetUnderlying(underlying *log.Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if underlying == nil {
		l.u = log.Default()
	} else {
		l.u = underlying
	}
}

// Unexported function for convenient log message forming.
//
// This internal function assumes internal caller has already locked the mutex.
func (l *Logger) getLogMsg(levelStr, msg string) string {
	ret := fmt.Sprintf("%s%s%s %s",
		l.c.LogLevelPrefix,
		levelStr,
		l.c.LogLevelSuffix,
		msg)
	return ret
}

// Unexported function for convenient log message printing.
// Returns string of the log message just printed.
//
// This internal function assumes internal caller has already locked the mutex.
func (l *Logger) logEntry(levelStr string, msg string) {
	str := l.getLogMsg(levelStr, msg)

	l.u.Print(str)
}

// Get the log LogLevel string depending on the configured flag for preferring short form.
//
// This internal function assumes internal caller has already locked the mutex.
func (l *Logger) getLevelStr(LogLevel LogLevel) string {
	if l.c.IsPreferShortName {
		return logLevelShortName[LogLevel]
	} else {
		return logLevelName[LogLevel]
	}
}

// Log trace LogLevel 3 message with usage equivalent to Print().
func (l *Logger) TraceL3(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelTraceL3

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprint(v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log trace LogLevel 3 message with usage equivalent to Printf().
func (l *Logger) TraceL3f(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelTraceL3

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log trace LogLevel 2 message with usage equivalent to Print().
func (l *Logger) TraceL2(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelTraceL2

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprint(v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log trace LogLevel 2 message with usage equivalent to Printf().
func (l *Logger) TraceL2f(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelTraceL2

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log trace LogLevel 1 message with usage equivalent to Print().
func (l *Logger) TraceL1(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelTraceL1

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprint(v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log trace LogLevel 1 message with usage equivalent to Printf().
func (l *Logger) TraceL1f(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelTraceL1

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log debug message with usage equivalent to Print().
func (l *Logger) Debug(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelDebug

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprint(v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log debug message with usage equivalent to Printf().
func (l *Logger) Debugf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelDebug

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log info message with usage equivalent to Print().
func (l *Logger) Info(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelInfo

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprint(v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log info message with usage equivalent to Printf().
func (l *Logger) Infof(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelInfo

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log notice message with usage equivalent to Print().
func (l *Logger) Notice(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelNotice

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprint(v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log notice message with usage equivalent to Printf().
func (l *Logger) Noticef(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelNotice

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log error message with usage equivalent to Print().
func (l *Logger) Error(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelError

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprint(v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log error message with usage equivalent to Printf().
func (l *Logger) Errorf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelError

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)
	l.logEntry(l.getLevelStr(funcLevel), msg)
}

// Log panic message with usage equivalent to Print().
//
// Beware that this function involves calling panic() for its message.
func (l *Logger) Panic(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelPanic

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprint(v...)
	str := l.getLogMsg(l.getLevelStr(funcLevel), msg)
	l.u.Panic(str)
}

// Log fatal message with usage equivalent to Printf().
//
// Beware that this function involves calling panic() for its message.
func (l *Logger) Panicf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelPanic

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)
	str := l.getLogMsg(l.getLevelStr(funcLevel), msg)
	l.u.Panic(str)
}

// Log fatal message with usage equivalent to Print().
//
// Beware that this function involves calling os.Exit(1) for its message.
func (l *Logger) Fatal(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelFatal

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprint(v...)
	str := l.getLogMsg(l.getLevelStr(funcLevel), msg)
	l.u.Fatal(str)
}

// Log fatal message with usage equivalent to Printf().
//
// Beware that this function involves calling os.Exit(1) for its message.
func (l *Logger) Fatalf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	const funcLevel LogLevel = LevelFatal

	if l.c.LogLevel > funcLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)
	str := l.getLogMsg(l.getLevelStr(funcLevel), msg)
	l.u.Fatal(str)
}
