package main

import (
	"encoding/json"
	"os"
	"strings"
)

// StringOrArray represents a field that can be either a single string
// or an array of strings in the JSON source.
type StringOrArray []string

// UnmarshalJSON implements json.Unmarshaler.
func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*s = []string{str}
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*s = arr
	return nil
}

// MarshalJSON implements json.Marshaler.
func (s StringOrArray) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(s))
}

// String returns the combined string.
func (s StringOrArray) String() string {
	return strings.Join(s, "")
}

// Output represents a cell execution output.
type Output struct {
	OutputType string `json:"output_type"`

	// For stream outputs (stdout/stderr)
	Name string        `json:"name,omitempty"`
	Text StringOrArray `json:"text,omitempty"`

	// For execute_result / display_data
	Data           map[string]interface{} `json:"data,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ExecutionCount *int                   `json:"execution_count,omitempty"`

	// For error outputs
	EName     string   `json:"ename,omitempty"`
	EValue    string   `json:"evalue,omitempty"`
	Traceback []string `json:"traceback,omitempty"`
}

// Cell represents a single notebook cell.
type Cell struct {
	CellType       string                 `json:"cell_type"`
	ExecutionCount *int                   `json:"execution_count"`
	ID             string                 `json:"id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
	Source         StringOrArray          `json:"source"`
	Outputs        []Output               `json:"outputs,omitempty"`
}

// Notebook represents the full runbook/notebook.
type Notebook struct {
	Cells         []Cell                 `json:"cells"`
	Metadata      map[string]interface{} `json:"metadata"`
	NbFormat      int                    `json:"nbformat"`
	NbFormatMinor int                    `json:"nbformat_minor"`
}

// Normalize ensures metadata and slices are not nil, formatting properly for Jupyter.
func (nb *Notebook) Normalize() {
	if nb.Metadata == nil {
		nb.Metadata = make(map[string]interface{})
	}
	for i := range nb.Cells {
		if nb.Cells[i].Metadata == nil {
			nb.Cells[i].Metadata = make(map[string]interface{})
		}
		if nb.Cells[i].CellType == "code" && nb.Cells[i].Outputs == nil {
			nb.Cells[i].Outputs = make([]Output, 0)
		}
	}
}

// LoadNotebook loads a notebook file from disk.
func LoadNotebook(filePath string) (*Notebook, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var nb Notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return nil, err
	}
	nb.Normalize()
	return &nb, nil
}

// SaveNotebook saves the notebook back to disk.
func SaveNotebook(filePath string, nb *Notebook) error {
	nb.Normalize()
	data, err := json.MarshalIndent(nb, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filePath, data, 0644)
}
