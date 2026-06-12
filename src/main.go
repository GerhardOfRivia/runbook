package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var Version = "dev"

type Config struct {
	ToMd     bool
	ToSh     bool
	FromMd   bool
	Version  bool
	FilePath string
}

func ParseArgs(args []string) (Config, error) {
	var cfg Config

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--to-md":
			cfg.ToMd = true
		case "--to-sh":
			cfg.ToSh = true
		case "--from-md":
			cfg.FromMd = true
		default:
			if strings.HasPrefix(arg, "-") {
				return Config{}, fmt.Errorf("unknown flag: %s", arg)
			}
			if cfg.FilePath != "" {
				return Config{}, fmt.Errorf("multiple file paths specified")
			}
			cfg.FilePath = arg
		}
	}

	if cfg.Version {
		return cfg, nil
	}

	if cfg.FilePath == "" {
		return Config{}, fmt.Errorf("no input file specified")
	}

	// Check flag mutual exclusivity
	flagCount := 0
	if cfg.ToMd {
		flagCount++
	}
	if cfg.ToSh {
		flagCount++
	}
	if cfg.FromMd {
		flagCount++
	}

	if flagCount > 1 {
		return Config{}, fmt.Errorf("flags --to-md, --to-sh, and --from-md are mutually exclusive")
	}

	return cfg, nil
}

func main() {
	cfg, err := ParseArgs(os.Args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		printUsage(Version)
		os.Exit(1)
	}

	if cfg.FromMd {
		// Read markdown, compile to JSON
		data, err := os.ReadFile(cfg.FilePath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
		nb, err := CompileFromMarkdown(string(data))
		if err != nil {
			fmt.Printf("Error compiling markdown: %v\n", err)
			os.Exit(1)
		}
		// Output JSON to stdout
		nb.Normalize()
		jsonData, err := json.MarshalIndent(nb, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling notebook: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
		return
	}

	if cfg.ToMd {
		notebook, err := LoadNotebook(cfg.FilePath)
		if err != nil {
			fmt.Printf("Error loading notebook: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(ExportToMarkdown(notebook))
		return
	}

	if cfg.ToSh {
		notebook, err := LoadNotebook(cfg.FilePath)
		if err != nil {
			fmt.Printf("Error loading notebook: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(ExportToShell(notebook))
		return
	}

	// Default TUI mode
	notebook, err := LoadNotebook(cfg.FilePath)
	if err != nil {
		fmt.Printf("Error loading notebook: %v\n", err)
		os.Exit(1)
	}

	model := NewTuiModel(notebook, cfg.FilePath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func printUsage(version string) {
	fmt.Printf("Usage: runbook (%s) [options] [file_name.shbn]\n", version)
	fmt.Println("  runbook <file_name>.shbn              # Run TUI for notebook")
	fmt.Println("  runbook --to-md <file_name>.shbn      # Export notebook to Markdown")
	fmt.Println("  runbook --to-sh <file_name>.shbn      # Export notebook to Shell Script")
	fmt.Println("  runbook --from-md <file_name>.md      # Compile Markdown to notebook JSON")
}
