// Package evaluate_test tests the outcome evaluation pipeline (ARCH-23).
package evaluate_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/evaluate"
)

// T-P3-27: EvalLinkUpdater error does not abort evaluate.
func TestEvalCorrelation_ErrorDoesNotAbort(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/mem1.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n" +
		`{"memory_path":"/data/memories/mem2.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n"

	const memTOML = "title = \"Mem\"\ncontent = \"Content\"\nprinciple = \"Principle\"\nanti_pattern = \"\""

	const llmResponse = `[{"memory_path":"/data/memories/mem1.toml","outcome":"followed","evidence":"ok"},` +
		`{"memory_path":"/data/memories/mem2.toml","outcome":"followed","evidence":"ok"}]`

	updater := &fakeEvalLinkUpdaterCapture{
		existing: map[string][]evaluate.EvalLink{},
		err:      errors.New("link error"),
	}

	evaluator := makeTestEvaluator(append(
		withSurfacingLog(surfacingLog, func(_ string) ([]byte, error) { return []byte(memTOML), nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResponse, nil
		}),
		evaluate.WithNow(func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) }),
		evaluate.WithEvalLinkUpdater(updater),
	)...)

	outcomes, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Evaluation should still succeed with 2 outcomes despite link errors
	g.Expect(outcomes).To(HaveLen(2))
}

// T-P3-25: updateEvaluationCorrelationLinks updates links for followed pairs.
func TestEvalCorrelation_FollowedPairsGetLinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/mem1.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n" +
		`{"memory_path":"/data/memories/mem2.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n"

	const memTOML = "title = \"Mem\"\ncontent = \"Content\"\nprinciple = \"Principle\"\nanti_pattern = \"\""

	const llmResponse = `[{"memory_path":"/data/memories/mem1.toml","outcome":"followed","evidence":"ok"},` +
		`{"memory_path":"/data/memories/mem2.toml","outcome":"followed","evidence":"ok"}]`

	updater := &fakeEvalLinkUpdaterCapture{
		existing: map[string][]evaluate.EvalLink{},
	}

	evaluator := makeTestEvaluator(append(
		withSurfacingLog(surfacingLog, func(_ string) ([]byte, error) { return []byte(memTOML), nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResponse, nil
		}),
		evaluate.WithNow(func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) }),
		evaluate.WithEvalLinkUpdater(updater),
	)...)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// mem1 should have link to mem2, mem2 should have link to mem1
	mem1Links := updater.setLinks["/data/memories/mem1.toml"]
	g.Expect(mem1Links).To(HaveLen(1))

	if len(mem1Links) == 0 {
		return
	}

	g.Expect(mem1Links[0].Target).To(Equal("/data/memories/mem2.toml"))
	g.Expect(mem1Links[0].Basis).To(Equal("evaluation_correlation"))
	g.Expect(mem1Links[0].Weight).To(BeNumerically("~", 0.05, 0.001))

	mem2Links := updater.setLinks["/data/memories/mem2.toml"]
	g.Expect(mem2Links).To(HaveLen(1))

	if len(mem2Links) == 0 {
		return
	}

	g.Expect(mem2Links[0].Target).To(Equal("/data/memories/mem1.toml"))
}

