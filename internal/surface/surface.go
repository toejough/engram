// Package surface implements memory surfacing for UC-2 (ARCH-12).
// Routes to SessionStart, UserPromptSubmit, or PreToolUse mode based on options.
package surface

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"engram/internal/bm25"
	"engram/internal/creationlog"
	"engram/internal/memory"
)

// Exported constants.
const (
	FormatJSON       = "json"
	ModePrompt       = "prompt"
	ModeSessionStart = "session-start"
	ModeTool         = "tool"
)

// Exported variables.
var (
	ErrUnknownMode = errors.New("surface: unknown mode")
)

// CreationLogReader reads and clears the creation log (ARCH-12).
type CreationLogReader interface {
	ReadAndClear(dataDir string) ([]LogEntry, error)
}

// EffectivenessComputer provides per-memory effectiveness aggregates (ARCH-24).
type EffectivenessComputer interface {
	Aggregate() (map[string]EffectivenessStat, error)
}

// EffectivenessStat holds aggregated effectiveness data for a single memory.
type EffectivenessStat struct {
	SurfacedCount      int
	EffectivenessScore float64 // followed% (0–100)
}

// LogEntry is an alias for creationlog.LogEntry (avoids coupling callers to creationlog package).
type LogEntry = creationlog.LogEntry

// MemoryRetriever lists stored memories from disk (ARCH-9).
type MemoryRetriever interface {
	ListMemories(ctx context.Context, dataDir string) ([]*memory.Stored, error)
}

// MemoryTracker records surfacing events for instrumentation (ARCH-19).
type MemoryTracker interface {
	RecordSurfacing(ctx context.Context, memories []*memory.Stored, mode string) error
}

// Options configures a surface invocation.
type Options struct {
	Mode      string
	DataDir   string
	Message   string // for prompt mode
	ToolName  string // for tool mode
	ToolInput string // for tool mode
	Format    string // output format: "" (plain) or "json"
}

// Result holds the structured output of a surface invocation.
type Result struct {
	Summary string `json:"summary"`
	Context string `json:"context"`
}

// Surfacer orchestrates memory surfacing.
type Surfacer struct {
	retriever             MemoryRetriever
	tracker               MemoryTracker
	logReader             CreationLogReader
	surfacingLogger       SurfacingEventLogger
	effectivenessComputer EffectivenessComputer
}

