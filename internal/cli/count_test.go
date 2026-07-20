package cli_test

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestRunCount_AbsentAttrBucket covers TDD item 8 (absent): notes lacking the
// group-by attr surface in the "(attr absent): N" bucket, not dropped.
func TestRunCount_AbsentAttrBucket(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"a.md": "---\ntype: feedback\nvocab: [x]\n---\nbody\n",
		"b.md": "---\ntype: feedback\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "vocab"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("x\t1\n(vocab absent): 1\ntotal: 2\n"))
}

// TestRunCount_BacklinksExceedGroupByForNonMemberLinkers locks in the real-vault
// relationship the clean agreement test does not cover: --backlinks-of counts
// EVERY linker (matching Obsidian's backlinks panel), while --group-by counts
// only frontmatter members. A non-member index/MOC page that links a value node
// without listing it in frontmatter makes backlinks-of exceed group-by by the
// number of such non-member linkers — a legitimate divergence, not a bug.
func TestRunCount_BacklinksExceedGroupByForNonMemberLinkers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"n1.md":        "---\ntype: feedback\nfoo: [alpha]\n---\nmember links [[foo.alpha]]\n",
		"idx.md":       "---\ntype: vocab-index\n---\nindex page links [[foo.alpha]]\n",
		"foo.alpha.md": "---\ntype: fact\n---\nalpha node\n",
	})
	deps := fixture.deps()

	var groupOut strings.Builder

	groupErr := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "foo"}, deps, &groupOut)
	g.Expect(groupErr).NotTo(HaveOccurred())

	if groupErr != nil {
		return
	}

	counts := parseGroupByCounts(groupOut.String())
	g.Expect(counts["alpha"]).To(Equal(1),
		"only the frontmatter member counts toward group-by — the index page does not")

	var linkOut strings.Builder

	linkErr := cli.RunCount(cli.CountArgs{Vault: countVault, BacklinksOf: "foo.alpha"}, deps, &linkOut)
	g.Expect(linkErr).NotTo(HaveOccurred())

	if linkErr != nil {
		return
	}

	g.Expect(linkOut.String()).To(Equal("in-degree: 2\nidx\nn1\n"),
		"backlinks-of counts every linker including the non-member index page")
	g.Expect(parseInDegree(linkOut.String())).To(Equal(counts["alpha"]+1),
		"backlinks-of exceeds group-by by the number of non-member (MOC/index) linkers")
}

// TestRunCount_BacklinksOf covers TDD item 6: in-degree + sorted linkers, and an
// unknown node → 0 with no linkers.
func TestRunCount_BacklinksOf(t *testing.T) {
	t.Parallel()

	fixture := newCountFS(map[string]string{
		"A.md": "links to [[C]]\n",
		"B.md": "links to [[C]]\n",
		"C.md": "no links here\n",
	})

	t.Run("known node lists sorted linkers", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var out strings.Builder

		err := cli.RunCount(cli.CountArgs{Vault: countVault, BacklinksOf: "C"}, fixture.deps(), &out)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(out.String()).To(Equal("in-degree: 2\nA\nB\n"))
	})

	t.Run("unknown node is zero with no linkers", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var out strings.Builder

		err := cli.RunCount(cli.CountArgs{Vault: countVault, BacklinksOf: "Z"}, fixture.deps(), &out)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(out.String()).To(Equal("in-degree: 0\n"))
	})
}

// TestRunCount_BadFilterErrors verifies that a --filter without '=' is rejected.
func TestRunCount_BadFilterErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{"a.md": "---\ntype: feedback\n---\nbody\n"})

	var out strings.Builder

	err := cli.RunCount(
		cli.CountArgs{Vault: countVault, GroupBy: "type", Filter: []string{"noequals"}},
		fixture.deps(), &out)
	g.Expect(err).To(MatchError(cli.ErrCountBadFilterForTest))
}

// TestRunCount_BothModesError verifies that supplying BOTH --group-by and
// --backlinks-of is rejected — the two modes are mutually exclusive.
func TestRunCount_BothModesError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{"a.md": "---\ntype: feedback\n---\nbody\n"})

	var out strings.Builder

	err := cli.RunCount(
		cli.CountArgs{Vault: countVault, GroupBy: "type", BacklinksOf: "a"},
		fixture.deps(), &out)
	g.Expect(err).To(MatchError(cli.ErrCountBothModesForTest))
}

// TestRunCount_EmptyFilterResultPrintsNothing covers TDD item 8 (empty): filters
// matching zero notes produce no output at all.
func TestRunCount_EmptyFilterResultPrintsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"a.md": "---\ntype: feedback\nvocab: [x]\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(
		cli.CountArgs{Vault: countVault, GroupBy: "vocab", Filter: []string{"type=nonexistent"}},
		fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(BeEmpty(), "an empty filter result prints nothing")
}