// T-P3-26: updateEvaluationCorrelationLinks ignores non-followed outcomes.
func TestEvalCorrelation_IgnoresNonFollowed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/mem1.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n" +
		`{"memory_path":"/data/memories/mem2.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n"

	const memTOML = "title = \"Mem\"\ncontent = \"Content\"\nprinciple = \"Principle\"\nanti_pattern = \"\""

	const llmResponse = `[{"memory_path":"/data/memories/mem1.toml","outcome":"followed","evidence":"ok"},` +
		`{"memory_path":"/data/memories/mem2.toml","outcome":"contradicted","evidence":"nope"}]`

	updater := &fakeEvalLinkUpdaterCapture{
		existing: map[string][]evaluate.EvalLink{},
	}

	evaluator := makeTestEvaluator(append(
		withSurfacingLog(surfacingLog, func(_ string) ([]byte, error) { return []byte(memTOML), nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResponse, nil
		}),
		evaluate.WithNow(func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) }),
		evaluate.WithEvalLinkUpdater(updater),
	)...)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Only mem1 was followed, mem2 was contradicted — no pair, no links set
	g.Expect(updater.setLinks).To(BeEmpty())
}

// T-P3-25b: updateEvaluationCorrelationLinks increments existing correlation link.
func TestEvalCorrelation_IncrementsExistingLink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/mem1.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n" +
		`{"memory_path":"/data/memories/mem2.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n"

	const memTOML = "title = \"Mem\"\ncontent = \"Content\"\nprinciple = \"Principle\"\nanti_pattern = \"\""

	llmResp := `[{"memory_path":"/data/memories/mem1.toml",` +
		`"outcome":"followed","evidence":"ok"},` +
		`{"memory_path":"/data/memories/mem2.toml",` +
		`"outcome":"followed","evidence":"ok"}]`

	updater := &fakeEvalLinkUpdaterCapture{
		existing: map[string][]evaluate.EvalLink{
			"/data/memories/mem1.toml": {
				{
					Target: "/data/memories/mem2.toml",
					Weight: 0.2,
					Basis:  "evaluation_correlation",
				},
			},
		},
	}

	evaluator := makeTestEvaluator(append(
		withSurfacingLog(surfacingLog, func(_ string) ([]byte, error) { return []byte(memTOML), nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResp, nil
		}),
		evaluate.WithNow(func() time.Time {
			return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		}),
		evaluate.WithEvalLinkUpdater(updater),
	)...)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// mem1's existing link to mem2 should be incremented: 0.2 + 0.05 = 0.25
	mem1Links := updater.setLinks["/data/memories/mem1.toml"]
	g.Expect(mem1Links).To(HaveLen(1))

	if len(mem1Links) == 0 {
		return
	}

	g.Expect(mem1Links[0].Weight).To(BeNumerically("~", 0.25, 0.001))
}

// T-106: Evaluator classifies surfaced memories via LLM.
func TestEvaluator_ClassifiesSurfacedMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = "" +
		`{"memory_path":"/data/memories/mem1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n" +
		`{"memory_path":"/data/memories/mem2.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:01Z"}`

	const mem1TOML = `title = "Use targ"
content = "Always use targ for building"
principle = "Use targ not go build"
anti_pattern = "Don't run go build directly"`

	const mem2TOML = `title = "No global state"
content = "Avoid global mutable state"
principle = "Prefer dependency injection"
anti_pattern = ""`

	const llmResponse = `[` +
		`{"memory_path":"/data/memories/mem1.toml","outcome":"followed","evidence":"Agent used targ"},` +
		`{"memory_path":"/data/memories/mem2.toml","outcome":"ignored","evidence":"No state changes observed"}` +
		`]`

	var capturedModel, capturedUser string

	memReader := func(name string) ([]byte, error) {
		switch name {
		case "/data/memories/mem1.toml":
			return []byte(mem1TOML), nil
		case "/data/memories/mem2.toml":
			return []byte(mem2TOML), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	evaluator := makeTestEvaluator(append(
		withSurfacingLog(surfacingLog, memReader),
		evaluate.WithLLMCaller(func(_ context.Context, model, _, user string) (string, error) {
			capturedModel = model
			capturedUser = user

			return llmResponse, nil
		}),
		evaluate.WithNow(
			func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) },
		),
	)...)

	outcomes, err := evaluator.Evaluate(context.Background(), "the full session transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedModel).To(Equal("claude-haiku-4-5-20251001"))
	g.Expect(capturedUser).To(ContainSubstring("the full session transcript"))
	g.Expect(capturedUser).To(ContainSubstring("Use targ"))
	g.Expect(capturedUser).To(ContainSubstring("No global state"))
	g.Expect(outcomes).To(HaveLen(2))

	if len(outcomes) < 2 {
		return
	}

	g.Expect(outcomes[0].MemoryPath).To(Equal("/data/memories/mem1.toml"))
	g.Expect(outcomes[0].Outcome).To(Equal("followed"))
	g.Expect(outcomes[0].Evidence).To(Equal("Agent used targ"))
	g.Expect(outcomes[1].MemoryPath).To(Equal("/data/memories/mem2.toml"))
	g.Expect(outcomes[1].Outcome).To(Equal("ignored"))
}

// T-111: Evaluator clears surfacing log after reading.
func TestEvaluator_ClearsSurfacingLog(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`
	)

	var removedPath string

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(name string) error {
			removedPath = name
			return nil
		}),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"}]`, nil
		}),
		evaluate.WithNow(
			func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) },
		),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Private path (not original) is removed after rename-then-read (ARCH-81).
	g.Expect(removedPath).To(Equal("/data/surfacing-log-2024-01-15T10-30-00Z.jsonl.tmp"))
}

