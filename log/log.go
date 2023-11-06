package log

import (
	"io"
	"log/slog"
	"os"
)

type Logger struct {
	slogger *slog.Logger
}

type LoggerFormat uint32

const (
	JsonFormat LoggerFormat = iota
	TextFormat
)

// NewLogger creates a new logger with the given name and handler
func NewLogger(name string, handler slog.Handler) Logger {
	logger := slog.New(handler)
	return Logger{logger.With("log", name)}
}

// SubLogger returns a new logger with the given name as a sublogger
func (l Logger) SubLogger(name string) Logger {
	if l.slogger == nil { // no-op logger
		return Logger{}
	}
	return Logger{
		slogger: l.slogger.With("log", name),
	}
}

// Default returns a logger that logs to stdout with the
// TextFormat and log level Info. This is the recommended logger to use
// You can supply your own logger if you want to, using NewLogger and NewHandler
func Default() Logger {
	return NewLogger("[engine]", NewHandler(os.Stdout, TextFormat, slog.LevelInfo))
}

func NewHandler(w io.Writer, format LoggerFormat, loglevel slog.Level) slog.Handler {
	switch format {
	case JsonFormat:
		return slog.NewTextHandler(w, &slog.HandlerOptions{
			Level: loglevel,
		})
	case TextFormat:
		return slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: loglevel,
		})
	default:
		panic("unknown format") // can't happen
	}
}

func (l Logger) Infow(msg string, args ...any) {
	if l.slogger == nil {
		return
	}
	l.slogger.Info(msg, args)
}

func (l Logger) Debugw(msg string, args ...any) {
	if l.slogger == nil {
		return
	}
	l.slogger.Debug(msg, args)
}

func (l Logger) Warnw(msg string, args ...any) {
	if l.slogger == nil {
		return
	}
	l.slogger.Warn(msg, args)
}

func (l Logger) Errorw(msg string, args ...any) {
	if l.slogger == nil {
		return
	}
	l.slogger.Error(msg, args)
}
