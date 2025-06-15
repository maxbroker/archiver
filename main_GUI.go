package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

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
	input.TextStyle.Bold = true
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
			input.SetText(c.URI().String())
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

		const (
			batchSize  = 20  // Количество файлов в одном обновлении
			maxResults = 500 // Максимальное количество хранимых результатов
		)
		var (
			batchBuffer    []string
			batchMutex     sync.Mutex
			processedFiles []string
		)

		// Создаем окно результатов заранее
		resultList := widget.NewList(
			func() int { return len(processedFiles) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				o.(*widget.Label).SetText(processedFiles[i])
			},
		)
		scrollContainer := container.NewScroll(resultList)

		action := "Сжатие"
		if !isCompress {
			action = "Восстановление"
		}

		progressBar.Show()
		progressBar.Refresh()
		progressContainer := container.NewVBox( // Явно создаем контейнер для прогресс-бара
			widget.NewLabel(fmt.Sprintf("Обработка %d файлов...", len(files))),
			progressBar,
		)
		progressContainer.Refresh() // Важно: обновляем контейнер

		progressDialog := dialog.NewCustom(
			fmt.Sprintf("%s в процессе...", action),
			"Закрыть",
			progressContainer,
			w,
		)
		progressDialog.Show()
		progressBar.Max = float64(len(files))
		progressBar.SetValue(0)
		progressDialog.Show()

		selectedAlgo := algoName.Selected
		processor := compressor.NewProcessor(compressors, selectedAlgo, workers, false)

		var wg sync.WaitGroup
		sem := make(chan struct{}, workers)

		// Канал для обновления UI в реальном времени
		updateUI := make(chan string, 100)

		// Горутина для обновления UI
		go func() {
			go func() {
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						batchMutex.Lock()
						if len(batchBuffer) > 0 {
							updateUI <- strings.Join(batchBuffer, "")
							batchBuffer = batchBuffer[:0]
						}
						batchMutex.Unlock()
					case msg, ok := <-updateUI:
						if !ok {
							return
						}
						batchMutex.Lock()
						batchBuffer = append(batchBuffer, msg)
						if len(batchBuffer) >= batchSize {
							updateUI <- strings.Join(batchBuffer, "")
							batchBuffer = batchBuffer[:0]
						}
						batchMutex.Unlock()
					}
				}
			}()

			for update := range updateUI {
				processedFiles = append(processedFiles, update)

				// Ограничиваем количество хранимых результатов
				if len(processedFiles) > maxResults {
					processedFiles = processedFiles[len(processedFiles)-maxResults:]
				}

				resultList.Refresh()
				scrollContainer.ScrollToBottom()

				// Даем время на обработку событий UI
				if len(processedFiles)%50 == 0 {
					time.Sleep(10 * time.Millisecond)
				}
			}

			progressDialog.Hide()
			if isCompress {
				// Для сжатия показываем детализированные результаты
				resultDialog := dialog.NewCustom(
					fmt.Sprintf("%s завершено!", action),
					"Закрыть",
					scrollContainer,
					w,
				)
				resultDialog.Resize(fyne.NewSize(400, 400))
				resultDialog.Show()
			} else {
				// Для восстановления просто показываем уведомление
				dialog.ShowInformation(
					"Восстановление завершено",
					"Файлы успешно восстановлены",
					w,
				)
			}
		}()

		for i, file := range files {
			wg.Add(1)
			sem <- struct{}{}

			go func(f string, idx int) {
				defer wg.Done()
				defer func() { <-sem }()

				if idx%10 == 0 {
					time.Sleep(5 * time.Millisecond)
				}

				var result *compressor.CompressionResult
				if isCompress {
					result = processor.ProcessFile(f, "")
				} else {
					result = processor.DecompressFile(f)
				}

				a.SendNotification(fyne.NewNotification("", ""))
				progressBar.SetValue(float64(idx + 1))
				progressBar.Refresh()

				if isCompress {
					var msg string
					if result.Error != nil {
						msg = fmt.Sprintf("%s: ошибка - %v\n", filepath.Base(f), result.Error)
					} else {
						originalSizeMB := float64(result.OriginalSize) / 1024 / 1024
						compressedSizeMB := float64(result.CompressedSize) / 1024 / 1024

						if originalSizeMB >= 0.05 || compressedSizeMB >= 0.05 {
							msg = fmt.Sprintf("%s: %.1f%% (%.1f MB -> %.1f MB)\n",
								filepath.Base(f),
								result.CompressionRatio,
								originalSizeMB,
								compressedSizeMB)
						}
					}

					if msg != "" {
						updateUI <- msg
					}
				}
			}(file, i)
		}

		go func() {
			wg.Wait()
			close(updateUI)
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