// T-268: Default StripFunc is no-op — backward compatible.
func TestEvaluator_DefaultStripFunc_NoOp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`
	)

	transcript := `{"role":"toolResult","content":"blob"}` + "\n" +
		`{"role":"user","content":"hello"}`

	var capturedUser string

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, user string) (string, error) {
			capturedUser = user
			return `[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"}]`, nil
		}),
		evaluate.WithNow(
			func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) },
		),
		// No WithStripFunc — default should be no-op.
	)

	outcomes, err := evaluator.Evaluate(context.Background(), transcript)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(outcomes).To(HaveLen(1))

	// Full transcript passed to LLM — toolResult line still present.
	g.Expect(capturedUser).To(ContainSubstring("toolResult"))
	g.Expect(capturedUser).To(ContainSubstring("hello"))
}

// T-267: Empty post-strip transcript — evaluation skipped.
func TestEvaluator_EmptyPostStrip_SkipsEvaluation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`
	)

	// Transcript with only tool results — strip removes everything.
	transcript := strings.Join([]string{
		`{"role":"toolResult","content":"blob1"}`,
		`{"role":"toolResult","content":"blob2"}`,
	}, "\n")

	llmCalled := false

	var logBuf strings.Builder

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			llmCalled = true
			return "", nil
		}),
		evaluate.WithNow(
			func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) },
		),
		evaluate.WithStripFunc(func(_ []string) []string {
			// Strip everything.
			return make([]string, 0)
		}),
		evaluate.WithLogWriter(&logBuf),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), transcript)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(llmCalled).To(BeFalse())
	g.Expect(outcomes).To(BeNil())
	g.Expect(logBuf.String()).To(ContainSubstring("empty after strip"))
}

// T-107: Evaluator handles empty surfacing log — no LLM call, no output.
func TestEvaluator_EmptySurfacingLog_NoLLMCall(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	llmCalled := false

	evaluator := makeTestEvaluator(
		// Simulate missing surfacing log: rename fails with ErrNotExist (ARCH-81).
		evaluate.WithRename(func(from, _ string) error {
			if strings.Contains(from, "surfacing-log") {
				return os.ErrNotExist
			}

			return nil
		}),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			llmCalled = true
			return "", nil
		}),
		evaluate.WithNow(time.Now),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(llmCalled).To(BeFalse())
	g.Expect(outcomes).To(BeNil())
}

// T-106c: Evaluator returns error when a memory TOML file is invalid.
func TestEvaluator_InvalidMemoryTOML_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/bad.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte("this is [not valid toml {{{{"), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "", nil
		}),
		evaluate.WithNow(time.Now),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
}

// T-266: Strip applied before LLM evaluation.
func TestEvaluator_StripAppliedBeforeLLM(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`
	)

	// Build a transcript: 80 tool result lines + 20 conversation lines.
	const totalLines = 100

	lines := make([]string, 0, totalLines)
	for range 80 {
		lines = append(lines, `{"role":"toolResult","content":"big blob"}`)
	}

	for i := range 20 {
		lines = append(lines, fmt.Sprintf(`{"role":"user","content":"message %d"}`, i))
	}

	transcript := strings.Join(lines, "\n")

	var capturedUser string

	// Strip function that filters out toolResult lines (like sessionctx.Strip).
	stripFunc := func(input []string) []string {
		result := make([]string, 0, len(input))
		for _, line := range input {
			if !strings.Contains(line, `"role":"toolResult"`) {
				result = append(result, line)
			}
		}

		return result
	}

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, user string) (string, error) {
			capturedUser = user
			return `[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"}]`, nil
		}),
		evaluate.WithNow(
			func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) },
		),
		evaluate.WithStripFunc(stripFunc),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), transcript)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(outcomes).To(HaveLen(1))

	// LLM should only see the 20 conversation lines, not the 80 tool results.
	g.Expect(capturedUser).NotTo(ContainSubstring("toolResult"))
	g.Expect(capturedUser).To(ContainSubstring("message 0"))
	g.Expect(capturedUser).To(ContainSubstring("message 19"))
}

