package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// FileAckWaiter waits for ACK/WAIT responses from all named recipients.
// All I/O is injected — no os.* calls in this package.
type FileAckWaiter struct {
	FilePath string
	Watch    func(ctx context.Context, agent string, cursor int, msgTypes []string) (Message, int, error)
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
//     c. Call Watch(ctx, callerAgent, cursor, ["ack","wait"]) for next message
//     d. WAIT found → return immediately; ACK found → mark recipient responded
func (w *FileAckWaiter) AckWait(
	ctx context.Context, callerAgent string, cursor int, recipients []string,
) (AckResult, error) {
	maxWait := w.MaxWait
	if maxWait == 0 {
		maxWait = defaultMaxWait
	}

	data, err := w.ReadFile(w.FilePath)
	if err != nil {
		return AckResult{}, fmt.Errorf("reading chat file: %w", err)
	}

	states := buildRecipientStates(data, recipients, w.NowFunc())
	currentCursor := cursor

	for {
		ctxErr := ctx.Err()
		if ctxErr != nil {
			return AckResult{}, fmt.Errorf("ack wait cancelled: %w", ctxErr)
		}

		nowCheck := w.NowFunc()
		applyOfflineImplicit(states, recipients, nowCheck)

		if result, timedOut := checkOnlineSilentTimeout(states, recipients, nowCheck, maxWait, currentCursor); timedOut {
			return result, nil
		}

		if allResponded(states) {
			return AckResult{Result: "ACK", NewCursor: currentCursor}, nil
		}

		msg, newCursor, watchErr := w.Watch(ctx, callerAgent, currentCursor, []string{"ack", "wait"})
		if watchErr != nil {
			ctxErr2 := ctx.Err()
			if ctxErr2 != nil {
				return AckResult{}, fmt.Errorf("ack wait cancelled: %w", ctxErr2)
			}

			// Watch timed out internally — loop back to re-check offline/online timeouts.
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
func (w *FileAckWaiter) applyMsg(msg Message, states map[string]*recipientState, currentCursor int) (AckResult, bool) {
	state, isPending := states[msg.From]
	if !isPending || state == nil || state.responded {
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
func applyOfflineImplicit(states map[string]*recipientState, recipients []string, now time.Time) {
	for _, r := range recipients {
		state := states[r]
		if state == nil || state.responded || state.online {
			continue
		}

		if now.Sub(state.waitStart) >= offlineImplicitWait {
			state.responded = true
		}
	}
}

// buildRecipientStates initialises per-recipient state from the full chat file data.
func buildRecipientStates(data []byte, recipients []string, now time.Time) map[string]*recipientState {
	fifteenMinAgo := now.Add(-15 * time.Minute)
	states := make(map[string]*recipientState, len(recipients))

	for _, r := range recipients {
		states[r] = &recipientState{
			online:    isOnline(data, r, fifteenMinAgo),
			waitStart: now,
		}
	}

	return states
}

// checkOnlineSilentTimeout returns TIMEOUT for the first online+silent recipient that has
// exceeded maxWait. Returns (zero, false) if no timeout has occurred.
func checkOnlineSilentTimeout(
	states map[string]*recipientState,
	recipients []string,
	now time.Time,
	maxWait time.Duration,
	currentCursor int,
) (AckResult, bool) {
	for _, r := range recipients {
		state := states[r]
		if state == nil || state.responded || !state.online {
			continue
		}

		if now.Sub(state.waitStart) >= maxWait {
			return AckResult{
				Result:    "TIMEOUT",
				NewCursor: currentCursor,
				Timeout:   &TimeoutResult{Recipient: r},
			}, true
		}
	}

	return AckResult{}, false
}

// isOnline returns true if the recipient posted any message within the cutoff window.
// This is a full-file scan by design — presence detection is the one exception to the cursor rule.
func isOnline(data []byte, recipient string, cutoff time.Time) bool {
	var parsed struct {
		Messages []Message `toml:"message"`
	}

	err := toml.Unmarshal(data, &parsed)
	if err != nil {
		return false
	}

	for _, msg := range parsed.Messages {
		if strings.EqualFold(msg.From, recipient) && msg.TS.After(cutoff) {
			return true
		}
	}

	return false
}
