package web

import (
	"bytes"
	"compress/gzip"
)

// CompressGzip compresses a byte slice using standard gzip (best compression).
func CompressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	if _, err := zw.Write(data); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
