package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseNotebook(t *testing.T) {
	jsonStr := `{
  "cells": [
    {
      "cell_type": "code",
      "execution_count": 1,
      "id": "a1b2c3d4",
      "metadata": {},
      "source": [
        "echo 'Hello, World!'"
      ],
      "outputs": [
        {
          "output_type": "stream",
          "name": "stdout",
          "text": [
            "Hello, World!\n"
          ]
        }
      ]
    },
    {
      "cell_type": "markdown",
      "id": "e5f6g7h8",
      "metadata": {},
      "source": [
        "# This is a Heading\n",
        "This is descriptive text."
      ]
    }
  ],
  "metadata": {},
  "nbformat": 4,
  "nbformat_minor": 5
}`

	var nb Notebook
	err := json.Unmarshal([]byte(jsonStr), &nb)
	if err != nil {
		t.Fatalf("Failed to unmarshal notebook: %v", err)
	}

	if len(nb.Cells) != 2 {
		t.Errorf("Expected 2 cells, got %d", len(nb.Cells))
	}

	// Test code cell
	c1 := nb.Cells[0]
	if c1.CellType != "code" {
		t.Errorf("Expected first cell to be code, got %s", c1.CellType)
	}
	if c1.ExecutionCount == nil || *c1.ExecutionCount != 1 {
		t.Errorf("Expected execution_count to be 1, got %v", c1.ExecutionCount)
	}
	if c1.Source.String() != "echo 'Hello, World!'" {
		t.Errorf("Expected source to be \"echo 'Hello, World!'\", got %q", c1.Source.String())
	}
	if len(c1.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(c1.Outputs))
	}
	if c1.Outputs[0].OutputType != "stream" || c1.Outputs[0].Name != "stdout" {
		t.Errorf("Expected stdout stream output, got type=%q name=%q", c1.Outputs[0].OutputType, c1.Outputs[0].Name)
	}
	if c1.Outputs[0].Text.String() != "Hello, World!\n" {
		t.Errorf("Expected stream output text \"Hello, World!\\n\", got %q", c1.Outputs[0].Text.String())
	}

	// Test markdown cell
	c2 := nb.Cells[1]
	if c2.CellType != "markdown" {
		t.Errorf("Expected second cell to be markdown, got %s", c2.CellType)
	}
	expectedMarkdown := "# This is a Heading\nThis is descriptive text."
	if c2.Source.String() != expectedMarkdown {
		t.Errorf("Expected markdown source %q, got %q", expectedMarkdown, c2.Source.String())
	}
}

func TestLoadSaveNotebook(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "notebook-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.shbn")
	initialJSON := `{
  "cells": [
    {
      "cell_type": "code",
      "execution_count": null,
      "source": "echo 'Testing string source'"
    }
  ],
  "nbformat": 4,
  "nbformat_minor": 5
}`

	if err := os.WriteFile(filePath, []byte(initialJSON), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	nb, err := LoadNotebook(filePath)
	if err != nil {
		t.Fatalf("Failed to load notebook: %v", err)
	}

	if len(nb.Cells) != 1 {
		t.Fatalf("Expected 1 cell, got %d", len(nb.Cells))
	}

	if nb.Cells[0].Source.String() != "echo 'Testing string source'" {
		t.Errorf("Expected source 'echo 'Testing string source'', got %q", nb.Cells[0].Source.String())
	}

	// Modify
	newCount := 42
	nb.Cells[0].ExecutionCount = &newCount
	nb.Cells[0].Outputs = []Output{
		{
			OutputType: "stream",
			Name:       "stdout",
			Text:       StringOrArray{"Testing string source\n"},
		},
	}

	err = SaveNotebook(filePath, nb)
	if err != nil {
		t.Fatalf("Failed to save notebook: %v", err)
	}

	// Reload
	nbReloaded, err := LoadNotebook(filePath)
	if err != nil {
		t.Fatalf("Failed to reload notebook: %v", err)
	}

	if nbReloaded.Cells[0].ExecutionCount == nil || *nbReloaded.Cells[0].ExecutionCount != 42 {
		t.Errorf("Expected execution count 42, got %v", nbReloaded.Cells[0].ExecutionCount)
	}

	if len(nbReloaded.Cells[0].Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(nbReloaded.Cells[0].Outputs))
	}

	if nbReloaded.Cells[0].Outputs[0].Text.String() != "Testing string source\n" {
		t.Errorf("Expected output text 'Testing string source\\n', got %q", nbReloaded.Cells[0].Outputs[0].Text.String())
	}
}

