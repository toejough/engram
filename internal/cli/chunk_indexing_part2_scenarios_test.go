package cli_test

// Scenario tests for the Chunk Indexing product capability, part 2 (861c2zzn): chunk boundaries and IDs, incremental
// re-indexing, identical content indexed once, stray vault-note copies, concurrent writers and prune (792a5h to
// 792a5q). Spec: docs/specs/product/chunk-indexing.md. Each scenario runs cli.RunIngest, cli.RunPrune or cli.RunQuery
// over the production compositions (cli.ExportNewIngestDeps, cli.ExportNewPruneDeps, cli.ExportNewQueryDeps) built from
// a cli.Deps whose file system is an in-memory tree (ci2Tree) and whose lock is an in-memory file locker, so the real
// sweep walk, the real transcript reader, the real chunker and the real manifest lock all run. The helpers carry the
// ci2 prefix. The scenarios of 792a5r are in chunk_indexing_part3_scenarios_test.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// Test792a5h1a_RunIngest_MarkdownIsChunkedByHeading verifies 792a5h1a (Markdown is chunked by heading): of d.md with
// text before any heading, a "## Setup" section and a "## Usage" section, each over 40 characters, the index holds
// the chunks d.md#preamble, d.md#Setup and d.md#Usage and no other.
func Test792a5h1a_RunIngest_MarkdownIsChunkedByHeading(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.put("/work/d.md", "This text comes before any heading and is long enough to be a chunk.\n\n"+
		"## Setup\nInstall the binary and point it at a vault before the first run.\n\n"+
		"## Usage\nRun the ingest command with a markdown file to index its sections.\n")

	_, err := machine.ingest(t, cli.IngestArgs{Markdowns: []string{"/work/d.md"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.chunkIDs(t)).To(ConsistOf("/work/d.md#preamble", "/work/d.md#Setup", "/work/d.md#Usage"))
}

// Test792a5h1b_RunIngest_ATranscriptIsChunkedByTurns verifies 792a5h1b (A transcript is chunked by turns): of s.jsonl
// with many short user and assistant turns, each chunk holds consecutive whole turns of about 500 characters, the
// first chunk is s.jsonl#turn-1, and each later chunk is anchored at the turn where it starts.
func Test792a5h1b_RunIngest_ATranscriptIsChunkedByTurns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		turns        = 30
		minChunkSize = 450
		maxChunkSize = 700
	)

	machine := ci2NewMachine("/work")
	machine.tree.putBytes("/work/s.jsonl", ci2SessionRows(turns, "the manifest lock"))

	_, err := machine.ingest(t, cli.IngestArgs{Transcripts: []string{"/work/s.jsonl"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	records := machine.recordsOf(t, "/work/s.jsonl")

	const minChunks = 3

	g.Expect(len(records)).To(BeNumerically(">=", minChunks), "many short turns make several chunks")

	nextTurn := 1

	for position, record := range records {
		wantAnchor := "turn-" + strconv.Itoa(nextTurn)
		g.Expect(record.Anchor).To(Equal(wantAnchor), "chunk %d starts where the previous one stopped", position)
		g.Expect(record.Text).To(HavePrefix(fmt.Sprintf("USER: question %d about", nextTurn)),
			"chunk %d begins with the user turn named by its anchor", position)

		if position < len(records)-1 {
			g.Expect(len(record.Text)).To(And(BeNumerically(">=", minChunkSize), BeNumerically("<=", maxChunkSize)),
				"chunk %d is about 500 characters", position)
		}

		nextTurn += strings.Count(record.Text, "USER: ")
	}

	g.Expect(nextTurn).To(Equal(turns+1), "the chunks together hold every turn, in order, once")
}

// Test792a5h1c_RunIngest_ShortSectionsAreNotIndexed verifies 792a5h1c (Short sections are not indexed): d.md whose only
// section is "## A" followed by "tiny" has no chunk in the index.
func Test792a5h1c_RunIngest_ShortSectionsAreNotIndexed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.put("/work/d.md", "## A\ntiny\n")

	_, err := machine.ingest(t, cli.IngestArgs{Markdowns: []string{"/work/d.md"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.recordsOf(t, "/work/d.md")).To(BeEmpty())
}

// Test792a5h1d_RunIngest_ALongSectionIsSplitUnderOneAnchor verifies 792a5h1d (A long section is split under one
// anchor): of d.md whose "## Big" section is three paragraphs of about 940 characters each, the index holds several
// chunks d.md#Big, each at most 1500 characters, that pack whole lines in order: every line is in exactly one chunk,
// and a new chunk starts only where its first line would have taken the previous one past 1500 characters.
func Test792a5h1d_RunIngest_ALongSectionIsSplitUnderOneAnchor(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		maxChars       = 1500
		paragraphs     = 3
		linesPerParagh = 10
		lineChars      = 93
	)

	var section strings.Builder

	allLines := make([]string, 0, paragraphs*linesPerParagh)

	section.WriteString("## Big\n")

	for paragraph := range paragraphs {
		lines := make([]string, 0, linesPerParagh)

		for line := range linesPerParagh {
			prefix := fmt.Sprintf("paragraph %d line %02d ", paragraph+1, line+1)
			lines = append(lines, prefix+strings.Repeat("x", lineChars-len(prefix)))
		}

		allLines = append(allLines, lines...)

		section.WriteString(strings.Join(lines, "\n"))
		section.WriteString("\n\n")
	}

	machine := ci2NewMachine("/work")
	machine.tree.put("/work/d.md", section.String())

	_, err := machine.ingest(t, cli.IngestArgs{Markdowns: []string{"/work/d.md"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	records := machine.recordsOf(t, "/work/d.md")
	g.Expect(len(records)).To(BeNumerically(">", 1), "a section over 1500 characters is split into several chunks")

	var seen []string

	for position, record := range records {
		g.Expect(record.Anchor).To(Equal("Big"), "chunk %d keeps the section's anchor", position)
		g.Expect(len(record.Text)).To(BeNumerically("<=", maxChars), "chunk %d is at most 1500 characters", position)

		chunkLines := strings.Split(record.Text, "\n")
		seen = append(seen, chunkLines...)

		if position+1 < len(records) {
			nextFirst, _, _ := strings.Cut(records[position+1].Text, "\n")
			g.Expect(len(record.Text)+1+len(nextFirst)).To(BeNumerically(">", maxChars),
				"chunk %d ended only because the next line did not fit", position)
		}
	}

	g.Expect(seen).To(Equal(append([]string{"## Big"}, allLines...)), "the chunks pack every line, in order, once")
}

// Test792a5i1a_RunIngest_AnUnchangedSourceCostsNothing verifies 792a5i1a (An unchanged source costs nothing): of a
// source that was ingested and has not changed since, a second ingest does not read it, embeds nothing and prints
// no line for it.
func Test792a5i1a_RunIngest_AnUnchangedSourceCostsNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.put("/work/doc.md", ci2TwoSectionDocument("first text", "second text"))

	args := cli.IngestArgs{Markdowns: []string{"/work/doc.md"}}

	first, err := machine.ingest(t, args)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(first).To(ContainSubstring("ingest /work/doc.md:"), "sanity: the first ingest indexes the source")

	readsBefore := machine.tree.readCount("/work/doc.md")
	embedsBefore := machine.embedder.calls.Load()

	second, err := machine.ingest(t, args)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.tree.readCount("/work/doc.md")).To(Equal(readsBefore), "the unchanged source is not read")
	g.Expect(machine.embedder.calls.Load()).To(Equal(embedsBefore), "nothing is embedded")
	g.Expect(second).To(BeEmpty(), "stdout has no line for the source")
}

// Test792a5i1b_RunIngest_AChangedSourceEmbedsOnlyItsNewText verifies 792a5i1b (A changed source embeds only its new
// text): of a Markdown document with sections A and B that was ingested, after which section B was edited, the second
// ingest prints "ingest <path>: 3 chunks (1 reused, 1 embedded)", embeds exactly one text (B's new text), and the
// index holds A's chunk, B's earlier text and B's new text.
func Test792a5i1b_RunIngest_AChangedSourceEmbedsOnlyItsNewText(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.put("/work/doc.md", ci2TwoSectionDocument("text of section A", "earlier text of section B"))

	args := cli.IngestArgs{Markdowns: []string{"/work/doc.md"}}

	_, err := machine.ingest(t, args)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	machine.tree.put("/work/doc.md", ci2TwoSectionDocument("text of section A", "edited text of section B"))

	embedsBefore := machine.embedder.calls.Load()

	out, err := machine.ingest(t, args)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("ingest /work/doc.md: 3 chunks (1 reused, 1 embedded)\n"))
	g.Expect(machine.embedder.calls.Load()-embedsBefore).To(BeEquivalentTo(1), "only the new text is embedded")

	texts := make([]string, 0, 3)
	for _, record := range machine.recordsOf(t, "/work/doc.md") {
		texts = append(texts, record.Text)
	}

	g.Expect(texts).To(HaveLen(3))
	g.Expect(ci2CountContaining(texts, "text of section A")).To(Equal(1))
	g.Expect(ci2CountContaining(texts, "earlier text of section B")).To(Equal(1))
	g.Expect(ci2CountContaining(texts, "edited text of section B")).To(Equal(1))
}

// Test792a5j1a_RunIngest_EditedAwayTextStaysSearchable verifies 792a5j1a (Edited-away text stays searchable): of a
// document ingested with the paragraph "use flock for the manifest", after the paragraph is replaced and the document
// ingested again, the index holds both the old paragraph's chunk and the new text's chunk.
func Test792a5j1a_RunIngest_EditedAwayTextStaysSearchable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.put("/work/doc.md", "## Notes\nuse flock for the manifest so that concurrent runs lose no update\n")

	args := cli.IngestArgs{Markdowns: []string{"/work/doc.md"}}

	_, err := machine.ingest(t, args)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	machine.tree.put("/work/doc.md",
		"## Notes\nwrite the manifest through a rename so that readers never see a torn file\n")

	_, err = machine.ingest(t, args)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	texts := make([]string, 0, 2)
	for _, record := range machine.recordsOf(t, "/work/doc.md") {
		texts = append(texts, record.Text)
	}

	g.Expect(ci2CountContaining(texts, "use flock for the manifest")).To(Equal(1), "the old paragraph's chunk stays")
	g.Expect(ci2CountContaining(texts, "through a rename")).To(Equal(1), "the new text's chunk is added")
}

// Test792a5k1a_RunIngest_AnEmptyTranscriptWritesNoIndexFile verifies 792a5k1a (An empty transcript writes no index
// file): ingesting an empty e.jsonl prints "ingest e.jsonl: 0 chunks (0 reused, 0 embedded)" and leaves no index
// file for it.
func Test792a5k1a_RunIngest_AnEmptyTranscriptWritesNoIndexFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.putBytes("/work/e.jsonl", nil)

	out, err := machine.ingest(t, cli.IngestArgs{Transcripts: []string{"/work/e.jsonl"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("ingest /work/e.jsonl: 0 chunks (0 reused, 0 embedded)\n"))
	g.Expect(machine.tree.has(ci2IndexPath("/work/e.jsonl"))).To(BeFalse(), "no index file exists for e.jsonl")
	g.Expect(machine.tree.indexFileNames()).To(BeEmpty(), "the chunk index holds no file at all")
}

// Test792a5l1a_RunIngest_ARepositoryFileOutranksItsCopyInAnAncestorClaudeDirectory verifies 792a5l1a (A repository
// file outranks its copy in an ancestor .claude directory): of byte-identical /r/doc.md and /r/.claude/doc.md,
// ingest --auto from /r indexes /r/doc.md and not /r/.claude/doc.md.
func Test792a5l1a_RunIngest_ARepositoryFileOutranksItsCopyInAnAncestorClaudeDirectory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/r")
	machine.tree.put("/r/doc.md", ci2IdenticalDocument)
	machine.tree.put("/r/.claude/doc.md", ci2IdenticalDocument)

	_, err := machine.ingest(t, cli.IngestArgs{Auto: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.indexedSources(t)).To(Equal([]string{"/r/doc.md"}))
}

// Test792a5l1b_RunIngest_ANamedFileOutranksTheRepositoryCopy verifies 792a5l1b (A named file outranks the repository
// copy): of byte-identical /x/doc.md and /r/doc.md in the repository, ingest --auto --markdown /x/doc.md from /r
// indexes /x/doc.md and not /r/doc.md. The path /r/doc.md sorts before /x/doc.md, so only the precedence of a named
// file puts /x/doc.md first.
func Test792a5l1b_RunIngest_ANamedFileOutranksTheRepositoryCopy(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/r")
	machine.tree.put("/x/doc.md", ci2IdenticalDocument)
	machine.tree.put("/r/doc.md", ci2IdenticalDocument)

	_, err := machine.ingest(t, cli.IngestArgs{Auto: true, Markdowns: []string{"/x/doc.md"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.indexedSources(t)).To(Equal([]string{"/x/doc.md"}))
}

// Test792a5l1c_RunIngest_TheNearerAncestorDirectoryWins verifies 792a5l1c (The nearer ancestor directory wins): of
// byte-identical files in /r/a/.pi and in /r/.claude with the working directory /r/a, ingest --auto from /r/a
// indexes the /r/a/.pi copy and not the /r/.claude copy. The /r/.claude copy has the fewer path separators, so only
// the nearness of the directory puts the /r/a/.pi copy first.
func Test792a5l1c_RunIngest_TheNearerAncestorDirectoryWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/r/a")
	machine.tree.put("/r/a/.pi/doc.md", ci2IdenticalDocument)
	machine.tree.put("/r/.claude/doc.md", ci2IdenticalDocument)

	_, err := machine.ingest(t, cli.IngestArgs{Auto: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.indexedSources(t)).To(Equal([]string{"/r/a/.pi/doc.md"}))
}

// Test792a5l1d_RunIngest_ATranscriptAndADocumentWithTheSameBytesAreBothIndexed verifies 792a5l1d (A transcript and a
// document with the same bytes are both indexed): ingesting a.jsonl and a.md with byte-identical content gives both
// files chunks in the index.
func Test792a5l1d_RunIngest_ATranscriptAndADocumentWithTheSameBytesAreBothIndexed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	rows := ci2SessionRows(1, "the manifest lock")

	machine := ci2NewMachine("/work")
	machine.tree.putBytes("/work/a.jsonl", rows)
	machine.tree.putBytes("/work/a.md", rows)

	_, err := machine.ingest(t, cli.IngestArgs{Transcripts: []string{"/work/a.jsonl"}, Markdowns: []string{"/work/a.md"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.recordsOf(t, "/work/a.jsonl")).NotTo(BeEmpty(), "the transcript has chunks")
	g.Expect(machine.recordsOf(t, "/work/a.md")).NotTo(BeEmpty(), "the document has chunks")
}

// Test792a5m1a_RunIngest_ACopyIndexedFirstGivesWayToThePreferredOne verifies 792a5m1a (A copy indexed first gives way
// to the preferred one): /r/.claude/doc.md indexed while /r/doc.md did not exist stops having chunks once /r/doc.md
// is created with the same bytes and ingest --auto runs from /r, which indexes /r/doc.md.
func Test792a5m1a_RunIngest_ACopyIndexedFirstGivesWayToThePreferredOne(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/r")
	machine.tree.put("/r/.claude/doc.md", ci2IdenticalDocument)

	_, err := machine.ingest(t, cli.IngestArgs{Auto: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.indexedSources(t)).To(Equal([]string{"/r/.claude/doc.md"}), "sanity: the only copy is indexed")

	machine.tree.put("/r/doc.md", ci2IdenticalDocument)

	out, err := machine.ingest(t, cli.IngestArgs{Auto: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.indexedSources(t)).To(Equal([]string{"/r/doc.md"}))
	g.Expect(out).NotTo(ContainSubstring("refusing to evict"))
}

// Test792a5m1b_RunIngest_ACopyWithAHistoryOfItsOwnIsKept verifies 792a5m1b (A copy with a history of its own is
// kept): of /r/.claude/doc.md indexed with an earlier version and then edited to equal a new /r/doc.md, ingest --auto
// from /r keeps the chunks of /r/.claude/doc.md and prints the refusal line naming both copies.
func Test792a5m1b_RunIngest_ACopyWithAHistoryOfItsOwnIsKept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		earlierText = "an earlier version of the document that only the .claude copy ever held"
		sharedText  = "the current text that both copies now share, long enough to make a chunk"
	)

	machine := ci2NewMachine("/r")
	machine.tree.put("/r/.claude/doc.md", "## Notes\n"+earlierText+"\n")

	_, err := machine.ingest(t, cli.IngestArgs{Auto: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	machine.tree.put("/r/.claude/doc.md", "## Notes\n"+sharedText+"\n")
	machine.tree.put("/r/doc.md", "## Notes\n"+sharedText+"\n")

	out, err := machine.ingest(t, cli.IngestArgs{Auto: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("ingest: refusing to evict /r/.claude/doc.md: canonical /r/doc.md does not " +
		"verifiably cover its records (needs review — run `engram prune --duplicates` once resolved)\n"))

	kept := make([]string, 0, 1)
	for _, record := range machine.recordsOf(t, "/r/.claude/doc.md") {
		kept = append(kept, record.Text)
	}

	g.Expect(ci2CountContaining(kept, earlierText)).To(Equal(1), "the earlier version's chunk is kept")
	g.Expect(machine.recordsOf(t, "/r/doc.md")).NotTo(BeEmpty(), "the preferred copy is indexed as well")
}

// Test792a5m1c_RunIngest_TheRemainingCopyIsIndexedWhenThePreferredOneGoesAway verifies 792a5m1c (The remaining copy is
// indexed when the preferred one goes away): of /r/doc.md indexed with /r/.claude/doc.md recorded as its copy,
// deleting /r/doc.md and running ingest --auto from /r indexes /r/.claude/doc.md.
func Test792a5m1c_RunIngest_TheRemainingCopyIsIndexedWhenThePreferredOneGoesAway(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/r")
	machine.tree.put("/r/doc.md", ci2IdenticalDocument)
	machine.tree.put("/r/.claude/doc.md", ci2IdenticalDocument)

	_, err := machine.ingest(t, cli.IngestArgs{Auto: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.indexedSources(t)).To(Equal([]string{"/r/doc.md"}), "sanity: the preferred copy is indexed")
	g.Expect(machine.manifest(t)["/r/.claude/doc.md"]["duplicate_of"]).To(Equal("/r/doc.md"),
		"sanity: the other copy is recorded as its copy")

	g.Expect(machine.tree.remove("/r/doc.md")).To(Succeed())

	_, err = machine.ingest(t, cli.IngestArgs{Auto: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.recordsOf(t, "/r/.claude/doc.md")).NotTo(BeEmpty(), "the remaining copy is indexed")
	g.Expect(machine.manifest(t)["/r/.claude/doc.md"]).NotTo(HaveKey("duplicate_of"), "it is no longer a copy")
}

// Test792a5n1a_RunIngest_AVaultNoteCopyOutsideTheVaultIsSkipped verifies 792a5n1a (A vault-note copy outside the vault
// is skipped): of a swept directory outside the vault holding 1.2026-10-09.x.md with 1.2026-10-09.x.vec.json beside it
// and plain.md with no sidecar, ingest --sweep indexes plain.md and not 1.2026-10-09.x.md.
func Test792a5n1a_RunIngest_AVaultNoteCopyOutsideTheVaultIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.put("/swept/1.2026-10-09.x.md", ci2IdenticalDocument)
	machine.tree.put("/swept/1.2026-10-09.x.vec.json", `{"schema_version":1}`)
	machine.tree.put("/swept/plain.md", "## Plain\nA plain document with no sidecar, long enough to be a chunk.\n")

	_, err := machine.ingest(t, cli.IngestArgs{Sweep: []string{"/swept"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.indexedSources(t)).To(Equal([]string{"/swept/plain.md"}))
}

// Test792a5o1a_RunIngestAndRunPrune_AtTheSameTimeBothTakeEffect verifies 792a5o1a (An ingest and a prune at the same
// time both take effect): of a chunk index that records a source whose file is gone, ingest --transcript new.jsonl and
// prune started at the same time leave the manifest recording new.jsonl and not the gone source. Each run reads the
// manifest and then waits until the other run has either read it too or been held up by the manifest lock, so
// without the lock both read the old manifest and the later write overwrites the earlier one's change.
func Test792a5o1a_RunIngestAndRunPrune_AtTheSameTimeBothTakeEffect(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.putBytes("/work/gone.jsonl", ci2SessionRows(1, "a source that is deleted"))
	machine.tree.putBytes("/work/new.jsonl", ci2SessionRows(1, "a source that is added"))

	_, err := machine.ingest(t, cli.IngestArgs{Transcripts: []string{"/work/gone.jsonl"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.tree.remove("/work/gone.jsonl")).To(Succeed())
	g.Expect(machine.manifest(t)).To(HaveKey("/work/gone.jsonl"), "sanity: the index records the gone source")

	gate := &ci2ManifestGate{bothRead: make(chan struct{}), lockContended: make(chan struct{})}
	machine.tree.afterRead = gate.afterRead
	machine.locker.onContention = gate.contended

	errs := make(chan error, 2)

	var running sync.WaitGroup

	running.Go(func() {
		_, ingestErr := machine.ingest(t, cli.IngestArgs{Transcripts: []string{"/work/new.jsonl"}})
		errs <- ingestErr
	})
	running.Go(func() {
		_, pruneErr := machine.prune(t, cli.PruneArgs{})
		errs <- pruneErr
	})

	running.Wait()
	close(errs)

	for runErr := range errs {
		g.Expect(runErr).NotTo(HaveOccurred())
	}

	recorded := machine.manifest(t)

	g.Expect(recorded).To(HaveKey("/work/new.jsonl"), "the ingest's change is kept")
	g.Expect(recorded).NotTo(HaveKey("/work/gone.jsonl"), "the prune's change is kept")
}

// Test792a5p1a_RunPrune_ADeletedSourceIsDetachedAndStaysSearchable verifies 792a5p1a (A deleted source is detached and
// stays searchable): of a transcript that was ingested and then deleted, prune prints "prune: detached 1 source(s) —
// embedded chunks preserved (still searchable)", and a query with a phrase matching it returns its chunks with
// nothing re-indexed.
func Test792a5p1a_RunPrune_ADeletedSourceIsDetachedAndStaysSearchable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.putBytes("/work/old.jsonl", ci2SessionRows(1, "the manifest lock"))

	_, err := machine.ingest(t, cli.IngestArgs{Transcripts: []string{"/work/old.jsonl"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.tree.remove("/work/old.jsonl")).To(Succeed())

	indexBefore := machine.tree.snapshot(ci2IndexPath("/work/old.jsonl"))
	embedsBefore := machine.embedder.calls.Load()

	out, err := machine.prune(t, cli.PruneArgs{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("prune: detached 1 source(s) — embedded chunks preserved (still searchable)\n"))

	payload, err := machine.query(t, "the manifest lock")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed struct {
		Items []struct {
			Path string `yaml:"path"`
			Kind string `yaml:"kind"`
		} `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal([]byte(payload), &parsed)).To(Succeed())

	chunkPaths := make([]string, 0, len(parsed.Items))

	for _, item := range parsed.Items {
		if item.Kind == "chunk" {
			chunkPaths = append(chunkPaths, item.Path)
		}
	}

	g.Expect(chunkPaths).To(ContainElement(HavePrefix("/work/old.jsonl#")), "the query returns the deleted source's chunk")
	g.Expect(machine.tree.snapshot(ci2IndexPath("/work/old.jsonl"))).To(Equal(indexBefore), "the index file is untouched")
	g.Expect(machine.embedder.calls.Load()-embedsBefore).To(BeEquivalentTo(1),
		"only the query phrase is embedded: nothing is re-indexed")
}

// Test792a5p1b_RunPrune_ASourceThatStillExistsIsKept verifies 792a5p1b (A source that still exists is kept): of an
// ingested source whose file still exists, prune prints "prune: no dead sources" and the source is still recorded.
func Test792a5p1b_RunPrune_ASourceThatStillExistsIsKept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.putBytes("/work/live.jsonl", ci2SessionRows(1, "the manifest lock"))

	_, err := machine.ingest(t, cli.IngestArgs{Transcripts: []string{"/work/live.jsonl"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out, err := machine.prune(t, cli.PruneArgs{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("prune: no dead sources\n"))
	g.Expect(machine.manifest(t)).To(HaveKey("/work/live.jsonl"))
}

// Test792a5p1c_RunPrune_AnEmptyIndexHasNothingToPrune verifies 792a5p1c (An empty index has nothing to prune): of a
// chunk index directory with nothing ingested, prune prints "prune: no manifest, nothing to prune" and succeeds.
func Test792a5p1c_RunPrune_AnEmptyIndexHasNothingToPrune(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	g.Expect(machine.tree.mkdirAll(ci2ChunksDir)).To(Succeed())

	out, err := machine.prune(t, cli.PruneArgs{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("prune: no manifest, nothing to prune\n"))
}

// Test792a5p1d_RunPrune_ADryRunChangesNothing verifies 792a5p1d (A dry run changes nothing): of an ingested source that
// was then deleted, prune --dry-run prints "[dry-run] prune: detached 1 source(s) — embedded chunks preserved (still
// searchable)" and the source is still recorded.
func Test792a5p1d_RunPrune_ADryRunChangesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.putBytes("/work/old.jsonl", ci2SessionRows(1, "the manifest lock"))

	_, err := machine.ingest(t, cli.IngestArgs{Transcripts: []string{"/work/old.jsonl"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(machine.tree.remove("/work/old.jsonl")).To(Succeed())

	manifestBefore := machine.tree.snapshot(ci2ChunksDir + "/manifest.json")

	out, err := machine.prune(t, cli.PruneArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("[dry-run] prune: detached 1 source(s) — embedded chunks preserved (still searchable)\n"))
	g.Expect(machine.tree.snapshot(ci2ChunksDir+"/manifest.json")).To(Equal(manifestBefore), "the manifest is untouched")
	g.Expect(machine.manifest(t)).To(HaveKey("/work/old.jsonl"), "the source is still recorded")
}

// Test792a5q1a_RunPruneEmpty_EmptyIndexFilesAreRemoved verifies 792a5q1a (Empty index files are removed): of a chunk
// index with three index files, one of them empty, prune --empty removes the empty file, keeps the other two and
// prints "prune: removed 1 empty chunk-index file(s) of 3 scanned".
func Test792a5q1a_RunPruneEmpty_EmptyIndexFilesAreRemoved(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	machine.tree.putBytes("/work/one.jsonl", ci2SessionRows(1, "the first source"))
	machine.tree.putBytes("/work/two.jsonl", ci2SessionRows(1, "the second source"))

	_, err := machine.ingest(t, cli.IngestArgs{Transcripts: []string{"/work/one.jsonl", "/work/two.jsonl"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	emptyName := "empty-00000000.jsonl"
	machine.tree.putBytes(ci2ChunksDir+"/"+emptyName, nil)

	kept := slices.DeleteFunc(machine.tree.indexFileNames(), func(name string) bool { return name == emptyName })
	g.Expect(kept).To(HaveLen(2), "sanity: two non-empty index files")

	out, err := machine.prune(t, cli.PruneArgs{Empty: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("prune: removed 1 empty chunk-index file(s) of 3 scanned\n"))
	g.Expect(machine.tree.indexFileNames()).To(Equal(kept), "only the empty file is gone")
}

// Test792a5q1b_RunPruneEmpty_ADryRunRemovesNothing verifies 792a5q1b (A dry run removes nothing): of a chunk index with
// one empty index file, prune --empty --dry-run leaves the file and prints "[dry-run] prune: removed 1 empty
// chunk-index file(s) of 1 scanned".
func Test792a5q1b_RunPruneEmpty_ADryRunRemovesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci2NewMachine("/work")
	emptyName := "empty-00000000.jsonl"
	machine.tree.putBytes(ci2ChunksDir+"/"+emptyName, nil)

	out, err := machine.prune(t, cli.PruneArgs{Empty: true, DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("[dry-run] prune: removed 1 empty chunk-index file(s) of 1 scanned\n"))
	g.Expect(machine.tree.indexFileNames()).To(Equal([]string{emptyName}), "the file remains")
}

// unexported constants.
const (
	ci2ChunksDir = "/state/chunks"
	// ci2IdenticalDocument is a Markdown document long enough to be a chunk, shared byte for byte by the copies of the
	// dedup scenarios.
	ci2IdenticalDocument = "## Notes\nIdentical content reachable from several swept paths, long enough to be a chunk.\n"
	ci2Vault             = "/state/vault"
	// ci2WaitBound bounds how long a gated run waits for its peer; it is a failure bound, not a synchronisation delay.
	ci2WaitBound = 10 * time.Second
)

// ci2Embedder is a counting embedder: a two-dimensional vector derived from the text length.
type ci2Embedder struct {
	calls atomic.Int64
}

// Dims returns the vector dimension.
func (e *ci2Embedder) Dims() int { return 2 }

// Embed counts the call and returns a vector derived from the text length.
func (e *ci2Embedder) Embed(_ context.Context, text string) ([]float32, error) {
	e.calls.Add(1)

	return []float32{float32(len(text)), 1}, nil
}

// ModelID names the embedder.
func (e *ci2Embedder) ModelID() string { return "ci2@2" }

// ci2File is one file of the in-memory tree: its bytes and its modification time.
type ci2File struct {
	data  []byte
	mtime int64
}

// ci2Locker is an in-memory file locker: one mutex per path, with a hook called when a lock request finds the path
// already held.
type ci2Locker struct {
	mu           sync.Mutex
	held         map[string]*sync.Mutex
	onContention func()
}

// Lock takes the mutex of path, calling onContention first when another holder has it.
func (l *ci2Locker) Lock(path string) (func() error, error) {
	l.mu.Lock()

	pathLock, found := l.held[path]
	if !found {
		pathLock = &sync.Mutex{}
		l.held[path] = pathLock
	}

	l.mu.Unlock()

	if !pathLock.TryLock() {
		if l.onContention != nil {
			l.onContention()
		}

		pathLock.Lock()
	}

	return func() error {
		pathLock.Unlock()

		return nil
	}, nil
}

// ci2Machine is one in-memory machine: a file tree, a lock, an embedder and a working directory.
type ci2Machine struct {
	tree     *ci2Tree
	locker   *ci2Locker
	embedder *ci2Embedder
	cwd      string
}

// chunkIDs returns the source#anchor ID of every record of the chunk index.
func (m *ci2Machine) chunkIDs(t *testing.T) []string {
	t.Helper()

	records := m.records(t)
	ids := make([]string, 0, len(records))

	for _, record := range records {
		ids = append(ids, record.Source+"#"+record.Anchor)
	}

	return ids
}

// deps composes the CLI-edge Deps over the in-memory tree.
func (m *ci2Machine) deps() cli.Deps {
	return cli.Deps{
		FS:          m.tree.edgeFS(),
		Lock:        fakeLocker{lock: m.locker.Lock},
		Embed:       m.embedder,
		Now:         func() time.Time { return time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC) },
		Getwd:       func() (string, error) { return m.cwd, nil },
		Getenv:      func(string) string { return "" },
		UserHomeDir: func() (string, error) { return "/home/user", nil },
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	}
}

// indexedSources returns the sorted sources that have at least one record in the chunk index.
func (m *ci2Machine) indexedSources(t *testing.T) []string {
	t.Helper()

	var sources []string

	for _, record := range m.records(t) {
		if !slices.Contains(sources, record.Source) {
			sources = append(sources, record.Source)
		}
	}

	slices.Sort(sources)

	return sources
}

// ingest runs `engram ingest` with the chunks directory and the vault of the machine and returns its stdout.
func (m *ci2Machine) ingest(t *testing.T, args cli.IngestArgs) (string, error) {
	t.Helper()

	args.ChunksDir = ci2ChunksDir
	args.Vault = ci2Vault

	var out bytes.Buffer

	err := cli.RunIngest(t.Context(), args, cli.ExportNewIngestDeps(m.deps(), m.embedder), &out)
	if err != nil {
		return out.String(), fmt.Errorf("ingest: %w", err)
	}

	return out.String(), nil
}

// manifest decodes the manifest of the chunk index into a map of source to its JSON fields.
func (m *ci2Machine) manifest(t *testing.T) map[string]map[string]any {
	t.Helper()

	data, found := m.tree.get(ci2ChunksDir + "/manifest.json")
	if !found {
		t.Fatalf("the chunk index has no manifest")

		return nil
	}

	decoded := map[string]map[string]any{}

	err := json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("decoding the manifest: %v", err)

		return nil
	}

	return decoded
}

// prune runs `engram prune` with the chunks directory of the machine and returns its stdout.
func (m *ci2Machine) prune(t *testing.T, args cli.PruneArgs) (string, error) {
	t.Helper()

	args.ChunksDir = ci2ChunksDir

	var out bytes.Buffer

	err := cli.RunPrune(t.Context(), args, cli.ExportNewPruneDeps(m.deps()), &out)
	if err != nil {
		return out.String(), fmt.Errorf("prune: %w", err)
	}

	return out.String(), nil
}

// query runs `engram query` for one phrase over an empty vault and the chunk index of the machine.
func (m *ci2Machine) query(t *testing.T, phrase string) (string, error) {
	t.Helper()

	const limit = 10

	var out bytes.Buffer

	args := cli.QueryArgs{Phrases: []string{phrase}, VaultPath: ci2Vault, ChunksDir: ci2ChunksDir, Limit: limit}

	err := cli.RunQuery(t.Context(), args, cli.ExportNewQueryDeps(m.deps()), &out)
	if err != nil {
		return out.String(), fmt.Errorf("query: %w", err)
	}

	return out.String(), nil
}

// records decodes every index file of the chunk index, in file name order.
func (m *ci2Machine) records(t *testing.T) []chunk.Record {
	t.Helper()

	var all []chunk.Record

	for _, name := range m.tree.indexFileNames() {
		data, _ := m.tree.get(ci2ChunksDir + "/" + name)

		decoded, err := chunk.DecodeRecords(data)
		if err != nil {
			t.Fatalf("decoding index file %s: %v", name, err)

			return nil
		}

		all = append(all, decoded...)
	}

	return all
}

// recordsOf returns the records of the chunk index that came from one source, in index file order.
func (m *ci2Machine) recordsOf(t *testing.T, source string) []chunk.Record {
	t.Helper()

	var matching []chunk.Record

	for _, record := range m.records(t) {
		if record.Source == source {
			matching = append(matching, record)
		}
	}

	return matching
}

// ci2ManifestGate holds each run that has read the manifest until its peer has read it too or has been held up by the
// manifest lock.
type ci2ManifestGate struct {
	reads         atomic.Int32
	bothRead      chan struct{}
	lockContended chan struct{}
	bothOnce      sync.Once
	contendedOnce sync.Once
}

// afterRead is called after each read of a path; a manifest read holds its run until the gate opens.
func (g *ci2ManifestGate) afterRead(path string) {
	if path != ci2ChunksDir+"/manifest.json" {
		return
	}

	const peers = 2

	if g.reads.Add(1) >= peers {
		g.bothOnce.Do(func() { close(g.bothRead) })
	}

	select {
	case <-g.bothRead:
	case <-g.lockContended:
	case <-time.After(ci2WaitBound):
	}
}

// contended is called when a lock request finds the manifest lock held.
func (g *ci2ManifestGate) contended() {
	g.contendedOnce.Do(func() { close(g.lockContended) })
}

// ci2Tree is an in-memory file tree safe for concurrent use. Directories exist when a file lies under them or when
// they were made.
type ci2Tree struct {
	mu        sync.Mutex
	files     map[string]ci2File
	dirs      map[string]bool
	reads     map[string]int
	clock     int64
	afterRead func(path string)
}

// children returns the names directly under dir, each flagged with whether it is a directory.
func (tr *ci2Tree) children(dir string) map[string]bool {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	prefix := strings.TrimSuffix(dir, "/") + "/"
	names := map[string]bool{}

	for path := range tr.files {
		if rest, under := strings.CutPrefix(path, prefix); under {
			name, _, nested := strings.Cut(rest, "/")
			names[name] = names[name] || nested
		}
	}

	for path := range tr.dirs {
		if rest, under := strings.CutPrefix(path, prefix); under && rest != "" {
			name, _, _ := strings.Cut(rest, "/")
			names[name] = true
		}
	}

	return names
}

// edgeFS returns the tree as the filesystem edge of a cli.Deps; only what the chunk-index commands use is set.
func (tr *ci2Tree) edgeFS() cli.EdgeFS {
	return fakeEdgeFS{
		readFile:        tr.readFile,
		writeFileAtomic: func(path string, data []byte, _ fs.FileMode) error { tr.putBytes(path, data); return nil },
		mkdirAll:        func(path string, _ fs.FileMode) error { return tr.mkdirAll(path) },
		stat:            tr.stat,
		readDir:         tr.readDir,
		remove:          tr.remove,
		walkDir:         tr.walkDir,
	}
}

// get returns a copy of the bytes of the file at path.
func (tr *ci2Tree) get(path string) ([]byte, bool) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	file, found := tr.files[path]
	if !found {
		return nil, false
	}

	return bytes.Clone(file.data), true
}

// has reports whether a file exists at path.
func (tr *ci2Tree) has(path string) bool {
	_, found := tr.get(path)

	return found
}

// indexFileNames returns the sorted names of the index files (.jsonl) in the chunks directory.
func (tr *ci2Tree) indexFileNames() []string {
	var names []string

	for name, isDir := range tr.children(ci2ChunksDir) {
		if !isDir && filepath.Ext(name) == ".jsonl" {
			names = append(names, name)
		}
	}

	slices.Sort(names)

	return names
}

// isDir reports whether path is a directory of the tree.
func (tr *ci2Tree) isDir(path string) bool {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if tr.dirs[path] {
		return true
	}

	prefix := strings.TrimSuffix(path, "/") + "/"

	for file := range tr.files {
		if strings.HasPrefix(file, prefix) {
			return true
		}
	}

	return false
}

// mkdirAll makes a directory.
func (tr *ci2Tree) mkdirAll(path string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.dirs[path] = true

	return nil
}

// put writes a text file with a new modification time.
func (tr *ci2Tree) put(path, text string) { tr.putBytes(path, []byte(text)) }

// putBytes writes a file with a new modification time.
func (tr *ci2Tree) putBytes(path string, data []byte) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.clock++
	tr.files[path] = ci2File{data: bytes.Clone(data), mtime: tr.clock}
}

// readCount returns how many times the file at path was read.
func (tr *ci2Tree) readCount(path string) int {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	return tr.reads[path]
}

// readDir lists the entries under dir, sorted by name.
func (tr *ci2Tree) readDir(dir string) ([]fs.DirEntry, error) {
	if !tr.isDir(dir) {
		return nil, fmt.Errorf("reading %s: %w", dir, fs.ErrNotExist)
	}

	children := tr.children(dir)
	names := make([]string, 0, len(children))

	for name := range children {
		names = append(names, name)
	}

	slices.Sort(names)

	entries := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, fakeDirEntry{name: name, dir: children[name]})
	}

	return entries, nil
}

// readFile returns the bytes of the file at path, counting the read and then calling the afterRead hook.
func (tr *ci2Tree) readFile(path string) ([]byte, error) {
	tr.mu.Lock()
	file, found := tr.files[path]
	tr.reads[path]++
	hook := tr.afterRead
	tr.mu.Unlock()

	if !found {
		return nil, fmt.Errorf("reading %s: %w", path, fs.ErrNotExist)
	}

	if hook != nil {
		hook(path)
	}

	return bytes.Clone(file.data), nil
}

// remove deletes the file at path.
func (tr *ci2Tree) remove(path string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, found := tr.files[path]; !found {
		return fmt.Errorf("removing %s: %w", path, fs.ErrNotExist)
	}

	delete(tr.files, path)

	return nil
}

// snapshot returns a copy of the bytes of the file at path, or nil when it does not exist.
func (tr *ci2Tree) snapshot(path string) []byte {
	data, _ := tr.get(path)

	return data
}

// stat reports a file with its size and modification time, or a directory.
func (tr *ci2Tree) stat(path string) (fs.FileInfo, error) {
	tr.mu.Lock()
	file, found := tr.files[path]
	tr.mu.Unlock()

	if found {
		return fakeFileInfo{mtime: time.Unix(0, file.mtime), size: int64(len(file.data))}, nil
	}

	if tr.isDir(path) {
		return fakeFileInfo{dir: true}, nil
	}

	return nil, fmt.Errorf("stat %s: %w", path, fs.ErrNotExist)
}

// walk visits path and, when it is a directory, everything under it in name order, honouring fs.SkipDir.
func (tr *ci2Tree) walk(path string, entry fs.DirEntry, visit fs.WalkDirFunc) error {
	err := visit(path, entry, nil)
	if err != nil {
		if errors.Is(err, fs.SkipDir) && entry.IsDir() {
			return nil
		}

		return err
	}

	if !entry.IsDir() {
		return nil
	}

	children := tr.children(path)
	names := make([]string, 0, len(children))

	for name := range children {
		names = append(names, name)
	}

	slices.Sort(names)

	for _, name := range names {
		err = tr.walk(filepath.Join(path, name), fakeDirEntry{name: name, dir: children[name]}, visit)
		if err != nil {
			return err
		}
	}

	return nil
}

// walkDir walks the tree from root like filepath.WalkDir.
func (tr *ci2Tree) walkDir(root string, visit fs.WalkDirFunc) error {
	return tr.walk(root, fakeDirEntry{name: filepath.Base(root), dir: true}, visit)
}

// ci2CountContaining counts the texts that contain the fragment.
func ci2CountContaining(texts []string, fragment string) int {
	count := 0

	for _, text := range texts {
		if strings.Contains(text, fragment) {
			count++
		}
	}

	return count
}

// ci2IndexPath is the path of the index file of a source.
func ci2IndexPath(source string) string {
	return ci2ChunksDir + "/" + cli.ExportIndexFileName(source)
}

// ci2NewMachine returns an empty machine whose working directory is cwd.
func ci2NewMachine(cwd string) *ci2Machine {
	return &ci2Machine{
		tree:     &ci2Tree{files: map[string]ci2File{}, dirs: map[string]bool{}, reads: map[string]int{}},
		locker:   &ci2Locker{held: map[string]*sync.Mutex{}},
		embedder: &ci2Embedder{},
		cwd:      cwd,
	}
}

// ci2SessionRows returns the JSONL of a session of the given number of turns: a short user row and a short assistant
// row each, a minute apart.
func ci2SessionRows(turns int, topic string) []byte {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type row struct {
		Type      string  `json:"type"`
		Timestamp string  `json:"timestamp"`
		Message   message `json:"message"`
	}

	start := time.Date(2026, time.May, 10, 14, 0, 0, 0, time.UTC)

	var out []byte

	for turn := 1; turn <= turns; turn++ {
		userText := fmt.Sprintf("question %d about %s", turn, topic)
		assistantText := fmt.Sprintf("answer %d: take the lock first", turn)

		pair := []row{
			{Type: "user", Message: message{Role: "user", Content: userText}},
			{Type: "assistant", Message: message{Role: "assistant", Content: assistantText}},
		}

		for position := range pair {
			stamp := start.Add(time.Duration(2*(turn-1)+position) * time.Minute)
			pair[position].Timestamp = stamp.Format(time.RFC3339)

			encoded, err := json.Marshal(pair[position])
			if err != nil {
				panic(fmt.Sprintf("encoding a session row: %v", err))
			}

			out = append(out, encoded...)
			out = append(out, '\n')
		}
	}

	return out
}

// ci2TwoSectionDocument returns a Markdown document of sections A and B, each long enough to be a chunk.
func ci2TwoSectionDocument(sectionA, sectionB string) string {
	return "## A\n" + sectionA + " that is long enough to be kept as a chunk of its own.\n\n" +
		"## B\n" + sectionB + " that is long enough to be kept as a chunk of its own.\n"
}
