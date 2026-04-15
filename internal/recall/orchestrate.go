package recall

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"engram/internal/memory"
)

// Exported constants.
const (
	DefaultExtractCap       = 10 * 1024 // 10KB of extracted content (mode B)
	DefaultMemoryLimit      = 10        // default max memories returned by RecallMemoriesOnly
	DefaultModeABudget      = 15 * 1024 // 15KB for mode A raw transcript
	DefaultModeBInputBudget = 50 * 1024 // 50KB accumulated input before summarizing (mode B)
	DefaultStripBudget      = 50 * 1024 // 50KB per-session read budget (mode B)
)

// Finder finds session transcript files.
type Finder interface {
	Find(projectDir string) ([]FileEntry, error)
}

// MemoryLister lists all memories from the data directory.
type MemoryLister interface {
	ListAllMemories(dataDir string) ([]*memory.Stored, error)
}

// Orchestrator composes the recall pipeline.
type Orchestrator struct {
	finder       Finder
	reader       Reader
	summarizer   SummarizerI
	memoryLister MemoryLister
	dataDir      string
	statusWriter io.Writer
}

// OrchestratorOption configures optional Orchestrator dependencies.
type OrchestratorOption func(*Orchestrator)

// WithStatusWriter sets a writer for progress messages during recall.
func WithStatusWriter(w io.Writer) OrchestratorOption {
	return func(o *Orchestrator) {
		o.statusWriter = w
	}
}

// NewOrchestrator creates an Orchestrator with the given collaborators.
// memoryLister and dataDir can be nil/empty to disable memory surfacing.
func NewOrchestrator(
	finder Finder,
	reader Reader,
	summarizer SummarizerI,
	memoryLister MemoryLister,
	dataDir string,
	opts ...OrchestratorOption,
) *Orchestrator {
	orch := &Orchestrator{
		finder:       finder,
		reader:       reader,
		summarizer:   summarizer,
		memoryLister: memoryLister,
		dataDir:      dataDir,
	}

	for _, opt := range opts {
		opt(orch)
	}

	return orch
}

// writeStatus writes a progress message if a status writer is configured.
func (o *Orchestrator) writeStatus(format string, args ...any) {
	if o.statusWriter == nil {
		return
	}

	fmt.Fprintf(o.statusWriter, format+"\n", args...)
}

// Recall executes the recall pipeline.
// If query is empty (mode A): find sessions, read+strip, return raw content.
// If query is non-empty (mode B): for each session, extract relevant content via LLM.
func (o *Orchestrator) Recall(
	ctx context.Context,
	projectDir, query string,
) (*Result, error) {
	sessions, err := o.finder.Find(projectDir)
	if err != nil {
		return nil, fmt.Errorf("recalling: %w", err)
	}

	if len(sessions) == 0 {
		return &Result{}, nil
	}

	if query == "" {
		return o.recallModeA(ctx, sessions)
	}

	return o.recallModeB(ctx, sessions, query)
}

// RecallMemoriesOnly searches memories using Haiku two-phase matching.
// Phase 1: build an index of all memories (type | name | situation).
// Phase 2: ask Haiku which names are relevant to the query.
// Phase 3: load full content of matched memories, sorted by source and recency.
func (o *Orchestrator) RecallMemoriesOnly(
	ctx context.Context,
	query string,
	limit int,
) (*Result, error) {
	if limit <= 0 {
		limit = DefaultMemoryLimit
	}

	memories, err := o.listAndMatchMemories(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("recalling memories: %w", err)
	}

	if len(memories) == 0 {
		return &Result{}, nil
	}

	return &Result{Memories: formatMemories(memories)}, nil
}

// accumulateSessionsAndMemories reads sessions newest-first, interleaving
// per-session memories, until the input budget is reached.
func (o *Orchestrator) accumulateSessionsAndMemories(
	ctx context.Context,
	sessions []FileEntry,
) string {
	allMemories := o.loadAllMemories()

	var builder strings.Builder

	bytesRead := 0

	for i, entry := range sessions {
		if ctx.Err() != nil {
			break
		}

		content, size, readErr := o.reader.Read(
			entry.Path, DefaultModeBInputBudget-bytesRead,
		)
		if readErr != nil {
			continue
		}

		builder.WriteString(content)

		bytesRead += size

		if allMemories != nil {
			window := buildSingleTimeWindow(sessions, i)
			matched := matchMemoriesToWindows(allMemories, []timeWindow{window})

			if len(matched) > 0 {
				memText := formatMemories(matched)
				builder.WriteString(memText)

				bytesRead += len(memText)
			}
		}

		if bytesRead >= DefaultModeBInputBudget {
			break
		}
	}

	return builder.String()
}

