package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// TestCapItemsToLimit_FewerThanLimitReturnsAllUnchanged verifies no
// truncation happens when the item count is already within the limit.
func TestCapItemsToLimit_FewerThanLimitReturnsAllUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	paths := []string{"a", "b"}
	out := cli.ExportCapItemsToLimit(paths, 5)

	g.Expect(out).To(Equal([]string{"a", "b"}))
}

// TestCapItemsToLimit_TruncatesToFirstN verifies the helper keeps only the
// first `limit` items, assuming callers already provide final rank order.
func TestCapItemsToLimit_TruncatesToFirstN(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	paths := []string{"a", "b", "c", "d", "e"}
	out := cli.ExportCapItemsToLimit(paths, 3)

	g.Expect(out).To(Equal([]string{"a", "b", "c"}))
}

// TestRunQuery_BudgetLimitStillReportsResolvedValue verifies Budget.Limit
// keeps reporting the resolved --limit value, unchanged metadata behavior,
// even though items[] is now actually capped to it.
func TestRunQuery_BudgetLimitStillReportsResolvedValue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantWithFixedVector(t, memFS, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nbody\n", []float32{1, 0, 0, 0})

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"x"}, VaultPath: vault, Limit: 7}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var parsed queryParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).To(Succeed())

	g.Expect(parsed.Budget.Limit).To(Equal(7))
}

// TestRunQuery_DefaultLimitCapsLargeResultSet verifies the default --limit
// (20) now truncates a result set that would otherwise exceed it — before
// this fix, --limit was report-only metadata and never truncated items[].
func TestRunQuery_DefaultLimitCapsLargeResultSet(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	const noteCount = 25
	for i := range noteCount {
		relPath := fmt.Sprintf("%d.fact.md", i+1)
		body := fmt.Sprintf("---\ntype: fact\ntier: L2\nsituation: x\n---\n\nbody %d\n", i)
		plantWithFixedVector(t, memFS, vault, relPath, body, []float32{1, 0, 0, 0})
	}

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"x"}, VaultPath: vault}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var parsed queryParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).To(Succeed())

	g.Expect(parsed.Items).To(HaveLen(20))
}

// TestRunQuery_ExplicitLimitCapsItems verifies an explicit --limit N
// truncates items[] to N.
func TestRunQuery_ExplicitLimitCapsItems(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	const noteCount = 10
	for i := range noteCount {
		relPath := fmt.Sprintf("%d.fact.md", i+1)
		body := fmt.Sprintf("---\ntype: fact\ntier: L2\nsituation: x\n---\n\nbody %d\n", i)
		plantWithFixedVector(t, memFS, vault, relPath, body, []float32{1, 0, 0, 0})
	}

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"x"}, VaultPath: vault, Limit: 3}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var parsed queryParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).To(Succeed())

	g.Expect(parsed.Items).To(HaveLen(3))
}

// TestRunQuery_FewerItemsThanLimitReturnsAllUnchanged verifies a result set
// smaller than the limit is returned in full, unchanged by the new cap.
func TestRunQuery_FewerItemsThanLimitReturnsAllUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantWithFixedVector(t, memFS, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nbody\n", []float32{1, 0, 0, 0})

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"x"}, VaultPath: vault}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var parsed queryParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).To(Succeed())

	g.Expect(parsed.Items).To(HaveLen(1))
}
