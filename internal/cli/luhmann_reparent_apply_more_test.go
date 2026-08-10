package cli_test

import (
	"bytes"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestReparentIDLess_NumericAndNonNumericFallback covers both ordering
// branches: numeric IDs compare by value; non-numeric IDs (defensive —
// should not occur among derive's top-level candidates) fall back to string
// comparison.
func TestReparentIDLess_NumericAndNonNumericFallback(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportReparentIDLess("2", "10")).To(BeTrue(), "numeric: 2 < 10")
	g.Expect(cli.ExportReparentIDLess("10", "2")).To(BeFalse())
	g.Expect(cli.ExportReparentIDLess("1a", "2b")).To(BeTrue(), "non-numeric falls back to string comparison")
}

// TestRunReparentLuhmann_ApplyAnswersFileUnreadable covers the --answers
// ReadFile-error branch.
func TestRunReparentLuhmann_ApplyAnswersFileUnreadable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, renames := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "/missing-answers.json", false, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(renames).To(BeEmpty())
}

// TestRunReparentLuhmann_ApplyBadAnswersJSONRejected covers the malformed
// --answers JSON branch.
func TestRunReparentLuhmann_ApplyBadAnswersJSONRejected(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, renames := newReparentDeps(files, names)
	files["/answers.json"] = []byte("{not valid json")

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "/answers.json", false, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(renames).To(BeEmpty())
}

// TestRunReparentLuhmann_ApplyUnknownNoteRejected covers
// buildReparentRenameMap's unknown-note branch: an --answers entry naming a
// note ID that does not exist in the vault.
func TestRunReparentLuhmann_ApplyUnknownNoteRejected(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	answers := `{"reparenting":[{"note":"999","position":"continuation","target":"7"}],"fingerprint":"` +
		fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "/answers.json", false, deps, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("unknown note")))
	g.Expect(renames).To(BeEmpty())
}

// TestRunReparentLuhmann_DeriveExcerptUnreadableNoteIsEmpty covers
// reparentExcerpt's unreadable-note branch: a note has an embedding sidecar
// but its own .md file is missing, so its excerpt is empty rather than
// erroring the whole derive pass.
func TestRunReparentLuhmann_DeriveExcerptUnreadableNoteIsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	delete(files, "/vault/7.2026-01-01.first-note.md")

	deps, _ := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).NotTo(BeEmpty())
}
