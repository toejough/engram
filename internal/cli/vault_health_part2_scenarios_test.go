package cli_test

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a9f1a_Check_LeavesTheVaultAsItWas verifies Scenario 792a9f1a: a check over a vault with notes, some with
// sidecars and some violating invariants, reports its findings and leaves every file byte-identical, creating none.
func Test792a9f1a_Check_LeavesTheVaultAsItWas(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	// Violations: a bare-ID link that should resolve (G0), a link to no note (G3), a fact with no situation (M5),
	// an old-schema-or-malformed sidecar (S1 FAIL) and a note with no sidecar (S1 WARN).
	fake.setNote("1.linker", vh2NoteBody("see [[2]] and [[ghost]]"))
	fake.setNote("2.target", vh2NoteBody("a target"))
	fake.setSidecar("2.target", vh2Sidecar(t, vh2ModelID, vh2NoteBody("a target")))
	fake.setNote("3.nosituation", "---\ntype: fact\n---\nno situation here\n")
	fake.setSidecar("3.nosituation", []byte("{not json"))

	before := fake.snapshot()

	outcome := vh2Run(t, fake, "check")

	g.Expect(outcome.stdout).To(ContainSubstring("FAIL  G0 graph-resolution"))
	g.Expect(outcome.stdout).To(ContainSubstring("WARN  G3 dangling"))
	g.Expect(outcome.stdout).To(ContainSubstring("FAIL  M5 situation-presence"))
	g.Expect(outcome.stdout).To(ContainSubstring("WARN  S1 sidecar-schema"))
	g.Expect(outcome.stdout).To(ContainSubstring("FAIL  S1 sidecar-schema"))
	g.Expect(outcome.exits).To(Equal([]int{1}), "the FAIL findings fail the command")
	g.Expect(fake.snapshot()).To(Equal(before), "every file and directory is as it was, none created")
	g.Expect(fake.touched).To(BeEmpty(), "no write, create, remove or rename was attempted")
}

// Test792a9g1a_EmbedStatus_NotesWithoutSidecarsAreCountedAsWithout verifies Scenario 792a9g1a: two notes and no
// sidecar print total 2, with-embeddings 0 and without 2.
func Test792a9g1a_EmbedStatus_NotesWithoutSidecarsAreCountedAsWithout(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.foo", vh2NoteBody("one"))
	fake.setNote("2.bar", vh2NoteBody("two"))

	outcome := vh2Run(t, fake, "embed", "status")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.stdout).To(ContainSubstring("total:           2"))
	g.Expect(outcome.stdout).To(ContainSubstring("with-embeddings: 0"))
	g.Expect(outcome.stdout).To(ContainSubstring("without:         2"))
}

// Test792a9g1b_EmbedStatus_EveryStateIsCounted verifies Scenario 792a9g1b: a vault of one current, one stale, one
// incompatible and one broken note prints total 4 and with-embeddings, stale, incompatible and broken at 1 each.
func Test792a9g1b_EmbedStatus_EveryStateIsCounted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.current", vh2NoteBody("current"))
	fake.setSidecar("1.current", vh2Sidecar(t, vh2ModelID, vh2NoteBody("current")))
	fake.setNote("2.stale", vh2NoteBody("changed after embedding"))
	fake.setSidecar("2.stale", vh2Sidecar(t, vh2ModelID, vh2NoteBody("as embedded")))
	fake.setNote("3.other", vh2NoteBody("other model"))
	fake.setSidecar("3.other", vh2Sidecar(t, vh2OtherModelID, vh2NoteBody("other model")))
	fake.setNote("4.broken", vh2NoteBody("broken"))
	fake.setSidecar("4.broken", []byte("{not json"))

	outcome := vh2Run(t, fake, "embed", "status")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(vh2Lines(outcome.stdout)).To(Equal([]string{
		"total:           4",
		"with-embeddings: 1",
		"without:         0",
		"stale:           1",
		"incompatible:    1",
		"broken:          1",
	}))
}

