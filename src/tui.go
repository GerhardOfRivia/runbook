package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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

	sudoConfirmStatusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FFB86C")). // Orange background
			Foreground(lipgloss.Color("#000000")). // Black text
			Bold(true).
			Padding(0, 1)
)

// MsgCellExecuted is sent when a cell finishes executing.
type MsgCellExecuted struct {
	CellIndex int
	Outputs   []Output
	StartTime time.Time
	EndTime   time.Time
	Err       error
}

// TuiModel is the Bubble Tea model for the terminal user interface.
type TuiModel struct {
	notebook                *Notebook
	filepath                string
	activeCellIndex         int
	scrollLineOffset        int
	terminalWidth           int
	terminalHeight          int
	executingCellIndex      int
	confirmingSudoCellIndex int
	enteringPasswordCellIdx int
	passwordInput           textinput.Model
	sudoPassword            string
	nextExecutionCount      int
	err                     error
	statusMessage           string
	unsavedChanges          bool
	searching               bool
	searchInput             textinput.Model
	searchQuery             string
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

	ti := textinput.New()
	ti.Placeholder = "Enter sudo password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()

	si := textinput.New()
	si.Prompt = "/"

	return &TuiModel{
		notebook:                nb,
		filepath:                filepath,
		activeCellIndex:         0,
		scrollLineOffset:        0,
		terminalWidth:           80, // Safe default before size msg
		terminalHeight:          24, // Safe default before size msg
		executingCellIndex:      -1, // -1 means none running
		confirmingSudoCellIndex: -1,
		enteringPasswordCellIdx: -1,
		passwordInput:           ti,
		searchInput:             si,
		nextExecutionCount:      maxCount + 1,
		statusMessage:           "Ready",
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
		if m.enteringPasswordCellIdx != -1 {
			switch msg.String() {
			case "enter":
				m.sudoPassword = m.passwordInput.Value()
				m.passwordInput.SetValue("")
				idx := m.enteringPasswordCellIdx
				m.enteringPasswordCellIdx = -1
				m.executingCellIndex = idx
				m.statusMessage = fmt.Sprintf("Running cell %d...", idx+1)
				cell := m.notebook.Cells[idx]
				codeStr := cell.Source.String()
				lang := "bash"
				if cell.Metadata != nil {
					if l, ok := cell.Metadata["language"].(string); ok && l != "" {
						lang = l
					}
				}

				return m, func() tea.Msg {
					outputs, startTime, endTime, err := RunCodeCell(codeStr, lang, m.sudoPassword)
					return MsgCellExecuted{
						CellIndex: idx,
						Outputs:   outputs,
						StartTime: startTime,
						EndTime:   endTime,
						Err:       err,
					}
				}
			case "esc", "ctrl+c":
				m.enteringPasswordCellIdx = -1
				m.passwordInput.SetValue("")
				m.statusMessage = "Execution cancelled."
				return m, nil
			default:
				var cmd tea.Cmd
				m.passwordInput, cmd = m.passwordInput.Update(msg)
				return m, cmd
			}
		}

		if m.confirmingSudoCellIndex != -1 {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "y", "Y":
				idx := m.confirmingSudoCellIndex
				m.confirmingSudoCellIndex = -1

				if m.sudoPassword == "" {
					m.enteringPasswordCellIdx = idx
					m.passwordInput.Focus()
					m.statusMessage = "Please enter sudo password:"
					return m, nil
				}

				m.executingCellIndex = idx
				m.statusMessage = fmt.Sprintf("Running cell %d...", idx+1)
				cell := m.notebook.Cells[idx]
				codeStr := cell.Source.String()
				lang := "bash"
				if cell.Metadata != nil {
					if l, ok := cell.Metadata["language"].(string); ok && l != "" {
						lang = l
					}
				}

				// Execute in background
				return m, func() tea.Msg {
					outputs, startTime, endTime, err := RunCodeCell(codeStr, lang, m.sudoPassword)
					return MsgCellExecuted{
						CellIndex: idx,
						Outputs:   outputs,
						StartTime: startTime,
						EndTime:   endTime,
						Err:       err,
					}
				}
			case "n", "N", "esc":
				m.confirmingSudoCellIndex = -1
				m.statusMessage = "Execution cancelled."
				return m, nil
			default:
				return m, nil
			}
		}

		if m.searching {
			switch msg.String() {
			case "enter":
				m.searchQuery = m.searchInput.Value()
				m.searching = false
				m.searchInput.Blur()
				m.searchForward(m.activeCellIndex)
				return m, nil
			case "esc", "ctrl+c":
				m.searching = false
				m.searchInput.Blur()
				m.statusMessage = "Search cancelled."
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		}

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

		case "/":
			m.searching = true
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			return m, textinput.Blink

		case "n":
			if m.searchQuery == "" {
				m.statusMessage = "No search pattern"
				return m, nil
			}
			m.searchForward(m.activeCellIndex + 1)
			return m, nil

		case "N":
			if m.searchQuery == "" {
				m.statusMessage = "No search pattern"
				return m, nil
			}
			m.searchBackward(m.activeCellIndex - 1)
			return m, nil

		case "enter", "r":
			// Execute selected code cell
			if m.executingCellIndex != -1 {
				m.statusMessage = "Wait, a cell is already running!"
				return m, nil
			}

			cell := m.notebook.Cells[m.activeCellIndex]
			if cell.CellType == "code" {
				codeStr := cell.Source.String()
				lang := "bash"
				if cell.Metadata != nil {
					if l, ok := cell.Metadata["language"].(string); ok && l != "" {
						lang = l
					}
				}

				if hasSudo(codeStr) && lang != "powershell" && lang != "pwsh" {
					m.confirmingSudoCellIndex = m.activeCellIndex
					m.statusMessage = fmt.Sprintf("Cell %d contains sudo. Run anyway? (y/n)", m.activeCellIndex+1)
					return m, nil
				}

				m.executingCellIndex = m.activeCellIndex
				m.statusMessage = fmt.Sprintf("Running cell %d...", m.activeCellIndex+1)

				// Execute in background
				return m, func() tea.Msg {
					outputs, startTime, endTime, err := RunCodeCell(codeStr, lang, "")
					return MsgCellExecuted{
						CellIndex: m.activeCellIndex,
						Outputs:   outputs,
						StartTime: startTime,
						EndTime:   endTime,
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

		if m.notebook.Cells[idx].Metadata == nil {
			m.notebook.Cells[idx].Metadata = make(map[string]interface{})
		}
		m.notebook.Cells[idx].Metadata["start_time"] = msg.StartTime.Format(time.RFC3339Nano)
		m.notebook.Cells[idx].Metadata["end_time"] = msg.EndTime.Format(time.RFC3339Nano)
		duration := msg.EndTime.Sub(msg.StartTime)
		m.notebook.Cells[idx].Metadata["duration_ms"] = duration.Milliseconds()

		m.executingCellIndex = -1
		m.unsavedChanges = true
		durationStr := formatDuration(duration)
		if msg.Err != nil {
			m.statusMessage = fmt.Sprintf("Cell %d finished with error in %s: %v", idx+1, durationStr, msg.Err)
		} else {
			m.statusMessage = fmt.Sprintf("Cell %d finished successfully in %s", idx+1, durationStr)
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
	var statusBar string
	if m.enteringPasswordCellIdx != -1 {
		promptText := fmt.Sprintf(" Sudo Password: %s", m.passwordInput.View())
		statusBar = sudoConfirmStatusStyle.Width(m.terminalWidth).Render(promptText)
	} else if m.confirmingSudoCellIndex != -1 {
		statusBar = sudoConfirmStatusStyle.Width(m.terminalWidth).Render(statusText)
	} else if m.searching {
		promptText := fmt.Sprintf(" %s", m.searchInput.View())
		statusBar = statusBarStyle.Width(m.terminalWidth).Render(promptText)
	} else {
		statusBar = statusBarStyle.Width(m.terminalWidth).Render(statusText)
	}

	// Help Line
	helpText := " [j/k/↑/↓] Navigate • [/] Search • [n/N] Next/Prev • [Enter/r] Run • [s/ctrl+s] Save • [q] Quit"
	if m.searching {
		helpText = " [Enter] Search • [Esc] Cancel"
	} else if m.enteringPasswordCellIdx != -1 {
		helpText = " [Enter] Submit • [Esc] Cancel"
	} else if m.confirmingSudoCellIndex != -1 {
		helpText = " [y] Run command with sudo • [n/esc] Cancel • [q] Quit"
	}
	helpBar := lipgloss.NewStyle().
		Foreground(subtleColor).
		Width(m.terminalWidth).
		Render(helpText)

	return header + "\n" + viewportContent + "\n" + statusBar + "\n" + helpBar
}

func renderCodeCell(cell Cell, isActive bool, width int, isExecuting bool) string {
	var durationInfo string
	if cell.Metadata != nil {
		if startVal, ok := cell.Metadata["start_time"]; ok {
			if endVal, ok := cell.Metadata["end_time"]; ok {
				if startStr, ok := startVal.(string); ok {
					if endStr, ok := endVal.(string); ok {
						if startTime, err := time.Parse(time.RFC3339Nano, startStr); err == nil {
							if endTime, err := time.Parse(time.RFC3339Nano, endStr); err == nil {
								durationInfo = " (" + formatDuration(endTime.Sub(startTime)) + ")"
							}
						}
					}
				}
			}
		}
	}

	// The prompt: e.g. In [1] (bash): or In [*] (bash):
	lang := "bash"
	if cell.Metadata != nil {
		if l, ok := cell.Metadata["language"].(string); ok && l != "" {
			lang = l
		}
	}

	var prompt string
	if isExecuting {
		prompt = fmt.Sprintf("In [*] (%s): ", lang)
	} else if cell.ExecutionCount == nil {
		prompt = fmt.Sprintf("In [ ] (%s): ", lang)
	} else {
		prompt = fmt.Sprintf("In [%d] (%s)%s: ", *cell.ExecutionCount, lang, durationInfo)
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

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

var sudoRegex = regexp.MustCompile(`\bsudo\b`)

func hasSudo(code string) bool {
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if sudoRegex.MatchString(trimmed) {
			return true
		}
	}
	return false
}

func cellMatches(cell Cell, query string) bool {
	if query == "" {
		return false
	}
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(cell.Source.String()), q) {
		return true
	}
	for _, out := range cell.Outputs {
		if out.OutputType == "stream" {
			if strings.Contains(strings.ToLower(out.Text.String()), q) {
				return true
			}
		} else if out.OutputType == "error" {
			if strings.Contains(strings.ToLower(out.EName), q) ||
				strings.Contains(strings.ToLower(out.EValue), q) {
				return true
			}
			for _, tb := range out.Traceback {
				if strings.Contains(strings.ToLower(tb), q) {
					return true
				}
			}
		}
	}
	return false
}

func (m *TuiModel) getMatchInfo() (currMatch int, totalMatches int) {
	if m.searchQuery == "" {
		return 0, 0
	}
	for i, cell := range m.notebook.Cells {
		if cellMatches(cell, m.searchQuery) {
			totalMatches++
			if i == m.activeCellIndex {
				currMatch = totalMatches
			}
		}
	}
	return currMatch, totalMatches
}

func (m *TuiModel) searchForward(startIdx int) bool {
	if m.searchQuery == "" {
		return false
	}
	nCells := len(m.notebook.Cells)
	if nCells == 0 {
		return false
	}

	// Normalize startIdx to be within bounds
	startIdx = (startIdx%nCells + nCells) % nCells

	for i := 0; i < nCells; i++ {
		idx := (startIdx + i) % nCells
		if cellMatches(m.notebook.Cells[idx], m.searchQuery) {
			m.activeCellIndex = idx
			curr, total := m.getMatchInfo()
			m.statusMessage = fmt.Sprintf("Found match in cell %d (match %d/%d): %q", idx+1, curr, total, m.searchQuery)
			return true
		}
	}
	m.statusMessage = fmt.Sprintf("Pattern not found: %s", m.searchQuery)
	return false
}

func (m *TuiModel) searchBackward(startIdx int) bool {
	if m.searchQuery == "" {
		return false
	}
	nCells := len(m.notebook.Cells)
	if nCells == 0 {
		return false
	}

	// Normalize startIdx to be within bounds
	startIdx = (startIdx%nCells + nCells) % nCells

	for i := 0; i < nCells; i++ {
		idx := (startIdx - i + nCells) % nCells
		if cellMatches(m.notebook.Cells[idx], m.searchQuery) {
			m.activeCellIndex = idx
			curr, total := m.getMatchInfo()
			m.statusMessage = fmt.Sprintf("Found match in cell %d (match %d/%d): %q", idx+1, curr, total, m.searchQuery)
			return true
		}
	}
	m.statusMessage = fmt.Sprintf("Pattern not found: %s", m.searchQuery)
	return false
}
