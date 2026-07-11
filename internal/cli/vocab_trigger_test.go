package cli_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

func TestCheckAndPersistVocabRefitTrigger_AlreadyPending_Idempotent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	centroids := cli.ExportVocabCentroidsDoc{
		RefitPending: true,
		RefitReason:  "growth: 40 notes, 15 days",
		LastRefit:    &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-15"},
	}

	centroidsData, marshalErr := json.Marshal(centroids)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	var writeCount int

	cli.ExportCheckAndPersistVocabRefitTrigger(
		"/vault",
		func(string) ([]string, error) { return []string{"1.note.md"}, nil },
		func(path string) ([]byte, error) {
			if strings.HasSuffix(path, "vocab.centroids.json") {
				return centroidsData, nil
			}

			return []byte("---\ntype: fact\n---\n"), nil
		},
		func(string, []byte) error { writeCount++; return nil },
		nil, time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
	)

	g.Expect(writeCount).To(Equal(0), "already-pending should not write again")
}

func TestCheckAndPersistVocabRefitTrigger_GrowthTrigger_SetsPending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Vault has 150 notes; last_refit was at 100, 20 days ago → growth trigger fires.
	names := make([]string, 150)
	for i := range names {
		names[i] = fmt.Sprintf("%d.2026-01-01.note.md", i+1)
	}

	centroids := cli.ExportVocabCentroidsDoc{
		SchemaVersion: 1,
		LastRefit:     &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-13"},
	}

	centroidsData, marshalErr := json.Marshal(centroids)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	var writtenData []byte

	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	cli.ExportCheckAndPersistVocabRefitTrigger(
		"/vault",
		func(string) ([]string, error) { return names, nil },
		func(path string) ([]byte, error) {
			if strings.HasSuffix(path, "vocab.centroids.json") {
				return centroidsData, nil
			}
			// note content: untagged (no vocab frontmatter key)
			return []byte("---\ntype: fact\n---\n"), nil
		},
		func(_ string, data []byte) error { writtenData = data; return nil },
		nil, now,
	)

	g.Expect(writtenData).NotTo(BeNil())

	var got cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(writtenData, &got)).NotTo(HaveOccurred())

	if err := json.Unmarshal(writtenData, &got); err != nil {
		return
	}

	g.Expect(got.RefitPending).To(BeTrue())
	g.Expect(got.RefitReason).To(ContainSubstring("growth"))
}

func TestCheckAndPersistVocabRefitTrigger_ListMDError_SeedsWithZeroCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// listMD returns an error → collectTriggerVaultStats returns (0,0,nil)
	// → checkAndPersist seeds last_refit with NoteCount: 0
	var writtenData []byte

	cli.ExportCheckAndPersistVocabRefitTrigger(
		"/vault",
		func(string) ([]string, error) { return nil, errors.New("list error") },
		func(string) ([]byte, error) { return nil, os.ErrNotExist },
		func(_ string, data []byte) error { writtenData = data; return nil },
		nil,
		time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
	)

	g.Expect(writtenData).NotTo(BeNil())

	var doc cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(writtenData, &doc)).NotTo(HaveOccurred())

	if err := json.Unmarshal(writtenData, &doc); err != nil {
		return
	}

	g.Expect(doc.LastRefit).NotTo(BeNil())

	if doc.LastRefit == nil {
		return
	}

	g.Expect(doc.LastRefit.NoteCount).To(Equal(0))
}

func TestCheckAndPersistVocabRefitTrigger_MissingCentroids_SeedsBaseline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// No centroids file → seeds last_refit, no trigger fires.
	names := []string{"1.2026-01-01.note.md", "2.2026-01-01.vocab-x-definition.md"}
	noteContent := "---\ntype: fact\ntierL2\nsituation: x\n---\n"
	definitionContent := "---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines x.\n"

	var writtenData []byte

	cli.ExportCheckAndPersistVocabRefitTrigger(
		"/vault",
		func(string) ([]string, error) { return names, nil },
		func(path string) ([]byte, error) {
			switch path {
			case "/vault/1.2026-01-01.note.md":
				return []byte(noteContent), nil
			case "/vault/2.2026-01-01.vocab-x-definition.md":
				return []byte(definitionContent), nil
			default:
				return nil, os.ErrNotExist
			}
		},
		func(_ string, data []byte) error { writtenData = data; return nil },
		nil,
		time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
	)

	g.Expect(writtenData).NotTo(BeNil())

	var doc cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(writtenData, &doc)).NotTo(HaveOccurred())

	if err := json.Unmarshal(writtenData, &doc); err != nil {
		return
	}

	g.Expect(doc.RefitPending).To(BeFalse(), "no trigger should fire on first seed")
	g.Expect(doc.LastRefit).NotTo(BeNil())

	if doc.LastRefit == nil {
		return
	}

	g.Expect(doc.LastRefit.NoteCount).To(Equal(1)) // definition note excluded by content, not filename
}

