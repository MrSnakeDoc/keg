package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/MrSnakeDoc/keg/internal/globalconfig"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type DefaultHTTPClient struct{ *http.Client }

type CancelOnClose struct {
	io.ReadCloser
	Cancel func()
}

func NewHTTPClient(timeout time.Duration) *DefaultHTTPClient {
	return &DefaultHTTPClient{Client: &http.Client{Timeout: timeout}}
}

func DownloadToFile(ctx context.Context, c HTTPClient, url, dst string, maxSize int64) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close failed: %w", cerr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close failed: %w", cerr)
		}
	}()

	var src io.Reader = resp.Body
	if maxSize > 0 {
		src = io.LimitReader(resp.Body, maxSize)
	}
	_, err = io.Copy(f, src)
	if err != nil {
		return fmt.Errorf("copy to file: %w", err)
	}

	return err
}

// ----------------------
// Advanced client (from kegdex, adapted)
// ----------------------

type AdvancedFetcher interface {
	FetchWithETag(ctx context.Context, url, prevETag string, maxBytes int64) (FetchResult, error)
}

type AdvancedHTTPClient struct {
	hc *http.Client
	ua string
}

func NewAdvancedHTTPClient(ua string) *AdvancedHTTPClient {
	dialer := &net.Dialer{
		Timeout:   globalconfig.DialTimeout,
		KeepAlive: 30 * time.Second,
	}

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   globalconfig.TLSHandshakeTimeout,
		ResponseHeaderTimeout: globalconfig.ResponseHeaderTimeout,
		DisableCompression:    true, // keep gzip raw
	}

	return &AdvancedHTTPClient{
		hc: &http.Client{Transport: tr},
		ua: normalizeUA(ua),
	}
}

func normalizeUA(ua string) string {
	u := strings.TrimSpace(ua)
	if u == "" {
		u = "keg/dev"
	}
	return u + " (go/" + runtime.Version() + ")"
}

type FetchResult struct {
	Body   io.ReadCloser // possibly gzip
	ETag   string
	Status int
	Length int64
}

func (c *AdvancedHTTPClient) FetchWithETag(ctx context.Context, url, prevETag string, maxBytes int64) (FetchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, globalconfig.RequestDeadline)

	req, err := c.prepareRequest(ctx, url, prevETag)
	if err != nil {
		cancel()
		return FetchResult{}, fmt.Errorf("prepare request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		cancel()
		return FetchResult{}, classifyNetErr(err)
	}

	return handleResponse(resp, maxBytes, cancel)
}

func (c *AdvancedHTTPClient) prepareRequest(ctx context.Context, url, prevETag string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.ua)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	if prevETag != "" {
		req.Header.Set("If-None-Match", prevETag)
	}
	return req, nil
}

func handleResponse(resp *http.Response, maxBytes int64, cancel func()) (f FetchResult, err error) {
	if resp == nil {
		cancel()
		return FetchResult{}, fmt.Errorf("nil response")
	}

	switch resp.StatusCode {
	case http.StatusNotModified:
		cancel()
		return FetchResult{Status: http.StatusNotModified, ETag: headerETag(resp)}, nil

	case http.StatusOK:

	case http.StatusTooManyRequests:
		cancel()
		return FetchResult{}, fmt.Errorf("429 too many requests")

	default:
		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			cancel()
			return FetchResult{}, fmt.Errorf("server error %d", resp.StatusCode)
		}
		cancel()
		return FetchResult{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if !isJSON(resp.Header.Get("Content-Type")) {
		cancel()
		return FetchResult{}, fmt.Errorf("unexpected content-type %q", resp.Header.Get("Content-Type"))
	}

	cl := headerContentLength(resp)
	if maxBytes > 0 && cl > 0 && cl > maxBytes {
		cancel()
		return FetchResult{}, fmt.Errorf("content-length %d exceeds limit %d bytes", cl, maxBytes)
	}

	var rc io.ReadCloser
	if maxBytes > 0 {
		rc = &struct {
			io.Reader
			io.Closer
		}{Reader: io.LimitReader(resp.Body, maxBytes), Closer: resp.Body}
	} else {
		rc = resp.Body
	}
	rc = &CancelOnClose{ReadCloser: rc, Cancel: cancel}

	return FetchResult{
		Status: http.StatusOK,
		ETag:   headerETag(resp),
		Body:   rc,
		Length: clOrMinusOne(cl),
	}, nil
}

// helpers

func isJSON(ct string) bool {
	if ct == "" {
		return true
	}
	ct = strings.ToLower(ct)
	return strings.HasPrefix(ct, "application/json")
}

func headerETag(resp *http.Response) string { return resp.Header.Get("ETag") }

func headerContentLength(resp *http.Response) int64 {
	v := resp.Header.Get("Content-Length")
	if v == "" {
		return -1
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return -1
	}
	return n
}

func classifyNetErr(err error) error {
	var nerr net.Error
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) ||
		(errors.As(err, &nerr) && nerr.Timeout()) {
		return fmt.Errorf("request timeout: %w", err)
	}
	return err
}

func clOrMinusOne(n int64) int64 {
	if n <= 0 {
		return -1
	}
	return n
}
