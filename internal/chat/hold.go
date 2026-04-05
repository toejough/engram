package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// HoldRecord is the payload of a hold-acquire message.
// Stored as JSON in Message.Text.
// JSON tags use kebab-case to match chat protocol field names.
//
//nolint:tagliatelle // chat protocol uses kebab-case field names: hold-id, acquired-ts
type HoldRecord struct {
	HoldID string `json:"hold-id"`
	Holder string `json:"holder"`
	Target string `json:"target"`
	// Condition values: "done:<agent>", "first-intent:<agent>", "lead-release:<tag>", or empty.
	Condition string `json:"condition,omitempty"`
	// Tag is a workflow label for bulk operations (e.g., "codesign-1", "plan-review-1").
	Tag        string    `json:"tag,omitempty"`
	AcquiredTS time.Time `json:"acquired-ts"`
}

// EvaluateCondition checks if hold condition is met against messages after hold.AcquiredTS.
// Conditions:
//
//	"done:<agent>"         → true when agent posts type="done" after AcquiredTS
//	"first-intent:<agent>" → true when agent posts type="intent" after AcquiredTS
//	"lead-release:<tag>"   → NEVER auto-evaluates to true (requires explicit release)
//	""                     → NEVER auto-evaluates to true (requires explicit release)
//
// Pure function — no I/O.
func EvaluateCondition(hold HoldRecord, messages []Message) (met bool, reason string) {
	condition := hold.Condition
	if condition == "" {
		return false, ""
	}

	if strings.HasPrefix(condition, "lead-release:") {
		return false, ""
	}

	if agent, ok := strings.CutPrefix(condition, "done:"); ok {
		if ts, found := scanForMessageType(messages, agent, "done", hold.AcquiredTS); found {
			return true, fmt.Sprintf("%s posted done at %s", agent, ts)
		}

		return false, ""
	}

	if agent, ok := strings.CutPrefix(condition, "first-intent:"); ok {
		if ts, found := scanForMessageType(messages, agent, "intent", hold.AcquiredTS); found {
			return true, fmt.Sprintf("%s posted intent at %s", agent, ts)
		}

		return false, ""
	}

	// Unknown condition prefix — never auto-releases
	return false, ""
}

// ScanActiveHolds returns holds with no matching release in messages.
// Pure function — no I/O. Both acquire and release messages are unmarshaled to
// extract HoldID for matching. Messages that fail to unmarshal are silently skipped.
func ScanActiveHolds(messages []Message) []HoldRecord {
	acquired := make(map[string]HoldRecord)

	for _, msg := range messages {
		switch msg.Type {
		case "hold-acquire":
			var record HoldRecord

			err := json.Unmarshal([]byte(msg.Text), &record)
			if err != nil {
				continue
			}

			acquired[record.HoldID] = record
		case "hold-release":
			var record HoldRecord

			err := json.Unmarshal([]byte(msg.Text), &record)
			if err != nil {
				continue
			}

			delete(acquired, record.HoldID)
		}
	}

	holds := make([]HoldRecord, 0, len(acquired))
	for _, record := range acquired {
		holds = append(holds, record)
	}

	return holds
}

// scanForMessageType returns the timestamp and true when a message from agent
// with msgType exists in messages after the given time.
func scanForMessageType(messages []Message, agent string, msgType string, after time.Time) (string, bool) {
	for _, msg := range messages {
		if msg.From != agent || msg.Type != msgType || !msg.TS.After(after) {
			continue
		}

		return msg.TS.Format(time.RFC3339), true
	}

	return "", false
}