// ── Task 3: checkAndPersistVocabRefitTrigger ─────────────────────────────────

func TestCheckAndPersistVocabRefitTrigger_NilDeps_NoOp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// nil listMD → no panic, no write
	var written bool

	cli.ExportCheckAndPersistVocabRefitTrigger(
		"/vault", nil, func(string) ([]byte, error) { return nil, nil },
		func(string, []byte) error { written = true; return nil },
		nil, time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
	)

	g.Expect(written).To(BeFalse())
}

func TestCheckAndPersistVocabRefitTrigger_WriteError_LogsWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// No centroids file → seeds; writeFile errors → logWarn should be called.
	var loggedMsg string

	cli.ExportCheckAndPersistVocabRefitTrigger(
		"/vault",
		func(string) ([]string, error) { return []string{"1.note.md"}, nil },
		func(string) ([]byte, error) { return nil, os.ErrNotExist },
		func(string, []byte) error { return errors.New("disk full") },
		func(format string, args ...any) { loggedMsg = fmt.Sprintf(format, args...) },
		time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
	)

	g.Expect(loggedMsg).To(ContainSubstring("seeding last_refit"))
}

// ── Task 2: bare-vocab definition exemption ──────────────────────────────────

// TestCollectTriggerVaultStats_DefinitionsAreNeitherMembersNorUntagged
// verifies that a bare-vocab definition note contributes to neither the
// member tally nor the untagged tally — it is excluded from vault stats
// entirely, unlike a regular non-vocab-tagged note (which does count as
// untagged). The tagged-member fixture uses the tags: vocab/<term>
// convention, proving member detection also reads the new namespace.
func TestCollectTriggerVaultStats_DefinitionsAreNeitherMembersNorUntagged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	writeNote(t, vault, "1.2026-07-10.tagged-member.md",
		"---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nBody.\n")
	writeNote(t, vault, "2.2026-07-10.untagged-member.md", "---\ntype: fact\n---\n\nBody.\n")
	writeNote(t, vault, "3.2026-07-10.vocab-retrieval-design-definition.md",
		"---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines.\n")

	osFS := cli.ExportNewOsVaultFS()

	totalNotes, untaggedCount, _ := cli.ExportCollectTriggerVaultStats(vault, osFS.ListMD, osFS.ReadFile)

	g.Expect(totalNotes).To(Equal(2))
	g.Expect(untaggedCount).To(Equal(1))
}

func TestCollectTriggerVaultStats_WithVocabTagsAndNoFrontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Mix: tagged note, untagged note, no-frontmatter note, bare-vocab definition note.
	names := []string{"tagged.md", "untagged.md", "no-fm.md", "def.vocab-x-definition.md"}

	readFile := func(path string) ([]byte, error) {
		switch path {
		case "/vault/tagged.md":
			return []byte("---\ntype: fact\ntags:\n    - vocab/my-term\n---\nbody\n"), nil
		case "/vault/untagged.md":
			return []byte("---\ntype: fact\n---\nbody\n"), nil
		case "/vault/no-fm.md":
			return []byte("no frontmatter at all"), nil
		case "/vault/def.vocab-x-definition.md":
			return []byte("---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines x.\n"), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	totalNotes, untaggedCount, memberCounts := cli.ExportCollectTriggerVaultStats(
		"/vault",
		func(string) ([]string, error) { return names, nil },
		readFile,
	)

	// def.vocab-x-definition.md is excluded by CONTENT (bare vocab tag), not filename.
	g.Expect(totalNotes).To(Equal(3))
	g.Expect(untaggedCount).To(Equal(2)) // untagged.md + no-fm.md

	counts := memberCounts

	g.Expect(counts["my-term"]).To(Equal(1))
}

// ── Coverage helpers ──────────────────────────────────────────────────────────

func TestEvaluateVocabTriggers_GrowthBelowDaysFloor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// growth >= 40 but only 4 days: no fire
	last := &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-29"}
	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) // 4 days

	fired, _ := cli.ExportEvaluateVocabTriggers(141, 5, nil, last, now)

	g.Expect(fired).To(BeFalse())
}

// ── Task 2: evaluateVocabTriggers ────────────────────────────────────────────

func TestEvaluateVocabTriggers_GrowthFires(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// growth >= 40 AND >= 14d: fires
	last := &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-15"}
	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) // 18 days later

	fired, reason := cli.ExportEvaluateVocabTriggers(141, 5, nil, last, now)

	g.Expect(fired).To(BeTrue())
	g.Expect(reason).To(ContainSubstring("growth"))
}

func TestEvaluateVocabTriggers_HubFires(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	last := &cli.ExportVocabLastRefitDoc{NoteCount: 130, Date: "2026-06-01"}
	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	// term "x" has 30/100 = 30% > 25%
	counts := map[string]int{"x": 30, "y": 5}

	fired, reason := cli.ExportEvaluateVocabTriggers(100, 0, counts, last, now)

	g.Expect(fired).To(BeTrue())
	g.Expect(reason).To(ContainSubstring("hub"))
}

