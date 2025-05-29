package algo

import (
	"bytes"
	"compress/zlib"
	"io"
)

type ZlibCompressor struct {
	level int
}

func NewZlib(level int) *ZlibCompressor {
	return &ZlibCompressor{level: level}
}

func (c *ZlibCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := zlib.NewWriterLevel(&buf, c.level)
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

func (c *ZlibCompressor) Decompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (c *ZlibCompressor) Extension() string {
	return ".zlib"
}
