// Package memory provides memory management operations for storing learnings.
package memory

import "fmt"

// LearnOpts holds options for learning storage.
type LearnOpts struct {
	Message    string
	Project    string
	MemoryRoot string
}

// Learn stores a learning in the memory index.
func Learn(opts LearnOpts) error {
	return fmt.Errorf("not implemented")
}
