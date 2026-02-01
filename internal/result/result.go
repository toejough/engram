// Package result defines the structured result format for skill outputs.
package result

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

// Parse parses a TOML result file.
func Parse(data []byte) (Result, error) {
	// TODO: Implement
	return Result{}, nil
}

// Marshal converts a Result to TOML bytes.
func Marshal(r Result) ([]byte, error) {
	// TODO: Implement
	return nil, nil
}