// Test792a9g1c_EmbedStatus_AnUnreadableNoteStopsTheCensus verifies Scenario 792a9g1c: with a note the user cannot
// read, embed status fails with "embed status: scan:" naming the file and prints no census.
func Test792a9g1c_EmbedStatus_AnUnreadableNoteStopsTheCensus(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.readable", vh2NoteBody("readable"))
	fake.setNote("2.locked", vh2NoteBody("locked"))
	fake.unreadable[vh2Path("2.locked.md")] = true

	outcome := vh2Run(t, fake, "embed", "status")

	g.Expect(outcome.stderr).To(HavePrefix("embed status: scan:"))
	g.Expect(outcome.stderr).To(ContainSubstring(vh2Path("2.locked.md")))
	g.Expect(outcome.stdout).To(BeEmpty(), "no census line is printed")
	g.Expect(outcome.exits).To(Equal([]int{1}))
}

// Test792a9h1a_EmbedApply_TheDefaultEmbedsOnlyNotesWithoutASidecar verifies Scenario 792a9h1a: with one note without
// a sidecar and one with a current sidecar, embed apply embeds only the first.
func Test792a9h1a_EmbedApply_TheDefaultEmbedsOnlyNotesWithoutASidecar(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.missing", vh2NoteBody("missing"))
	fake.setNote("2.current", vh2NoteBody("current"))
	fake.setSidecar("2.current", vh2Sidecar(t, vh2ModelID, vh2NoteBody("current")))

	outcome := vh2Run(t, fake, "embed", "apply")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(vh2Lines(outcome.stdout)).To(Equal([]string{"embedded  1.missing.md (missing)"}))
	g.Expect(fake.touched).To(Equal([]string{vh2Path("1.missing.vec.json")}), "only that sidecar was written")
}

// Test792a9h1b_EmbedApply_StaleEmbedsChangedNotesAndNotAnotherModelsSidecars verifies Scenario 792a9h1b: with a note
// whose body changed after its sidecar and a note whose sidecar names another model, --stale re-embeds the changed
// note and leaves the other model's sidecar unchanged.
func Test792a9h1b_EmbedApply_StaleEmbedsChangedNotesAndNotAnotherModelsSidecars(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.changed", vh2NoteBody("the new body"))
	fake.setSidecar("1.changed", vh2Sidecar(t, vh2ModelID, vh2NoteBody("the old body")))
	fake.setNote("2.other", vh2NoteBody("other model"))
	otherSidecar := vh2Sidecar(t, vh2OtherModelID, vh2NoteBody("other model"))
	fake.setSidecar("2.other", otherSidecar)

	outcome := vh2Run(t, fake, "embed", "apply", "--stale")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(vh2Lines(outcome.stdout)).To(Equal([]string{"embedded  1.changed.md (stale)"}))
	g.Expect(fake.files[vh2Path("1.changed.vec.json")]).To(
		Equal(vh2Sidecar(t, vh2ModelID, vh2NoteBody("the new body"))), "the changed note is embedded as it now reads")
	g.Expect(fake.files[vh2Path("2.other.vec.json")]).To(Equal(otherSidecar), "the other model's sidecar is unchanged")
	g.Expect(fake.touched).To(Equal([]string{vh2Path("1.changed.vec.json")}))
}

// Test792a9h1c_EmbedApply_ForceReplacesAnotherModelsSidecar verifies Scenario 792a9h1c: --force rewrites a sidecar
// that names another model under the current model.
func Test792a9h1c_EmbedApply_ForceReplacesAnotherModelsSidecar(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.other", vh2NoteBody("other model"))
	fake.setSidecar("1.other", vh2Sidecar(t, vh2OtherModelID, vh2NoteBody("other model")))

	outcome := vh2Run(t, fake, "embed", "apply", "--force")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(vh2Lines(outcome.stdout)).To(Equal([]string{"embedded  1.other.md (incompatible)"}))

	sidecar, err := embed.UnmarshalSidecar(fake.files[vh2Path("1.other.vec.json")])
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(sidecar.EmbeddingModelID).To(Equal(vh2ModelID), "the sidecar is now under the current model")
}

