package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRequireLLMCmd_ErrorsWhenMissing(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	err := cli.ExportRequireLLMCmd("")
	g.Expect(err).To(MatchError(ContainSubstring("llm-cmd is required")))
	g.Expect(err).To(MatchError(ContainSubstring("ENGRAM_LLM_CMD")))
}

func TestRequireLLMCmd_OkWhenFlagSet(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	err := cli.ExportRequireLLMCmd("opencode run -m foo")
	g.Expect(err).NotTo(HaveOccurred())
}

func TestResolveLLMCmd_EmptyWhenNeitherSet(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	got := cli.ExportResolveLLMCmd("")
	g.Expect(got).To(Equal(""))
}

func TestResolveLLMCmd_FallsBackToEnv(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "from-env")

	got := cli.ExportResolveLLMCmd("")
	g.Expect(got).To(Equal("from-env"))
}

func TestResolveLLMCmd_PrefersFlagOverEnv(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "from-env")

	got := cli.ExportResolveLLMCmd("from-flag")
	g.Expect(got).To(Equal("from-flag"))
}