// New creates a Surfacer.
func New(retriever MemoryRetriever, opts ...SurfacerOption) *Surfacer {
	s := &Surfacer{
		retriever: retriever,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Run executes the surface subcommand, writing output to w.
//
//nolint:cyclop // orchestration function: routes modes, logs events, writes result — inherent branching
func (s *Surfacer) Run(ctx context.Context, w io.Writer, opts Options) error {
	// Fetch effectiveness data upfront (fire-and-forget on error per ARCH-6).
	var effectiveness map[string]EffectivenessStat

	if s.effectivenessComputer != nil {
		effData, effErr := s.effectivenessComputer.Aggregate()
		if effErr == nil {
			effectiveness = effData
		}
	}

	var (
		result  Result
		matched []*memory.Stored
		err     error
	)

	switch opts.Mode {
	case ModeSessionStart:
		result, matched, err = s.runSessionStart(ctx, opts.DataDir, effectiveness)
	case ModePrompt:
		result, matched, err = s.runPrompt(ctx, opts.DataDir, opts.Message, effectiveness)
	case ModeTool:
		result, matched, err = s.runTool(ctx, opts, effectiveness)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownMode, opts.Mode)
	}

	if err != nil {
		return err
	}

	if s.tracker != nil && len(matched) > 0 {
		_ = s.tracker.RecordSurfacing(ctx, matched, opts.Mode)
	}

	if s.surfacingLogger != nil {
		now := time.Now()
		for _, mem := range matched {
			_ = s.surfacingLogger.LogSurfacing(mem.FilePath, opts.Mode, now)
		}
	}

	return s.writeResult(w, result, opts.Format)
}

func (s *Surfacer) runPrompt(
	ctx context.Context,
	dataDir, message string,
	effectiveness map[string]EffectivenessStat,
) (Result, []*memory.Stored, error) {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, nil, fmt.Errorf("surface: %w", err)
	}

	matches := matchPromptMemories(message, memories)
	if len(matches) == 0 {
		return Result{}, nil, nil
	}

	var buf strings.Builder

	_, _ = fmt.Fprintf(&buf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(&buf, "[engram] Relevant memories:\n")

	for _, match := range matches {
		annotation := formatEffectivenessAnnotation(match.mem.FilePath, effectiveness)
		_, _ = fmt.Fprintf(&buf, "  - %s%s\n",
			filenameSlug(match.mem.FilePath), annotation)
	}

	_, _ = fmt.Fprintf(&buf, "</system-reminder>\n")

	promptMems := make([]*memory.Stored, 0, len(matches))
	for _, m := range matches {
		promptMems = append(promptMems, m.mem)
	}

	var summaryBuf strings.Builder

	_, _ = fmt.Fprintf(&summaryBuf, "[engram] %d relevant memories:\n", len(matches))

	for _, match := range matches {
		annotation := formatEffectivenessAnnotation(match.mem.FilePath, effectiveness)
		_, _ = fmt.Fprintf(&summaryBuf, "  - %s%s\n",
			filenameSlug(match.mem.FilePath), annotation)
	}

	return Result{
		Summary: strings.TrimRight(summaryBuf.String(), "\n"),
		Context: buf.String(),
	}, promptMems, nil
}

func (s *Surfacer) runSessionStart(
	ctx context.Context,
	dataDir string,
	effectiveness map[string]EffectivenessStat,
) (Result, []*memory.Stored, error) {
	// Step 1: Read creation log (ARCH-12). Errors are fire-and-forget.
	var logEntries []LogEntry

	if s.logReader != nil {
		entries, logErr := s.logReader.ReadAndClear(dataDir)
		if logErr == nil {
			logEntries = entries
		}
	}

	// Step 2: List memories for recency surfacing.
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, nil, fmt.Errorf("surface: %w", err)
	}

	// Take top N by recency (already sorted by retriever).
	count := len(memories)
	if count > sessionStartLimit {
		count = sessionStartLimit
		memories = memories[:count]
	}

	// Nothing to surface at all.
	if len(logEntries) == 0 && count == 0 {
		return Result{}, nil, nil
	}

	var (
		summaryBuf strings.Builder
		contextBuf strings.Builder
	)

	writeCreationSection(&summaryBuf, &contextBuf, logEntries)
	writeRecencySection(&summaryBuf, &contextBuf, memories[:count], effectiveness)

	return Result{
		Summary: strings.TrimRight(summaryBuf.String(), "\n"),
		Context: contextBuf.String(),
	}, memories, nil
}

