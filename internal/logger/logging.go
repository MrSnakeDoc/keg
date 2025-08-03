package logger

import (
	"fmt"
	"os"

	"github.com/MrSnakeDoc/keg/internal/printer"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
)

var (
	log = func() *logrus.Logger {
		l := logrus.New()
		l.SetOutput(os.Stdout)
		l.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp:       true,
			DisableLevelTruncation: true,
			PadLevelText:           true,
			ForceColors:            true,
		})
		return l
	}()
	p = printer.NewColorPrinter()
)

func SetLevel(level string) {
	switch level {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}
}

func Info(msg string, args ...interface{}) {
	fmt.Println(p.Info("✨ "+msg, args...))
}

func Success(msg string, args ...interface{}) {
	fmt.Println(p.Success("✅ "+msg, args...))
}

func LogError(msg string, args ...interface{}) {
	formatted := p.Error("❌ "+msg, args...)
	log.Error(formatted)
}

func Warn(msg string, args ...interface{}) {
	fmt.Println(p.Warning("⚠️ "+msg, args...))
}

func WarnInline(msg string, args ...interface{}) {
	fmt.Print(p.Warning("⚠️ "+msg, args...))
}

func Debug(msg string, args ...interface{}) {
	if os.Getenv("KEG_DEBUG") != "" {
		fmt.Println(p.Debug("🛠️ "+msg, args...))
	}
}

func CreateTable(headers []string) *tablewriter.Table {
	table := tablewriter.NewTable(os.Stdout)

	table.Header(headers)
	return table
}
