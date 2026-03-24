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
	"slices"
	"sort"
	"strings"
	"time"

	"engram/internal/bm25"
	"engram/internal/contradict"
	"engram/internal/creationlog"
	"engram/internal/frecency"
	"engram/internal/memory"
	"engram/internal/signal"
	"engram/internal/toolgate"
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

// CrossRefChecker checks if a memory is covered by an external source such as CLAUDE.md,
// a rule file, or a skill (REQ-P4f-2). IsCoveredBySource returns (covered, source, err).
type CrossRefChecker interface {
	IsCoveredBySource(memoryID string) (covered bool, source string, err error)
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

// LinkGraphLink represents a typed link in the memory graph (P3).
type LinkGraphLink struct {
	Target           string
	Weight           float64
	Basis            string
	CoSurfacingCount int
}

// LinkReader reads memory graph links for spreading activation and cluster notes (P3).
type LinkReader interface {
	GetEntryLinks(id string) ([]LinkGraphLink, error)
}

// LinkUpdater reads and updates memory graph links (P3: co_surfacing, spreading activation).
type LinkUpdater interface {
	GetEntryLinks(id string) ([]LinkGraphLink, error)
	SetEntryLinks(id string, links []LinkGraphLink) error
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
	Mode             string
	DataDir          string
	Message          string // for prompt mode
	ToolName         string // for tool mode
	ToolInput        string // for tool mode
	ToolOutput       string // for tool mode: tool result or error text
	ToolErrored      bool   // for tool mode: true if tool call failed
	Format           string // output format: "" (plain) or "json"
	Budget           int    // token budget override (precompact mode)
	TranscriptWindow string // recent transcript text for transcript suppression (REQ-P4f-3)
}

// Result holds the structured output of a surface invocation.
type Result struct {
	Summary          string            `json:"summary"`
	Context          string            `json:"context"`
	SuppressionStats *SuppressionStats `json:"suppressionStats,omitempty"`
}

// SignalEmitter emits maintenance signals into the proposal queue (UC-P1-1).
type SignalEmitter interface {
	Emit(s signal.Signal) error
}

// SuppressionEventLogger records suppression decisions (REQ-P4f-4).
type SuppressionEventLogger interface {
	LogSuppression(event SuppressionEvent) error
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
	recordSurfacing       func(path string) error // UC-23: records surfacing event per memory
	enforcementReader     EnforcementReader
	contradictionDetector ContradictionDetector
	signalEmitter         SignalEmitter
	linkUpdater           LinkUpdater            // P3: co_surfacing + spreading activation
	linkReader            LinkReader             // P3: spreading activation + cluster notes
	titleFetcher          TitleFetcher           // P3: cluster notes
	clusterDedupReader    LinkReader             // P4f: cluster dedup (separate from spreading activation)
	crossRefChecker       CrossRefChecker        // P4f: cross-source suppression
	suppressionLogger     SuppressionEventLogger // P4f: suppression event logging
	toolGate              ToolGater              // frecency gate for tool calls
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
		result            Result
		matched           []*memory.Stored
		suppressionEvents []SuppressionEvent
		err               error
	)

	switch opts.Mode {
	case ModeSessionStart:
		result, matched, suppressionEvents, err = s.runSessionStart(ctx, opts.DataDir, effectiveness, opts)
	case ModePrompt:
		result, matched, suppressionEvents, err = s.runPrompt(
			ctx, opts.DataDir, opts.Message, opts.TranscriptWindow, effectiveness, scorer,
		)
	case ModeTool:
		result, matched, suppressionEvents, err = s.runTool(ctx, opts, effectiveness, scorer)
	case ModePreCompact:
		budget := opts.Budget
		if budget == 0 && s.budgetConfig != nil {
			budget = s.budgetConfig.ForMode(ModePreCompact)
		}

		if budget == 0 {
			budget = DefaultPreCompactBudget
		}

		result, matched, suppressionEvents, err = s.runPreCompact(ctx, opts.DataDir, budget, effectiveness)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownMode, opts.Mode)
	}

	if err != nil {
		return err
	}

	// REQ-P4f-4: log all suppression events.
	if s.suppressionLogger != nil {
		for _, event := range suppressionEvents {
			_ = s.suppressionLogger.LogSuppression(event)
		}
	}

	// REQ-P4f-5: compute suppression rate metric.
	result.SuppressionStats = computeSuppressionStats(len(suppressionEvents), len(matched))

	if s.tracker != nil && len(matched) > 0 {
		_ = s.tracker.RecordSurfacing(ctx, matched, opts.Mode)
	}

	if s.surfacingLogger != nil {
		now := time.Now()
		for _, mem := range matched {
			_ = s.surfacingLogger.LogSurfacing(mem.FilePath, opts.Mode, now)
		}
	}

	if s.recordSurfacing != nil {
		for _, mem := range matched {
			_ = s.recordSurfacing(mem.FilePath)
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

// appendClusterNotes adds top-linked memory titles as cluster annotations (P3, REQ-P3-7).
func (s *Surfacer) appendClusterNotes(buf *strings.Builder, memories []*memory.Stored) {
	const maxClusterNotes = 2

	for _, mem := range memories {
		links, err := s.linkReader.GetEntryLinks(mem.FilePath)
		if err != nil || len(links) == 0 {
			continue
		}

		// Sort by weight descending to pick top-2
		sorted := make([]LinkGraphLink, len(links))
		copy(sorted, links)
		slices.SortFunc(sorted, func(a, b LinkGraphLink) int {
			if a.Weight > b.Weight {
				return -1
			}

			if a.Weight < b.Weight {
				return 1
			}

			return 0
		})

		noteCount := 0

		for _, link := range sorted {
			if noteCount >= maxClusterNotes {
				break
			}

			title, ok := s.titleFetcher.GetTitle(link.Target)
			if !ok {
				continue
			}

			_, _ = fmt.Fprintf(buf, "  • see also: %s\n", title)
			noteCount++
		}
	}
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

//nolint:unparam // error return is always nil — kept for uniform internal signature
func (s *Surfacer) renderToolAdvisories(
	candidates []toolMatch,
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
	_, _ = fmt.Fprintf(&contextBuf, "[engram] Memories — for any relevant memory, call "+
		"`engram show --name <name>` for full details. "+
		"After your turn, call `engram feedback --name <name> --relevant|--irrelevant "+
		"--used|--notused` for each:\n")

	toolMems := make([]*memory.Stored, 0, len(candidates))

	for _, match := range candidates {
		toolMems = append(toolMems, match.mem)
		level := s.enforcementLevelFor(match.mem.FilePath)
		line := formatMemoryLine(
			filenameSlug(match.mem.FilePath),
			match.mem.Principle,
			level,
		)
		_, _ = fmt.Fprint(&summaryBuf, line)
		_, _ = fmt.Fprint(&contextBuf, line)
	}

	_, _ = fmt.Fprintf(&contextBuf, "</system-reminder>\n")

	return Result{
		Summary: strings.TrimRight(summaryBuf.String(), "\n"),
		Context: contextBuf.String(),
	}, toolMems, nil
}

//nolint:cyclop,funlen,unparam // effectiveness filtering + budget enforcement; []SuppressionEvent always nil here
func (s *Surfacer) runPreCompact(
	ctx context.Context,
	dataDir string,
	budget int,
	effectiveness map[string]EffectivenessStat,
) (Result, []*memory.Stored, []SuppressionEvent, error) {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, nil, nil, fmt.Errorf("surface: %w", err)
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
		return Result{}, nil, nil, nil
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
		return Result{}, nil, nil, nil
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
	}, candidates, nil, nil
}

//nolint:cyclop,funlen // orchestration function: BM25 filtering + suppression + budget: inherent branching
func (s *Surfacer) runPrompt(
	ctx context.Context,
	dataDir, message, transcriptWindow string,
	effectiveness map[string]EffectivenessStat,
	scorer *frecency.Scorer,
) (Result, []*memory.Stored, []SuppressionEvent, error) {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, nil, nil, fmt.Errorf("surface: %w", err)
	}

	matches := matchPromptMemories(message, memories, effectiveness)
	if len(matches) == 0 {
		return Result{}, nil, nil, nil
	}

	// Re-rank by frecency activation (ARCH-35).
	sortPromptMatchesByActivation(matches, scorer)

	// #307: cold-start budget — limit unproven to 1 per invocation.
	matches = applyColdStartBudgetPrompt(matches, effectiveness)

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
		return Result{}, nil, nil, nil
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
		return Result{}, nil, nil, nil
	}

	// Collect mem slice for P4f suppression passes.
	promptMems = make([]*memory.Stored, 0, len(matches))
	for _, m := range matches {
		promptMems = append(promptMems, m.mem)
	}

	// REQ-P4f-2: cross-source suppression.
	promptMems, crossEvents := suppressByCrossRef(promptMems, s.crossRefChecker)

	// REQ-P4f-3: transcript suppression.
	promptMems, transcriptEvents := suppressByTranscript(promptMems, transcriptWindow)

	suppressionEvents := make([]SuppressionEvent, 0, len(crossEvents)+len(transcriptEvents))
	suppressionEvents = append(suppressionEvents, crossEvents...)
	suppressionEvents = append(suppressionEvents, transcriptEvents...)

	// Rebuild matches from post-suppression set.
	keptPaths := make(map[string]bool, len(promptMems))

	for _, m := range promptMems {
		keptPaths[m.FilePath] = true
	}

	finalMatches := make([]promptMatch, 0, len(promptMems))

	for _, m := range matches {
		if keptPaths[m.mem.FilePath] {
			finalMatches = append(finalMatches, m)
		}
	}

	matches = finalMatches

	if len(matches) == 0 {
		return Result{}, nil, suppressionEvents, nil
	}

	var buf strings.Builder

	_, _ = fmt.Fprintf(&buf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(&buf, "[engram] Memories — for any relevant memory, call "+
		"`engram show --name <name>` for full details. "+
		"After your turn, call `engram feedback --name <name> --relevant|--irrelevant "+
		"--used|--notused` for each:\n")

	for _, match := range matches {
		_, _ = fmt.Fprintf(&buf, "  - %s: %s\n",
			filenameSlug(match.mem.FilePath), match.mem.Principle)
	}

	_, _ = fmt.Fprintf(&buf, "</system-reminder>\n")

	// Collect final memory list for tracking.
	finalMems := make([]*memory.Stored, 0, len(matches))
	for _, m := range matches {
		finalMems = append(finalMems, m.mem)
	}

	var summaryBuf strings.Builder

	_, _ = fmt.Fprintf(&summaryBuf, "[engram] %d relevant memories:\n", len(matches))

	for _, match := range matches {
		_, _ = fmt.Fprintf(&summaryBuf, "  - %s: %s\n",
			filenameSlug(match.mem.FilePath), match.mem.Principle)
	}

	return Result{
		Summary: strings.TrimRight(summaryBuf.String(), "\n"),
		Context: buf.String(),
	}, finalMems, suppressionEvents, nil
}

//nolint:cyclop,funlen // session-start orchestration: reads log, loads memories, ranks, P3+P4f — inherent branching
func (s *Surfacer) runSessionStart(
	ctx context.Context,
	dataDir string,
	effectiveness map[string]EffectivenessStat,
	opts Options,
) (Result, []*memory.Stored, []SuppressionEvent, error) {
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
		return Result{}, nil, nil, fmt.Errorf("surface: %w", err)
	}

	// REQ-P4e-1: gate out memories with sufficient data (>=5 surfacings) but low effectiveness (<=40%).
	memories = filterByEffectivenessGate(memories, effectiveness)

	// REQ-P4e-1: rank by effectiveness descending; insufficient-data memories use default score.
	sortByEffectivenessScore(memories, effectiveness)

	// REQ-P3-6: re-rank using spreading activation if link reader is available.
	if s.linkReader != nil {
		activated := applySpreadingActivation(memories, effectiveness, s.linkReader)
		sortByActivatedScore(memories, activated)
	}

	// #307: cold-start budget — limit unproven to 1 per invocation.
	memories = applyColdStartBudgetStored(memories, effectiveness)

	// REQ-P4e-2: take top-7.
	count := len(memories)
	if count > sessionStartLimit {
		count = sessionStartLimit
		memories = memories[:count]
	}

	// Post-ranking: suppress contradicting memories (UC-P1-1).
	memories = s.suppressContradictions(ctx, memories)

	// REQ-P4f-1/2/3: cluster dedup, cross-source, transcript suppression passes.
	memories, dedupEvents := suppressClusterDuplicates(memories, effectiveness, s.clusterDedupReader)
	memories, crossEvents := suppressByCrossRef(memories, s.crossRefChecker)
	memories, transcriptEvents := suppressByTranscript(memories, opts.TranscriptWindow)

	suppressionEvents := make([]SuppressionEvent, 0, len(dedupEvents)+len(crossEvents)+len(transcriptEvents))
	suppressionEvents = append(suppressionEvents, dedupEvents...)
	suppressionEvents = append(suppressionEvents, crossEvents...)
	suppressionEvents = append(suppressionEvents, transcriptEvents...)

	count = len(memories)

	// Nothing to surface at all.
	if len(logEntries) == 0 && count == 0 {
		return Result{}, nil, suppressionEvents, nil
	}

	var (
		summaryBuf strings.Builder
		contextBuf strings.Builder
	)

	writeCreationSection(&summaryBuf, &contextBuf, logEntries)
	writeRecencySection(&summaryBuf, &contextBuf, memories[:count])

	// P3: update co_surfacing links for memories surfaced together (REQ-P3-5).
	if s.linkUpdater != nil && count > 0 {
		s.updateCoSurfacingLinks(memories[:count])
	}

	// P3: annotate context with cluster notes from linked memories (REQ-P3-7).
	if s.linkReader != nil && s.titleFetcher != nil && count > 0 {
		s.appendClusterNotes(&contextBuf, memories[:count])
	}

	return Result{
		Summary: strings.TrimRight(summaryBuf.String(), "\n"),
		Context: contextBuf.String(),
	}, memories, suppressionEvents, nil
}

