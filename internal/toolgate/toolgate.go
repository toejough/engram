// Package toolgate provides frecency-based gating for tool calls,
// suppressing repetitive suggestions that have low impact.
package toolgate

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// CounterEntry tracks call frequency for a single command key.
type CounterEntry struct {
	Count int       `json:"count"`
	Last  time.Time `json:"last"`
}

// CounterStore abstracts persistent storage of tool call counters.
type CounterStore interface {
	Load() (map[string]CounterEntry, error)
	Save(counters map[string]CounterEntry) error
}

// Gate decides whether to surface memories for a given command,
// based on persistent call frequency.
type Gate struct {
	store  CounterStore
	randFn func() float64
}

// NewGate creates a Gate. store handles persistence, randFn provides randomness.
func NewGate(store CounterStore, randFn func() float64) *Gate {
	return &Gate{store: store, randFn: randFn}
}

// Check reads the current count for key, rolls the probability gate,
// then increments and persists the counter. Returns true if surfacing
// should proceed.
func (g *Gate) Check(key string) (bool, error) {
	counters, err := g.store.Load()
	if err != nil {
		return true, nil //nolint:nilerr // fail-open by design: surface on read error
	}

	if counters == nil {
		counters = make(map[string]CounterEntry)
	}

	entry := counters[key]
	prob := SurfaceProbability(entry.Count)
	shouldSurface := g.randFn() < prob

	entry.Count++
	entry.Last = time.Now()
	counters[key] = entry

	saveErr := g.store.Save(counters)
	if saveErr != nil {
		return shouldSurface, fmt.Errorf("toolgate save: %w", saveErr)
	}

	return shouldSurface, nil
}

// CommandKey extracts a stable identity from a bash command string.
// Strips leading VAR=val env assignments, takes first two tokens,
// drops the second token if it starts with "-" (flags aren't identity).
func CommandKey(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}

	// Strip leading env var assignments (contain '=' but not as first char).
	for len(fields) > 0 && strings.Contains(fields[0], "=") && !strings.HasPrefix(fields[0], "=") {
		fields = fields[1:]
	}

	if len(fields) == 0 {
		return ""
	}

	if len(fields) == 1 {
		return fields[0]
	}

	// Drop second token if it's a flag.
	if strings.HasPrefix(fields[1], "-") {
		return fields[0]
	}

	return fields[0] + " " + fields[1]
}

// ExtractBashCommand extracts the .command field from Bash tool input JSON.
func ExtractBashCommand(toolInput string) string {
	var input struct {
		Command string `json:"command"`
	}

	jsonErr := json.Unmarshal([]byte(toolInput), &input)
	if jsonErr != nil {
		return ""
	}

	return input.Command
}

// SurfaceProbability computes the probability of surfacing memories for a
// command that has been called count times. Uses smooth logarithmic decay:
// P = 1 / (1 + ln(1 + count)).
func SurfaceProbability(count int) float64 {
	return 1.0 / (1.0 + math.Log(1.0+float64(count)))
}