func (s *Surfacer) runTool(
	ctx context.Context,
	opts Options,
	effectiveness map[string]EffectivenessStat,
) (Result, []*memory.Stored, error) {
	memories, err := s.retriever.ListMemories(ctx, opts.DataDir)
	if err != nil {
		return Result{}, nil, fmt.Errorf("surface: %w", err)
	}

	candidates := matchToolMemories(opts.ToolName, opts.ToolInput, memories)
	if len(candidates) == 0 {
		return Result{}, nil, nil
	}

	var (
		summaryBuf strings.Builder
		contextBuf strings.Builder
	)

	_, _ = fmt.Fprintf(&summaryBuf, "[engram] %d tool advisories:\n", len(candidates))
	_, _ = fmt.Fprintf(&contextBuf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(&contextBuf, "[engram] Tool call advisory:\n")

	toolMems := make([]*memory.Stored, 0, len(candidates))

	for _, match := range candidates {
		toolMems = append(toolMems, match.mem)
		annotation := formatEffectivenessAnnotation(match.mem.FilePath, effectiveness)
		line := fmt.Sprintf("  - %s%s\n",
			filenameSlug(match.mem.FilePath), annotation)
		_, _ = fmt.Fprint(&summaryBuf, line)
		_, _ = fmt.Fprint(&contextBuf, line)
	}

	_, _ = fmt.Fprintf(&contextBuf, "</system-reminder>\n")

	return Result{
		Summary: strings.TrimRight(summaryBuf.String(), "\n"),
		Context: contextBuf.String(),
	}, toolMems, nil
}

func (s *Surfacer) writeResult(w io.Writer, result Result, format string) error {
	if result.Context == "" {
		return nil
	}

	if format == FormatJSON {
		encodeErr := json.NewEncoder(w).Encode(result)
		if encodeErr != nil {
			return fmt.Errorf("surface: encoding JSON: %w", encodeErr)
		}

		return nil
	}

	_, _ = fmt.Fprint(w, result.Context)

	return nil
}

// SurfacerOption configures a Surfacer.
type SurfacerOption func(*Surfacer)

// SurfacingEventLogger logs individual memory surfacing events (ARCH-22).
type SurfacingEventLogger interface {
	LogSurfacing(memoryPath, mode string, timestamp time.Time) error
}

// WithEffectiveness sets the effectiveness computer for memory annotations (ARCH-24).
func WithEffectiveness(computer EffectivenessComputer) SurfacerOption {
	return func(s *Surfacer) { s.effectivenessComputer = computer }
}

// WithLogReader sets the creation log reader for session-start mode.
func WithLogReader(reader CreationLogReader) SurfacerOption {
	return func(s *Surfacer) { s.logReader = reader }
}

// WithSurfacingLogger sets the surfacing event logger (ARCH-22).
func WithSurfacingLogger(logger SurfacingEventLogger) SurfacerOption {
	return func(s *Surfacer) { s.surfacingLogger = logger }
}

// WithTracker sets the memory tracker for surfacing instrumentation.
func WithTracker(tracker MemoryTracker) SurfacerOption {
	return func(s *Surfacer) { s.tracker = tracker }
}

// unexported constants.
const (
	sessionStartLimit = 20
)

// promptMatch holds a memory for prompt mode.
type promptMatch struct {
	mem *memory.Stored
}

// toolMatch holds a memory for tool mode.
type toolMatch struct {
	mem *memory.Stored
}

// concatenatePromptFields builds searchable text for prompt mode.
func concatenatePromptFields(mem *memory.Stored) string {
	var parts []string

	if mem.Title != "" {
		parts = append(parts, mem.Title)
	}

	if mem.Content != "" {
		parts = append(parts, mem.Content)
	}

	if mem.Principle != "" {
		parts = append(parts, mem.Principle)
	}

	parts = append(parts, mem.Keywords...)

	parts = append(parts, mem.Concepts...)

	return strings.Join(parts, " ")
}

// concatenateToolFields builds searchable text for tool mode.
func concatenateToolFields(mem *memory.Stored) string {
	var parts []string

	if mem.Title != "" {
		parts = append(parts, mem.Title)
	}

	if mem.Principle != "" {
		parts = append(parts, mem.Principle)
	}

	if mem.AntiPattern != "" {
		parts = append(parts, mem.AntiPattern)
	}

	parts = append(parts, mem.Keywords...)

	return strings.Join(parts, " ")
}

// filenameSlug strips directory path and .toml extension from a memory file path.
func filenameSlug(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".toml")
}

// formatEffectivenessAnnotation returns a formatted annotation for a memory path,
// or "" when no effectiveness data exists for that path.
func formatEffectivenessAnnotation(
	filePath string,
	effectiveness map[string]EffectivenessStat,
) string {
	if effectiveness == nil {
		return ""
	}

	stat, ok := effectiveness[filePath]
	if !ok {
		return ""
	}

	return fmt.Sprintf(
		" (surfaced %d times, followed %d%%)",
		stat.SurfacedCount,
		int(stat.EffectivenessScore),
	)
}

