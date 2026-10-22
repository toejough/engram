package cli_test

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func Test792b17c3a6_AncestorDepthIsThePhysicalDistance(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := specFS{dirs: map[string]bool{
		"/home/dev/proj/.pi":  true, // in the working directory: level 0
		"/home/dev/.claude":   true, // nearest .claude, one level up: level 1
		"/.claude":            true, // a farther .claude, three levels up: level 3
		"/home/dev/proj/.git": true,
	}}

	roots := cli.ResolveSweepRoots(cli.DefaultSweepSpec(), cli.SweepEnv{
		Cwd: "/home/dev/proj", SessionDir: "", IsDir: fs.isDir,
	})

	depthByPath := map[string]int{}
	for _, root := range roots {
		depthByPath[root.Path] = root.AncestorDepth
	}

	g.Expect(depthByPath).To(gomega.HaveKeyWithValue("/home/dev/proj/.pi", 0), "the .pi root is at the cwd")
	g.Expect(depthByPath).To(gomega.HaveKeyWithValue("/home/dev/.claude", 1),
		"the nearest .claude root is one level up, not the 0th .claude found")
	g.Expect(depthByPath).To(gomega.HaveKeyWithValue("/.claude", 3), "a farther .claude keeps its physical level")
}

func Test792b17c5a4_AnInvalidConfigurationFileIsAnError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		repoRoot     = "/repo"
		sweepFile    = ".engram/sweep.json"
		invalidJSON  = "{not valid json"
		chunksDirTop = "/chunks"
	)

	fs := newSweepFS()
	fs.put(filepath.Join(repoRoot, "README.md"), "## Conventions\nAlways wrap errors with context.\n", 1)
	fs.put(filepath.Join(repoRoot, sweepFile), invalidJSON, 1)

	deps := sweepDeps(fs, &countingEmbedder{})
	deps.IsDir = func(path string) bool { return path == filepath.Join(repoRoot, ".git") }
	deps.Getwd = func() (string, error) { return repoRoot, nil }
	deps.SessionDir = func(string) string { return "" }

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Auto: true, ChunksDir: chunksDirTop}, deps, io.Discard)

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(sweepFile)), "the error names the sweep file")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(filepath.Join(repoRoot, sweepFile))))

	_, indexed := fs.files[chunksDirTop+"/"+cli.ExportIndexFileName(filepath.Join(repoRoot, "README.md"))]
	g.Expect(indexed).To(gomega.BeFalse(), "nothing is ingested once gathering fails")
}

func Test792b17c8a1_OnlyDocumentsAndTranscriptsAreSources(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		rootDir   = "/root"
		chunksDir = "/chunks"
	)

	docPath := rootDir + "/a.md"
	transcriptPath := rootDir + "/b.jsonl"
	otherPath := rootDir + "/c.json"

	fs := newSweepFS()
	fs.put(docPath, "## Conventions\nAlways wrap errors with context before returning them.\n", 1)
	fs.put(transcriptPath, isd1TranscriptContent, 1)
	fs.put(otherPath, "## Conventions\nThis file has a document-like body but the wrong extension.\n", 1)

	// a lister that, like the walking one, reports every file regardless of extension.
	deps := sweepDeps(fs, &countingEmbedder{}, docPath, transcriptPath, otherPath)

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Sweep: []string{rootDir}, ChunksDir: chunksDir}, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(isd1Indexed(fs, chunksDir, docPath)).To(gomega.BeTrue(), "a.md is a source")
	g.Expect(isd1Indexed(fs, chunksDir, transcriptPath)).To(gomega.BeTrue(), "b.jsonl is a source")
	g.Expect(isd1Indexed(fs, chunksDir, otherPath)).To(gomega.BeFalse(), "c.json is not a source")
}

func Test792b17c8a2_TheChunkIndexIsNeverASource(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const (
		rootDir   = "/data"
		chunksDir = "/data/chunks"
	)

	notePath := rootDir + "/note.md"
	indexPath := chunksDir + "/some-source.jsonl"
	strayPath := chunksDir + "/nested/stray.md"
	siblingPath := rootDir + "/chunks-extra/keep.md" // shares the "chunks" name prefix but is not under the index

	fs := newSweepFS()
	fs.put(notePath, "## Conventions\nAlways wrap errors with context before returning them.\n", 1)
	fs.put(indexPath, isd1TranscriptContent, 1)
	fs.put(strayPath, "## Conventions\nA markdown file that happens to live under the chunk index directory.\n", 1)
	fs.put(siblingPath, "## Conventions\nA sibling directory whose name only starts like the index directory.\n", 1)

	// the swept root contains the chunk index directory, so the lister reports files under it.
	deps := sweepDeps(fs, &countingEmbedder{}, notePath, indexPath, strayPath, siblingPath)

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Sweep: []string{rootDir}, ChunksDir: chunksDir}, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(isd1Indexed(fs, chunksDir, notePath)).To(gomega.BeTrue(), "a file beside the index is a source")
	g.Expect(isd1Indexed(fs, chunksDir, siblingPath)).To(gomega.BeTrue(),
		"a sibling whose name merely starts like the index directory is a source")
	g.Expect(isd1Indexed(fs, chunksDir, indexPath)).To(gomega.BeFalse(), "an index file is not a source")
	g.Expect(isd1Indexed(fs, chunksDir, strayPath)).To(gomega.BeFalse(), "nothing under the index directory is")
}

// unexported constants.
const (
	isd1TranscriptContent = "USER: a conversation long enough to clear the ingest noise floor comfortably\n\n" +
		"ASSISTANT: acknowledged, indexing this conversation for recall"
)

// isd1Indexed reports whether ingest wrote a per-source index for source under chunksDir.
func isd1Indexed(fs *sweepFS, chunksDir, source string) bool {
	_, ok := fs.files[chunksDir+"/"+cli.ExportIndexFileName(source)]

	return ok
}