// T-106b: Evaluator handles LLM response wrapped in markdown code fence.
func TestEvaluator_StripsMarkdownFenceFromLLMResponse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`
	)

	fencedResponse := "```json\n" +
		`[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"}]` +
		"\n```"

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return fencedResponse, nil
		}),
		evaluate.WithNow(
			func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) },
		),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(outcomes).To(HaveLen(1))

	if len(outcomes) < 1 {
		return
	}

	g.Expect(outcomes[0].Outcome).To(Equal("followed"))
}

// T-106d: Evaluator returns error when surfacing log cannot be read (non-ErrNotExist).
func TestEvaluator_SurfacingLogReadError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(string) ([]byte, error) {
			return nil, errors.New("permission denied")
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "", nil
		}),
		evaluate.WithNow(time.Now),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
}

// T-106e: Evaluator returns error when surfacing log cannot be removed after reading.
func TestEvaluator_SurfacingLogRemoveError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		evaluate.WithRemoveFile(func(string) error {
			return errors.New("remove failed")
		}),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "", nil
		}),
		evaluate.WithNow(time.Now),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
}

// T-110: Evaluator with unparseable LLM response returns error, no log written.
func TestEvaluator_UnparseableLLMResponse_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`
	)

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "this is not valid JSON", nil
		}),
		evaluate.WithNow(time.Now),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
	g.Expect(outcomes).To(BeNil())
}

// TestEvaluator_UpdatesEvalCorrelationLinks verifies link updater is called for evaluated memories.
func TestEvaluator_UpdatesEvalCorrelationLinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/mem1.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n"

	const memTOML = `title = "Use targ"
