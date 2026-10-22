package cli_test

// Scenario tests for the Chunk Indexing product capability, part 1 (861c2zzm): where the index lives (792a5a),
// the explicitly named sources (792a5b), swept directories (792a5c), the automatic sweep roots (792a5d), the
// repository sweep configuration (792a5e), throwaway workspaces (792a5f) and transcript content (792a5g).
// Spec: docs/specs/product/chunk-indexing.md. Every scenario runs `engram ingest` (and `engram show-chunk`) through
// cli.Targets, the command path, over an in-memory file system (ci1FS) with the environment, the home directory,
// the working directory, the clock, the lock and the embedder injected through cli.Deps, so no test reads the real
// home, environment, vault or chunk index and none writes to disk (no t.TempDir). The vault is pinned with --vault
// to a directory the file system already holds, so the first-use vault creation never runs. The helpers carry the
// ci1 prefix. The scenarios of 792a5h and later are in chunk_indexing_part2_scenarios_test.go.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// Test792a5a1a_Ingest_TheFlagWinsOverTheEnvironment verifies 792a5a1a (The flag wins over the environment): with
// ENGRAM_CHUNKS_DIR set to /env/chunks, `engram ingest --chunks-dir /flag/chunks --markdown notes.md` indexes
// notes.md under /flag/chunks, and nothing is written under /env/chunks.
func Test792a5a1a_Ingest_TheFlagWinsOverTheEnvironment(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.env["ENGRAM_CHUNKS_DIR"] = "/env/chunks"
	machine.fsys.put(ci1Notes, ci1Markdown("notes"))

	out := machine.ingest(t, "--chunks-dir", "/flag/chunks", "--markdown", ci1Notes)

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, "/flag/chunks")).To(Equal([]string{ci1Notes}), "indexed under the flag")
	g.Expect(machine.fsys.under("/env")).To(BeEmpty(), "nothing is written under the environment directory")
}

// Test792a5a1b_Ingest_TheEnvironmentVariableIsUsedWithoutAFlag verifies 792a5a1b (The environment variable is used
// without a flag): with ENGRAM_CHUNKS_DIR set to /env/chunks, `engram ingest --markdown notes.md` indexes notes.md
// under /env/chunks, although XDG_DATA_HOME and the home directory are set as well.
func Test792a5a1b_Ingest_TheEnvironmentVariableIsUsedWithoutAFlag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.env["ENGRAM_CHUNKS_DIR"] = "/env/chunks"
	machine.env["XDG_DATA_HOME"] = "/data"
	machine.fsys.put(ci1Notes, ci1Markdown("notes"))

	out := machine.ingest(t, "--markdown", ci1Notes)

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, "/env/chunks")).To(Equal([]string{ci1Notes}), "indexed under the environment")
	g.Expect(machine.fsys.under("/data")).To(BeEmpty(), "nothing is written under the XDG directory")
	g.Expect(machine.fsys.under(ci1Home)).To(BeEmpty(), "nothing is written under the home directory")
}

// Test792a5a1c_Ingest_TheXDGDataDirectoryIsTheDefault verifies 792a5a1c (The XDG data directory is the default):
// with neither --chunks-dir nor ENGRAM_CHUNKS_DIR set and XDG_DATA_HOME at /data, `engram ingest --markdown
// notes.md` indexes notes.md under /data/engram/chunks.
func Test792a5a1c_Ingest_TheXDGDataDirectoryIsTheDefault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.env["XDG_DATA_HOME"] = "/data"
	machine.fsys.put(ci1Notes, ci1Markdown("notes"))

	out := machine.ingest(t, "--markdown", ci1Notes)

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, "/data/engram/chunks")).To(Equal([]string{ci1Notes}))
	g.Expect(machine.fsys.under(ci1Home)).To(BeEmpty(), "nothing is written under the home directory")
}

// Test792a5a1d_Ingest_TheHomeDirectoryIsTheLastFallback verifies 792a5a1d (The home directory is the last
// fallback): with none of --chunks-dir, ENGRAM_CHUNKS_DIR and XDG_DATA_HOME set, `engram ingest --markdown
// notes.md` indexes notes.md under ~/.local/share/engram/chunks.
func Test792a5a1d_Ingest_TheHomeDirectoryIsTheLastFallback(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.fsys.put(ci1Notes, ci1Markdown("notes"))

	out := machine.ingest(t, "--markdown", ci1Notes)

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Home+"/.local/share/engram/chunks")).To(Equal([]string{ci1Notes}))
}

