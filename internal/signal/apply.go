package signal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"engram/internal/keyword"
	"engram/internal/memory"
)

// Exported variables.
var (
	ErrMissingDependency = errors.New("missing required dependency")
	ErrMissingField      = errors.New("missing required field")
	ErrUnsupportedAction = errors.New("unsupported action")
)

// Applier dispatches apply-proposal actions to handlers.
type Applier struct {
	readMemory func(path string) (*memory.Stored, error)
	writeMem   MemoryWriter
	removeFile func(string) error
	extractor  Extractor
	archiver   Archiver
	loadRecord func(string) (*memory.MemoryRecord, error)
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
func (a *Applier) Apply(ctx context.Context, action ApplyAction) (ApplyResult, error) {
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
	case actionRefineKeywords:
		err = a.applyRefine(action)
	case actionConsolidate:
		err = a.applyConsolidate(ctx, action)
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

	stored.Keywords = append(stored.Keywords, keyword.NormalizeAll(action.Keywords)...)

	writeErr := a.writeMem.Write(action.Memory, stored)
	if writeErr != nil {
		return fmt.Errorf("writing broadened memory: %w", writeErr)
	}

	return nil
}

func (a *Applier) applyConsolidate(ctx context.Context, action ApplyAction) error {
	if a.extractor == nil {
		return fmt.Errorf("consolidate extractor: %w", ErrMissingDependency)
	}

	if a.loadRecord == nil {
		return fmt.Errorf("consolidate loadRecord: %w", ErrMissingDependency)
	}

	if a.writeMem == nil {
		return fmt.Errorf("consolidate memory writer: %w", ErrMissingDependency)
	}

	memberPaths, parseErr := parseMemberPaths(action.Fields)
	if parseErr != nil {
		return fmt.Errorf("consolidate: %w", parseErr)
	}

	members, loadErr := a.loadMembers(memberPaths)
	if loadErr != nil {
		return loadErr
	}

	cluster := ConfirmedCluster{Members: members}

	consolidated, extractErr := a.extractor.ExtractPrinciple(ctx, cluster)
	if extractErr != nil {
		return fmt.Errorf("consolidate: extracting principle: %w", extractErr)
	}

	TransferFields(consolidated, members, time.Now())

	stored := consolidated.ToStored(action.Memory)

	writeErr := a.writeMem.Write(action.Memory, stored)
	if writeErr != nil {
		return fmt.Errorf("consolidate: writing: %w", writeErr)
	}

	return a.archiveNonSurvivors(memberPaths, action.Memory)
}

func (a *Applier) applyRefine(action ApplyAction) error {
	stored, err := a.readMemory(action.Memory)
	if err != nil {
		return fmt.Errorf("reading memory for refine: %w", err)
	}

	if stored == nil {
		return fmt.Errorf("reading memory for refine: %w", os.ErrNotExist)
	}

	removeSet := toStringSet(action.Fields["remove_keywords"])
	addKeywords := toStringSlice(action.Fields["add_keywords"])

	// Remove specified keywords.
	filtered := make([]string, 0, len(stored.Keywords))

	for _, kw := range stored.Keywords {
		if !removeSet[kw] {
			filtered = append(filtered, kw)
		}
	}

	filtered = append(filtered, keyword.NormalizeAll(addKeywords)...)
	stored.Keywords = filtered
	stored.IrrelevantQueries = nil

	writeErr := a.writeMem.Write(action.Memory, stored)
	if writeErr != nil {
		return fmt.Errorf("writing refined memory: %w", writeErr)
	}

	return nil
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

func (a *Applier) archiveNonSurvivors(memberPaths []string, survivor string) error {
	if a.archiver == nil {
		return nil
	}

	for _, path := range memberPaths {
		if path == survivor {
			continue
		}

		archErr := a.archiver.Archive(path)
		if archErr != nil {
			return fmt.Errorf("consolidate: archiving %s: %w", path, archErr)
		}
	}

	return nil
}

func (a *Applier) loadMembers(paths []string) ([]*memory.MemoryRecord, error) {
	members := make([]*memory.MemoryRecord, 0, len(paths))

	for _, path := range paths {
		rec, loadErr := a.loadRecord(path)
		if loadErr != nil {
			return nil, fmt.Errorf("consolidate: loading %s: %w", path, loadErr)
		}

		members = append(members, rec)
	}

	return members, nil
}

// ApplierOption configures an Applier.
type ApplierOption func(*Applier)

// ApplyAction describes a maintenance action to execute (ARCH-77).
type ApplyAction struct {
	Action   string         `json:"action"`
	Memory   string         `json:"memory"`
	Fields   map[string]any `json:"fields,omitempty"`
	Keywords []string       `json:"keywords,omitempty"`
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

// WithApplyArchiver sets the archiver for consolidation.
func WithApplyArchiver(arch Archiver) ApplierOption {
	return func(a *Applier) { a.archiver = arch }
}

// WithApplyExtractor sets the principle extractor for consolidation.
func WithApplyExtractor(e Extractor) ApplierOption {
	return func(a *Applier) { a.extractor = e }
}

// WithLoadRecord sets the function to load a memory record by path.
func WithLoadRecord(fn func(string) (*memory.MemoryRecord, error)) ApplierOption {
	return func(a *Applier) { a.loadRecord = fn }
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
	actionConsolidate     = "consolidate"
	actionRefineKeywords  = "refine_keywords"
	actionRemove          = "remove"
	actionRewrite         = "rewrite"
)

func applyFields(stored *memory.Stored, fields map[string]any) {
	for key, val := range fields {
		applyStringField(stored, key, val)
	}

	applyKeywordsField(stored, fields)
}

func applyKeywordsField(stored *memory.Stored, fields map[string]any) {
	kw, ok := fields["keywords"]
	if !ok {
		return
	}

	slice, isSlice := kw.([]any)
	if !isSlice {
		return
	}

	keywords := make([]string, 0, len(slice))

	for _, item := range slice {
		if strItem, isStr := item.(string); isStr {
			keywords = append(keywords, keyword.Normalize(strItem))
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

func parseMemberPaths(fields map[string]any) ([]string, error) {
	raw, ok := fields["members"]
	if !ok {
		return nil, fmt.Errorf("members: %w", ErrMissingField)
	}

	type memberEntry struct {
		Path string `json:"path"`
	}

	var membersList []memberEntry

	switch v := raw.(type) {
	case json.RawMessage:
		unmarshalErr := json.Unmarshal(v, &membersList)
		if unmarshalErr != nil {
			return nil, fmt.Errorf("parsing members: %w", unmarshalErr)
		}
	case []any:
		for _, item := range v {
			if m, isMap := item.(map[string]any); isMap {
				if p, hasPath := m["path"].(string); hasPath {
					membersList = append(membersList, memberEntry{Path: p})
				}
			}
		}
	default:
		return nil, fmt.Errorf("members type %T: %w", raw, ErrUnsupportedAction)
	}

	paths := make([]string, 0, len(membersList))

	for _, m := range membersList {
		paths = append(paths, m.Path)
	}

	return paths, nil
}

func toStringSet(val any) map[string]bool {
	set := make(map[string]bool)

	items, ok := val.([]any)
	if !ok {
		return set
	}

	for _, item := range items {
		if s, ok := item.(string); ok {
			set[s] = true
		}
	}

	return set
}

func toStringSlice(val any) []string {
	items, ok := val.([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(items))

	for _, item := range items {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}

	return result
}