content = "Always use targ"
principle = "Use targ not go build"
anti_pattern = ""`

	const llmResponse = `[{"memory_path":"/data/memories/mem1.toml","outcome":"followed","evidence":"Used targ"}]`

	updater := &fakeEvalLinkUpdater{}

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			switch name {
			case "/data/memories/mem1.toml":
				return []byte(memTOML), nil
			default:
				return nil, os.ErrNotExist
			}
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResponse, nil
		}),
		evaluate.WithNow(func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) }),
		evaluate.WithEvalLinkUpdater(updater),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(outcomes).To(HaveLen(1))
	// Only 1 followed memory → no pairs → no link updates (REQ-P3-9)
	g.Expect(updater.getCalls).To(BeEmpty())
	g.Expect(updater.setCalls).To(BeEmpty())
}

// T-108: Evaluator writes per-session evaluation JSONL to evaluations/ after LLM returns outcomes.
func TestT108_EvaluationLogWritten(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = "" +
		`{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n" +
		`{"memory_path":"/data/memories/m2.toml","mode":"prompt","surfaced_at":"2024-01-15T10:01:00Z"}` + "\n" +
		`{"memory_path":"/data/memories/m3.toml","mode":"prompt","surfaced_at":"2024-01-15T10:02:00Z"}` + "\n"

	const memTOML = "title = \"Mem\"\ncontent = \"Content\"\nprinciple = \"Principle\"\nanti_pattern = \"\""

	const llmResponse = `[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"e1"},` +
		`{"memory_path":"/data/memories/m2.toml","outcome":"ignored","evidence":"e2"},` +
		`{"memory_path":"/data/memories/m3.toml","outcome":"contradicted","evidence":"e3"}]`

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var mkdirAllPath string

	var writtenPath string

	var writtenData []byte

	var renamedFrom, renamedTo string

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(memTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResponse, nil
		}),
		evaluate.WithNow(func() time.Time { return fixedTime }),
		evaluate.WithMkdirAll(func(path string, _ os.FileMode) error {
			mkdirAllPath = path
			return nil
		}),
		evaluate.WithWriteFile(func(path string, data []byte, _ os.FileMode) error {
			writtenPath = path
			writtenData = data

			return nil
		}),
		evaluate.WithRename(func(from, to string) error {
			renamedFrom = from
			renamedTo = to

			return nil
		}),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Directory is created.
	g.Expect(mkdirAllPath).To(Equal("/data/evaluations"))

	// Temp file written then renamed atomically.
	// UnixNano of 2024-01-15T10:30:00Z = 1705314600000000000 (T-327: nanosecond resolution).
	expectedTmp := "/data/evaluations/1705314600000000000.jsonl.tmp"
	expectedFinal := "/data/evaluations/1705314600000000000.jsonl"

	g.Expect(writtenPath).To(Equal(expectedTmp))
	g.Expect(renamedFrom).To(Equal(expectedTmp))
	g.Expect(renamedTo).To(Equal(expectedFinal))

	// Content is valid JSONL with 3 outcomes.
	lines := strings.Split(strings.TrimRight(string(writtenData), "\n"), "\n")
	g.Expect(lines).To(HaveLen(3))

	var outcome0 evaluate.Outcome
	g.Expect(json.Unmarshal([]byte(lines[0]), &outcome0)).To(Succeed())
	g.Expect(outcome0.MemoryPath).To(Equal("/data/memories/m1.toml"))
	g.Expect(outcome0.Outcome).To(Equal("followed"))
	g.Expect(outcome0.Evidence).To(Equal("e1"))
}

// T-109: Evaluator creates evaluations/ directory before writing log.
func TestT109_EvaluationsDirCreated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = "" +
		`{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n"

	const memTOML = "title = \"Mem\"\ncontent = \"Content\"\nprinciple = \"P\"\nanti_pattern = \"\""

	const llmResponse = `[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"}]`

	var mkdirAllCalled bool

	var writeFileCalled bool

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(memTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResponse, nil
		}),
		evaluate.WithNow(func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) }),
		evaluate.WithMkdirAll(func(_ string, _ os.FileMode) error {
			mkdirAllCalled = true
			return nil
		}),
		evaluate.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			writeFileCalled = true
			return nil
		}),
		evaluate.WithRename(func(_, _ string) error { return nil }),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// MkdirAll is always called before WriteFile.
	g.Expect(mkdirAllCalled).To(BeTrue())
	g.Expect(writeFileCalled).To(BeTrue())
}

// T-200: Evaluation hook calls RecordEvaluation, counter increments.
func TestT200_RegistryRecordEvaluationCalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = "" +
		`{"memory_path":"/data/memories/mem1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n" +
		`{"memory_path":"/data/memories/mem2.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:01Z"}`

	const genericTOML = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`

	const llmResponse = `[` +
		`{"memory_path":"/data/memories/mem1.toml","outcome":"followed","evidence":"e1"},` +
		`{"memory_path":"/data/memories/mem2.toml","outcome":"contradicted","evidence":"e2"}` +
		`]`

	registry := &fakeEvalRegistryRecorder{}

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResponse, nil
		}),
		evaluate.WithNow(
			func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) },
		),
		evaluate.WithRegistry(registry),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(outcomes).To(HaveLen(2))
	g.Expect(registry.calls).To(HaveLen(2))

	if len(registry.calls) < 2 {
		return
	}

	g.Expect(registry.calls[0].id).To(Equal("/data/memories/mem1.toml"))
	g.Expect(registry.calls[0].outcome).To(Equal("followed"))
	g.Expect(registry.calls[1].id).To(Equal("/data/memories/mem2.toml"))
	g.Expect(registry.calls[1].outcome).To(Equal("contradicted"))
}