// Test792a5b1a_Ingest_ANamedTranscriptAndDocumentAreIndexed verifies 792a5b1a (A named transcript and document are
// indexed): `engram ingest --transcript s.jsonl --markdown d.md` leaves chunks of both files in the chunk index,
// and stdout has one "ingest <path>: N chunks" line for each, N being the number of chunks the file holds.
func Test792a5b1a_Ingest_ANamedTranscriptAndDocumentAreIndexed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		session  = "/work/s.jsonl"
		document = "/work/d.md"
	)

	machine := ci1NewMachine()
	machine.fsys.put(session, ci1Session("session"))
	machine.fsys.put(document, ci1Markdown("first")+"\n"+ci1Markdown("second"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--transcript", session, "--markdown", document)

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{document, session}))

	sessionChunks := len(machine.records(t, ci1Chunks, session))
	documentChunks := len(machine.records(t, ci1Chunks, document))

	g.Expect(sessionChunks).To(BeNumerically(">", 0))
	g.Expect(documentChunks).To(BeNumerically(">", sessionChunks), "the two files hold different numbers of chunks")
	g.Expect(ci1IngestLines(out.stdout)).To(ConsistOf(
		ci1IngestLine{Path: session, Chunks: sessionChunks},
		ci1IngestLine{Path: document, Chunks: documentChunks},
	), "one ingest line for each file")
}

// Test792a5b1b_Ingest_APiSessionsDirectoryIsIndexedWithoutItsScratch verifies 792a5b1b (A Pi sessions directory is
// indexed without its scratch): of a directory p holding a.jsonl, jobs/b.jsonl, projects/c.jsonl and
// .hidden/d.jsonl, `engram ingest --pi-sessions p` indexes a.jsonl and none of the other three files.
func Test792a5b1b_Ingest_APiSessionsDirectoryIsIndexedWithoutItsScratch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const piDir = "/work/p"

	machine := ci1NewMachine()
	machine.fsys.put(piDir+"/a.jsonl", ci1Session("a"))
	machine.fsys.put(piDir+"/jobs/b.jsonl", ci1Session("b"))
	machine.fsys.put(piDir+"/projects/c.jsonl", ci1Session("c"))
	machine.fsys.put(piDir+"/.hidden/d.jsonl", ci1Session("d"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--pi-sessions", piDir)

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{piDir + "/a.jsonl"}))
}

// Test792a5b1c_Ingest_APiSessionsValueThatIsNotADirectoryIsIgnored verifies 792a5b1c (A Pi sessions value that is
// not a directory is ignored): `engram ingest --pi-sessions f`, f being a regular file (with a source extension,
// so that walking it as a root would index it), succeeds and indexes nothing.
func Test792a5b1c_Ingest_APiSessionsValueThatIsNotADirectoryIsIgnored(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const regular = "/work/f.jsonl"

	machine := ci1NewMachine()
	machine.fsys.put(regular, ci1Session("regular"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--pi-sessions", regular)

	g.Expect(out.stderr).To(BeEmpty(), "the command succeeds")
	g.Expect(out.exits).To(BeEmpty())
	g.Expect(out.stdout).To(BeEmpty(), "nothing is ingested")
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(BeEmpty())
	g.Expect(machine.fsys.files()).NotTo(ContainElement(HavePrefix(ci1Chunks+"/")), "no index file is written")
}

// Test792a5b1d_Ingest_AnUnreadableNamedFileFailsTheCommand verifies the outline 792a5b1d (An unreadable named file
// fails the command): with no file at missing.md (named by --markdown) or at missing.jsonl (named by
// --transcript), `engram ingest` fails with an error naming that file. The file system reports the absence
// without the path, so the name in the error is the command's.
func Test792a5b1d_Ingest_AnUnreadableNamedFileFailsTheCommand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		flag string
		file string
	}{
		{flag: "--markdown", file: "/work/missing.md"},
		{flag: "--transcript", file: "/work/missing.jsonl"},
	}

	for _, tc := range cases {
		t.Run(tc.flag, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := ci1NewMachine()

			out := machine.ingest(t, "--chunks-dir", ci1Chunks, tc.flag, tc.file)

			g.Expect(out.exits).To(Equal([]int{1}), "the command fails")
			g.Expect(out.stderr).To(ContainSubstring(tc.file), "the error names the file")
			g.Expect(machine.indexedSources(t, ci1Chunks)).To(BeEmpty())
		})
	}
}

// Test792a5c1a_Ingest_ASweptDirectorysDocumentsAndTranscriptsAreIndexed verifies 792a5c1a (A swept directory's
// documents and transcripts are indexed): of a directory r holding a.md, sub/b.jsonl, node_modules/c.md, .git/d.md
// and notes.txt, `engram ingest --sweep r` indexes a.md and sub/b.jsonl and none of the others. The directory
// holds as well a document in each other directory the requirement skips by name and in a hidden directory.
func Test792a5c1a_Ingest_ASweptDirectorysDocumentsAndTranscriptsAreIndexed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const root = "/work/r"

	machine := ci1NewMachine()
	machine.fsys.put(root+"/a.md", ci1Markdown("alpha"))
	machine.fsys.put(root+"/sub/b.jsonl", ci1Session("beta"))
	machine.fsys.put(root+"/notes.txt", ci1Markdown("text file"))

	skipped := []string{
		"node_modules", "vendor", ".git", ".hg", ".jj", "dist", "build", "target", ".venv", "venv", "__pycache__",
		".next", ".cache", ".idea", ".hidden",
	}
	for _, name := range skipped {
		machine.fsys.put(root+"/"+name+"/c.md", ci1Markdown("inside "+name))
	}

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--sweep", root)

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{root + "/a.md", root + "/sub/b.jsonl"}))
}

