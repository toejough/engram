package cli

import (
	"bytes"
	"io"
	"io/fs"
	"testing"

	"github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// TestCanonicalCoversDuplicateRecords pins the record-level subset gate
// directly (2026-07-26 ship-readiness finding): mergeChunkRecords is
// append-only WITHIN a source, so a source's .jsonl index file accumulates
// every chunk record from every past ingest of that path — not just its
// current content. Two sources can therefore be byte-identical RIGHT NOW
// (same content hash, same hash group) while their index files hold
// entirely disjoint historical chunk records. Byte-hash identity of the
// CURRENT content — or even the mere EXISTENCE of the canonical's index
// file — is necessary but not sufficient proof that evicting the
// duplicate's index loses nothing; every one of the duplicate's own
// record content-hashes must also appear in the canonical's.
func TestCanonicalCoversDuplicateRecords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		chunksDir = "/chunks"
		canonical = "/repo/a.md"
		duplicate = "/repo/.claude/a.md"
	)

	canonicalIdx := indexPathFor(chunksDir, canonical)
	duplicateIdx := indexPathFor(chunksDir, duplicate)

	record := func(contentHash string) string {
		return `{"source":"x","anchor":"a","content_hash":"` + contentHash + `","text":"t"}` + "\n"
	}

	cases := []struct {
		name  string
		files map[string]string
		want  bool
	}{
		{
			name:  "canonical has no index file at all (zero-chunk canonical) -> refused",
			files: map[string]string{duplicateIdx: record("sha256:a")},
			want:  false,
		},
		{
			name: "duplicate's records are all present in canonical's -> covered",
			files: map[string]string{
				canonicalIdx: record("sha256:a") + record("sha256:b"),
				duplicateIdx: record("sha256:a"),
			},
			want: true,
		},
		{
			name: "duplicate holds a record the canonical lacks -> refused",
			files: map[string]string{
				canonicalIdx: record("sha256:a"),
				duplicateIdx: record("sha256:a") + record("sha256:unique-to-duplicate"),
			},
			want: false,
		},
		{
			name:  "duplicate has no index file at all -> vacuously covered",
			files: map[string]string{canonicalIdx: record("sha256:a")},
			want:  true,
		},
		{
			name: "canonical index file present but empty on disk -> refused (can't verify anything)",
			files: map[string]string{
				canonicalIdx: "",
				duplicateIdx: record("sha256:a"),
			},
			want: false,
		},
	}

	for _, tc := range cases {
		files := tc.files
		deps := IngestDeps{
			ReadFile: func(path string) ([]byte, error) {
				content, ok := files[path]
				if !ok {
					return nil, fs.ErrNotExist
				}

				return []byte(content), nil
			},
			Stat: func(path string) (SourceStat, error) {
				if _, ok := files[path]; ok {
					return SourceStat{}, nil
				}

				return SourceStat{}, fs.ErrNotExist
			},
		}

		got := canonicalCoversDuplicateRecords(chunksDir, canonical, duplicate, deps)
		g.Expect(got).To(gomega.Equal(tc.want), tc.name)
	}
}

