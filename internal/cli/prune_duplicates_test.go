package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestPruneDuplicatesCollapsesGroupToCanonical asserts a real run removes the
// duplicate's index file and manifest entry, keeps the canonical's index
// file and manifest entry untouched, and rewrites the manifest.
func TestPruneDuplicatesCollapsesGroupToCanonical(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical = "/repo/a.md"
		duplicate = "/repo/sub/b.md"
	)

	fs := newPruneFS()
	fs.files["/chunks/"+cli.ExportIndexFileName(canonical)] = []byte(`{"source":"a"}`)
	fs.files["/chunks/"+cli.ExportIndexFileName(duplicate)] = []byte(`{"source":"b"}`)
	// pruneFS.Exists is backed by a map separate from files (mirroring the
	// source-vs-index-path distinction the dead-source prune tests rely on),
	// so every index file genuinely present must ALSO be marked existing.
	fs.exists["/chunks/"+cli.ExportIndexFileName(canonical)] = true
	fs.exists["/chunks/"+cli.ExportIndexFileName(duplicate)] = true

	manifest := pruneDuplicatesManifest("sha256:x", canonical, duplicate)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, fs.pruneDeps(), io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	_, dupIdxPresent := fs.files["/chunks/"+cli.ExportIndexFileName(duplicate)]
	g.Expect(dupIdxPresent).To(gomega.BeFalse(), "duplicate's index file must be removed")

	_, canonicalIdxPresent := fs.files["/chunks/"+cli.ExportIndexFileName(canonical)]
	g.Expect(canonicalIdxPresent).To(gomega.BeTrue(), "canonical's index file must survive")

	var rewritten map[string]map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).NotTo(gomega.HaveKey(duplicate), "duplicate's manifest entry must be dropped")
	g.Expect(rewritten).To(gomega.HaveKey(canonical), "canonical's manifest entry must survive")
}

// TestPruneDuplicatesCollapsesStructuralRefusalsIntoSummary asserts the
// common case — the entire hash group is a zero-chunk source, so NEITHER
// the canonical NOR its duplicate ever had an index file — is reported as
// ONE bulk summary line, not one "refusing to remove ..." line per item.
// Gate B measured a real run printing 47,396 near-identical lines for
// exactly this case, burying the genuine removals and anomalies.
func TestPruneDuplicatesCollapsesStructuralRefusalsIntoSummary(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical = "/repo/a.md"
		duplicate = "/repo/sub/b.md"
	)

	fs := newPruneFS()
	// Neither path has an index file anywhere in fs.files/fs.exists: this is
	// the structural zero-chunk-source case, not a crash-window anomaly.

	manifest := pruneDuplicatesManifest("sha256:zero-chunk", canonical, duplicate)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	out := &strings.Builder{}

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, fs.pruneDeps(), out)
	g.Expect(err).NotTo(gomega.HaveOccurred(), "a structural refusal is not a removal failure")

	g.Expect(out.String()).NotTo(gomega.ContainSubstring("refusing to remove"),
		"the structural zero-chunk case must not print a per-item refusal line")
	g.Expect(out.String()).To(gomega.ContainSubstring(
		"refused 1 removal(s): canonical has no index file (zero-chunk sources; nothing to lose)"))

	var rewritten map[string]map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).To(gomega.HaveKey(duplicate), "refused duplicate's manifest entry must survive")
}

// TestPruneDuplicatesDropsManifestEntryWhenIndexAlreadyGone asserts the
// self-heal/idempotent edge covered by removeDuplicateIndex: a duplicate's
// OWN index file is already absent (e.g. a previous partial cleanup) while
// its canonical twin's index IS verified present. Remove must never be
// called for a file that isn't there, but the duplicate's stale manifest
// entry must still be dropped.
func TestPruneDuplicatesDropsManifestEntryWhenIndexAlreadyGone(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical = "/repo/a.md"
		duplicate = "/repo/sub/b.md" // its index file was never seeded below
	)

	fs := newPruneFS()
	fs.files["/chunks/"+cli.ExportIndexFileName(canonical)] = []byte(`{"source":"a"}`)
	fs.exists["/chunks/"+cli.ExportIndexFileName(canonical)] = true
	// duplicate's index file is deliberately absent from both files and exists.

	manifest := pruneDuplicatesManifest("sha256:z", canonical, duplicate)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	removeCalled := false
	deps := fs.pruneDeps()
	deps.Remove = func(path string) error {
		removeCalled = true

		delete(fs.files, path)

		return nil
	}

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(removeCalled).To(gomega.BeFalse(), "Remove must not be called for an index file that is already absent")

	var rewritten map[string]map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).NotTo(gomega.HaveKey(duplicate), "the stale manifest entry must still be dropped")
}

