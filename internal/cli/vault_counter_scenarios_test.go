package cli_test

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// Test792b25c1a3_VaultCounter_BasenameAloneSelectsIncomingLinkReport verifies that a basename without
// an attribute runs the incoming-link report: the vault is scanned and the attribute path is never taken.
func Test792b25c1a3_VaultCounter_BasenameAloneSelectsIncomingLinkReport(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"A.md": "links to [[C]]\n",
		"C.md": "no links here\n",
	})
	deps := fixture.deps()

	var listCalls, scanCalls int

	listMD := deps.ListMD
	deps.ListMD = func(vault string) ([]string, error) {
		listCalls++

		return listMD(vault)
	}

	scan := deps.Scan
	deps.Scan = func(vault string) ([]vaultgraph.Note, error) {
		scanCalls++

		return scan(vault)
	}

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, BacklinksOf: "C"}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("in-degree: 1\nA\n"))
	g.Expect(scanCalls).To(Equal(1), "the vault is scanned once for the incoming-link report")
	g.Expect(listCalls).To(BeZero(), "the attribute count never lists the vault in this mode")
}

// Test792b25c1a4_VaultCounter_AttributeAloneSelectsAttributeCount verifies that an attribute without a
// basename runs the attribute count: the notes are listed and read, and the link scan is never run.
func Test792b25c1a4_VaultCounter_AttributeAloneSelectsAttributeCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"a.md": "---\ntype: feedback\n---\nlinks to [[b]]\n",
		"b.md": "---\ntype: fact\n---\nbody\n",
	})
	deps := fixture.deps()

	var scanCalls int

	scan := deps.Scan
	deps.Scan = func(vault string) ([]vaultgraph.Note, error) {
		scanCalls++

		return scan(vault)
	}

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "type"}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("fact\t1\nfeedback\t1\ntotal: 2\n"))
	g.Expect(scanCalls).To(BeZero(), "the incoming-link scan is never run in this mode")
}

// Test792b25c2a4_VaultCounter_EmptyValueCountsOnlyTowardTotal verifies that a key with no value, an
// empty list and a scalar 3 give: 3 counts 1, nothing absent, total 3.
func Test792b25c2a4_VaultCounter_EmptyValueCountsOnlyTowardTotal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"nil.md":    "---\nfoo:\n---\nbody\n",
		"empty.md":  "---\nfoo: []\n---\nbody\n",
		"scalar.md": "---\nfoo: 3\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "foo"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("3\t1\ntotal: 3\n"),
		"3 counts 1, no absent line, and the two empty values count toward the total only")
}

// Test792b25c3a4_VaultCounter_NoteWithoutFilterAttributeDoesNotMatch verifies that a note with no outcome
// key is not counted under outcome=pass, though the same note is counted when no filter is given.
func Test792b25c3a4_VaultCounter_NoteWithoutFilterAttributeDoesNotMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"no-outcome.md": "---\ntype: fact\n---\nbody\n",
	})

	var filtered strings.Builder

	filterErr := cli.RunCount(
		cli.CountArgs{Vault: countVault, GroupBy: "type", Filter: []string{"outcome=pass"}},
		fixture.deps(), &filtered)
	g.Expect(filterErr).NotTo(HaveOccurred())

	if filterErr != nil {
		return
	}

	g.Expect(filtered.String()).To(BeEmpty(), "the note without the outcome key is not counted")

	var unfiltered strings.Builder

	plainErr := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "type"}, fixture.deps(), &unfiltered)
	g.Expect(plainErr).NotTo(HaveOccurred())

	if plainErr != nil {
		return
	}

	g.Expect(unfiltered.String()).To(Equal("fact\t1\ntotal: 1\n"),
		"control: without the filter the same note is counted")
}

// Test792b25c4a2_VaultCounter_UnreadableNoteIsSkipped verifies that a listed note whose file cannot be read
// is read once, counted nowhere and does not produce an error.
func Test792b25c4a2_VaultCounter_UnreadableNoteIsSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var readPaths []string

	deps := cli.CountDeps{
		ListMD: func(string) ([]string, error) { return []string{"ghost.md"}, nil },
		ReadFile: func(path string) ([]byte, error) {
			readPaths = append(readPaths, path)

			return nil, &testNotFoundError{path: path}
		},
		Scan: func(string) ([]vaultgraph.Note, error) { return nil, nil },
	}

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "type"}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(readPaths).To(Equal([]string{filepath.Join(countVault, "ghost.md")}),
		"the unreadable note was attempted exactly once")
	g.Expect(out.String()).To(BeEmpty(), "nothing is counted: no value, no absent line, no total")
}