// Test792a5c1b_Ingest_ASweptFileThatVanishesIsSkipped verifies 792a5c1b (A swept file that vanishes is skipped):
// of a swept directory whose file gone.md cannot be read when the sweep reaches it, `engram ingest --sweep <dir>`
// prints a "skip <dir>/gone.md:" line on stdout, indexes the directory's other files and succeeds.
func Test792a5c1b_Ingest_ASweptFileThatVanishesIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const dir = "/work/swept"

	machine := ci1NewMachine()
	machine.fsys.put(dir+"/gone.md", ci1Markdown("gone"))
	machine.fsys.put(dir+"/live.md", ci1Markdown("live"))
	machine.fsys.put(dir+"/other.jsonl", ci1Session("other"))
	machine.fsys.unreadable[dir+"/gone.md"] = errors.New("the file vanished after the walk")

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--sweep", dir)

	g.Expect(out.stderr).To(BeEmpty(), "the command succeeds")
	g.Expect(out.exits).To(BeEmpty())

	skips := ci1LinesWithPrefix(out.stdout, "skip ")

	g.Expect(skips).To(HaveLen(1))

	if len(skips) != 1 {
		return
	}

	g.Expect(skips[0]).To(HavePrefix("skip " + dir + "/gone.md:"))
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{dir + "/live.md", dir + "/other.jsonl"}))
}

// Test792a5c1c_Ingest_TheChunkIndexDirectoryIsNeverIndexed verifies 792a5c1c (The chunk index directory is never
// indexed): of a chunk index directory inside the swept directory r, which holds a stray document and, on the
// second run, the index files of the first, `engram ingest --sweep r --chunks-dir r/chunks` indexes no file under
// r/chunks, on either run.
func Test792a5c1c_Ingest_TheChunkIndexDirectoryIsNeverIndexed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		root   = "/work/r"
		chunks = "/work/r/chunks"
	)

	machine := ci1NewMachine()
	machine.fsys.put(root+"/note.md", ci1Markdown("note"))
	machine.fsys.put(chunks+"/stray.md", ci1Markdown("stray"))

	for run := range 2 {
		out := machine.ingest(t, "--sweep", root, "--chunks-dir", chunks)

		g.Expect(out.stderr).To(BeEmpty(), "run %d", run)
		g.Expect(machine.indexedSources(t, chunks)).To(Equal([]string{root + "/note.md"}), "run %d", run)
		g.Expect(machine.manifestSources(t, chunks)).To(Equal([]string{root + "/note.md"}), "run %d", run)
	}
}

// Test792a5d1a_Ingest_TheRepositoryItsAncestorsAndTheSessionLogsAreSwept verifies 792a5d1a (The repository, its
// ancestors and the session logs are swept): of a working directory /r/sub inside a git repository /r, a
// /r/.claude directory, a /r/sub/.pi directory and session logs under ~/.claude/projects, `engram ingest --auto`
// from /r/sub indexes the Markdown under /r, the files of /r/.claude and /r/sub/.pi, and the session transcripts.
func Test792a5d1a_Ingest_TheRepositoryItsAncestorsAndTheSessionLogsAreSwept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.chdir("/r/sub")
	machine.fsys.dir("/r/.git")
	machine.fsys.put("/r/readme.md", ci1Markdown("readme"))
	machine.fsys.put("/r/sub/guide.md", ci1Markdown("guide"))
	machine.fsys.put("/r/.claude/notes.md", ci1Markdown("claude notes"))
	machine.fsys.put("/r/sub/.pi/session.jsonl", ci1Session("pi"))
	machine.fsys.put(ci1Home+"/.claude/projects/-Users-u-repo/log.jsonl", ci1Session("log"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{
		"/home/u/.claude/projects/-Users-u-repo/log.jsonl",
		"/r/.claude/notes.md",
		"/r/readme.md",
		"/r/sub/.pi/session.jsonl",
		"/r/sub/guide.md",
	}))
}

// Test792a5d1b_Ingest_WithoutARepositoryMarkerTheWorkingDirectoryIsTheRoot verifies 792a5d1b (Without a
// repository marker the working directory is the root): from a working directory /w with no .git, .hg or .jj in it
// or any ancestor, `engram ingest --auto` indexes the Markdown under /w, and no Markdown outside it.
func Test792a5d1b_Ingest_WithoutARepositoryMarkerTheWorkingDirectoryIsTheRoot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.chdir("/w")
	machine.fsys.put("/w/readme.md", ci1Markdown("readme"))
	machine.fsys.put("/w/deep/guide.md", ci1Markdown("guide"))
	machine.fsys.put("/elsewhere/outside.md", ci1Markdown("outside"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{"/w/deep/guide.md", "/w/readme.md"}))
}