// TestPruneDuplicatesDryRunRemovesNothing asserts --duplicates --dry-run
// reports the group's collapse (1 removed, 1 retained) without calling
// WriteFile or Remove — nothing on disk changes.
func TestPruneDuplicatesDryRunRemovesNothing(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical = "/repo/a.md"     // 2 separators: selectCanonical's tie-break winner
		duplicate = "/repo/sub/b.md" // 3 separators
	)

	manifest := pruneDuplicatesManifest("sha256:x", canonical, duplicate)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	canonicalIdx := "/chunks/" + cli.ExportIndexFileName(canonical)
	duplicateIdx := "/chunks/" + cli.ExportIndexFileName(duplicate)

	deps := cli.PruneDeps{
		ReadFile: func(path string) ([]byte, error) {
			if path == "/chunks/manifest.json" {
				return manBytes, nil
			}

			return nil, io.ErrUnexpectedEOF
		},
		WriteFile: func(string, []byte) error {
			t.Fatal("WriteFile must not be called in --dry-run")

			return nil
		},
		Exists: func(path string) bool {
			return path == canonicalIdx || path == duplicateIdx
		},
		Remove: func(string) error {
			t.Fatal("Remove must not be called in --dry-run")

			return nil
		},
		LogWarning: func(format string, args ...any) {
			t.Fatalf("LogWarning must not be called in --dry-run: "+format, args...)
		},
	}

	out := &strings.Builder{}

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true, DryRun: true}, deps, out)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(out.String()).To(gomega.ContainSubstring("[dry-run] "))
	g.Expect(out.String()).To(gomega.ContainSubstring("removed 1"))
	g.Expect(out.String()).To(gomega.ContainSubstring("retained 1"))
}

// TestPruneDuplicatesManifestUnmarshalErrorPropagates asserts a malformed
// manifest.json surfaces a wrapped error rather than being silently
// swallowed or crashing.
func TestPruneDuplicatesManifestUnmarshalErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newPruneFS()
	fs.files["/chunks/manifest.json"] = []byte("{not valid json")

	err := cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, fs.pruneDeps(), io.Discard)

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("reading manifest")))
}

// TestPruneDuplicatesNeverGroupsAcrossChunkingClasses: a .md and a .jsonl
// manifest entry sharing the SAME file_hash must never be treated as
// duplicates of each other. chunkSource dispatches by extension (.jsonl ->
// transcript strip, everything else -> markdown on raw bytes), so two
// sources with identical bytes but different extensions hold genuinely
// different, non-empty chunk content (Gate B finding, shared with Unit 3's
// ingest-time dedup). Grouping by hash alone would let --duplicates
// permanently delete one's distinct chunking; both must survive untouched.
func TestPruneDuplicatesNeverGroupsAcrossChunkingClasses(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		mdPath    = "/repo/a.md"
		jsonlPath = "/repo/a.jsonl"
	)

	fs := newPruneFS()
	fs.files["/chunks/"+cli.ExportIndexFileName(mdPath)] = []byte(`{"source":"md"}`)
	fs.files["/chunks/"+cli.ExportIndexFileName(jsonlPath)] = []byte(`{"source":"jsonl"}`)
	fs.exists["/chunks/"+cli.ExportIndexFileName(mdPath)] = true
	fs.exists["/chunks/"+cli.ExportIndexFileName(jsonlPath)] = true

	manifest := pruneDuplicatesManifest("sha256:shared", mdPath, jsonlPath)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	removeCalled := false
	deps := fs.pruneDeps()
	deps.Remove = func(path string) error {
		removeCalled = true

		delete(fs.files, path)

		return nil
	}

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(removeCalled).To(gomega.BeFalse(),
		"a .md and a .jsonl sharing a file_hash must never be grouped as duplicates of each other")

	_, mdStillIndexed := fs.files["/chunks/"+cli.ExportIndexFileName(mdPath)]
	g.Expect(mdStillIndexed).To(gomega.BeTrue(), "the markdown source's index must survive untouched")

	_, jsonlStillIndexed := fs.files["/chunks/"+cli.ExportIndexFileName(jsonlPath)]
	g.Expect(jsonlStillIndexed).To(gomega.BeTrue(), "the jsonl source's index must survive untouched")

	g.Expect(fs.files["/chunks/manifest.json"]).To(gomega.Equal(manBytes),
		"the manifest must be left byte-identical: nothing was removed")
}

