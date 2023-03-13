package log

import (
	"io"

	"github.com/sirupsen/logrus"
)

type M map[string]any

type Level uint32

const (
	LevelTrace = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

func SetOutput(w io.Writer) {
	logrus.SetOutput(w)
}

func SetLevel(level Level) {
	var l logrus.Level
	switch level {
	case LevelTrace:
		l = logrus.TraceLevel
	case LevelInfo:
		l = logrus.InfoLevel
	case LevelWarn:
		l = logrus.WarnLevel
	case LevelError:
		l = logrus.ErrorLevel
	case LevelFatal:
		l = logrus.FatalLevel
	case LevelPanic:
		l = logrus.PanicLevel
	}
	logrus.SetLevel(l)
}

func Infow(msg string, args M) {
	logrus.WithFields(logrus.Fields(args)).Info(msg)
}

func Debugw(msg string, args M) {
	logrus.WithFields(logrus.Fields(args)).Debug(msg)
}

func Warnw(msg string, args M) {
	logrus.WithFields(logrus.Fields(args)).Warn(msg)
}

func Errorw(msg string, args M) {
	logrus.WithFields(logrus.Fields(args)).Error(msg)
}

func Tracew(msg string, args M) {
	logrus.WithFields(logrus.Fields(args)).Trace(msg)
}

func Fatalw(msg string, args M) {
	logrus.WithFields(logrus.Fields(args)).Fatal(msg)
}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}
