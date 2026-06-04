package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styling tokens
var (
	cursorColor        = lipgloss.Color("#7D56F4") // Vibrant Purple
	borderColor        = lipgloss.Color("#3C3C3C") // Dark Gray
	activeBorderColor  = lipgloss.Color("#7D56F4") // Purple
	textColor          = lipgloss.Color("#D9D9D9") // Light Gray
	subtleColor        = lipgloss.Color("#6272A4") // Muted blue/gray
	successColor       = lipgloss.Color("#50FA7B") // Green
	errorColor         = lipgloss.Color("#FF5555") // Red
	promptColor        = lipgloss.Color("#8BE9FD") // Cyan

	// Code cell border styles
	activeCodeStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(activeBorderColor).
			Padding(0, 1)

	inactiveCodeStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	// Markdown cell vertical bar styles
	activeMarkdownStyle = lipgloss.NewStyle().
				Border(lipgloss.Border{Left: "▌"}, false, false, false, true).
				BorderForeground(activeBorderColor).
				PaddingLeft(2)

	markdownStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{Left: "▌"}, false, false, false, true).
			BorderForeground(borderColor).
			PaddingLeft(2)

	// App layout styles
	titleStyle = lipgloss.NewStyle().
			Background(cursorColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Background(borderColor).
			Foreground(textColor).
			Padding(0, 1)
)

// MsgCellExecuted is sent when a cell finishes executing.
type MsgCellExecuted struct {
	CellIndex int
	Outputs   []Output
	Err       error
}

// TuiModel is the Bubble Tea model for the terminal user interface.
type TuiModel struct {
	notebook            *Notebook
	filepath            string
	activeCellIndex     int
	scrollLineOffset    int
	terminalWidth       int
	terminalHeight      int
	executingCellIndex  int
	nextExecutionCount  int
	err                 error
	statusMessage       string
	unsavedChanges      bool
}

// NewTuiModel initializes a new TuiModel.
func NewTuiModel(nb *Notebook, filepath string) *TuiModel {
	// Find the max execution count to increment from
	maxCount := 0
	for _, c := range nb.Cells {
		if c.ExecutionCount != nil && *c.ExecutionCount > maxCount {
			maxCount = *c.ExecutionCount
		}
	}

	return &TuiModel{
		notebook:           nb,
		filepath:           filepath,
		activeCellIndex:    0,
		scrollLineOffset:   0,
		terminalWidth:      80, // Safe default before size msg
		terminalHeight:     24, // Safe default before size msg
		executingCellIndex: -1, // -1 means none running
		nextExecutionCount: maxCount + 1,
		statusMessage:      "Ready",
	}
}

// Init initializes the Bubble Tea program.
func (m TuiModel) Init() tea.Cmd {
	return nil
}

// Update handles message updates.
func (m TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.activeCellIndex > 0 {
				m.activeCellIndex--
			}
			return m, nil

		case "down", "j":
			if m.activeCellIndex < len(m.notebook.Cells)-1 {
				m.activeCellIndex++
			}
			return m, nil

		case "enter", "r":
			// Execute selected code cell
			if m.executingCellIndex != -1 {
				m.statusMessage = "Wait, a cell is already running!"
				return m, nil
			}

			cell := m.notebook.Cells[m.activeCellIndex]
			if cell.CellType == "code" {
				m.executingCellIndex = m.activeCellIndex
				m.statusMessage = fmt.Sprintf("Running cell %d...", m.activeCellIndex+1)
				codeStr := cell.Source.String()

				// Execute in background
				return m, func() tea.Msg {
					outputs, err := RunCodeCell(codeStr)
					return MsgCellExecuted{
						CellIndex: m.activeCellIndex,
						Outputs:   outputs,
						Err:       err,
					}
				}
			}

		case "s", "ctrl+s":
			// Save the notebook
			err := SaveNotebook(m.filepath, m.notebook)
			if err != nil {
				m.err = err
				m.statusMessage = fmt.Sprintf("Error saving: %v", err)
			} else {
				m.unsavedChanges = false
				m.statusMessage = "Saved successfully!"
			}
			return m, nil
		}

	case MsgCellExecuted:
		// Ensure it updates the cell that was actually running
		idx := msg.CellIndex
		m.notebook.Cells[idx].Outputs = msg.Outputs
		m.notebook.Cells[idx].ExecutionCount = &m.nextExecutionCount
		m.nextExecutionCount++

		m.executingCellIndex = -1
		m.unsavedChanges = true
		if msg.Err != nil {
			m.statusMessage = fmt.Sprintf("Cell %d finished with error: %v", idx+1, msg.Err)
		} else {
			m.statusMessage = fmt.Sprintf("Cell %d finished successfully", idx+1)
		}
		return m, nil
	}

	return m, nil
}

