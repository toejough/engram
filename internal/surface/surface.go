// Package surface implements memory surfacing for UC-2 (ARCH-12).
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
)

// Exported constants.
const (
	FormatJSON = "json"
	ModePrompt = "prompt"
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

// InvocationTokenLogger records per-invocation token counts in the surfacing event log (REQ-P4e-5).
type InvocationTokenLogger interface {
	LogInvocationTokens(mode string, tokenCount int, timestamp time.Time) error
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
	Format             string // output format: "" (plain) or "json"
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

	contradictionDetector ContradictionDetector
	signalEmitter         SignalEmitter
	crossRefChecker       CrossRefChecker        // P4f: cross-source suppression
	suppressionLogger     SuppressionEventLogger // P4f: suppression event logging
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

//nolint:cyclop,funlen // BM25 filtering + suppression + budget: inherent branching
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

	matches := matchPromptMemories(message, memories)
	if len(matches) == 0 {
		return Result{}, nil, nil, nil
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

// WithCrossRefChecker sets the cross-reference checker for cross-source suppression (REQ-P4f-2).
func WithCrossRefChecker(checker CrossRefChecker) SurfacerOption {
	return func(s *Surfacer) { s.crossRefChecker = checker }
}

// WithEffectiveness sets the effectiveness computer for memory annotations (ARCH-24).
func WithEffectiveness(computer EffectivenessComputer) SurfacerOption {
	return func(s *Surfacer) { s.effectivenessComputer = computer }
}

// WithInvocationTokenLogger sets the invocation token logger for per-call token tracking (REQ-P4e-5).
func WithInvocationTokenLogger(logger InvocationTokenLogger) SurfacerOption {
	return func(s *Surfacer) { s.invocationTokenLogger = logger }
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

// WithTracker sets the memory tracker for surfacing instrumentation.
func WithTracker(tracker MemoryTracker) SurfacerOption {
	return func(s *Surfacer) { s.tracker = tracker }
}

// unexported constants.
const (
	coldStartBudget            = 2
	defaultEffectiveness       = 50.0 // neutral default for memories with no effectiveness data
	irrelevancePenaltyHalfLife = 5
	promptLimit                = 2
)

// promptMatch holds a memory for prompt mode.
type promptMatch struct {
	mem       *memory.Stored
	bm25Score float64
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

// irrelevancePenalty computes a continuous BM25 score multiplier based on irrelevant feedback count.
// Uses the formula K/(K+count) where K=irrelevancePenaltyHalfLife (5).
// At count=0 → 1.0, count=5 → 0.5, count=10 → 0.33.
func irrelevancePenalty(irrelevantCount int) float64 {
	return float64(irrelevancePenaltyHalfLife) /
		float64(irrelevancePenaltyHalfLife+irrelevantCount)
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
func matchPromptMemories(
	message string,
	memories []*memory.Stored,
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

	// Apply irrelevance penalty and collect all matches (#343).
	matches := make([]promptMatch, 0, len(scored))
	for _, result := range scored {
		mem, ok := memoryIndex[result.ID]
		if !ok || mem == nil {
			continue
		}

		penalizedScore := result.Score * irrelevancePenalty(mem.IrrelevantCount)
		matches = append(matches, promptMatch{mem: mem, bm25Score: penalizedScore})
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
		si := scorer.CombinedScore(matches[i].bm25Score, gi, toFrecencyInput(matches[i].mem))
		sj := scorer.CombinedScore(matches[j].bm25Score, gj, toFrecencyInput(matches[j].mem))

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
		Tier:              mem.Confidence,
	}
}
