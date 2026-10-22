package cli_test

import (
	"encoding/json"
	"io/fs"
	"maps"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

func Test792b18c15a10_ChunkIndexStore_MalformedManifestIsNoBacklog(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := cis1NewRecordingFS()
	fileSystem.files[cis1ChunksDir+"/manifest.json"] = []byte("{not valid json")

	var got bool

	g.Expect(func() {
		got = cli.ExportChunkIndexHasPrunableDuplicates(cis1ChunksDir, fileSystem)
	}).NotTo(Panic())
	g.Expect(got).To(BeFalse())
	g.Expect(fileSystem.mutations).To(BeEmpty())
}

func Test792b18c15a2_ChunkIndexStore_EmptyFileAmongNonEmptyIsFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newU1FS()
	fileSystem.files[cis1ChunksDir+"/a.jsonl"] = []byte("x\n")
	fileSystem.files[cis1ChunksDir+"/b.jsonl"] = []byte{}

	g.Expect(cli.ExportChunkIndexHasEmptyFiles(cis1ChunksDir, fileSystem)).To(BeTrue())
}

func Test792b18c15a3_ChunkIndexStore_NonEmptyFilesAreNotABacklog(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newU1FS()
	fileSystem.files[cis1ChunksDir+"/a.jsonl"] = []byte("x\n")

	g.Expect(cli.ExportChunkIndexHasEmptyFiles(cis1ChunksDir, fileSystem)).To(BeFalse())
}

func Test792b18c15a4_ChunkIndexStore_EmptyFileOfAnotherKindIsIgnored(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newU1FS()
	fileSystem.files[cis1ChunksDir+"/manifest.json"] = []byte{}

	g.Expect(cli.ExportChunkIndexHasEmptyFiles(cis1ChunksDir, fileSystem)).To(BeFalse())
}

func Test792b18c15a5_ChunkIndexStore_MissingDirectoryHasNoBacklog(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newU1FS()

	// Nothing seeded: ReadDir on the chunks directory returns fs.ErrNotExist.
	_, readErr := fileSystem.ReadDir(cis1ChunksDir)
	g.Expect(readErr).To(MatchError(fs.ErrNotExist))

	var got bool

	g.Expect(func() {
		got = cli.ExportChunkIndexHasEmptyFiles(cis1ChunksDir, fileSystem)
	}).NotTo(Panic())
	g.Expect(got).To(BeFalse())
}

func Test792b18c15a7_ChunkIndexStore_NonRemovableGroupsAreIgnored(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		manifest map[string]map[string]any
	}{
		{
			name: "distinct hashes",
			manifest: map[string]map[string]any{
				"/repo/a.md": cis1Entry("sha256:x"),
				"/repo/b.md": cis1Entry("sha256:y"),
			},
		},
		{
			name: "shared hash but different kinds",
			manifest: map[string]map[string]any{
				"/repo/a.md":    cis1Entry("sha256:x"),
				"/repo/a.jsonl": cis1Entry("sha256:x"),
			},
		},
		{
			name: "entry already recorded as a copy",
			manifest: map[string]map[string]any{
				"/repo/a.md": cis1Entry("sha256:x"),
				"/repo/b.md": cis1Entry("sha256:x", "/repo/a.md"),
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := cis1NewRecordingFS()
			cis1SeedManifest(g, fileSystem, testCase.manifest)

			g.Expect(cli.ExportChunkIndexHasPrunableDuplicates(cis1ChunksDir, fileSystem)).To(BeFalse())
			g.Expect(fileSystem.mutations).To(BeEmpty())
		})
	}
}

