package cli_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestIngestDedupsExactContentByPrecedence: two byte-identical sources
// discovered in the SAME --auto sweep — one under the repo-markdown root,
// one under its own ancestor .claude dir — must produce exactly one index
// build, and the canonical must be the repo-root copy (selectCanonical rule
// 2 beats rule 3). The loser gets a manifest entry recording it as a
// duplicate (so a later run never re-reads it) but no index file.
func TestIngestDedupsExactContentByPrecedence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "## Notes\nIdentical content copied into two places, long enough to form a real chunk.\n"

	fs := newSweepFS()
	fs.put("/repo/notes/a.md", content, 100)
	fs.put("/repo/.claude/copy/a.md", content, 100)

	emb := &countingEmbedder{}
	deps := dedupAutoDeps(fs, emb, "/repo", map[string][]string{
		"/repo":         {"/repo/notes/a.md"},
		"/repo/.claude": {"/repo/.claude/copy/a.md"},
	})

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Auto: true, ChunksDir: "/chunks"}, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	repoIdx := "/chunks/" + cli.ExportIndexFileName("/repo/notes/a.md")
	claudeIdx := "/chunks/" + cli.ExportIndexFileName("/repo/.claude/copy/a.md")

	_, repoIndexed := fs.files[repoIdx]
	g.Expect(repoIndexed).To(gomega.BeTrue(), "repo-root copy must be the indexed canonical")

	_, claudeIndexed := fs.files[claudeIdx]
	g.Expect(claudeIndexed).To(gomega.BeFalse(), "ancestor .claude copy must NOT be indexed — it is the duplicate")

	g.Expect(emb.calls).To(gomega.Equal(1),
		"exactly one chunk is embedded: the single canonical member's one chunk, never the duplicate's")

	manifest := decodeManifest(g, fs.files)

	claudeEntry, present := manifest["/repo/.claude/copy/a.md"]
	g.Expect(present).To(gomega.BeTrue(), "duplicate must still get a manifest entry so it isn't re-read next run")
	g.Expect(claudeEntry["duplicate_of"]).To(gomega.Equal("/repo/notes/a.md"))

	repoEntry, present := manifest["/repo/notes/a.md"]
	g.Expect(present).To(gomega.BeTrue())

	_, hasDup := repoEntry["duplicate_of"]
	g.Expect(hasDup).To(gomega.BeFalse(), "canonical entry must not carry duplicate_of")
}

// TestIngestDoesNotReReadUnchangedKnownSource pins Unit 3 step 1's
// load-bearing seeding directly: a source already known to the manifest,
// unchanged on disk, must NOT be re-read on a second ingest. This is
// distinct from "must not re-embed" (TestSweepSkipsUnchangedWithoutEmbedding
// in ingest_sweep_test.go) — canonicalUpToDate's hash short-circuit
// guarantees that independently of whether cheapSkipEligible ever ran, so a
// mutation that makes cheapSkipEligible always return false (forcing a
// read+hash of every source on every run) would sail through the embed-only
// assertion. Counting ReadFile calls catches it directly.
func TestIngestDoesNotReReadUnchangedKnownSource(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	source := "/docs/stable.md"
	content := "## Stable\nContent that never changes across two ingest runs, long enough for a chunk.\n"

	fs := newSweepFS()
	fs.put(source, content, 100)

	readCalls := 0
	deps := sweepDeps(fs, &countingEmbedder{}, source)
	deps.Stat = realisticStat(fs)

	realRead := deps.ReadFile
	deps.ReadFile = func(path string) ([]byte, error) {
		if path == source {
			readCalls++
		}

		return realRead(path)
	}

	args := cli.IngestArgs{Sweep: []string{"/docs"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())
	g.Expect(readCalls).To(gomega.Equal(1), "first ingest of a new source must read it exactly once")

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())
	g.Expect(readCalls).To(gomega.Equal(1),
		"second ingest of an unchanged known source must NOT re-read it: cheapSkipEligible must reuse "+
			"the manifest's cached hash rather than re-deriving it")
}

