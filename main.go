// Package main implements helmfmt.
//
//go:generate go run gen_stubs.go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Version can be set at build time with -ldflags "-X main.Version=v1.2.3"
var Version = "dev"

type Config struct {
	IndentSize int         `json:"indent_size"`
	Extensions []string    `json:"extensions"`
	Rules      RulesConfig `json:"rules"`
}

type RulesConfig struct {
	Indent map[string]RuleConfig `json:"indent"`
}

type RuleConfig struct {
	Disabled bool     `json:"disabled"`
	Exclude  []string `json:"exclude"`
}

func loadConfig() *Config {
	// Default config
	config := &Config{
		IndentSize: 2,
		Extensions: []string{".yaml", ".yml", ".tpl"},
		Rules: RulesConfig{
			Indent: map[string]RuleConfig{
				"tpl":      {Disabled: true, Exclude: []string{}},
				"toYaml":   {Disabled: true, Exclude: []string{}},
				"template": {Disabled: true, Exclude: []string{}},
				"include":  {Disabled: true, Exclude: []string{}},
				"printf":   {Disabled: false, Exclude: []string{}},
				"fail":     {Disabled: false, Exclude: []string{}},
			},
		},
	}

	// Try to load from home directory first
	if homeDir, err := os.UserHomeDir(); err == nil {
		homeConfigPath := filepath.Join(homeDir, ".helmfmt")
		loadConfigFile(homeConfigPath, config)
	}

	// Try to load from current directory (overrides home config)
	loadConfigFile(".helmfmt", config)

	return config
}

func loadConfigFile(path string, config *Config) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // File doesn't exist, skip silently
	}

	if err := json.Unmarshal(data, config); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error parsing config from %s: %v\n", path, err)
	}
}

func main() {
	os.Exit(run())
}

func run() int {
	config := loadConfig()
	var stdout, files, check bool
	var disableRules, enableRules []string

	var rootCmd = &cobra.Command{
		Use:     "helmfmt [flags] [chart-path | file1 file2 ...]",
		Short:   "Format Helm templates",
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if check && stdout {
				return fmt.Errorf("--check and --stdout are mutually exclusive")
			}

			// Apply rule overrides from flags
			for _, rule := range disableRules {
				if _, exists := config.Rules.Indent[rule]; exists {
					ruleConfig := config.Rules.Indent[rule]
					ruleConfig.Disabled = true
					config.Rules.Indent[rule] = ruleConfig
				} else {
					return fmt.Errorf("unknown rule: %s", rule)
				}
			}

			for _, rule := range enableRules {
				if _, exists := config.Rules.Indent[rule]; exists {
					ruleConfig := config.Rules.Indent[rule]
					ruleConfig.Disabled = false
					config.Rules.Indent[rule] = ruleConfig
				} else {
					return fmt.Errorf("unknown rule: %s", rule)
				}
			}

			// Check if stdin is being piped
			stat, _ := os.Stdin.Stat()
			stdinPiped := (stat.Mode() & os.ModeCharDevice) == 0

			// If --files flag is used, process the provided files
			if files {
				if len(args) == 0 {
					// --files with no args means read filenames from stdin (pre-commit style)
					if stdinPiped {
						return processFilesFromStdin(config, stdout, check)
					}
					return fmt.Errorf("--files requires at least one file argument")
				}
				// --files with args means process those files
				exitCode := process(args, stdout, check, config)
				if exitCode != 0 {
					os.Exit(exitCode)
				}
				return nil
			}

			// If stdin is piped and no --files flag, process stdin as content
			if stdinPiped && len(args) == 0 {
				return processStdin(config, check)
			}

			// Chart mode
			if len(args) != 1 {
				return fmt.Errorf("chart mode requires exactly one chart path")
			}

			root := filepath.Join(args[0], "templates")
			if _, err := os.Stat(root); err != nil {
				return err
			}

			chartFiles, err := collectFiles(root, config)
			if err != nil {
				return err
			}
			exitCode := process(chartFiles, false, check, config)
			if exitCode != 0 {
				os.Exit(exitCode)
			}
			return nil
		},
	}

	rootCmd.Flags().BoolVar(&files, "files", false, "Process specific files")
	rootCmd.Flags().BoolVar(&stdout, "stdout", false, "Output to stdout")
	rootCmd.Flags().BoolVar(&check, "check", false, "Check formatting without modifying files (exit 1 if unformatted)")
	rootCmd.Flags().StringSliceVar(&disableRules, "disable-indent", []string{}, "Disable specific indent rules (e.g., --disable-indent=printf,include)")
	rootCmd.Flags().StringSliceVar(&enableRules, "enable-indent", []string{}, "Enable specific indent rules (e.g., --enable-indent=printf,include)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func processFilesFromStdin(config *Config, stdout bool, check bool) error {
	// Read filenames from stdin (one per line)
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(input)), "\n")
	var filenames []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			filenames = append(filenames, line)
		}
	}

	if len(filenames) == 0 {
		return fmt.Errorf("no files provided via stdin")
	}

	exitCode := process(filenames, stdout, check, config)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

func processStdin(config *Config, check bool) error {
	// Read all input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}

	orig := string(input)

	// Validate syntax
	if err := validateTemplateSyntax(orig); err != nil {
		return fmt.Errorf("invalid syntax: %w", err)
	}

	formatted := ensureTrailingNewline(formatIndentation(orig, config, "<stdin>"))

	if check {
		if formatted != orig && formatted != orig+"\n" {
			fmt.Fprintln(os.Stderr, "[UNFORMATTED] <stdin>")
			os.Exit(1)
		}
		return nil
	}

	// Format and output to stdout
	fmt.Print(formatted)

	return nil
}

func collectFiles(root string, config *Config) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Walk error at %s: %v\n", path, err)
			return nil
		}
		if d.IsDir() || !wanted(path, config) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	return out, err
}

func process(files []string, stdout bool, check bool, config *Config) int {
	var total, updated, failed, unformatted int

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

		formatted := ensureTrailingNewline(formatIndentation(orig, config, file))

		if check {
			if formatted != orig && formatted != orig+"\n" {
				fmt.Fprintf(os.Stderr, "[UNFORMATTED] %s\n", file)
				unformatted++
			}
			continue
		}

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

	if check {
		if unformatted > 0 || failed > 0 {
			fmt.Fprintf(os.Stderr, "\n%d file(s) need formatting, %d error(s)\n", unformatted, failed)
			return 1
		}
		fmt.Printf("All %d file(s) are properly formatted\n", total)
		return 0
	}

	if !stdout {
		fmt.Printf("\nProcessed: %d files, Updated: %d, Errors: %d\n", total, updated, failed)
	}
	if failed > 0 {
		return 1
	}
	return 0
}

func wanted(path string, config *Config) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, validExt := range config.Extensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
