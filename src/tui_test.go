package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHasSudo(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{
			name: "Simple sudo command",
			code: "sudo apt-get update",
			want: true,
		},
		{
			name: "Sudo in pipeline",
			code: "echo 'hello' | sudo tee /etc/test",
			want: true,
		},
		{
			name: "Commented sudo should be ignored",
			code: "# sudo rm -rf /",
			want: false,
		},
		{
			name: "Sudo on second line, first is comment",
			code: "# Run with root\nsudo systemctl restart docker",
			want: true,
		},
		{
			name: "Sudo as substring should be ignored",
			code: "echo pseudocode\nvarsudo=1\nsudo_param=true",
			want: false,
		},
		{
			name: "Sudo inside quotes",
			code: "echo \"sudo command\"",
			want: true,
		},
		{
			name: "Sudo command ending with semicolon",
			code: "sudo apt-get update; echo done",
			want: true,
		},
		{
			name: "Sudo inside shell operators",
			code: "sudo(apt-get)",
			want: true,
		},
		{
			name: "No sudo at all",
			code: "ls -la\necho 'hello world'",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasSudo(tt.code)
			if got != tt.want {
				t.Errorf("hasSudo() = %v, want %v. Code:\n%s", got, tt.want, tt.code)
			}
		})
	}
}

