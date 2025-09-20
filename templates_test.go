package main

import (
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

			result := formatIndentation(testCase.Input)
			if result != testCase.Expected {
				t.Errorf("Test '%s' failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s",
					testCase.Name, testCase.Input, testCase.Expected, result)
			}
		})
	}
}
