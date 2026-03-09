// Package remind implements UC-18 PostToolUse proactive reminders (ARCH-46).
// Matches file paths to instruction sets via glob patterns, selects the
// highest-effectiveness instruction, checks suppression, and emits reminders.
package remind

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ConfigReader reads the reminders configuration (pattern → instruction IDs).
type ConfigReader interface {
	ReadConfig() (map[string][]string, error)
}

// EffectivenessProvider returns the effectiveness score for a memory ID.
type EffectivenessProvider interface {
	Score(instructionID string) float64
}

// MemoryLoader loads a memory's principle text by instruction ID.
type MemoryLoader interface {
	LoadPrinciple(ctx context.Context, instructionID string) (string, error)
}

// Option configures a Reminder.
type Option func(*Reminder)

// Reminder orchestrates PostToolUse proactive reminders.
type Reminder struct {
	config         ConfigReader
	loader         MemoryLoader
	transcript     TranscriptReader
	logger         SurfacingLogger
	effectiveness  EffectivenessProvider
	estimateTokens func(string) int
}

// New creates a Reminder with the given dependencies.
func New(
	config ConfigReader,
	loader MemoryLoader,
	transcript TranscriptReader,
	opts ...Option,
) *Reminder {
	r := &Reminder{
		config:         config,
		loader:         loader,
		transcript:     transcript,
		estimateTokens: defaultEstimateTokens,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Run executes the reminder pipeline for a tool call.
// Returns the reminder text or empty string if no reminder applies.
//

func (r *Reminder) Run(ctx context.Context, input ToolCallInput) (string, error) {
	// Step 1: Load config.
	patterns, err := r.config.ReadConfig()
	if err != nil {
		return "", fmt.Errorf("remind: reading config: %w", err)
	}

	if len(patterns) == 0 {
		return "", nil
	}

	// Step 2: Match glob patterns against file path.
	instructionIDs := r.matchPatterns(patterns, input.FilePath)
	if len(instructionIDs) == 0 {
		return "", nil
	}

	// Step 3: Resolve instructions and select highest effectiveness.
	bestID, principle, resolveErr := r.selectBest(ctx, instructionIDs)
	if resolveErr != nil {
		return "", fmt.Errorf("remind: resolving instructions: %w", resolveErr)
	}

	if principle == "" {
		return "", nil
	}

	// Step 4: Check suppression via keyword matching on recent transcript.
	suppressed, suppressErr := r.isSuppressed(principle)
	if suppressErr != nil {
		return "", fmt.Errorf("remind: checking suppression: %w", suppressErr)
	}

	if suppressed {
		return "", nil
	}

	// Step 5: Cap at budget.
	reminder := r.capTokens(principle)

	// Step 6: Format output.
	output := "[engram] Reminder: " + reminder

	// Step 7: Log surfacing event.
	if r.logger != nil {
		_ = r.logger.LogSurfacing(bestID, "PostToolUse", time.Now())
	}

	return output, nil
}

// capTokens truncates text to fit within the reminder token budget.
func (r *Reminder) capTokens(text string) string {
	if r.estimateTokens(text) <= maxReminderTokens {
		return text
	}

	// Truncate by characters (4 chars ≈ 1 token).
	maxChars := maxReminderTokens * estimateTokensDivisor
	if len(text) > maxChars {
		text = text[:maxChars]
	}

	return text
}

// isSuppressed checks if the instruction's principle text appears in recent transcript.
func (r *Reminder) isSuppressed(principle string) (bool, error) {
	if r.transcript == nil {
		return false, nil
	}

	recent, err := r.transcript.ReadRecent(suppressionTranscriptTokens)
	if err != nil {
		return false, err
	}

	// Keyword matching: check if any significant word from principle appears in transcript.
	words := extractKeywords(principle)
	recentLower := strings.ToLower(recent)

	for _, word := range words {
		if strings.Contains(recentLower, strings.ToLower(word)) {
			return true, nil
		}
	}

	return false, nil
}

// matchPatterns returns deduplicated instruction IDs for patterns matching filePath.
func (r *Reminder) matchPatterns(
	patterns map[string][]string,
	filePath string,
) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for pattern, ids := range patterns {
		matched, matchErr := filepath.Match(pattern, filepath.Base(filePath))
		if matchErr != nil {
			continue
		}

		if !matched {
			continue
		}

		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				result = append(result, id)
			}
		}
	}

	return result
}

// selectBest resolves instruction IDs and returns the one with highest effectiveness.
func (r *Reminder) selectBest(
	ctx context.Context,
	instructionIDs []string,
) (string, string, error) {
	var (
		bestID        string
		bestPrinciple string
		bestScore     float64
		firstErr      error
	)

	initialized := false

	for _, id := range instructionIDs {
		principle, loadErr := r.loader.LoadPrinciple(ctx, id)
		if loadErr != nil {
			if firstErr == nil {
				firstErr = loadErr
			}

			continue
		}

		if principle == "" {
			continue
		}

		score := float64(0)
		if r.effectiveness != nil {
			score = r.effectiveness.Score(id)
		}

		if !initialized || score > bestScore {
			bestID = id
			bestPrinciple = principle
			bestScore = score
			initialized = true
		}
	}

	if !initialized && firstErr != nil {
		return "", "", firstErr
	}

	return bestID, bestPrinciple, nil
}

// SurfacingLogger logs surfacing events for effectiveness tracking (ARCH-22).
type SurfacingLogger interface {
	LogSurfacing(memoryPath, mode string, timestamp time.Time) error
}

// ToolCallInput holds the context of a PostToolUse hook invocation.
type ToolCallInput struct {
	ToolName string
	FilePath string
}

// TranscriptReader reads recent transcript text for suppression checks.
type TranscriptReader interface {
	ReadRecent(maxTokens int) (string, error)
}

// WithEffectiveness sets the effectiveness provider.
func WithEffectiveness(provider EffectivenessProvider) Option {
	return func(r *Reminder) { r.effectiveness = provider }
}

// WithEstimateTokens overrides the token estimator (default: len/4).
func WithEstimateTokens(fn func(string) int) Option {
	return func(r *Reminder) { r.estimateTokens = fn }
}

// WithSurfacingLogger sets the surfacing logger.
func WithSurfacingLogger(logger SurfacingLogger) Option {
	return func(r *Reminder) { r.logger = logger }
}

// unexported constants.
const (
	estimateTokensDivisor       = 4
	maxReminderTokens           = 100
	minKeywordLength            = 4
	suppressionTranscriptTokens = 500
)

func defaultEstimateTokens(text string) int {
	return len(text) / estimateTokensDivisor
}

// extractKeywords splits principle into words of 4+ characters for suppression matching.
func extractKeywords(text string) []string {
	words := strings.Fields(text)
	result := make([]string, 0, len(words))

	for _, word := range words {
		cleaned := strings.Trim(word, ".,;:!?\"'()[]{}—-")
		if len(cleaned) >= minKeywordLength {
			result = append(result, cleaned)
		}
	}

	return result
}