// TestPruneDuplicatesNoManifestIsNoOp asserts an absent manifest is treated
// as an empty index, not an error: --duplicates against a chunks dir with
// no manifest.json yet must report "no manifest" and exit cleanly.
func TestPruneDuplicatesNoManifestIsNoOp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newPruneFS() // empty — no manifest file

	out := &strings.Builder{}

	err := cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, fs.pruneDeps(), out)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(out.String()).To(gomega.ContainSubstring("no manifest"))
}

// TestPruneDuplicatesRefusesWhenCanonicalIndexMissing asserts that when the
// retained twin's index file is NOT present right now, the duplicate is
// refused rather than removed — the deletion-safety gate must block the
// removal outright, not merely warn after the fact.
func TestPruneDuplicatesRefusesWhenCanonicalIndexMissing(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical = "/repo/a.md"
		duplicate = "/repo/sub/b.md"
	)

	fs := newPruneFS()
	// Canonical's index file is deliberately ABSENT — the crash-window /
	// corrupt-manifest case the gate exists for. The duplicate's OWN index
	// file IS genuinely present (both in fs.files and fs.exists — pruneFS's
	// Exists is backed by a separate map from files, see fixtures above),
	// which is what makes this the ANOMALOUS case: a sibling's real content
	// survives, proving the group is not a zero-chunk source.
	fs.files["/chunks/"+cli.ExportIndexFileName(duplicate)] = []byte(`{"source":"b"}`)
	fs.exists["/chunks/"+cli.ExportIndexFileName(duplicate)] = true

	manifest := pruneDuplicatesManifest("sha256:x", canonical, duplicate)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	removeCalled := false
	deps := fs.pruneDeps()
	deps.Remove = func(path string) error {
		removeCalled = true

		return fs.pruneDeps().Remove(path)
	}

	out := &strings.Builder{}

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, deps, out)
	g.Expect(err).NotTo(gomega.HaveOccurred(), "a refusal is not a removal failure")

	g.Expect(removeCalled).To(gomega.BeFalse(), "Remove must never be called when the canonical twin is missing")

	_, dupIdxPresent := fs.files["/chunks/"+cli.ExportIndexFileName(duplicate)]
	g.Expect(dupIdxPresent).To(gomega.BeTrue(), "duplicate's index file must survive an unverified deletion")

	var rewritten map[string]map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).To(gomega.HaveKey(duplicate), "refused duplicate's manifest entry must survive")

	// The duplicate's own index file DOES exist (seeded above), so this
	// refusal is the anomalous case (a sibling's content survives even
	// though the selected canonical's own index does not) — it must be
	// printed individually, not folded into the bulk structural summary.
	g.Expect(out.String()).To(gomega.ContainSubstring("refusing to remove " + duplicate))
	g.Expect(out.String()).To(gomega.ContainSubstring("refused 1 removal(s) listed above"))
}

