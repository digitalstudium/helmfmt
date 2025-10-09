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
				"template": {Disabled: false, Exclude: []string{}},
				"printf":   {Disabled: false, Exclude: []string{}},
				"include":  {Disabled: false, Exclude: []string{}},
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
	var stdout, files bool
	var disableRules, enableRules []string

	var rootCmd = &cobra.Command{
		Use:     "helmfmt [flags] [chart-path | file1 file2 ...]",
		Short:   "Format Helm templates",
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if stdinPiped {
				// Process from stdin
				if len(args) > 0 {
					return fmt.Errorf("cannot specify files when reading from stdin")
				}
				return processStdin(config)
			}

			if files {
				// Files mode
				if len(args) == 0 {
					return fmt.Errorf("--files requires at least one file argument")
				}
				exitCode := process(args, stdout, config)
				if exitCode != 0 {
					os.Exit(exitCode)
				}
				return nil
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
			exitCode := process(chartFiles, false, config)
			if exitCode != 0 {
				os.Exit(exitCode)
			}
			return nil
		},
	}

	rootCmd.Flags().BoolVar(&files, "files", false, "Process specific files")
	rootCmd.Flags().BoolVar(&stdout, "stdout", false, "Output to stdout")
	rootCmd.Flags().StringSliceVar(&disableRules, "disable-indent", []string{}, "Disable specific indent rules (e.g., --disable-indent=printf,include)")
	rootCmd.Flags().StringSliceVar(&enableRules, "enable-indent", []string{}, "Enable specific indent rules (e.g., --enable-indent=printf,include)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func processStdin(config *Config) error {
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

	// Format and output to stdout
	formatted := ensureTrailingNewline(formatIndentation(orig, config, "<stdin>"))
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

func process(files []string, stdout bool, config *Config) int {
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

		formatted := ensureTrailingNewline(formatIndentation(orig, config, file))

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
