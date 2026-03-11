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
	"sort"
	"strings"
	"time"

	"engram/internal/bm25"
	"engram/internal/contradict"
	"engram/internal/creationlog"
	"engram/internal/frecency"
	"engram/internal/memory"
	"engram/internal/signal"
)

// Exported constants.
const (
	FormatJSON       = "json"
	ModePreCompact   = "precompact"
	ModePrompt       = "prompt"
	ModeSessionStart = "session-start"
	ModeTool         = "tool"
)

// Exported variables.
var (
	ErrUnknownMode = errors.New("surface: unknown mode")
)

// ContradictionDetector detects contradicting memory pairs at surface time (UC-P1-1, ARCH-P1-2).
type ContradictionDetector interface {
	Check(ctx context.Context, candidates []*memory.Stored) ([]contradict.Pair, error)
}

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

// EnforcementReader returns the enforcement level for a registered instruction.
type EnforcementReader interface {
	GetEnforcementLevel(id string) (string, error)
}

// InvocationTokenLogger records per-invocation token counts in the surfacing event log (REQ-P4e-5).
type InvocationTokenLogger interface {
	LogInvocationTokens(mode string, tokenCount int, timestamp time.Time) error
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
	Budget    int    // token budget override (precompact mode)
}

// RegistryRecorder records surfacing events in the instruction registry (UC-23).
type RegistryRecorder interface {
	RecordSurfacing(id string) error
}

// Result holds the structured output of a surface invocation.
type Result struct {
	Summary string `json:"summary"`
	Context string `json:"context"`
}

// SignalEmitter emits maintenance signals into the proposal queue (UC-P1-1).
type SignalEmitter interface {
	Emit(s signal.Signal) error
}

