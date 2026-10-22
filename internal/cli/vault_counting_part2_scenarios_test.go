package cli_test

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792a8e1a_Count_BacklinksListsEachLinkerInAscendingOrder verifies that three notes linking to
// foo.alpha give "in-degree: 3" and the three basenames in ascending order. The vault lists the linkers out
// of order and the report is rebuilt many times, so an order left to a map would not pass by luck.
func Test792a8e1a_Count_BacklinksListsEachLinkerInAscendingOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	for run := range vcn2RepeatedRuns {
		vault := vcn2NewVault(map[string]string{
			"foo.alpha.md": "the note everything links to\n",
			"alpha.md":     "links to [[foo.alpha]]\n",
			"charlie.md":   "links to [[foo.alpha]]\n",
			"bravo.md":     "links to [[foo.alpha]]\n",
		})
		vault.listing = []string{"foo.alpha.md", "alpha.md", "charlie.md", "bravo.md"}

		out, err := vcn2Backlinks(vault, "foo.alpha")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(out).To(Equal("in-degree: 3\nalpha\nbravo\ncharlie\n"), "run %d", run)
	}
}

// Test792a8e1b_Count_NoteWithNoIncomingLinksHasZeroInDegree verifies that a note nothing links to prints
// "in-degree: 0" and no linker line.
func Test792a8e1b_Count_NoteWithNoIncomingLinksHasZeroInDegree(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"unknown.note.md": "an unlinked note\n",
		"first.md":        "links to [[second]] and [[unknown]]\n",
		"second.md":       "links to nothing\n",
	})

	out, err := vcn2Backlinks(vault, "unknown.note")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("in-degree: 0\n"))
}

// Test792a8e1c_Count_BasenameThatIsNotANoteHasZeroInDegree verifies that a basename that appears nowhere
// in the vault prints "in-degree: 0" and no linker line.
func Test792a8e1c_Count_BasenameThatIsNotANoteHasZeroInDegree(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"first.md":  "links to [[second]]\n",
		"second.md": "links to nothing\n",
	})

	out, err := vcn2Backlinks(vault, "nosuchfile")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("in-degree: 0\n"))
}

// Test792a8e1d_Count_RepeatedLinkCountsOnce verifies that a note holding [[y]] twice is one linker of y.
func Test792a8e1d_Count_RepeatedLinkCountsOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"x.md": "first [[y]] and again [[y]]\n",
		"y.md": "the linked note\n",
	})

	out, err := vcn2Backlinks(vault, "y")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("in-degree: 1\nx\n"))
}

// Test792a8e1e_Count_NoteLinkToItselfDoesNotCount verifies that a note whose only link is [[x]] has an
// in-degree of 0, and that a link to x from another note still counts (control).
func Test792a8e1e_Count_NoteLinkToItselfDoesNotCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"x.md": "see [[x]]\n",
	})

	out, err := vcn2Backlinks(vault, "x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("in-degree: 0\n"))

	vault.files[filepath.Join(countVault, "other.md")] = "see [[x]]\n"

	controlOut, controlErr := vcn2Backlinks(vault, "x")
	g.Expect(controlErr).NotTo(HaveOccurred())

	if controlErr != nil {
		return
	}

	g.Expect(controlOut).To(Equal("in-degree: 1\nother\n"), "control: the same note counts when another note links it")
}

// Test792a8e1f_Count_LinkInsideFencedCodeBlockDoesNotCount verifies that [[z]] inside a fenced code block
// is not a link to z, while the same text outside a fence is (control).
func Test792a8e1f_Count_LinkInsideFencedCodeBlockDoesNotCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"x.md": "before\n```\n[[z]]\n```\nafter\n",
		"z.md": "the note the fence names\n",
	})

	out, err := vcn2Backlinks(vault, "z")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("in-degree: 0\n"))

	vault.files[filepath.Join(countVault, "x.md")] = "before\n[[z]]\nafter\n"

	controlOut, controlErr := vcn2Backlinks(vault, "z")
	g.Expect(controlErr).NotTo(HaveOccurred())

	if controlErr != nil {
		return
	}

	g.Expect(controlOut).To(Equal("in-degree: 1\nx\n"), "control: the same link outside a fence counts")
}

// Test792a8e1g_Count_LinkEndingInMdCountsAsTheBareLink verifies that [[y.md]] is a link to y.
func Test792a8e1g_Count_LinkEndingInMdCountsAsTheBareLink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"x.md": "links to [[y.md]]\n",
		"y.md": "the linked note\n",
	})

	out, err := vcn2Backlinks(vault, "y")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("in-degree: 1\nx\n"))
}

