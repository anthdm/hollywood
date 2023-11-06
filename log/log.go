package log

import (
	"io"
	"log/slog"
)

type Logger struct {
	slogger *slog.Logger
}

type LoggerFormat uint32

const (
	JsonFormat LoggerFormat = iota
	TextFormat
)

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

// New creates a new logger. You can specify an optional handler. If
// no handler is given, the logger will be a no-op logger, which is the default.
func NewLogger(name string, handler slog.Handler) Logger {
	logger := slog.New(handler)
	return Logger{logger.With("log", name)}
}

func (l Logger) SubLogger(name string) Logger {
	if l.slogger == nil { // no-op logger
		return Logger{}
	}
	return Logger{
		slogger: l.slogger.With("log", name),
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
