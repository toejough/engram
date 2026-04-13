package chat

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"
)

// FileAckWaiter waits for ACK/WAIT responses from all named recipients.
// All I/O is injected — no os.* calls in this package.
type FileAckWaiter struct {
	FilePath string
	Watcher  Watcher // watches for new ack/wait messages
	ReadFile func(path string) ([]byte, error)
	NowFunc  func() time.Time // injectable for online/offline detection tests
	MaxWait  time.Duration    // default 30s; per-online-silent-recipient timeout
}

// AckWait blocks until all recipients respond or a timeout/error occurs.
// Algorithm:
//  1. Read full chat file → detect online/offline per recipient (full-file scan, not cursor-bounded)
//  2. Build per-recipient state
//  3. Loop:
//     a. Check offline implicit ACK (elapsed ≥ 5s) and online+silent TIMEOUT (elapsed ≥ MaxWait)
//     b. All responded → return ACK
//     c. Call Watcher.Watch(ctx, callerAgent, cursor, ["ack","wait"]) for next message
//     d. WAIT found → return immediately; ACK found → mark recipient responded
func (w *FileAckWaiter) AckWait( //nolint:funlen // atomic algorithm: extracting helpers adds coverage tradeoffs
	ctx context.Context, callerAgent string, cursor int, recipients []string,
) (AckResult, error) {
	maxWait := w.MaxWait
	if maxWait == 0 {
		maxWait = defaultMaxWait
	}

	// waitStart is the reference point for both per-recipient offline detection and the
	// watch deadline. Captured once so NowFunc call count is unchanged from the original,
	// preserving existing fake-time test behaviour.
	waitStart := w.NowFunc()

	// watchDeadline is the fixed point at which Watch must unblock so AckWait can
	// re-evaluate the online-silent timeout. Fixed so unrelated file-change events
	// (health-checker heartbeats etc.) cannot reset the clock.
	watchDeadline := waitStart.Add(maxWait)

	data, err := readFileOptional(w.ReadFile, w.FilePath)
	if err != nil {
		return AckResult{}, err
	}

	states := buildRecipientStates(data, recipients, waitStart)
	currentCursor := cursor

	for {
		ctxErr := ctx.Err()
		if ctxErr != nil {
			return AckResult{}, fmt.Errorf("ack wait cancelled: %w", ctxErr)
		}

		nowCheck := w.NowFunc()
		applyOfflineImplicit(states, nowCheck)

		if result, timedOut := checkOnlineSilentTimeout(
			states,
			nowCheck,
			maxWait,
			currentCursor,
		); timedOut {
			return result, nil
		}

		if allResponded(states) {
			return AckResult{Result: "ACK", NewCursor: currentCursor}, nil
		}

		watchCtx, watchCancel := context.WithDeadline(ctx, watchDeadline)
		msg, newCursor, watchErr := w.Watcher.Watch(
			watchCtx,
			callerAgent,
			currentCursor,
			[]string{"ack", "wait"},
		)

		watchCancel()

		if watchErr != nil {
			if !errors.Is(watchErr, context.DeadlineExceeded) {
				return AckResult{}, fmt.Errorf("watching for ack: %w", watchErr)
			}
			// Watch's internal deadline exceeded — loop back to re-check offline/online timeouts.
			continue
		}

		currentCursor = newCursor

		if result, done := w.applyMsg(msg, states, currentCursor); done {
			return result, nil
		}
	}
}

// applyMsg processes a received ack/wait message, updating recipient state.
// Returns (result, true) if the wait is resolved (WAIT received), or (zero, false) to continue.
func (w *FileAckWaiter) applyMsg(
	msg Message,
	states map[string]*recipientState,
	currentCursor int,
) (AckResult, bool) {
	state, isPending := states[strings.ToLower(msg.From)]
	if !isPending || state.responded {
		return AckResult{}, false
	}

	if msg.Type == "wait" {
		return AckResult{
			Result:    "WAIT",
			NewCursor: currentCursor,
			Wait:      &WaitResult{From: msg.From, Text: msg.Text},
		}, true
	}

	if msg.Type == "ack" {
		state.responded = true
	}

	return AckResult{}, false
}

// unexported constants.
const (
	defaultMaxWait      = 30 * time.Second
	offlineImplicitWait = 5 * time.Second
)

// recipientState tracks per-recipient ACK progress.
type recipientState struct {
	responded bool
	online    bool      // true if posted any message within last 15 min
	waitStart time.Time // when we started waiting for this recipient
}

// allResponded returns true if every recipient has responded.
func allResponded(states map[string]*recipientState) bool {
	for _, state := range states {
		if !state.responded {
			return false
		}
	}

	return true
}

// applyOfflineImplicit marks offline recipients as responded when offlineImplicitWait has elapsed.
func applyOfflineImplicit(states map[string]*recipientState, now time.Time) {
	for _, state := range states {
		if state.responded || state.online {
			continue
		}

		if now.Sub(state.waitStart) >= offlineImplicitWait {
			state.responded = true
		}
	}
}

// buildRecipientStates initialises per-recipient state from the full chat file data.
// Recipient names are normalized to lowercase so map lookups are case-insensitive.
// ParseMessages is called once and the result shared across all recipients.
func buildRecipientStates(
	data []byte,
	recipients []string,
	now time.Time,
) map[string]*recipientState {
	fifteenMinAgo := now.Add(-15 * time.Minute)
	states := make(map[string]*recipientState, len(recipients))

	messages, _ := ParseMessages(data) // errors treated as empty message list

	for _, r := range recipients {
		states[strings.ToLower(r)] = &recipientState{
			online:    isOnline(messages, r, fifteenMinAgo),
			waitStart: now,
		}
	}

	return states
}

// checkOnlineSilentTimeout returns TIMEOUT for the first online+silent recipient that has
// exceeded maxWait. Returns (zero, false) if no timeout has occurred.
func checkOnlineSilentTimeout(
	states map[string]*recipientState,
	now time.Time,
	maxWait time.Duration,
	currentCursor int,
) (AckResult, bool) {
	for recipient, state := range states {
		if state.responded || !state.online {
			continue
		}

		if now.Sub(state.waitStart) >= maxWait {
			return AckResult{
				Result:    "TIMEOUT",
				NewCursor: currentCursor,
				Timeout:   &TimeoutResult{Recipient: recipient},
			}, true
		}
	}

	return AckResult{}, false
}

// isOnline returns true if the recipient posted any message within the cutoff window.
// This is a full-file scan by design — presence detection is the one exception to the cursor rule.
func isOnline(messages []Message, recipient string, cutoff time.Time) bool {
	for _, msg := range messages {
		if strings.EqualFold(msg.From, recipient) && msg.TS.After(cutoff) {
			return true
		}
	}

	return false
}

// readFileOptional reads path, returning nil data if the file does not exist.
// Callers treat nil data as an empty file (no messages → all recipients offline).
func readFileOptional(readFile func(string) ([]byte, error), path string) ([]byte, error) {
	data, err := readFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading chat file: %w", err)
	}

	return data, nil
}
