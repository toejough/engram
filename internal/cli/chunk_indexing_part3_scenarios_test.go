package cli_test

// Scenarios of 792a5r (Prune removes duplicate index files) in docs/specs/product/chunk-indexing.md. Each runs
// `engram prune --duplicates` through cli.Targets over cli.Deps whose file system edge is an in-memory fake chunk
// directory, so the command path (flag parsing, chunks-dir resolution, the manifest lock, the report, the failure
// exit) is the one the user runs and nothing touches the real disk.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/toejough/targ"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

func Test792a5r1a_ChunkIndexing_ADuplicateGroupIsCollapsed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// /a/doc.md has the fewest separators; /a/b/doc.md sorts first byte-wise, so a plain lexicographic choice
	// would keep the wrong copy.
	machine := ci3NewMachine(g, ci3Sources{
		kept:    ci3Doc,
		removed: []string{ci3DocDeep},
	})

	outcome := machine.run("prune", "--duplicates", "--chunks-dir", ci3ChunksDir)

	g.Expect(outcome.stdout).To(Equal("prune: removed 1 duplicate(s), retained 1 canonical(s)\n"))
	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.exits).To(BeEmpty())

	g.Expect(machine.files).To(HaveKey(ci3IndexPath(ci3Doc)), "the kept copy keeps its chunks")
	g.Expect(machine.files).NotTo(HaveKey(ci3IndexPath(ci3DocDeep)), "the removed copy no longer has chunks")

	manifest := machine.manifest(g)
	g.Expect(manifest).To(HaveKey(ci3Doc))
	g.Expect(manifest).NotTo(HaveKey(ci3DocDeep), "the removed copy no longer has a record")
}

func Test792a5r1b_ChunkIndexing_ADuplicateWithChunksOfItsOwnIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci3NewMachine(g, ci3Sources{
		kept:    ci3Doc,
		removed: []string{ci3DocDeep},
		extraChunks: map[string][]string{
			ci3DocDeep: {"a section only this copy holds"},
		},
	})
	before := maps.Clone(machine.files)

	outcome := machine.run("prune", "--duplicates", "--chunks-dir", ci3ChunksDir)

	g.Expect(ci3Lines(outcome.stdout)).To(ContainElement(HavePrefix("prune: refusing to remove " + ci3DocDeep + ":")))
	g.Expect(outcome.stdout).To(ContainSubstring("removed 0 duplicate(s), retained 1 canonical(s)"))
	g.Expect(outcome.exits).To(BeEmpty(), "a refusal is not a failure")

	g.Expect(machine.files).To(Equal(before), "both copies keep their chunks and the manifest is untouched")
	g.Expect(machine.mutations).To(BeEmpty())
}

func Test792a5r1c_ChunkIndexing_ADryRunRemovesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci3NewMachine(g, ci3Sources{
		kept:    ci3Doc,
		removed: []string{ci3DocDeep},
	})
	before := maps.Clone(machine.files)

	outcome := machine.run("prune", "--duplicates", "--dry-run", "--chunks-dir", ci3ChunksDir)

	g.Expect(outcome.stdout).To(Equal("[dry-run] prune: removed 1 duplicate(s), retained 1 canonical(s)\n"))
	g.Expect(outcome.exits).To(BeEmpty())

	g.Expect(machine.files).To(HaveKey(ci3IndexPath(ci3Doc)))
	g.Expect(machine.files).To(HaveKey(ci3IndexPath(ci3DocDeep)))
	g.Expect(machine.files).To(Equal(before), "both copies and the manifest are as they were")
	g.Expect(machine.mutations).To(BeEmpty(), "a dry run neither removes nor writes")
}

func Test792a5r1d_ChunkIndexing_ASecondRunRemovesNothingMore(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci3NewMachine(g, ci3Sources{
		kept:    ci3Doc,
		removed: []string{ci3DocDeep},
	})

	// The Given: the first run has just happened, through the command.
	first := machine.run("prune", "--duplicates", "--chunks-dir", ci3ChunksDir)
	g.Expect(first.exits).To(BeEmpty())
	g.Expect(machine.files).NotTo(HaveKey(ci3IndexPath(ci3DocDeep)), "the first run collapsed the group")

	settled := maps.Clone(machine.files)
	machine.mutations = nil

	outcome := machine.run("prune", "--duplicates", "--chunks-dir", ci3ChunksDir)

	g.Expect(outcome.stdout).To(Equal("prune: removed 0 duplicate(s), retained 0 canonical(s)\n"))
	g.Expect(outcome.exits).To(BeEmpty())
	g.Expect(machine.files).To(Equal(settled), "nothing changes")
	g.Expect(machine.mutations).To(BeEmpty(), "the second run neither removes nor writes")
}

