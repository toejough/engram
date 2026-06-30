package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

func TestBuildChunkIDSet_ReturnsSourceAnchorKeys(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	sessionRecord := chunk.Record{
		Source: "/sessions/a.jsonl", Anchor: "turn-1",
		ContentHash: "sha256:aaa", Text: "hi", Vector: []float32{0.1},
	}
	docRecord := chunk.Record{
		Source: "/docs/b.md", Anchor: "Heading",
		ContentHash: "sha256:bbb", Text: "bye", Vector: []float32{0.2},
	}

	encoded1, err := chunk.EncodeRecords([]chunk.Record{sessionRecord})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	encoded2, err := chunk.EncodeRecords([]chunk.Record{docRecord})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	files := map[string][]byte{
		"/chunks/a.jsonl": encoded1,
		"/chunks/b.jsonl": encoded2,
	}
	readFile := func(path string) ([]byte, error) {
		data, ok := files[path]
		if !ok {
			return nil, fmt.Errorf("not found: %s", path)
		}

		return data, nil
	}
	listIndexes := func(string) ([]string, error) {
		return []string{"/chunks/a.jsonl", "/chunks/b.jsonl"}, nil
	}

	// Simulate the AmendDeps.LoadChunkIDs call pattern.
	ids, loadErr := cli.ExportBuildChunkIDSet("/chunks", listIndexes, readFile)
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	// Ids are source#anchor, NOT content hashes.
	g.Expect(ids["/sessions/a.jsonl#turn-1"]).To(BeTrue(), "r1 source#anchor must be in set")
	g.Expect(ids["/docs/b.md#Heading"]).To(BeTrue(), "r2 source#anchor must be in set")
	g.Expect(ids["sha256:aaa"]).To(BeFalse(), "content hash must NOT be in set")
	g.Expect(ids["nonexistent#anchor"]).To(BeFalse(), "absent id must not be in set")
}

func TestRunAmend_Activate_BumpsLastUsed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	noteContent := makeFactNote("ctx", "A", "has", "B", "")
	// A valid sidecar (correct schema version + matching Dims/vectors) so
	// bumpLastUsed's UnmarshalSidecar accepts it; the plan's bare LastUsed-only
	// fixture fails the schema-version guard and never writes.
	sidecarContent := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@1",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.2},
		ContentHash:      "sha256:x",
		LastUsed:         "2025-01-01",
	})

	var writtenPaths []string

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".vec.json") {
				return sidecarContent, nil
			}

			return noteContent, nil
		},
		Write: func(path string, _ []byte) error {
			writtenPaths = append(writtenPaths, path)

			return nil
		},
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:    "/vault",
		Target:   "1aa",
		Activate: true,
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Sidecar write must have happened (the .vec.json path)
	var hasSidecar bool

	for _, p := range writtenPaths {
		if strings.HasSuffix(p, ".vec.json") {
			hasSidecar = true
		}
	}

	g.Expect(hasSidecar).To(BeTrue(), "activate must write the sidecar")
}

func TestRunAmend_Activate_PreservesLastUsedAcrossReEmbed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test"

	// A content change triggers a re-embed, which writes a FRESH sidecar with
	// LastUsed="". --activate must then stamp today's date on that fresh sidecar,
	// so the final LastUsed is today — not "" (re-embed lost it) nor the stale date.
	noteContent := makeFactNote("ctx", "OldSubject", "has", "B", "")

	store := map[string][]byte{}

	var lastSidecar []byte

	embedCalled := false
	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read: func(path string) ([]byte, error) {
			if data, ok := store[path]; ok {
				return data, nil
			}

			return noteContent, nil
		},
		Write: func(path string, data []byte) error {
			store[path] = data
			if strings.HasSuffix(path, ".vec.json") {
				lastSidecar = data
			}

			return nil
		},
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now:      func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) },
		Embedder: &spyEmbedder{called: &embedCalled},
	}
	args := cli.AmendArgs{
		Vault:    "/vault",
		Target:   "1aa",
		Subject:  "NewSubject", // semantic change → re-embed
		Activate: true,
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(embedCalled).To(BeTrue(), "a subject change must re-embed")
	g.Expect(lastSidecar).NotTo(BeNil(), "a sidecar must have been written")

	final, unmarshalErr := embed.UnmarshalSidecar(lastSidecar)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(final.LastUsed).To(Equal("2026-06-01"),
		"activate must stamp today's LastUsed on the re-embedded sidecar")
}

