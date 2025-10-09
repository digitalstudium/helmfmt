package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type TestCase struct {
	Name         string  `yaml:"name"`
	Config       *Config `yaml:"config,omitempty"` // Full config structure
	InputFile    string  `yaml:"input_file"`
	ExpectedFile string  `yaml:"expected_file"`
}

func TestFormatIndentationFromTemplates(t *testing.T) {
	testDir := "templates_test"

	files, err := filepath.Glob(filepath.Join(testDir, "*.yaml"))
	if err != nil {
		t.Fatalf("Failed to read test template files: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No test template files found")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("Failed to read test file %s: %v", file, err)
			}

			var testCase TestCase
			if err := yaml.Unmarshal(data, &testCase); err != nil {
				t.Fatalf("Failed to parse test file %s: %v", file, err)
			}

			t.Logf("Running test: %s", testCase.Name)

			// Load default config and apply test-specific overrides
			config := loadConfig()
			if testCase.Config != nil {
				// Merge test config with default config
				if testCase.Config.IndentSize != 0 {
					config.IndentSize = testCase.Config.IndentSize
				}
				if testCase.Config.Extensions != nil {
					config.Extensions = testCase.Config.Extensions
				}
				if testCase.Config.Rules.Indent != nil {
					// Merge indent rules
					for ruleName, ruleConfig := range testCase.Config.Rules.Indent {
						config.Rules.Indent[ruleName] = ruleConfig
					}
				}
			}

			// Read input file
			inputPath := filepath.Join(testDir, testCase.InputFile)
			inputContent, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("Failed to read input file %s: %v", inputPath, err)
			}

			// Read expected file
			expectedPath := filepath.Join(testDir, testCase.ExpectedFile)
			expectedContent, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read expected file %s: %v", expectedPath, err)
			}

			// Test 1: Direct formatting (file mode)
			result := formatIndentation(string(inputContent), config, testCase.InputFile)
			result = ensureTrailingNewline(result)
			expected := string(expectedContent)

			// Compare result
			if result != expected {
				t.Errorf("Test '%s' failed\nInput file: %s\nExpected file: %s\nExpected:\n%s\n\nGot:\n%s",
					testCase.Name, testCase.InputFile, testCase.ExpectedFile, expected, result)
			}

			// Test 2: stdin mode (skip if test uses file exclusion patterns)
			hasExcludePatterns := false
			if testCase.Config != nil && testCase.Config.Rules.Indent != nil {
				for _, ruleConfig := range testCase.Config.Rules.Indent {
					if len(ruleConfig.Exclude) > 0 {
						hasExcludePatterns = true
						break
					}
				}
			}

			if !hasExcludePatterns {
				t.Run("stdin", func(t *testing.T) {
					// Setup stdin/stdout pipes
					oldStdin, oldStdout := os.Stdin, os.Stdout
					r, w, _ := os.Pipe()
					os.Stdin = r

					rOut, wOut, _ := os.Pipe()
					os.Stdout = wOut

					// Write input and process
					go func() {
						w.Write(inputContent)
						w.Close()
					}()

					processStdin(config)

					wOut.Close()
					os.Stdin, os.Stdout = oldStdin, oldStdout

					// Read output
					var buf bytes.Buffer
					io.Copy(&buf, rOut)

					if buf.String() != expected {
						t.Errorf("Test '%s' failed (stdin)\nExpected:\n%s\n\nGot:\n%s",
							testCase.Name, expected, buf.String())
					}
				})
			}
		})
	}
}
