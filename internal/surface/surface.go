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
	"engram/internal/memory"
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
	TranscriptWindow   string // recent transcript text for transcript suppression
	CurrentProjectSlug string // derived from data-dir for cross-project penalty
	SessionID          string // session ID for pending evaluation tracking
	UserPrompt         string // original user prompt for pending evaluation
}

// Result holds the structured output of a surface invocation.
type Result struct {
	Summary          string            `json:"summary"`
	Context          string            `json:"context"`
	SuppressionStats *SuppressionStats `json:"suppressionStats,omitempty"`
}

// Surfacer orchestrates memory surfacing.
type Surfacer struct {
	retriever             MemoryRetriever
	tracker               MemoryTracker
	surfacingLogger       SurfacingEventLogger
	invocationTokenLogger InvocationTokenLogger
	budgetConfig          *BudgetConfig
	config                SurfaceConfig
	recordSurfacing       func(path string) error // UC-23: records surfacing event per memory
	haikuGate             HaikuCallerFunc
	pendingEvalModifier   ModifyFunc
}

// New creates a Surfacer.
func New(retriever MemoryRetriever, opts ...SurfacerOption) *Surfacer {
	s := &Surfacer{
		retriever: retriever,
		config:    DefaultSurfaceConfig(),
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
	var (
		result  Result
		matched []*memory.Stored
		err     error
	)

	switch opts.Mode {
	case ModePrompt:
		result, matched, err = s.runPrompt(ctx, opts.DataDir, opts.Message, opts.TranscriptWindow,
			opts.CurrentProjectSlug)
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

	if s.recordSurfacing != nil {
		for _, mem := range matched {
			_ = s.recordSurfacing(mem.FilePath)
		}
	}

	if s.pendingEvalModifier != nil && len(matched) > 0 {
		_ = WritePendingEvaluations(
			matched, s.pendingEvalModifier,
			opts.SessionID, opts.CurrentProjectSlug, opts.UserPrompt,
			time.Now(),
		)
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

//nolint:cyclop,funlen // BM25 filtering + budget: inherent branching
func (s *Surfacer) runPrompt(
	ctx context.Context,
	dataDir, message, transcriptWindow, currentProjectSlug string,
) (Result, []*memory.Stored, error) {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, nil, fmt.Errorf("surface: %w", err)
	}

	matches := matchPromptMemories(message, memories, s.config.IrrelevanceHalfLife)

	// Apply BM25 threshold filter.
	if s.config.BM25Threshold > 0 {
		filtered := make([]promptMatch, 0, len(matches))
		for _, m := range matches {
			if m.bm25Score >= s.config.BM25Threshold {
				filtered = append(filtered, m)
			}
		}

		matches = filtered
	}

	if len(matches) == 0 {
		return Result{}, nil, nil
	}

	// Sort by BM25 score with project scope penalty.
	sortPromptMatchesByScore(matches, currentProjectSlug)

	// Limit to top CandidateCountMax results.
	if len(matches) > s.config.CandidateCountMax {
		matches = matches[:s.config.CandidateCountMax]
	}

	// Apply token budget cap (ARCH-40).
	if s.budgetConfig != nil {
		budget := s.budgetConfig.ForMode(ModePrompt)
		matches = applyPromptBudget(matches, budget)
	}

	if len(matches) == 0 {
		return Result{}, nil, nil
	}

	// Suppress memories mentioned in recent transcript.
	promptMems := make([]*memory.Stored, 0, len(matches))
	for _, m := range matches {
		promptMems = append(promptMems, m.mem)
	}

	// Apply cold-start budget for unproven memories.
	promptMems = ApplyColdStartBudget(promptMems, s.config.ColdStartBudget)

	// Apply Haiku semantic gate if configured.
	if s.haikuGate != nil && s.config.GateHaikuPrompt != "" {
		promptMems, _ = GateMemories(
			ctx, promptMems, message, s.haikuGate, s.config.GateHaikuPrompt,
		)
	}

	promptMems, _ = suppressByTranscript(promptMems, transcriptWindow)

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
		return Result{}, nil, nil
	}

	var buf strings.Builder

	_, _ = fmt.Fprintf(&buf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(&buf, "%s\n", s.config.InjectionPreamble)

	for i, match := range matches {
		slug := filenameSlug(match.mem.FilePath)
		_, _ = fmt.Fprintf(&buf, "  %d. %s\n", i+1, slug)
		_, _ = fmt.Fprintf(&buf, "     Situation: %s\n", match.mem.Situation)
		_, _ = fmt.Fprintf(&buf, "     Behavior to avoid: %s\n", match.mem.Behavior)
		_, _ = fmt.Fprintf(&buf, "     Impact if ignored: %s\n", match.mem.Impact)
		_, _ = fmt.Fprintf(&buf, "     Action: %s\n", match.mem.Action)
	}

	_, _ = fmt.Fprintf(&buf, "</system-reminder>\n")

	var summaryBuf strings.Builder

	_, _ = fmt.Fprintf(&summaryBuf, "[engram] %d relevant memories:\n", len(matches))

	for _, match := range matches {
		_, _ = fmt.Fprintf(&summaryBuf, "  - %s: %s\n",
			filenameSlug(match.mem.FilePath), match.mem.Action)
	}

	return Result{
		Summary: strings.TrimRight(summaryBuf.String(), "\n"),
		Context: buf.String(),
	}, promptMems, nil
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

// WithEffectiveness sets the effectiveness computer (stubbed during SBIA migration).
func WithEffectiveness(_ EffectivenessComputer) SurfacerOption {
	return func(_ *Surfacer) {}
}

// WithInvocationTokenLogger sets the invocation token logger (REQ-P4e-5).
func WithInvocationTokenLogger(logger InvocationTokenLogger) SurfacerOption {
	return func(s *Surfacer) { s.invocationTokenLogger = logger }
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

// promptMatch holds a memory for prompt mode.
type promptMatch struct {
	mem        *memory.Stored
	bm25Score  float64
	searchText string // cached SearchText() result
}

// filenameSlug strips directory path and .toml extension from a memory file path.
func filenameSlug(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".toml")
}

// irrelevancePenalty computes a continuous BM25 score multiplier based on irrelevant feedback count.
func irrelevancePenalty(irrelevantCount, halfLife int) float64 {
	return float64(halfLife) / float64(halfLife+irrelevantCount)
}

// matchPromptMemories returns top memories ranked by BM25 relevance to message.
func matchPromptMemories(
	message string,
	memories []*memory.Stored,
	halfLife int,
) []promptMatch {
	docs := make([]bm25.Document, 0, len(memories))
	memoryIndex := make(map[string]*memory.Stored, len(memories))
	searchTextIndex := make(map[string]string, len(memories))

	for _, mem := range memories {
		text := mem.SearchText()

		docs = append(docs, bm25.Document{
			ID:   mem.FilePath,
			Text: text,
		})

		memoryIndex[mem.FilePath] = mem
		searchTextIndex[mem.FilePath] = text
	}

	scorer := bm25.New()
	scored := scorer.Score(message, docs)

	matches := make([]promptMatch, 0, len(scored))
	for _, result := range scored {
		mem, ok := memoryIndex[result.ID]
		if !ok || mem == nil {
			continue
		}

		penalizedScore := result.Score * irrelevancePenalty(mem.IrrelevantCount, halfLife)
		matches = append(matches, promptMatch{
			mem:        mem,
			bm25Score:  penalizedScore,
			searchText: searchTextIndex[result.ID],
		})
	}

	return matches
}

// sortPromptMatchesByScore sorts prompt matches by BM25 score with project scope penalty.
func sortPromptMatchesByScore(
	matches []promptMatch, currentProjectSlug string,
) {
	sort.SliceStable(matches, func(i, j int) bool {
		gi := GenFactor(matches[i].mem.ProjectScoped, matches[i].mem.ProjectSlug, currentProjectSlug)
		gj := GenFactor(matches[j].mem.ProjectScoped, matches[j].mem.ProjectSlug, currentProjectSlug)
		si := matches[i].bm25Score * gi
		sj := matches[j].bm25Score * gj

		return si > sj
	})
}