// matchPromptMemories returns top 10 memories ranked by BM25 relevance to message.
// Concatenates title, content, principle, keywords, and concepts for scoring.
func matchPromptMemories(message string, memories []*memory.Stored) []promptMatch {
	// Build documents for BM25 scoring
	docs := make([]bm25.Document, 0, len(memories))
	memoryIndex := make(map[string]*memory.Stored)

	for _, mem := range memories {
		// Concatenate searchable fields
		searchText := concatenatePromptFields(mem)

		docs = append(docs, bm25.Document{
			ID:   mem.FilePath,
			Text: searchText,
		})

		memoryIndex[mem.FilePath] = mem
	}

	// Score using BM25
	scorer := bm25.New()
	scored := scorer.Score(message, docs)

	// Limit to top 10 results
	limit := 10
	if len(scored) > limit {
		scored = scored[:limit]
	}

	// Build results
	matches := make([]promptMatch, 0, len(scored))
	for _, sd := range scored {
		mem := memoryIndex[sd.ID]
		matches = append(matches, promptMatch{mem: mem})
	}

	return matches
}

// matchToolMemories returns top 5 memories with non-empty anti_pattern, ranked by BM25.
// Only considers anti-pattern memories (tier-aware per REQ-7).
// Concatenates title, principle, anti_pattern, and keywords for scoring.
func matchToolMemories(_, toolInput string, memories []*memory.Stored) []toolMatch {
	// Filter to only anti-pattern memories (enforcement candidates)
	candidates := make([]*memory.Stored, 0)

	for _, mem := range memories {
		if mem.AntiPattern != "" {
			candidates = append(candidates, mem)
		}
	}

	if len(candidates) == 0 {
		return []toolMatch{}
	}

	// Build documents for BM25 scoring
	docs := make([]bm25.Document, 0, len(candidates))
	memoryIndex := make(map[string]*memory.Stored)

	for _, mem := range candidates {
		// Concatenate searchable fields
		searchText := concatenateToolFields(mem)

		docs = append(docs, bm25.Document{
			ID:   mem.FilePath,
			Text: searchText,
		})

		memoryIndex[mem.FilePath] = mem
	}

	// Score using BM25
	scorer := bm25.New()
	scored := scorer.Score(toolInput, docs)

	// Limit to top 5 results
	limit := 5
	if len(scored) > limit {
		scored = scored[:limit]
	}

	// Build results
	matches := make([]toolMatch, 0, len(scored))
	for _, sd := range scored {
		mem := memoryIndex[sd.ID]
		matches = append(matches, toolMatch{mem: mem})
	}

	return matches
}

// writeCreationSection appends the creation report to summary and context buffers.
func writeCreationSection(summaryBuf, contextBuf *strings.Builder, entries []LogEntry) {
	if len(entries) == 0 {
		return
	}

	creationSummary := fmt.Sprintf("[engram] Created %d memories since last session:", len(entries))

	_, _ = fmt.Fprintf(summaryBuf, "%s\n", creationSummary)
	_, _ = fmt.Fprintf(contextBuf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(contextBuf, "%s\n", creationSummary)

	for _, entry := range entries {
		entryLine := fmt.Sprintf(
			"  - \"%s\" [%s] (%s)\n",
			entry.Title,
			entry.Tier,
			entry.Filename,
		)
		_, _ = fmt.Fprint(summaryBuf, entryLine)
		_, _ = fmt.Fprint(contextBuf, entryLine)
	}

	_, _ = fmt.Fprintf(contextBuf, "</system-reminder>\n")
}

// writeRecencySection appends the recency surfacing section to summary and context buffers.
func writeRecencySection(
	summaryBuf, contextBuf *strings.Builder,
	memories []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
) {
	if len(memories) == 0 {
		return
	}

	recencySummary := fmt.Sprintf("[engram] Loaded %d memories.", len(memories))

	_, _ = fmt.Fprintf(summaryBuf, "%s\n", recencySummary)
	_, _ = fmt.Fprintf(contextBuf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(contextBuf, "%s\n", recencySummary)

	for _, mem := range memories {
		annotation := formatEffectivenessAnnotation(mem.FilePath, effectiveness)
		memLine := fmt.Sprintf("  - %s%s\n", filenameSlug(mem.FilePath), annotation)
		_, _ = fmt.Fprint(summaryBuf, memLine)
		_, _ = fmt.Fprint(contextBuf, memLine)
	}

	_, _ = fmt.Fprintf(contextBuf, "</system-reminder>\n")
}
