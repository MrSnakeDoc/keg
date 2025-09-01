package logger

import (
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/MrSnakeDoc/keg/internal/printer"
	"github.com/olekukonko/tablewriter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	Level string    // "debug","info","warn","error"
	JSON  bool      // JSON output (CI)
	Color bool      // colorize (console)
	Out   io.Writer // default os.Stdout
}

var (
	mu       sync.RWMutex
	zlog     *zap.SugaredLogger
	out      io.Writer = os.Stdout
	p        *printer.ColorPrinter
	curLevel = zapcore.InfoLevel
	ready    atomic.Bool
)

// Configure sets up the global logger.
func Configure(opts Options) {
	mu.Lock()
	defer mu.Unlock()

	if opts.Out != nil {
		out = opts.Out
	}
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = ""
	encCfg.LevelKey = ""
	encCfg.CallerKey = ""
	encCfg.MessageKey = "msg"

	var enc zapcore.Encoder
	if opts.JSON {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else {
		enc = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
	}

	level := parseLevel(opts.Level)
	ws := zapcore.AddSync(writerAdapter{out})
	core := zapcore.NewCore(enc, ws, level)

	base := zap.New(core)
	zlog = base.Sugar()

	if p == nil {
		p = printer.NewColorPrinter()
	}

	ready.Store(true)
}

// SetLevel adjusts current level at runtime ("debug","info","warn","error").
func SetLevel(level string) {
	mu.Lock()
	defer mu.Unlock()
	curLevel = parseLevel(level)
	if zlog == nil {
		Configure(Options{Level: level})
		return
	}
	// rebuild core with new level
	Configure(Options{Level: level, Out: out})
}

// SetOutput replaces the logger writer (use io.Discard in tests).
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	if w == nil {
		w = os.Stdout
	}
	out = w
	if zlog == nil {
		Configure(Options{})
		return
	}
	Configure(Options{Level: curLevel.String(), Out: out})
}

// UseTestMode silences logs during tests.
func UseTestMode() {
	Configure(Options{
		Level: "error", // only errors
		Color: false,
		JSON:  false,
		Out:   io.Discard,
	})
}

// Out returns the current output writer (for tables).
func Out() io.Writer {
	mu.RLock()
	defer mu.RUnlock()
	return out
}

// ---- Public logging API (kept stable) ----

func Info(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	mu.RLock()
	zlog.Infof(p.Info("✨ "+msg, args...))
	mu.RUnlock()
}

func Success(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	mu.RLock()
	zlog.Infof(p.Success("✅ "+msg, args...))
	mu.RUnlock()
}

func LogError(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	mu.RLock()
	zlog.Errorf(p.Error("❌ "+msg, args...))
	mu.RUnlock()
}

func Warn(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	mu.RLock()
	zlog.Warnf(p.Warning("⚠️ "+msg, args...))
	mu.RUnlock()
}

func WarnInline(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	// inline write directly to out to preserve non-line break semantics
	mu.RLock()
	defer mu.RUnlock()
	if out == nil {
		out = os.Stdout
	}
	_, _ = io.WriteString(out, p.Warning("⚠️ "+msg))
}

func Debug(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	mu.RLock()
	zlog.Debugf(p.Debug("🛠️ "+msg, args...))
	mu.RUnlock()
}

// ---- Tables ----

func CreateTable(headers []string) *tablewriter.Table {
	mu.RLock()
	defer mu.RUnlock()
	t := tablewriter.NewTable(out)
	t.Header(headers)
	return t
}

// ---- internals ----

type writerAdapter struct{ w io.Writer }

func (wa writerAdapter) Write(p []byte) (int, error) { return wa.w.Write(p) }

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		curLevel = zapcore.DebugLevel
	case "info", "":
		curLevel = zapcore.InfoLevel
	case "warn":
		curLevel = zapcore.WarnLevel
	case "error":
		curLevel = zapcore.ErrorLevel
	default:
		curLevel = zapcore.InfoLevel
	}
	return curLevel
}

// ---- helpers ----

func ensureReady() bool {
	if !ready.Load() {
		return false
	}
	if p == nil || zlog == nil {
		return false
	}
	return true
}