// TestPruneDuplicatesRefusesWhenDuplicateRecordsNotSubsetOfCanonical asserts
// the record-level subset gate a ship-readiness review found missing
// (2026-07-26): the canonical's index file EXISTS (the old, insufficient
// gate would have allowed removal), but the duplicate's own index holds a
// record — presumably accumulated from a past ingest of this path, before
// it became byte-identical to the canonical — whose content hash the
// canonical's index does not have. Removing the duplicate would destroy
// that record permanently, so the removal must be refused, classified as
// the anomalous case (both files hold real content), and printed
// individually rather than folded into the bulk structural summary.
func TestPruneDuplicatesRefusesWhenDuplicateRecordsNotSubsetOfCanonical(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical = "/repo/a.md"
		duplicate = "/repo/sub/b.md"
	)

	canonicalIdx := "/chunks/" + cli.ExportIndexFileName(canonical)
	duplicateIdx := "/chunks/" + cli.ExportIndexFileName(duplicate)

	fs := newPruneFS()
	fs.files[canonicalIdx] = []byte(
		`{"source":"a","anchor":"x","content_hash":"sha256:shared","text":"t"}` + "\n")
	fs.files[duplicateIdx] = []byte(
		`{"source":"b","anchor":"x","content_hash":"sha256:shared","text":"t"}` + "\n" +
			`{"source":"b","anchor":"y","content_hash":"sha256:unique-to-duplicate","text":"old"}` + "\n")
	fs.exists[canonicalIdx] = true
	fs.exists[duplicateIdx] = true

	manifest := pruneDuplicatesManifest("sha256:x", canonical, duplicate)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	removeCalled := false
	deps := fs.pruneDeps()
	deps.Remove = func(path string) error {
		removeCalled = true

		return fs.pruneDeps().Remove(path)
	}

	out := &strings.Builder{}

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, deps, out)
	g.Expect(err).NotTo(gomega.HaveOccurred(), "a refusal is not a removal failure")

	g.Expect(removeCalled).To(gomega.BeFalse(),
		"the duplicate holds a record the canonical does not — removal must be refused even though "+
			"the canonical's index file exists")

	_, dupIdxPresent := fs.files[duplicateIdx]
	g.Expect(dupIdxPresent).To(gomega.BeTrue(), "duplicate's index file must survive an unverified deletion")

	var rewritten map[string]map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).To(gomega.HaveKey(duplicate), "refused duplicate's manifest entry must survive")

	g.Expect(out.String()).To(gomega.ContainSubstring("refusing to remove " + duplicate))
	g.Expect(out.String()).To(gomega.ContainSubstring("refused 1 removal(s) listed above"))
}

// TestPruneDuplicatesRemovesWhenDuplicateRecordsAreSubsetOfCanonical asserts
// the positive case with REALISTIC, distinct content hashes (not the
// incidental empty-content-hash coincidence other fixtures share): the
// canonical's index is a proper superset of the duplicate's — every one of
// the duplicate's own records is present in the canonical's — so removal
// proceeds normally.
func TestPruneDuplicatesRemovesWhenDuplicateRecordsAreSubsetOfCanonical(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical = "/repo/a.md"
		duplicate = "/repo/sub/b.md"
	)

	canonicalIdx := "/chunks/" + cli.ExportIndexFileName(canonical)
	duplicateIdx := "/chunks/" + cli.ExportIndexFileName(duplicate)

	fs := newPruneFS()
	fs.files[canonicalIdx] = []byte(
		`{"source":"a","anchor":"x","content_hash":"sha256:shared","text":"t"}` + "\n" +
			`{"source":"a","anchor":"y","content_hash":"sha256:extra-in-canonical","text":"t2"}` + "\n")
	fs.files[duplicateIdx] = []byte(
		`{"source":"b","anchor":"x","content_hash":"sha256:shared","text":"t"}` + "\n")
	fs.exists[canonicalIdx] = true
	fs.exists[duplicateIdx] = true

	manifest := pruneDuplicatesManifest("sha256:x", canonical, duplicate)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, fs.pruneDeps(), io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	_, dupIdxPresent := fs.files[duplicateIdx]
	g.Expect(dupIdxPresent).To(gomega.BeFalse(),
		"the duplicate's records are all covered by the canonical's — removal must proceed")

	_, canonicalIdxPresent := fs.files[canonicalIdx]
	g.Expect(canonicalIdxPresent).To(gomega.BeTrue(), "canonical's index file must survive")

	var rewritten map[string]map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).NotTo(gomega.HaveKey(duplicate), "duplicate's manifest entry must be dropped")
}

