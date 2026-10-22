package cli_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
)

// Test792a8a1a_Count_ScalarAttributeCountedPerValue verifies 792a8a1a (vault-counting.md): a vault of 349
// feedback, 192 fact and 8 qa-answer notes prints each type with its count, ordered by count, then the total.
func Test792a8a1a_Count_ScalarAttributeCountedPerValue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault of 349 feedback notes, 192 fact notes and 8 qa-answer notes
	const (
		feedbackNotes = 349
		factNotes     = 192
		qaAnswerNotes = 8
	)

	notes := map[string]string{}
	vcn1AddTypedNotes(notes, "feedback", feedbackNotes)
	vcn1AddTypedNotes(notes, "fact", factNotes)
	vcn1AddTypedNotes(notes, "qa-answer", qaAnswerNotes)

	// When the user runs engram count --group-by type
	outcome := vcn1Count(t, vcn1Vault{files: notes}, vcn1GroupByFlag, vcn1TypeAttr)

	// Then stdout is "feedback", a tab and 349, then "fact", a tab and 192, then "qa-answer", a tab and 8,
	// then the line "total: 549"
	g.Expect(outcome.stdout).To(Equal("feedback\t349\nfact\t192\nqa-answer\t8\ntotal: 549\n"))
	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8a1b_Count_ListAttributeCountsEachDistinctElementOncePerNote verifies 792a8a1b (vault-counting.md):
