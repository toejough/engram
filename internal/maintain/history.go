package maintain

import (
	"engram/internal/memory"
)

// TOMLHistoryRecorder reads memory TOML files. Stubbed during SBIA migration.
type TOMLHistoryRecorder struct{}

// NewTOMLHistoryRecorder creates a stubbed TOMLHistoryRecorder.
func NewTOMLHistoryRecorder(_ ...any) *TOMLHistoryRecorder {
	return &TOMLHistoryRecorder{}
}

// ReadRecord is stubbed — returns nil record.
func (r *TOMLHistoryRecorder) ReadRecord(_ string) (*memory.MemoryRecord, error) {
	return &memory.MemoryRecord{}, nil
}
