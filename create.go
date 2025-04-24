package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// setupFolder creates nFiles files in nDirectories subdirectories.
func setupFolder(root string, nFiles, nDirectories int) error {
	dirPath, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	if err := createFilesInDir(dirPath, nFiles); err != nil {
		return err
	}

	// Create subdirectories for files in each directory
	for i := range nDirectories {
		dirPath = filepath.Join(dirPath, fmt.Sprintf("subdir_%d", i))
		if err := createFilesInDir(dirPath, nFiles); err != nil {
			return err
		}
	}

	return nil
}

// createFilesInDir creates `total` empty files in p.
func createFilesInDir(p string, total int) error {
	for i := range total {
		f, err := os.OpenFile(filepath.Join(p, fmt.Sprintf("file_%d", i)), os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