// Test792b25c5a1_VaultCounter_ValuesOrderedByCountThenValue verifies the line order for counts of 349
// feedback, 192 fact, 8 qa-answer and 2 each for b and a.
func Test792b25c5a1_VaultCounter_ValuesOrderedByCountThenValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		feedbackNotes = 349
		factNotes     = 192
		qaAnswerNotes = 8
		tiedNotes     = 2
	)

	notes := map[string]string{}
	vcnAddTypedNotes(notes, "b", tiedNotes)
	vcnAddTypedNotes(notes, "qa-answer", qaAnswerNotes)
	vcnAddTypedNotes(notes, "fact", factNotes)
	vcnAddTypedNotes(notes, "a", tiedNotes)
	vcnAddTypedNotes(notes, "feedback", feedbackNotes)

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "type"}, newCountFS(notes).deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	const totalNotes = feedbackNotes + factNotes + qaAnswerNotes + 2*tiedNotes

	want := fmt.Sprintf("feedback\t%d\nfact\t%d\nqa-answer\t%d\na\t%d\nb\t%d\ntotal: %d\n",
		feedbackNotes, factNotes, qaAnswerNotes, tiedNotes, tiedNotes, totalNotes)
	g.Expect(out.String()).To(Equal(want), "lines are feedback, fact, qa-answer, a, b")
}

// Test792b25c5a2_VaultCounter_AbsentLinePrecedesTotal verifies the report layout for one note with vocab
// [x] and one without vocab: the value line, then the absent line, then the total.
func Test792b25c5a2_VaultCounter_AbsentLinePrecedesTotal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"with.md":    "---\nvocab: [x]\n---\nbody\n",
		"without.md": "---\ntype: fact\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "vocab"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("x\t1\n(vocab absent): 1\ntotal: 2\n"))
}

// Test792b25c5a3_VaultCounter_EveryNoteAbsentPrintsOnlyAbsentAndTotal verifies that when no note has the
// attribute the output is only the absent line and the total.
func Test792b25c5a3_VaultCounter_EveryNoteAbsentPrintsOnlyAbsentAndTotal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"one.md":   "---\ntype: fact\n---\nbody\n",
		"two.md":   "---\ntype: feedback\n---\nbody\n",
		"three.md": "---\ntype: fact\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "vocab"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("(vocab absent): 3\ntotal: 3\n"))
}

// Test792b25c5a5_VaultCounter_TotalCountsNotesNotValues verifies that one note with three values reports
// each value once and a total of 1.
func Test792b25c5a5_VaultCounter_TotalCountsNotesNotValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"only.md": "---\nvocab: [x, y, z]\n---\nbody\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, GroupBy: "vocab"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("x\t1\ny\t1\nz\t1\ntotal: 1\n"),
		"the three values count 1 each while the total counts the one note")
}

// Test792b25c6a2_VaultCounter_NoteNothingLinksToHasZeroInDegree verifies that an existing note that no
// other note links to reports an in-degree of 0 and no linker line.
func Test792b25c6a2_VaultCounter_NoteNothingLinksToHasZeroInDegree(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"unknown.note.md": "an unlinked note\n",
		"A.md":            "links to [[B]]\n",
		"B.md":            "no links here\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, BacklinksOf: "unknown.note"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("in-degree: 0\n"))
}

// Test792b25c6a3_VaultCounter_BasenameThatIsNotANoteHasZeroInDegree verifies that a basename that is
// not a scanned note reports an in-degree of 0 and no linker line.
func Test792b25c6a3_VaultCounter_BasenameThatIsNotANoteHasZeroInDegree(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newCountFS(map[string]string{
		"A.md": "links to [[B]]\n",
		"B.md": "no links here\n",
	})

	var out strings.Builder

	err := cli.RunCount(cli.CountArgs{Vault: countVault, BacklinksOf: "nosuchfile"}, fixture.deps(), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("in-degree: 0\n"))
}

// Test792b25c8a1_VaultCounter_CountWritesNothing verifies, through the production count wiring over a
// recording filesystem, that counting in each mode creates, changes or deletes no file.
func Test792b25c8a1_VaultCounter_CountWritesNothing(t *testing.T) {
	t.Parallel()

	modes := map[string]cli.CountArgs{
		"attribute count":      {Vault: countVault, GroupBy: "type"},
		"incoming-link report": {Vault: countVault, BacklinksOf: "b"},
	}

	for name, args := range modes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			recorder := vcnNewRecordingFS(map[string]string{
				"a.md": "---\ntype: fact\n---\nlinks to [[b]]\n",
				"b.md": "---\ntype: feedback\n---\nbody\n",
			})

			var out strings.Builder

			err := cli.RunCount(args, cli.ExportNewCountDeps(recorder.edgeFS()), &out)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(out.String()).NotTo(BeEmpty(), "the mode ran and reported")
			g.Expect(recorder.reads).NotTo(BeEmpty(), "the notes were read")
			g.Expect(recorder.mutations).To(BeEmpty(), "no create, modify or delete call reached the filesystem")
		})
	}
}

