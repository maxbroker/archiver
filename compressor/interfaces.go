package compressor

// Compressor - интерфейс для алгоритмов сжатия
type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
	Extension() string
}
