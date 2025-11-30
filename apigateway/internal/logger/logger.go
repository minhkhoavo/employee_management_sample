package logger

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var globalLogger zerolog.Logger

// InitLogging configures the global zerolog logger.
var once sync.Once

func InitLogging(logFilePath string) {
	once.Do(func() {
		var writers []io.Writer
		writers = append(writers, os.Stdout)

		if logFilePath != "" {
			file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
			if err != nil {
				// Fallback to stdout only if file cannot be opened
				// We can't use the logger yet, so just print to stderr
				os.Stderr.WriteString("Failed to open log file: " + err.Error() + "\n")
			} else {
				writers = append(writers, file)
			}
		}

		multi := zerolog.MultiLevelWriter(writers...)
		logger := zerolog.New(multi).With().Timestamp().Logger()
		logger = logger.Level(zerolog.InfoLevel)
		globalLogger = logger
		// Set the global logger used by the zerolog/log package for convenience.
		log.Logger = logger
	})
}

// WithLogger returns a new context containing the logger with additional fields.
func WithLogger(ctx context.Context, fields map[string]interface{}) context.Context {
	// Attach fields to the logger and store in context.
	l := globalLogger.With().Fields(fields).Logger()
	return l.WithContext(ctx)
}

// getLogger extracts the zerolog logger from the context, falling back to the global logger.
func getLogger(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx)
	// zerolog.Ctx returns a disabled logger if none is in context
	if l.GetLevel() == zerolog.Disabled {
		return &globalLogger
	}
	return l
}

// DebugLog logs a debug level message.
func DebugLog(ctx context.Context, msg string, args ...interface{}) {
	getLogger(ctx).Debug().Msgf(msg, args...)
}

// InfoLog logs an info level message.
func InfoLog(ctx context.Context, msg string, args ...interface{}) {
	getLogger(ctx).Info().Msgf(msg, args...)
}

// WarnLog logs a warning level message.
func WarnLog(ctx context.Context, msg string, args ...interface{}) {
	getLogger(ctx).Warn().Msgf(msg, args...)
}

// ErrorLog logs an error level message.
func ErrorLog(ctx context.Context, msg string, args ...interface{}) {
	l := getLogger(ctx)
	if len(args) > 0 {
		// If the first argument is an error, log it with Err for structured output
		if err, ok := args[0].(error); ok {
			l.Error().Err(err).Msg(msg)
		} else {
			l.Error().Msgf(msg, args...)
		}
	} else {
		l.Error().Msg(msg)
	}
}
