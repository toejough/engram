package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRunCycle_RequiresLLMCmd(t *testing.T) {
	// No t.Parallel() — t.Setenv is incompatible with parallel tests.
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	args := cli.CycleArgs{ProjectDir: "/tmp/x"}

	var stdout bytes.Buffer

	err := cli.RunCycle(context.Background(), args, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("llm-cmd is required")))
}

func TestRunCycle_EmitsValidJSONWithEmptyArrays(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Sentinel responses. The transcript reader returns empty content because
	// the project dir has no sessions. The cycle's runner is called for the
	// recall step (since recaller != nil); it returns NO QUERIES so no recall
	// is run. The learn step is also called; it returns [] so no persist.
	cmdString := `printf 'NO QUERIES'`

	args := cli.CycleArgs{
		ProjectDir: dir,
		LLMCmd:     cmdString,
		DataDir:    dir,
	}

	var stdout bytes.Buffer

	err := cli.RunCycle(context.Background(), args, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var decoded struct {
		Learned  []any `json:"learned"`
		Recalled []any `json:"recalled"`
	}

	g.Expect(json.Unmarshal(stdout.Bytes(), &decoded)).To(Succeed())
	g.Expect(decoded.Learned).To(BeEmpty())
	g.Expect(decoded.Recalled).To(BeEmpty())
}
