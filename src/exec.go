package main

import (
	"bytes"
	"os/exec"
	"time"
)

// RunCodeCell executes the given code snippet in a bash shell and returns
// the corresponding outputs, start time, end time, and error.
func RunCodeCell(code string) ([]Output, time.Time, time.Time, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	// We run the command inside bash to support piping, env variables, etc.
	cmd := exec.Command("bash", "-c", code)
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