// findSessionMemories returns formatted memories whose UpdatedAt falls within
// any of the given sessions' time windows. Returns empty string if no
// memoryLister is configured or no memories match.
func (o *Orchestrator) findSessionMemories(sessions []FileEntry) string {
	if o.memoryLister == nil || o.dataDir == "" {
		return ""
	}

	allMemories, err := o.memoryLister.ListAllMemories(o.dataDir)
	if err != nil || len(allMemories) == 0 {
		return ""
	}

	windows := buildTimeWindows(sessions)
	matched := matchMemoriesToWindows(allMemories, windows)

	if len(matched) == 0 {
		return ""
	}

	return formatMemories(matched)
}

// listAndMatchMemories runs the Haiku two-phase memory search.
// Returns nil if no summarizer, no memory lister, or no matches.
func (o *Orchestrator) listAndMatchMemories(
	ctx context.Context,
	query string,
	limit int,
) ([]*memory.Stored, error) {
	if o.summarizer == nil || o.memoryLister == nil || o.dataDir == "" {
		return nil, nil
	}

	allMemories, err := o.memoryLister.ListAllMemories(o.dataDir)
	if err != nil || len(allMemories) == 0 {
		return nil, nil //nolint:nilerr // empty list is not an error for callers
	}

	index := buildMemoryIndex(allMemories)

	matchPrompt := fmt.Sprintf(
		"Return ONLY the names of memories relevant to this query, one per line."+
			" Max %d names. Return NOTHING if none match.\n\nQuery: %s",
		limit, query,
	)

	response, extractErr := o.summarizer.ExtractRelevant(ctx, index, matchPrompt)
	if extractErr != nil {
		return nil, fmt.Errorf("matching memories: %w", extractErr)
	}

	names := parseMemoryNames(response)
	if len(names) == 0 {
		return nil, nil
	}

	matched := filterMemoriesByName(allMemories, names)
	sortMemoriesBySourceAndRecency(matched)

	if len(matched) > limit {
		matched = matched[:limit]
	}

	return matched, nil
}

// loadAllMemories returns all stored memories, or nil if not configured.
func (o *Orchestrator) loadAllMemories() []*memory.Stored {
	if o.memoryLister == nil || o.dataDir == "" {
		return nil
	}

	allMemories, err := o.memoryLister.ListAllMemories(o.dataDir)
	if err != nil || len(allMemories) == 0 {
		return nil
	}

	return allMemories
}

func (o *Orchestrator) recallModeA(
	ctx context.Context,
	sessions []FileEntry,
) (*Result, error) {
	var builder strings.Builder

	bytesRead := 0

	for _, entry := range sessions {
		if ctx.Err() != nil {
			break
		}

		content, size, readErr := o.reader.Read(entry.Path, DefaultModeABudget-bytesRead)
		if readErr != nil {
			continue
		}

		builder.WriteString(content)

		bytesRead += size
		if bytesRead >= DefaultModeABudget {
			break
		}
	}

	memories := o.findSessionMemories(sessions)

	//nolint:nilerr // cancellation returns partial results, not an error.
	return &Result{Summary: builder.String(), Memories: memories}, nil
}

func (o *Orchestrator) recallModeB(
	ctx context.Context,
	sessions []FileEntry,
	query string,
) (*Result, error) {
	if o.summarizer == nil {
		return &Result{}, nil
	}

	accumulated := o.accumulateSessionsAndMemories(ctx, sessions)
	if accumulated == "" {
		return &Result{}, nil
	}

	summary, err := o.summarizer.ExtractRelevant(ctx, accumulated, query)
	if err != nil {
		return nil, fmt.Errorf("summarizing recall: %w", err)
	}

	return &Result{Summary: summary}, nil
}

// Reader reads and strips a transcript file.
type Reader interface {
	Read(path string, budgetBytes int) (string, int, error)
}

// Result holds the output of a recall operation.
type Result struct {
	Summary  string `json:"summary"`
	Memories string `json:"memories,omitempty"`
}

// SummarizerI extracts relevant content from transcripts via LLM.
type SummarizerI interface {
	ExtractRelevant(ctx context.Context, content, query string) (string, error)
	SummarizeFindings(ctx context.Context, content, query string) (string, error)
}

// FormatResult writes the recall result as plain text with an optional memories section.
func FormatResult(w io.Writer, result *Result) error {
	_, err := fmt.Fprint(w, result.Summary)
	if err != nil {
		return fmt.Errorf("writing summary: %w", err)
	}

	if result.Memories != "" {
		_, err = fmt.Fprintf(w, "\n=== MEMORIES ===\n%s", result.Memories)
		if err != nil {
			return fmt.Errorf("writing memories: %w", err)
		}
	}

	return nil
}

// unexported constants.
const (
	defaultSessionWindow = 24 * time.Hour
)

// timeWindow represents the start and end of a session's time range.
type timeWindow struct {
	start time.Time
	end   time.Time
}