func Test792b18c15a8_ChunkIndexStore_CollapsibleGroupIsReportedWithoutChange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		indexFiles map[string]string
	}{
		{
			name: "duplicate never indexed",
			indexFiles: map[string]string{
				"/repo/a.md": cis1Record("/repo/a.md", "sha256:aaa"),
			},
		},
		{
			name: "records of the duplicate covered by the kept entry",
			indexFiles: map[string]string{
				"/repo/a.md": cis1Record("/repo/a.md", "sha256:aaa") + cis1Record("/repo/a.md", "sha256:bbb"),
				"/repo/b.md": cis1Record("/repo/b.md", "sha256:aaa"),
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := cis1NewRecordingFS()
			cis1SeedManifest(g, fileSystem, map[string]map[string]any{
				"/repo/a.md": cis1Entry("sha256:x"),
				"/repo/b.md": cis1Entry("sha256:x"),
			})

			for source, content := range testCase.indexFiles {
				fileSystem.files[cli.ExportIndexPathFor(cis1ChunksDir, source)] = []byte(content)
			}

			before := maps.Clone(fileSystem.files)

			g.Expect(cli.ExportChunkIndexHasPrunableDuplicates(cis1ChunksDir, fileSystem)).To(BeTrue())

			// Nothing is removed or written: no mutating call, and every file is as it was.
			g.Expect(fileSystem.mutations).To(BeEmpty())
			g.Expect(fileSystem.files).To(Equal(before))
		})
	}
}

func Test792b18c15a9_ChunkIndexStore_RefusalOnlyGroupIsSilent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		indexFiles map[string]string
	}{
		{
			name: "kept entry has no index file while the other's exists",
			indexFiles: map[string]string{
				"/repo/b.md": cis1Record("/repo/b.md", "sha256:aaa"),
			},
		},
		{
			name:       "no member is indexed",
			indexFiles: map[string]string{},
		},
		{
			name: "other holds a record the kept entry lacks",
			indexFiles: map[string]string{
				"/repo/a.md": cis1Record("/repo/a.md", "sha256:aaa"),
				"/repo/b.md": cis1Record("/repo/b.md", "sha256:ccc"),
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := cis1NewRecordingFS()
			cis1SeedManifest(g, fileSystem, map[string]map[string]any{
				"/repo/a.md": cis1Entry("sha256:x"),
				"/repo/b.md": cis1Entry("sha256:x"),
			})

			for source, content := range testCase.indexFiles {
				fileSystem.files[cli.ExportIndexPathFor(cis1ChunksDir, source)] = []byte(content)
			}

			g.Expect(cli.ExportChunkIndexHasPrunableDuplicates(cis1ChunksDir, fileSystem)).To(BeFalse())
			g.Expect(fileSystem.mutations).To(BeEmpty())
		})
	}
}

// unexported constants.
const (
	cis1ChunksDir = "/chunks"
)

// unexported variables.
var (
	_ update.Filesystem = (*cis1RecordingFS)(nil)
)

// cis1RecordingFS is a u1FS that records every mutating call, so a test can
// assert that a read-only check changed nothing.
type cis1RecordingFS struct {
	*u1FS

	mutations []string
}

func (r *cis1RecordingFS) MkdirAll(path string, perm fs.FileMode) error {
	r.mutations = append(r.mutations, "MkdirAll "+path)

	return r.u1FS.MkdirAll(path, perm)
}

func (r *cis1RecordingFS) RemoveAll(path string) error {
	r.mutations = append(r.mutations, "RemoveAll "+path)

	return r.u1FS.RemoveAll(path)
}

func (r *cis1RecordingFS) Symlink(target, link string) error {
	r.mutations = append(r.mutations, "Symlink "+link)

	return r.u1FS.Symlink(target, link)
}

func (r *cis1RecordingFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	r.mutations = append(r.mutations, "WriteFile "+path)

	return r.u1FS.WriteFile(path, data, perm)
}

// cis1Entry builds a manifest entry; the optional argument marks it a copy of that source.
func cis1Entry(hash string, duplicateOf ...string) map[string]any {
	entry := map[string]any{"mtime_unix_nano": 1, "size": 10, "file_hash": hash}
	if len(duplicateOf) > 0 {
		entry["duplicate_of"] = duplicateOf[0]
	}

	return entry
}

func cis1NewRecordingFS() *cis1RecordingFS {
	return &cis1RecordingFS{u1FS: newU1FS()}
}

// cis1Record builds one per-source index record line.
func cis1Record(source, hash string) string {
	return `{"source":"` + source + `","anchor":"turn-1","content_hash":"` + hash +
		`","text":"t","vector":[1]}` + "\n"
}

// cis1SeedManifest writes the manifest of the chunks directory.
func cis1SeedManifest(g *WithT, fileSystem *cis1RecordingFS, manifest map[string]map[string]any) {
	data, err := json.Marshal(manifest)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	fileSystem.files[cis1ChunksDir+"/manifest.json"] = data
}
