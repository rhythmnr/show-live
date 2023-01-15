package log

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"show-live/utils"
)

var Logger *logrus.Logger
var suffix string
var dir string
var currentLogFile string

func InitLogger(logSuffix string, logDir string) {
	Logger = &logrus.Logger{
		Level: logrus.DebugLevel,
		Formatter: &prefixed.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			ForceFormatting: true,
		},
	}
	suffix = logSuffix
	dir = logDir
	setLogoutput()
}

func logFileNow() string {
	now := time.Now()
	file := fmt.Sprintf("%d_%d_%d_%s.log", now.Year(), now.Month(), now.Day(), suffix)

	exists, err := utils.PathExists(dir)
	if err != nil {
		panic(err)
	}
	if !exists {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			panic(err)
		}
	}

	return file
}

func setLogoutput() {
	logFileNow := logFileNow()
	if currentLogFile != logFileNow {
		file, err := os.OpenFile(path.Join(dir, logFileNow), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		writers := []io.Writer{file}
		fileAndStdoutWriter := io.MultiWriter(writers...)
		Logger.SetOutput(fileAndStdoutWriter)
		currentLogFile = logFileNow
	}
}