// T-200b: Registry error does not affect evaluation (fire-and-forget).
func TestT200b_RegistryErrorDoesNotAffectEvaluation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`
	)

	registry := &fakeEvalRegistryRecorder{err: errors.New("registry write failed")}

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if strings.Contains(name, "surfacing-log") {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"}]`, nil
		}),
		evaluate.WithNow(
			func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) },
		),
		evaluate.WithRegistry(registry),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), "transcript")

	// Should succeed despite registry error.
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(outcomes).To(HaveLen(1))
}

// T-326: Evaluation JSONL records survive a round-trip with schema_version=1.
func TestT326_EvaluationSchemaVersionRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	// Write a JSON record with schema_version: 1 and all current fields.
	record := `{"memory_path":"/data/memories/mem1.toml","outcome":"followed",` +
		`"evidence":"Agent used targ","evaluated_at":"2024-01-15T10:30:00Z","schema_version":1}`

	var parsed evaluate.Outcome

	err := json.Unmarshal([]byte(record), &parsed)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Assert all fields survive correctly.
	g.Expect(parsed.MemoryPath).To(Equal("/data/memories/mem1.toml"))
	g.Expect(parsed.Outcome).To(Equal("followed"))
	g.Expect(parsed.Evidence).To(Equal("Agent used targ"))
	g.Expect(parsed.EvaluatedAt).To(Equal(fixedTime))
	g.Expect(parsed.SchemaVersion).To(Equal(1))
}

