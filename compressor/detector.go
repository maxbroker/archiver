package compressor

import (
	"bytes"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// Detector определяет подходящий алгоритм сжатия для файла
type Detector struct {
	compressors map[string]Compressor
}

// NewDetector создает новый детектор
func NewDetector(compressors map[string]Compressor) *Detector {
	return &Detector{
		compressors: compressors,
	}
}

// DetectCompressor определяет подходящий компрессор для файла
func (d *Detector) DetectCompressor(filePath string) Compressor {
	// Сначала проверяем, не является ли файл уже сжатым
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, c := range d.compressors {
		if c.Extension() == ext {
			return nil // Файл уже сжат
		}
	}

	mimeType := mime.TypeByExtension(ext)

	file, err := os.Open(filePath)
	if err != nil {
		return d.compressors["gzip"] // В случае ошибки используем gzip
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return d.compressors["gzip"]
	}

	switch {
	case strings.HasPrefix(mimeType, "text/"):
		// Для текстовых файлов используем brotli
		return d.compressors["brotli"]

	case strings.HasPrefix(mimeType, "image/"):
		// Для изображений используем lz4, так как они обычно уже сжаты
		return d.compressors["lz4"]

	case strings.HasPrefix(mimeType, "application/json"):
		// Для JSON используем brotli
		return d.compressors["brotli"]

	case strings.HasPrefix(mimeType, "application/xml"):
		// Для XML используем brotli
		return d.compressors["brotli"]

	case bytes.Contains(buffer, []byte("PNG")) ||
		bytes.Contains(buffer, []byte("JFIF")) ||
		bytes.Contains(buffer, []byte("GIF")):
		// Для уже сжатых изображений используем lz4
		return d.compressors["lz4"]

	case bytes.Contains(buffer, []byte("PK\x03\x04")):
		// Для ZIP-файлов используем lz4
		return d.compressors["lz4"]

	default:
		// Для остальных файлов используем gzip
		return d.compressors["gzip"]
	}
}