// TestRunCount_FilterListContains covers TDD item 4 (list): --filter
// vocab=scope-discipline restricts on list membership.
func TestRunCount_FilterListContains(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"a.md": "---\ntype: feedback\nvocab: [scope-discipline, other]\n---\nbody\n",
		"b.md": "---\ntype: fact\nvocab: [unrelated]\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(
		cli.CountArgs{Vault: countVault, GroupBy: "type", Filter: []string{"vocab=scope-discipline"}},
		fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("feedback\t1\ntotal: 1\n"))
}

// TestRunCount_FilterScalarEquality covers TDD item 4 (scalar): --filter
// type=feedback restricts the note set before counting.
func TestRunCount_FilterScalarEquality(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"a.md": "---\ntype: feedback\nvocab: [x]\n---\nbody\n",
		"b.md": "---\ntype: fact\nvocab: [y]\n---\nbody\n",
		"c.md": "---\ntype: feedback\nvocab: [x, z]\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(
		cli.CountArgs{Vault: countVault, GroupBy: "vocab", Filter: []string{"type=feedback"}},
		fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("x\t2\nz\t1\ntotal: 2\n"))
}

// TestRunCount_GroupByBacklinksAgreement covers TDD item 7 (the acceptance
// proof): over a dual-representation fixture — frontmatter attr AND matching
// [[attr.val]] wikilinks agree, including a duplicate-listing note — the
// group-by per-value count equals the backlinks-of in-degree for every value.
func TestRunCount_GroupByBacklinksAgreement(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"n1.md":        "---\ntype: feedback\nfoo: [alpha, beta]\n---\nlinks [[foo.alpha]] and [[foo.beta]]\n",
		"n2.md":        "---\ntype: feedback\nfoo: [alpha]\n---\nlinks [[foo.alpha]]\n",
		"n3.md":        "---\ntype: feedback\nfoo: [alpha, alpha]\n---\nlinks [[foo.alpha]] [[foo.alpha]]\n",
		"foo.alpha.md": "---\ntype: fact\n---\nalpha node\n",
		"foo.beta.md":  "---\ntype: fact\n---\nbeta node\n",
	})
	deps := fixture.deps()

	var groupOut strings.Builder

	groupErr := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "foo"}, deps, &groupOut)
	g.Expect(groupErr).NotTo(HaveOccurred())

	if groupErr != nil {
		return
	}

	counts := parseGroupByCounts(groupOut.String())
	g.Expect(counts).To(Equal(map[string]int{"alpha": 3, "beta": 1}),
		"group-by must count distinct frontmatter membership")

	for value, count := range counts {
		var linkOut strings.Builder

		linkErr := cli.RunCount(
			cli.CountArgs{Vault: countVault, BacklinksOf: "foo." + value}, deps, &linkOut)
		g.Expect(linkErr).NotTo(HaveOccurred())

		if linkErr != nil {
			return
		}

		g.Expect(parseInDegree(linkOut.String())).To(Equal(count),
			"group-by count for %q must equal backlinks-of in-degree of foo.%s", value, value)
	}
}

// TestRunCount_GroupByDedupMembership covers TDD item 3: a note listing the same
// value twice counts once toward it (locks the InDegree-agreement contract).
func TestRunCount_GroupByDedupMembership(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"a.md": "---\ntype: feedback\nvocab: [dup, dup, other]\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "vocab"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("dup\t1\nother\t1\ntotal: 1\n"),
		"a value listed twice within one note must count once")
}

// TestRunCount_GroupByListAttr covers TDD item 2: a list attr adds 1 to each of
// its distinct element buckets.
func TestRunCount_GroupByListAttr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"a.md": "---\ntype: feedback\nvocab: [x, y, z]\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "vocab"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("x\t1\ny\t1\nz\t1\ntotal: 1\n"))
}

// TestRunCount_GroupByScalarAttr covers TDD item 1: a scalar frontmatter attr
// (type) counts each note once toward its value.
func TestRunCount_GroupByScalarAttr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"a.md": "---\ntype: feedback\n---\nbody\n",
		"b.md": "---\ntype: fact\n---\nbody\n",
		"c.md": "---\ntype: feedback\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "type"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("feedback\t2\nfact\t1\ntotal: 3\n"))
}