// TestIsVaultCopyOutsideVault pins Rule B's four early-outs plus its two
// sidecar-presence outcomes, since the integration test only exercises the
// no-sidecar and sidecar-present cases end to end.
func TestIsVaultCopyOutsideVault(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	statFound := func(string) (SourceStat, error) { return SourceStat{}, nil }
	statMissing := func(string) (SourceStat, error) { return SourceStat{}, io.ErrUnexpectedEOF }

	cases := []struct {
		name  string
		src   sourceRef
		vault string
		stat  func(string) (SourceStat, error)
		want  bool
	}{
		{
			name:  "explicit source is exempt even with a sidecar present",
			src:   sourceRef{path: "/snap/vault/1.md", explicit: true},
			vault: "/vault",
			stat:  statFound,
			want:  false,
		},
		{
			name:  "non-.md source is never a vault copy",
			src:   sourceRef{path: "/snap/vault/1.jsonl"},
			vault: "/vault",
			stat:  statFound,
			want:  false,
		},
		{
			name:  "a .md file already under the vault is the canonical copy",
			src:   sourceRef{path: "/vault/1.md"},
			vault: "/vault",
			stat:  statFound,
			want:  false,
		},
		{
			name:  "nil Stat cannot confirm a sidecar, so it never flags",
			src:   sourceRef{path: "/snap/vault/1.md"},
			vault: "/vault",
			stat:  nil,
			want:  false,
		},
		{
			name:  "no sidecar present: not a vault copy",
			src:   sourceRef{path: "/snap/vault/1.md"},
			vault: "/vault",
			stat:  statMissing,
			want:  false,
		},
		{
			name:  "sidecar present outside the vault: a stray vault copy",
			src:   sourceRef{path: "/snap/vault/1.md"},
			vault: "/vault",
			stat:  statFound,
			want:  true,
		},
	}

	for _, tc := range cases {
		got := isVaultCopyOutsideVault(tc.src, tc.vault, IngestDeps{Stat: tc.stat})
		g.Expect(got).To(gomega.Equal(tc.want), tc.name)
	}
}

// TestRawForRebuild pins each of rawForRebuild's branches directly: reusing
// already-read bytes, a fresh read (the rare promotion/self-heal case),
// tolerating a vanished non-explicit source, and erroring loudly for a
// vanished explicit one.
func TestRawForRebuild(t *testing.T) {
	t.Parallel()

	t.Run("reuses already-read bytes without a fresh read", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		deps := IngestDeps{ReadFile: func(string) ([]byte, error) {
			t.Fatal("must not read when raw bytes are already available")

			return nil, nil
		}}

		member := hashedSource{ref: sourceRef{path: "/a.md"}, raw: []byte("cached")}

		raw, skip, err := rawForRebuild(member, deps, io.Discard)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(skip).To(gomega.BeFalse())
		g.Expect(raw).To(gomega.Equal([]byte("cached")))
	})

	t.Run("reads fresh when raw was not read this run", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		deps := IngestDeps{ReadFile: func(string) ([]byte, error) { return []byte("fresh"), nil }}

		raw, skip, err := rawForRebuild(hashedSource{ref: sourceRef{path: "/a.md"}}, deps, io.Discard)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(skip).To(gomega.BeFalse())
		g.Expect(raw).To(gomega.Equal([]byte("fresh")))
	})

	t.Run("tolerates a vanished non-explicit source", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var out bytes.Buffer

		deps := IngestDeps{ReadFile: func(string) ([]byte, error) { return nil, io.ErrUnexpectedEOF }}
		member := hashedSource{ref: sourceRef{path: "/a.md", explicit: false}}

		raw, skip, err := rawForRebuild(member, deps, &out)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(skip).To(gomega.BeTrue())
		g.Expect(raw).To(gomega.BeNil())
		g.Expect(out.String()).To(gomega.ContainSubstring("skip /a.md"))
	})

	t.Run("errors loudly for a vanished explicit source", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		deps := IngestDeps{ReadFile: func(string) ([]byte, error) { return nil, io.ErrUnexpectedEOF }}
		member := hashedSource{ref: sourceRef{path: "/a.md", explicit: true}}

		_, skip, err := rawForRebuild(member, deps, io.Discard)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(skip).To(gomega.BeFalse())
	})
}

