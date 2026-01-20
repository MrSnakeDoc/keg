package logger

import (
	"fmt"
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
	useJSON  atomic.Bool // Track if we're in JSON mode
)

// Configure sets up the global logger.
func Configure(opts Options) {
	mu.Lock()
	defer mu.Unlock()

	if opts.Out != nil {
		out = opts.Out
	}

	// Store JSON mode for fast path
	useJSON.Store(opts.JSON)

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

// ---- Public logging API (optimized for minimal lock contention) ----

func Info(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	// Format message outside the lock
	formatted := formatMessage("✨ ", msg, args...)

	// Fast path: direct write for non-JSON mode
	if !useJSON.Load() {
		mu.RLock()
		colorized := p.Info(formatted)
		mu.RUnlock()
		writeDirect(colorized)
		return
	}

	// JSON mode: use zap
	mu.RLock()
	zlog.Info(formatted)
	mu.RUnlock()
}

func Success(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	formatted := formatMessage("✅ ", msg, args...)

	if !useJSON.Load() {
		mu.RLock()
		colorized := p.Success(formatted)
		mu.RUnlock()
		writeDirect(colorized)
		return
	}

	mu.RLock()
	zlog.Info(p.Success(formatted))
	mu.RUnlock()
}

func LogError(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	formatted := formatMessage("❌ ", msg, args...)

	if !useJSON.Load() {
		mu.RLock()
		colorized := p.Error(formatted)
		mu.RUnlock()
		writeDirect(colorized)
		return
	}

	mu.RLock()
	zlog.Error(p.Error(formatted))
	mu.RUnlock()
}

func Warn(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	formatted := formatMessage("⚠️ ", msg, args...)

	if !useJSON.Load() {
		mu.RLock()
		colorized := p.Warning(formatted)
		mu.RUnlock()
		writeDirect(colorized)
		return
	}

	mu.RLock()
	zlog.Warn(p.Warning(formatted))
	mu.RUnlock()
}

func Fatal(msg string, args ...interface{}) {
	if !ensureReady() {
		os.Exit(1)
	}
	formatted := formatMessage("💥 ", msg, args...)

	mu.RLock()
	zlog.Fatal(p.Error(formatted))
	mu.RUnlock()
}

func WarnInline(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}
	formatted := formatMessage("⚠️ ", msg, args...)
	mu.RLock()
	colorized := p.Warning(formatted)
	mu.RUnlock()
	writeDirect(colorized)
}

func Debug(msg string, args ...interface{}) {
	if !ensureReady() {
		return
	}

	// Check if debug level is enabled
	mu.RLock()
	if curLevel > zapcore.DebugLevel {
		mu.RUnlock()
		return
	}
	mu.RUnlock()

	formatted := formatMessage("🛠️  ", msg, args...)

	if !useJSON.Load() {
		mu.RLock()
		colorized := p.Debug(formatted)
		mu.RUnlock()
		writeDirect(colorized)
		return
	}

	mu.RLock()
	zlog.Debug(p.Debug(formatted))
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

func RenderRow(table *tablewriter.Table, name, ver, status, pkgType string) error {
	return table.Append([]string{name, ver, status, pkgType})
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

// formatMessage formats a message with optional args outside of any lock.
func formatMessage(prefix, msg string, args ...interface{}) string {
	if len(args) > 0 {
		return prefix + fmt.Sprintf(msg, args...)
	}
	return prefix + msg
}

// writeDirect writes directly to output with minimal locking.
// Used for non-JSON mode to bypass zap overhead.
// Expects msg to be already fully formatted (emoji + color).
func writeDirect(msg string) {
	mu.RLock()
	w := out
	mu.RUnlock()

	if w == nil {
		w = os.Stdout
	}

	// Write directly without holding lock
	_, _ = fmt.Fprintln(w, msg)
}
