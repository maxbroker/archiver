package compressor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CompressionResult содержит информацию о результате сжатия/распаковки
type CompressionResult struct {
	OriginalSize     int64
	CompressedSize   int64
	CompressionRatio float64
	Error            error
}

// Processor обрабатывает файлы с использованием выбранного алгоритма сжатия
type Processor struct {
	compressors  map[string]Compressor
	detector     *Detector
	workers      int
	verbose      bool
	selectedAlgo string
}

// NewProcessor создает новый процессор
func NewProcessor(compressors map[string]Compressor, algo string, workers int, verbose bool) *Processor {
	return &Processor{
		compressors:  compressors,
		detector:     NewDetector(compressors),
		workers:      workers,
		verbose:      verbose,
		selectedAlgo: algo,
	}
}

// IsCompressed проверяет, является ли файл уже сжатым
func (p *Processor) IsCompressed(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, c := range p.compressors {
		if c.Extension() == ext {
			return true
		}
	}
	return false
}

// ProcessFile сжимает файл
func (p *Processor) ProcessFile(inputPath, outputPath string) *CompressionResult {
	result := &CompressionResult{}

	if p.IsCompressed(inputPath) {
		result.Error = fmt.Errorf("файл %s уже сжат", inputPath)
		return result
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		result.Error = fmt.Errorf("ошибка чтения файла: %v", err)
		return result
	}
	result.OriginalSize = int64(len(data))

	var compressor Compressor
	if p.selectedAlgo == "auto" {
		compressor = p.detector.DetectCompressor(inputPath)
		if compressor == nil {
			compressor = p.compressors["gzip"]
		}
	} else {
		compressor = p.compressors[p.selectedAlgo]
		if compressor == nil {
			result.Error = fmt.Errorf("неизвестный алгоритм сжатия: %s", p.selectedAlgo)
			return result
		}
	}

	compressed, err := compressor.Compress(data)
	if err != nil {
		result.Error = fmt.Errorf("ошибка сжатия: %v", err)
		return result
	}
	result.CompressedSize = int64(len(compressed))
	result.CompressionRatio = float64(result.CompressedSize) / float64(result.OriginalSize) * 100

	if outputPath == "" {
		ext := compressor.Extension()
		outputPath = p.getOutputPath(inputPath, "", strings.TrimPrefix(ext, "."))

		outputDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			result.Error = fmt.Errorf("ошибка создания директории: %v", err)
			return result
		}
	}

	if err := os.WriteFile(outputPath, compressed, 0644); err != nil {
		result.Error = fmt.Errorf("ошибка записи файла: %v", err)
		return result
	}

	return result
}

// DecompressFile разжимает файл
func (p *Processor) DecompressFile(inputPath string) *CompressionResult {
	result := &CompressionResult{}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		result.Error = fmt.Errorf("ошибка чтения файла: %v", err)
		return result
	}
	result.CompressedSize = int64(len(data))

	ext := filepath.Ext(inputPath)
	var compressor Compressor
	for _, c := range p.compressors {
		if c.Extension() == ext {
			compressor = c
			break
		}
	}

	if compressor == nil {
		result.Error = fmt.Errorf("неизвестный формат сжатия для файла: %s", inputPath)
		return result
	}

	decompressed, err := compressor.Decompress(data)
	if err != nil {
		result.Error = fmt.Errorf("ошибка разжатия: %v", err)
		return result
	}
	result.OriginalSize = int64(len(decompressed))
	result.CompressedSize = int64(len(data))
	result.CompressionRatio = float64(result.CompressedSize) / float64(result.OriginalSize) * 100

	outputPath := strings.TrimSuffix(inputPath, ext)

	if err := os.WriteFile(outputPath, decompressed, 0644); err != nil {
		result.Error = fmt.Errorf("ошибка записи файла: %v", err)
		return result
	}

	return result
}

func (p *Processor) getOutputPath(inputPath, outputDir, ext string) string {
	if outputDir == "" {
		dir := filepath.Dir(inputPath)
		base := filepath.Base(inputPath)
		newDir := strings.TrimSuffix(dir, "/") + "_" + ext
		return filepath.Join(newDir, base+"."+ext)
	}
	relPath, _ := filepath.Rel(filepath.Dir(inputPath), inputPath)
	return filepath.Join(outputDir, relPath+"."+ext)
}
