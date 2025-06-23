// service/http.go
package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/MrSnakeDoc/keg/internal/utils"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type DefaultHTTPClient struct{ *http.Client }

func NewHTTPClient(timeout time.Duration) *DefaultHTTPClient {
	return &DefaultHTTPClient{Client: &http.Client{Timeout: timeout}}
}

func DownloadToFile(ctx context.Context, c HTTPClient, url, dst string, maxSize int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer utils.Try(resp.Body.Close)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer utils.Close(f)

	var src io.Reader = resp.Body
	if maxSize > 0 {
		src = io.LimitReader(resp.Body, maxSize)
	}
	_, err = io.Copy(f, src)
	return err
}