// Test792a5d1c_Ingest_HarnessScratchAndStateAreNotSwept verifies 792a5d1c (Harness scratch and state are not
// swept): of an ancestor .claude directory holding notes.md, jobs/snap.md, projects/p/s.jsonl and plugins/x.md,
// `engram ingest --auto` indexes notes.md and none of the others. The directory holds as well a document in each
// other subdirectory the requirement skips, and an ancestor .pi directory holds its jobs and projects.
func Test792a5d1c_Ingest_HarnessScratchAndStateAreNotSwept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.chdir("/w/sub")
	machine.fsys.put("/w/.claude/notes.md", ci1Markdown("notes"))
	machine.fsys.put("/w/.claude/projects/p/s.jsonl", ci1Session("project"))

	skipped := []string{
		"jobs", "plugins", "cache", "todos", "shell-snapshots", "file-history", "history", "ide", "statsig",
		"session-env", "debug", "worktrees",
	}
	for _, name := range skipped {
		machine.fsys.put("/w/.claude/"+name+"/x.md", ci1Markdown("claude "+name))
	}

	machine.fsys.put("/w/.pi/s.jsonl", ci1Session("pi"))
	machine.fsys.put("/w/.pi/jobs/x.jsonl", ci1Session("pi jobs"))
	machine.fsys.put("/w/.pi/projects/y.jsonl", ci1Session("pi projects"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{"/w/.claude/notes.md", "/w/.pi/s.jsonl"}))
}

// Test792a5d1d_Ingest_EngramTranscriptDirNamesTheSessionLogs verifies 792a5d1d (ENGRAM_TRANSCRIPT_DIR names the
// session logs): with ENGRAM_TRANSCRIPT_DIR set to /logs, which holds a.jsonl, `engram ingest --auto` indexes
// /logs/a.jsonl and no transcript under ~/.claude/projects.
func Test792a5d1d_Ingest_EngramTranscriptDirNamesTheSessionLogs(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.chdir("/w")
	machine.env["ENGRAM_TRANSCRIPT_DIR"] = "/logs"
	machine.fsys.put("/logs/a.jsonl", ci1Session("logs"))
	machine.fsys.put(ci1Home+"/.claude/projects/-Users-u-repo/s.jsonl", ci1Session("home"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{"/logs/a.jsonl"}))
}

// Test792a5e1a_Ingest_ARepositoryAddsARootAndTurnsOneOff verifies 792a5e1a (A repository adds a root and turns one
// off): of a repository whose .engram/sweep.json is {"session_logs": false, "extra_roots": ["/x"]}, `engram
// ingest --auto` inside it sweeps /x and the repository (a setting the file omits keeps its default) and indexes
// no session log.
func Test792a5e1a_Ingest_ARepositoryAddsARootAndTurnsOneOff(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.chdir("/r")
	machine.fsys.dir("/r/.git")
	machine.fsys.put("/r/.engram/sweep.json", `{"session_logs": false, "extra_roots": ["/x"]}`)
	machine.fsys.put("/r/readme.md", ci1Markdown("readme"))
	machine.fsys.put("/x/extra.md", ci1Markdown("extra"))
	machine.fsys.put(ci1Home+"/.claude/projects/-Users-u-repo/s.jsonl", ci1Session("home"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{"/r/readme.md", "/x/extra.md"}))
}

// Test792a5e1b_Ingest_AMalformedSweepFileIsRefused verifies 792a5e1b (A malformed sweep file is refused): of a
// repository whose .engram/sweep.json is not valid JSON, `engram ingest --auto` inside it fails with an error
// naming .engram/sweep.json, and indexes nothing.
func Test792a5e1b_Ingest_AMalformedSweepFileIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.chdir("/r")
	machine.fsys.dir("/r/.git")
	machine.fsys.put("/r/.engram/sweep.json", "{this is not json")
	machine.fsys.put("/r/readme.md", ci1Markdown("readme"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

	g.Expect(out.exits).To(Equal([]int{1}), "the command fails")
	g.Expect(out.stderr).To(ContainSubstring("/r/.engram/sweep.json"), "the error names the sweep file")
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(BeEmpty(), "nothing is indexed")
	g.Expect(machine.fsys.files()).NotTo(ContainElement(HavePrefix(ci1Chunks+"/")), "no index file is written")
}

// Test792a5f1a_Ingest_SessionsRecordedInAThrowawayDirectoryAreSkipped verifies 792a5f1a (Sessions recorded in a
// throwaway directory are skipped): of session logs under ~/.claude/projects/-private-tmp-eval-1/ and
// ~/.claude/projects/-Users-u-repo/, `engram ingest --auto` indexes the -Users-u-repo sessions and not the
// -private-tmp-eval-1 ones. The other three prefixes of the requirement are skipped the same way.
func Test792a5f1a_Ingest_SessionsRecordedInAThrowawayDirectoryAreSkipped(t *testing.T) {
	t.Parallel()

	for _, prefix := range []string{"-private-tmp-", "-tmp-", "-var-folders-", "-private-var-folders-"} {
		t.Run(prefix, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			projects := ci1Home + "/.claude/projects"

			machine := ci1NewMachine()
			machine.chdir("/w")
			machine.fsys.put(projects+"/"+prefix+"eval-1/s.jsonl", ci1Session("throwaway "+prefix))
			machine.fsys.put(projects+"/-Users-u-repo/s.jsonl", ci1Session("persistent"))

			out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

			g.Expect(out.stderr).To(BeEmpty())
			g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{projects + "/-Users-u-repo/s.jsonl"}))
		})
	}
}

