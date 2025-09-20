package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}

	stdout := false
	var files []string

	// Pre-commit/IDE mode: --files <file1> <file2> ... [--stdout]
	if args[0] == "--files" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: --files requires at least one file argument")
			return 2
		}
		for _, a := range args[1:] {
			if a == "--stdout" {
				stdout = true
				continue
			}
			files = append(files, a)
		}
		if len(files) == 0 {
			fmt.Fprintln(os.Stderr, "Error: --files requires at least one file argument")
			return 2
		}
		return process(files, stdout)
	}

	// Chart directory mode: <chart-path>
	if len(args) != 1 {
		usage()
		return 2
	}
	root := filepath.Join(args[0], "templates")
	if _, err := os.Stat(root); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	files, err := collectFiles(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WalkDir error: %v\n", err)
		return 1
	}
	return process(files, false)
}

func usage() {
	prog := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage: %s <chart-path> OR %s --files <file1> <file2> ... [--stdout]\n", prog, prog)
}

func collectFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Walk error at %s: %v\n", path, err)
			return nil
		}
		if d.IsDir() || !wanted(path) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	return out, err
}

func process(files []string, stdout bool) int {
	var total, updated, failed int

	for _, file := range files {
		total++

		b, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR]  %s: %v\n", file, err)
			failed++
			continue
		}
		orig := string(b)

		if err := validateTemplateSyntax(orig); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR]  Invalid syntax %s: %v\n", file, err)
			failed++
			continue
		}

		formatted := ensureTrailingNewline(formatIndentation(orig))

		if stdout {
			fmt.Print(formatted)
			continue
		}

		// In-place mode: don't write if the only change is a trailing newline
		if formatted == orig || formatted == orig+"\n" {
			continue
		}

		info, err := os.Stat(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR]  %s: %v\n", file, err)
			failed++
			continue
		}
		if err := os.WriteFile(file, []byte(formatted), info.Mode()); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR]  %s: %v\n", file, err)
			failed++
			continue
		}

		fmt.Printf("[UPDATED] %s\n", file)
		updated++
	}

	if !stdout {
		fmt.Printf("\nProcessed: %d files, Updated: %d, Errors: %d\n", total, updated, failed)
	}
	if failed > 0 {
		return 1
	}
	return 0
}

func wanted(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".tpl":
		return true
	}
	return false
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
