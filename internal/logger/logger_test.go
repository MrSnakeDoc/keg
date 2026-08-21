package logger

import (
	"io"
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