// Test792a9h1d_EmbedApply_AllEmbedsEveryNote verifies Scenario 792a9h1d: --all embeds a note whose sidecar is
// already current, again.
func Test792a9h1d_EmbedApply_AllEmbedsEveryNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.current", vh2NoteBody("current"))
	fake.setSidecar("1.current", vh2Sidecar(t, vh2ModelID, vh2NoteBody("current")))

	outcome := vh2Run(t, fake, "embed", "apply", "--all")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(vh2Lines(outcome.stdout)).To(Equal([]string{"embedded  1.current.md (ok)"}))
	g.Expect(fake.touched).To(Equal([]string{vh2Path("1.current.vec.json")}), "the sidecar was written again")
}

// Test792a9h1e_EmbedApply_ModeFlagsCombine verifies Scenario 792a9h1e: --stale --force embeds a stale and an
// incompatible note and not a missing one.
func Test792a9h1e_EmbedApply_ModeFlagsCombine(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.stale", vh2NoteBody("the new body"))
	fake.setSidecar("1.stale", vh2Sidecar(t, vh2ModelID, vh2NoteBody("the old body")))
	fake.setNote("2.other", vh2NoteBody("other model"))
	fake.setSidecar("2.other", vh2Sidecar(t, vh2OtherModelID, vh2NoteBody("other model")))
	fake.setNote("3.missing", vh2NoteBody("missing"))

	outcome := vh2Run(t, fake, "embed", "apply", "--stale", "--force")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(vh2Lines(outcome.stdout)).To(Equal([]string{
		"embedded  1.stale.md (stale)",
		"embedded  2.other.md (incompatible)",
	}))
	g.Expect(fake.files).NotTo(HaveKey(vh2Path("3.missing.vec.json")), "the missing note is not embedded")
}

// Test792a9h1f_EmbedApply_ABrokenSidecarIsAlwaysReplaced verifies Scenario 792a9h1f: a note whose sidecar is
// malformed is embedded by embed apply, by --stale and by --force.
func Test792a9h1f_EmbedApply_ABrokenSidecarIsAlwaysReplaced(t *testing.T) {
	t.Parallel()

	selections := map[string][]string{
		"no mode flag": {},
		"stale":        {"--stale"},
		"force":        {"--force"},
	}

	for name, flags := range selections {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fake := newVh2FS()
			fake.setNote("1.broken", vh2NoteBody("broken"))
			fake.setSidecar("1.broken", []byte("{not json"))

			args := append([]string{"embed", "apply"}, flags...)
			outcome := vh2Run(t, fake, args...)

			g.Expect(outcome.stderr).To(BeEmpty())
			g.Expect(vh2Lines(outcome.stdout)).To(Equal([]string{"embedded  1.broken.md (broken)"}))
			g.Expect(fake.files[vh2Path("1.broken.vec.json")]).To(
				Equal(vh2Sidecar(t, vh2ModelID, vh2NoteBody("broken"))), "the malformed sidecar is replaced")
		})
	}
}

// Test792a9i1a_EmbedApply_AMissingSidecarIsWritten verifies Scenario 792a9i1a: a note 1.foo.md with no sidecar is
// reported "embedded  1.foo.md (missing)" and gets the sidecar 1.foo.vec.json.
func Test792a9i1a_EmbedApply_AMissingSidecarIsWritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.foo", vh2NoteBody("foo"))

	outcome := vh2Run(t, fake, "embed", "apply")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.stdout).To(ContainSubstring("embedded  1.foo.md (missing)"))
	g.Expect(fake.files).To(HaveKey(vh2Path("1.foo.vec.json")))

	sidecar, err := embed.UnmarshalSidecar(fake.files[vh2Path("1.foo.vec.json")])
	g.Expect(err).NotTo(HaveOccurred(), "the written sidecar is a current one")

	if err != nil {
		return
	}

	g.Expect(sidecar.EmbeddingModelID).To(Equal(vh2ModelID))
}

