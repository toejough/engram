package remind_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/remind"
)

// selectBest returns error when all instruction loads fail.
func TestRun_AllLoadersFail(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"bad-id-1", "bad-id-2"},
		},
	}
	loader := &fakeLoader{
		err: errors.New("memory corrupted"),
	}
	transcript := &fakeTranscript{text: ""}

	r := remind.New(config, loader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	_, err := r.Run(ctx, input)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("resolving instructions")))
}

// selectBest returns empty when all principles are empty (no error).
func TestRun_AllPrinciplesEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"empty-1", "empty-2"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"empty-1": "",
			"empty-2": "",
		},
	}
	transcript := &fakeTranscript{text: ""}

	r := remind.New(config, loader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

// T-209: Reminder capped at 100 tokens.
func TestRun_CappedAt100Tokens(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create a principle that is ~200 tokens (800 chars).
	longPrinciple := strings.Repeat("word ", 160) // 800 chars = 200 tokens

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"verbose-rule"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"verbose-rule": longPrinciple,
		},
	}
	transcript := &fakeTranscript{text: ""}

	estimator := func(text string) int { return len(text) / 4 }

	r := remind.New(config, loader, transcript,
		remind.WithEstimateTokens(estimator),
	)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Edit",
		FilePath: "internal/bar.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Strip prefix to measure reminder body.
	body := strings.TrimPrefix(result, "[engram] Reminder: ")
	tokens := len(body) / 4
	g.Expect(tokens).To(BeNumerically("<=", 100))
}

// T-214 variant: Config read error returns no output (graceful).
func TestRun_ConfigReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		err: errors.New("file not found"),
	}
	loader := &fakeLoader{principles: map[string]string{}}
	transcript := &fakeTranscript{text: ""}

	r := remind.New(config, loader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	_, err := r.Run(ctx, input)
	g.Expect(err).To(HaveOccurred())
}

// T-211: Reminder emitted when no compliance evidence.
func TestRun_EmittedWhenNoCompliance(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"targ-rule"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"targ-rule": "use targ not go test",
		},
	}
	// Transcript has no compliance evidence.
	transcript := &fakeTranscript{text: "just chatting about nothing relevant"}

	r := remind.New(config, loader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HavePrefix("[engram] Reminder: "))
	g.Expect(result).To(ContainSubstring("targ"))
}

// T-207: Glob pattern matches file path to instruction set.
func TestRun_GlobPatternMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"go-conventions"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"go-conventions": "Use targ for all build operations",
		},
	}
	transcript := &fakeTranscript{text: ""}
	effectiveness := &fakeEffectiveness{
		scores: map[string]float64{"go-conventions": 0.8},
	}

	r := remind.New(config, loader, transcript,
		remind.WithEffectiveness(effectiveness),
	)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("[engram] Reminder:"))
	g.Expect(result).To(ContainSubstring("targ"))
}

// Test: Multiple patterns, highest effectiveness wins.
func TestRun_HighestEffectivenessSelected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"low-score", "high-score"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"low-score":  "low priority rule",
			"high-score": "high priority rule",
		},
	}
	transcript := &fakeTranscript{text: ""}
	effectiveness := &fakeEffectiveness{
		scores: map[string]float64{
			"low-score":  0.2,
			"high-score": 0.9,
		},
	}

	r := remind.New(config, loader, transcript,
		remind.WithEffectiveness(effectiveness),
	)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("high priority rule"))
}

// T-212: Reminder logged to surfacing log for effectiveness.
func TestRun_LoggedToSurfacingLog(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"go-conventions"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"go-conventions": "Use targ for all build operations",
		},
	}
	transcript := &fakeTranscript{text: ""}

	logger := &fakeLogger{}

	r := remind.New(config, loader, transcript,
		remind.WithSurfacingLogger(logger),
	)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeEmpty())
	g.Expect(logger.events).To(HaveLen(1))
	g.Expect(logger.events[0].memoryPath).To(Equal("go-conventions"))
	g.Expect(logger.events[0].mode).To(Equal("PostToolUse"))
}

