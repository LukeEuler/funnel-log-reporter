package log

import (
	"fmt"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

func AddConsoleOut(level int) {
	DisableDefaultConsole()
	logger.AddHook(newConsoleHook(level))
}

type consoleHook struct {
	formatter logrus.Formatter
	levels    []logrus.Level
}

func (c *consoleHook) Fire(entry *logrus.Entry) error {
	formatBytes, err := c.formatter.Format(entry)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unable to fortmat the log line on consoleHook %s", err)
		return err
	}
	fmt.Print(string(formatBytes))
	return nil
}

func (c *consoleHook) Levels() []logrus.Level {
	return c.levels
}

func newConsoleHook(level int) *consoleHook {
	// logrus.TextFormatter 不支持对 logrus.Fields 的value数据进行换行处理：https://github.com/sirupsen/logrus/issues/608
	// 所以换成使用 nested.Formatter：方便测试和线上查看定位问题
	plainFormatter := &nested.Formatter{
		NoFieldsColors:        true,
		CustomCallerFormatter: callerFormatter,
	}
	return &consoleHook{plainFormatter, getHookLevel(level)}
}
