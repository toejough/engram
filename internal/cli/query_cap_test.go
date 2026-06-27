package cli_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestCapChunkContent_CapsChunksBeyondBudget verifies the budget keeps the
// first N chunks full, snippets later chunks, and never touches notes.
func TestCapChunkContent_CapsChunksBeyondBudget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	long := "line one\nline two with many words " + strings.Repeat("x", 200)
	kinds := []string{"fact", "chunk", "chunk", "chunk", "chunk"}
	contents := []string{"NOTE FULL", long, long, long, long}

	out, snipped := cli.ExportCapChunkContent(kinds, contents, 2)

	g.Expect(out[0]).To(Equal("NOTE FULL"), "note must be untouched")
	g.Expect(out[1]).To(Equal(long), "chunk #1 within budget stays full")
	g.Expect(out[2]).To(Equal(long), "chunk #2 within budget stays full")
	g.Expect(out[3]).NotTo(Equal(long), "chunk #3 beyond budget must be snippeted")
	g.Expect(out[4]).NotTo(Equal(long), "chunk #4 beyond budget must be snippeted")
	g.Expect(out[3]).To(HaveSuffix("…"))
	g.Expect(strings.Contains(out[3], "\n")).To(BeFalse(), "snippet collapses newlines")
	g.Expect(len([]rune(out[3]))).To(BeNumerically("<=", capSnippetMaxRunes))
	g.Expect(snipped).To(Equal(2))
}

// TestCapChunkContent_ZeroBudgetIsNoOp verifies budget<=0 leaves everything full.
func TestCapChunkContent_ZeroBudgetIsNoOp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	long := strings.Repeat("y", 500)
	kinds := []string{"chunk", "chunk", "chunk"}
	contents := []string{long, long, long}

	out, snipped := cli.ExportCapChunkContent(kinds, contents, 0)

	g.Expect(out).To(Equal(contents), "budget 0 = unlimited, no snipping")
	g.Expect(snipped).To(Equal(0))
}

// TestClearChunkContent_ClearsChunksKeepsNotes verifies lazy mode zeroes
// chunk-item content while leaving note content intact.
func TestClearChunkContent_ClearsChunksKeepsNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	kinds := []string{"fact", "chunk", "chunk"}
	contents := []string{"noteC", "c1", "c2"}

	out := cli.ExportClearChunkContent(kinds, contents)

	g.Expect(out[0]).To(Equal("noteC"), "note content must be preserved")
	g.Expect(out[1]).To(Equal(""), "chunk #1 content must be cleared")
	g.Expect(out[2]).To(Equal(""), "chunk #2 content must be cleared")
}

// TestRenderQueryPayload_LazyChunksSurfacedAndNotSnippeted verifies lazy mode
// surfaces lazy_chunks in the budget and skips capChunkContent (snipped stays 0).
func TestRenderQueryPayload_LazyChunksSurfacedAndNotSnippeted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	long := "line one\nline two " + strings.Repeat("x", 300)
	kinds := []string{"fact", "chunk", "chunk"}
	contents := []string{"NOTE", long, long}

	out, err := cli.ExportRenderQueryPayloadBudget(kinds, contents, true, 1)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("lazy_chunks: true"), "lazy mode must surface in budget")
	g.Expect(out).To(ContainSubstring("chunks_snippeted: 0"), "lazy mode skips capChunkContent")
}

// TestRenderQueryPayload_NonLazyOmitsLazyAndStillCaps verifies non-lazy mode
// omits lazy_chunks (omitempty) and still snippets chunks beyond the budget.
func TestRenderQueryPayload_NonLazyOmitsLazyAndStillCaps(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	long := "line one\nline two " + strings.Repeat("x", 300)
	kinds := []string{"fact", "chunk", "chunk"}
	contents := []string{"NOTE", long, long}

	out, err := cli.ExportRenderQueryPayloadBudget(kinds, contents, false, 1)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).NotTo(ContainSubstring("lazy_chunks"), "non-lazy omits the field (omitempty)")
	g.Expect(out).To(ContainSubstring("chunks_snippeted: 1"), "non-lazy still caps the over-budget chunk")
}

// TestResolveContentBudget_DefaultsAndOverrides verifies the default-bake logic:
// unset (0) → the baked default; negative → unlimited (0 = no-op); positive → itself.
func TestResolveContentBudget_DefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportResolveContentBudget(0)).To(Equal(cli.ExportDefaultContentBudget), "unset → default")
	g.Expect(cli.ExportResolveContentBudget(8)).To(Equal(8), "positive → itself")
	g.Expect(cli.ExportResolveContentBudget(-1)).To(BeNumerically("<=", 0), "negative → unlimited (no-op)")
}

// TestResolveRecentFill_DefaultsAndOverrides verifies the recency-channel
// fill-count resolution: unset (0) → the baked default (25); positive → itself;
// negative → 0 (channel off).
func TestResolveRecentFill_DefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportResolveRecentFill(0)).To(Equal(cli.ExportDefaultRecentFill), "unset → default")
	g.Expect(cli.ExportResolveRecentFill(5)).To(Equal(5), "positive → itself")
	g.Expect(cli.ExportResolveRecentFill(-1)).To(Equal(0), "negative → channel off")
}

// TestSnippet_CollapsesAndTruncates verifies the snippet algorithm.
func TestSnippet_CollapsesAndTruncates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Short multiline: collapses whitespace, no ellipsis.
	g.Expect(cli.ExportSnippet("  hello\n\n  world  ")).To(Equal("hello world"))

	// Long: truncates to <=160 runes with a trailing ellipsis.
	long := strings.Repeat("a", 500)
	got := cli.ExportSnippet(long)
	g.Expect(got).To(HaveSuffix("…"))
	g.Expect(len([]rune(got))).To(BeNumerically("<=", capSnippetMaxRunes))
}

// unexported constants.
const (
	capSnippetMaxRunes = 160
)
