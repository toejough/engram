package llmcmd_test

import (
	"context"
	"testing"
	"time"

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

func TestRun_TimeoutKillsProcess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runner := llmcmd.NewWithTimeout("sleep 60", 50*time.Millisecond)

	start := time.Now()
	_, err := runner.Run(context.Background(), "irrelevant")
	elapsed := time.Since(start)

	g.Expect(err).To(MatchError(ContainSubstring("timeout")))
	g.Expect(elapsed).To(BeNumerically("<", 5*time.Second))
}
