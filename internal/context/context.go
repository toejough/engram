// Package context provides session continuity for LLM agents (UC-14).
// It reads transcript deltas, strips noisy content, summarizes via Haiku,
// and persists session context files. All I/O is injected via DI interfaces.
package context

import (
	"context"
	"time"
)

// DirCreator creates directories. Wire os.MkdirAll in production.
type DirCreator interface {
	MkdirAll(path string) error
}

// FileReader reads file contents by path. Wire os.ReadFile in production.
type FileReader interface {
	Read(path string) ([]byte, error)
}

// FileWriter writes file contents by path. Wire os.WriteFile in production.
type FileWriter interface {
	Write(path string, content []byte) error
}

// HaikuClient calls the Haiku API to summarize context.
// Wire a real HTTP-based client in production.
type HaikuClient interface {
	Summarize(ctx context.Context, previousSummary, delta string) (string, error)
}

// Renamer renames a file atomically. Wire os.Rename in production.
type Renamer interface {
	Rename(oldpath, newpath string) error
}

// Timestamper provides the current time. Wire a real clock in production.
type Timestamper interface {
	Now() time.Time
}