// Test792a9i1b_EmbedApply_ADryRunWritesNothing verifies Scenario 792a9i1b: --dry-run prints
// "would-embed 1.foo.md (missing)" and leaves no sidecar.
func Test792a9i1b_EmbedApply_ADryRunWritesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.foo", vh2NoteBody("foo"))

	before := fake.snapshot()

	outcome := vh2Run(t, fake, "embed", "apply", "--dry-run")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.stdout).To(ContainSubstring("would-embed 1.foo.md (missing)"))
	g.Expect(fake.files).NotTo(HaveKey(vh2Path("1.foo.vec.json")))
	g.Expect(fake.snapshot()).To(Equal(before))
	g.Expect(fake.touched).To(BeEmpty())
}

// Test792a9i1c_EmbedApply_OnlySidecarsAreWritten verifies Scenario 792a9i1c: after embed apply --all over notes
// needing embedding, every note file is byte-identical, none holds a vector or a model ID, and the only files
// created or written are sidecars.
func Test792a9i1c_EmbedApply_OnlySidecarsAreWritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.missing", vh2NoteBody("missing"))
	fake.setNote("2.stale", vh2NoteBody("the new body"))
	fake.setSidecar("2.stale", vh2Sidecar(t, vh2ModelID, vh2NoteBody("the old body")))
	fake.setNote("3.other", vh2NoteBody("other model"))
	fake.setSidecar("3.other", vh2Sidecar(t, vh2OtherModelID, vh2NoteBody("other model")))

	before := fake.snapshot()

	outcome := vh2Run(t, fake, "embed", "apply", "--all")
	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(vh2Lines(outcome.stdout)).To(HaveLen(3), "all three notes were embedded")

	after := fake.snapshot()

	for _, name := range []string{"1.missing", "2.stale", "3.other"} {
		notePath := vh2Path(name + ".md")
		g.Expect(after[notePath]).To(Equal(before[notePath]), "%s is byte-identical", name)

		for _, forbidden := range []string{"situation_vector", "body_vector", vh2ModelID, vh2OtherModelID} {
			g.Expect(after[notePath]).NotTo(ContainSubstring(forbidden), "%s holds no %s", name, forbidden)
		}
	}

	for path := range after {
		_, existed := before[path]
		if !existed {
			g.Expect(path).To(HaveSuffix(".vec.json"), "the only files created are sidecars")
		}
	}

	g.Expect(fake.touched).To(ConsistOf(
		vh2Path("1.missing.vec.json"), vh2Path("2.stale.vec.json"), vh2Path("3.other.vec.json")))
}

// Test792a9i1d_EmbedApply_AFailedWriteIsReportedAndTheRunGoesOn verifies Scenario 792a9i1d: with a directory where
// the first note's sidecar would be, embed apply prints a "fail      " line for the first note ending with the
// write error, embeds the second note, and exits zero.
func Test792a9i1d_EmbedApply_AFailedWriteIsReportedAndTheRunGoesOn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.first", vh2NoteBody("first"))
	fake.dirs[vh2Path("1.first.vec.json")] = true
	fake.setNote("2.second", vh2NoteBody("second"))

	outcome := vh2Run(t, fake, "embed", "apply")

	writeErr := &fs.PathError{Op: "rename", Path: vh2Path("1.first.vec.json"), Err: errVh2IsDirectory}
	lines := vh2Lines(outcome.stdout)

	g.Expect(lines).To(HaveLen(2))

	if len(lines) != 2 {
		return
	}

	g.Expect(lines[0]).To(HavePrefix("fail      1.first.md: write: "))
	g.Expect(lines[0]).To(HaveSuffix(writeErr.Error()), "the line ends with the write error")
	g.Expect(lines[1]).To(Equal("embedded  2.second.md (missing)"))
	g.Expect(fake.files).To(HaveKey(vh2Path("2.second.vec.json")), "the second note is embedded")
	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.exits).To(BeEmpty(), "the exit status is zero")
}

