package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	logger zerolog.Logger
}

type LoggerConfig struct {
	Level  string
	Format string
}

func New(config LoggerConfig) *Logger {
	// Set log level
	level := parseLogLevel(config.Level)
	zerolog.SetGlobalLevel(level)

	// Configure output format
	var logger zerolog.Logger
	if config.Format == "console" {
		logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	} else {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	return &Logger{logger: logger}
}

func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

func (l *Logger) With() zerolog.Context {
	return l.logger.With()
}

func (l *Logger) WithContext(ctx context.Context) zerolog.Logger {
	return l.logger.With().Logger()
}

func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// Global logger functions
var globalLogger *Logger

func Init(config LoggerConfig) {
	globalLogger = New(config)
}

func Info() *zerolog.Event {
	if globalLogger == nil {
		return log.Info()
	}
	return globalLogger.Info()
}

func Error() *zerolog.Event {
	if globalLogger == nil {
		return log.Error()
	}
	return globalLogger.Error()
}

func Debug() *zerolog.Event {
	if globalLogger == nil {
		return log.Debug()
	}
	return globalLogger.Debug()
}

func Warn() *zerolog.Event {
	if globalLogger == nil {
		return log.Warn()
	}
	return globalLogger.Warn()
}

func Fatal() *zerolog.Event {
	if globalLogger == nil {
		return log.Fatal()
	}
	return globalLogger.Fatal()
}