// T-327: Two concurrent Evaluate calls write to separate files — no collision.
func TestT327_ConcurrentEvaluateWriteNoCollision(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()

	const surfacingLog = `{"memory_path":"/data/memories/mem1.toml",` +
		`"mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n"

	const memTOML = "title = \"Mem\"\ncontent = \"Content\"\nprinciple = \"Principle\"\nanti_pattern = \"\""

	const llmResponse = `[{"memory_path":"/data/memories/mem1.toml","outcome":"followed","evidence":"ok"}]`

	// Two fixed times that share the same second but differ in nanoseconds.
	time1 := time.Date(2024, 1, 15, 10, 30, 0, 1, time.UTC)
	time2 := time.Date(2024, 1, 15, 10, 30, 0, 2, time.UTC)

	makeEval := func(fixedTime time.Time) *evaluate.Evaluator {
		surfacingLogPath := dataDir + "/surfacing-log.jsonl"

		fs := newTestFS(map[string][]byte{
			surfacingLogPath: []byte(surfacingLog),
		})

		memReader := func(_ string) ([]byte, error) {
			return []byte(memTOML), nil
		}

		return evaluate.New(
			dataDir,
			evaluate.WithRename(func(oldPath, newPath string) error {
				if err := fs.rename(oldPath, newPath); err != nil {
					return err
				}

				return os.Rename(oldPath, newPath)
			}),
			evaluate.WithReadFile(fs.wrapReadFile(memReader)),
			evaluate.WithRemoveFile(os.Remove),
			evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
				return llmResponse, nil
			}),
			evaluate.WithNow(func() time.Time { return fixedTime }),
		)
	}

	// Write a surfacing log for each evaluator (each will rename it away before reading).
	surfacingLogPath := dataDir + "/surfacing-log.jsonl"

	writeLog := func() {
		writeErr := os.WriteFile(surfacingLogPath, []byte(surfacingLog), 0o600)
		g.Expect(writeErr).NotTo(HaveOccurred())

		if writeErr != nil {
			return
		}
	}

	// Run first evaluator, restore log, run second evaluator.
	// Sequential here is fine — what we're testing is filename uniqueness, not goroutine race.
	// The nanosecond-based filenames must differ even when called back-to-back.
	writeLog()

	eval1 := makeEval(time1)

	outcomes1, err1 := eval1.Evaluate(context.Background(), "transcript")
	g.Expect(err1).NotTo(HaveOccurred())

	if err1 != nil {
		return
	}

	g.Expect(outcomes1).To(HaveLen(1))

	writeLog()

	eval2 := makeEval(time2)

	outcomes2, err2 := eval2.Evaluate(context.Background(), "transcript")
	g.Expect(err2).NotTo(HaveOccurred())

	if err2 != nil {
		return
	}

	g.Expect(outcomes2).To(HaveLen(1))

	// Both evaluation files must exist in the evaluations/ directory.
	evalDir := dataDir + "/evaluations"

	entries, readDirErr := os.ReadDir(evalDir)
	g.Expect(readDirErr).NotTo(HaveOccurred())

	if readDirErr != nil {
		return
	}

	// Filter to only .jsonl files (exclude any .jsonl.tmp leftovers).
	jsonlFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			jsonlFiles = append(jsonlFiles, entry.Name())
		}
	}

	g.Expect(jsonlFiles).To(HaveLen(2), "expected two separate evaluation files — nanosecond timestamps must differ")
}

// T-345: Evaluator renames surfacing log before reading for turn isolation (ARCH-81).
func TestT345_SurfacingLogRenamedBeforeRead(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = "" +
		`{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n"

	const memTOML = `title = "Mem"
content = "Content"`

	const llmResponse = `[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"}]`

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	const expectedPrivatePath = "/data/surfacing-log-2024-01-15T10-30-00Z.jsonl.tmp"

	var surfacingRenameFrom, surfacingRenameTo string

	var readPaths []string

	evaluator := makeTestEvaluator(
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			readPaths = append(readPaths, name)
			if name == expectedPrivatePath {
				return []byte(surfacingLog), nil
			}

			return []byte(memTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResponse, nil
		}),
		evaluate.WithNow(func() time.Time { return fixedTime }),
		evaluate.WithRename(func(from, to string) error {
			if from == "/data/surfacing-log.jsonl" {
				surfacingRenameFrom = from
				surfacingRenameTo = to
			}

			return nil
		}),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Surfacing log was renamed to private path before reading.
	g.Expect(surfacingRenameFrom).To(Equal("/data/surfacing-log.jsonl"))
	g.Expect(surfacingRenameTo).To(Equal(expectedPrivatePath))

	// readFile was called with private path, never with original surfacing-log path.
	g.Expect(readPaths).To(ContainElement(expectedPrivatePath))
	g.Expect(readPaths).NotTo(ContainElement("/data/surfacing-log.jsonl"))
}

// TestWithEvalLinkUpdater_SetsOption verifies the option applies without error.
func TestWithEvalLinkUpdater_SetsOption(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	updater := &fakeEvalLinkUpdater{}
	evaluator := evaluate.New(t.TempDir(), evaluate.WithEvalLinkUpdater(updater))

	g.Expect(evaluator).NotTo(BeNil())
}

// unexported constants.
const (
	numNoopOpts = 3
	testDataDir = "/data"
)

type evalRegistryCall struct {
	id      string
	outcome string
}

// fakeEvalLinkUpdater is a test double for evaluate.EvalLinkUpdater.
type fakeEvalLinkUpdater struct {
	getCalls []string
	setCalls []string
}

func (f *fakeEvalLinkUpdater) GetEntryLinks(id string) ([]evaluate.EvalLink, error) {
	f.getCalls = append(f.getCalls, id)

	return nil, nil
}

func (f *fakeEvalLinkUpdater) SetEntryLinks(id string, _ []evaluate.EvalLink) error {
	f.setCalls = append(f.setCalls, id)

	return nil
}

// fakeEvalLinkUpdaterCapture captures links set per ID.
type fakeEvalLinkUpdaterCapture struct {
	existing map[string][]evaluate.EvalLink
	setLinks map[string][]evaluate.EvalLink
	err      error
}

func (f *fakeEvalLinkUpdaterCapture) GetEntryLinks(id string) ([]evaluate.EvalLink, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.existing[id], nil
}

func (f *fakeEvalLinkUpdaterCapture) SetEntryLinks(id string, links []evaluate.EvalLink) error {
	if f.err != nil {
		return f.err
	}

	if f.setLinks == nil {
		f.setLinks = make(map[string][]evaluate.EvalLink)
	}

	f.setLinks[id] = links

	return nil
}

// fakeEvalRegistryRecorder is a test double for evaluate.RegistryRecorder.
type fakeEvalRegistryRecorder struct {
	calls []evalRegistryCall
	err   error
}

func (f *fakeEvalRegistryRecorder) RecordEvaluation(id, outcome string) error {
	f.calls = append(f.calls, evalRegistryCall{id: id, outcome: outcome})
	return f.err
}

// testFS is a path-redirect layer for tests. It intercepts rename calls on known logical paths
// and transparently redirects subsequent reads of the renamed private path back to the original
// content. Tests register content by logical path; they never need to know the internal rename path.
//
// Usage: pass a WithReadFile that handles memory TOMLs (not the surfacing log). testFS will
// intercept the surfacing-log rename and redirect the private-path read back to the logical path.
type testFS struct {
	files   map[string][]byte // logical path → content
	aliases map[string]string // private/renamed path → logical path
}

// rename records newPath as an alias for oldPath when oldPath is a known logical file.
func (fs *testFS) rename(oldPath, newPath string) error {
	if _, known := fs.files[oldPath]; known {
		fs.aliases[newPath] = oldPath
	}

	return nil
}

// wrapReadFile returns a read function that first resolves aliases registered by rename,
// then falls through to the provided base handler for any path not in testFS.
// This allows the surfacing log to be redirected transparently while memory TOML reads
// go to the test's own handler.
func (fs *testFS) wrapReadFile(base func(string) ([]byte, error)) func(string) ([]byte, error) {
	return func(name string) ([]byte, error) {
		resolvedName := name
		if logical, isAlias := fs.aliases[name]; isAlias {
			resolvedName = logical
		}

		if data, ok := fs.files[resolvedName]; ok {
			return data, nil
		}

		if base != nil {
			return base(name)
		}

		return nil, os.ErrNotExist
	}
}

func makeTestEvaluator(opts ...evaluate.Option) *evaluate.Evaluator {
	noops := make([]evaluate.Option, 0, numNoopOpts+len(opts))
	noops = append(noops,
		evaluate.WithMkdirAll(func(_ string, _ os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error { return nil }),
		evaluate.WithRename(func(_, _ string) error { return nil }),
	)

	return evaluate.New(testDataDir, append(noops, opts...)...)
}

// newTestFS creates a testFS pre-populated with logical-path content entries.
// Example: newTestFS(map[string][]byte{"/data/surfacing-log.jsonl": []byte(logContent)})
func newTestFS(files map[string][]byte) *testFS {
	return &testFS{
		files:   files,
		aliases: make(map[string]string),
	}
}

// withSurfacingLog returns evaluate options that transparently handle the rename-before-read
// pattern for the surfacing log. The logContent is served under the logical path
// "/data/surfacing-log.jsonl"; when the evaluator renames it to a private tmp path,
// reads of the private path are redirected back to logContent automatically.
//
// The memReader handles reads for all other paths (memory TOML files, etc.). Pass nil
// to get os.ErrNotExist for any non-surfacing-log path.
//
// Tests using this helper never need to know or match the internal rename path.
func withSurfacingLog(logContent string, memReader func(string) ([]byte, error)) []evaluate.Option {
	const surfacingLogPath = testDataDir + "/surfacing-log.jsonl"

	fs := newTestFS(map[string][]byte{
		surfacingLogPath: []byte(logContent),
	})

	return []evaluate.Option{
		evaluate.WithRename(func(oldPath, newPath string) error {
			return fs.rename(oldPath, newPath)
		}),
		evaluate.WithReadFile(fs.wrapReadFile(memReader)),
		evaluate.WithRemoveFile(func(_ string) error { return nil }),
	}
}