// Test792a5f1b_Ingest_AThrowawayWorkingDirectoryIsNotSwept verifies 792a5f1b (A throwaway working directory is not
// swept): from a working directory /tmp/test-pool/run123 with no repository marker, `engram ingest --auto`
// indexes no file under it.
func Test792a5f1b_Ingest_AThrowawayWorkingDirectoryIsNotSwept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.chdir("/tmp/test-pool/run123")
	machine.fsys.put("/tmp/test-pool/run123/notes.md", ci1Markdown("notes"))
	machine.fsys.put("/tmp/test-pool/run123/deep/guide.md", ci1Markdown("guide"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(BeEmpty())
	g.Expect(out.stdout).To(BeEmpty())
}

// Test792a5f1c_Ingest_AnAncestorDirectoryUnderTheMacOSTempRootIsDropped verifies 792a5f1c (An ancestor directory
// under the macOS temp root is dropped): from a working directory under /private/var/folders/xy/ with a .claude
// directory on its ancestor chain there, `engram ingest --auto` indexes no file of that .claude directory.
func Test792a5f1c_Ingest_AnAncestorDirectoryUnderTheMacOSTempRootIsDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const temp = "/private/var/folders/xy/T"

	machine := ci1NewMachine()
	machine.chdir(temp + "/proj")
	machine.fsys.put(temp+"/.claude/notes.md", ci1Markdown("notes"))
	machine.fsys.put(temp+"/.claude/skills/skill.md", ci1Markdown("skill"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(BeEmpty())
	g.Expect(out.stdout).To(BeEmpty())
}

// Test792a5f1d_Ingest_ExplicitSourcesAreIndexedAnywhere verifies 792a5f1d (Explicit sources are indexed
// anywhere): of a transcript /tmp/explicit-session.jsonl and a directory /tmp/sweep-me holding a.md, `engram
// ingest --transcript /tmp/explicit-session.jsonl --sweep /tmp/sweep-me` indexes both. A named document and a Pi
// sessions directory under /tmp are indexed the same way.
func Test792a5f1d_Ingest_ExplicitSourcesAreIndexedAnywhere(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ci1NewMachine()
	machine.fsys.put("/tmp/explicit-session.jsonl", ci1Session("explicit"))
	machine.fsys.put("/tmp/sweep-me/a.md", ci1Markdown("swept"))
	machine.fsys.put("/tmp/explicit-doc.md", ci1Markdown("named"))
	machine.fsys.put("/tmp/pi/p.jsonl", ci1Session("pi"))

	out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--transcript", "/tmp/explicit-session.jsonl",
		"--sweep", "/tmp/sweep-me", "--markdown", "/tmp/explicit-doc.md", "--pi-sessions", "/tmp/pi")

	g.Expect(out.stderr).To(BeEmpty())
	g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{
		"/tmp/explicit-doc.md", "/tmp/explicit-session.jsonl", "/tmp/pi/p.jsonl", "/tmp/sweep-me/a.md",
	}))
}

// Test792a5g1a_Ingest_AConversationIsIndexedAsTextWithToolSummaries verifies 792a5g1a (A conversation is indexed
// as text with tool summaries): of a transcript where the user asks to run the tests, the assistant runs a Bash
// tool that prints "ok", and a system reminder is injected, `engram ingest --transcript <file>` followed by
// `engram show-chunk` prints a chunk with a USER: line, an ASSISTANT: line and a "[tool] Bash(command=...) →
// exit 0 | ok" line, and no system-reminder text.
func Test792a5g1a_Ingest_AConversationIsIndexedAsTextWithToolSummaries(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const session = "/work/session.jsonl"

	rows := []string{
		`{"type":"user","message":{"role":"user","content":"please run the tests for the whole repository"}}`,
		`{"type":"user","message":{"role":"user","content":` +
			`"<system-reminder>engram hook output that is not conversation</system-reminder>"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[` +
			`{"type":"text","text":"Running the test suite now."},` +
			`{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"targ test"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[` +
			`{"type":"tool_result","tool_use_id":"t1","content":"ok","is_error":false}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"All tests pass."}]}}`,
	}

	machine := ci1NewMachine()
	machine.fsys.put(session, strings.Join(rows, "\n")+"\n")

	ingestOut := machine.ingest(t, "--chunks-dir", ci1Chunks, "--transcript", session)

	g.Expect(ingestOut.stderr).To(BeEmpty())

	records := machine.records(t, ci1Chunks, session)

	g.Expect(records).To(HaveLen(1), "the conversation is one chunk")

	if len(records) != 1 {
		return
	}

	ref := records[0].Source + "#" + records[0].Anchor
	showOut := machine.run(t, "show-chunk", ref, "--chunks-dir", ci1Chunks)

	g.Expect(showOut.stderr).To(BeEmpty())
	g.Expect(strings.Split(strings.TrimRight(showOut.stdout, "\n"), "\n")).To(Equal([]string{
		"USER: please run the tests for the whole repository",
		"ASSISTANT: Running the test suite now.",
		`[tool] Bash(command="targ test") → exit 0 | ok`,
		"ASSISTANT: All tests pass.",
	}))
	g.Expect(showOut.stdout).NotTo(ContainSubstring("system-reminder"))
	g.Expect(showOut.stdout).NotTo(ContainSubstring("engram hook output"))
}