// TestPruneDuplicatesReportsFailureAndContinues asserts a removal error for
// one duplicate is written to stderr as "[FAIL] <path>: <err>", does not
// stop the run (a sibling duplicate in the same hash group still gets
// removed), and the overall run exits non-zero naming "N of M removals
// failed".
func TestPruneDuplicatesReportsFailureAndContinues(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical  = "/repo/a.md"
		failing    = "/repo/sub/b.md"
		succeeding = "/repo/sub2/c.md"
	)

	fs := newPruneFS()
	fs.files["/chunks/"+cli.ExportIndexFileName(canonical)] = []byte(`{"source":"a"}`)
	fs.files["/chunks/"+cli.ExportIndexFileName(failing)] = []byte(`{"source":"b"}`)
	fs.files["/chunks/"+cli.ExportIndexFileName(succeeding)] = []byte(`{"source":"c"}`)
	fs.exists["/chunks/"+cli.ExportIndexFileName(canonical)] = true
	fs.exists["/chunks/"+cli.ExportIndexFileName(failing)] = true
	fs.exists["/chunks/"+cli.ExportIndexFileName(succeeding)] = true

	manifest := pruneDuplicatesManifest("sha256:y", canonical, failing, succeeding)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	failingIdx := "/chunks/" + cli.ExportIndexFileName(failing)
	removeErr := errors.New("disk full")

	deps := fs.pruneDeps()
	deps.Remove = func(path string) error {
		if path == failingIdx {
			return removeErr
		}

		delete(fs.files, path)

		return nil
	}

	out := &strings.Builder{}
	errOut := &strings.Builder{}

	deps.LogWarning = func(format string, args ...any) {
		_, _ = fmt.Fprintf(errOut, format+"\n", args...)
	}

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, deps, out)

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("1 of 2 removals failed")))

	g.Expect(errOut.String()).To(gomega.ContainSubstring("[FAIL] " + failing + ":"))

	_, failingIdxPresent := fs.files[failingIdx]
	g.Expect(failingIdxPresent).To(gomega.BeTrue(), "a failed removal must leave the index file in place")

	_, succeedingIdxPresent := fs.files["/chunks/"+cli.ExportIndexFileName(succeeding)]
	g.Expect(succeedingIdxPresent).To(gomega.BeFalse(), "a sibling duplicate's removal must still proceed")

	var rewritten map[string]map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).To(gomega.HaveKey(failing), "the failed removal's manifest entry must survive for retry")
	g.Expect(rewritten).NotTo(gomega.HaveKey(succeeding), "the successful removal's manifest entry must be dropped")
}