func Test792a5r1e_ChunkIndexing_AFailedRemovalIsReportedAndFailsTheRun(t *testing.T) {
	t.Parallel()

	// Two removable duplicates; the first or the last in the order the command visits them cannot be removed, so
	// "the other is removed" shows the run goes on past a failure in either position.
	cases := []struct {
		name    string
		failing string
		other   string
	}{
		{name: "the first duplicate cannot be removed", failing: ci3DocDeep, other: ci3DocSibling},
		{name: "the last duplicate cannot be removed", failing: ci3DocSibling, other: ci3DocDeep},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := ci3NewMachine(g, ci3Sources{
				kept:    ci3Doc,
				removed: []string{ci3DocDeep, ci3DocSibling},
			})
			machine.removeFailures[ci3IndexPath(testCase.failing)] = errCi3Locked

			outcome := machine.run("prune", "--duplicates", "--chunks-dir", ci3ChunksDir)

			failLines := ci3FailLines(outcome.stderr)
			g.Expect(failLines).To(HaveLen(1))

			if len(failLines) != 1 {
				return
			}

			g.Expect(failLines[0]).To(ContainSubstring("[FAIL] " + testCase.failing + ":"))
			g.Expect(failLines[0]).To(ContainSubstring(ci3IndexPath(testCase.failing)))
			g.Expect(outcome.stderr).To(ContainSubstring("prune: 1 of 2 removals failed"))
			g.Expect(outcome.exits).To(Equal([]int{1}))

			g.Expect(machine.files).To(HaveKey(ci3IndexPath(testCase.failing)), "the file that cannot be removed stays")
			g.Expect(machine.files).NotTo(HaveKey(ci3IndexPath(testCase.other)), "the other duplicate is removed")
			g.Expect(machine.files).To(HaveKey(ci3IndexPath(ci3Doc)))

			manifest := machine.manifest(g)
			g.Expect(manifest).To(HaveKey(testCase.failing), "the failed removal's record stays for a retry")
			g.Expect(manifest).NotTo(HaveKey(testCase.other))
			g.Expect(manifest).To(HaveKey(ci3Doc))
		})
	}
}

// unexported constants.
const (
	ci3ChunksDir   = "/chunks"
	ci3Doc         = "/a/doc.md"
	ci3DocDeep     = "/a/b/doc.md"
	ci3DocSibling  = "/a/c/doc.md"
	ci3ManifestKey = "ci3-shared-content-hash"
)

// unexported variables.
var (
	errCi3Locked = errors.New("ci3: the index file is locked")
	errCi3Unused = errors.New("ci3: the fake file system does not serve this call")
)

// ci3Machine is an in-memory chunk directory behind cli.Deps: the files by path, the removals that fail, and
// every mutating call the command made.
type ci3Machine struct {
	files          map[string][]byte
	removeFailures map[string]error
	mutations      []string
}

// edgeFS returns the machine as the file system edge of cli.Deps.
func (m *ci3Machine) edgeFS() cli.EdgeFS {
	return fakeEdgeFS{
		readFile: func(path string) ([]byte, error) {
			data, found := m.files[path]
			if !found {
				return nil, fmt.Errorf("read %s: %w", path, fs.ErrNotExist)
			}

			return data, nil
		},
		writeFile: func(path string, data []byte, _ fs.FileMode) error { return m.write("write", path, data) },
		writeFileAtomic: func(path string, data []byte, _ fs.FileMode) error {
			return m.write("write atomic", path, data)
		},
		writeFileExcl: func(string, []byte, fs.FileMode) error { return errCi3Unused },
		mkdirAll:      func(string, fs.FileMode) error { return nil },
		mkdirTemp:     func(string, string) (string, error) { return "", errCi3Unused },
		stat:          m.stat,
		lstat:         m.stat,
		readDir:       func(string) ([]fs.DirEntry, error) { return nil, errCi3Unused },
		remove:        m.remove,
		removeAll:     func(string) error { return errCi3Unused },
		rename:        func(string, string) error { return errCi3Unused },
		symlink:       func(string, string) error { return errCi3Unused },
		readlink:      func(string) (string, error) { return "", errCi3Unused },
		walkDir:       func(string, fs.WalkDirFunc) error { return errCi3Unused },
	}
}

