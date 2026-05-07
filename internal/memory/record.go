package memory

import (
	"fmt"
	"os"
	"time"
)

// ContentFields holds type-specific content for a memory record.
// Feedback memories use Behavior/Impact/Action; fact memories use Subject/Predicate/Object.
type ContentFields struct {
	Behavior  string `json:"behavior,omitempty"  toml:"behavior,omitempty"`
	Impact    string `json:"impact,omitempty"    toml:"impact,omitempty"`
	Action    string `json:"action,omitempty"    toml:"action,omitempty"`
	Subject   string `json:"subject,omitempty"   toml:"subject,omitempty"`
	Predicate string `json:"predicate,omitempty" toml:"predicate,omitempty"`
	Object    string `json:"object,omitempty"    toml:"object,omitempty"`
}

// MemoryRecord is the canonical struct for reading and writing memory TOML files.
//
// ALL code that touches memory TOML must use this struct to prevent field loss.
// See #353 for the bug caused by divergent struct definitions.
//
//nolint:revive // "memory.MemoryRecord" stutter is intentional for clarity. See #353.
type MemoryRecord struct {
	SchemaVersion int    `json:"schemaVersion,omitempty" toml:"schema_version,omitempty"`
	Type          string `json:"type"                    toml:"type"`
	Source        string `json:"source"                  toml:"source"`
	Situation     string `json:"situation"               toml:"situation"`

	Content ContentFields `json:"content" toml:"content"`

	CreatedAt string `json:"createdAt" toml:"created_at"`
	UpdatedAt string `json:"updatedAt" toml:"updated_at"`
}

// TargetDir returns the data-dir-relative directory where this record
// should be written, dispatched on r.Type. Unknown types fall back to the
// legacy memories/ directory.
func (r *MemoryRecord) TargetDir(dataDir string) string {
	switch r.Type {
	case "fact":
		return FactsDir(dataDir)
	case "feedback":
		return FeedbackDir(dataDir)
	default:
		return MemoriesDir(dataDir)
	}
}

// ToStored converts a MemoryRecord to a Stored for in-memory use.
func (r *MemoryRecord) ToStored(filePath string) *Stored {
	updatedAt, parseErr := time.Parse(time.RFC3339, r.UpdatedAt)
	if parseErr != nil && r.UpdatedAt != "" {
		fmt.Fprintf(
			os.Stderr,
			"engram: memory: parsing updated_at %q for %s: %v\n",
			r.UpdatedAt,
			filePath,
			parseErr,
		)
	}

	return &Stored{
		Type:      r.Type,
		Situation: r.Situation,
		Source:    r.Source,
		Content:   r.Content,
		UpdatedAt: updatedAt,
		FilePath:  filePath,
	}
}
