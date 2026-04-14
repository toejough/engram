package memory

import (
	"fmt"
	"os"
	"time"
)

// ContentFields holds type-specific content for a memory record.
// Feedback memories use Behavior/Impact/Action; fact memories use Subject/Predicate/Object.
type ContentFields struct {
	Behavior  string `toml:"behavior,omitempty"`
	Impact    string `toml:"impact,omitempty"`
	Action    string `toml:"action,omitempty"`
	Subject   string `toml:"subject,omitempty"`
	Predicate string `toml:"predicate,omitempty"`
	Object    string `toml:"object,omitempty"`
}

// MemoryRecord is the canonical struct for reading and writing memory TOML files.
//
// ALL code that touches memory TOML must use this struct to prevent field loss.
// See #353 for the bug caused by divergent struct definitions.
//
//nolint:revive // "memory.MemoryRecord" stutter is intentional for clarity. See #353.
type MemoryRecord struct {
	SchemaVersion int    `toml:"schema_version,omitempty"`
	Type          string `toml:"type"`
	Source        string `toml:"source"`
	Situation     string `toml:"situation"`

	Content ContentFields `toml:"content"`

	CreatedAt string `toml:"created_at"`
	UpdatedAt string `toml:"updated_at"`
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