// TestSelectCanonicalIsOrderIndependent is a rapid property: shuffling a
// hash-group's candidate order must never change selectCanonical's winner.
// Candidates are drawn from a fixed pool spanning every origin/depth/path
// combination selectCanonical's precedence rule distinguishes, so the
// property exercises rule 5's tie-break as well as rules 1-4.
func TestSelectCanonicalIsOrderIndependent(t *testing.T) {
	t.Parallel()

	pool := []sourceRef{
		{path: "/repo/notes/a.md", explicit: true, origin: originExplicit},
		{path: "/repo/a.md", origin: originRepo},
		{path: "/repo/.claude/a.md", origin: originClaudeAncestor, ancestorDepth: 0},
		{path: "/home/.claude/a.md", origin: originClaudeAncestor, ancestorDepth: 1},
		{path: "/repo/.pi/a.md", origin: originPiAncestor, ancestorDepth: 0},
		{path: "/sessions/a.md", origin: originSessionLog},
		{path: "/tmp/sweep/a.md", origin: originManualSweep},
		{path: "/tmp/sweep/sub/dir/a.md", origin: originManualSweep},
	}

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := gomega.NewWithT(rt)

		size := rapid.IntRange(1, len(pool)).Draw(rt, "size")
		indices := rapid.Permutation(indexRange(len(pool))).Draw(rt, "order")[:size]

		candidates := make([]sourceRef, size)
		for i, idx := range indices {
			candidates[i] = pool[idx]
		}

		want := selectCanonical(candidates)

		// Shuffle via a second, independently-drawn permutation of the SAME
		// candidate set and assert the winner is identical.
		shuffledIdx := rapid.Permutation(indexRange(size)).Draw(rt, "shuffle")
		shuffled := make([]sourceRef, size)

		for i, idx := range shuffledIdx {
			shuffled[i] = candidates[idx]
		}

		got := selectCanonical(shuffled)

		gExpect.Expect(got).To(gomega.Equal(want),
			"selectCanonical must be order-independent for the same candidate set")
	})
}

// TestSelectCanonicalPrecedenceRules pins each of the five precedence rules
// down with a concrete example, so a future edit that silently reorders the
// rank table gets caught by name rather than only by the property test.
func TestSelectCanonicalPrecedenceRules(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	cases := []struct {
		name  string
		cands []sourceRef
		want  string
	}{
		{
			name: "rule 1: explicit beats everything",
			cands: []sourceRef{
				{path: "/repo/a.md", origin: originRepo},
				{path: "/other/a.md", explicit: true, origin: originExplicit},
			},
			want: "/other/a.md",
		},
		{
			name: "rule 2: repo beats ancestor",
			cands: []sourceRef{
				{path: "/repo/.claude/a.md", origin: originClaudeAncestor},
				{path: "/repo/a.md", origin: originRepo},
			},
			want: "/repo/a.md",
		},
		{
			name: "rule 3: ancestor beats anything-else",
			cands: []sourceRef{
				{path: "/sessions/a.md", origin: originSessionLog},
				{path: "/repo/.pi/a.md", origin: originPiAncestor},
			},
			want: "/repo/.pi/a.md",
		},
		{
			name: "rule 3: closest ancestor wins over a farther one",
			cands: []sourceRef{
				{path: "/home/.claude/a.md", origin: originClaudeAncestor, ancestorDepth: 1},
				{path: "/repo/.claude/a.md", origin: originClaudeAncestor, ancestorDepth: 0},
			},
			want: "/repo/.claude/a.md",
		},
		{
			name: "rule 4: manual-sweep and session-log are both anything-else",
			cands: []sourceRef{
				{path: "/tmp/sweep/a.md", origin: originManualSweep},
				{path: "/sessions/a.md", origin: originSessionLog},
			},
			// Neither rank differs (both rank 4); rule 5 tie-breaks by fewest
			// path separators, then lexicographic. "/sessions/a.md" has 2
			// separators vs "/tmp/sweep/a.md"'s 3.
			want: "/sessions/a.md",
		},
		{
			name: "rule 5: fewest path separators wins a same-rank tie",
			cands: []sourceRef{
				{path: "/repo/deep/nested/a.md", origin: originRepo},
				{path: "/repo/a.md", origin: originRepo},
			},
			want: "/repo/a.md",
		},
		{
			name: "rule 5: lexicographic breaks an equal-separator tie",
			cands: []sourceRef{
				{path: "/repo/b.md", origin: originRepo},
				{path: "/repo/a.md", origin: originRepo},
			},
			want: "/repo/a.md",
		},
	}

	for _, tc := range cases {
		got := selectCanonical(tc.cands)
		g.Expect(got.path).To(gomega.Equal(tc.want), tc.name)
	}
}

// indexRange returns [0, n).
func indexRange(n int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}

	return idx
}
