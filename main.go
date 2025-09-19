package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s <chart-path> OR %s --files <file1> <file2> ...\n",
			filepath.Base(os.Args[0]), filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	// Check if running in pre-commit mode (processing individual files)
	if args[0] == "--files" {
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: --files requires at least one file argument\n")
			os.Exit(2)
		}
		processFiles(args[1:])
		return
	}

	// Original chart directory mode
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <chart-path>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	processChartDirectory(args[0])
}

func processFiles(files []string) {
	var total, updated, failed int

	for _, file := range files {
		// Only process YAML and template files
		if !hasWantedExt(file) {
			continue
		}

		// Skip if file doesn't exist or is not in a templates directory
		if !isHelmTemplate(file) {
			continue
		}

		total++

		// Format the file and check if it changed
		changed, err := formatFileInPlace(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR]  %s: %v\n", file, err)
			failed++
			continue
		}
		if changed {
			fmt.Printf("[UPDATED] %s\n", file)
			updated++
		}
	}

	fmt.Printf("\nProcessed: %d files, Updated: %d, Errors: %d\n", total, updated, failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func processChartDirectory(chartPath string) {
	root := filepath.Join(chartPath, "templates")
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

	if failed > 0 {
		os.Exit(1)
	}
}

// Check if file is likely a Helm template (in a templates directory)
func isHelmTemplate(filePath string) bool {
	dir := filepath.Dir(filePath)
	return strings.Contains(dir, "templates") ||
		strings.HasSuffix(dir, "templates") ||
		filepath.Base(dir) == "templates"
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