//nolint:unparam // []SuppressionEvent always nil here — suppression not yet applied in tool mode
func (s *Surfacer) runTool(
	ctx context.Context,
	opts Options,
	effectiveness map[string]EffectivenessStat,
	scorer *frecency.Scorer,
) (Result, []*memory.Stored, []SuppressionEvent, error) {
	// Defense-in-depth: non-Bash tools should not reach here (shell filters first).
	if opts.ToolName != "Bash" {
		return Result{}, nil, nil, nil
	}

	// Frecency gate: extract command key, check counter, maybe skip.
	if !s.toolGateAllows(opts.ToolInput) {
		return Result{}, nil, nil, nil
	}

	memories, err := s.retriever.ListMemories(ctx, opts.DataDir)
	if err != nil {
		return Result{}, nil, nil, fmt.Errorf("surface: %w", err)
	}

	candidates := matchToolMemories(
		opts.ToolName, opts.ToolInput, opts.ToolOutput, opts.ToolErrored, memories, effectiveness,
	)
	if len(candidates) == 0 {
		return Result{}, nil, nil, nil
	}

	// REQ-P4e-4: gate out memories with sufficient data but low effectiveness.
	candidates = filterToolMatchesByEffectivenessGate(candidates, effectiveness)
	if len(candidates) == 0 {
		return Result{}, nil, nil, nil
	}

	// Re-rank by frecency activation (ARCH-35).
	sortToolMatchesByActivation(candidates, scorer)

	// #307: cold-start budget — limit unproven to 1 per invocation.
	candidates = applyColdStartBudgetTool(candidates, effectiveness)

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
		return Result{}, nil, nil, nil
	}

	result, mems, err := s.renderToolAdvisories(candidates)

	return result, mems, nil, err
}