// T-214: Missing reminders.toml produces no output.
func TestRun_MissingConfig(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: nil,
		err:      nil, // No error, just empty config.
	}
	loader := &fakeLoader{principles: map[string]string{}}
	transcript := &fakeTranscript{text: ""}

	r := remind.New(config, loader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

// T-208: No pattern match returns empty output.
func TestRun_NoPatternMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"go-conventions"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"go-conventions": "Use targ for all build operations",
		},
	}
	transcript := &fakeTranscript{text: ""}

	r := remind.New(config, loader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "README.md",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

// selectBest: partial loader failure, one succeeds — uses the successful one.
func TestRun_PartialLoaderFailure(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	loadCount := 0

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"fail-id", "good-id"},
		},
	}
	// Custom loader: first call fails, second succeeds.
	customLoader := &selectiveLoader{
		results: map[string]loaderResult{
			"fail-id": {err: errors.New("not found")},
			"good-id": {principle: "use gofmt always"},
		},
	}
	_ = loadCount
	transcript := &fakeTranscript{text: ""}

	r := remind.New(config, customLoader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("gofmt"))
}

// selectBest skips empty principles and picks the non-empty one.
func TestRun_SkipsEmptyPrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"empty-rule", "real-rule"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"empty-rule": "",
			"real-rule":  "always use gofmt",
		},
	}
	transcript := &fakeTranscript{text: ""}

	r := remind.New(config, loader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("gofmt"))
}

// T-210: Suppression when model already complied.
func TestRun_SuppressionWhenComplied(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"targ-rule"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"targ-rule": "use targ not go test",
		},
	}
	// Transcript contains evidence of compliance.
	transcript := &fakeTranscript{text: "running targ test ./..."}

	r := remind.New(config, loader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	result, err := r.Run(ctx, input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

// selectBest: transcript read error propagates.
func TestRun_TranscriptReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config := &fakeConfig{
		patterns: map[string][]string{
			"*.go": {"rule-1"},
		},
	}
	loader := &fakeLoader{
		principles: map[string]string{
			"rule-1": "always use gofmt",
		},
	}
	transcript := &fakeTranscript{err: errors.New("transcript read failed")}

	r := remind.New(config, loader, transcript)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "Write",
		FilePath: "internal/foo.go",
	}

	_, err := r.Run(ctx, input)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("checking suppression")))
}

// Test fakes.

type fakeConfig struct {
	patterns map[string][]string
	err      error
}

func (f *fakeConfig) ReadConfig() (map[string][]string, error) {
	return f.patterns, f.err
}

type fakeEffectiveness struct {
	scores map[string]float64
}

func (f *fakeEffectiveness) Score(id string) float64 {
	return f.scores[id]
}

type fakeLoader struct {
	principles map[string]string
	err        error
}

func (f *fakeLoader) LoadPrinciple(_ context.Context, id string) (string, error) {
	if f.err != nil {
		return "", f.err
	}

	return f.principles[id], nil
}

type fakeLogger struct {
	events []logEvent
}

func (f *fakeLogger) LogSurfacing(memoryPath, mode string, _ time.Time) error {
	f.events = append(f.events, logEvent{memoryPath: memoryPath, mode: mode})

	return nil
}

type fakeTranscript struct {
	text string
	err  error
}

func (f *fakeTranscript) ReadRecent(_ int) (string, error) {
	return f.text, f.err
}

type loaderResult struct {
	principle string
	err       error
}

type logEvent struct {
	memoryPath string
	mode       string
}

type selectiveLoader struct {
	results map[string]loaderResult
}

func (s *selectiveLoader) LoadPrinciple(_ context.Context, id string) (string, error) {
	r, ok := s.results[id]
	if !ok {
		return "", nil
	}

	return r.principle, r.err
}