// Test792b25c8a2_VaultCounter_NotesCountedWithoutSidecars verifies that 600 notes with no sidecar are all
// counted, and that no sidecar is looked for: only the notes themselves are read.
func Test792b25c8a2_VaultCounter_NotesCountedWithoutSidecars(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const noteCount = 600

	files := make(map[string]string, noteCount)
	for index := range noteCount {
		files[fmt.Sprintf("note-%04d.md", index)] = "---\ntype: fact\n---\nbody\n"
	}

	recorder := vcnNewRecordingFS(files)

	var out strings.Builder

	err := cli.RunCount(
		cli.CountArgs{Vault: countVault, GroupBy: "type"}, cli.ExportNewCountDeps(recorder.edgeFS()), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("fact\t600\ntotal: 600\n"), "all 600 notes are counted")
	g.Expect(recorder.reads).To(HaveLen(noteCount), "one read per note and nothing else")

	for _, path := range recorder.reads {
		g.Expect(path).To(HaveSuffix(".md"), "only notes are read, never a sidecar")
	}
}

// vcnRecordingFS is an in-memory vault filesystem for the vault-counter scenarios: it serves the listed
// notes, records every read, and records every call that could create, modify or delete a file.
type vcnRecordingFS struct {
	files     map[string]string
	reads     []string
	mutations []string
}

// edgeFS adapts the recorder to the cli.EdgeFS the production count wiring consumes.
func (r *vcnRecordingFS) edgeFS() cli.EdgeFS {
	mutate := func(op string) func(string) error {
		return func(path string) error {
			r.mutations = append(r.mutations, op+" "+path)

			return nil
		}
	}

	write := func(op string) func(string, []byte, fs.FileMode) error {
		return func(path string, _ []byte, _ fs.FileMode) error {
			r.mutations = append(r.mutations, op+" "+path)

			return nil
		}
	}

	return fakeEdgeFS{
		readFile: func(path string) ([]byte, error) {
			r.reads = append(r.reads, path)

			content, ok := r.files[path]
			if !ok {
				return nil, &testNotFoundError{path: path}
			}

			return []byte(content), nil
		},
		readDir:         r.readDir,
		writeFile:       write("WriteFile"),
		writeFileAtomic: write("WriteFileAtomic"),
		writeFileExcl:   write("WriteFileExcl"),
		mkdirAll: func(path string, _ fs.FileMode) error {
			r.mutations = append(r.mutations, "MkdirAll "+path)

			return nil
		},
		mkdirTemp: func(dir, _ string) (string, error) {
			r.mutations = append(r.mutations, "MkdirTemp "+dir)

			return dir, nil
		},
		remove:    mutate("Remove"),
		removeAll: mutate("RemoveAll"),
		rename: func(oldPath, _ string) error {
			r.mutations = append(r.mutations, "Rename "+oldPath)

			return nil
		},
		symlink: func(_, link string) error {
			r.mutations = append(r.mutations, "Symlink "+link)

			return nil
		},
	}
}

// readDir lists the vault root's files in name order.
func (r *vcnRecordingFS) readDir(dir string) ([]fs.DirEntry, error) {
	names := make([]string, 0, len(r.files))

	for path := range r.files {
		if filepath.Dir(path) == dir {
			names = append(names, filepath.Base(path))
		}
	}

	sort.Strings(names)

	entries := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, fakeDirEntry{name: name})
	}

	return entries, nil
}

// vcnAddTypedNotes adds count well-formed notes of the given type to notes, named after the type.
func vcnAddTypedNotes(notes map[string]string, noteType string, count int) {
	for index := range count {
		notes[fmt.Sprintf("%s-%04d.md", noteType, index)] = "---\ntype: " + noteType + "\n---\nbody\n"
	}
}

// vcnNewRecordingFS builds a recorder serving the named notes under the fixture vault root.
func vcnNewRecordingFS(notes map[string]string) *vcnRecordingFS {
	files := make(map[string]string, len(notes))
	for name, content := range notes {
		files[filepath.Join(countVault, name)] = content
	}

	return &vcnRecordingFS{files: files}
}
