package logger

import (
	"io"
	"os"
)

var (
	FlagVerboseCount int  // -V, -VV, -VVV
	FlagQuiet        bool // --quiet/-q
	FlagSilent       bool // --silent/-s
	FlagJSON         bool // optionnel pour CI
)

func ConfigureLoggerFromFlags() {
	var out io.Writer = os.Stdout
	var level string
	switch {
	case FlagQuiet:
		level = "error"
		out = os.Stdout // errors only
	case FlagSilent:
		level = "error" // silent = no output at all, even errors
		out = io.Discard
	default:
		// map -V levels
		switch FlagVerboseCount {
		case 0:
			level = "info"
		case 1:
			level = "debug"
		default:
			level = "debug" // -VV, -VVV... keep debug (could add trace later)
		}
	}

	Configure(Options{
		Level: level,
		JSON:  FlagJSON,
		Color: !FlagJSON,
		Out:   out,
	})
}
