package registry

import "errors"

// Outcome represents the result of evaluating an instruction.
type Outcome string

// Outcome values.
const (
	Followed     Outcome = "followed"
	Contradicted Outcome = "contradicted"
	Ignored      Outcome = "ignored"
)

// Sentinel errors.
var (
	ErrNotFound      = errors.New("instruction not found")
	ErrDuplicateID   = errors.New("instruction ID already exists")
	ErrMergeNotFound = errors.New("merge source or target not found")
)

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
