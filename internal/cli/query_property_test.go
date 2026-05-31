package cli_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestQueryProperty_ClusterCountInRange — clusters_found ∈ [0, 7] always.
func TestQueryProperty_ClusterCountInRange(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := NewWithT(rt)

		const maxNotes = 30

		noteCount := rapid.IntRange(1, maxNotes).Draw(rt, "noteCount")

		vault := t.TempDir()
		memFS := newInMemoryFS()

		for i := range noteCount {
			relPath := "Permanent/" + propertyNodeName(i) + ".md"

			outgoing := propertyOutgoing(rt, i, noteCount)
			body := "---\ntype: fact\n---\nbody " + propertyNodeName(i) + "\n" + outgoing

			plantNoteWithSidecar(t, memFS, filepath.Clean(vault), relPath, body)
		}

		var out bytes.Buffer

		err := cli.RunQuery(context.Background(),
			cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault, Limit: 5},
			newQueryDeps(memFS), &out)
		gExpect.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		var parsed queryParsed

		gExpect.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
		gExpect.Expect(parsed.Budget.ClustersFound).To(BeNumerically(">=", 0))
		gExpect.Expect(parsed.Budget.ClustersFound).To(BeNumerically("<=", 7))
	})
}

// TestQueryProperty_SubgraphSizeBounded — random graphs, verify
// subgraph_size <= 200 and <= total_notes always.
func TestQueryProperty_SubgraphSizeBounded(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := NewWithT(rt)

		const maxNotes = 30

		noteCount := rapid.IntRange(1, maxNotes).Draw(rt, "noteCount")

		vault := t.TempDir()
		memFS := newInMemoryFS()

		for i := range noteCount {
			relPath := "Permanent/" + propertyNodeName(i) + ".md"

			outgoing := propertyOutgoing(rt, i, noteCount)
			body := "---\ntype: fact\n---\nbody content\n" + outgoing

			plantNoteWithSidecar(t, memFS, filepath.Clean(vault), relPath, body)
		}

		var out bytes.Buffer

		err := cli.RunQuery(context.Background(),
			cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault, Limit: 5},
			newQueryDeps(memFS), &out)
		gExpect.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		var parsed queryParsed

		gExpect.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
		gExpect.Expect(parsed.Budget.SubgraphSize).To(BeNumerically("<=", 200))
		gExpect.Expect(parsed.Budget.SubgraphSize).To(BeNumerically("<=", parsed.Budget.TotalNotes))
	})
}

// propertyNodeName returns a unique short basename for property notes.
func propertyNodeName(idx int) string {
	const wrap = 26

	first := byte('A' + (idx/wrap)%wrap)
	second := byte('A' + idx%wrap)

	return string([]byte{first, second})
}

// propertyOutgoing returns a 0..K random wikilink block targeting other
// notes by basename. Keeps wikilinks within the known note set.
func propertyOutgoing(rt *rapid.T, idx, total int) string {
	const maxOutgoing = 4

	const linkOverhead = 6 // "[[", "]]", "\n"

	count := rapid.IntRange(0, maxOutgoing).Draw(rt, "outgoingCount")

	sb := make([]byte, 0, count*linkOverhead)

	for i := range count {
		target := (idx + i + 1) % total

		sb = append(sb, '[', '[')
		sb = append(sb, propertyNodeName(target)...)
		sb = append(sb, ']', ']', '\n')
	}

	return string(sb)
}
