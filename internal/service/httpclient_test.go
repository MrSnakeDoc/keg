package service

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

type trackingBody struct {
	io.Reader
	closed atomic.Bool
}

type staticHTTPClient struct {
	response *http.Response
}

func (c staticHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	return c.response, nil
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

func TestDownloadToFileAcceptsResponsesWithinLimit(t *testing.T) {
	for _, test := range []struct {
		name  string
		body  string
		limit int64
	}{
		{name: "below limit", body: "hey", limit: 5},
		{name: "exact limit", body: "hello", limit: 5},
	} {
		t.Run(test.name+"-"+strconv.FormatInt(test.limit, 10), func(t *testing.T) {
			dst := filepath.Join(t.TempDir(), "download")
			client := staticHTTPClient{response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(test.body))),
			}}

			if err := DownloadToFile(t.Context(), client, "https://example.com/file", dst, test.limit); err != nil {
				t.Fatalf("DownloadToFile failed: %v", err)
			}
			data, err := os.ReadFile(dst)
			if err != nil {
				t.Fatalf("read downloaded file: %v", err)
			}
			if string(data) != test.body {
				t.Fatalf("downloaded data is %q, want %q", data, test.body)
			}
		})
	}
}

func TestDownloadToFileRejectsOversizedResponses(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "download")
	client := staticHTTPClient{response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte("hello!"))),
	}}

	err := DownloadToFile(t.Context(), client, "https://example.com/file", dst, 5)
	if err == nil {
		t.Fatal("DownloadToFile succeeded, want size-limit error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("error is %q, want size-limit error", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("oversized partial file still exists, stat error: %v", err)
	}
}