// TestRunCount_GroupBySumProperty covers TDD item 9: the sum of group-by counts
// equals the number of DISTINCT (note, value) memberships, order-independent.
func TestRunCount_GroupBySumProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		noteCount := rapid.IntRange(0, 6).Draw(rt, "noteCount")
		alphabet := rapid.SampledFrom([]string{"a", "b", "c", "d"})

		notes := map[string]string{}
		expected := map[string]int{}

		for i := range noteCount {
			tokens := rapid.SliceOfN(alphabet, 0, 5).Draw(rt, "tokens")
			notes[strconv.Itoa(i)+".md"] =
				"---\ntype: feedback\nfoo: [" + strings.Join(tokens, ", ") + "]\n---\nbody\n"

			for _, value := range distinctStrings(tokens) {
				expected[value]++
			}
		}

		fixture := newCountFS(notes)

		var out strings.Builder

		err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "foo"}, fixture.deps(), &out)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(parseGroupByCounts(out.String())).To(Equal(expected),
			"per-value counts must equal distinct (note, value) memberships")
	})
}

// TestRunCount_ListErrorPropagates verifies that a ListMD failure surfaces as a
// wrapped error in group-by mode.
func TestRunCount_ListErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.CountDeps{
		ListMD:   func(string) ([]string, error) { return nil, &testNotFoundError{path: countVault} },
		ReadFile: func(string) ([]byte, error) { return nil, nil },
		Scan:     func(string) ([]vaultgraph.Note, error) { return nil, nil },
	}

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "type"}, deps, &out)
	g.Expect(err).To(HaveOccurred())
}

// TestRunCount_MultipleFiltersAreANDed covers TDD item 5: two filters are
// AND-ed — a note matching one but not both is excluded.
func TestRunCount_MultipleFiltersAreANDed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"match.md":      "---\ntype: feedback\nvocab: [x]\n---\nbody\n",
		"wrongvocab.md": "---\ntype: feedback\nvocab: [y]\n---\nbody\n",
		"wrongtype.md":  "---\ntype: fact\nvocab: [x]\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(
		cli.CountArgs{
			Vault:   countVault,
			GroupBy: "vocab",
			Filter:  []string{"type=feedback", "vocab=x"},
		},
		fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("x\t1\ntotal: 1\n"),
		"only the note satisfying BOTH filters must be counted")
}

// TestRunCount_NoModeErrors verifies that neither --group-by nor --backlinks-of
// is an error (nothing to count).
func TestRunCount_NoModeErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{"a.md": "---\ntype: feedback\n---\nbody\n"})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault}, fixture.deps(), &out)
	g.Expect(err).To(MatchError(cli.ErrCountNoModeForTest))
}

// TestRunCount_OsDepsReadRealVault exercises newCountDeps over a real-FS
// EdgeFS against a real on-disk vault, covering the ListMD, ReadFile, and
// Scan wiring for both modes.
func TestRunCount_OsDepsReadRealVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	writeVaultFile(t, vault, "m1.md", "---\ntype: feedback\nfoo: [alpha]\n---\nlinks [[foo.alpha]]\n")
	writeVaultFile(t, vault, "foo.alpha.md", "---\ntype: fact\n---\nalpha node\n")

	deps := cli.ExportNewCountDeps(realFSForTest())

	var groupOut strings.Builder

	groupErr := cli.RunCount(cli.CountArgs{Vault: vault, GroupBy: "foo"}, deps, &groupOut)
	g.Expect(groupErr).NotTo(HaveOccurred())

	if groupErr != nil {
		return
	}

	g.Expect(groupOut.String()).To(ContainSubstring("alpha\t1"))

	var linkOut strings.Builder

	linkErr := cli.RunCount(cli.CountArgs{Vault: vault, BacklinksOf: "foo.alpha"}, deps, &linkOut)
	g.Expect(linkErr).NotTo(HaveOccurred())

	if linkErr != nil {
		return
	}

	g.Expect(linkOut.String()).To(Equal("in-degree: 1\nm1\n"))
}

// TestRunCount_ScanErrorPropagates verifies that a Scan failure surfaces as a
// wrapped error in backlinks mode.
func TestRunCount_ScanErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.CountDeps{
		ListMD:   func(string) ([]string, error) { return nil, nil },
		ReadFile: func(string) ([]byte, error) { return nil, nil },
		Scan:     func(string) ([]vaultgraph.Note, error) { return nil, &testNotFoundError{path: countVault} },
	}

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, BacklinksOf: "C"}, deps, &out)
	g.Expect(err).To(HaveOccurred())
}