// TestIngestEvictsLowerPrecedenceDuplicate: a lower-precedence copy (ancestor
// .claude) is indexed first, alone. A later run discovers a byte-identical
// higher-precedence twin (repo root) — the twin must become canonical AND
// the stale .claude index/manifest entry must be evicted (structural change
// 2), not left to accumulate alongside the new one.
func TestIngestEvictsLowerPrecedenceDuplicate(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "## Notes\nIdentical content migrating from a low-precedence copy to the repo root.\n"

	fs := newSweepFS()
	fs.put("/repo/.claude/copy/a.md", content, 100)

	listByRoot := map[string][]string{
		"/repo":         nil,
		"/repo/.claude": {"/repo/.claude/copy/a.md"},
	}
	deps := dedupAutoDeps(fs, &countingEmbedder{}, "/repo", listByRoot)
	args := cli.IngestArgs{Auto: true, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	claudeIdx := "/chunks/" + cli.ExportIndexFileName("/repo/.claude/copy/a.md")
	_, indexedBefore := fs.files[claudeIdx]
	g.Expect(indexedBefore).To(gomega.BeTrue(), "sole candidate must be indexed as canonical on the first run")

	// The repo-root copy appears: same content, higher precedence.
	fs.put("/repo/notes/a.md", content, 100)

	listByRoot["/repo"] = []string{"/repo/notes/a.md"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	repoIdx := "/chunks/" + cli.ExportIndexFileName("/repo/notes/a.md")
	_, repoIndexed := fs.files[repoIdx]
	g.Expect(repoIndexed).To(gomega.BeTrue(), "repo-root copy must now be indexed as canonical")

	_, claudeStillIndexed := fs.files[claudeIdx]
	g.Expect(claudeStillIndexed).To(gomega.BeFalse(), "stale .claude index must be evicted, not left alongside")

	manifest := decodeManifest(g, fs.files)

	claudeEntry, present := manifest["/repo/.claude/copy/a.md"]
	g.Expect(present).To(gomega.BeTrue())
	g.Expect(claudeEntry["duplicate_of"]).To(gomega.Equal("/repo/notes/a.md"))
}

// TestIngestPrefersPhysicallyCloserAncestorAcrossTypes: a .pi dir sits AT
// cwd (physical level 0) while the nearest .claude dir is one level
// farther up (physical level 1) — the exact cross-type scenario where a
// naive per-type ordinal AncestorDepth (each walk's own "0th hit") breaks
// selectCanonical's rule 3 ("closest ancestor first"), since both would
// then read depth 0 and fall through to the separator/lexicographic
// tie-break instead. AncestorDepth must reflect real physical distance from
// cwd so the .pi copy — genuinely closer — wins.
func TestIngestPrefersPhysicallyCloserAncestorAcrossTypes(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "## Notes\nIdentical content reachable via both a .pi dir at cwd and a farther .claude dir.\n"

	fs := newSweepFS()
	fs.put("/repo/sub/.pi/copy/a.md", content, 100)
	fs.put("/repo/.claude/copy/a.md", content, 100)

	listByRoot := map[string][]string{
		"/repo":         nil,
		"/repo/sub/.pi": {"/repo/sub/.pi/copy/a.md"},
		"/repo/.claude": {"/repo/.claude/copy/a.md"},
	}

	dirs := map[string]bool{"/repo/.git": true, "/repo/sub/.pi": true, "/repo/.claude": true}

	deps := cli.IngestDeps{
		ReadFile:  fs.read,
		WriteFile: fs.write,
		Stat:      realisticStat(fs),
		Remove:    fs.remove,
		ListSources: func(root cli.SweepRoot) ([]string, error) {
			return listByRoot[root.Path], nil
		},
		ReadTranscript: transcriptReader(""),
		Embedder:       &countingEmbedder{},
		IsDir:          func(path string) bool { return dirs[path] },
		Getwd:          func() (string, error) { return "/repo/sub", nil },
		SessionDir:     func(string) string { return "" },
	}

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Auto: true, ChunksDir: "/chunks"}, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	piIdx := "/chunks/" + cli.ExportIndexFileName("/repo/sub/.pi/copy/a.md")
	claudeIdx := "/chunks/" + cli.ExportIndexFileName("/repo/.claude/copy/a.md")

	_, piIndexed := fs.files[piIdx]
	g.Expect(piIndexed).To(gomega.BeTrue(),
		"the .pi dir AT cwd (physical level 0) must win over the farther .claude dir (level 1)")

	_, claudeIndexed := fs.files[claudeIdx]
	g.Expect(claudeIndexed).To(gomega.BeFalse(), "the physically farther ancestor copy must be the duplicate")
}

// TestIngestPromotesDuplicateWhenCanonicalVanishes: the canonical repo copy
// is indexed first; a later run no longer sweeps it (deleted, moved, or
// simply out of scope), leaving its former duplicate as the sole remaining
// hash-group member. The duplicate must be promoted to canonical and
// actually indexed — exercising the rare path where its bytes were never
// read this run (cheap-skip reused its cached hash) and must be read fresh.
func TestIngestPromotesDuplicateWhenCanonicalVanishes(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "## Notes\nContent whose sole canonical copy vanishes, promoting the duplicate.\n"

	fs := newSweepFS()
	fs.put("/repo/notes/a.md", content, 100)
	fs.put("/repo/.claude/copy/a.md", content, 100)

	listByRoot := map[string][]string{
		"/repo":         {"/repo/notes/a.md"},
		"/repo/.claude": {"/repo/.claude/copy/a.md"},
	}
	deps := dedupAutoDeps(fs, &countingEmbedder{}, "/repo", listByRoot)
	args := cli.IngestArgs{Auto: true, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	claudeIdx := "/chunks/" + cli.ExportIndexFileName("/repo/.claude/copy/a.md")
	_, claudeIndexedBefore := fs.files[claudeIdx]
	g.Expect(claudeIndexedBefore).To(gomega.BeFalse(), "sanity: duplicate is not indexed on the first run")

	// The repo copy is no longer swept; only the duplicate remains.
	listByRoot["/repo"] = nil

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	_, claudeIndexedAfter := fs.files[claudeIdx]
	g.Expect(claudeIndexedAfter).To(gomega.BeTrue(), "former duplicate must be promoted and indexed")

	manifest := decodeManifest(g, fs.files)

	claudeEntry, present := manifest["/repo/.claude/copy/a.md"]
	g.Expect(present).To(gomega.BeTrue())

	_, hasDup := claudeEntry["duplicate_of"]
	g.Expect(hasDup).To(gomega.BeFalse(), "promoted entry must no longer carry duplicate_of")
}

// TestIngestRebuildsWhenIndexFileMissing: a manifest entry is present and
// the source is unchanged, but its .jsonl index file has vanished — the
// crash window between removing an index file and its manifest entry
// (cross-unit-coordination note). The cheap-skip must NOT trust the
// manifest alone; the next ingest must rebuild the index from scratch
// (re-embedding, since the old vectors were lost with the file).
func TestIngestRebuildsWhenIndexFileMissing(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stripped := "USER: content that must survive a crash window between removing the index file and its manifest entry"
	source := "/sessions/crash.jsonl"

	fs := newSweepFS()
	fs.put(source, "{raw jsonl bytes}", 100)

	emb := &countingEmbedder{}
	deps := sweepDeps(fs, emb, source)
	deps.Stat = realisticStat(fs)
	deps.ReadTranscript = transcriptReader(stripped)

	args := cli.IngestArgs{Sweep: []string{"/sessions"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	indexKey := "/chunks/" + cli.ExportIndexFileName(source)
	_, indexedBefore := fs.files[indexKey]
	g.Expect(indexedBefore).To(gomega.BeTrue(), "sanity: index file exists after the first ingest")

	// Simulate the crash window: the index file is gone, but the manifest
	// entry (and the source's stat) are untouched.
	delete(fs.files, indexKey)

	before := emb.calls

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	_, indexedAfter := fs.files[indexKey]
	g.Expect(indexedAfter).To(gomega.BeTrue(),
		"index file missing despite an unchanged known source must self-heal: the next ingest rebuilds it")

	g.Expect(emb.calls).To(gomega.BeNumerically(">", before),
		"a rebuilt-from-scratch index must re-embed — the old vectors were lost with the index file")
}

// TestIngestSkipsVaultCopyOutsideCanonicalVault: a swept .md file with a
// sibling .vec.json sidecar is structurally a vault-note copy (Rule B) —
// content need not even match a live note (the drifted case exact-content
// dedup misses); sitting outside the resolved vault path is enough to skip
// it. A normal swept .md with no sidecar must still be ingested in the same
// run, proving the rule doesn't over-trigger.
func TestIngestSkipsVaultCopyOutsideCanonicalVault(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		strayNote = "/snap/vault/1.md" // "vault" here is just a directory name, NOT the configured vault
		strayVec  = "/snap/vault/1.vec.json"
		otherDoc  = "/snap/other/2.md"
	)

	fs := &memFS{files: map[string][]byte{
		strayNote: []byte("## Drifted\nThis stray copy has drifted from the live vault note, long enough for a chunk.\n"),
		strayVec:  []byte(`{"schema_version":1}`),
		otherDoc:  []byte("## Fine\nA normal swept markdown file with no sidecar, long enough for a chunk.\n"),
	}}

	deps := cli.IngestDeps{
		ReadFile:  fs.read,
		WriteFile: fs.write,
		Stat: func(path string) (cli.SourceStat, error) {
			if path == strayVec {
				return cli.SourceStat{}, nil
			}

			return cli.SourceStat{}, io.ErrUnexpectedEOF
		},
		ListSources:    func(cli.SweepRoot) ([]string, error) { return []string{strayNote, otherDoc}, nil },
		ReadTranscript: transcriptReader(""),
		Embedder:       fakeIngestEmbedder{},
	}

	args := cli.IngestArgs{Sweep: []string{"/snap"}, ChunksDir: "/chunks", Vault: "/vault"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	_, strayIndexed := fs.files["/chunks/"+cli.ExportIndexFileName(strayNote)]
	g.Expect(strayIndexed).To(gomega.BeFalse(), "a .md with a sibling .vec.json outside the vault must be skipped")

	_, otherIndexed := fs.files["/chunks/"+cli.ExportIndexFileName(otherDoc)]
	g.Expect(otherIndexed).To(gomega.BeTrue(), "a normal swept .md with no sidecar must still be ingested")

	manifest := decodeManifest(g, fs.files)

	_, strayInManifest := manifest[strayNote]
	g.Expect(strayInManifest).To(gomega.BeFalse(), "the stray vault copy must not appear in the manifest at all")
}

// remove deletes path from the fake filesystem, implementing IngestDeps.Remove.
func (m *memFS) remove(path string) error {
	delete(m.files, path)

	return nil
}

// decodeManifest reads and JSON-decodes the fake filesystem's manifest.json
// into a plain map, so tests can assert on the unexported manifestEntry
// fields (file_hash, duplicate_of) by their JSON keys. files is the fake
// filesystem's raw path->bytes map (memFS.files, or sweepFS.files via
// embedding-promotion), so this works for either fixture type.
func decodeManifest(g *gomega.WithT, files map[string][]byte) map[string]map[string]any {
	data, ok := files["/chunks/manifest.json"]
	g.Expect(ok).To(gomega.BeTrue(), "manifest.json must have been written")

	var manifest map[string]map[string]any

	g.Expect(json.Unmarshal(data, &manifest)).To(gomega.Succeed())

	return manifest
}

// dedupAutoDeps builds IngestDeps for --auto with a repo root that owns its
// own .claude ancestor directory (cwd = repoRoot), giving Unit 3's dedup
// tests two distinguishable origins reachable from ONE cwd fixture: a
// source under repoRoot gets origin=repo, one under repoRoot/.claude gets
// origin=claude-ancestor. listByRoot maps a resolved SweepRoot's Path to the
// files ListSources returns for it; roots absent from the map yield none.
func dedupAutoDeps(
	fs *sweepFS,
	emb *countingEmbedder,
	repoRoot string,
	listByRoot map[string][]string,
) cli.IngestDeps {
	return cli.IngestDeps{
		ReadFile:  fs.read,
		WriteFile: fs.write,
		Stat:      realisticStat(fs),
		Remove:    fs.remove,
		ListSources: func(root cli.SweepRoot) ([]string, error) {
			return listByRoot[root.Path], nil
		},
		ReadTranscript: transcriptReader(""),
		Embedder:       emb,
		IsDir: func(path string) bool {
			return path == repoRoot+"/.git" || path == repoRoot+"/.claude"
		},
		Getwd:      func() (string, error) { return repoRoot, nil },
		SessionDir: func(string) string { return "" },
	}
}

// realisticStat wraps a sweepFS with a Stat function that additionally
// reports existence (a zero-value SourceStat) for any written-but-
// unregistered path — chunk index files in particular, which sweepFS.put
// never sees since they're produced by WriteFile, not a fixture seed. This
// lets Unit 3's cheap-skip/eviction tests observe real index-file presence
// the same way production Stat does, without changing sweepFS.stat's
// existing contract (other tests rely on it erroring for any untracked
// path).
func realisticStat(fs *sweepFS) func(string) (cli.SourceStat, error) {
	return func(path string) (cli.SourceStat, error) {
		if st, ok := fs.stats[path]; ok {
			return st, nil
		}

		if _, ok := fs.files[path]; ok {
			return cli.SourceStat{}, nil
		}

		return cli.SourceStat{}, io.ErrUnexpectedEOF
	}
}
