// Package chat provides pure domain types and interfaces for the engram chat protocol.
// No os.* calls. All I/O is injected.
package chat

import (
	"context"
	"time"
)

// LockFile creates an exclusive lock file compatible with bash shlock convention.
// Returns an unlock function to release the lock.
// Implemented via os.OpenFile(O_CREATE|O_EXCL) at the CLI wiring layer.
type LockFile func(name string) (unlock func() error, err error)

// Message is a single chat protocol message.
type Message struct {
	From   string    `toml:"from"`
	To     string    `toml:"to"`
	Thread string    `toml:"thread"`
	Type   string    `toml:"type"`
	TS     time.Time `toml:"ts"`
	Text   string    `toml:"text"`
}

// Poster appends messages to the chat file atomically.
type Poster interface {
	Post(msg Message) (newCursor int, err error)
}

// Watcher blocks until a matching message arrives after cursor.
// msgTypes filters by message type; empty slice matches all types.
// agent matches messages where the To field contains agent or "all".
type Watcher interface {
	Watch(ctx context.Context, agent string, cursor int, msgTypes []string) (Message, int, error)
}