// Test792a9i1e_EmbedApply_NotesAreEmbeddedInOrder verifies Scenario 792a9i1e: three notes without sidecars are
// embedded in ascending basename order. The basenames 1.a and 1.a.b sort the other way as file names (1.a.b.md
// before 1.a.md) and the directory listing is rebuilt from a map on every run, so only the sort by basename gives
// this order.
func Test792a9i1e_EmbedApply_NotesAreEmbeddedInOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	for range vh2OrderRuns {
		fake := newVh2FS()
		fake.setNote("2.zeta", vh2NoteBody("zeta"))
		fake.setNote("1.a.b", vh2NoteBody("a b"))
		fake.setNote("1.a", vh2NoteBody("a"))

		outcome := vh2Run(t, fake, "embed", "apply")

		g.Expect(outcome.stderr).To(BeEmpty())
		g.Expect(vh2Lines(outcome.stdout)).To(Equal([]string{
			"embedded  1.a.md (missing)",
			"embedded  1.a.b.md (missing)",
			"embedded  2.zeta.md (missing)",
		}))
	}
}

// Test792a9i1f_EmbedApply_AnUnreadableNoteStopsTheRun verifies Scenario 792a9i1f: with a note the user cannot read,
// embed apply fails with "embed apply: scan:" and embeds nothing, not even the readable notes before it.
func Test792a9i1f_EmbedApply_AnUnreadableNoteStopsTheRun(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fake := newVh2FS()
	fake.setNote("1.readable", vh2NoteBody("readable"))
	fake.setNote("2.locked", vh2NoteBody("locked"))
	fake.unreadable[vh2Path("2.locked.md")] = true

	before := fake.snapshot()

	outcome := vh2Run(t, fake, "embed", "apply")

	g.Expect(outcome.stderr).To(HavePrefix("embed apply: scan:"))
	g.Expect(outcome.stderr).To(ContainSubstring(vh2Path("2.locked.md")))
	g.Expect(outcome.stdout).To(BeEmpty(), "no embedded line is printed")
	g.Expect(outcome.exits).To(Equal([]int{1}))
	g.Expect(fake.snapshot()).To(Equal(before), "nothing is embedded")
	g.Expect(fake.touched).To(BeEmpty())
}

// Test792a9j1a_EmbedStatus_ACRLFNoteIsNotStale verifies Scenario 792a9j1a: a note embedded with LF line endings and
// then converted to CRLF is counted with-embeddings, and stale is 0.
func Test792a9j1a_EmbedStatus_ACRLFNoteIsNotStale(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lfBody := vh2NoteBody("line one\nline two")
	crlfBody := strings.ReplaceAll(lfBody, "\n", "\r\n")

	fake := newVh2FS()
	fake.setNote("1.foo", crlfBody)
	fake.setSidecar("1.foo", vh2Sidecar(t, vh2ModelID, lfBody))

	g.Expect(crlfBody).NotTo(Equal(lfBody), "the note really has CRLF line endings")

	outcome := vh2Run(t, fake, "embed", "status")

	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(vh2Lines(outcome.stdout)).To(Equal([]string{
		"total:           1",
		"with-embeddings: 1",
		"without:         0",
		"stale:           0",
		"incompatible:    0",
		"broken:          0",
	}))
}

