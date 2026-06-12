package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestInvariant_R1_RecallMirror locks invariant R1: a note written with
// situation S is retrievable by a query phrased as S — a purely
// deterministic cosine match, no LLM. learn and recall are inverse over the
// situation field. We plant several notes whose embed source IS the
// situation string, then query each situation and assert the matching note
// is returned in the top-k items. Because the test embedder is deterministic
// and the note's vector equals embed(S), the self-match has cosine 1.0 and
// must rank in items.
func TestInvariant_R1_RecallMirror(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := NewWithT(rt)

		// A pool of clearly-distinct situation strings. We pick a subset so
		// the property runs over many situation combinations, not one case.
		pool := []string{
			"orchestrating multi-step work under context pressure",
			"reasoning about agent coordination and dispatch",
			"writing concurrent go code with cancellation",
			"sharpening a spec before dispatching the work",
			"clustering vault embeddings for recall synthesis",
			"normalizing wikilink targets to basenames",
			"tracking marker forward-progress per source",
			"verifying idempotence of the update command",
		}

		count := rapid.IntRange(2, len(pool)).Draw(rt, "count")
		// Distinct indices into the pool → distinct situations.
		idxs := rapid.SampledFrom([]int{0, 1, 2, 3, 4, 5, 6, 7})

		chosen := make([]string, 0, count)
		seen := map[int]bool{}

		for len(chosen) < count {
			i := idxs.Draw(rt, fmt.Sprintf("idx-%d", len(chosen)))
			if seen[i] {
				continue
			}

			seen[i] = true

			chosen = append(chosen, pool[i])
		}

		vault := t.TempDir()
		memFS := newInMemoryFS()

		// One note per situation; the body (embed source) IS the situation.
		noteForSituation := make(map[string]string, len(chosen))

		for i, situation := range chosen {
			base := "r1-" + strconv.Itoa(i) + ".md"
			relPath := "" + base
			body := "---\ntype: fact\n---\n" + situation + "\n"

			plantNoteWithSidecar(t, memFS, filepath.Clean(vault), relPath, body)

			noteForSituation[situation] = base
		}

		// Querying each situation must surface its own note AND rank it at the
		// top by cosine — the recall mirror. Because the note's embed source IS
		// the situation and the embedder is deterministic, embed(query) ==
		// embed(note), so the self-match has the maximum possible score. We
		// assert its score equals the top item's score (tie-tolerant: a crude
		// stub-embedder collision could let another note tie at the max, but
		// nothing may outrank the mirror).
		for _, situation := range chosen {
			items := r1QueryItems(t, gExpect, memFS, vault, situation)

			want := noteForSituation[situation]

			gExpect.Expect(items).NotTo(BeEmpty(),
				"R1: querying situation %q returned no items", situation)

			if len(items) == 0 {
				return
			}

			// Compute the max score across items rather than assuming a sort
			// order, then require the mirror note to attain it.
			topScore := items[0].score
			matchScore, found := float32(-1), false

			for _, item := range items {
				if item.score > topScore {
					topScore = item.score
				}

				if strings.HasSuffix(item.path, want) {
					matchScore = item.score
					found = true
				}
			}

			gExpect.Expect(found).To(BeTrue(),
				"R1: querying situation %q must return its note %q in items; got %v",
				situation, want, r1Paths(items))
			gExpect.Expect(matchScore).To(BeNumerically(">=", topScore),
				"R1: note %q for situation %q must rank at the top (score %v < top %v)",
				want, situation, matchScore, topScore)
		}
	})
}

// r1Item is a path+score pair extracted from the query payload.
type r1Item struct {
	path  string
	score float32
}

// r1Paths projects item paths for assertion messages.
func r1Paths(items []r1Item) []string {
	paths := make([]string, 0, len(items))
	for _, item := range items {
		paths = append(paths, item.path)
	}

	return paths
}

// r1QueryItems runs a single-phrase query and returns the items (path+score)
// in payload order (highest score first).
func r1QueryItems(t *testing.T, gExpect *WithT, memFS *inMemoryFS, vault, phrase string) []r1Item {
	t.Helper()

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{phrase}, VaultPath: vault, Limit: 10},
		newQueryDeps(memFS), &out)
	gExpect.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return nil
	}

	var parsed queryParsed

	gExpect.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	items := make([]r1Item, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		items = append(items, r1Item{path: item.Path, score: item.Score})
	}

	return items
}
