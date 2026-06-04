package main

import (
	"bytes"
	"os/exec"
)

// RunCodeCell executes the given code snippet in a bash shell and returns
// the corresponding outputs in the Jupyter notebook format.
func RunCodeCell(code string) ([]Output, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	// We run the command inside bash to support piping, env variables, etc.
	cmd := exec.Command("bash", "-c", code)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

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

	return outputs, err
}