// TestRunCount_SkipsUncountableNotes verifies that notes which are unreadable,
// carry no frontmatter, or have malformed YAML are skipped (not counted) — the
// three defensive branches of readNoteAttrs.
func TestRunCount_SkipsUncountableNotes(t *testing.T) {
	t.Parallel()

	t.Run("no-frontmatter and malformed-yaml notes are skipped", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newCountFS(map[string]string{
			"good.md":    "---\ntype: feedback\n---\nbody\n",
			"nofm.md":    "just a body with no frontmatter\n",
			"badyaml.md": "---\ntype: [unterminated\n---\nbody\n",
		})

		var out strings.Builder

		err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "type"}, fixture.deps(), &out)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(out.String()).To(Equal("feedback\t1\ntotal: 1\n"),
			"only the well-formed note must be counted")
	})

	t.Run("unreadable notes are skipped", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		deps := cli.CountDeps{
			ListMD:   func(string) ([]string, error) { return []string{"ghost.md"}, nil },
			ReadFile: func(path string) ([]byte, error) { return nil, &testNotFoundError{path: path} },
			Scan:     func(string) ([]vaultgraph.Note, error) { return nil, nil },
		}

		var out strings.Builder

		err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "type"}, deps, &out)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(out.String()).To(BeEmpty(), "an unreadable note is skipped, leaving nothing to print")
	})
}

// TestTargets_CountEmptyVault exercises the count closure end-to-end through
// Targets() so the newCountDeps + resolveVault wiring is covered. An empty
// vault matches zero notes, so group-by prints nothing and stderr stays clean.
func TestTargets_CountEmptyVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	stderr := executeForTest(t, []string{"engram", "count", "--group-by", "type", "--vault", vault})
	g.Expect(stderr).To(BeEmpty())
}

// unexported constants.
const (
	countVault = "/vault"
)

// mapVaultFS is a map-backed vault used by count tests: it satisfies
// vaultgraph.VaultFS (ListMD + ReadFile) so ScanVault runs over the SAME files
// that ListMD/ReadFile expose. This lets a single fixture drive both the
// frontmatter group-by path and the wikilink backlinks path — the agreement
// the acceptance test asserts.
type mapVaultFS struct {
	files map[string]string
}

// ListMD returns the .md filenames at the root of dir.
func (m *mapVaultFS) ListMD(dir string) ([]string, error) {
	names := make([]string, 0, len(m.files))

	for path := range m.files {
		if filepath.Dir(path) == dir && strings.HasSuffix(path, ".md") {
			names = append(names, filepath.Base(path))
		}
	}

	sort.Strings(names)

	return names, nil
}

// ReadFile returns the fixture bytes at path, or a not-found error.
func (m *mapVaultFS) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, &testNotFoundError{path: path}
	}

	return []byte(data), nil
}

// deps wires CountDeps over the fixture so both the frontmatter and the graph
// paths read the same notes.
func (m *mapVaultFS) deps() cli.CountDeps {
	return cli.CountDeps{
		ListMD:   m.ListMD,
		ReadFile: m.ReadFile,
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(m, vault)
		},
	}
}

// distinctStrings returns the deduped members of values, order unspecified.
func distinctStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))

	for _, value := range values {
		if _, dup := seen[value]; dup {
			continue
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	return out
}

// newCountFS builds a map-backed fixture FS from name→content pairs, keying each
// file under the fixture vault root.
func newCountFS(notes map[string]string) *mapVaultFS {
	files := make(map[string]string, len(notes))
	for name, content := range notes {
		files[filepath.Join(countVault, name)] = content
	}

	return &mapVaultFS{files: files}
}

// parseGroupByCounts extracts the "value\tcount" lines of a group-by report into
// a map, ignoring the "(attr absent): N" and "total: N" trailer lines.
func parseGroupByCounts(out string) map[string]int {
	counts := map[string]int{}

	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		value, countText, found := strings.Cut(line, "\t")
		if !found {
			continue
		}

		count, convErr := strconv.Atoi(countText)
		if convErr != nil {
			continue
		}

		counts[value] = count
	}

	return counts
}

// parseInDegree reads the "in-degree: N" line of a backlinks report. Returns -1
// when the line is absent.
func parseInDegree(out string) int {
	for line := range strings.SplitSeq(out, "\n") {
		rest, found := strings.CutPrefix(line, "in-degree: ")
		if !found {
			continue
		}

		degree, convErr := strconv.Atoi(rest)
		if convErr != nil {
			return -1
		}

		return degree
	}

	return -1
}

// writeVaultFile writes a note into a real vault dir for the os-deps test.
func writeVaultFile(t *testing.T, vault, name, content string) {
	t.Helper()

	path := filepath.Join(vault, name)

	err := os.WriteFile(path, []byte(content), 0o600)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
}
