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
	"engram/internal/frecency"
	"engram/internal/memory"
	"engram/internal/signal"
	"engram/internal/toolgate"
)

// Exported constants.
const (
	FormatJSON = "json"
	ModePrompt = "prompt"
	ModeTool   = "tool"
)

// Exported variables.
var (
	ErrUnknownMode = errors.New("surface: unknown mode")
)

// ContradictionDetector detects contradicting memory pairs at surface time (UC-P1-1, ARCH-P1-2).
type ContradictionDetector interface {
	Check(ctx context.Context, candidates []*memory.Stored) ([]contradict.Pair, error)
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
	Mode               string
	DataDir            string
	Message            string // for prompt mode
	ToolName           string // for tool mode
	ToolInput          string // for tool mode
	ToolOutput         string // for tool mode: tool result or error text
	ToolErrored        bool   // for tool mode: true if tool call failed
	Format             string // output format: "" (plain) or "json"
	Budget             int    // token budget override (precompact mode)
	TranscriptWindow   string // recent transcript text for transcript suppression (REQ-P4f-3)
	CurrentProjectSlug string // derived from data-dir for cross-project penalty
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
	surfacingLogger       SurfacingEventLogger
	invocationTokenLogger InvocationTokenLogger
	effectivenessComputer EffectivenessComputer
	budgetConfig          *BudgetConfig
	recordSurfacing       func(path string) error // UC-23: records surfacing event per memory
	enforcementReader     EnforcementReader
	contradictionDetector ContradictionDetector
	signalEmitter         SignalEmitter
	linkReader            LinkReader             // P3: spreading activation + cluster notes
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

	// Create frecency scorer with current time and corpus-wide max surfaced count (ARCH-35).
	maxSurfaced := 0
	for _, stat := range effectiveness {
		if stat.SurfacedCount > maxSurfaced {
			maxSurfaced = stat.SurfacedCount
		}
	}

	scorer := frecency.New(time.Now(), maxSurfaced)

	var (
		result            Result
		matched           []*memory.Stored
		suppressionEvents []SuppressionEvent
		err               error
	)

	switch opts.Mode {
	case ModePrompt:
		result, matched, suppressionEvents, err = s.runPrompt(
			ctx, opts.DataDir, opts.Message, opts.TranscriptWindow,
			opts.CurrentProjectSlug, effectiveness, scorer,
		)
	case ModeTool:
		result, matched, suppressionEvents, err = s.runTool(ctx, opts, effectiveness, scorer)
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

//nolint:cyclop,funlen,gocognit // BM25 filtering + suppression + budget + spreading: inherent branching
func (s *Surfacer) runPrompt(
	ctx context.Context,
	dataDir, message, transcriptWindow, currentProjectSlug string,
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

	// BM25-seeded spreading activation: neighbors of BM25 matches join the candidate pool (#374).
	if s.linkReader != nil {
		// Index all memories by file path for neighbor lookup.
		memByPath := make(map[string]*memory.Stored, len(memories))
		for _, mem := range memories {
			memByPath[mem.FilePath] = mem
		}

		// Build bm25Matches map from existing BM25 matches.
		bm25Matches := make(map[string]float64, len(matches))
		for _, match := range matches {
			bm25Matches[match.mem.FilePath] = match.bm25Score
		}

		spreading := computeSpreading(bm25Matches, s.linkReader)

		// Set spreading scores on existing BM25 matches.
		for i := range matches {
			matches[i].spreadingScore = spreading[matches[i].mem.FilePath]
		}

		// Add spreading-only neighbors not already in matches.
		existingPaths := make(map[string]bool, len(matches))
		for _, match := range matches {
			existingPaths[match.mem.FilePath] = true
		}

		for neighborPath, spreadScore := range spreading {
			if existingPaths[neighborPath] {
				continue
			}

			neighborMem, ok := memByPath[neighborPath]
			if !ok {
				continue
			}

			matches = append(matches, promptMatch{
				mem:            neighborMem,
				bm25Score:      0,
				spreadingScore: spreadScore,
			})
		}
	}

	// Re-rank by frecency activation (ARCH-35).
	sortPromptMatchesByActivation(matches, scorer, currentProjectSlug)

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

//nolint:cyclop,funlen,unparam // BM25 filtering + spreading activation + tool gate: inherent branching
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

	// BM25-seeded spreading activation: neighbors of BM25 matches join the candidate pool (#374).
	// Only anti-pattern memories qualify as tool candidates.
	if s.linkReader != nil {
		// Index anti-pattern memories by file path for neighbor lookup.
		memByPath := make(map[string]*memory.Stored, len(memories))
		for _, mem := range memories {
			if mem.AntiPattern != "" {
				memByPath[mem.FilePath] = mem
			}
		}

		// Build bm25Matches map from existing BM25 candidates.
		bm25Matches := make(map[string]float64, len(candidates))
		for _, candidate := range candidates {
			bm25Matches[candidate.mem.FilePath] = candidate.bm25Score
		}

		spreading := computeSpreading(bm25Matches, s.linkReader)

		// Set spreading scores on existing BM25 candidates.
		for i := range candidates {
			candidates[i].spreadingScore = spreading[candidates[i].mem.FilePath]
		}

		// Add spreading-only neighbors not already in candidates.
		existingPaths := make(map[string]bool, len(candidates))
		for _, candidate := range candidates {
			existingPaths[candidate.mem.FilePath] = true
		}

		for neighborPath, spreadScore := range spreading {
			if existingPaths[neighborPath] {
				continue
			}

			neighborMem, ok := memByPath[neighborPath]
			if !ok {
				continue
			}

			candidates = append(candidates, toolMatch{
				mem:            neighborMem,
				bm25Score:      0,
				spreadingScore: spreadScore,
			})
		}
	}

	// Re-rank by frecency activation (ARCH-35).
	sortToolMatchesByActivation(candidates, scorer, opts.CurrentProjectSlug)

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

// ToolGater decides whether to surface memories for a tool call.
type ToolGater interface {
	Check(commandKey string) (bool, error)
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
	coldStartBudget               = 1
	defaultEffectiveness          = 50.0 // neutral default for memories with no effectiveness data
	enforcementAdvisory           = "advisory"
	enforcementEmphasizedAdvisory = "emphasized_advisory"
	enforcementReminder           = "reminder"
	irrelevancePenaltyHalfLife    = 5
	minEffectivenessFloor         = 40.0 // gate threshold
	minRelevanceScore             = 0.05 // DES-P4e-3: raised BM25 floor for tighter filtering
	promptLimit                   = 2
	toolLimit                     = 2 // REQ-P4e-4: top-2
	unprovenBM25FloorPrompt       = 0.20
	unprovenBM25FloorTool         = 0.30
)

// promptMatch holds a memory for prompt mode.
type promptMatch struct {
	mem            *memory.Stored
	bm25Score      float64
	spreadingScore float64
}

// toolMatch holds a memory for tool mode.
type toolMatch struct {
	mem            *memory.Stored
	bm25Score      float64
	spreadingScore float64
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

// computeSpreading computes BM25-seeded spreading activation scores.
// For each BM25 match, its graph neighbors get a boost proportional to
// the match's BM25 score × link weight, normalized by linker count.
// Returns a map of memory file path → spreading score.
func computeSpreading(
	bm25Matches map[string]float64, // filePath → bm25 score
	linkReader LinkReader,
) map[string]float64 {
	if linkReader == nil {
		return nil
	}

	spreading := make(map[string]float64)
	linkerCounts := make(map[string]int)

	for matchPath, matchBM25 := range bm25Matches {
		links, err := linkReader.GetEntryLinks(matchPath)
		if err != nil {
			continue
		}

		for _, link := range links {
			spreading[link.Target] += matchBM25 * link.Weight
			linkerCounts[link.Target]++
		}
	}

	// Normalize by linker count.
	for target, count := range linkerCounts {
		if count > 0 {
			spreading[target] /= float64(count)
		}
	}

	return spreading
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
// Two-tier logic:
//   - No effectiveness data exists for path: defaultEffectiveness (50%) — neutral default
//   - Data exists: recorded EffectivenessScore
func effectivenessScoreFor(path string, effectiveness map[string]EffectivenessStat) float64 {
	if effectiveness == nil {
		return defaultEffectiveness
	}

	stat, ok := effectiveness[path]
	if !ok || stat.SurfacedCount == 0 {
		return defaultEffectiveness
	}

	return stat.EffectivenessScore
}

// filenameSlug strips directory path and .toml extension from a memory file path.
func filenameSlug(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".toml")
}

// filterToolMatchesByEffectivenessGate applies effectiveness gating to tool matches (REQ-P4e-4).
// Memories with no effectiveness data default to 50% and always pass the gate.
func filterToolMatchesByEffectivenessGate(
	candidates []toolMatch,
	effectiveness map[string]EffectivenessStat,
) []toolMatch {
	filtered := make([]toolMatch, 0, len(candidates))

	for _, candidate := range candidates {
		score := effectivenessScoreFor(candidate.mem.FilePath, effectiveness)
		if score <= minEffectivenessFloor {
			continue // gated out
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

		matches = append(matches, promptMatch{mem: mem, bm25Score: penalizedScore})
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

		matches = append(matches, toolMatch{mem: mem, bm25Score: penalizedScore})
	}

	return matches
}

// sortPromptMatchesByActivation sorts prompt matches by combined score descending.
func sortPromptMatchesByActivation(
	matches []promptMatch, scorer *frecency.Scorer, currentProjectSlug string,
) {
	sort.SliceStable(matches, func(i, j int) bool {
		gi := GenFactor(matches[i].mem.Generalizability, matches[i].mem.ProjectSlug, currentProjectSlug)
		gj := GenFactor(matches[j].mem.Generalizability, matches[j].mem.ProjectSlug, currentProjectSlug)
		si := scorer.CombinedScore(matches[i].bm25Score, matches[i].spreadingScore, gi, toFrecencyInput(matches[i].mem))
		sj := scorer.CombinedScore(matches[j].bm25Score, matches[j].spreadingScore, gj, toFrecencyInput(matches[j].mem))

		return si > sj
	})
}

// sortToolMatchesByActivation sorts tool matches by combined score descending.
func sortToolMatchesByActivation(
	matches []toolMatch, scorer *frecency.Scorer, currentProjectSlug string,
) {
	sort.SliceStable(matches, func(i, j int) bool {
		gi := GenFactor(matches[i].mem.Generalizability, matches[i].mem.ProjectSlug, currentProjectSlug)
		gj := GenFactor(matches[j].mem.Generalizability, matches[j].mem.ProjectSlug, currentProjectSlug)
		si := scorer.CombinedScore(matches[i].bm25Score, matches[i].spreadingScore, gi, toFrecencyInput(matches[i].mem))
		sj := scorer.CombinedScore(matches[j].bm25Score, matches[j].spreadingScore, gj, toFrecencyInput(matches[j].mem))

		return si > sj
	})
}

// toFrecencyInput converts a stored memory to a frecency input.
func toFrecencyInput(mem *memory.Stored) frecency.Input {
	return frecency.Input{
		SurfacedCount:     mem.SurfacedCount,
		LastSurfacedAt:    mem.LastSurfacedAt,
		UpdatedAt:         mem.UpdatedAt,
		FollowedCount:     mem.FollowedCount,
		ContradictedCount: mem.ContradictedCount,
		IgnoredCount:      mem.IgnoredCount,
		FilePath:          mem.FilePath,
	}
}