func TestRunAmend_FieldReplacement_Fact_SubjectOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	noteContent := makeFactNote("ctx", "OldSubject", "has", "B", "")

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return noteContent, nil },
		Write:         func(_ string, data []byte) error { written = data; return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:   "/vault",
		Target:  "1aa",
		Subject: "NewSubject",
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(written)
	g.Expect(body).To(ContainSubstring("subject: NewSubject"))
	g.Expect(body).To(ContainSubstring("predicate: has"))
	g.Expect(body).To(ContainSubstring("object: B"))
	g.Expect(body).To(ContainSubstring("situation: ctx"))
	g.Expect(body).To(ContainSubstring("luhmann: \"1aa\""))
	// renderFactFrontmatter emits created as a double-quoted scalar (yaml.v3
	// quotes date-like strings) — matches learn output, not the plan's unquoted form.
	g.Expect(body).To(ContainSubstring("created: \"2026-01-01\""))
}

func TestRunAmend_FieldReplacement_Feedback_ActionAndProvMerge(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.fb.md"

	const dupID = "/sessions/s.jsonl#turn-1"

	const newID = "/sessions/s.jsonl#turn-2"

	// note already carries dupID in sources: so merging dupID again must dedup.
	noteContent := makeFeedbackNote("ctx", "did X", "broke Y", "do Z instead", dupID)

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return noteContent, nil },
		Write:         func(_ string, data []byte) error { written = data; return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{dupID: true, newID: true}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:        "/vault",
		Target:       "1aa",
		Action:       "do W instead",
		ChunkSources: []string{dupID, newID}, // dupID overlaps existing -> dedup path
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(written)
	g.Expect(body).To(ContainSubstring("type: feedback"))
	g.Expect(body).To(ContainSubstring("action: do W instead"))
	g.Expect(body).To(ContainSubstring("behavior: did X"))
	g.Expect(body).To(ContainSubstring(newID))
	// dupID present exactly once (existing + incoming deduped).
	g.Expect(strings.Count(body, dupID)).To(Equal(1))
	// body formula re-rendered with the new action.
	g.Expect(body).To(ContainSubstring("Lesson learned: when ctx, do W instead."))
}

func TestRunAmend_FieldReplacement_NoContentChange_NoReEmbed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	const relBasename = "105.2026-01-01.foo.md"

	noteContent := makeFactNote("ctx", "A", "has", "B", "")

	embedCalled := false
	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return noteContent, nil },
		Write:         func(string, []byte) error { return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename, relBasename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now:      func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		Embedder: &spyEmbedder{called: &embedCalled},
	}
	args := cli.AmendArgs{
		Vault:     "/vault",
		Target:    "1aa",
		Relations: []string{relBasename + "|why"},
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(embedCalled).To(BeFalse(), "relation-only change must not trigger re-embed")
}

func TestRunAmend_MalformedCreated_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	// A fact note whose created: date does not parse — applyTypedAmend decodes
	// the doc, then parseCreated must fail loud rather than write a note with a
	// zero-value date.
	noteContent := []byte(
		"---\ntype: fact\nsituation: x\nsubject: A\npredicate: has\nobject: B\n" +
			"luhmann: \"1aa\"\ncreated: not-a-date\nsource: test\n---\n\nbody\n",
	)

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return noteContent, nil },
		Write:         func(string, []byte) error { return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Subject: "NewSubject"}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("parsing created date")))
}

func TestRunAmend_NoFrontmatter_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.raw.md"

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return []byte("no frontmatter here\n"), nil },
		Write:         func(string, []byte) error { return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Situation: "y"}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("no parseable frontmatter")))
}

func TestRunAmend_NoteNotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{}, nil
		},
		Read:     func(string) ([]byte, error) { return nil, nil },
		Write:    func(string, []byte) error { return nil },
		Embedder: nil,
	}
	args := cli.AmendArgs{
		Vault:  "/vault",
		Target: "999",
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("note not found")))
}

func TestRunAmend_ProvMerge_ChunkSources_Written(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	const chunkID = "/sessions/s.jsonl#turn-1"

	noteContent := makeFactNote("ctx", "A", "has", "B", "")

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return noteContent, nil },
		Write:         func(_ string, data []byte) error { written = data; return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{chunkID: true}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:        "/vault",
		Target:       "1aa",
		ChunkSources: []string{chunkID},
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).To(ContainSubstring("sources:"))
	g.Expect(string(written)).To(ContainSubstring(chunkID))
}

func TestRunAmend_ProvMerge_UnresolvedChunkSource_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return makeFactNote("ctx", "A", "has", "B", ""), nil },
		Write:         func(string, []byte) error { return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil // empty — id won't resolve
		},
	}
	args := cli.AmendArgs{
		Vault:        "/vault",
		Target:       "1aa",
		ChunkSources: []string{"/sessions/s.jsonl#turn-1"},
		ChunksDir:    "/chunks",
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("unresolved chunk-source id")))
}