// buildMemoryIndex creates the type | name | situation index for Haiku matching.
func buildMemoryIndex(memories []*memory.Stored) string {
	var builder strings.Builder

	for _, mem := range memories {
		name := memory.NameFromPath(mem.FilePath)
		fmt.Fprintf(&builder, "%s | %s | %s\n", mem.Type, name, mem.Situation)
	}

	return builder.String()
}

// buildSingleTimeWindow creates a time window for the session at index i.
func buildSingleTimeWindow(sessions []FileEntry, i int) timeWindow {
	end := sessions[i].Mtime

	var start time.Time
	if i < len(sessions)-1 {
		start = sessions[i+1].Mtime
	} else {
		start = end.Add(-defaultSessionWindow)
	}

	return timeWindow{start: start, end: end}
}

// buildTimeWindows creates time windows from session entries.
// Sessions are expected in mtime-descending order.
// Each session's end is its mtime; start is the previous session's mtime
// (or 24h before for the first/only session).
func buildTimeWindows(sessions []FileEntry) []timeWindow {
	if len(sessions) == 0 {
		return nil
	}

	windows := make([]timeWindow, 0, len(sessions))

	for i, entry := range sessions {
		end := entry.Mtime

		var start time.Time
		if i < len(sessions)-1 {
			// Previous session's mtime (next in the slice since sorted desc).
			start = sessions[i+1].Mtime
		} else {
			// First/only session: go back 24h.
			start = end.Add(-defaultSessionWindow)
		}

		windows = append(windows, timeWindow{start: start, end: end})
	}

	return windows
}

// filterMemoriesByName returns memories whose NameFromPath matches any of the given names.
func filterMemoriesByName(memories []*memory.Stored, names []string) []*memory.Stored {
	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		nameSet[name] = struct{}{}
	}

	matched := make([]*memory.Stored, 0, len(names))

	for _, mem := range memories {
		name := memory.NameFromPath(mem.FilePath)
		if _, ok := nameSet[name]; ok {
			matched = append(matched, mem)
		}
	}

	return matched
}

// formatFactFields writes fact-specific fields to the builder.
func formatFactFields(builder *strings.Builder, mem *memory.Stored) {
	builder.WriteString("  subject: " + mem.Content.Subject)

	if mem.Content.Predicate != "" {
		builder.WriteString(" | predicate: " + mem.Content.Predicate)
	}

	if mem.Content.Object != "" {
		builder.WriteString(" | object: " + mem.Content.Object)
	}

	builder.WriteByte('\n')
}

// formatFeedbackFields writes feedback-specific fields to the builder.
func formatFeedbackFields(builder *strings.Builder, mem *memory.Stored) {
	if mem.Content.Behavior != "" {
		builder.WriteString("  behavior: " + mem.Content.Behavior + "\n")
	}

	if mem.Content.Action != "" {
		builder.WriteString("  action: " + mem.Content.Action + "\n")
	}
}

// formatMemories renders memories as human-readable text.
func formatMemories(memories []*memory.Stored) string {
	var builder strings.Builder

	for i, mem := range memories {
		if i > 0 {
			builder.WriteByte('\n')
		}

		builder.WriteString(formatSingleMemory(mem))
	}

	return builder.String()
}

// formatSingleMemory formats one memory for display.
func formatSingleMemory(mem *memory.Stored) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "[%s] situation: %s\n", mem.Type, mem.Situation)

	if mem.Type == "feedback" {
		formatFeedbackFields(&builder, mem)
	} else if mem.Content.Subject != "" {
		formatFactFields(&builder, mem)
	}

	return builder.String()
}

// matchMemoriesToWindows returns memories whose UpdatedAt falls within any window.
func matchMemoriesToWindows(memories []*memory.Stored, windows []timeWindow) []*memory.Stored {
	matched := make([]*memory.Stored, 0)

	for _, mem := range memories {
		for _, win := range windows {
			if !mem.UpdatedAt.Before(win.start) && !mem.UpdatedAt.After(win.end) {
				matched = append(matched, mem)

				break
			}
		}
	}

	return matched
}

// parseMemoryNames extracts memory names from the Haiku response (one per line).
func parseMemoryNames(response string) []string {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	names := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}

	return names
}

// sortMemoriesBySourceAndRecency sorts memories: human-sourced first, then
// agent-sourced, within each group by most recent UpdatedAt first.
func sortMemoriesBySourceAndRecency(memories []*memory.Stored) {
	sort.Slice(memories, func(i, j int) bool {
		iHuman := memories[i].Source == "human"
		jHuman := memories[j].Source == "human"

		if iHuman != jHuman {
			return iHuman
		}

		return memories[i].UpdatedAt.After(memories[j].UpdatedAt)
	})
}