// View renders the terminal screen.
func (m TuiModel) View() string {
	// Header/Title
	changedIndicator := ""
	if m.unsavedChanges {
		changedIndicator = " [modified]"
	}
	headerText := fmt.Sprintf("RUNBOOK: %s%s", m.filepath, changedIndicator)
	header := titleStyle.Width(m.terminalWidth).Render(headerText)

	// Viewport heights
	headerHeight := lipgloss.Height(header)
	footerHeight := 2 // status line + help line
	viewportHeight := m.terminalHeight - headerHeight - footerHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	// Render cells and compute line ranges
	var renderedCells []string
	var cellLineRanges []struct{ start, end int }
	currentLine := 0

	for i, cell := range m.notebook.Cells {
		isActive := (i == m.activeCellIndex)
		isExecuting := (m.executingCellIndex == i)

		var cellStr string
		if cell.CellType == "code" {
			cellStr = renderCodeCell(cell, isActive, m.terminalWidth, isExecuting)
		} else {
			cellStr = renderMarkdownCell(cell, isActive, m.terminalWidth)
		}

		lines := strings.Split(cellStr, "\n")
		lineCount := len(lines)
		cellLineRanges = append(cellLineRanges, struct{ start, end int }{
			start: currentLine,
			end:   currentLine + lineCount - 1,
		})
		renderedCells = append(renderedCells, cellStr)
		currentLine += lineCount + 1 // +1 for joining via \n\n
	}

	fullContent := strings.Join(renderedCells, "\n\n")
	allLines := strings.Split(fullContent, "\n")

	// Adjust scroll offset to keep selected cell in view
	if m.activeCellIndex >= 0 && m.activeCellIndex < len(cellLineRanges) {
		activeRange := cellLineRanges[m.activeCellIndex]
		cellHeight := activeRange.end - activeRange.start + 1

		if cellHeight >= viewportHeight {
			// If cell is too tall to fit, align top
			if m.scrollLineOffset > activeRange.start {
				m.scrollLineOffset = activeRange.start
			} else if m.scrollLineOffset+viewportHeight-1 < activeRange.start {
				m.scrollLineOffset = activeRange.start
			}
		} else {
			// standard scrolling
			if activeRange.start < m.scrollLineOffset {
				m.scrollLineOffset = activeRange.start
			} else if activeRange.end > m.scrollLineOffset+viewportHeight-1 {
				m.scrollLineOffset = activeRange.end - viewportHeight + 1
			}
		}
	}

	// Clamp scroll offset
	maxOffset := len(allLines) - viewportHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollLineOffset > maxOffset {
		m.scrollLineOffset = maxOffset
	}
	if m.scrollLineOffset < 0 {
		m.scrollLineOffset = 0
	}

	// Extract lines for the viewport
	var visibleLines []string
	for i := m.scrollLineOffset; i < m.scrollLineOffset+viewportHeight && i < len(allLines); i++ {
		visibleLines = append(visibleLines, allLines[i])
	}

	// Pad viewport to match requested height
	for len(visibleLines) < viewportHeight {
		visibleLines = append(visibleLines, "")
	}

	viewportContent := strings.Join(visibleLines, "\n")

	// Status Line
	cellInfo := fmt.Sprintf("Cell %d of %d", m.activeCellIndex+1, len(m.notebook.Cells))
	statusText := fmt.Sprintf(" %-50s %29s", m.statusMessage, cellInfo)
	statusBar := statusBarStyle.Width(m.terminalWidth).Render(statusText)

	// Help Line
	helpText := " [j/k/↑/↓] Navigate • [Enter/r] Run • [s/ctrl+s] Save • [q] Quit"
	helpBar := lipgloss.NewStyle().
		Foreground(subtleColor).
		Width(m.terminalWidth).
		Render(helpText)

	return header + "\n" + viewportContent + "\n" + statusBar + "\n" + helpBar
}

