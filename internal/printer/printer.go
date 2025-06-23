package printer

import (
	"github.com/fatih/color"
)

type ColorPrinter struct {
	Success func(format string, a ...interface{}) string
	Error   func(format string, a ...interface{}) string
	Warning func(format string, a ...interface{}) string
	Info    func(format string, a ...interface{}) string
	Debug   func(format string, a ...interface{}) string
}

func NewColorPrinter() *ColorPrinter {
	return &ColorPrinter{
		Success: color.New(color.FgGreen).SprintfFunc(),
		Error:   color.New(color.FgRed).SprintfFunc(),
		Warning: color.New(color.FgYellow).SprintfFunc(),
		Info:    color.New(color.FgBlue).SprintfFunc(),
		Debug:   color.New(color.FgCyan).SprintfFunc(),
	}
}
