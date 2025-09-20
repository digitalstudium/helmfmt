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
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Input       string `yaml:"input"`
	Expected    string `yaml:"expected"`
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
			if testCase.Description != "" {
				t.Logf("Description: %s", testCase.Description)
			}

			// Create temp file with test input
			tmpfile, err := os.CreateTemp("", "helmfmt-test-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name()) // cleanup after test
			defer tmpfile.Close()

			if _, err := tmpfile.Write([]byte(testCase.Input)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call the REAL process function from main.go with stdout=true
			exitCode := process([]string{tmpfile.Name()}, true)

			// Close writer, restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatal(err)
			}
			result := buf.String()

			// Check for processing errors
			if exitCode != 0 {
				t.Fatalf("process() failed with exit code %d for test '%s'", exitCode, testCase.Name)
			}

			// Compare result
			if result != testCase.Expected {
				t.Errorf("Test '%s' failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s",
					testCase.Name, testCase.Input, testCase.Expected, result)
			}
		})
	}
}