// TestIngest_AutoRepositoryRootFollowsEveryMarker supports 792a5d1a: the repository root of `engram ingest --auto`
// is the nearest ancestor of the working directory that contains .git, .hg or .jj, so from /r/sub the Markdown
// of /r is indexed for a marker of each of the three kinds and the Markdown outside /r is not.
func TestIngest_AutoRepositoryRootFollowsEveryMarker(t *testing.T) {
	t.Parallel()

	for _, marker := range []string{".git", ".hg", ".jj"} {
		t.Run(marker, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := ci1NewMachine()
			machine.chdir("/r/sub")
			machine.fsys.dir("/r/" + marker)
			machine.fsys.put("/r/readme.md", ci1Markdown("readme"))
			machine.fsys.put("/r/sub/guide.md", ci1Markdown("guide"))
			machine.fsys.put("/elsewhere/outside.md", ci1Markdown("outside"))

			out := machine.ingest(t, "--chunks-dir", ci1Chunks, "--auto")

			g.Expect(out.stderr).To(BeEmpty())
			g.Expect(machine.indexedSources(t, ci1Chunks)).To(Equal([]string{"/r/readme.md", "/r/sub/guide.md"}))
		})
	}
}

// unexported constants.
const (
	ci1Chunks   = "/chunks"
	ci1FileMode = 0o600
	ci1Home     = "/home/u"
	ci1Notes    = "/work/notes.md"
	ci1Vault    = "/vault"
	ci1Work     = "/work"
)

// unexported variables.
var (
	ci1Clock       = time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)
	ci1IngestRegex = regexp.MustCompile(`^ingest (\S+): (\d+) chunks `)
)

// ci1FS is an in-memory cli.EdgeFS: absolute slash paths, parents created on write, deterministic lexical walks.
// A path listed in unreadable exists (it is walked and statted) but its read fails with the given error, the way
// a file that a harness deletes between the walk and the read does. Reading a missing path fails with a bare
// fs.ErrNotExist, without the path, so an error that names the path got it from the command. It is not safe for
// concurrent use: every test builds its own.
type ci1FS struct {
	content    map[string]ci1File
	dirs       map[string]bool
	unreadable map[string]error
	clock      int64
}

func (f *ci1FS) Lstat(path string) (fs.FileInfo, error) { return f.Stat(path) }

func (f *ci1FS) MkdirAll(path string, _ fs.FileMode) error {
	f.dir(path)

	return nil
}

func (f *ci1FS) MkdirTemp(string, string) (string, error) { return "", errors.ErrUnsupported }

func (f *ci1FS) ReadDir(path string) ([]fs.DirEntry, error) {
	path = filepath.Clean(path)
	if !f.dirs[path] {
		return nil, fmt.Errorf("readdir %s: %w", path, fs.ErrNotExist)
	}

	names := make([]string, 0, len(f.content)+len(f.dirs))

	for file := range f.content {
		if filepath.Dir(file) == path {
			names = append(names, filepath.Base(file))
		}
	}

	for dir := range f.dirs {
		if dir != path && filepath.Dir(dir) == path {
			names = append(names, filepath.Base(dir))
		}
	}

	sort.Strings(names)

	entries := make([]fs.DirEntry, 0, len(names))

	for _, name := range names {
		info, err := f.Stat(filepath.Join(path, name))
		if err != nil {
			return nil, err
		}

		entries = append(entries, fs.FileInfoToDirEntry(info))
	}

	return entries, nil
}

func (f *ci1FS) ReadFile(path string) ([]byte, error) {
	path = filepath.Clean(path)

	failure, failing := f.unreadable[path]
	if failing {
		return nil, failure
	}

	file, found := f.content[path]
	if !found {
		return nil, fs.ErrNotExist
	}

	return slices.Clone(file.data), nil
}

func (f *ci1FS) Readlink(string) (string, error) { return "", errors.ErrUnsupported }

func (f *ci1FS) Remove(path string) error {
	path = filepath.Clean(path)
	delete(f.content, path)
	delete(f.dirs, path)

	return nil
}

func (f *ci1FS) RemoveAll(path string) error {
	path = filepath.Clean(path)

	for file := range f.content {
		if file == path || strings.HasPrefix(file, path+"/") {
			delete(f.content, file)
		}
	}

	for dir := range f.dirs {
		if dir == path || strings.HasPrefix(dir, path+"/") {
			delete(f.dirs, dir)
		}
	}

	return nil
}

func (f *ci1FS) Rename(oldPath, newPath string) error {
	oldPath, newPath = filepath.Clean(oldPath), filepath.Clean(newPath)

	file, found := f.content[oldPath]
	if !found {
		return fmt.Errorf("rename %s: %w", oldPath, fs.ErrNotExist)
	}

	delete(f.content, oldPath)
	f.store(newPath, file.data)

	return nil
}

