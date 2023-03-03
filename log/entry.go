package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	logger *logrus.Logger
	Entry  *PkgErrorEntry
)

func callerPrettyfier(f *runtime.Frame) (string, string) {
	fileName := fmt.Sprintf("%s:%d", f.File, f.Line)
	funcName := f.Function
	list := strings.Split(funcName, "/")
	if len(list) > 0 {
		funcName = list[len(list)-1]
	}
	return funcName, fileName
}

// for stdout
func callerFormatter(f *runtime.Frame) string {
	funcName, fileName := callerPrettyfier(f)
	return " @" + funcName + " " + fileName
}

func init() {
	logger = logrus.New()
	logger.SetReportCaller(true)
	logger.SetLevel(logrus.DebugLevel)
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: false,
		CallerPrettyfier: callerPrettyfier,
	})

	e := logrus.NewEntry(logger).
		WithField("devlang", "golang")

	Entry = &PkgErrorEntry{
		Entry: e,
		Depth: 3,
	}
}

// AddField add log fields
func AddField(key, value string) {
	if len(key) == 0 {
		return
	}
	if len(value) == 0 {
		return
	}
	Entry.Entry = Entry.Entry.WithField(key, value)
}

// DisableDefaultConsole 取消默认的控制台输出
func DisableDefaultConsole() {
	logger.SetOutput(io.Discard)
}

type PkgErrorEntry struct {
	*logrus.Entry

	// Depth defines how much of the stacktrace you want.
	Depth int
}

func (e *PkgErrorEntry) WithError(err error) *logrus.Entry {
	out := e.Entry

	type causer interface {
		Cause() error
	}

	// This is dirty pkg/errors.
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	var st stackTracer

	for {
		_, ok := err.(stackTracer)
		if ok {
			st = err.(stackTracer)
		}
		c, ok := err.(causer)
		if ok {
			err = c.Cause()
		} else {
			break
		}
	}

	if st != nil {
		depth := 3
		if e.Depth != 0 {
			depth = e.Depth
		}
		var stack string
		for i, f := range st.StackTrace() {
			if i >= depth {
				break
			}
			stack += fmt.Sprintf("\n%+v", f)
		}
		out = out.WithField("stack", stack)
	}

	return out.WithError(err)
}

func getHookLevel(level int) []logrus.Level {
	if level < 0 || level > 5 {
		level = 5
	}
	return logrus.AllLevels[:level+1]
}
