package cli_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// ── Unit 5: --supersedes flag ─────────────────────────────────────────────────

// TestAmendFactNote_Supersedes_WrittenToFrontmatterAndBody proves that --supersedes
// on amend of a FACT note writes both frontmatter and body, exercising the
// overrideFactFields path with parsedSupersedes != nil.
func TestAmendFactNote_Supersedes_WrittenToFrontmatterAndBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1cc.2026-01-01.test.md"

	noteContent := []byte(
		"---\ntype: fact\ntier: L2\nsituation: ctx\nsubject: A\npredicate: has\nobject: B\n" +
			"luhmann: \"1cc\"\ncreated: 2026-01-01\nsource: test\n---\n\n" +
			"Information learned: when in ctx, A has B.\n\n",
	)

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1cc"}}, nil
		},
		Read:  func(string) ([]byte, error) { return noteContent, nil },
		Write: func(_ string, data []byte) error { written = data; return nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.AmendArgs{
		Vault:      "/vault",
		Target:     "1cc",
		Supersedes: []string{"9d.2026-01-01.old-fact|updates|the old fact was incomplete"},
	}

	var buf strings.Builder

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(written)

	g.Expect(body).To(ContainSubstring("supersedes:"), "fact note must have supersedes in frontmatter")
	g.Expect(body).To(ContainSubstring("note: 9d.2026-01-01.old-fact"))
	g.Expect(body).To(ContainSubstring("type: updates"))
	g.Expect(body).To(ContainSubstring(
		"Supersedes: [[9d.2026-01-01.old-fact]] — updates: the old fact was incomplete",
	))
}

// TestAmend_Supersedes_WrittenToFrontmatterAndBody proves that --supersedes on
// amend writes both channels and replaces any existing entries.
func TestAmend_Supersedes_WrittenToFrontmatterAndBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1bb.2026-01-01.test.md"

	noteContent := []byte(
		"---\ntype: feedback\ntier: L2\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
			"luhmann: \"1bb\"\ncreated: 2026-01-01\nsource: test\n---\n\nLesson learned: when ctx, a.\n\n",
	)

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1bb"}}, nil
		},
		Read:  func(string) ([]byte, error) { return noteContent, nil },
		Write: func(_ string, data []byte) error { written = data; return nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.AmendArgs{
		Vault:      "/vault",
		Target:     "1bb",
		Supersedes: []string{"9c.2026-01-01.old|refutes|old claim was wrong"},
	}

	var buf strings.Builder

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(written)

	g.Expect(body).To(ContainSubstring("supersedes:"))
	g.Expect(body).To(ContainSubstring("note: 9c.2026-01-01.old"))
	g.Expect(body).To(ContainSubstring("type: refutes"))
	g.Expect(body).To(ContainSubstring("Supersedes: [[9c.2026-01-01.old]] — refutes: old claim was wrong"))
}

// TestBuildSupersedesInverse proves the inverse scan helper correctly maps
// superseded→[]superseder.
func TestBuildSupersedesInverse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	input := map[string][]cli.ExportSupersedesEntry{
		"9z.2026-01-01.new-note": {
			{Note: "9a.2026-01-01.old-note", Type: "updates", Claim: "old action was wrong"},
			{Note: "9b.2026-01-01.other-old", Type: "narrows", Claim: "only applies to case X"},
		},
		"9y.2026-01-01.another-new": {
			{Note: "9a.2026-01-01.old-note", Type: "refutes", Claim: "directly contradicts"},
		},
	}

	inverse := cli.ExportBuildSupersedesInverse(input)

	// 9a.2026-01-01.old-note is superseded by two notes.
	g.Expect(inverse).To(HaveKey("9a.2026-01-01.old-note"))
	g.Expect(inverse["9a.2026-01-01.old-note"]).To(HaveLen(2))

	// 9b.2026-01-01.other-old is superseded by one note.
	g.Expect(inverse).To(HaveKey("9b.2026-01-01.other-old"))
	g.Expect(inverse["9b.2026-01-01.other-old"]).To(HaveLen(1))
}

// TestLearnFeedback_Supersedes_WrittenToFrontmatterAndBody proves that the
// --supersedes flag writes both the frontmatter list and the body line on learn.
func TestLearnFeedback_Supersedes_WrittenToFrontmatterAndBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var written []byte

	args := cli.LearnArgs{
		Type: "feedback", Slug: "supersedes-test", Position: "top",
		Source: "test", Situation: "testing supersedes",
		Behavior: "old behavior", Impact: "bad impact", Action: "new action",
		Supersedes: []string{"9a.2026-01-01.old-note|updates|old action was insufficient"},
	}
	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(_ string, data []byte) error { written = data; return nil },
	}

	var buf strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(written)

	g.Expect(body).To(ContainSubstring("supersedes:"),
		"frontmatter must contain supersedes: list")
	g.Expect(body).To(ContainSubstring("note: 9a.2026-01-01.old-note"),
		"frontmatter supersedes entry must have note field")
	g.Expect(body).To(ContainSubstring("type: updates"),
		"frontmatter supersedes entry must have type field")
	g.Expect(body).To(ContainSubstring("claim: old action was insufficient"),
		"frontmatter supersedes entry must have claim field")
	g.Expect(body).To(ContainSubstring("Supersedes: [[9a.2026-01-01.old-note]] — updates: old action was insufficient"),
		"body must contain Supersedes wikilink line")
}

func TestParseSupersedesFlag_ClaimWithPipeInContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// The claim may contain | if the flag uses SplitN(..., 3).
	entry, err := cli.ExportParseSupersedesFlag("note|updates|claim with | pipe")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entry.Claim).To(Equal("claim with | pipe"))
}

func TestParseSupersedesFlag_EmptyClaim_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportParseSupersedesFlag("note|updates|")
	g.Expect(err).To(HaveOccurred())
}

func TestParseSupersedesFlag_EmptyNote_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportParseSupersedesFlag("|updates|claim")
	g.Expect(err).To(HaveOccurred())
}

func TestParseSupersedesFlag_InvalidType_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportParseSupersedesFlag("note|corrects|claim")
	g.Expect(err).To(MatchError(ContainSubstring("type must be updates|narrows|refutes")))
}

func TestParseSupersedesFlag_MissingFields_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportParseSupersedesFlag("note|updates")
	g.Expect(err).To(MatchError(ContainSubstring("format must be")))
}

func TestParseSupersedesFlag_ValidNarrows(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	entry, err := cli.ExportParseSupersedesFlag("9a.2026-01-01.foo|narrows|narrowed scope")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entry.Type).To(Equal("narrows"))
}

func TestParseSupersedesFlag_ValidRefutes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	entry, err := cli.ExportParseSupersedesFlag("9b.2026-01-01.bar|refutes|the prior claim")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entry.Type).To(Equal("refutes"))
}

func TestParseSupersedesFlag_ValidUpdates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	entry, err := cli.ExportParseSupersedesFlag("105.2026-01-01.old-note|updates|the original claim here")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entry.Note).To(Equal("105.2026-01-01.old-note"))
	g.Expect(entry.Type).To(Equal("updates"))
	g.Expect(entry.Claim).To(Equal("the original claim here"))
}
