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

	evaluator := evaluate.New(
		"/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			switch name {
			case "/data/surfacing-log.jsonl":
				return []byte(surfacingLog), nil
			case "/data/memories/mem1.toml":
				return []byte(mem1TOML), nil
			case "/data/memories/mem2.toml":
				return []byte(mem2TOML), nil
			default:
				return nil, os.ErrNotExist
			}
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, model, _, user string) (string, error) {
			capturedModel = model
			capturedUser = user

			return llmResponse, nil
		}),
		evaluate.WithNow(
			func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) },
		),
	)

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

	evaluator := evaluate.New(
		"/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(name string) error {
			removedPath = name
			return nil
		}),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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

	g.Expect(removedPath).To(Equal("/data/surfacing-log.jsonl"))
}

// T-109: Evaluator creates evaluations directory if missing.
func TestEvaluator_CreatesEvaluationsDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`
	)

	var mkdirCalledPath string

	evaluator := evaluate.New(
		"/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(path string, _ os.FileMode) error {
			mkdirCalledPath = path
			return nil
		}),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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

	g.Expect(mkdirCalledPath).To(Equal("/data/evaluations"))
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

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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
	writeCallCount := 0

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error {
			writeCallCount++
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
	g.Expect(writeCallCount).To(Equal(0))
	g.Expect(outcomes).To(BeNil())
}

// T-106c: Evaluator returns error when a memory TOML file is invalid.
func TestEvaluator_InvalidMemoryTOML_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = `{"memory_path":"/data/memories/bad.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte("this is [not valid toml {{{{"), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "", nil
		}),
		evaluate.WithNow(time.Now),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
}

// writeEvaluationLog: mkdirAll error is returned to caller.
func TestEvaluator_MkdirAllError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = "title = \"Memory\"\ncontent = \"Content\"\nprinciple = \"Principle\"\nanti_pattern = \"\""
	)

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error {
			return errors.New("mkdir failed")
		}),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"}]`, nil
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

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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

	evaluator := evaluate.New(
		"/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(string) ([]byte, error) {
			return nil, errors.New("permission denied")
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return nil, os.ErrNotExist
		}),
		evaluate.WithRemoveFile(func(string) error {
			return errors.New("remove failed")
		}),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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

	writeCallCount := 0

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error {
			writeCallCount++
			return nil
		}),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return "this is not valid JSON", nil
		}),
		evaluate.WithNow(time.Now),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
	g.Expect(outcomes).To(BeNil())
	g.Expect(writeCallCount).To(Equal(0))
}

// writeEvaluationLog: writeFile error is returned to caller.
func TestEvaluator_WriteFileError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML  = "title = \"Memory\"\ncontent = \"Content\"\nprinciple = \"Principle\"\nanti_pattern = \"\""
	)

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error {
			return errors.New("write failed")
		}),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return `[{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"}]`, nil
		}),
		evaluate.WithNow(time.Now),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).To(HaveOccurred())
}

// T-108: Evaluator writes per-session evaluation log.
func TestEvaluator_WritesEvaluationLog(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const surfacingLog = "" +
		`{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}` + "\n" +
		`{"memory_path":"/data/memories/m2.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:01Z"}` + "\n" +
		`{"memory_path":"/data/memories/m3.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:02Z"}`

	const genericTOML = `title = "A Memory"
content = "Some content"
principle = "Some principle"
anti_pattern = ""`

	const llmResponse = `[` +
		`{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"e1"},` +
		`{"memory_path":"/data/memories/m2.toml","outcome":"contradicted","evidence":"e2"},` +
		`{"memory_path":"/data/memories/m3.toml","outcome":"ignored","evidence":"e3"}` +
		`]`

	var (
		writtenPath string
		writtenData []byte
	)

	fixedNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(name string, data []byte, _ os.FileMode) error {
			writtenPath = name
			writtenData = data

			return nil
		}),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			return llmResponse, nil
		}),
		evaluate.WithNow(func() time.Time { return fixedNow }),
	)

	_, err := evaluator.Evaluate(context.Background(), "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writtenPath).To(Equal("/data/evaluations/2024-01-15T10-30-00Z.jsonl"))

	lines := strings.Split(strings.TrimRight(string(writtenData), "\n"), "\n")
	g.Expect(lines).To(HaveLen(3))

	for _, line := range lines {
		var record map[string]any

		parseErr := json.Unmarshal([]byte(line), &record)
		g.Expect(parseErr).NotTo(HaveOccurred())
		g.Expect(record).To(HaveKey("memory_path"))
		g.Expect(record).To(HaveKey("outcome"))
		g.Expect(record).To(HaveKey("evidence"))
		g.Expect(record).To(HaveKey("evaluated_at"))
	}
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

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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

	evaluator := evaluate.New("/data",
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
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

// T-P3-9b: updateEvalCorrelationLinks reads and sets entry links for each outcome.
func TestUpdateEvalCorrelationLinks_CallsUpdater(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		surfacingLog = `{"memory_path":"/data/memories/m1.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}
{"memory_path":"/data/memories/m2.toml","mode":"prompt","surfaced_at":"2024-01-15T10:00:00Z"}`
		genericTOML = `title = "Memory"
content = "Content"
principle = "Principle"
anti_pattern = ""`
	)

	updater := &fakeEvalLinkUpdater{}

	evaluator := evaluate.New("/data",
		evaluate.WithEvalLinkUpdater(updater),
		evaluate.WithReadFile(func(name string) ([]byte, error) {
			if name == "/data/surfacing-log.jsonl" {
				return []byte(surfacingLog), nil
			}

			return []byte(genericTOML), nil
		}),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
		evaluate.WithLLMCaller(func(_ context.Context, _, _, _ string) (string, error) {
			const response = `[` +
				`{"memory_path":"/data/memories/m1.toml","outcome":"followed","evidence":"ok"},` +
				`{"memory_path":"/data/memories/m2.toml","outcome":"followed","evidence":"ok"}` +
				`]`

			return response, nil
		}),
		evaluate.WithNow(func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) }),
	)

	outcomes, err := evaluator.Evaluate(context.Background(), "transcript")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(outcomes).To(HaveLen(2))
	g.Expect(updater.getCalls).To(BeNumerically(">=", 2))
}

// T-P3-9a: WithEvalLinkUpdater sets the link updater on the evaluator.
func TestWithEvalLinkUpdater_SetsOption(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	updater := &fakeEvalLinkUpdater{}
	evaluator := evaluate.New("/data",
		evaluate.WithEvalLinkUpdater(updater),
		evaluate.WithReadFile(func(string) ([]byte, error) { return nil, os.ErrNotExist }),
		evaluate.WithRemoveFile(func(string) error { return nil }),
		evaluate.WithMkdirAll(func(string, os.FileMode) error { return nil }),
		evaluate.WithWriteFile(func(string, []byte, os.FileMode) error { return nil }),
	)

	// Should not panic when no surfacing log exists — evaluator handles it gracefully.
	_, err := evaluator.Evaluate(context.Background(), "transcript")

	g.Expect(err).NotTo(HaveOccurred())
}

type evalRegistryCall struct {
	id      string
	outcome string
}

// fakeEvalLinkUpdater is a test double for evaluate.EvalLinkUpdater.
type fakeEvalLinkUpdater struct {
	getCalls int
}

func (f *fakeEvalLinkUpdater) GetEntryLinks(_ string) ([]evaluate.EvalLink, error) {
	f.getCalls++

	return nil, nil
}

func (f *fakeEvalLinkUpdater) SetEntryLinks(_ string, _ []evaluate.EvalLink) error {
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