// Test792a8e1h_Count_LinkToMissingNoteCountsForNoNote verifies that [[nowhere]] with no note named
// nowhere gives "in-degree: 0" for nowhere.
func Test792a8e1h_Count_LinkToMissingNoteCountsForNoNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"x.md": "links to [[nowhere]]\n",
	})

	out, err := vcn2Backlinks(vault, "nowhere")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("in-degree: 0\n"))
}

// Test792a8e1i_Count_OnlyNotesAtTheVaultRootLink verifies that a note in a subdirectory linking to x does
// not count toward x at the vault root. The vault also holds a directory whose name ends in .md, which is
// not a note.
func Test792a8e1i_Count_OnlyNotesAtTheVaultRootLink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"x.md":            "the note at the vault root\n",
		"archive/deep.md": "links to [[x]]\n",
	})
	vault.subdirs = []string{"archive", "drafts.md"}

	out, err := vcn2Backlinks(vault, "x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("in-degree: 0\n"))
	g.Expect(vault.reads).To(Equal([]string{filepath.Join(countVault, "x.md")}), "only the root note is read")
}

// Test792a8e1j_Count_UnreadableNoteFailsTheReport verifies that a root note the user cannot read fails the
// report with an error naming the file, and that nothing is printed.
func Test792a8e1j_Count_UnreadableNoteFailsTheReport(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"x.md":      "the note counted\n",
		"locked.md": "links to [[x]]\n",
	})
	vault.unreadable[filepath.Join(countVault, "locked.md")] = struct{}{}

	out, err := vcn2Backlinks(vault, "x")
	g.Expect(err).To(MatchError(ContainSubstring("locked.md")))
	g.Expect(err).To(MatchError(fs.ErrPermission))
	g.Expect(out).To(BeEmpty(), "nothing is printed to stdout")
}

// Test792a8f1a_Count_BothFlagsAreRefused verifies that --group-by with --backlinks-of fails with the
// mutual-exclusion message and prints nothing.
func Test792a8f1a_Count_BothFlagsAreRefused(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{"note.name.md": "---\ntype: fact\n---\nbody\n"})

	out, err := vcn2Count(vault, cli.CountArgs{Vault: countVault, GroupBy: "type", BacklinksOf: "note.name"})
	g.Expect(err).To(MatchError("count: --group-by and --backlinks-of are mutually exclusive"))
	g.Expect(out).To(BeEmpty(), "nothing is printed to stdout")
}

// Test792a8f1b_Count_NeitherFlagIsRefused verifies that no --group-by and no --backlinks-of fails with the
// specify-a-mode message and prints nothing.
func Test792a8f1b_Count_NeitherFlagIsRefused(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{"note.md": "---\ntype: fact\n---\nbody\n"})

	out, err := vcn2Count(vault, cli.CountArgs{Vault: countVault})
	g.Expect(err).To(MatchError("count: specify --group-by <attr> or --backlinks-of <basename>"))
	g.Expect(out).To(BeEmpty(), "nothing is printed to stdout")
}

// Test792a8f1c_Count_FilterAloneIsNotAMode verifies that --filter type=fact with neither mode flag fails
// with the specify-a-mode message and prints nothing.
func Test792a8f1c_Count_FilterAloneIsNotAMode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{"note.md": "---\ntype: fact\n---\nbody\n"})

	out, err := vcn2Count(vault, cli.CountArgs{Vault: countVault, Filter: []string{"type=fact"}})
	g.Expect(err).To(MatchError("count: specify --group-by <attr> or --backlinks-of <basename>"))
	g.Expect(out).To(BeEmpty(), "nothing is printed to stdout")
}

// Test792a8g1a_Count_ModesAgreeWhenEveryLinkerIsAMember verifies that two notes carrying foo: [alpha] and
// linking to foo.alpha give a --group-by count of 2 for alpha and an in-degree of 2.
func Test792a8g1a_Count_ModesAgreeWhenEveryLinkerIsAMember(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"n1.md":        "---\nfoo: [alpha]\n---\nlinks to [[foo.alpha]]\n",
		"n2.md":        "---\nfoo: [alpha]\n---\nlinks to [[foo.alpha]]\n",
		"foo.alpha.md": "---\ntype: fact\n---\nthe alpha note\n",
	})

	groupOut, groupErr := vcn2Count(vault, cli.CountArgs{Vault: countVault, GroupBy: "foo"})
	g.Expect(groupErr).NotTo(HaveOccurred())

	if groupErr != nil {
		return
	}

	g.Expect(groupOut).To(Equal("alpha\t2\n(foo absent): 1\ntotal: 3\n"))

	linkOut, linkErr := vcn2Backlinks(vault, "foo.alpha")
	g.Expect(linkErr).NotTo(HaveOccurred())

	if linkErr != nil {
		return
	}

	g.Expect(linkOut).To(Equal("in-degree: 2\nn1\nn2\n"))
}

