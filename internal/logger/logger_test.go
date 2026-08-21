package logger

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRuntimeReconfigurationDoesNotDeadlock(t *testing.T) {
	Configure(Options{Level: "info", Out: io.Discard})

	done := make(chan struct{})
	go func() {
		SetLevel("debug")
		SetOutput(io.Discard)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("logger reconfiguration deadlocked")
	}
}

func TestSetOutputBeforeConfigureDoesNotDeadlock(t *testing.T) {
	mu.Lock()
	zlog = nil
	p = nil
	ready.Store(false)
	mu.Unlock()

	done := make(chan struct{})
	go func() {
		SetOutput(io.Discard)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("initial logger configuration deadlocked")
	}
}

func TestWarnInlineFormatsArguments(t *testing.T) {
	var output bytes.Buffer
	Configure(Options{Level: "info", Out: &output})

	WarnInline("package %s failed", "keg")

	if got, want := output.String(), "⚠️ package keg failed"; got != want {
		t.Fatalf("WarnInline output is %q, want %q", got, want)
	}
}

type concurrentWriter struct {
	active  atomic.Int32
	overlap atomic.Bool
}

func (w *concurrentWriter) Write(p []byte) (int, error) {
	if w.active.Add(1) > 1 {
		w.overlap.Store(true)
	}
	time.Sleep(time.Millisecond)
	w.active.Add(-1)
	return len(p), nil
}

func TestLoggerSerializesOutputWrites(t *testing.T) {
	output := &concurrentWriter{}
	Configure(Options{Level: "info", Out: output})

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			WarnInline("message")
		}()
	}
	wg.Wait()

	if output.overlap.Load() {
		t.Fatal("logger output writes overlapped")
	}
}
