// Package result defines the structured result format for skill outputs.
package result

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
)

// Status indicates whether the skill completed successfully.
type Status struct {
	Success bool `toml:"success"`
}

// Outputs describes what the skill produced.
type Outputs struct {
	FilesModified []string `toml:"files_modified"`
}

// Decision captures a choice made during skill execution.
type Decision struct {
	Context      string   `toml:"context"`
	Choice       string   `toml:"choice"`
	Reason       string   `toml:"reason"`
	Alternatives []string `toml:"alternatives,omitempty"`
}

// Learning captures something learned during skill execution.
type Learning struct {
	Content string `toml:"content"`
}

// Result is the complete skill result.
type Result struct {
	Status    Status     `toml:"status"`
	Outputs   Outputs    `toml:"outputs"`
	Decisions []Decision `toml:"decisions,omitempty"`
	Learnings []Learning `toml:"learnings,omitempty"`
}

// rawResult is used for detecting missing sections during parsing.
type rawResult struct {
	Status    *Status    `toml:"status"`
	Outputs   *Outputs   `toml:"outputs"`
	Decisions []Decision `toml:"decisions"`
	Learnings []Learning `toml:"learnings"`
}

// Parse parses a TOML result file.
func Parse(data []byte) (Result, error) {
	var raw rawResult
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Result{}, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Validate required sections
	if raw.Status == nil {
		return Result{}, fmt.Errorf("missing required section: status")
	}
	if raw.Outputs == nil {
		return Result{}, fmt.Errorf("missing required section: outputs")
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

// Marshal converts a Result to TOML bytes.
func Marshal(r Result) ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(r); err != nil {
		return nil, fmt.Errorf("failed to encode TOML: %w", err)
	}
	return buf.Bytes(), nil
}
