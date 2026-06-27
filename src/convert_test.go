package main

import (
	"testing"
)

func TestCompileFromMarkdownEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		markdownText  string
		expectedCells []struct {
			cellType string
			source   string
		}
	}{
		{
			name: "Standard bash code block",
			markdownText: "```bash\necho \"hello\"\n```",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{cellType: "code", source: "echo \"hello\""},
			},
		},
		{
			name: "Alternative language tags: sh and shell",
			markdownText: "```sh\necho \"sh\"\n```\n\n```shell\necho \"shell\"\n```",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{cellType: "code", source: "echo \"sh\""},
				{cellType: "code", source: "echo \"shell\""},
			},
		},
		{
			name: "Powershell and pwsh code blocks",
			markdownText: "```powershell\nGet-Process\n```\n\n```pwsh\nGet-Service\n```",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{cellType: "code", source: "Get-Process"},
				{cellType: "code", source: "Get-Service"},
			},
		},
		{
			name: "Flexible spacing inside fences",
			markdownText: "  ``` bash \necho \"space\"\n  ```\n\n```    shell\necho \"more space\"\n```",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{cellType: "code", source: "echo \"space\""},
				{cellType: "code", source: "echo \"more space\""},
			},
		},
		{
			name: "Code fence with attributes",
			markdownText: "```bash { id=run-me }\necho \"attrs\"\n```",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{cellType: "code", source: "echo \"attrs\""},
			},
		},
		{
			name: "Code block of different language should stay markdown",
			markdownText: "Some text\n```python\nprint(\"hello\")\n```\nMore text",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{cellType: "markdown", source: "Some text\n```python\nprint(\"hello\")\n```\nMore text"},
			},
		},
		{
			name: "Invalid prefix starting with bash/sh/shell should stay markdown",
			markdownText: "```bashful\necho \"not bash\"\n```\n\n```shbn\necho \"not sh\"\n```powershell-test\nGet-Process\n```\n```pwsh-test\nGet-Service\n```\n```",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{
					cellType: "markdown",
					source:   "```bashful\necho \"not bash\"\n```\n\n```shbn\necho \"not sh\"\n```powershell-test\nGet-Process\n```\n```pwsh-test\nGet-Service\n```\n```",
				},
			},
		},
		{
			name: "Windows CRLF line endings",
			markdownText: "# Heading\r\n\r\n```bash\r\necho \"CRLF\"\r\n```\r\n",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{cellType: "markdown", source: "# Heading"},
				{cellType: "code", source: "echo \"CRLF\""},
			},
		},
		{
			name: "Unclosed code block at end of file",
			markdownText: "# Heading\n```bash\necho \"unclosed\"",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{cellType: "markdown", source: "# Heading"},
				{cellType: "code", source: "echo \"unclosed\""},
			},
		},
		{
			name: "Empty markdown lines only",
			markdownText: "\n\n\n",
			expectedCells: []struct {
				cellType string
				source   string
			}{}, // Expect no cells because of pruning
		},
		{
			name: "Empty code cell in markdown",
			markdownText: "```bash\n```",
			expectedCells: []struct {
				cellType string
				source   string
			}{
				{cellType: "code", source: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nb, err := CompileFromMarkdown(tt.markdownText)
			if err != nil {
				t.Fatalf("Failed to compile: %v", err)
			}

			if len(nb.Cells) != len(tt.expectedCells) {
				t.Fatalf("Expected %d cells, got %d", len(tt.expectedCells), len(nb.Cells))
			}

			for i, ec := range tt.expectedCells {
				cell := nb.Cells[i]
				if cell.CellType != ec.cellType {
					t.Errorf("Cell %d type mismatch: expected %q, got %q", i, ec.cellType, cell.CellType)
				}
				actualSource := cell.Source.String()
				if actualSource != ec.source {
					t.Errorf("Cell %d source mismatch:\nExpected:\n%q\nGot:\n%q", i, ec.source, actualSource)
				}
			}

			if tt.name == "Powershell and pwsh code blocks" {
				if lang, ok := nb.Cells[0].Metadata["language"].(string); !ok || lang != "pwsh" {
					t.Errorf("Cell 0 language expected \"pwsh\", got %q", lang)
				}
				if lang, ok := nb.Cells[1].Metadata["language"].(string); !ok || lang != "pwsh" {
					t.Errorf("Cell 1 language expected \"pwsh\", got %q", lang)
				}
			}
		})
	}
}

func TestExportToMarkdownAndShellEdgeCases(t *testing.T) {
	nb := &Notebook{
		Cells: []Cell{
			{
				CellType: "markdown",
				Source:   StringOrArray{"# Heading\r\n", "Second line\r\n"},
				Metadata: make(map[string]interface{}),
			},
			{
				CellType: "code",
				Source:   StringOrArray{"echo \"test\"\r\n"},
				Metadata: make(map[string]interface{}),
			},
		},
	}

	// Test ExportToMarkdown CRLF preservation/handling
	mdOut := ExportToMarkdown(nb)
	if !contains(mdOut, "# Heading\r\nSecond line\r\n") {
		t.Errorf("Markdown export did not preserve content properly, got: %q", mdOut)
	}

	// Test ExportToShell CRLF handling
	shOut := ExportToShell(nb)
	expectedSh := "#!/bin/sh\n\n# # Heading\r\n# Second line\r\n\necho \"test\"\r\n\n"
	if shOut != expectedSh {
		t.Errorf("Shell export mismatch:\nExpected:\n%q\nGot:\n%q", expectedSh, shOut)
	}
}

func TestExportToMarkdownAndShellPowerShell(t *testing.T) {
	nb := &Notebook{
		Cells: []Cell{
			{
				CellType: "code",
				Source:   StringOrArray{"Get-Process\n"},
				Metadata: map[string]interface{}{"language": "pwsh"},
			},
			{
				CellType: "code",
				Source:   StringOrArray{"echo \"bash\"\n"},
				Metadata: map[string]interface{}{},
			},
		},
	}

	mdOut := ExportToMarkdown(nb)
	if !contains(mdOut, "```pwsh\nGet-Process") {
		t.Errorf("Expected markdown to contain ```pwsh, got: %q", mdOut)
	}

	shOut := ExportToShell(nb)
	expectedSh := "pwsh -NoProfile - <<'EOF'\nGet-Process\nEOF"
	if !contains(shOut, expectedSh) {
		t.Errorf("Expected shell export to wrap powershell cell, got:\n%s", shOut)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
