package trafilatura

import "github.com/sirupsen/logrus"

func logInfo(opts Options, format string, args ...interface{}) {
	if opts.EnableLog {
		logrus.Infof(format, args...)
	}
}

func logError(opts Options, format string, args ...interface{}) {
	if opts.EnableLog {
		logrus.Errorf(format, args...)
	}
}

func logWarn(opts Options, format string, args ...interface{}) {
	if opts.EnableLog {
		logrus.Warnf(format, args...)
	}
}
