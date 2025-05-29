package algo

import (
	"bytes"
	"compress/gzip"
	"io"
)

type GzipCompressor struct {
	level int
}

func NewGzip(level int) *GzipCompressor {
	return &GzipCompressor{level: level}
}

func (c *GzipCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, c.level)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (c *GzipCompressor) Extension() string {
	return ".gz"
}
