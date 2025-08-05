package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// runBench opens all files within p after collecting them. If p is a file, only that one will be opened.
// count is the number of runs to proceed.
// within a run, files can be opened in parallel or not.
func runBench(root string, count uint, parallel bool) error {
	// measurements is a mapping between a file path and the elapsed time to open this file.
	// We are going to pre-populate this and only measure one syscall per file concurrently, so we can
	// avoid using a sync.Map for this.
	measurements := make(map[string][]uint64)

	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	// First, ensure we list everything we want to measure. This is outside of the measurement itself as it can trigger
	// prompting for no rules stored
	slog.Info(fmt.Sprintf("Prescanning: %s", root))
	rootIsDir, err := discoverContent(root, measurements)
	if err != nil {
		return fmt.Errorf("error while prescanning: %v", err)
	}

	slog.Info(fmt.Sprintf("Starting measuring %d time(s)", count))
	if parallel {
		slog.Info("Runs opening calls are in parallel")
	}
	for range count {
		openAllFiles(parallel, measurements)
	}

	slog.Info("Compute measurement statistics")
	// Relative path for file targetting.
	if !rootIsDir {
		root = filepath.Dir(root)
	}
	printFileMeasurements(measurements, root)

	return nil
}

// discoverContent populate a global map with an index of every files in folder
func discoverContent(root string, measurements map[string][]uint64) (rootIsDir bool, err error) {
	err = filepath.Walk(root, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if p == root {
				rootIsDir = true
			}
			return nil
		}

		measurements[p] = nil

		// filepath.Walk() only trigger an open on the directory to list its content.
		// Ensure in case of "always allow" registration, we also went over each files too
		// to creat the snap rules.
		fd, err := syscall.Open(p, syscall.O_RDONLY, 0)
		if err != nil {
			return err
		}
		defer syscall.Close(fd)

		return nil
	})

	return rootIsDir, err
}

// openAllFiles open all the files from the measurements map
func openAllFiles(parallel bool, measurements map[string][]uint64) {
	var wg sync.WaitGroup
	for p := range maps.Clone(measurements) {
		if parallel {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := measureFileOpening(p, measurements); err != nil {
					slog.Error(fmt.Sprintf("can't measuring file: %v", err))
				}
			}()
			continue
		}

		if err := measureFileOpening(p, measurements); err != nil {
			slog.Error(fmt.Sprintf("can't measuring file: %v", err))
		}
	}
	wg.Wait()
}

var mu sync.Mutex

// measureFileOpening measures the time the syscall takes to open a single file.
func measureFileOpening(p string, measurements map[string][]uint64) error {
	start := time.Now()
	fd, err := syscall.Open(p, syscall.O_RDONLY, 0)
	elapsed := time.Since(start).Nanoseconds()
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	mu.Lock()
	measurements[p] = append(measurements[p], uint64(elapsed))
	mu.Unlock()

	return nil
}

// printFileMeasurements prints a CSV file with:
// filename,distance,average_time,max,min,std_dev
func printFileMeasurements(measurements map[string][]uint64, root string) {
	// Create a slice to store the relative paths and sort them in ASCII order
	// If targetting a single file, keep the whole file path.
	relPaths := make([]string, 0, len(measurements))
	rootToStrip := root + "/"
	for p := range measurements {
		relPaths = append(relPaths, strings.TrimPrefix(p, rootToStrip))
	}
	sort.Strings(relPaths)

	fmt.Println("filename,distance,average_time,max,min,std_dev")

	for i, relPath := range relPaths {
		elapsedTimes := measurements[filepath.Join(root, relPath)]
		if len(elapsedTimes) == 0 {
			slog.Warn(fmt.Sprintf("no measurements for %s", relPath))
			continue
		}

		avg, maxT, minT, dev := timeStats(elapsedTimes)
		fmt.Printf("%s,%d,%d,%d,%d,%d\n", relPath, i+1, avg, maxT, minT, dev)
	}
}
