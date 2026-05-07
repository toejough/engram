package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

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