// suppressContradictions runs contradiction detection on candidates and returns a filtered slice
// with lower-ranked contradicting memories removed. Emits KindContradiction signals for each
// suppressed memory. Fire-and-forget: errors from detector return candidates unchanged (UC-P1-1).
func (s *Surfacer) suppressContradictions(
	ctx context.Context,
	candidates []*memory.Stored,
) []*memory.Stored {
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

// toolGateAllows returns true if the frecency gate permits surfacing for the given tool input.
// Returns true when no gate is set or the command key is empty (fail-open).
func (s *Surfacer) toolGateAllows(toolInput string) bool {
	if s.toolGate == nil {
		return true
	}

	key := toolgate.CommandKey(toolgate.ExtractBashCommand(toolInput))
	if key == "" {
		return true
	}

	shouldSurface, _ := s.toolGate.Check(key)

	return shouldSurface
}

// updateCoSurfacingLinks increments co_surfacing link weights for all pairs in the surfaced set (P3, REQ-P3-5).
func (s *Surfacer) updateCoSurfacingLinks(memories []*memory.Stored) {
	const (
		coSurfacingInitWeight = 0.1
		coSurfacingIncrement  = 0.1
		coSurfacingWeightCap  = 1.0
	)

	for i, mem := range memories {
		existing, err := s.linkUpdater.GetEntryLinks(mem.FilePath)
		if err != nil {
			continue
		}

		updated := make([]LinkGraphLink, 0, len(existing))
		updated = append(updated, existing...)

		for j, peer := range memories {
			if i == j {
				continue
			}

			found := false

			for linkIdx, link := range updated {
				if link.Target == peer.FilePath {
					updated[linkIdx].CoSurfacingCount++
					updated[linkIdx].Weight += coSurfacingIncrement

					if updated[linkIdx].Weight > coSurfacingWeightCap {
						updated[linkIdx].Weight = coSurfacingWeightCap
					}

					found = true

					break
				}
			}

			if !found {
				updated = append(updated, LinkGraphLink{
					Target:           peer.FilePath,
					Weight:           coSurfacingInitWeight,
					Basis:            "co_surfacing",
					CoSurfacingCount: 1,
				})
			}
		}

		_ = s.linkUpdater.SetEntryLinks(mem.FilePath, updated)
	}
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

// TitleFetcher fetches memory titles for cluster notes (P3).
type TitleFetcher interface {
	GetTitle(id string) (string, bool)
}

// ToolGater decides whether to surface memories for a tool call.
type ToolGater interface {
	Check(commandKey string) (bool, error)
}

// WithClusterDedupReader sets the link reader used for cluster dedup (REQ-P4f-1).
// Separate from WithLinkReader to avoid interfering with spreading activation.
func WithClusterDedupReader(reader LinkReader) SurfacerOption {
	return func(s *Surfacer) { s.clusterDedupReader = reader }
}

// WithContradictionDetector sets the contradiction detector for post-ranking suppression (UC-P1-1).
func WithContradictionDetector(d ContradictionDetector) SurfacerOption {
	return func(s *Surfacer) { s.contradictionDetector = d }
}

// WithCrossRefChecker sets the cross-reference checker for cross-source suppression (REQ-P4f-2).
func WithCrossRefChecker(checker CrossRefChecker) SurfacerOption {
	return func(s *Surfacer) { s.crossRefChecker = checker }
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

// WithLinkReader sets the link reader for spreading activation and cluster notes (P3, REQ-P3-6, REQ-P3-7).
func WithLinkReader(reader LinkReader) SurfacerOption {
	return func(s *Surfacer) { s.linkReader = reader }
}

// WithLinkUpdater sets the link updater for co_surfacing and spreading activation (P3, REQ-P3-5, REQ-P3-6).
func WithLinkUpdater(updater LinkUpdater) SurfacerOption {
	return func(s *Surfacer) { s.linkUpdater = updater }
}

// WithLogReader sets the creation log reader for session-start mode.
func WithLogReader(reader CreationLogReader) SurfacerOption {
	return func(s *Surfacer) { s.logReader = reader }
}

// WithSignalEmitter sets the signal emitter for contradiction signals (UC-P1-1).
func WithSignalEmitter(e SignalEmitter) SurfacerOption {
	return func(s *Surfacer) { s.signalEmitter = e }
}

// WithSuppressionEventLogger sets the suppression event logger (REQ-P4f-4).
func WithSuppressionEventLogger(logger SuppressionEventLogger) SurfacerOption {
	return func(s *Surfacer) { s.suppressionLogger = logger }
}

// WithSurfacingLogger sets the surfacing event logger (ARCH-22).
func WithSurfacingLogger(logger SurfacingEventLogger) SurfacerOption {
	return func(s *Surfacer) { s.surfacingLogger = logger }
}

// WithSurfacingRecorder sets a function called once per surfaced memory to record the event (UC-23).
func WithSurfacingRecorder(fn func(path string) error) SurfacerOption {
	return func(s *Surfacer) { s.recordSurfacing = fn }
}

// WithTitleFetcher sets the title fetcher for cluster notes (P3, REQ-P3-7).
func WithTitleFetcher(fetcher TitleFetcher) SurfacerOption {
	return func(s *Surfacer) { s.titleFetcher = fetcher }
}

// WithToolGate sets the tool frecency gate.
func WithToolGate(gate ToolGater) SurfacerOption {
	return func(s *Surfacer) { s.toolGate = gate }
}

// WithTracker sets the memory tracker for surfacing instrumentation.
func WithTracker(tracker MemoryTracker) SurfacerOption {
	return func(s *Surfacer) { s.tracker = tracker }
}

// unexported constants.
const (
	coldStartBudget                  = 1
	enforcementAdvisory              = "advisory"
	enforcementEmphasizedAdvisory    = "emphasized_advisory"
	enforcementReminder              = "reminder"
	insufficientDataThreshold        = 5 // REQ-P4e-1: <5 surfacings → insufficient data, skip gating
	irrelevancePenaltyHalfLife       = 5
	minEffectivenessFloor            = 40.0 // REQ-P4e-1/REQ-P4e-4: gate threshold
	minPreCompactEffectiveness       = 40.0
	minRelevanceScore                = 0.05 // DES-P4e-3: raised BM25 floor for tighter filtering
	preCompactLimit                  = 5
	promptLimit                      = 2
	sessionStartDefaultEffectiveness = 50.0 // DES-P4e-1: default for memories with 1-4 surfacings
	sessionStartLimit                = 7    // REQ-P4e-2: top-7
	spreadingActivationDecay         = 0.3  // REQ-P3-6: spreading activation decay factor
	toolLimit                        = 2    // REQ-P4e-4: top-2
	unprovenBM25FloorPrompt          = 0.20
	unprovenBM25FloorTool            = 0.30
	unprovenDefaultEffectiveness     = 30.0
)

// promptMatch holds a memory for prompt mode.
type promptMatch struct {
	mem *memory.Stored
}

// toolMatch holds a memory for tool mode.
type toolMatch struct {
	mem *memory.Stored
}

// applyColdStartBudgetPrompt keeps all proven matches plus at most coldStartBudget unproven.
// No-op when effectiveness is nil (no data available to distinguish proven from unproven).
func applyColdStartBudgetPrompt(
	candidates []promptMatch,
	effectiveness map[string]EffectivenessStat,
) []promptMatch {
	if effectiveness == nil {
		return candidates
	}

	result := make([]promptMatch, 0, len(candidates))
	unprovenCount := 0

	for _, match := range candidates {
		if isUnproven(match.mem.FilePath, effectiveness) {
			unprovenCount++
			if unprovenCount > coldStartBudget {
				continue
			}
		}

		result = append(result, match)
	}

	return result
}

// applyColdStartBudgetStored keeps all proven memories plus at most coldStartBudget unproven.
// No-op when effectiveness is nil (no data available to distinguish proven from unproven).
func applyColdStartBudgetStored(
	candidates []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
) []*memory.Stored {
	if effectiveness == nil {
		return candidates
	}

	result := make([]*memory.Stored, 0, len(candidates))
	unprovenCount := 0

	for _, mem := range candidates {
		if isUnproven(mem.FilePath, effectiveness) {
			unprovenCount++
			if unprovenCount > coldStartBudget {
				continue
			}
		}

		result = append(result, mem)
	}

	return result
}

// applyColdStartBudgetTool keeps all proven matches plus at most coldStartBudget unproven.
// No-op when effectiveness is nil (no data available to distinguish proven from unproven).
func applyColdStartBudgetTool(
	candidates []toolMatch,
	effectiveness map[string]EffectivenessStat,
) []toolMatch {
	if effectiveness == nil {
		return candidates
	}

	result := make([]toolMatch, 0, len(candidates))
	unprovenCount := 0

	for _, match := range candidates {
		if isUnproven(match.mem.FilePath, effectiveness) {
			unprovenCount++
			if unprovenCount > coldStartBudget {
				continue
			}
		}

		result = append(result, match)
	}

	return result
}

// applySpreadingActivation re-scores memories using the P3 spreading activation formula (REQ-P3-6):
//
//	activated[id] = base[id] + 0.3 × Σ(base[linked_id] × link.Weight)
//
// Only linked memories in the candidate set contribute to the spread term.
// Returns a map from FilePath to activated score.
func applySpreadingActivation(
	memories []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
	linkReader LinkReader,
) map[string]float64 {
	// Build base score index.
	base := make(map[string]float64, len(memories))

	for _, mem := range memories {
		base[mem.FilePath] = effectivenessScoreFor(mem.FilePath, effectiveness)
	}

	activated := make(map[string]float64, len(memories))

	for _, mem := range memories {
		spread := 0.0

		links, err := linkReader.GetEntryLinks(mem.FilePath)
		if err == nil {
			for _, link := range links {
				if linkedBase, ok := base[link.Target]; ok {
					spread += linkedBase * link.Weight
				}
			}
		}

		activated[mem.FilePath] = base[mem.FilePath] + spreadingActivationDecay*spread
	}

	return activated
}

// bm25FloorForTool returns the BM25 relevance floor for a tool-mode memory.
// Unproven memories get a higher floor unless the tool errored.
func bm25FloorForTool(
	memID string,
	errored bool,
	effectiveness map[string]EffectivenessStat,
) float64 {
	if isUnproven(memID, effectiveness) && !errored {
		return unprovenBM25FloorTool
	}

	return minRelevanceScore
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
// Unproven memories (0 surfacings) get unprovenDefaultEffectiveness (30%).
// Memories with 1-4 surfacings (insufficient data) get sessionStartDefaultEffectiveness (50%).
// Memories with >=insufficientDataThreshold surfacings use their recorded score (REQ-P4e-1).
func effectivenessScoreFor(path string, effectiveness map[string]EffectivenessStat) float64 {
	if effectiveness == nil {
		return unprovenDefaultEffectiveness
	}

	stat, ok := effectiveness[path]
	if !ok || stat.SurfacedCount == 0 {
		return unprovenDefaultEffectiveness
	}

	if stat.SurfacedCount < insufficientDataThreshold {
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
func filterByEffectivenessGate(
	memories []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
) []*memory.Stored {
	filtered := make([]*memory.Stored, 0, len(memories))

	for _, mem := range memories {
		if effectiveness != nil {
			stat, ok := effectiveness[mem.FilePath]
			if ok && stat.SurfacedCount >= insufficientDataThreshold &&
				stat.EffectivenessScore <= minEffectivenessFloor {
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
			if ok && stat.SurfacedCount >= insufficientDataThreshold &&
				stat.EffectivenessScore <= minEffectivenessFloor {
				continue // gated out
			}
		}

		filtered = append(filtered, candidate)
	}

	return filtered
}

// formatMemoryLine formats a single memory entry based on its enforcement level.
func formatMemoryLine(slug, principle, level string) string {
	switch level {
	case enforcementEmphasizedAdvisory:
		return fmt.Sprintf("  - IMPORTANT: **%s: %s**\n", slug, principle)
	case enforcementReminder:
		return fmt.Sprintf("  - REMINDER: %s: %s\n", slug, principle)
	default:
		return fmt.Sprintf("  - %s: %s\n", slug, principle)
	}
}

// irrelevancePenalty computes a continuous BM25 score multiplier based on irrelevant feedback count.
// Uses the formula K/(K+count) where K=irrelevancePenaltyHalfLife (5).
// At count=0 → 1.0, count=5 → 0.5, count=10 → 0.33.
func irrelevancePenalty(irrelevantCount int) float64 {
	return float64(irrelevancePenaltyHalfLife) /
		float64(irrelevancePenaltyHalfLife+irrelevantCount)
}

// isEmphasized reports whether the level should be prioritized in the output.
func isEmphasized(level string) bool {
	return level == enforcementEmphasizedAdvisory || level == enforcementReminder
}

// isUnproven reports whether a memory has never been surfaced (cold-start).
func isUnproven(path string, effectiveness map[string]EffectivenessStat) bool {
	if effectiveness == nil {
		return true
	}

	stat, ok := effectiveness[path]

	return !ok || stat.SurfacedCount == 0
}

// matchPromptMemories returns top 10 memories ranked by BM25 relevance to message.
// Concatenates title, content, principle, keywords, and concepts for scoring.
// Unproven memories (never surfaced) require a higher BM25 floor than proven ones.
func matchPromptMemories(
	message string,
	memories []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
) []promptMatch {
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

	// Build results, filtering by relevance floor.
	// Unproven memories require a higher floor to avoid cold-start noise.
	// Apply irrelevance penalty before floor comparison (#343).
	matches := make([]promptMatch, 0, len(scored))
	for _, result := range scored {
		mem, ok := memoryIndex[result.ID]
		if !ok || mem == nil {
			continue
		}

		penalizedScore := result.Score * irrelevancePenalty(mem.IrrelevantCount)

		floor := minRelevanceScore
		if isUnproven(result.ID, effectiveness) {
			floor = unprovenBM25FloorPrompt
		}

		if penalizedScore < floor {
			continue
		}

		matches = append(matches, promptMatch{mem: mem})
	}

	return matches
}

// matchToolMemories returns top 5 memories with non-empty anti_pattern, ranked by BM25.
// Only considers anti-pattern memories (tier-aware per REQ-7).
// Concatenates title, principle, anti_pattern, and keywords for scoring.
// Unproven memories (never surfaced) require a higher BM25 floor than proven ones.
func matchToolMemories(
	_, toolInput, toolOutput string,
	errored bool,
	memories []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
) []toolMatch {
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

	// Score using BM25; enrich query with tool output if present.
	query := toolInput
	if toolOutput != "" {
		query = toolInput + " " + toolOutput
	}

	scorer := bm25.New()
	scored := scorer.Score(query, docs)

	// Build results, filtering by relevance floor.
	// Unproven memories require a higher floor to avoid cold-start noise,
	// unless the tool errored — any relevant memory is high-value on failure.
	// Apply irrelevance penalty before floor comparison (#343).
	matches := make([]toolMatch, 0, len(scored))
	for _, result := range scored {
		mem, ok := memoryIndex[result.ID]
		if !ok || mem == nil {
			continue
		}

		penalizedScore := result.Score * irrelevancePenalty(mem.IrrelevantCount)

		if penalizedScore < bm25FloorForTool(result.ID, errored, effectiveness) {
			continue
		}

		matches = append(matches, toolMatch{mem: mem})
	}

	return matches
}

// sortByActivatedScore sorts memories by activated score descending (REQ-P3-6).
func sortByActivatedScore(memories []*memory.Stored, scores map[string]float64) {
	sort.SliceStable(memories, func(i, j int) bool {
		return scores[memories[i].FilePath] > scores[memories[j].FilePath]
	})
}

// sortByEffectivenessScore sorts memories by effectiveness score descending (REQ-P4e-1).
// Insufficient-data memories use sessionStartDefaultEffectiveness.
func sortByEffectivenessScore(
	memories []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
) {
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
) {
	if len(memories) == 0 {
		return
	}

	recencySummary := fmt.Sprintf("[engram] Loaded %d memories.", len(memories))

	_, _ = fmt.Fprintf(summaryBuf, "%s\n", recencySummary)
	_, _ = fmt.Fprintf(contextBuf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(contextBuf, "%s\n", recencySummary)

	for _, mem := range memories {
		memLine := fmt.Sprintf("  - %s: %s\n", filenameSlug(mem.FilePath), mem.Principle)
		_, _ = fmt.Fprint(summaryBuf, memLine)
		_, _ = fmt.Fprint(contextBuf, memLine)
	}

	_, _ = fmt.Fprintf(contextBuf, "</system-reminder>\n")
}