func (f *ci1FS) Stat(path string) (fs.FileInfo, error) {
	path = filepath.Clean(path)

	file, found := f.content[path]
	if found {
		return ci1Info{name: filepath.Base(path), size: int64(len(file.data)), mtime: time.Unix(0, file.mtime)}, nil
	}

	if f.dirs[path] {
		return ci1Info{name: filepath.Base(path), dir: true}, nil
	}

	return nil, fmt.Errorf("stat %s: %w", path, fs.ErrNotExist)
}

func (f *ci1FS) Symlink(string, string) error { return errors.ErrUnsupported }

func (f *ci1FS) WalkDir(root string, walk fs.WalkDirFunc) error {
	root = filepath.Clean(root)

	info, err := f.Stat(root)
	if err != nil {
		err = walk(root, nil, err)
	} else {
		err = f.walk(root, fs.FileInfoToDirEntry(info), walk)
	}

	if errors.Is(err, fs.SkipDir) || errors.Is(err, fs.SkipAll) {
		return nil
	}

	return err
}

func (f *ci1FS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	f.store(path, data)

	return nil
}

func (f *ci1FS) WriteFileAtomic(path string, data []byte, _ fs.FileMode) error {
	f.store(path, data)

	return nil
}

func (f *ci1FS) WriteFileExcl(path string, data []byte, _ fs.FileMode) error {
	if _, found := f.content[filepath.Clean(path)]; found {
		return fmt.Errorf("create %s: %w", path, fs.ErrExist)
	}

	f.store(path, data)

	return nil
}

// dir creates a directory and its parents.
func (f *ci1FS) dir(path string) {
	path = filepath.Clean(path)

	for path != "/" && path != "." && !f.dirs[path] {
		f.dirs[path] = true
		path = filepath.Dir(path)
	}
}

// files returns every file path, sorted.
func (f *ci1FS) files() []string {
	paths := make([]string, 0, len(f.content))

	for path := range f.content {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	return paths
}

// put stores a file the scenario's Given has, without touching the other files.
func (f *ci1FS) put(path, content string) { f.store(path, []byte(content)) }

// store writes a file with a modification time later than that of every earlier write, creating its parents.
func (f *ci1FS) store(path string, data []byte) {
	path = filepath.Clean(path)
	f.clock++
	f.content[path] = ci1File{data: slices.Clone(data), mtime: f.clock}
	f.dir(filepath.Dir(path))
}

// under returns the sorted paths of the files and directories at or below a directory.
func (f *ci1FS) under(dir string) []string {
	dir = filepath.Clean(dir)
	paths := make([]string, 0, len(f.content)+len(f.dirs))

	for file := range f.content {
		if strings.HasPrefix(file, dir+"/") {
			paths = append(paths, file)
		}
	}

	for known := range f.dirs {
		if known == dir || strings.HasPrefix(known, dir+"/") {
			paths = append(paths, known)
		}
	}

	sort.Strings(paths)

	return paths
}

// walk mirrors the walk of the standard library over the in-memory tree.
func (f *ci1FS) walk(path string, entry fs.DirEntry, walk fs.WalkDirFunc) error {
	err := walk(path, entry, nil)
	if err != nil || !entry.IsDir() {
		if errors.Is(err, fs.SkipDir) && entry.IsDir() {
			err = nil
		}

		return err
	}

	children, err := f.ReadDir(path)
	if err != nil {
		return walk(path, entry, err)
	}

	for _, child := range children {
		err = f.walk(filepath.Join(path, child.Name()), child, walk)
		if err != nil {
			if errors.Is(err, fs.SkipDir) {
				break
			}

			return err
		}
	}

	return nil
}

// ci1File is one file of the in-memory file system.
type ci1File struct {
	data  []byte
	mtime int64
}

// ci1Info is the fs.FileInfo of a file or directory of ci1FS.
type ci1Info struct {
	name  string
	size  int64
	dir   bool
	mtime time.Time
}

func (i ci1Info) IsDir() bool { return i.dir }

func (i ci1Info) ModTime() time.Time { return i.mtime }

func (i ci1Info) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | ci1FileMode
	}

	return ci1FileMode
}

func (i ci1Info) Name() string { return i.name }

func (i ci1Info) Size() int64 { return i.size }

func (i ci1Info) Sys() any { return nil }

// ci1IngestLine is one "ingest <path>: N chunks" line of stdout.
type ci1IngestLine struct {
	Path   string
	Chunks int
}

// ci1Machine is the world a command runs in: a file system, the environment variables that are set, the home
// directory and the working directory. The vault directory exists, so no run creates a vault.
type ci1Machine struct {
	fsys *ci1FS
	env  map[string]string
	home string
	cwd  string
}

// chdir makes a directory, which exists, the working directory.
func (m *ci1Machine) chdir(dir string) {
	m.fsys.dir(dir)
	m.cwd = dir
}