func renderCodeCell(cell Cell, isActive bool, width int, isExecuting bool) string {
	// The prompt: e.g. In [1]: or In [*]: or In [ ]:
	var prompt string
	if isExecuting {
		prompt = "In [*]: "
	} else if cell.ExecutionCount == nil {
		prompt = "In [ ]: "
	} else {
		prompt = fmt.Sprintf("In [%d]: ", *cell.ExecutionCount)
	}

	// Indent code lines and add prompt prefix
	codeStr := cell.Source.String()
	codeLines := strings.Split(codeStr, "\n")
	var indentedCode []string
	codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))
	promptStyle := lipgloss.NewStyle().Foreground(promptColor).Bold(true)

	for i, l := range codeLines {
		if i == 0 {
			indentedCode = append(indentedCode, promptStyle.Render(prompt)+codeStyle.Render(l))
		} else {
			indentedCode = append(indentedCode, strings.Repeat(" ", len(prompt))+codeStyle.Render(l))
		}
	}
	renderedCode := strings.Join(indentedCode, "\n")

	// Outputs container
	var outputsStr string
	if len(cell.Outputs) > 0 {
		var outputLines []string
		for _, out := range cell.Outputs {
			if out.OutputType == "stream" {
				var streamStyle lipgloss.Style
				if out.Name == "stderr" {
					streamStyle = lipgloss.NewStyle().Foreground(errorColor)
				} else {
					streamStyle = lipgloss.NewStyle().Foreground(textColor)
				}
				outputLines = append(outputLines, streamStyle.Render(out.Text.String()))
			} else if out.OutputType == "error" {
				errStyle := lipgloss.NewStyle().Foreground(errorColor).Bold(true)
				outputLines = append(outputLines, errStyle.Render(out.EName+": "+out.EValue))
				for _, tbLine := range out.Traceback {
					outputLines = append(outputLines, lipgloss.NewStyle().Foreground(errorColor).Render(tbLine))
				}
			}
		}
		if len(outputLines) > 0 {
			borderStyle := lipgloss.NewStyle().Foreground(subtleColor)
			outputsStr = "\n" + borderStyle.Render("── Outputs ──") + "\n" + strings.Join(outputLines, "\n")
		}
	}

	content := renderedCode + outputsStr

	// Box border styling
	if isActive {
		return activeCodeStyle.Width(width - 4).Render(content)
	}
	return inactiveCodeStyle.Width(width - 4).Render(content)
}

func renderMarkdownCell(cell Cell, isActive bool, width int) string {
	content := renderMarkdown(cell.Source.String(), width-6)
	if isActive {
		return activeMarkdownStyle.Width(width - 4).Render(content)
	}
	return markdownStyle.Width(width - 4).Render(content)
}

func renderMarkdown(source string, width int) string {
	lines := strings.Split(source, "\n")
	var formatted []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true).Underline(true)
			formatted = append(formatted, style.Render(title))
		} else if strings.HasPrefix(trimmed, "## ") {
			title := strings.TrimPrefix(trimmed, "## ")
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Bold(true)
			formatted = append(formatted, style.Render(title))
		} else if strings.HasPrefix(trimmed, "### ") {
			title := strings.TrimPrefix(trimmed, "### ")
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)
			formatted = append(formatted, style.Render(title))
		} else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			bullet := "• " + strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			formatted = append(formatted, lipgloss.NewStyle().Foreground(textColor).Render(bullet))
		} else {
			formatted = append(formatted, lipgloss.NewStyle().Foreground(textColor).Render(line))
		}
	}
	return strings.Join(formatted, "\n")
}
