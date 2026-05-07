package llmcmd_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/llmcmd"
)

func TestRun_NonZeroExitReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runner := llmcmd.New("false")

	_, err := runner.Run(context.Background(), "anything")
	g.Expect(err).To(MatchError(ContainSubstring("llm-cmd exited")))
}

func TestRun_PipesPromptToStdinAndReturnsStdout(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// `cat` echoes stdin to stdout — perfect filter for testing.
	runner := llmcmd.New("cat")

	out, err := runner.Run(context.Background(), "hello world")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("hello world"))
}
