package algo

import (
	"github.com/pierrec/lz4/v4"
)

type LZ4Compressor struct{}

func NewLZ4() *LZ4Compressor {
	return &LZ4Compressor{}
}

func (c *LZ4Compressor) Compress(data []byte) ([]byte, error) {
	buf := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := lz4.CompressBlock(data, buf, nil)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (c *LZ4Compressor) Decompress(data []byte) ([]byte, error) {
	buf := make([]byte, len(data)*4) // Предполагаем, что разжатые данные не более чем в 4 раза больше
	n, err := lz4.UncompressBlock(data, buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (c *LZ4Compressor) Extension() string {
	return ".lz4"
}
