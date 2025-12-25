package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func Init(appEnv string) {
	Log = logrus.New()

	Log.SetOutput(os.Stdout)

	if appEnv == "development" {
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		Log.SetLevel(logrus.DebugLevel)
	} else {
		Log.SetFormatter(&logrus.JSONFormatter{})
		Log.SetLevel(logrus.InfoLevel)
	}

	Log.Info("Logger initialized")
}
