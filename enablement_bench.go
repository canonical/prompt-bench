package main

import (
	"fmt"
	"iter"
	"log/slog"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

var (
	measureForSnaps      = []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 20, 25, 30, 40, 50, 75, 100, 125, 150, 175, 200}
	snapPatternToInstall = "prompt-bench-%d_0.1_amd64.snap"
)

// runEnablementBench installs more and more snaps, and measure the time for the snap permission prompting enablement
// to proceed.
func runEnablementBench(snapToInstallDir string, count uint) error {
	// measurements is a mapping between a number of snaps and the elapsed time to enable the feature.
	measurements := make(map[uint64][]uint64)

	if err := disablePermissionPrompting(); err != nil {
		return fmt.Errorf("failed to initially disable permission prompting: %v", err)
	}

	for nSnaps, err := range nextSnapsBenchIteration(snapToInstallDir) {
		if err != nil {
			return fmt.Errorf("failed while setting up snaps for next iteration: %v", err)
		}

		for i := range count {
			slog.Info(fmt.Sprintf("Measuring enablement for %d snaps, iteration %d/%d", nSnaps, i+1, count))
			time, err := measureEnablement()
			if err != nil {
				return fmt.Errorf("failed to measure enablement for %d snaps: %v", nSnaps, err)
			}
			measurements[nSnaps] = append(measurements[nSnaps], time)
		}
	}

	slog.Info("Compute measurement statistics")
	printEnablementMeasurements(measurements)

	return nil
}

// nextSnapsBenchIteration returns an iterator that install the snaps until reaching the next
// breakpoint of number of snaps to measure.
func nextSnapsBenchIteration(snapToInstallDir string) iter.Seq2[uint64, error] {
	return func(yield func(uint64, error) bool) {
		// use snap list to determine how many snaps are already installed based on the number of lines -1
		cmd := exec.Command("snap", "list")
		out, err := cmd.CombinedOutput()
		if err != nil {
			yield(0, fmt.Errorf("snap list returned: %v\n%v", err, string(out)))
			return
		}
		currentInstallSnaps := uint64(len(strings.Split(string(out), "\n")) - 2) // -2 to remove the header and the last empty line

		for currentInstallSnaps <= measureForSnaps[len(measureForSnaps)-1] {
			if slices.Contains(measureForSnaps, currentInstallSnaps) {
				if !yield(currentInstallSnaps, nil) {
					return
				}
			}
			out, err := exec.Command("snap", "install", "--dangerous", filepath.Join(snapToInstallDir, fmt.Sprintf(snapPatternToInstall, currentInstallSnaps))).CombinedOutput()
			if err != nil {
				yield(0, fmt.Errorf("snap install returned: %v\n%v", err, string(out)))
				return
			}
			currentInstallSnaps++
		}
	}
}

// measureEnablement measures the time it takes to enable prompting.
func measureEnablement() (m uint64, err error) {
	cmd := exec.Command("snap", "set", "system", "experimental.apparmor-prompting=true")
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start).Nanoseconds()
	if err != nil {
		return 0, fmt.Errorf("failed to enable experimental.apparmor-prompting setting: %v\n%v", err, string(out))
	}

	// Seems snapd needs some time to process the setting change, so we wait a bit before measuring
	time.Sleep(5 * time.Second)

	// Cleanup
	defer func() {
		if err != nil {
			return
		}

		// Reset the setting to false after measuring
		err = disablePermissionPrompting()
	}()

	return uint64(elapsed), nil
}

func disablePermissionPrompting() error {
	defer func() {
		// Seems snapd needs some time to process the setting change, so we wait a bit before measuring
		time.Sleep(5 * time.Second)
	}()
	out, err := exec.Command("snap", "set", "system", "experimental.apparmor-prompting=false").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reset experimental.apparmor-prompting setting: %v\n%v", err, string(out))
	}

	return nil
}

// printEnablementMeasurements prints a CSV file with:
// number_snaps,average_time,max,min,std_dev
func printEnablementMeasurements(measurements map[uint64][]uint64) {
	// Create a slice to store the relative paths and sort them in ASCII order
	// If targetting a single file, keep the whole file path.
	numberOfSnaps := make([]uint64, 0, len(measurements))
	for n := range measurements {
		numberOfSnaps = append(numberOfSnaps, n)
	}
	slices.Sort(numberOfSnaps)

	fmt.Println("nsnaps,average_time,max,min,std_dev")

	for _, n := range numberOfSnaps {
		elapsedTimes := measurements[n]
		if len(elapsedTimes) == 0 {
			slog.Warn(fmt.Sprintf("no measurements for %d snaps installed", n))
			continue
		}

		avg, maxT, minT, dev := timeStats(elapsedTimes)
		fmt.Printf("%d,%d,%d,%d,%d\n", n, avg, maxT, minT, dev)
	}
}