// Test792a9j1b_EmbedApply_ACRLFNoteGetsItsLFFormsSidecar verifies Scenario 792a9j1b: embed apply --all over a note
// with LF line endings and over the same note with CRLF line endings writes byte-identical sidecars.
func Test792a9j1b_EmbedApply_ACRLFNoteGetsItsLFFormsSidecar(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lfBody := vh2NoteBody("line one\nline two")
	crlfBody := strings.ReplaceAll(lfBody, "\n", "\r\n")

	lfVault := newVh2FS()
	lfVault.setNote("1.foo", lfBody)

	crlfVault := newVh2FS()
	crlfVault.setNote("1.foo", crlfBody)

	lfOutcome := vh2Run(t, lfVault, "embed", "apply", "--all")
	crlfOutcome := vh2Run(t, crlfVault, "embed", "apply", "--all")

	g.Expect(lfOutcome.stderr).To(BeEmpty())
	g.Expect(crlfOutcome.stderr).To(BeEmpty())

	sidecarPath := vh2Path("1.foo.vec.json")
	lfSidecar := lfVault.files[sidecarPath]
	crlfSidecar := crlfVault.files[sidecarPath]

	g.Expect(lfSidecar).NotTo(BeEmpty(), "the LF note got a sidecar")
	g.Expect(crlfVault.files[vh2Path("1.foo.md")]).To(Equal([]byte(crlfBody)), "the CRLF note stayed CRLF")
	g.Expect(crlfSidecar).To(Equal(lfSidecar), "the two sidecars are byte-identical")
}

// unexported constants.
const (
	vh2Dims         = 4
	vh2Home         = "/vh2/home"
	vh2ModelID      = "m@4"
	vh2OrderRuns    = 25
	vh2OtherModelID = "OLD@384"
	vh2Vault        = "/vh2/vault"
)

// unexported variables.
var (
	errVh2IsDirectory = errors.New("is a directory")
	errVh2Unexpected  = errors.New("vh2 fake file system: unexpected call")
)

// vh2FS is an in-memory cli.EdgeFS holding one vault. Files and directories live in maps, so a directory listing
// comes back in map order. A mutating call is recorded in touched. unreadable names files whose read is denied.
type vh2FS struct {
	files      map[string][]byte
	dirs       map[string]bool
	unreadable map[string]bool
	touched    []string
}

func (f *vh2FS) Lstat(path string) (fs.FileInfo, error) { return f.Stat(path) }

func (f *vh2FS) MkdirAll(path string, _ fs.FileMode) error {
	f.touched = append(f.touched, path)
	f.dirs[filepath.Clean(path)] = true

	return nil
}

func (f *vh2FS) MkdirTemp(string, string) (string, error) { return "", errVh2Unexpected }

func (f *vh2FS) ReadDir(path string) ([]fs.DirEntry, error) {
	dir := filepath.Clean(path)
	if !f.dirs[dir] {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}

	entries := make([]fs.DirEntry, 0, len(f.files)+len(f.dirs))

	for file := range f.files {
		if filepath.Dir(file) == dir {
			entries = append(entries, fakeDirEntry{name: filepath.Base(file)})
		}
	}

	for sub := range f.dirs {
		if sub != dir && filepath.Dir(sub) == dir {
			entries = append(entries, fakeDirEntry{name: filepath.Base(sub), dir: true})
		}
	}

	return entries, nil
}

func (f *vh2FS) ReadFile(path string) ([]byte, error) {
	clean := filepath.Clean(path)

	if f.unreadable[clean] {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}
	}

	if f.dirs[clean] {
		return nil, &fs.PathError{Op: "read", Path: path, Err: errVh2IsDirectory}
	}

	data, ok := f.files[clean]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}

	return bytes.Clone(data), nil
}

func (f *vh2FS) Readlink(string) (string, error) { return "", errVh2Unexpected }

func (f *vh2FS) Remove(path string) error {
	f.touched = append(f.touched, path)

	return errVh2Unexpected
}

func (f *vh2FS) RemoveAll(path string) error {
	f.touched = append(f.touched, path)

	return errVh2Unexpected
}

func (f *vh2FS) Rename(oldPath, _ string) error {
	f.touched = append(f.touched, oldPath)

	return errVh2Unexpected
}

func (f *vh2FS) Stat(path string) (fs.FileInfo, error) {
	clean := filepath.Clean(path)

	if f.dirs[clean] {
		return fakeFileInfo{dir: true}, nil
	}

	data, ok := f.files[clean]
	if !ok {
		return nil, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}

	return fakeFileInfo{size: int64(len(data))}, nil
}

