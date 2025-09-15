package utils

import (
	"bufio"
	"compress/gzip"
	"io"
	"strings"
)

// readCloser ties a Reader to a Closer (composite).
type readCloser struct {
	io.Reader
	io.Closer
}

// maybeGunzip returns a reader that yields the decompressed stream if 'src' is gzip,
// else returns src as-is. It preserves the ability to Close().
func MaybeGunzip(src io.ReadCloser) (io.ReadCloser, error) {
	br := bufio.NewReader(src)
	// Peek 2 bytes for gzip magic
	hdr, err := br.Peek(2)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(hdr) >= 2 && hdr[0] == 0x1f && hdr[1] == 0x8b {
		// gzip
		gr, err := gzip.NewReader(br)
		if err != nil {
			return nil, err
		}
		return readCloser{Reader: gr, Closer: src}, nil
	}
	// not gzip
	return readCloser{Reader: br, Closer: src}, nil
}

func GzipBytes(src []byte) ([]byte, error) {
	var b strings.Builder
	zw := gzip.NewWriter(&b)

	if _, err := zw.Write(src); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}