// TestPruneDuplicatesSecondRunIsNoOp asserts convergence: once a hash
// group has collapsed to its canonical, a second --duplicates run finds
// nothing left to remove and exits 0, leaving the manifest byte-identical.
func TestPruneDuplicatesSecondRunIsNoOp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical = "/repo/a.md"
		duplicate = "/repo/sub/b.md"
	)

	fs := newPruneFS()
	fs.files["/chunks/"+cli.ExportIndexFileName(canonical)] = []byte(`{"source":"a"}`)
	fs.files["/chunks/"+cli.ExportIndexFileName(duplicate)] = []byte(`{"source":"b"}`)
	fs.exists["/chunks/"+cli.ExportIndexFileName(canonical)] = true
	fs.exists["/chunks/"+cli.ExportIndexFileName(duplicate)] = true

	manifest := pruneDuplicatesManifest("sha256:x", canonical, duplicate)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	args := cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}
	deps := fs.pruneDeps()

	g.Expect(cli.RunPrune(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	_, dupIdxPresentAfterFirstRun := fs.files["/chunks/"+cli.ExportIndexFileName(duplicate)]
	g.Expect(dupIdxPresentAfterFirstRun).To(gomega.BeFalse(), "sanity: the first run must actually remove the duplicate")

	afterFirstRun := append([]byte(nil), fs.files["/chunks/manifest.json"]...)

	// A second run must not even ATTEMPT to write the manifest — json.Marshal
	// sorting map keys makes an unnecessary rewrite byte-identical, so a
	// byte-equality check alone would not catch a regression that dropped
	// the "only write if something changed" guard (Gate B finding).
	writeCalls := 0
	deps.WriteFile = func(path string, data []byte) error {
		writeCalls++

		return fs.pruneDeps().WriteFile(path, data)
	}

	out := &strings.Builder{}

	err = cli.RunPrune(context.Background(), args, deps, out)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(out.String()).To(gomega.ContainSubstring("removed 0"))
	g.Expect(writeCalls).To(gomega.Equal(0), "a no-op run must not call WriteFile at all")
	g.Expect(fs.files["/chunks/manifest.json"]).To(gomega.Equal(afterFirstRun),
		"a second run must leave the manifest byte-identical")
}

// TestPruneDuplicatesSkipsAlreadyMarkedDuplicates asserts entries Unit 3's
// forward-looking dedup already tagged DuplicateOf are left alone: they were
// never indexed (no index file to remove) and deleting their manifest entry
// would only force a needless re-read on the next ingest.
func TestPruneDuplicatesSkipsAlreadyMarkedDuplicates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical        = "/repo/a.md"
		alreadyDuplicate = "/repo/sub/b.md"
	)

	fs := newPruneFS()
	fs.files["/chunks/"+cli.ExportIndexFileName(canonical)] = []byte(`{"source":"a"}`)

	manBytes, err := json.Marshal(map[string]map[string]any{
		canonical: {"mtime_unix_nano": 1, "size": 10, "file_hash": "sha256:x"},
		alreadyDuplicate: {
			"mtime_unix_nano": 2, "size": 10, "file_hash": "sha256:x", "duplicate_of": canonical,
		},
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	removeCalled := false
	deps := fs.pruneDeps()
	deps.Remove = func(string) error {
		removeCalled = true

		return nil
	}

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(removeCalled).To(gomega.BeFalse(), "an already-marked duplicate has no index file to remove")

	var rewritten map[string]map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).To(gomega.HaveKey(alreadyDuplicate), "an already-marked duplicate's entry must survive untouched")
}

// TestPruneDuplicatesWriteManifestErrorPropagates asserts a WriteFile
// failure while persisting the collapsed manifest is wrapped and returned,
// not swallowed — the removals that already happened in memory must not be
// silently lost from the report.
func TestPruneDuplicatesWriteManifestErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		canonical = "/repo/a.md"
		duplicate = "/repo/sub/b.md"
	)

	fs := newPruneFS()
	fs.files["/chunks/"+cli.ExportIndexFileName(canonical)] = []byte(`{"source":"a"}`)
	fs.files["/chunks/"+cli.ExportIndexFileName(duplicate)] = []byte(`{"source":"b"}`)
	fs.exists["/chunks/"+cli.ExportIndexFileName(canonical)] = true
	fs.exists["/chunks/"+cli.ExportIndexFileName(duplicate)] = true

	manifest := pruneDuplicatesManifest("sha256:x", canonical, duplicate)
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs.files["/chunks/manifest.json"] = manBytes

	writeFailure := errors.New("disk full")
	deps := fs.pruneDeps()
	deps.WriteFile = func(string, []byte) error { return writeFailure }

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks", Duplicates: true}, deps, io.Discard)

	g.Expect(err).To(gomega.MatchError(writeFailure))
}

// pruneDuplicatesManifest builds a manifest fixture where every listed path
// shares the given content hash and carries no DuplicateOf tag — the
// pre-Unit-3 backlog shape `prune --duplicates` must retroactively collapse.
func pruneDuplicatesManifest(hash string, paths ...string) map[string]map[string]any {
	manifest := make(map[string]map[string]any, len(paths))
	for i, path := range paths {
		manifest[path] = map[string]any{
			"mtime_unix_nano": i + 1,
			"size":            10,
			"file_hash":       hash,
		}
	}

	return manifest
}