func (f *vh2FS) Symlink(_, link string) error {
	f.touched = append(f.touched, link)

	return errVh2Unexpected
}

func (f *vh2FS) WalkDir(string, fs.WalkDirFunc) error { return errVh2Unexpected }

func (f *vh2FS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	return f.write(path, data)
}

// WriteFileAtomic fails when a directory stands at path, as the rename of the temporary file onto it would.
func (f *vh2FS) WriteFileAtomic(path string, data []byte, _ fs.FileMode) error {
	if f.dirs[filepath.Clean(path)] {
		f.touched = append(f.touched, path)

		return &fs.PathError{Op: "rename", Path: path, Err: errVh2IsDirectory}
	}

	return f.write(path, data)
}

func (f *vh2FS) WriteFileExcl(path string, data []byte, _ fs.FileMode) error {
	_, exists := f.files[filepath.Clean(path)]
	if exists {
		f.touched = append(f.touched, path)

		return &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
	}

	return f.write(path, data)
}

func (f *vh2FS) setNote(name, body string) {
	f.files[vh2Path(name+".md")] = []byte(body)
}

func (f *vh2FS) setSidecar(name string, data []byte) {
	f.files[vh2Path(name+".vec.json")] = data
}

// snapshot is every file's content and every directory, by path, to compare before and after a command.
func (f *vh2FS) snapshot() map[string]string {
	state := make(map[string]string, len(f.files)+len(f.dirs))

	for path, data := range f.files {
		state[path] = string(data)
	}

	for path := range f.dirs {
		state[path] = "<directory>"
	}

	return state
}

func (f *vh2FS) write(path string, data []byte) error {
	f.touched = append(f.touched, path)
	f.files[filepath.Clean(path)] = bytes.Clone(data)

	return nil
}

// vh2Outcome is what one run of the command printed and the exit statuses it asked for.
type vh2Outcome struct {
	stdout string
	stderr string
	exits  []int
}

func newVh2FS() *vh2FS {
	return &vh2FS{
		files:      map[string][]byte{},
		dirs:       map[string]bool{vh2Vault: true},
		unreadable: map[string]bool{},
	}
}

// vh2Lines splits command output into its lines, none for empty output.
func vh2Lines(output string) []string {
	trimmed := strings.TrimSuffix(output, "\n")
	if trimmed == "" {
		return []string{}
	}

	return strings.Split(trimmed, "\n")
}

// vh2NoteBody is a fact note whose body is text.
func vh2NoteBody(text string) string {
	return "---\ntype: fact\nsituation: when testing\n---\n" + text + "\n"
}

func vh2Path(file string) string { return filepath.Join(vh2Vault, file) }

// vh2Run runs `engram <args> --vault <vault>` through cli.Targets over the fake file system and a stub embedder.
// Nothing else of the machine is read: the environment is empty and the home is a path no file system holds.
func vh2Run(t *testing.T, fake *vh2FS, args ...string) vh2Outcome {
	t.Helper()

	var exits []int

	argv := append([]string{"engram"}, args...)
	argv = append(argv, "--vault", vh2Vault)

	stdout, stderr := executeCapturingBoth(t, argv, func(deps *cli.Deps) {
		deps.FS = fake
		deps.Embed = stubEmbedder{modelID: vh2ModelID, dims: vh2Dims}
		deps.Getenv = func(string) string { return "" }
		deps.UserHomeDir = func() (string, error) { return vh2Home, nil }
		deps.DebugLog = nil
		deps.Exit = func(code int) { exits = append(exits, code) }
	})

	return vh2Outcome{stdout: stdout, stderr: stderr, exits: exits}
}

// vh2Sidecar is the sidecar embed apply would write for body under the given model.
func vh2Sidecar(t *testing.T, modelID, body string) []byte {
	t.Helper()

	g := NewWithT(t)

	sidecar, err := embed.BuildSidecar(t.Context(), stubEmbedder{modelID: modelID, dims: vh2Dims}, []byte(body))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return nil
	}

	return embed.MarshalSidecar(sidecar)
}
