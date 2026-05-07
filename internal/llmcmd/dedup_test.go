package llmcmd_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/llmcmd"
)

func TestCallerFunc_PromptIncludesSystemAndUser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	call := llmcmd.CallerFunc(llmcmd.New("cat"))

	out, err := call(context.Background(), "ignored-model", "system part", "user part")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("system part"))
	g.Expect(out).To(ContainSubstring("user part"))
}
