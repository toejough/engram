package maintain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Exported constants.
const (
	ActionDelete    = "delete"
	ActionMerge     = "merge"
	ActionRecommend = "recommend"
	ActionUpdate    = "update"
)

// Proposal represents a single maintenance action to be applied to a memory.
type Proposal struct {
	ID        string   `json:"id"`
	Action    string   `json:"action"`
	Target    string   `json:"target"`
	Field     string   `json:"field,omitempty"`
	Value     string   `json:"value,omitempty"`
	Related   []string `json:"related,omitempty"`
	Rationale string   `json:"rationale"`
}

// ReadFileFunc reads a file and returns its contents.
type ReadFileFunc func(string) ([]byte, error)

// WriteFileFunc writes data to a file with the given permissions.
type WriteFileFunc func(string, []byte, os.FileMode) error

// ReadProposals reads proposals from a JSON file at the given path.
// Returns an empty slice if the file does not exist.
func ReadProposals(path string, readFile ReadFileFunc) ([]Proposal, error) {
	data, err := readFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Proposal{}, nil
		}

		return nil, fmt.Errorf("reading proposals: %w", err)
	}

	proposals := make([]Proposal, 0)

	unmarshalErr := json.Unmarshal(data, &proposals)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshalling proposals: %w", unmarshalErr)
	}

	return proposals, nil
}

// WriteProposals writes proposals to a JSON file at the given path.
func WriteProposals(path string, proposals []Proposal, writeFile WriteFileFunc) error {
	const proposalFileMode = 0o644

	data, err := json.MarshalIndent(proposals, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling proposals: %w", err)
	}

	writeErr := writeFile(path, data, proposalFileMode)
	if writeErr != nil {
		return fmt.Errorf("writing proposals: %w", writeErr)
	}

	return nil
}
