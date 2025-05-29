package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"archiver/compressor"
	"archiver/compressor/algo"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.NewWithID("com.archiver.app")
	w := a.NewWindow("Компрессор файлов")

	input := widget.NewMultiLineEntry()
	input.SetPlaceHolder("Выберите файлы и директории")
	input.Disable()

	algoName := widget.NewSelect([]string{"auto", "gzip", "brotli", "lz4", "zlib"}, nil)
	algoName.SetSelected("auto")

	// Создаем список доступных потоков
	var threadOptions []string
	for i := 1; i <= runtime.NumCPU(); i++ {
		threadOptions = append(threadOptions, fmt.Sprintf("%d", i))
	}
	concurrency := widget.NewSelect(threadOptions, nil)
	concurrency.SetSelected(fmt.Sprintf("%d", runtime.NumCPU()))

	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	compressors := map[string]compressor.Compressor{
		"gzip":   algo.NewGzip(6),
		"brotli": algo.NewBrotli(6),
		"lz4":    algo.NewLZ4(),
		"zlib":   algo.NewZlib(6),
	}

	selectFilesButton := widget.NewButton("Выбрать файлы", func() {
		dialog.ShowFileOpen(func(c fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if c == nil { // Cancel
				return
			}
			defer c.Close()
			input.SetText(c.URI().Path())
		}, w)
	})

	selectFolderButton := widget.NewButton("Выбрать директорию", func() {
		dialog.ShowFolderOpen(func(c fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if c == nil { // Cancel
				return
			}
			input.SetText(c.Path())
		}, w)
	})

	processFiles := func(isCompress bool) {
		inputPath := input.Text
		if inputPath == "" {
			dialog.ShowError(fmt.Errorf("необходимо выбрать файлы или директорию"), w)
			return
		}

		workers, err := strconv.Atoi(concurrency.Selected)
		if err != nil {
			dialog.ShowError(fmt.Errorf("некорректное количество потоков"), w)
			return
		}

		files, err := getFiles(inputPath)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		progressBar.Show()
		progressBar.Max = float64(len(files))
		progressBar.SetValue(0)

		selectedAlgo := algoName.Selected
		processor := compressor.NewProcessor(compressors, selectedAlgo, workers, false)

		var wg sync.WaitGroup
		sem := make(chan struct{}, workers)

		// Создаем канал для сбора результатов
		results := make(chan struct {
			file   string
			result *compressor.CompressionResult
		}, len(files))

		for i, file := range files {
			wg.Add(1)
			sem <- struct{}{}

			go func(f string, idx int) {
				defer wg.Done()
				defer func() { <-sem }()

				var result *compressor.CompressionResult
				if isCompress {
					result = processor.ProcessFile(f, "")
				} else {
					result = processor.DecompressFile(f)
				}

				if result.Error != nil {
					log.Printf("Ошибка при обработке %s: %v", f, result.Error)
				}

				results <- struct {
					file   string
					result *compressor.CompressionResult
				}{f, result}

				progressBar.SetValue(float64(idx + 1))
			}(file, i)
		}

		go func() {
			wg.Wait()
			close(results)
			progressBar.Hide()

			// Собираем результаты
			var successCount int
			var details strings.Builder
			for r := range results {
				if r.result.Error == nil && r.result.OriginalSize > 0 {
					successCount++
					details.WriteString(fmt.Sprintf("%s: %.1f%% (%.1f MB -> %.1f MB)\n",
						filepath.Base(r.file),
						r.result.CompressionRatio,
						float64(r.result.OriginalSize)/1024/1024,
						float64(r.result.CompressedSize)/1024/1024))
				} else if r.result.Error != nil {
					details.WriteString(fmt.Sprintf("%s: ошибка - %v\n",
						filepath.Base(r.file),
						r.result.Error))
				}
			}

			action := "Сжатие"
			if !isCompress {
				action = "Восстановление"
			}

			var message string
			if successCount == 0 {
				message = "Ошибка: не найдено файлов для обработки"
			} else {
				message = fmt.Sprintf("Успешно обработано %d файлов\n\nДетали:\n%s",
					successCount, details.String())
			}

			dialog.ShowInformation(fmt.Sprintf("%s завершено!", action), message, w)
		}()
	}

	compressButton := widget.NewButton("Сжать", func() {
		processFiles(true)
	})

	decompressButton := widget.NewButton("Восстановить", func() {
		processFiles(false)
	})

	w.SetContent(container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Файлы и директории", input),
			widget.NewFormItem("Алгоритм сжатия", algoName),
			widget.NewFormItem("Количество потоков", concurrency),
		),
		selectFilesButton,
		selectFolderButton,
		progressBar,
		container.NewHBox(compressButton, decompressButton),
	))

	w.Resize(fyne.NewSize(600, 500))
	w.ShowAndRun()
}

func getFiles(input string) ([]string, error) {
	var files []string
	paths := strings.Split(input, ",")
	for _, path := range paths {
		path = strings.TrimSpace(path)
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, filePath)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}
