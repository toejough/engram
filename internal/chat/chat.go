// Package chat provides pure domain types and interfaces for the engram chat protocol.
// No os.* calls. All I/O is injected.
package chat

import (
	"context"
	"time"
)

// AckResult is the result of an AckWait call.
// Result is "ACK", "WAIT", or "TIMEOUT".
// Exit code 0 for all three; non-zero only for system errors.
type AckResult struct {
	Result    string         `json:"result"`
	Wait      *WaitResult    `json:"wait,omitempty"`
	Timeout   *TimeoutResult `json:"timeout,omitempty"`
	NewCursor int            `json:"cursor"`
}

// AckWaiter blocks until all recipients respond.
// Invariants: when AckResult.Result=="WAIT", AckResult.Wait is non-nil.
//
//	when AckResult.Result=="TIMEOUT", AckResult.Timeout is non-nil.
//	when AckResult.Result=="ACK", both Wait and Timeout are nil.
type AckWaiter interface {
	AckWait(ctx context.Context, agent string, cursor int, recipients []string) (AckResult, error)
}

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

// TimeoutResult names the online-but-silent recipient.
type TimeoutResult struct {
	Recipient string `json:"recipient"`
}

// WaitResult carries the WAIT message details.
type WaitResult struct {
	From string `json:"from"`
	Text string `json:"text"`
}

// Watcher blocks until a matching message arrives after cursor.
// msgTypes filters by message type; empty slice matches all types.
// agent matches messages where the To field contains agent or "all".
type Watcher interface {
	Watch(ctx context.Context, agent string, cursor int, msgTypes []string) (Message, int, error)
}
