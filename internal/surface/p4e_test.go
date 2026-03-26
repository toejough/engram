package surface_test

// P4-early: Budget quick wins — effectiveness gating + BM25 floor (issue #88)
// REQ-P4e-1: SessionStart ranks by effectiveness, gates on >40% (no-data defaults to 50%)
// REQ-P4e-2: SessionStart top-7 limit, 600 token default budget
// REQ-P4e-3: UserPromptSubmit 250 token default budget
// REQ-P4e-5: InvocationTokenLogger called after each surface invocation

import (
	"bytes"
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// T-P4e-8: InvocationTokenLogger is called with output token count after surface.
func TestTP4e8_InvocationTokenLoggerCalled(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:     "Alpha Memory",
			FilePath:  "alpha.toml",
			Keywords:  []string{"alphatoken"},
			Principle: "always check alphatoken",
			UpdatedAt: time.Now(),
		},
		{
			Title:    "Filler B",
			FilePath: "filler-b.toml",
			Keywords: []string{"unrelated"},
		},
		{
			Title:    "Filler C",
			FilePath: "filler-c.toml",
			Keywords: []string{"other"},
		},
	}

	retriever := &fakeRetriever{memories: memories}
	tokenLogger := &fakeInvocationTokenLogger{}
	s := surface.New(retriever, surface.WithInvocationTokenLogger(tokenLogger))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "alphatoken check",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(tokenLogger.calls).To(HaveLen(1))
	g.Expect(tokenLogger.calls[0].mode).To(Equal(surface.ModePrompt))
	g.Expect(tokenLogger.calls[0].tokenCount).To(BeNumerically(">", 0))
}

// fakeInvocationTokenLogger captures LogInvocationTokens calls.
type fakeInvocationTokenLogger struct {
	calls []invocationTokenCall
}

func (f *fakeInvocationTokenLogger) LogInvocationTokens(
	mode string,
	tokenCount int,
	_ time.Time,
) error {
	f.calls = append(f.calls, invocationTokenCall{mode: mode, tokenCount: tokenCount})

	return nil
}

type invocationTokenCall struct {
	mode       string
	tokenCount int
}
