package signal

import (
	"context"
	"errors"
	"fmt"
	"os"

	"engram/internal/maintain"
	"engram/internal/memory"
)

// Exported variables.
var (
	ErrLevelOutOfRange   = errors.New("escalation level out of range")
	ErrUnsupportedAction = errors.New("unsupported action")
	ErrZeroLevel         = errors.New("escalation requires non-zero level")
)

// Applier dispatches apply-proposal actions to handlers.
type Applier struct {
	readMemory         func(path string) (*memory.Stored, error)
	writeMem           MemoryWriter
	removeFile         func(string) error
	enforcementApplier maintain.EnforcementApplier
}

// NewApplier creates an Applier with the given options.
func NewApplier(opts ...ApplierOption) *Applier {
	a := &Applier{
		removeFile: os.Remove,
	}
	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Apply dispatches the action to the appropriate handler.
func (a *Applier) Apply(_ context.Context, action ApplyAction) (ApplyResult, error) {
	result := ApplyResult{
		Action: action.Action,
		Memory: action.Memory,
	}

	var err error

	switch action.Action {
	case actionRemove:
		err = a.applyRemove(action)
	case actionRewrite:
		err = a.applyRewrite(action)
	case actionBroadenKeywords:
		err = a.applyBroaden(action)
	case actionEscalate:
		err = a.applyEscalate(action)
	default:
		return result, fmt.Errorf("%w: %s", ErrUnsupportedAction, action.Action)
	}

	if err != nil {
		result.Error = err.Error()

		return result, err
	}

	result.Success = true

	return result, nil
}

func (a *Applier) applyBroaden(action ApplyAction) error {
	stored, err := a.readMemory(action.Memory)
	if err != nil {
		return fmt.Errorf("reading memory for broaden: %w", err)
	}

	if stored == nil {
		return fmt.Errorf("reading memory for broaden: %w", os.ErrNotExist)
	}

	stored.Keywords = append(stored.Keywords, action.Keywords...)

	writeErr := a.writeMem.Write(action.Memory, stored)
	if writeErr != nil {
		return fmt.Errorf("writing broadened memory: %w", writeErr)
	}

	return nil
}

func (a *Applier) applyEscalate(action ApplyAction) error {
	if action.Level == 0 {
		return fmt.Errorf("escalation: %w", ErrZeroLevel)
	}

	if action.Level < 1 || action.Level > len(escalationLadder) {
		return fmt.Errorf("escalation: %w: %d", ErrLevelOutOfRange, action.Level)
	}

	proposedLevel := escalationLadder[action.Level-1]

	proposal := maintain.EscalationProposal{
		MemoryPath:    action.Memory,
		ProposalType:  "escalate",
		ProposedLevel: string(proposedLevel),
		Rationale:     "applied via apply-proposal",
	}

	return maintain.ApplyEscalationProposal(
		proposal, a.enforcementApplier,
	)
}

func (a *Applier) applyRemove(action ApplyAction) error {
	err := a.removeFile(action.Memory)
	if err != nil {
		return fmt.Errorf("removing memory file: %w", err)
	}

	return nil
}

func (a *Applier) applyRewrite(action ApplyAction) error {
	stored, err := a.readMemory(action.Memory)
	if err != nil {
		return fmt.Errorf("reading memory for rewrite: %w", err)
	}

	if stored == nil {
		return fmt.Errorf("reading memory for rewrite: %w", os.ErrNotExist)
	}

	applyFields(stored, action.Fields)

	writeErr := a.writeMem.Write(action.Memory, stored)
	if writeErr != nil {
		return fmt.Errorf("writing rewritten memory: %w", writeErr)
	}

	return nil
}

// ApplierOption configures an Applier.
type ApplierOption func(*Applier)

// ApplyAction describes a maintenance action to execute (ARCH-77).
type ApplyAction struct {
	Action   string         `json:"action"`
	Memory   string         `json:"memory"`
	Fields   map[string]any `json:"fields,omitempty"`
	Keywords []string       `json:"keywords,omitempty"`
	Level    int            `json:"level,omitempty"`
}

// ApplyResult holds the outcome of an apply operation.
type ApplyResult struct {
	Success bool   `json:"success"`
	Action  string `json:"action"`
	Memory  string `json:"memory"`
	Error   string `json:"error,omitempty"`
}

// MemoryWriter writes a memory TOML back to disk.
type MemoryWriter interface {
	Write(path string, stored *memory.Stored) error
}

// WithEnforcementApplier sets the enforcement level applier for escalation.
func WithEnforcementApplier(applier maintain.EnforcementApplier) ApplierOption {
	return func(a *Applier) {
		a.enforcementApplier = applier
	}
}

// WithReadMemory sets the memory reader function.
func WithReadMemory(fn func(string) (*memory.Stored, error)) ApplierOption {
	return func(a *Applier) {
		a.readMemory = fn
	}
}

// WithRemoveFile sets the file removal function.
func WithRemoveFile(fn func(string) error) ApplierOption {
	return func(a *Applier) {
		a.removeFile = fn
	}
}

// WithWriteMemory sets the memory writer.
func WithWriteMemory(w MemoryWriter) ApplierOption {
	return func(a *Applier) {
		a.writeMem = w
	}
}

// unexported constants.
const (
	actionBroadenKeywords = "broaden_keywords"
	actionEscalate        = "escalate"
	actionRemove          = "remove"
	actionRewrite         = "rewrite"
)

// unexported variables.
var (
	escalationLadder = []maintain.EscalationLevel{ //nolint:gochecknoglobals // constant table
		maintain.LevelAdvisory,
		maintain.LevelEmphasizedAdvisory,
		maintain.LevelReminder,
	}
)

func applyFields(stored *memory.Stored, fields map[string]any) {
	for key, val := range fields {
		applyStringField(stored, key, val)
	}

	applyKeywordsField(stored, fields)
}

func applyKeywordsField(stored *memory.Stored, fields map[string]any) {
	keyword, ok := fields["keywords"]
	if !ok {
		return
	}

	slice, isSlice := keyword.([]any)
	if !isSlice {
		return
	}

	keywords := make([]string, 0, len(slice))

	for _, item := range slice {
		if strItem, isStr := item.(string); isStr {
			keywords = append(keywords, strItem)
		}
	}

	stored.Keywords = keywords
}

func applyStringField(stored *memory.Stored, key string, val any) {
	strVal, ok := val.(string)
	if !ok {
		return
	}

	switch key {
	case "title":
		stored.Title = strVal
	case "content":
		stored.Content = strVal
	case "principle":
		stored.Principle = strVal
	case "anti_pattern":
		stored.AntiPattern = strVal
	}
}
