package log

import "github.com/sirupsen/logrus"

type M map[string]any

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

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}