// Surfacer orchestrates memory surfacing.
type Surfacer struct {
	retriever             MemoryRetriever
	tracker               MemoryTracker
	logReader             CreationLogReader
	surfacingLogger       SurfacingEventLogger
	invocationTokenLogger InvocationTokenLogger
	effectivenessComputer EffectivenessComputer
	budgetConfig          *BudgetConfig
	registry              RegistryRecorder
	enforcementReader     EnforcementReader
	contradictionDetector ContradictionDetector
	signalEmitter         SignalEmitter
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
//nolint:cyclop,funlen // orchestration function: routes modes, logs events, writes result — inherent branching
func (s *Surfacer) Run(ctx context.Context, w io.Writer, opts Options) error {
	// Fetch effectiveness data upfront (fire-and-forget on error per ARCH-6).
	var effectiveness map[string]EffectivenessStat

	if s.effectivenessComputer != nil {
		effData, effErr := s.effectivenessComputer.Aggregate()
		if effErr == nil {
			effectiveness = effData
		}
	}

	// Create frecency scorer with current time and effectiveness data (ARCH-35).
	var frecencyEff map[string]frecency.EffectivenessStat
	if effectiveness != nil {
		frecencyEff = make(map[string]frecency.EffectivenessStat, len(effectiveness))
		for path, stat := range effectiveness {
			frecencyEff[path] = frecency.EffectivenessStat{
				EffectivenessScore: stat.EffectivenessScore,
			}
		}
	}

	scorer := frecency.New(time.Now(), frecencyEff)

	var (
		result  Result
		matched []*memory.Stored
		err     error
	)

	switch opts.Mode {
	case ModeSessionStart:
		result, matched, err = s.runSessionStart(ctx, opts.DataDir, effectiveness)
	case ModePrompt:
		result, matched, err = s.runPrompt(ctx, opts.DataDir, opts.Message, effectiveness, scorer)
	case ModeTool:
		result, matched, err = s.runTool(ctx, opts, effectiveness, scorer)
	case ModePreCompact:
		budget := opts.Budget
		if budget == 0 && s.budgetConfig != nil {
			budget = s.budgetConfig.ForMode(ModePreCompact)
		}

		if budget == 0 {
			budget = DefaultPreCompactBudget
		}

		result, matched, err = s.runPreCompact(ctx, opts.DataDir, budget, effectiveness)
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

	if s.registry != nil {
		for _, mem := range matched {
			_ = s.registry.RecordSurfacing(mem.FilePath)
		}
	}

	writeErr := s.writeResult(w, result, opts.Format)
	if writeErr != nil {
		return writeErr
	}

	// REQ-P4e-5: record output token count for this invocation.
	if s.invocationTokenLogger != nil && result.Context != "" {
		tokenCount := EstimateTokens(result.Context)
		_ = s.invocationTokenLogger.LogInvocationTokens(opts.Mode, tokenCount, time.Now())
	}

	return nil
}

// enforcementLevelFor returns the enforcement level for a memory path, defaulting to "advisory".
func (s *Surfacer) enforcementLevelFor(memPath string) string {
	if s.enforcementReader == nil {
		return enforcementAdvisory
	}

	level, err := s.enforcementReader.GetEnforcementLevel(memPath)
	if err != nil {
		return enforcementAdvisory
	}

	return level
}

func (s *Surfacer) renderToolAdvisories(
	candidates []toolMatch,
	effectiveness map[string]EffectivenessStat,
) (Result, []*memory.Stored, error) {
	// Sort emphasized/reminder memories first (REQ-P6e-1: higher budget priority).
	sort.SliceStable(candidates, func(i, j int) bool {
		li := isEmphasized(s.enforcementLevelFor(candidates[i].mem.FilePath))
		lj := isEmphasized(s.enforcementLevelFor(candidates[j].mem.FilePath))

		return li && !lj
	})

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
		level := s.enforcementLevelFor(match.mem.FilePath)
		annotation := formatEffectivenessAnnotation(match.mem.FilePath, effectiveness)
		line := formatMemoryLine(filenameSlug(match.mem.FilePath), match.mem.Principle, level, annotation)
		_, _ = fmt.Fprint(&summaryBuf, line)
		_, _ = fmt.Fprint(&contextBuf, line)
	}

	_, _ = fmt.Fprintf(&contextBuf, "</system-reminder>\n")

	return Result{
		Summary: strings.TrimRight(summaryBuf.String(), "\n"),
		Context: contextBuf.String(),
	}, toolMems, nil
}

//nolint:cyclop,funlen // effectiveness filtering + budget enforcement: inherent branching
func (s *Surfacer) runPreCompact(
	ctx context.Context,
	dataDir string,
	budget int,
	effectiveness map[string]EffectivenessStat,
) (Result, []*memory.Stored, error) {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, nil, fmt.Errorf("surface: %w", err)
	}

	// Filter to memories with effectiveness >= minPreCompactEffectiveness.
	candidates := make([]*memory.Stored, 0, len(memories))

	for _, mem := range memories {
		stat, ok := effectiveness[mem.FilePath]
		if !ok || stat.EffectivenessScore < minPreCompactEffectiveness {
			continue
		}

		candidates = append(candidates, mem)
	}

	if len(candidates) == 0 {
		return Result{}, nil, nil
	}

	// Sort by effectiveness descending.
	sort.SliceStable(candidates, func(i, j int) bool {
		return effectiveness[candidates[i].FilePath].EffectivenessScore >
			effectiveness[candidates[j].FilePath].EffectivenessScore
	})

	// Apply top-5 count limit.
	if len(candidates) > preCompactLimit {
		candidates = candidates[:preCompactLimit]
	}

	// Apply token budget.
	if budget > 0 {
		accumulated := 0
		limited := make([]*memory.Stored, 0, len(candidates))

		for _, mem := range candidates {
			tokens := EstimateTokens(mem.Principle)
			if accumulated+tokens > budget {
				break
			}

			accumulated += tokens

			limited = append(limited, mem)
		}

		candidates = limited
	}

	if len(candidates) == 0 {
		return Result{}, nil, nil
	}

	const header = "[engram] Preserving top memories through compaction:"

	var (
		summaryBuf strings.Builder
		contextBuf strings.Builder
	)

	_, _ = fmt.Fprintf(&summaryBuf, "%s\n", header)
	_, _ = fmt.Fprintf(&contextBuf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(&contextBuf, "%s\n", header)

	for _, mem := range candidates {
		line := fmt.Sprintf("- %s\n", mem.Principle)
		_, _ = fmt.Fprint(&summaryBuf, line)
		_, _ = fmt.Fprint(&contextBuf, line)
	}

	_, _ = fmt.Fprintf(&contextBuf, "</system-reminder>\n")

	return Result{
		Summary: strings.TrimRight(summaryBuf.String(), "\n"),
		Context: contextBuf.String(),
	}, candidates, nil
}

//nolint:cyclop,funlen // orchestration function: BM25 filtering + suppression + budget: inherent branching
func (s *Surfacer) runPrompt(
	ctx context.Context,
	dataDir, message string,
	effectiveness map[string]EffectivenessStat,
	scorer *frecency.Scorer,
) (Result, []*memory.Stored, error) {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, nil, fmt.Errorf("surface: %w", err)
	}

	matches := matchPromptMemories(message, memories)
	if len(matches) == 0 {
		return Result{}, nil, nil
	}

	// Re-rank by frecency activation (ARCH-35).
	sortPromptMatchesByActivation(matches, scorer)

	// Limit to top promptLimit results.
	if len(matches) > promptLimit {
		matches = matches[:promptLimit]
	}

	// Apply token budget cap (ARCH-40).
	if s.budgetConfig != nil {
		budget := s.budgetConfig.ForMode(ModePrompt)
		matches = applyPromptBudget(matches, budget)
	}

	if len(matches) == 0 {
		return Result{}, nil, nil
	}

	// Post-ranking: suppress contradicting memories (UC-P1-1).
	promptMems := make([]*memory.Stored, 0, len(matches))
	for _, m := range matches {
		promptMems = append(promptMems, m.mem)
	}

	promptMems = s.suppressContradictions(ctx, promptMems)

	// Rebuild matches from suppressed set.
	suppressedPaths := make(map[string]bool, len(promptMems))
	for _, m := range promptMems {
		suppressedPaths[m.FilePath] = true
	}

	filtered := make([]promptMatch, 0, len(promptMems))

	for _, m := range matches {
		if suppressedPaths[m.mem.FilePath] {
			filtered = append(filtered, m)
		}
	}

	matches = filtered

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

	// Collect final memory list for tracking (suppression already applied above).
	finalMems := make([]*memory.Stored, 0, len(matches))
	for _, m := range matches {
		finalMems = append(finalMems, m.mem)
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
	}, finalMems, nil
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

	// Step 2: List memories for surfacing (REQ-P4e-1: effectiveness-ranked + gated).
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, nil, fmt.Errorf("surface: %w", err)
	}

	// REQ-P4e-1: gate out memories with sufficient data (>=5 surfacings) but low effectiveness (<=40%).
	memories = filterByEffectivenessGate(memories, effectiveness)

	// REQ-P4e-1: rank by effectiveness descending; insufficient-data memories use default score.
	sortByEffectivenessScore(memories, effectiveness)

	// REQ-P4e-2: take top-7.
	count := len(memories)
	if count > sessionStartLimit {
		count = sessionStartLimit
		memories = memories[:count]
	}

	// Post-ranking: suppress contradicting memories (UC-P1-1).
	memories = s.suppressContradictions(ctx, memories)
	count = len(memories)

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
	scorer *frecency.Scorer,
) (Result, []*memory.Stored, error) {
	memories, err := s.retriever.ListMemories(ctx, opts.DataDir)
	if err != nil {
		return Result{}, nil, fmt.Errorf("surface: %w", err)
	}

	candidates := matchToolMemories(opts.ToolName, opts.ToolInput, memories)
	if len(candidates) == 0 {
		return Result{}, nil, nil
	}

	// REQ-P4e-4: gate out memories with sufficient data but low effectiveness.
	candidates = filterToolMatchesByEffectivenessGate(candidates, effectiveness)
	if len(candidates) == 0 {
		return Result{}, nil, nil
	}

	// Re-rank by frecency activation (ARCH-35).
	sortToolMatchesByActivation(candidates, scorer)

	// REQ-P4e-4: limit to top-2.
	if len(candidates) > toolLimit {
		candidates = candidates[:toolLimit]
	}

	// Apply token budget cap (ARCH-40).
	if s.budgetConfig != nil {
		budget := s.budgetConfig.ForMode(ModeTool)
		candidates = applyToolBudget(candidates, budget)
	}

	if len(candidates) == 0 {
		return Result{}, nil, nil
	}

	return s.renderToolAdvisories(candidates, effectiveness)
}