// the one note with vocab: [x, y, z] contributes each element once.
func Test792a8a1b_Count_ListAttributeCountsEachDistinctElementOncePerNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given the only note with vocab values is one whose frontmatter has vocab: [x, y, z]
	vault := vcn1Vault{files: map[string]string{"only.md": vcn1Note("vocab: [x, y, z]")}}

	// When the user runs engram count --group-by vocab
	outcome := vcn1Count(t, vault, vcn1GroupByFlag, "vocab")

	// Then each of x, y and z is printed with the count 1, followed by "total: 1"
	g.Expect(outcome.stdout).To(Equal("x\t1\ny\t1\nz\t1\ntotal: 1\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8a1c_Count_RepeatedElementCountsOnce verifies 792a8a1c (vault-counting.md): a note listing dup
// twice counts once for dup.
func Test792a8a1c_Count_RepeatedElementCountsOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note whose frontmatter has vocab: [dup, dup, other]
	vault := vcn1Vault{files: map[string]string{"repeat.md": vcn1Note("vocab: [dup, dup, other]")}}

	// When the user runs engram count --group-by vocab
	outcome := vcn1Count(t, vault, vcn1GroupByFlag, "vocab")

	// Then dup and other are each printed with the count 1, not dup with 2, followed by "total: 1"
	g.Expect(outcome.stdout).To(Equal("dup\t1\nother\t1\ntotal: 1\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8a1d_Count_EqualCountsAreOrderedByValue verifies 792a8a1d (vault-counting.md): with two fact and
// two feedback notes, fact is printed before feedback. The files sort so that feedback is met first, and the
// command is rebuilt and run many times, so neither first-seen order nor a map's iteration order can pass.
func Test792a8a1d_Count_EqualCountsAreOrderedByValue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	for run := range vcn1TieRuns {
		// Given two notes with type: fact and two notes with type: feedback
		vault := vcn1Vault{files: map[string]string{
			"a-feedback-1.md": vcn1Note("type: feedback"),
			"b-feedback-2.md": vcn1Note("type: feedback"),
			"c-fact-1.md":     vcn1Note("type: fact"),
			"d-fact-2.md":     vcn1Note("type: fact"),
		}}

		// When the user runs engram count --group-by type
		outcome := vcn1Count(t, vault, vcn1GroupByFlag, vcn1TypeAttr)

		// Then fact is printed before feedback
		g.Expect(outcome.stdout).To(Equal("fact\t2\nfeedback\t2\ntotal: 4\n"), "run %d", run)
	}
}

// Test792a8a1e_Count_OnlyNotesAtTheVaultRootAreCounted verifies 792a8a1e (vault-counting.md): a type: fact
// note in a subdirectory is not counted beside the one at the root.
func Test792a8a1e_Count_OnlyNotesAtTheVaultRootAreCounted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with one note at its root and another note in a subdirectory, both with type: fact
	vault := vcn1Vault{files: map[string]string{
		"root.md":       vcn1Note("type: fact"),
		"sub/nested.md": vcn1Note("type: fact"),
	}}

	// When the user runs engram count --group-by type
	outcome := vcn1Count(t, vault, vcn1GroupByFlag, vcn1TypeAttr)

	// Then fact is printed with the count 1
	g.Expect(outcome.stdout).To(Equal("fact\t1\ntotal: 1\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8b1a_Count_NotesWithoutTheAttributeAreReported verifies 792a8b1a (vault-counting.md): the absent
// line follows the value lines and precedes the total.
func Test792a8b1a_Count_NotesWithoutTheAttributeAreReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given one note with vocab: [x] and one note with no vocab key
	vault := vcn1Vault{files: map[string]string{
		"has.md":  vcn1Note("vocab: [x]"),
		"none.md": vcn1Note("type: fact"),
	}}

	// When the user runs engram count --group-by vocab
	outcome := vcn1Count(t, vault, vcn1GroupByFlag, "vocab")

	// Then stdout is "x", a tab and 1, then "(vocab absent): 1", then "total: 2"
	g.Expect(outcome.stdout).To(Equal("x\t1\n(vocab absent): 1\ntotal: 2\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8b1b_Count_NoNoteHasTheAttribute verifies 792a8b1b (vault-counting.md): 557 notes without the
// attribute print the absent line and the total, with no value line.
func Test792a8b1b_Count_NoNoteHasTheAttribute(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given 557 notes and none has the attribute attr
	const noteCount = 557

	notes := map[string]string{}
	vcn1AddTypedNotes(notes, "fact", noteCount)

	// When the user runs engram count --group-by attr
	outcome := vcn1Count(t, vcn1Vault{files: notes}, vcn1GroupByFlag, "attr")

	// Then stdout is "(attr absent): 557" then "total: 557", with no value line
	g.Expect(outcome.stdout).To(Equal("(attr absent): 557\ntotal: 557\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8b1d_Count_AnEmptyAttributeCountsOnlyTowardTheTotal verifies 792a8b1d (vault-counting.md): a note
// whose key foo has no value is in the total, in no value line and not in the absent line.
func Test792a8b1d_Count_AnEmptyAttributeCountsOnlyTowardTheTotal(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note whose frontmatter has the key foo with no value
	vault := vcn1Vault{files: map[string]string{"empty.md": vcn1Note("foo:")}}

	// When the user runs engram count --group-by foo
	outcome := vcn1Count(t, vault, vcn1GroupByFlag, "foo")

	// Then stdout is "total: 1"
	g.Expect(outcome.stdout).To(Equal("total: 1\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8c1a_Count_AScalarFilterMatchesByEquality verifies 792a8c1a (vault-counting.md): --filter
// type=feedback counts the feedback notes only and prints no other type.
func Test792a8c1a_Count_AScalarFilterMatchesByEquality(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given notes of type feedback and notes of type fact
	notes := map[string]string{}
	vcn1AddTypedNotes(notes, "feedback", 3)
	vcn1AddTypedNotes(notes, "fact", 2)

	// When the user runs engram count --group-by type --filter type=feedback
	outcome := vcn1Count(t, vcn1Vault{files: notes}, vcn1GroupByFlag, vcn1TypeAttr, vcn1FilterFlag, "type=feedback")

	// Then only the feedback notes are counted, and no other type is printed
	g.Expect(outcome.stdout).To(Equal("feedback\t3\ntotal: 3\n"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("fact"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8c1b_Count_AListFilterMatchesByMembership verifies 792a8c1b (vault-counting.md): a note whose
// tags list holds tier/haiku is counted under --filter tags=tier/haiku. A note listing other tags only is
// seeded so that a filter that matched everything would count it.
func Test792a8c1b_Count_AListFilterMatchesByMembership(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note with tags: [tier/haiku, vocab/x]
	vault := vcn1Vault{files: map[string]string{
		"member.md":    vcn1Note("tags: [tier/haiku, vocab/x]"),
		"nonmember.md": vcn1Note("tags: [tier/sonnet]"),
	}}

	// When the user runs engram count --group-by tags --filter tags=tier/haiku
	outcome := vcn1Count(t, vault, vcn1GroupByFlag, "tags", vcn1FilterFlag, "tags=tier/haiku")

	// Then the note is counted
	g.Expect(outcome.stdout).To(Equal("tier/haiku\t1\nvocab/x\t1\ntotal: 1\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8c1c_Count_EveryFilterMustMatch verifies 792a8c1c (vault-counting.md): with --filter given twice,
// only the note satisfying both is counted.
func Test792a8c1c_Count_EveryFilterMustMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a fact note with outcome: pass, a fact note with outcome: fail, and a feedback note with
	// outcome: pass
	vault := vcn1Vault{files: map[string]string{
		"fact-pass.md":     vcn1Note("type: fact", "outcome: pass"),
		"fact-fail.md":     vcn1Note("type: fact", "outcome: fail"),
		"feedback-pass.md": vcn1Note("type: feedback", "outcome: pass"),
	}}

	// When the user runs engram count --group-by outcome --filter type=fact --filter outcome=pass
	outcome := vcn1Count(t, vault,
		vcn1GroupByFlag, "outcome", vcn1FilterFlag, "type=fact", vcn1FilterFlag, "outcome=pass")

	// Then only the first note is counted, and the output is "pass", a tab and 1, then "total: 1"
	g.Expect(outcome.stdout).To(Equal("pass\t1\ntotal: 1\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8c1d_Count_ANoteWithoutTheFiltersAttributeIsExcluded verifies 792a8c1d (vault-counting.md): a
// note with no outcome key is not counted under --filter outcome=pass. A matching note is seeded beside it,
// so that counting the other note would change the printed count and total.
func Test792a8c1d_Count_ANoteWithoutTheFiltersAttributeIsExcluded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note that has no outcome key
	vault := vcn1Vault{files: map[string]string{
		"no-outcome.md": vcn1Note("type: fact"),
		"passing.md":    vcn1Note("type: fact", "outcome: pass"),
	}}

	// When the user runs engram count --group-by type --filter outcome=pass
	outcome := vcn1Count(t, vault, vcn1GroupByFlag, vcn1TypeAttr, vcn1FilterFlag, "outcome=pass")

	// Then the note is not counted
	g.Expect(outcome.stdout).To(Equal("fact\t1\ntotal: 1\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8c1e_Count_NoNoteMatchesTheFilter verifies 792a8c1e (vault-counting.md): a filter that no note
// satisfies prints nothing, not even a total.
func Test792a8c1e_Count_NoNoteMatchesTheFilter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with no feedback note
	notes := map[string]string{}
	vcn1AddTypedNotes(notes, "fact", 2)

	// When the user runs engram count --group-by type --filter type=feedback
	outcome := vcn1Count(t, vcn1Vault{files: notes}, vcn1GroupByFlag, vcn1TypeAttr, vcn1FilterFlag, "type=feedback")

	// Then nothing is printed
	g.Expect(outcome.stdout).To(BeEmpty())
	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8c1f_Count_AFilterWithoutEqualsIsRefused verifies 792a8c1f (vault-counting.md): --filter bad fails
// with the attr=value message quoting the filter, a non-zero exit and nothing on stdout.
func Test792a8c1f_Count_AFilterWithoutEqualsIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault
	notes := map[string]string{}
	vcn1AddTypedNotes(notes, "fact", 2)

	// When the user runs engram count --group-by type --filter bad
	outcome := vcn1Count(t, vcn1Vault{files: notes}, vcn1GroupByFlag, vcn1TypeAttr, vcn1FilterFlag, "bad")

	// Then it fails with an error saying --filter must be attr=value and quoting "bad", and prints nothing
	// to stdout
	g.Expect(outcome.stderr).To(Equal("count: --filter must be attr=value: \"bad\"\n"))
	g.Expect(outcome.exits).To(Equal([]int{1}))
	g.Expect(outcome.stdout).To(BeEmpty())
}

// Test792a8d1a_Count_ANoteWithoutUsableFrontmatterIsSkipped verifies 792a8d1a (vault-counting.md): a note
// with no frontmatter and a note with malformed YAML are in no value line, not absent and not in the total.
func Test792a8d1a_Count_ANoteWithoutUsableFrontmatterIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given one well-formed note with type: feedback, one note with no frontmatter, and one note whose
	// frontmatter is malformed YAML
	vault := vcn1Vault{files: map[string]string{
		"good.md":      vcn1Note("type: feedback"),
		"no-header.md": "just a body, no frontmatter\n",
		"malformed.md": "---\ntype: [unterminated\n---\nbody\n",
	}}

	// When the user runs engram count --group-by type
	outcome := vcn1Count(t, vault, vcn1GroupByFlag, vcn1TypeAttr)

	// Then stdout is "feedback", a tab and 1, then "total: 1"
	g.Expect(outcome.stdout).To(Equal("feedback\t1\ntotal: 1\n"))
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8d1b_Count_AnUnreadableNoteIsSkipped verifies 792a8d1b (vault-counting.md): a type: fact note whose
// read is denied is skipped, and the readable type: fact note is counted, with no error.
func Test792a8d1b_Count_AnUnreadableNoteIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with a note with type: fact that the user cannot read, and a readable note with
	// type: fact
	vault := vcn1Vault{
		files: map[string]string{
			"locked.md":   vcn1Note("type: fact"),
			"readable.md": vcn1Note("type: fact"),
		},
		unreadable: []string{"locked.md"},
	}

	// When the user runs engram count --group-by type
	outcome := vcn1Count(t, vault, vcn1GroupByFlag, vcn1TypeAttr)

	// Then fact is printed with the count 1, followed by "total: 1"
	g.Expect(outcome.stdout).To(Equal("fact\t1\ntotal: 1\n"))
	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.exits).To(BeEmpty())
}

// Test792a8d1c_Count_ACRLFNoteIsCounted verifies 792a8d1c (vault-counting.md): a note whose lines end in
// \r\n is counted exactly as the same note with LF line endings.
func Test792a8d1c_Count_ACRLFNoteIsCounted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note with type: feedback whose lines end in \r\n
	lfNote := vcn1Note("type: feedback")
	crlfNote := strings.ReplaceAll(lfNote, "\n", "\r\n")

	g.Expect(crlfNote).To(ContainSubstring("---\r\ntype: feedback\r\n---\r\n"))

	// When the user runs engram count --group-by type
	crlf := vcn1Count(t, vcn1Vault{files: map[string]string{"crlf.md": crlfNote}}, vcn1GroupByFlag, vcn1TypeAttr)
	lf := vcn1Count(t, vcn1Vault{files: map[string]string{"lf.md": lfNote}}, vcn1GroupByFlag, vcn1TypeAttr)

	// Then feedback is printed with the count 1, followed by "total: 1"
	g.Expect(crlf.stdout).To(Equal("feedback\t1\ntotal: 1\n"))
	g.Expect(crlf.exits).To(BeEmpty())
	g.Expect(crlf.stdout).To(Equal(lf.stdout), "the CRLF note counts exactly as the LF note")
}

// unexported constants.
const (
	vcn1FilterFlag  = "--filter"
	vcn1GroupByFlag = "--group-by"
	// vcn1TieRuns is how many times the equal-counts case is rebuilt and run.
	vcn1TieRuns   = 25
	vcn1TypeAttr  = "type"
	vcn1VaultPath = "/vcn1/vault"
)

// vcn1Outcome is what one engram count run printed and the exit codes it asked for.
type vcn1Outcome struct {
	stdout string
	stderr string
	exits  []int
}

// vcn1Vault is an in-memory vault: files maps a path relative to the vault directory to its text, and each
// name in unreadable is a file whose read is denied.
type vcn1Vault struct {
	files      map[string]string
	unreadable []string
}

// edgeFS serves the vault directory as existing, lists a directory's files and subdirectories, and reads
// the files. A write-capable call is left unset, so the command that makes one panics.
func (v vcn1Vault) edgeFS() fakeEdgeFS {
	denied := make(map[string]struct{}, len(v.unreadable))
	for _, name := range v.unreadable {
		denied[filepath.Join(vcn1VaultPath, name)] = struct{}{}
	}

	return fakeEdgeFS{
		readFile: func(path string) ([]byte, error) {
			clean := filepath.Clean(path)

			_, blocked := denied[clean]
			if blocked {
				return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}
			}

			for name, content := range v.files {
				if filepath.Join(vcn1VaultPath, name) == clean {
					return []byte(content), nil
				}
			}

			return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
		},
		readDir: func(dir string) ([]fs.DirEntry, error) {
			return v.listDir(filepath.Clean(dir)), nil
		},
		stat: func(path string) (fs.FileInfo, error) {
			if filepath.Clean(path) == vcn1VaultPath {
				return fakeFileInfo{dir: true}, nil
			}

			return nil, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
		},
	}
}

// listDir returns the entries directly inside dir, name-ordered: a file for each file there, and a directory
// entry for each subdirectory holding a file below it.
func (v vcn1Vault) listDir(dir string) []fs.DirEntry {
	isDir := map[string]bool{}

	for name := range v.files {
		rel, inside := strings.CutPrefix(filepath.Join(vcn1VaultPath, name), dir+string(filepath.Separator))
		if !inside {
			continue
		}

		first, _, nested := strings.Cut(rel, string(filepath.Separator))
		isDir[first] = isDir[first] || nested
	}

	names := make([]string, 0, len(isDir))
	for name := range isDir {
		names = append(names, name)
	}

	sort.Strings(names)

	entries := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, fakeDirEntry{name: name, dir: isDir[name]})
	}

	return entries
}

// vcn1AddTypedNotes adds count well-formed notes of the given type to notes, named after the type.
func vcn1AddTypedNotes(notes map[string]string, noteType string, count int) {
	for index := range count {
		notes[fmt.Sprintf("%s-%04d.md", noteType, index)] = vcn1Note("type: " + noteType)
	}
}

// vcn1Count runs `engram count <args>` through cli.Targets over the in-memory vault. A targ.Execute error
// fails the test and is kept out of stderr, which holds only what the command wrote.
func vcn1Count(t *testing.T, vault vcn1Vault, args ...string) vcn1Outcome {
	t.Helper()

	var (
		stdout, stderr bytes.Buffer
		exits          []int
	)

	deps := newTestDeps(&stdout, &stderr)
	deps.FS = vault.edgeFS()
	deps.DebugLog = nil
	deps.Getenv = func(string) string { return "" }
	deps.UserHomeDir = func() (string, error) { return vcn1VaultPath, nil }
	deps.Now = func() time.Time { return time.Time{} }
	deps.Exit = func(code int) { exits = append(exits, code) }

	argv := append([]string{"engram", "count", "--vault", vcn1VaultPath}, args...)

	_, err := targ.Execute(argv, cli.Targets(deps)...)
	if err != nil {
		t.Errorf("vcn1Count: targ.Execute: %v", err)
	}

	return vcn1Outcome{stdout: stdout.String(), stderr: stderr.String(), exits: exits}
}

// vcn1Note renders a note whose frontmatter is the given lines, with a one-line body.
func vcn1Note(frontmatterLines ...string) string {
	return "---\n" + strings.Join(frontmatterLines, "\n") + "\n---\nbody\n"
}