func TestEvaluateVocabTriggers_NilLastRefit_NoFire(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	fired, _ := cli.ExportEvaluateVocabTriggers(100, 5, nil, nil, now)

	g.Expect(fired).To(BeFalse())
}

func TestEvaluateVocabTriggers_UntaggedRateFires(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	last := &cli.ExportVocabLastRefitDoc{NoteCount: 130, Date: "2026-06-01"}
	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	// 10/100 = 10% > 8%
	fired, reason := cli.ExportEvaluateVocabTriggers(100, 10, nil, last, now)

	g.Expect(fired).To(BeTrue())
	g.Expect(reason).To(ContainSubstring("untagged"))
}

// TestVocabRefitSeed_MatchesTriggerCheckUnits is the final-review blocking-fix
// regression: RunVocabBootstrap seeds last_refit.NoteCount using the SAME
// content-based measure the trigger check itself uses
// (collectTriggerVaultStats) — both a bare-vocab DEFINITION note and a QA
// question note must be excluded from both sides identically. Before the fix,
// the seed used a filename-only count (countNonVocabNoteFiles) that included
// bare-vocab definition notes (their filenames carry no vocab-kind prefix —
// only isVocabDefinitionNote's CONTENT check excludes them), so the seed and
// the check diverged by exactly the definition-note count.
func TestVocabRefitSeed_MatchesTriggerCheckUnits(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	// Member notes: one tagged, one untagged — both count toward totalNotes
	// on both sides.
	writeNote(t, vault, "1.2026-07-09.tagged-member.md",
		"---\ntype: fact\ntags:\n    - vocab/existing-topic\n---\n\nTagged body.\n")
	writeNote(t, vault, "2.2026-07-09.untagged-member.md",
		"---\ntype: fact\n---\n\nUntagged body.\n")

	// A pre-existing bare-vocab DEFINITION note: content-based scans exclude
	// it entirely (isVocabDefinitionNote); the old filename-only seed counted
	// it as a plain note, since its filename has no vocab-kind prefix.
	writeNote(t, vault, "3.2026-07-09.vocab-legacy-term-definition.md",
		"---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines legacy-term.\n")

	// A QA question note — excluded by filename from both scans.
	writeNote(t, vault, "qa.1.q.md", "---\ntype: qa-question\n---\n\nWhat is X?\n")

	seed := []cli.SeedTerm{{Term: "eval-methodology", Description: "how we evaluate"}}
	seedYAML, marshalErr := yaml.Marshal(seed)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	seedPath := filepath.Join(vault, "seed.yaml")
	g.Expect(os.WriteFile(seedPath, seedYAML, 0o600)).To(Succeed())

	deps := cli.ExportNewOsVocabDeps()
	deps.Embedder = &fakeEmbedder{} // deterministic, no bundled-model cost

	args := cli.VocabBootstrapArgs{Vault: vault, SeedFile: seedPath, Floor: 0.30}

	var stdout strings.Builder

	// Bootstrap mints its own family note + the seed term's definition note
	// (both bare-vocab-tagged) INTO the vault before seeding last_refit — the
	// seed count must still exclude them, matching the trigger check.
	g.Expect(cli.RunVocabBootstrap(t.Context(), args, deps, &stdout)).To(Succeed())

	centroidsRaw, readErr := os.ReadFile(filepath.Join(vault, "vocab.centroids.json"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var doc cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(centroidsRaw, &doc)).To(Succeed())
	g.Expect(doc.LastRefit).NotTo(BeNil())

	if doc.LastRefit == nil {
		return
	}

	// The trigger check's own content-based measure, run fresh against the
	// vault's final on-disk state (seed-minted definition notes included).
	osFS := cli.ExportNewOsVaultFS()
	wantTotal, _, _ := cli.ExportCollectTriggerVaultStats(vault, osFS.ListMD, osFS.ReadFile)

	g.Expect(doc.LastRefit.NoteCount).To(Equal(wantTotal),
		"seeded last_refit.NoteCount must match the trigger check's content-based totalNotes exactly")

	// Sanity: only the two member notes count — every definition note
	// (pre-existing + the family note + the seed-minted term note) and the QA
	// question note must be excluded from the seed.
	g.Expect(doc.LastRefit.NoteCount).To(Equal(2),
		"seeded count must exclude definition notes (pre-existing + minted) and the QA question note")
}

func TestWriteCentroidsDocRaw_MarshalError_ReturnsWrappedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// NaN float32 in the Terms vector causes json.Marshal to fail.
	nan := float32(math.NaN())
	doc := cli.ExportVocabCentroidsDoc{
		Terms: map[string]cli.ExportVocabCentroidEntry{
			"x": {Vector: []float32{nan}},
		},
	}

	err := cli.ExportWriteCentroidsDocRaw("/vault", doc, func(string, []byte) error { return nil })

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("marshaling centroids"))
}