// Test792a8g1b_Count_InDegreeExceedsCountByNonMemberLinkers verifies that a member and a non-member linker
// give a --group-by count of 1 for alpha and an in-degree of 2 listing both linkers in ascending order. The
// vault lists the linkers out of order and the report is rebuilt many times, so an order left to a map
// would not pass by luck.
func Test792a8g1b_Count_InDegreeExceedsCountByNonMemberLinkers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	for run := range vcn2RepeatedRuns {
		vault := vcn2NewVault(map[string]string{
			"n1.md":        "---\nfoo: [alpha]\n---\nlinks to [[foo.alpha]]\n",
			"idx.md":       "---\ntype: index\n---\nlinks to [[foo.alpha]]\n",
			"foo.alpha.md": "---\ntype: fact\n---\nthe alpha note\n",
		})
		vault.listing = []string{"n1.md", "idx.md", "foo.alpha.md"}

		groupOut, groupErr := vcn2Count(vault, cli.CountArgs{Vault: countVault, GroupBy: "foo"})
		g.Expect(groupErr).NotTo(HaveOccurred())

		if groupErr != nil {
			return
		}

		g.Expect(groupOut).To(Equal("alpha\t1\n(foo absent): 2\ntotal: 3\n"), "run %d", run)

		linkOut, linkErr := vcn2Backlinks(vault, "foo.alpha")
		g.Expect(linkErr).NotTo(HaveOccurred())

		if linkErr != nil {
			return
		}

		g.Expect(linkOut).To(Equal("in-degree: 2\nidx\nn1\n"), "run %d", run)
	}
}

// Test792a8h1a_Count_CountLeavesTheVaultAsItWas verifies that --group-by type and then --backlinks-of x
// leave every file byte-identical, create no file, and reach the filesystem with no write call.
func Test792a8h1a_Count_CountLeavesTheVaultAsItWas(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := vcn2NewVault(map[string]string{
		"x.md":          "---\ntype: fact\n---\nthe note counted\n",
		"y.md":          "---\ntype: feedback\n---\nlinks to [[x]]\n",
		"x.vec.json":    "{\"sidecar\":true}\n",
		"archive/z.md":  "---\ntype: fact\n---\nbelow the root\n",
		"README.engram": "not a note\n",
	})
	vault.subdirs = []string{"archive"}

	before := maps.Clone(vault.files)

	groupOut, groupErr := vcn2Count(vault, cli.CountArgs{Vault: countVault, GroupBy: "type"})
	g.Expect(groupErr).NotTo(HaveOccurred())

	if groupErr != nil {
		return
	}

	linkOut, linkErr := vcn2Backlinks(vault, "x")
	g.Expect(linkErr).NotTo(HaveOccurred())

	if linkErr != nil {
		return
	}

	g.Expect(groupOut).To(Equal("fact\t1\nfeedback\t1\ntotal: 2\n"), "the group-by count ran")
	g.Expect(linkOut).To(Equal("in-degree: 1\ny\n"), "the backlinks count ran")
	g.Expect(vault.files).To(Equal(before), "every file is byte-identical and none was created or deleted")
	g.Expect(vault.mutations).To(BeEmpty(), "no create, modify or delete call reached the filesystem")
}