func TestRunAmend_ReEmbedFailure_WarnsAndContinues(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	noteContent := makeFactNote("ctx", "OldSubject", "has", "B", "")

	var logged string

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return noteContent, nil },
		Write:         func(string, []byte) error { return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		Embedder:   &failingEmbedder{},
		LogWarning: func(format string, a ...any) { logged = fmt.Sprintf(format, a...) },
	}
	args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Subject: "NewSubject"}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	// embed failure must NOT fail the amend (the note write already succeeded).
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(logged).To(ContainSubstring("embed failed"))
}

func TestRunAmend_RelationMerge_Idempotent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	const relBasename = "105.2026-01-01.foo.md"

	existing := "Related to:\n- [[" + relBasename + "]] — why.\n"
	noteContent := makeFactNote("ctx", "A", "has", "B", existing)

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:  func(string) ([]byte, error) { return noteContent, nil },
		Write: func(_ string, data []byte) error { written = data; return nil },
		ListBasenames: func(string) ([]string, error) {
			return []string{basename, relBasename}, nil
		},
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:     "/vault",
		Target:    "1aa",
		Relations: []string{relBasename + "|why"},
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Should contain the relation exactly once.
	body := string(written)
	count := strings.Count(body, "[["+relBasename+"]]")
	g.Expect(count).To(Equal(1))
}

func TestRunAmend_RelationMerge_Idempotent_MixedMdForm(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test"

	// The existing bullet carries a trailing ".md" (a hand-edited / legacy form),
	// while ListBasenames returns the bare basename (production: cli.go strips
	// ".md"), so resolveRelationTargetsStrict yields the bare target. The dedup
	// must still recognise the duplicate — otherwise a re-run appends a second
	// bullet and amend is not idempotent.
	const relBare = "105.2026-01-01.foo"

	existing := "Related to:\n- [[" + relBare + ".md]] — why.\n"
	noteContent := makeFactNote("ctx", "A", "has", "B", existing)

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:  func(string) ([]byte, error) { return noteContent, nil },
		Write: func(_ string, data []byte) error { written = data; return nil },
		ListBasenames: func(string) ([]string, error) {
			return []string{basename, relBare}, nil
		},
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:     "/vault",
		Target:    "1aa",
		Relations: []string{"105|why"},
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// The bare basename must appear exactly once (the existing ".md" bullet),
	// with no second bullet appended for the ".md"-stripped resolved target.
	g.Expect(strings.Count(string(written), relBare)).To(Equal(1),
		"dedup must normalize a trailing .md so the merge stays idempotent")
}

func TestRunAmend_RelationMerge_NewRelationAdded_ToExisting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	const relA = "105.2026-01-01.foo.md"

	const relB = "106.2026-01-01.bar.md"

	existing := "Related to:\n- [[" + relA + "]] — first.\n"
	noteContent := makeFactNote("ctx", "A", "has", "B", existing)

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:  func(string) ([]byte, error) { return noteContent, nil },
		Write: func(_ string, data []byte) error { written = data; return nil },
		ListBasenames: func(string) ([]string, error) {
			return []string{basename, relA, relB}, nil
		},
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:     "/vault",
		Target:    "1aa",
		Relations: []string{relB + "|second"},
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(written)
	g.Expect(body).To(ContainSubstring("[[" + relA + "]]"))
	g.Expect(body).To(ContainSubstring("[[" + relB + "]]"))
}

func TestRunAmend_RelationMerge_NewRelation_Appended(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	const relBasename = "105.2026-01-01.foo.md"

	noteContent := makeFactNote("ctx", "A", "has", "B", "")

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:  func(string) ([]byte, error) { return noteContent, nil },
		Write: func(_ string, data []byte) error { written = data; return nil },
		ListBasenames: func(string) ([]string, error) {
			return []string{basename, relBasename}, nil
		},
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:     "/vault",
		Target:    "1aa",
		Relations: []string{relBasename + "|why"},
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).To(ContainSubstring("Related to:"))
	g.Expect(string(written)).To(ContainSubstring("[[" + relBasename + "]]"))
}

func TestRunAmend_RelationMerge_UnresolvedRelation_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	noteContent := makeFactNote("ctx", "A", "has", "B", "")

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return noteContent, nil },
		Write:         func(string, []byte) error { return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:     "/vault",
		Target:    "1aa",
		Relations: []string{"999|why"},
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("unresolved relation target")))
}

// TestRunAmend_ResolvesTargetWithMdSuffix asserts that RunAmend resolves a
// --target passed as "basename.md" (the form emitted by the recall skill's
// Step-2.5C amend calls) rather than returning "note not found".
func TestRunAmend_ResolvesTargetWithMdSuffix(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1.linting"

	noteContent := makeFactNote("ctx", "A", "has", "B", "")
	writeCalled := false

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1"}}, nil
		},
		Read:          func(string) ([]byte, error) { return noteContent, nil },
		Write:         func(string, []byte) error { writeCalled = true; return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{
		Vault:  "/vault",
		Target: "1.linting.md", // .md-suffixed form — must resolve to "1.linting"
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writeCalled).To(BeTrue(), "amend must write the note when target resolves via .md suffix")
}

