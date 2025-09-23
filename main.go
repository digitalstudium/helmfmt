package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Version can be set at build time with -ldflags "-X main.Version=v1.2.3"
var Version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	// Handle version subcommand early
	if len(args) > 0 && args[0] == "version" {
		fmt.Println(Version)
		return 0
	}

	fs := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ContinueOnError)
	fs.Usage = func() {
		prog := fs.Name()
		fmt.Fprintf(fs.Output(), "Usage: %s [--stdout] --files <file1> <file2> ...\n", prog)
		fmt.Fprintf(fs.Output(), "   OR: %s <chart-path>\n", prog)
		fmt.Fprintf(fs.Output(), "   OR: %s version\n", prog)
		fmt.Fprintf(fs.Output(), "\nFlags:\n")
		fs.PrintDefaults()
	}

	filesMode := fs.Bool("files", false, "Process specific files (remaining args are file paths)")
	stdout := fs.Bool("stdout", false, "Output to stdout instead of modifying files")
	version := fs.Bool("version", false, "Show version and exit")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	if *version {
		fmt.Println(Version)
		return 0
	}

	remaining := fs.Args()

	if *filesMode {
		if len(remaining) == 0 {
			fmt.Fprintln(os.Stderr, "Error: --files requires at least one file argument")
			return 2
		}
		return process(remaining, *stdout)
	}

	// Chart directory mode
	if len(remaining) != 1 {
		fs.Usage()
		return 2
	}

	root := filepath.Join(remaining[0], "templates")
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
