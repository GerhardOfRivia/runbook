package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: runbook <file_name>.shbn")
		os.Exit(1)
	}

	filePath := os.Args[1]
	notebook, err := LoadNotebook(filePath)
	if err != nil {
		fmt.Printf("Error loading notebook: %v\n", err)
		os.Exit(1)
	}

	model := NewTuiModel(notebook, filePath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