func TestConversions(t *testing.T) {
	nb := &Notebook{
		Cells: []Cell{
			{
				CellType: "markdown",
				Source:   StringOrArray{"# Welcome to Runbook!\n", "This is a markdown cell explaining how to use it."},
				Metadata: make(map[string]interface{}),
			},
			{
				CellType: "code",
				Source:   StringOrArray{"echo 'Running a bash command...'\n", "ls -la"},
				Metadata: make(map[string]interface{}),
			},
		},
		Metadata:      make(map[string]interface{}),
		NbFormat:      4,
		NbFormatMinor: 5,
	}

	// Test ExportToMarkdown
	mdOutput := ExportToMarkdown(nb)
	expectedMd := `# Welcome to Runbook!
This is a markdown cell explaining how to use it.

` + "```" + `bash
echo 'Running a bash command...'
ls -la
` + "```" + `

`
	if mdOutput != expectedMd {
		t.Errorf("ExportToMarkdown output mismatch.\nExpected:\n%q\nGot:\n%q", expectedMd, mdOutput)
	}

	// Test ExportToShell
	shOutput := ExportToShell(nb)
	expectedSh := `#!/bin/sh

# # Welcome to Runbook!
# This is a markdown cell explaining how to use it.

echo 'Running a bash command...'
ls -la

`
	if shOutput != expectedSh {
		t.Errorf("ExportToShell output mismatch.\nExpected:\n%q\nGot:\n%q", expectedSh, shOutput)
	}

	// Test CompileFromMarkdown
	compiledNb, err := CompileFromMarkdown(expectedMd)
	if err != nil {
		t.Fatalf("CompileFromMarkdown failed: %v", err)
	}

	if len(compiledNb.Cells) != 2 {
		t.Fatalf("Expected 2 cells, got %d", len(compiledNb.Cells))
	}

	c1 := compiledNb.Cells[0]
	if c1.CellType != "markdown" {
		t.Errorf("Expected cell 1 to be markdown, got %s", c1.CellType)
	}
	expectedC1Source := "# Welcome to Runbook!\nThis is a markdown cell explaining how to use it."
	if c1.Source.String() != expectedC1Source {
		t.Errorf("Cell 1 source mismatch. Expected %q, got %q", expectedC1Source, c1.Source.String())
	}

	c2 := compiledNb.Cells[1]
	if c2.CellType != "code" {
		t.Errorf("Expected cell 2 to be code, got %s", c2.CellType)
	}
	expectedC2Source := "echo 'Running a bash command...'\nls -la"
	if c2.Source.String() != expectedC2Source {
		t.Errorf("Cell 2 source mismatch. Expected %q, got %q", expectedC2Source, c2.Source.String())
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    Config
		wantErr bool
	}{
		{
			name: "export to md",
			args: []string{"runbook", "--to-md", "file.shbn"},
			want: Config{
				ToMd:     true,
				FilePath: "file.shbn",
			},
			wantErr: false,
		},
		{
			name: "export to sh",
			args: []string{"runbook", "--to-sh", "file.shbn"},
			want: Config{
				ToSh:     true,
				FilePath: "file.shbn",
			},
			wantErr: false,
		},
		{
			name: "compile from md",
			args: []string{"runbook", "--from-md", "file.md"},
			want: Config{
				FromMd:   true,
				FilePath: "file.md",
			},
			wantErr: false,
		},
		{
			name: "no flags default TUI",
			args: []string{"runbook", "file.shbn"},
			want: Config{
				FilePath: "file.shbn",
			},
			wantErr: false,
		},
		{
			name:    "missing file path",
			args:    []string{"runbook"},
			want:    Config{},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"runbook", "--invalid"},
			want:    Config{},
			wantErr: true,
		},
		{
			name:    "multiple files",
			args:    []string{"runbook", "file1.shbn", "file2.shbn"},
			want:    Config{},
			wantErr: true,
		},
		{
			name:    "mutually exclusive flags",
			args:    []string{"runbook", "--to-md", "--to-sh", "file.shbn"},
			want:    Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.ToMd != tt.want.ToMd || got.ToSh != tt.want.ToSh || got.FromMd != tt.want.FromMd || got.Version != tt.want.Version || got.FilePath != tt.want.FilePath {
					t.Errorf("ParseArgs() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}