func TestRunAmend_RoundTrip_FactNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chunksDir := t.TempDir()

	// Vault basenames are the .md-stripped keys (vaultgraph.ParseBasename); the
	// real on-disk files carry the .md extension. The --relation target is the
	// bare Luhmann id (Obsidian convention), which the strict resolver expands
	// to the full basename.
	const noteKey = "1aa.2026-01-01.test"

	const relKey = "105.2026-01-01.foo"

	const chunkID = "/sessions/s.jsonl#turn-1"

	notePath := filepath.Join(dir, noteKey+".md")

	// write initial note
	noteContent := makeFactNote("original ctx", "OldSubject", "has", "B", "")
	err := os.WriteFile(notePath, noteContent, 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// write a chunk index with one record
	records := []chunk.Record{{
		Source: "/sessions/s.jsonl", Anchor: "turn-1",
		ContentHash: chunk.HashText("t"),
		Text:        "t", Vector: []float32{0.1},
	}}
	data, _ := chunk.EncodeRecords(records)
	err = os.WriteFile(filepath.Join(chunksDir, "s.jsonl"), data, 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// write a relation-target note
	err = os.WriteFile(filepath.Join(dir, relKey+".md"), makeFactNote("r ctx", "X", "is", "Y", ""), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Use the production deps end-to-end (real Scan/Read/Write/ListBasenames),
	// overriding only the clock for determinism. This exercises newOsAmendDeps.
	deps := cli.ExportNewOsAmendDeps()
	deps.Now = func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) }

	args := cli.AmendArgs{
		Vault:        dir,
		Target:       "1aa",
		Subject:      "NewSubject",
		Relations:    []string{"105|because"},
		ChunkSources: []string{chunkID},
		ChunksDir:    chunksDir,
	}

	var buf bytes.Buffer

	err = cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Read the amended note back from disk (the production Write persisted it).
	final, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	body := string(final)
	g.Expect(body).To(ContainSubstring("subject: NewSubject"))
	g.Expect(body).To(ContainSubstring("luhmann: \"1aa\""))
	g.Expect(body).To(ContainSubstring("created: \"2026-01-01\""))
	g.Expect(body).To(ContainSubstring("sources:"))
	g.Expect(body).To(ContainSubstring(chunkID))
	g.Expect(body).To(ContainSubstring("Related to:"))
	g.Expect(body).To(ContainSubstring("[[" + relKey + "]]"))
}

func TestRunAmend_UnknownNoteType_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.weird.md"

	// a note whose type is neither fact nor feedback
	noteContent := []byte("---\ntype: episode\nsituation: x\ncreated: 2026-01-01\nsource: test\n---\n\nbody\n")

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:          func(string) ([]byte, error) { return noteContent, nil },
		Write:         func(string, []byte) error { return nil },
		ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Situation: "y"}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("unknown note type")))
}

// spyEmbedder is an embed.Embedder that records whether Embed was called.
type spyEmbedder struct {
	called *bool
}

func (s *spyEmbedder) Dims() int { return 1 }

func (s *spyEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	if s.called != nil {
		*s.called = true
	}

	return []float32{0.1}, nil
}

func (s *spyEmbedder) ModelID() string { return "spy" }

// makeFactNote renders a minimal fact note (frontmatter + formula + optional
// related section) for amend tests.
func makeFactNote(situation, subject, predicate, object, relatedSection string) []byte {
	frontmatter := "---\ntype: fact\ntier: L2\n" +
		fmt.Sprintf("situation: %s\nsubject: %s\npredicate: %s\nobject: %s\n", situation, subject, predicate, object) +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\n"
	formula := fmt.Sprintf("Information learned: when in %s, %s %s %s.\n", situation, subject, predicate, object)

	return []byte(frontmatter + formula + "\n" + relatedSection)
}

// makeFeedbackNote renders a minimal feedback note. When source is non-empty it
// is recorded as a single sources: provenance entry, exercising the
// existing-sources merge path.
func makeFeedbackNote(situation, behavior, impact, action, chunkSource string) []byte {
	frontmatter := "---\ntype: feedback\ntier: L2\n" +
		fmt.Sprintf("situation: %s\nbehavior: %s\nimpact: %s\naction: %s\n", situation, behavior, impact, action) +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n"
	if chunkSource != "" {
		frontmatter += fmt.Sprintf("sources:\n  - %s\n", chunkSource)
	}

	frontmatter += "---\n\n"
	formula := fmt.Sprintf("Lesson learned: when %s, %s.\n", situation, action)

	return []byte(frontmatter + formula + "\n")
}
