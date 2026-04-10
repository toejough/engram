// Package agent provides pure domain types for the engram agent lifecycle.
// All functions are pure (no I/O). OS wiring lives in internal/cli.
package agent

import (
	"bytes"
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
)

// AgentRecord holds binary bookkeeping for a spawned agent (spec §6.3).
// Argument state fields enforce the 3-argument cap from SPEECH-2/SKILL-2
// and must be persisted across engram agent resume invocations (Phase 5).
//
//nolint:revive,tagliatelle // "agent.AgentRecord" stutter is intentional; kebab-case matches state file protocol.
type AgentRecord struct {
	Name           string    `json:"name"                     toml:"name"`
	PaneID         string    `json:"pane-id"                  toml:"pane-id"`
	SessionID      string    `json:"session-id"               toml:"session-id"`
	State          string    `json:"state"                    toml:"state"` // STARTING | ACTIVE | SILENT | DEAD
	SpawnedAt      time.Time `json:"spawned-at"               toml:"spawned-at"`
	LastResumedAt  time.Time `json:"last-resumed-at,omitzero" toml:"last-resumed-at,omitzero"`
	ArgumentWith   string    `json:"argument-with"            toml:"argument-with"`
	ArgumentCount  int       `json:"argument-count"           toml:"argument-count"`
	ArgumentThread string    `json:"argument-thread"          toml:"argument-thread"`
}

// HoldEntry is the state-file representation of an active hold.
// Mirrors chat.HoldRecord but stored as TOML struct (not JSON-in-text).
// The state file is the snapshot authority for hold state after Phase 3.
type HoldEntry struct {
	HoldID     string    `toml:"hold-id"`
	Holder     string    `toml:"holder"`
	Target     string    `toml:"target"`
	Condition  string    `toml:"condition,omitempty"`
	Tag        string    `toml:"tag,omitempty"`
	AcquiredTS time.Time `toml:"acquired-ts"`
}

// StateFile holds the parsed contents of the state TOML file.
// The binary is the sole writer — no skill writes this file directly.
type StateFile struct {
	Agents []AgentRecord `toml:"agent"`
	Holds  []HoldEntry   `toml:"hold"`
}

// ActiveWorkerCount returns the number of agents in STARTING or ACTIVE state.
// Pure function — no I/O.
func ActiveWorkerCount(sf StateFile) int {
	count := 0

	for _, rec := range sf.Agents {
		if rec.State == "STARTING" || rec.State == "ACTIVE" {
			count++
		}
	}

	return count
}

// AddAgent returns a new StateFile with record appended to Agents.
// The original StateFile is not modified (pure function).
func AddAgent(stateFile StateFile, record AgentRecord) StateFile {
	result := stateFile
	result.Agents = append(make([]AgentRecord, 0, len(stateFile.Agents)+1), stateFile.Agents...)
	result.Agents = append(result.Agents, record)

	return result
}

// AddHold returns a new StateFile with hold appended to Holds.
// The original StateFile is not modified (pure function).
func AddHold(stateFile StateFile, hold HoldEntry) StateFile {
	result := stateFile
	result.Holds = append(make([]HoldEntry, 0, len(stateFile.Holds)+1), stateFile.Holds...)
	result.Holds = append(result.Holds, hold)

	return result
}

// MarshalStateFile serializes a StateFile to TOML bytes.
func MarshalStateFile(stateFile StateFile) ([]byte, error) {
	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(stateFile)
	if err != nil {
		return nil, fmt.Errorf("marshaling state file: %w", err)
	}

	return buf.Bytes(), nil
}

// ParseStateFile deserializes TOML state file data.
// Returns an empty StateFile for nil or empty data.
func ParseStateFile(data []byte) (StateFile, error) {
	if len(data) == 0 {
		return StateFile{}, nil
	}

	var parsed StateFile

	err := toml.Unmarshal(data, &parsed)
	if err != nil {
		return StateFile{}, fmt.Errorf("parsing state file: %w", err)
	}

	return parsed, nil
}

// RemoveAgent returns a new StateFile with the named agent removed from Agents.
// If no agent with that name exists, returns stateFile unchanged.
// The original StateFile is not modified (pure function).
func RemoveAgent(stateFile StateFile, name string) StateFile {
	filtered := make([]AgentRecord, 0, len(stateFile.Agents))

	for _, rec := range stateFile.Agents {
		if rec.Name != name {
			filtered = append(filtered, rec)
		}
	}

	result := stateFile
	result.Agents = filtered

	return result
}

// RemoveHold returns a new StateFile with the hold identified by holdID removed.
// If no hold with that ID exists, returns stateFile unchanged.
// The original StateFile is not modified (pure function).
func RemoveHold(stateFile StateFile, holdID string) StateFile {
	filtered := make([]HoldEntry, 0, len(stateFile.Holds))

	for _, hold := range stateFile.Holds {
		if hold.HoldID != holdID {
			filtered = append(filtered, hold)
		}
	}

	result := stateFile
	result.Holds = filtered

	return result
}
