package algo

import (
	"bytes"
	"io"

	"github.com/andybalholm/brotli"
)

type BrotliCompressor struct {
	level int
}

func NewBrotli(level int) *BrotliCompressor {
	return &BrotliCompressor{level: level}
}

func (c *BrotliCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := brotli.NewWriterLevel(&buf, c.level)
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *BrotliCompressor) Decompress(data []byte) ([]byte, error) {
	reader := brotli.NewReader(bytes.NewReader(data))
	return io.ReadAll(reader)
}

func (c *BrotliCompressor) Extension() string {
	return ".br"
}