// manifest decodes the manifest of the chunk directory as the machine holds it now.
func (m *ci3Machine) manifest(g *WithT) map[string]map[string]any {
	manifest := map[string]map[string]any{}

	data, found := m.files[ci3ChunksDir+"/manifest.json"]
	g.Expect(found).To(BeTrue(), "the chunk directory has a manifest")

	if !found {
		return manifest
	}

	g.Expect(json.Unmarshal(data, &manifest)).To(Succeed())

	return manifest
}

// remove drops the file unless the machine was told that removal fails.
func (m *ci3Machine) remove(path string) error {
	m.mutations = append(m.mutations, "remove "+path)

	failure, failing := m.removeFailures[path]
	if failing {
		return failure
	}

	delete(m.files, path)

	return nil
}

// run runs `engram <args>` through cli.Targets over the machine and collects what it printed and which exit
// statuses it asked for.
func (m *ci3Machine) run(args ...string) ci3Outcome {
	var (
		stdout, stderr bytes.Buffer
		exits          []int
	)

	deps := cli.Deps{
		Stdout:      &stdout,
		Stderr:      &stderr,
		Exit:        func(code int) { exits = append(exits, code) },
		Getenv:      func(string) string { return "" },
		Now:         func() time.Time { return time.Time{} },
		UserHomeDir: func() (string, error) { return "/home/ci3", nil },
		FS:          m.edgeFS(),
		Lock: fakeLocker{lock: func(string) (func() error, error) {
			return func() error { return nil }, nil
		}},
	}

	_, err := targ.Execute(append([]string{"engram"}, args...), cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return ci3Outcome{stdout: stdout.String(), stderr: stderr.String(), exits: exits}
}

// stat answers for the files of the machine.
func (m *ci3Machine) stat(path string) (fs.FileInfo, error) {
	data, found := m.files[path]
	if !found {
		return nil, fmt.Errorf("stat %s: %w", path, fs.ErrNotExist)
	}

	return fakeFileInfo{size: int64(len(data))}, nil
}

// write stores the file and records the call.
func (m *ci3Machine) write(operation, path string, data []byte) error {
	m.mutations = append(m.mutations, operation+" "+path)
	m.files[path] = data

	return nil
}

// ci3Outcome is what one run of a command produced.
type ci3Outcome struct {
	stdout string
	stderr string
	exits  []int
}

// ci3Sources describes the indexed copies of one content: the copy the command keeps, the copies it should
// remove, and chunks that only some copy holds.
type ci3Sources struct {
	kept        string
	removed     []string
	extraChunks map[string][]string
}

// ci3FailLines returns the lines of the output that report a failed removal.
func ci3FailLines(output string) []string {
	lines := ci3Lines(output)

	return slices.DeleteFunc(lines, func(line string) bool { return !strings.Contains(line, "[FAIL]") })
}

// ci3IndexPath is the index file of a source in the chunk directory, named the way ingestion names it.
func ci3IndexPath(source string) string {
	return ci3ChunksDir + "/" + cli.ExportIndexFileName(source)
}

// ci3Lines splits output into its non-empty lines.
func ci3Lines(output string) []string {
	return slices.DeleteFunc(strings.Split(output, "\n"), func(line string) bool { return line == "" })
}

// ci3NewMachine builds a chunk directory in which every listed source is recorded with one content hash and
// indexed with the same two chunks, plus the extra chunks that a source holds on its own.
func ci3NewMachine(g *WithT, sources ci3Sources) *ci3Machine {
	machine := &ci3Machine{files: map[string][]byte{}, removeFailures: map[string]error{}}
	shared := []string{"the first section of the document", "the second section of the document"}
	all := append([]string{sources.kept}, sources.removed...)

	for _, source := range all {
		texts := append(slices.Clone(shared), sources.extraChunks[source]...)
		records := make([]chunk.Record, 0, len(texts))

		for position, text := range texts {
			records = append(records, chunk.Record{
				Source:      source,
				Anchor:      fmt.Sprintf("section-%d", position+1),
				ContentHash: chunk.HashText(text),
				Text:        text,
				Vector:      []float32{1},
			})
		}

		encoded, err := chunk.EncodeRecords(records)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return machine
		}

		machine.files[ci3IndexPath(source)] = encoded
	}

	manifest := map[string]map[string]any{}
	for _, source := range all {
		manifest[source] = cis1Entry(ci3ManifestKey)
	}

	encoded, err := json.Marshal(manifest)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return machine
	}

	machine.files[ci3ChunksDir+"/manifest.json"] = encoded

	return machine
}
