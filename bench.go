package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// runBench opens all files within p after collecting them. If p is a file, only that one will be opened.
// count is the number of runs to proceed.
// within a run, files can be opened in parallel or not.
func runBench(p string, count uint, parallel bool) error {
	// measurements is a mapping between a file path and the elapsed time to open this file.
	// We are going to pre-populate this and only measure one syscall per file concurrently, so we can
	// avoid using a sync.Map for this.
	measurements := make(map[string][]uint64)

	// First, ensure we list everything we want to measure. This is outside of the measurement itself as it can trigger
	// prompting for no rules stored
	slog.Info(fmt.Sprintf("Prescanning: %s", p))
	if err := discoverContent(p, measurements); err != nil {
		return fmt.Errorf("error while prescanning: %v", err)
	}

	slog.Info(fmt.Sprintf("Starting measuring %d time(s)", count))
	if parallel {
		slog.Info("Runs opening calls are in parallel")
	}
	for range count {
		openAllFiles(parallel, measurements)
	}

	return nil
}

// discoverContent populate a global map with an index of every files in folder
func discoverContent(root string, measurements map[string][]uint64) error {
	return filepath.Walk(root, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		measurements[p] = nil

		return nil
	})
}

// openAllFiles open all the files from the measurements map
func openAllFiles(parallel bool, measurements map[string][]uint64) {
	var wg sync.WaitGroup
	for p := range measurements {
		if parallel {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := measureFileOpening(p, measurements); err != nil {
					slog.Error(fmt.Sprintf("can't measuring file: %v", err))
				}
			}()
			break
		}
		if err := measureFileOpening(p, measurements); err != nil {
			slog.Error(fmt.Sprintf("can't measuring file: %v", err))
		}
	}
	wg.Wait()
}

// measureFileOpening measures the time the syscall takes to open a single file.
func measureFileOpening(p string, measurements map[string][]uint64) error {
	start := time.Now()
	f, err := os.Open(p)
	elapsed := time.Since(start).Nanoseconds()
	if err != nil {
		return err
	}
	defer f.Close()

	measurements[p] = append(measurements[p], uint64(elapsed))

	return nil
}