// indexedSources returns the sorted distinct sources of the records of every index file under a chunk index
// directory.
func (m *ci1Machine) indexedSources(t *testing.T, chunksDir string) []string {
	t.Helper()

	seen := map[string]bool{}

	for _, path := range m.fsys.under(chunksDir) {
		if filepath.Ext(path) != ".jsonl" {
			continue
		}

		records, err := chunk.DecodeRecords(m.fsys.content[path].data)
		if err != nil {
			t.Fatalf("decoding the index file %s: %v", path, err)
		}

		for _, record := range records {
			seen[record.Source] = true
		}
	}

	sources := make([]string, 0, len(seen))

	for source := range seen {
		sources = append(sources, source)
	}

	sort.Strings(sources)

	return sources
}

// ingest runs `engram ingest --vault <vault> <args>`.
func (m *ci1Machine) ingest(t *testing.T, args ...string) ci1Outcome {
	t.Helper()

	return m.run(t, append([]string{"ingest", "--vault", ci1Vault}, args...)...)
}

// manifestSources returns the sorted sources the manifest of a chunk index directory records.
func (m *ci1Machine) manifestSources(t *testing.T, chunksDir string) []string {
	t.Helper()

	manifest := map[string]json.RawMessage{}

	file, found := m.fsys.content[chunksDir+"/manifest.json"]
	if found {
		err := json.Unmarshal(file.data, &manifest)
		if err != nil {
			t.Fatalf("decoding the manifest: %v", err)
		}
	}

	sources := make([]string, 0, len(manifest))

	for source := range manifest {
		sources = append(sources, source)
	}

	sort.Strings(sources)

	return sources
}

// records returns the records of the index file of a source, none when it has no index file.
func (m *ci1Machine) records(t *testing.T, chunksDir, source string) []chunk.Record {
	t.Helper()

	file, found := m.fsys.content[chunksDir+"/"+cli.ExportIndexFileName(source)]
	if !found {
		return []chunk.Record{}
	}

	records, err := chunk.DecodeRecords(file.data)
	if err != nil {
		t.Fatalf("decoding the index file of %s: %v", source, err)
	}

	return records
}

// run runs `engram <args>` through cli.Targets with the machine's world injected.
func (m *ci1Machine) run(t *testing.T, args ...string) ci1Outcome {
	t.Helper()

	var stdout, stderr bytes.Buffer

	var exits []int

	deps := newTestDeps(&stdout, &stderr)
	deps.FS = m.fsys
	deps.Lock = fakeLocker{lock: func(string) (func() error, error) { return func() error { return nil }, nil }}
	deps.Embed = fakeIngestEmbedder{}
	deps.DebugLog = nil
	deps.Getenv = func(key string) string { return m.env[key] }
	deps.UserHomeDir = func() (string, error) { return m.home, nil }
	deps.Getwd = func() (string, error) { return m.cwd, nil }
	deps.Now = func() time.Time { return ci1Clock }
	deps.Exit = func(code int) { exits = append(exits, code) }

	_, err := targ.Execute(append([]string{"engram"}, args...), cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(err.Error() + "\n")
	}

	return ci1Outcome{stdout: stdout.String(), stderr: stderr.String(), exits: exits}
}

// ci1Outcome is what one command printed and the exit codes it asked for.
type ci1Outcome struct {
	stdout string
	stderr string
	exits  []int
}

// ci1IngestLines returns the "ingest <path>: N chunks" lines of stdout.
func ci1IngestLines(stdout string) []ci1IngestLine {
	var lines []ci1IngestLine

	for line := range strings.Lines(stdout) {
		match := ci1IngestRegex.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		chunks, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}

		lines = append(lines, ci1IngestLine{Path: match[1], Chunks: chunks})
	}

	return lines
}

// ci1LinesWithPrefix returns the lines of stdout that start with a prefix.
func ci1LinesWithPrefix(stdout, prefix string) []string {
	var lines []string

	for line := range strings.Lines(stdout) {
		if strings.HasPrefix(line, prefix) {
			lines = append(lines, strings.TrimRight(line, "\n"))
		}
	}

	return lines
}

// ci1Markdown is a Markdown document of one section on a topic, long enough to be a chunk. Documents on different
// topics differ, so no two of them are the identical content that ingest indexes once.
func ci1Markdown(topic string) string {
	return "## " + topic + "\nThe section on " + topic + " holds enough words to clear the chunk noise floor.\n"
}

// ci1NewFS returns an empty file system.
func ci1NewFS() *ci1FS {
	return &ci1FS{content: map[string]ci1File{}, dirs: map[string]bool{}, unreadable: map[string]error{}}
}

// ci1NewMachine returns a machine with an empty file system but for the vault, no environment variable set, the
// home directory /home/u and the working directory /work.
func ci1NewMachine() *ci1Machine {
	machine := &ci1Machine{fsys: ci1NewFS(), env: map[string]string{}, home: ci1Home, cwd: ci1Work}
	machine.fsys.dir(ci1Vault)

	return machine
}

// ci1Session is a session transcript of one user turn and one assistant turn on a topic, long enough to be a
// chunk and different from the session on any other topic.
func ci1Session(topic string) string {
	return fmt.Sprintf(`{"type":"user","message":{"role":"user","content":%q}}`+"\n"+
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":%q}]}}`+"\n",
		"please explain the "+topic+" part of the design in some detail",
		"the "+topic+" part of the design is explained in detail here")
}
