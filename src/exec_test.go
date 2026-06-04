package main

import (
	"testing"
)

func TestRunCodeCellSuccess(t *testing.T) {
	outputs, err := RunCodeCell("echo 'Hello World'")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(outputs) != 1 {
		t.Fatalf("Expected 1 output stream, got %d", len(outputs))
	}

	out := outputs[0]
	if out.OutputType != "stream" || out.Name != "stdout" {
		t.Errorf("Expected stdout stream, got %s:%s", out.OutputType, out.Name)
	}

	if out.Text.String() != "Hello World\n" {
		t.Errorf("Expected 'Hello World\\n', got %q", out.Text.String())
	}
}

func TestRunCodeCellFailure(t *testing.T) {
	outputs, err := RunCodeCell("echo 'Failed' >&2 && exit 3")
	if err == nil {
		t.Fatal("Expected error running failing command, got nil")
	}

	// Should have stderr and error output
	if len(outputs) != 2 {
		t.Fatalf("Expected 2 outputs (stderr stream + error), got %d", len(outputs))
	}

	stderrOut := outputs[0]
	if stderrOut.OutputType != "stream" || stderrOut.Name != "stderr" {
		t.Errorf("Expected stderr stream, got %s:%s", stderrOut.OutputType, stderrOut.Name)
	}
	if stderrOut.Text.String() != "Failed\n" {
		t.Errorf("Expected 'Failed\\n', got %q", stderrOut.Text.String())
	}

	errOut := outputs[1]
	if errOut.OutputType != "error" {
		t.Errorf("Expected output type error, got %s", errOut.OutputType)
	}
	if errOut.EName != "ExitError" {
		t.Errorf("Expected EName ExitError, got %q", errOut.EName)
	}
}