// suppressContradictions runs contradiction detection on candidates and returns a filtered slice
// with lower-ranked contradicting memories removed. Emits KindContradiction signals for each
// suppressed memory. Fire-and-forget: errors from detector return candidates unchanged (UC-P1-1).
func (s *Surfacer) suppressContradictions(ctx context.Context, candidates []*memory.Stored) []*memory.Stored {
	if s.contradictionDetector == nil || len(candidates) < 2 {
		return candidates
	}

	pairs, err := s.contradictionDetector.Check(ctx, candidates)
	if err != nil || len(pairs) == 0 {
		return candidates
	}

	// Build set of suppressed file paths (lower-ranked B member of each pair).
	suppressed := make(map[string]bool, len(pairs))

	for _, pair := range pairs {
		suppressed[pair.B.FilePath] = true

		if s.signalEmitter != nil {
			_ = s.signalEmitter.Emit(signal.Signal{
				Type:       signal.TypeMaintain,
				SourceID:   pair.B.FilePath,
				SignalKind: signal.KindContradiction,
				Summary:    "contradicts " + filenameSlug(pair.A.FilePath),
			})
		}
	}

	filtered := make([]*memory.Stored, 0, len(candidates)-len(suppressed))

	for _, mem := range candidates {
		if !suppressed[mem.FilePath] {
			filtered = append(filtered, mem)
		}
	}

	return filtered
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

// WithContradictionDetector sets the contradiction detector for post-ranking suppression (UC-P1-1).
func WithContradictionDetector(d ContradictionDetector) SurfacerOption {
	return func(s *Surfacer) { s.contradictionDetector = d }
}

// WithEffectiveness sets the effectiveness computer for memory annotations (ARCH-24).
func WithEffectiveness(computer EffectivenessComputer) SurfacerOption {
	return func(s *Surfacer) { s.effectivenessComputer = computer }
}

// WithEnforcementReader sets the enforcement level reader for level-aware rendering (REQ-P6e-1).
func WithEnforcementReader(reader EnforcementReader) SurfacerOption {
	return func(s *Surfacer) { s.enforcementReader = reader }
}

// WithInvocationTokenLogger sets the invocation token logger for per-call token tracking (REQ-P4e-5).
func WithInvocationTokenLogger(logger InvocationTokenLogger) SurfacerOption {
	return func(s *Surfacer) { s.invocationTokenLogger = logger }
}

// WithLogReader sets the creation log reader for session-start mode.
func WithLogReader(reader CreationLogReader) SurfacerOption {
	return func(s *Surfacer) { s.logReader = reader }
}

// WithRegistry sets the registry recorder for surfacing events (UC-23).
func WithRegistry(recorder RegistryRecorder) SurfacerOption {
	return func(s *Surfacer) { s.registry = recorder }
}

// WithSignalEmitter sets the signal emitter for contradiction signals (UC-P1-1).
func WithSignalEmitter(e SignalEmitter) SurfacerOption {
	return func(s *Surfacer) { s.signalEmitter = e }
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
	enforcementAdvisory              = "advisory"
	enforcementEmphasizedAdvisory    = "emphasized_advisory"
	enforcementReminder              = "reminder"
	insufficientDataThreshold        = 5    // REQ-P4e-1: <5 surfacings → insufficient data, skip gating
	minEffectivenessFloor            = 40.0 // REQ-P4e-1/REQ-P4e-4: gate threshold
	minPreCompactEffectiveness       = 40.0
	minRelevanceScore                = 0.05 // DES-P4e-3: raised BM25 floor for tighter filtering
	preCompactLimit                  = 5
	promptLimit                      = 10
	sessionStartDefaultEffectiveness = 50.0 // DES-P4e-1: default for new memories
	sessionStartLimit                = 7    // REQ-P4e-2: top-7
	toolLimit                        = 2    // REQ-P4e-4: top-2
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

// effectivenessScoreFor returns the effectiveness score for a memory path.
// Memories with <insufficientDataThreshold surfacings or no data use sessionStartDefaultEffectiveness (REQ-P4e-1).
func effectivenessScoreFor(path string, effectiveness map[string]EffectivenessStat) float64 {
	if effectiveness == nil {
		return sessionStartDefaultEffectiveness
	}

	stat, ok := effectiveness[path]
	if !ok || stat.SurfacedCount < insufficientDataThreshold {
		return sessionStartDefaultEffectiveness
	}

	return stat.EffectivenessScore
}

// filenameSlug strips directory path and .toml extension from a memory file path.
func filenameSlug(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".toml")
}

// filterByEffectivenessGate removes memories with sufficient data (>=5 surfacings) and low effectiveness (<=40%).
// Memories with <5 surfacings or no data are always kept (REQ-P4e-1: insufficient data).
func filterByEffectivenessGate(memories []*memory.Stored, effectiveness map[string]EffectivenessStat) []*memory.Stored {
	filtered := make([]*memory.Stored, 0, len(memories))

	for _, mem := range memories {
		if effectiveness != nil {
			stat, ok := effectiveness[mem.FilePath]
			if ok && stat.SurfacedCount >= insufficientDataThreshold && stat.EffectivenessScore <= minEffectivenessFloor {
				continue // gated out
			}
		}

		filtered = append(filtered, mem)
	}

	return filtered
}

// filterToolMatchesByEffectivenessGate applies effectiveness gating to tool matches (REQ-P4e-4).
func filterToolMatchesByEffectivenessGate(
	candidates []toolMatch,
	effectiveness map[string]EffectivenessStat,
) []toolMatch {
	filtered := make([]toolMatch, 0, len(candidates))

	for _, candidate := range candidates {
		if effectiveness != nil {
			stat, ok := effectiveness[candidate.mem.FilePath]
			if ok && stat.SurfacedCount >= insufficientDataThreshold && stat.EffectivenessScore <= minEffectivenessFloor {
				continue // gated out
			}
		}

		filtered = append(filtered, candidate)
	}

	return filtered
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

// formatMemoryLine formats a single memory entry based on its enforcement level.
func formatMemoryLine(slug, principle, level, annotation string) string {
	switch level {
	case enforcementEmphasizedAdvisory:
		return fmt.Sprintf("  - IMPORTANT: **%s**%s\n", slug, annotation)
	case enforcementReminder:
		return fmt.Sprintf("  - REMINDER: %s — %s%s\n", slug, principle, annotation)
	default:
		return fmt.Sprintf("  - %s%s\n", slug, annotation)
	}
}

// isEmphasized reports whether the level should be prioritized in the output.
func isEmphasized(level string) bool {
	return level == enforcementEmphasizedAdvisory || level == enforcementReminder
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

	// Build results, filtering by relevance floor
	matches := make([]promptMatch, 0, len(scored))
	for _, sd := range scored {
		if sd.Score < minRelevanceScore {
			continue
		}

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

	// Build results, filtering by relevance floor
	matches := make([]toolMatch, 0, len(scored))
	for _, sd := range scored {
		if sd.Score < minRelevanceScore {
			continue
		}

		mem := memoryIndex[sd.ID]
		matches = append(matches, toolMatch{mem: mem})
	}

	return matches
}

// sortByEffectivenessScore sorts memories by effectiveness score descending (REQ-P4e-1).
// Insufficient-data memories use sessionStartDefaultEffectiveness.
func sortByEffectivenessScore(memories []*memory.Stored, effectiveness map[string]EffectivenessStat) {
	sort.SliceStable(memories, func(i, j int) bool {
		si := effectivenessScoreFor(memories[i].FilePath, effectiveness)
		sj := effectivenessScoreFor(memories[j].FilePath, effectiveness)

		return si > sj
	})
}

// sortPromptMatchesByActivation sorts prompt matches by frecency activation descending.
func sortPromptMatchesByActivation(matches []promptMatch, scorer *frecency.Scorer) {
	sort.SliceStable(matches, func(i, j int) bool {
		return scorer.Activation(toFrecencyInput(matches[i].mem)) >
			scorer.Activation(toFrecencyInput(matches[j].mem))
	})
}

// sortToolMatchesByActivation sorts tool matches by frecency activation descending.
func sortToolMatchesByActivation(matches []toolMatch, scorer *frecency.Scorer) {
	sort.SliceStable(matches, func(i, j int) bool {
		return scorer.Activation(toFrecencyInput(matches[i].mem)) >
			scorer.Activation(toFrecencyInput(matches[j].mem))
	})
}

// toFrecencyInput converts a stored memory to a frecency input.
// Tracking fields (SurfacedCount, LastSurfaced, SurfacingContexts) are no longer
// stored inline in TOMLs — they default to zero, falling back to recency-only ranking.
// Future: populate from instruction registry for full frecency scoring.
func toFrecencyInput(mem *memory.Stored) frecency.Input {
	return frecency.Input{
		UpdatedAt: mem.UpdatedAt,
		FilePath:  mem.FilePath,
	}
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
