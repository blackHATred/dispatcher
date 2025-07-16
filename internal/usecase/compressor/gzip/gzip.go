package compressor

import (
	"bytes"
	"compress/gzip"
	"dispatcher/internal/usecase"
	"io"
)

type GzipCompressor struct{}

func NewGzipCompressor() usecase.PointCloudCompressor {
	return &GzipCompressor{}
}

func (c *GzipCompressor) Compress(data []byte) ([]byte, error) {
	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	_, err := zw.Write(data)
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (c *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
