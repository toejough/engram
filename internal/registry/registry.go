package registry

import "errors"

// Exported constants.
const (
	Contradicted Outcome = "contradicted"
	Followed     Outcome = "followed"
	Ignored      Outcome = "ignored"
)

// Exported variables.
var (
	ErrDuplicateID   = errors.New("instruction ID already exists")
	ErrMergeNotFound = errors.New("merge source or target not found")
	ErrNotFound      = errors.New("instruction not found")
)

// Outcome represents the result of evaluating an instruction.
type Outcome string

// Registry defines the interface for managing registered instructions.
type Registry interface {
	Register(entry InstructionEntry) error
	RecordSurfacing(id string) error
	RecordEvaluation(id string, outcome Outcome) error
	Merge(sourceID, targetID string) error
	Remove(id string) error
	List() ([]InstructionEntry, error)
	Get(id string) (*InstructionEntry, error)
}
