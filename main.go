package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Check command line arguments
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <chart-path>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	root := filepath.Join(os.Args[1], "templates")
	if _, err := os.Stat(root); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var total, updated, failed int

	// Walk through all files in the directory
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Walk error at %s: %v\n", path, err)
			failed++
			return nil
		}
		if d.IsDir() {
			return nil
		}

		// Only process YAML and template files
		if !hasWantedExt(path) {
			return nil
		}
		total++

		// Format the file and check if it changed
		changed, ferr := formatFileInPlace(path)
		if ferr != nil {
			fmt.Fprintf(os.Stderr, "[ERROR]  %s: %v\n", path, ferr)
			failed++
			return nil
		}
		if changed {
			fmt.Printf("[UPDATED] %s\n", path)
			updated++
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "WalkDir error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nProcessed: %d files, Updated: %d, Errors: %d\n", total, updated, failed)
}

// Check if file has the extensions we want to process
func hasWantedExt(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml", ".tpl":
		return true
	default:
		return false
	}
}

// Read file, format it, and write back if changed
func formatFileInPlace(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	orig := string(b)
	formatted := formatIndentation(orig)

	// Only write if content changed
	if formatted == orig {
		return false, nil
	}

	// Preserve original file permissions
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(path, []byte(formatted), info.Mode()); err != nil {
		return false, err
	}
	return true, nil
}
