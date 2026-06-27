package main

import (
	"bytes"
	"os/exec"
	"strings"
	"time"
)

// RunCodeCell executes the given code snippet in a bash shell or PowerShell and returns
// the corresponding outputs, start time, end time, and error.
func RunCodeCell(code string, language string, password string) ([]Output, time.Time, time.Time, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	var cmd *exec.Cmd
	if language == "powershell" || language == "pwsh" {
		exe := "pwsh"
		if _, err := exec.LookPath("pwsh"); err != nil {
			if _, err2 := exec.LookPath("powershell"); err2 == nil {
				exe = "powershell"
			}
		}
		cmd = exec.Command(exe, "-NoProfile", "-NonInteractive", "-Command", code)
	} else {
		// Default to bash
		if password != "" {
			wrappedCode := `sudo() { command sudo -S "$@"; }; ` + code
			cmd = exec.Command("bash", "-c", wrappedCode)
			stdinPipe, err := cmd.StdinPipe()
			if err == nil {
				go func() {
					defer stdinPipe.Close()
					stdinPipe.Write([]byte(password + "\n"))
				}()
			}
		} else {
			cmd = exec.Command("bash", "-c", code)
		}
	}

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	startTime := time.Now()
	err := cmd.Run()
	endTime := time.Now()

	var outputs []Output

	// Capture stdout stream
	stdoutStr := stdoutBuf.String()
	if len(stdoutStr) > 0 {
		outputs = append(outputs, Output{
			OutputType: "stream",
			Name:       "stdout",
			Text:       StringOrArray{stdoutStr},
		})
	}

	// Capture stderr stream
	stderrStr := stderrBuf.String()
	if password != "" {
		stderrStr = cleanStderr(stderrStr)
	}
	if len(stderrStr) > 0 {
		outputs = append(outputs, Output{
			OutputType: "stream",
			Name:       "stderr",
			Text:       StringOrArray{stderrStr},
		})
	}

	// If there's an error, add an error output cell
	if err != nil {
		outputs = append(outputs, Output{
			OutputType: "error",
			EName:       "ExitError",
			EValue:      err.Error(),
			Traceback:   []string{},
		})
	}

	return outputs, startTime, endTime, err
}

func cleanStderr(stderr string) string {
	lines := strings.Split(stderr, "\n")
	var clean []string
	for _, line := range lines {
		if strings.Contains(line, "[sudo] password for") {
			continue
		}
		clean = append(clean, line)
	}
	return strings.Join(clean, "\n")
}