// Test792a8h1b_Count_EveryNoteIsCountedWithoutSidecars verifies that 600 notes of type fact and no sidecar
// give "fact", a tab and 600, then "total: 600", with only the notes read.
func Test792a8h1b_Count_EveryNoteIsCountedWithoutSidecars(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const noteCount = 600

	notes := make(map[string]string, noteCount)
	for index := range noteCount {
		notes[fmt.Sprintf("note-%04d.md", index)] = "---\ntype: fact\n---\nbody\n"
	}

	vault := vcn2NewVault(notes)

	out, err := vcn2Count(vault, cli.CountArgs{Vault: countVault, GroupBy: "type"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("fact\t600\ntotal: 600\n"), "all 600 notes are counted, not a truncated subset")
	g.Expect(vault.reads).To(HaveLen(noteCount), "one read per note")

	for _, path := range vault.reads {
		g.Expect(path).To(HaveSuffix(".md"), "only notes are read, never a sidecar")
	}
}

// unexported constants.
const (
	// vcn2RepeatedRuns is how many times the ordering scenario rebuilds and reruns the report.
	vcn2RepeatedRuns = 30
)

// unexported variables.
var (
	errVcn2IsDirectory = errors.New("is a directory")
)

// vcn2Vault is an in-memory vault filesystem for the vault-counting scenarios. It lists the root notes in
// the order given by listing (sorted when empty) followed by the subdirectories, can refuse to read a
// file, applies every write, remove and rename to its files, and records each read and each call that
// could create, modify or delete a file.
type vcn2Vault struct {
	files      map[string]string
	listing    []string
	subdirs    []string
	unreadable map[string]struct{}
	reads      []string
	mutations  []string
}

// edgeFS adapts the vault to the cli.EdgeFS the production count wiring consumes.
func (v *vcn2Vault) edgeFS() cli.EdgeFS {
	return fakeEdgeFS{
		readFile:        v.readFile,
		readDir:         v.readDir,
		writeFile:       v.write("WriteFile"),
		writeFileAtomic: v.write("WriteFileAtomic"),
		writeFileExcl:   v.write("WriteFileExcl"),
		mkdirAll: func(path string, _ fs.FileMode) error {
			v.mutations = append(v.mutations, "MkdirAll "+path)

			return nil
		},
		mkdirTemp: func(dir, _ string) (string, error) {
			v.mutations = append(v.mutations, "MkdirTemp "+dir)

			return dir, nil
		},
		remove:    v.removePath("Remove"),
		removeAll: v.removePath("RemoveAll"),
		rename: func(oldPath, newPath string) error {
			v.mutations = append(v.mutations, "Rename "+oldPath)
			v.files[newPath] = v.files[oldPath]
			delete(v.files, oldPath)

			return nil
		},
		symlink: func(_, link string) error {
			v.mutations = append(v.mutations, "Symlink "+link)

			return nil
		},
	}
}

// readDir lists the vault root, or the files directly inside a subdirectory, in name order.
func (v *vcn2Vault) readDir(dir string) ([]fs.DirEntry, error) {
	entries := make([]fs.DirEntry, 0, len(v.files)+len(v.subdirs))

	if dir == countVault {
		for _, name := range v.subdirs {
			entries = append(entries, fakeDirEntry{name: name, dir: true})
		}
	}

	names := make([]string, 0, len(v.files))

	for path := range v.files {
		if filepath.Dir(path) == dir {
			names = append(names, filepath.Base(path))
		}
	}

	sort.Strings(names)

	if dir == countVault && len(v.listing) > 0 {
		names = v.listing
	}

	for _, name := range names {
		entries = append(entries, fakeDirEntry{name: name})
	}

	return entries, nil
}

// readFile serves a note, refusing the unreadable ones and any directory.
func (v *vcn2Vault) readFile(path string) ([]byte, error) {
	v.reads = append(v.reads, path)

	if _, locked := v.unreadable[path]; locked {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}
	}

	for _, name := range v.subdirs {
		if path == filepath.Join(countVault, name) {
			return nil, &fs.PathError{Op: "read", Path: path, Err: errVcn2IsDirectory}
		}
	}

	content, ok := v.files[path]
	if !ok {
		return nil, &testNotFoundError{path: path}
	}

	return []byte(content), nil
}

// removePath returns a remove func that deletes the path and records the call under op.
func (v *vcn2Vault) removePath(op string) func(string) error {
	return func(path string) error {
		v.mutations = append(v.mutations, op+" "+path)
		delete(v.files, path)

		return nil
	}
}

// write returns a write func that stores the data and records the call under op.
func (v *vcn2Vault) write(op string) func(string, []byte, fs.FileMode) error {
	return func(path string, data []byte, _ fs.FileMode) error {
		v.mutations = append(v.mutations, op+" "+path)
		v.files[path] = string(data)

		return nil
	}
}

// vcn2Backlinks runs engram count --backlinks-of target over the vault and returns stdout and the error.
func vcn2Backlinks(vault *vcn2Vault, target string) (string, error) {
	return vcn2Count(vault, cli.CountArgs{Vault: countVault, BacklinksOf: target})
}

// vcn2Count runs the count command over the vault through the production composition of its
// dependencies and returns what it printed to stdout and its error.
func vcn2Count(vault *vcn2Vault, args cli.CountArgs) (string, error) {
	var out strings.Builder

	err := cli.RunCount(args, cli.ExportNewCountDeps(vault.edgeFS()), &out)

	return out.String(), err
}

// vcn2NewVault builds a vault serving the named files under the fixture vault root.
func vcn2NewVault(notes map[string]string) *vcn2Vault {
	files := make(map[string]string, len(notes))
	for name, content := range notes {
		files[filepath.Join(countVault, name)] = content
	}

	return &vcn2Vault{files: files, unreadable: map[string]struct{}{}}
}
