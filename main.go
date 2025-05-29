package main

// import (
// 	"flag"
// 	"fmt"
// 	"log"
// 	"runtime"
// 	"sync"

// 	"archiver/compressor"
// 	"archiver/compressor/algo"

// 	"github.com/schollz/progressbar/v3"
// )

// func main1() {
// 	input := flag.String("i", "", "Input files/directories (required)")
// 	output := flag.String("o", "", "Output directory (default: <input>_<algo>)")
// 	algoName := flag.String("a", "auto", "Algorithm (gzip, brotli, lz4, zlib, auto)")
// 	concurrency := flag.Int("c", runtime.NumCPU(), "Number of workers")
// 	verbose := flag.Bool("v", false, "Verbose output")
// 	showProgress := flag.Bool("progress", false, "Show progress bar")
// 	flag.Parse()

// 	if *input == "" {
// 		log.Fatal("Input parameter (-i) is required")
// 	}

// 	compressors := map[string]compressor.Compressor{
// 		"gzip":   algo.NewGzip(6),
// 		"brotli": algo.NewBrotli(6),
// 		"lz4":    algo.NewLZ4(),
// 		"zlib":   algo.NewZlib(6),
// 	}

// 	files, err := getFiles(*input)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	var bar *progressbar.ProgressBar
// 	if *showProgress {
// 		bar = progressbar.Default(int64(len(files)))
// 	}

// 	// Создаем обработчик
// 	processor := compressor.NewProcessor(compressors, *algoName, *concurrency, *verbose)

// 	// Параллельная обработка
// 	var wg sync.WaitGroup
// 	sem := make(chan struct{}, *concurrency)

// 	for _, file := range files {
// 		wg.Add(1)
// 		sem <- struct{}{}

// 		go func(f string) {
// 			defer wg.Done()
// 			defer func() { <-sem }()

// 			if err := processor.ProcessFile(f, *output); err != nil && *verbose {
// 				log.Printf("Error processing %s: %v", f, err)
// 			}

// 			if *showProgress {
// 				bar.Add(1)
// 			}
// 		}(file)
// 	}

// 	wg.Wait()
// 	fmt.Println("\nCompression completed!")
// }
