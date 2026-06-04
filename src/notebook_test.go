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
