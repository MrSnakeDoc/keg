package checker

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/MrSnakeDoc/keg/internal/config"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type trackingCheckerBody struct {
	io.Reader
	closed atomic.Bool
}

func (b *trackingCheckerBody) Close() error {
	b.closed.Store(true)
	return nil
}

type checkerHTTPClientFunc func(*http.Request) (*http.Response, error)

func (f checkerHTTPClientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestMakeHTTPRequestClosesNonOKBody(t *testing.T) {
	body := &trackingCheckerBody{Reader: strings.NewReader("failure")}
	client := checkerHTTPClientFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: body}, nil
	})

	_, err := MakeHTTPRequest(t.Context(), client, "https://example.com/releases/latest")
	if err == nil {
		t.Fatal("MakeHTTPRequest succeeded, want error")
	}
	if !body.closed.Load() {
		t.Fatal("non-success response body was not closed")
	}
}

func TestFetchChecksumClosesBody(t *testing.T) {
	release := &GitHubRelease{TagName: "v1.2.3"}
	body := &trackingCheckerBody{
		Reader: strings.NewReader("checksum  " + utils.AssetName("1.2.3") + "\n"),
	}
	client := checkerHTTPClientFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	})
	controller := &CheckerController{
		Config:     &config.Config{ChecksumBaseURL: "https://example.com/releases"},
		HTTPClient: client,
	}

	if _, err := controller.fetchChecksum(t.Context(), release); err != nil {
		t.Fatalf("fetchChecksum failed: %v", err)
	}
	if !body.closed.Load() {
		t.Fatal("checksum response body was not closed")
	}
}