func TestCellMatches(t *testing.T) {
	cellSourceOnly := Cell{
		CellType: "markdown",
		Source:   StringOrArray{"This is a markdown cell containing apple."},
	}
	cellWithOutputs := Cell{
		CellType: "code",
		Source:   StringOrArray{"echo 'orange'"},
		Outputs: []Output{
			{
				OutputType: "stream",
				Name:       "stdout",
				Text:       StringOrArray{"Found orange in output."},
			},
		},
	}
	cellWithError := Cell{
		CellType: "code",
		Source:   StringOrArray{"cat non_existent_file"},
		Outputs: []Output{
			{
				OutputType: "error",
				EName:      "FileNotFoundError",
				EValue:     "No such file or directory",
				Traceback:  []string{"Error occurred in line 5", "stack trace"},
			},
		},
	}

	tests := []struct {
		name  string
		cell  Cell
		query string
		want  bool
	}{
		{"Matches source case-sensitive (ignored as case-insensitive is used)", cellSourceOnly, "apple", true},
		{"Matches source case-insensitive", cellSourceOnly, "APPLE", true},
		{"Does not match source", cellSourceOnly, "banana", false},
		{"Matches output stream", cellWithOutputs, "orange", true},
		{"Matches output error EName", cellWithError, "FileNotFound", true},
		{"Matches output error EValue", cellWithError, "No such file", true},
		{"Matches output traceback", cellWithError, "line 5", true},
		{"Empty query doesn't match", cellSourceOnly, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cellMatches(tt.cell, tt.query)
			if got != tt.want {
				t.Errorf("cellMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchForwardAndBackward(t *testing.T) {
	nb := &Notebook{
		Cells: []Cell{
			{
				CellType: "markdown",
				Source:   StringOrArray{"First cell with apple"},
			},
			{
				CellType: "code",
				Source:   StringOrArray{"Second cell with banana"},
			},
			{
				CellType: "code",
				Source:   StringOrArray{"Third cell with apple"},
			},
		},
	}
	m := NewTuiModel(nb, "dummy.shbn")

	// Search apple
	m.searchQuery = "apple"

	// 1. Search forward starting at 0: should find cell 0
	found := m.searchForward(0)
	if !found || m.activeCellIndex != 0 {
		t.Errorf("Expected to find apple at cell 0, got found=%v, index=%d", found, m.activeCellIndex)
	}
	expectedStatus := `Found match in cell 1 (match 1/2): "apple"`
	if m.statusMessage != expectedStatus {
		t.Errorf("Expected status message %q, got %q", expectedStatus, m.statusMessage)
	}

	// 2. Search forward starting at 1: should find cell 2
	found = m.searchForward(1)
	if !found || m.activeCellIndex != 2 {
		t.Errorf("Expected to find apple at cell 2, got found=%v, index=%d", found, m.activeCellIndex)
	}
	expectedStatus = `Found match in cell 3 (match 2/2): "apple"`
	if m.statusMessage != expectedStatus {
		t.Errorf("Expected status message %q, got %q", expectedStatus, m.statusMessage)
	}

	// 3. Search forward starting at 3 (wrap-around): should find cell 0
	found = m.searchForward(3)
	if !found || m.activeCellIndex != 0 {
		t.Errorf("Expected to find apple at cell 0 (wrapped), got found=%v, index=%d", found, m.activeCellIndex)
	}
	expectedStatus = `Found match in cell 1 (match 1/2): "apple"`
	if m.statusMessage != expectedStatus {
		t.Errorf("Expected status message %q, got %q", expectedStatus, m.statusMessage)
	}

	// 4. Search backward starting at 1: should find cell 0
	found = m.searchBackward(1)
	if !found || m.activeCellIndex != 0 {
		t.Errorf("Expected to find apple at cell 0 searching backward, got found=%v, index=%d", found, m.activeCellIndex)
	}
	expectedStatus = `Found match in cell 1 (match 1/2): "apple"`
	if m.statusMessage != expectedStatus {
		t.Errorf("Expected status message %q, got %q", expectedStatus, m.statusMessage)
	}

	// 5. Search backward starting at 2: should find cell 2
	found = m.searchBackward(2)
	if !found || m.activeCellIndex != 2 {
		t.Errorf("Expected to find apple at cell 2 searching backward, got found=%v, index=%d", found, m.activeCellIndex)
	}
	expectedStatus = `Found match in cell 3 (match 2/2): "apple"`
	if m.statusMessage != expectedStatus {
		t.Errorf("Expected status message %q, got %q", expectedStatus, m.statusMessage)
	}

	// 6. Search backward starting at -1 (wrap-around): should find cell 2
	found = m.searchBackward(-1)
	if !found || m.activeCellIndex != 2 {
		t.Errorf("Expected to find apple at cell 2 (wrapped backward), got found=%v, index=%d", found, m.activeCellIndex)
	}
	expectedStatus = `Found match in cell 3 (match 2/2): "apple"`
	if m.statusMessage != expectedStatus {
		t.Errorf("Expected status message %q, got %q", expectedStatus, m.statusMessage)
	}

	// 7. Non-existent pattern
	m.searchQuery = "cherry"
	found = m.searchForward(0)
	if found {
		t.Errorf("Expected not to find cherry")
	}
}

func TestTuiSearchKeys(t *testing.T) {
	nb := &Notebook{
		Cells: []Cell{
			{
				CellType: "markdown",
				Source:   StringOrArray{"First cell with apple"},
			},
			{
				CellType: "code",
				Source:   StringOrArray{"Second cell with banana"},
			},
		},
	}
	m := NewTuiModel(nb, "dummy.shbn")

	// Trigger search mode
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	mVal := model.(TuiModel)
	m = &mVal
	if !m.searching {
		t.Error("Expected searching to be true after '/' key")
	}
	_ = cmd // ignore command

	// Type query
	for _, r := range "banana" {
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		mVal = model.(TuiModel)
		m = &mVal
	}

	if m.searchInput.Value() != "banana" {
		t.Errorf("Expected search input value to be 'banana', got %q", m.searchInput.Value())
	}

	// Press Enter to confirm search
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mVal = model.(TuiModel)
	m = &mVal

	if m.searching {
		t.Error("Expected searching to be false after Enter")
	}
	if m.activeCellIndex != 1 {
		t.Errorf("Expected active cell to be 1 (banana), got %d", m.activeCellIndex)
	}
	if m.searchQuery != "banana" {
		t.Errorf("Expected searchQuery to be 'banana', got %q", m.searchQuery)
	}

	// Search non-existent using "n"
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	mVal = model.(TuiModel)
	m = &mVal
	for _, r := range "cherry" {
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		mVal = model.(TuiModel)
		m = &mVal
	}
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mVal = model.(TuiModel)
	m = &mVal
	if m.activeCellIndex != 1 {
		t.Errorf("Expected active cell to stay 1 on mismatch, got %d", m.activeCellIndex)
	}
	if !strings.Contains(m.statusMessage, "Pattern not found") {
		t.Errorf("Expected status message to contain pattern not found, got %q", m.statusMessage)
	}

	// Pressing n/N without query
	m.searchQuery = ""
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	mVal = model.(TuiModel)
	m = &mVal
	if m.statusMessage != "No search pattern" {
		t.Errorf("Expected status 'No search pattern', got %q", m.statusMessage)
	}
}

func TestTuiCommandKeys(t *testing.T) {
	nb := &Notebook{
		Cells: []Cell{
			{
				CellType: "markdown",
				Source:   StringOrArray{"First cell with apple"},
			},
		},
	}
	m := NewTuiModel(nb, "dummy.shbn")

	// 1. Trigger command mode
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	mVal := model.(TuiModel)
	m = &mVal
	if !m.enteringCommand {
		t.Error("Expected enteringCommand to be true after '!' key")
	}
	_ = cmd

	// 2. Type command "ls"
	for _, r := range "ls" {
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		mVal = model.(TuiModel)
		m = &mVal
	}

	if m.commandInput.Value() != "ls" {
		t.Errorf("Expected command input value to be 'ls', got %q", m.commandInput.Value())
	}

	// 3. Press Enter to submit the command
	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mVal = model.(TuiModel)
	m = &mVal

	if m.enteringCommand {
		t.Error("Expected enteringCommand to be false after Enter")
	}
	if cmd == nil {
		t.Error("Expected a command to be returned on Enter")
	}

	// 4. Test MsgCommandFinished with no error
	model, cmd = m.Update(MsgCommandFinished{Err: nil})
	mVal = model.(TuiModel)
	m = &mVal
	if m.statusMessage != "Command finished successfully" {
		t.Errorf("Expected status message to be 'Command finished successfully', got %q", m.statusMessage)
	}
	if cmd == nil {
		t.Error("Expected tea.ClearScreen to be returned after MsgCommandFinished")
	}

	// 5. Test MsgCommandFinished with error
	model, cmd = m.Update(MsgCommandFinished{Err: fmt.Errorf("some error")})
	mVal = model.(TuiModel)
	m = &mVal
	if m.statusMessage != "Command failed: some error" {
		t.Errorf("Expected status message to be 'Command failed: some error', got %q", m.statusMessage)
	}

	// 6. Test cancellation with Esc
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	mVal = model.(TuiModel)
	m = &mVal
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	mVal = model.(TuiModel)
	m = &mVal
	if m.enteringCommand {
		t.Error("Expected enteringCommand to be false after Esc")
	}
	if m.statusMessage != "Command execution cancelled." {
		t.Errorf("Expected status 'Command execution cancelled.', got %q", m.statusMessage)
	}
}

func TestShortenPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Skipping TestShortenPath: UserHomeDir not available")
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "Home directory exactly",
			path: home,
			want: "~",
		},
		{
			name: "Subdirectory of home",
			path: home + "/git/runbook",
			want: "~/git/runbook",
		},
		{
			name: "Path outside home",
			path: "/tmp/runbook-test",
			want: "/tmp/runbook-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortenPath(tt.path)
			if got != tt.want {
				t.Errorf("shortenPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}


