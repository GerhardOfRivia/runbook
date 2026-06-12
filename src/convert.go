package main

import (
	"strings"
)

// ExportToMarkdown converts a notebook to a markdown string.
func ExportToMarkdown(nb *Notebook) string {
	var sb strings.Builder
	for _, cell := range nb.Cells {
		if cell.CellType == "code" {
			sb.WriteString("```bash\n")
			sourceStr := cell.Source.String()
			sb.WriteString(sourceStr)
			if !strings.HasSuffix(sourceStr, "\n") {
				sb.WriteByte('\n')
			}
			sb.WriteString("```\n")
		} else { // markdown or other text cells
			sourceStr := cell.Source.String()
			sb.WriteString(sourceStr)
			if !strings.HasSuffix(sourceStr, "\n") {
				sb.WriteByte('\n')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ExportToShell converts a notebook to a shell script string.
func ExportToShell(nb *Notebook) string {
	var sb strings.Builder
	sb.WriteString("#!/bin/sh\n\n")
	for _, cell := range nb.Cells {
		if cell.CellType == "code" {
			sourceStr := cell.Source.String()
			sb.WriteString(sourceStr)
			if !strings.HasSuffix(sourceStr, "\n") {
				sb.WriteByte('\n')
			}
		} else { // markdown or other text cells
			sourceStr := cell.Source.String()
			if sourceStr == "" {
				continue
			}
			lines := strings.Split(sourceStr, "\n")
			for i, line := range lines {
				if i == len(lines)-1 && line == "" {
					continue
				}
				sb.WriteString("# ")
				sb.WriteString(line)
				sb.WriteByte('\n')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// CompileFromMarkdown parses a markdown string and reconstructs a Notebook.
func CompileFromMarkdown(markdownText string) (*Notebook, error) {
	lines := strings.Split(markdownText, "\n")
	var cells []Cell
	var currentLines []string
	isCode := false

	for i, line := range lines {
		// Skip the very last empty line resulting from a trailing newline in the file
		if i == len(lines)-1 && line == "" {
			break
		}

		if !isCode {
			if strings.HasPrefix(strings.TrimSpace(line), "```bash") {
				if len(currentLines) > 0 {
					pruned := pruneEmptyLines(currentLines)
					if len(pruned) > 0 {
						sourceStr := strings.Join(pruned, "\n")
						cells = append(cells, Cell{
							CellType: "markdown",
							Source:   toSourceArray(sourceStr),
							Metadata: make(map[string]interface{}),
						})
					}
					currentLines = nil
				}
				isCode = true
			} else {
				currentLines = append(currentLines, line)
			}
		} else {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				sourceStr := strings.Join(currentLines, "\n")
				cells = append(cells, Cell{
					CellType:       "code",
					Source:         toSourceArray(sourceStr),
					Metadata:       make(map[string]interface{}),
					ExecutionCount: nil,
					Outputs:        []Output{},
				})
				currentLines = nil
				isCode = false
			} else {
				currentLines = append(currentLines, line)
			}
		}
	}

	if len(currentLines) > 0 {
		cellType := "markdown"
		if isCode {
			cellType = "code"
		}
		if cellType == "markdown" {
			pruned := pruneEmptyLines(currentLines)
			if len(pruned) > 0 {
				sourceStr := strings.Join(pruned, "\n")
				cells = append(cells, Cell{
					CellType:       cellType,
					Source:         toSourceArray(sourceStr),
					Metadata:       make(map[string]interface{}),
					ExecutionCount: nil,
					Outputs:        []Output{},
				})
			}
		} else {
			sourceStr := strings.Join(currentLines, "\n")
			cells = append(cells, Cell{
				CellType:       cellType,
				Source:         toSourceArray(sourceStr),
				Metadata:       make(map[string]interface{}),
				ExecutionCount: nil,
				Outputs:        []Output{},
			})
		}
	}

	nb := &Notebook{
		Cells:         cells,
		Metadata:      make(map[string]interface{}),
		NbFormat:      4,
		NbFormatMinor: 5,
	}
	nb.Normalize()
	return nb, nil
}

func toSourceArray(text string) StringOrArray {
	if text == "" {
		return StringOrArray{}
	}
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, text[start:i+1])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return StringOrArray(lines)
}

func pruneEmptyLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}
