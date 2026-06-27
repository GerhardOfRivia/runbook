package main

import (
	"strings"
	"testing"
)

func TestRunCodeCellSuccess(t *testing.T) {
	outputs, start, end, err := RunCodeCell("echo 'Hello World'", "bash", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if start.IsZero() || end.IsZero() {
		t.Error("Expected non-zero start/end times")
	}

	if end.Before(start) {
		t.Error("Expected end time to be after or equal to start time")
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
	outputs, _, _, err := RunCodeCell("echo 'Failed' >&2 && exit 3", "bash", "")
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

func TestCleanStderr(t *testing.T) {
	input := "[sudo] password for user:\nSome normal stderr message\n[sudo] password for user:\nAnother message"
	got := cleanStderr(input)
	want := "Some normal stderr message\nAnother message"
	if got != want {
		t.Errorf("cleanStderr() = %q, want %q", got, want)
	}
}

func TestRunCodeCellPowerShell(t *testing.T) {
	outputs, _, _, err := RunCodeCell("Write-Output 'Hello PowerShell'", "pwsh", "")
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "executable file not found") || strings.Contains(errStr, "no such file or directory") {
			t.Skip("pwsh not installed, skipping PowerShell execution test")
		} else {
			t.Fatalf("Expected no error or missing executable error, got %v", err)
		}
	} else {
		if len(outputs) != 1 {
			t.Fatalf("Expected 1 output, got %d", len(outputs))
		}
		out := outputs[0]
		if out.Text.String() != "Hello PowerShell\n" && out.Text.String() != "Hello PowerShell\r\n" {
			t.Errorf("Unexpected output: %q", out.Text.String())
		}
	}
}
