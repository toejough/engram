// Package result defines the structured result format for skill outputs.
package result

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Exported constants.
const (
	ContextDir = "context"
)

// CollectedResults holds the merged results from multiple tasks.
type CollectedResults struct {
	Succeeded     int
	Failed        int
	Total         int
	FailedTasks   []string
	Learnings     []Learning
	FilesModified []string
}

// Decision captures a choice made during skill execution.
type Decision struct {
	Context      string   `toml:"context"`
	Choice       string   `toml:"choice"`
	Reason       string   `toml:"reason"`
	Alternatives []string `toml:"alternatives,omitempty"`
}

// FileSystem provides file system operations for result collection.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
}

// Learning captures something learned during skill execution.
type Learning struct {
	Content string `toml:"content"`
}

// Outputs describes what the skill produced.
type Outputs struct {
	FilesModified []string `toml:"files_modified"`
}

// RealFS implements FileSystem using the real file system.
type RealFS struct{}

// ReadFile reads a file using os.ReadFile.
func (RealFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Result is the complete skill result.
type Result struct {
	Status    Status     `toml:"status"`
	Outputs   Outputs    `toml:"outputs"`
	Decisions []Decision `toml:"decisions,omitempty"`
	Learnings []Learning `toml:"learnings,omitempty"`
}

// Status indicates whether the skill completed successfully.
type Status struct {
	Success bool `toml:"success"`
}

// Collect reads and merges result files for multiple tasks.
func Collect(dir string, tasks []string, skill string, fs FileSystem) (CollectedResults, error) {
	collected := CollectedResults{
		Total: len(tasks),
	}

	for _, task := range tasks {
		resultPath := filepath.Join(dir, ContextDir, fmt.Sprintf("%s-%s.result.toml", task, skill))

		data, err := fs.ReadFile(resultPath)
		if err != nil {
			// Missing result file = failed
			collected.Failed++
			collected.FailedTasks = append(collected.FailedTasks, task)

			continue
		}

		r, err := Parse(data)
		if err != nil {
			// Invalid result file = failed
			collected.Failed++
			collected.FailedTasks = append(collected.FailedTasks, task)

			continue
		}

		if !r.Status.Success {
			collected.Failed++
			collected.FailedTasks = append(collected.FailedTasks, task)
		} else {
			collected.Succeeded++
		}

		// Merge learnings and files
		collected.Learnings = append(collected.Learnings, r.Learnings...)
		collected.FilesModified = append(collected.FilesModified, r.Outputs.FilesModified...)
	}

	return collected, nil
}

// Marshal converts a Result to TOML bytes.
func Marshal(r Result) ([]byte, error) {
	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to encode TOML: %w", err)
	}

	return buf.Bytes(), nil
}

// Parse parses a TOML result file.
func Parse(data []byte) (Result, error) {
	var raw rawResult

	err := toml.Unmarshal(data, &raw)
	if err != nil {
		return Result{}, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Validate required sections
	if raw.Status == nil {
		return Result{}, errors.New("missing required section: status")
	}

	if raw.Outputs == nil {
		return Result{}, errors.New("missing required section: outputs")
	}

	// Validate decisions
	for i, d := range raw.Decisions {
		if d.Context == "" {
			return Result{}, fmt.Errorf("decision[%d]: missing required field: context", i)
		}

		if d.Choice == "" {
			return Result{}, fmt.Errorf("decision[%d]: missing required field: choice", i)
		}

		if d.Reason == "" {
			return Result{}, fmt.Errorf("decision[%d]: missing required field: reason", i)
		}
	}

	// Validate learnings
	for i, l := range raw.Learnings {
		if l.Content == "" {
			return Result{}, fmt.Errorf("learning[%d]: missing required field: content", i)
		}
	}

	return Result{
		Status:    *raw.Status,
		Outputs:   *raw.Outputs,
		Decisions: raw.Decisions,
		Learnings: raw.Learnings,
	}, nil
}

// rawResult is used for detecting missing sections during parsing.
type rawResult struct {
	Status    *Status    `toml:"status"`
	Outputs   *Outputs   `toml:"outputs"`
	Decisions []Decision `toml:"decisions"`
	Learnings []Learning `toml:"learnings"`
}
