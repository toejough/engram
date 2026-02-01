// Package state manages the project state machine stored as a TOML file.
package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// StateFile is the filename for the project state.
const StateFile = "state.toml"

// Project holds the project identification and phase.
type Project struct {
	Name    string    `toml:"name"`
	Created time.Time `toml:"created"`
	Phase   string    `toml:"phase"`
}

// Progress tracks implementation progress.
type Progress struct {
	CurrentTask     string `toml:"current_task"`
	CurrentSubphase string `toml:"current_subphase"`
	TasksComplete   int    `toml:"tasks_complete"`
	TasksTotal      int    `toml:"tasks_total"`
	TasksEscalated  int    `toml:"tasks_escalated"`
}

// Conflicts tracks conflict state.
type Conflicts struct {
	Open          int      `toml:"open"`
	BlockingTasks []string `toml:"blocking_tasks"`
}

// Meta tracks meta-audit state.
type Meta struct {
	CorrectionsSinceLastAudit int       `toml:"corrections_since_last_audit"`
	LastMetaAudit             time.Time `toml:"last_meta_audit,omitempty"`
}

// PhaseTransition records a state transition.
type PhaseTransition struct {
	Timestamp time.Time `toml:"timestamp"`
	Phase     string    `toml:"phase"`
}

// State is the complete project state.
type State struct {
	Project   Project           `toml:"project"`
	Progress  Progress          `toml:"progress"`
	Conflicts Conflicts         `toml:"conflicts"`
	Meta      Meta              `toml:"meta"`
	History   []PhaseTransition `toml:"history"`
}

// Init creates a new state file in the given directory.
func Init(dir string, name string, now func() time.Time) (State, error) {
	statePath := filepath.Join(dir, StateFile)

	if _, err := os.Stat(statePath); err == nil {
		return State{}, fmt.Errorf("state file already exists: %s", statePath)
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return State{}, fmt.Errorf("directory does not exist: %s", dir)
	}

	t := now()

	s := State{
		Project: Project{
			Name:    name,
			Created: t,
			Phase:   "init",
		},
		History: []PhaseTransition{
			{Timestamp: t, Phase: "init"},
		},
	}

	f, err := os.Create(statePath)
	if err != nil {
		return State{}, fmt.Errorf("failed to create state file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(s); err != nil {
		return State{}, fmt.Errorf("failed to encode state: %w", err)
	}

	return s, nil
}

// TransitionOpts holds optional fields for a state transition.
type TransitionOpts struct {
	Task     string
	Subphase string
	Force    bool // Bypass precondition checks (not transition graph checks)
}

// PreconditionChecker validates preconditions for phase transitions.
type PreconditionChecker interface {
	RequirementsExist(dir string) bool
	RequirementsHaveIDs(dir string) bool
	DesignExists(dir string) bool
	DesignHasIDs(dir string) bool
	TraceValidationPasses(dir string) bool
	TestsExist(dir string) bool
	TestsFail(dir string) bool
	TestsPass(dir string) bool
}

// Preconditions maps phases to their required preconditions.
var Preconditions = map[string]func(dir string, checker PreconditionChecker) error{
	"pm-complete": func(dir string, c PreconditionChecker) error {
		if !c.RequirementsExist(dir) {
			return fmt.Errorf("precondition failed: requirements.md must exist")
		}
		if !c.RequirementsHaveIDs(dir) {
			return fmt.Errorf("precondition failed: requirements.md must contain REQ-NNN IDs")
		}
		return nil
	},
	"design-complete": func(dir string, c PreconditionChecker) error {
		if !c.DesignExists(dir) {
			return fmt.Errorf("precondition failed: design.md must exist")
		}
		if !c.DesignHasIDs(dir) {
			return fmt.Errorf("precondition failed: design.md must contain DES-NNN IDs")
		}
		return nil
	},
	"architect-complete": func(dir string, c PreconditionChecker) error {
		if !c.TraceValidationPasses(dir) {
			return fmt.Errorf("precondition failed: trace validation must pass")
		}
		return nil
	},
	"task-complete": func(dir string, c PreconditionChecker) error {
		if !c.TraceValidationPasses(dir) {
			return fmt.Errorf("precondition failed: trace validation must pass")
		}
		return nil
	},
	"tdd-green": func(dir string, c PreconditionChecker) error {
		if !c.TestsExist(dir) {
			return fmt.Errorf("precondition failed: test files must exist")
		}
		if !c.TestsFail(dir) {
			return fmt.Errorf("precondition failed: tests must currently fail")
		}
		return nil
	},
	"tdd-refactor": func(dir string, c PreconditionChecker) error {
		if !c.TestsPass(dir) {
			return fmt.Errorf("precondition failed: all tests must pass")
		}
		return nil
	},
}

// Transition moves the project to a new phase, validating the transition is legal.
// Writes atomically (temp file + rename) to avoid corruption.
// This variant does not check preconditions - use TransitionWithChecker for that.
func Transition(dir string, to string, opts TransitionOpts, now func() time.Time) (State, error) {
	return TransitionWithChecker(dir, to, opts, now, nil)
}

// TransitionWithChecker moves the project to a new phase, validating both
// the transition graph and preconditions. If checker is nil, preconditions
// are not validated.
func TransitionWithChecker(dir string, to string, opts TransitionOpts, now func() time.Time, checker PreconditionChecker) (State, error) {
	s, err := Get(dir)
	if err != nil {
		return State{}, err
	}

	from := s.Project.Phase
	if !IsLegalTransition(from, to) {
		targets := LegalTargets(from)
		return State{}, fmt.Errorf(
			"illegal transition: %s → %s (legal targets: %v)",
			from, to, targets,
		)
	}

	// Check preconditions if checker provided and not forcing
	if checker != nil && !opts.Force {
		if precondCheck, ok := Preconditions[to]; ok {
			if err := precondCheck(dir, checker); err != nil {
				return State{}, err
			}
		}
	}

	t := now()
	s.Project.Phase = to
	s.History = append(s.History, PhaseTransition{Timestamp: t, Phase: to})

	if opts.Task != "" {
		s.Progress.CurrentTask = opts.Task
	}

	if opts.Subphase != "" {
		s.Progress.CurrentSubphase = opts.Subphase
	}

	if err := writeAtomic(dir, s); err != nil {
		return State{}, err
	}

	return s, nil
}

func writeAtomic(dir string, s State) error {
	statePath := filepath.Join(dir, StateFile)
	tmpPath := statePath + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if err := toml.NewEncoder(f).Encode(s); err != nil {
		f.Close()
		os.Remove(tmpPath)

		return fmt.Errorf("failed to encode state: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)

		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		os.Remove(tmpPath)

		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Get reads the state file from the given directory.
func Get(dir string) (State, error) {
	statePath := filepath.Join(dir, StateFile)

	var s State

	if _, err := toml.DecodeFile(statePath, &s); err != nil {
		return State{}, fmt.Errorf("failed to read state file: %w", err)
	}

	return s, nil
}
