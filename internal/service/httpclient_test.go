package service

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

type trackingBody struct {
	io.Reader
	closed atomic.Bool
}

func (b *trackingBody) Close() error {
	b.closed.Store(true)
	return nil
}

func TestHandleResponseClosesRejectedBodies(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		contentType string
		length      string
		maxBytes    int64
		wantResult  bool
	}{
		{name: "not modified", status: http.StatusNotModified, wantResult: true},
		{name: "rate limited", status: http.StatusTooManyRequests},
		{name: "server error", status: http.StatusBadGateway},
		{name: "wrong content type", status: http.StatusOK, contentType: "text/plain"},
		{name: "too large", status: http.StatusOK, contentType: "application/json", length: "10", maxBytes: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &trackingBody{Reader: strings.NewReader("payload")}
			resp := &http.Response{
				StatusCode: tt.status,
				Header:     make(http.Header),
				Body:       body,
			}
			if tt.contentType != "" {
				resp.Header.Set("Content-Type", tt.contentType)
			}
			if tt.length != "" {
				resp.Header.Set("Content-Length", tt.length)
			}

			cancelled := false
			result, err := handleResponse(resp, tt.maxBytes, func() { cancelled = true })
			if tt.wantResult && err != nil {
				t.Fatalf("handleResponse failed: %v", err)
			}
			if !tt.wantResult && err == nil {
				t.Fatal("handleResponse succeeded, want error")
			}
			if tt.wantResult && result.Status != http.StatusNotModified {
				t.Fatalf("result status is %d, want %d", result.Status, http.StatusNotModified)
			}
			if !body.closed.Load() {
				t.Fatal("response body was not closed")
			}
			if !cancelled {
				t.Fatal("request context was not canceled")
			}
		})
	}
}

func TestCancelOnCloseClosesBodyAndCancels(t *testing.T) {
	body := &trackingBody{Reader: strings.NewReader("payload")}
	cancelled := false
	wrapped := &CancelOnClose{ReadCloser: body, Cancel: func() { cancelled = true }}

	if err := wrapped.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if !body.closed.Load() {
		t.Fatal("wrapped body was not closed")
	}
	if !cancelled {
		t.Fatal("request context was not canceled")
	}
}
