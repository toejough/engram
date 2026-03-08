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

// Test fakes.

type fakeConfig struct {
	patterns map[string][]string
	err      error
}

func (f *fakeConfig) ReadConfig() (map[string][]string, error) {
	return f.patterns, f.err
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

type fakeTranscript struct {
	text string
	err  error
}

func (f *fakeTranscript) ReadRecent(_ int) (string, error) {
	return f.text, f.err
}

type fakeLogger struct {
	events []logEvent
}

type logEvent struct {
	memoryPath string
	mode       string
}

func (f *fakeLogger) LogSurfacing(memoryPath, mode string, _ time.Time) error {
	f.events = append(f.events, logEvent{memoryPath: memoryPath, mode: mode})

	return nil
}

type fakeEffectiveness struct {
	scores map[string]float64
}

func (f *fakeEffectiveness) Score(id string) float64 {
	return f.scores[id]
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
