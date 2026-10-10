# Chunk Indexing - Product Spec

`engram ingest` turns session transcripts and Markdown documents into a chunk index: the text of
each source is cut into chunks, each chunk is embedded, and the chunks are stored outside the vault
so that `engram query` can search them as evidence. `engram prune` maintains that index. No LLM is
involved. Searching chunks and fetching one chunk's text (`engram query`, `query-chunks`,
`show-chunk`) is the memory-query capability ([memory-query](./memory-query.md)), and the backlog notice that `engram update` prints for
duplicate index files belongs to [install-and-update](./install-and-update.md). A chunk is named by its chunk ID,
`<source path>#<anchor>`.

## 792a5a Chunk index location

Requirement: [792a5a Chunk index location](#792a5a-chunk-index-location) Every engram command that reads or writes the chunk index MUST use the
directory given by `--chunks-dir` when it is non-empty, else `ENGRAM_CHUNKS_DIR` when it is
non-empty, else `$XDG_DATA_HOME/engram/chunks` when `XDG_DATA_HOME` is non-empty, else
`~/.local/share/engram/chunks`.

Rationale: the index follows the same precedence as the vault, so a test or eval run can point both
at an isolated location (`ENGRAM_CHUNKS_DIR` with `ENGRAM_VAULT_PATH`) without touching the real
index.

### 792a5a1 UAT Scenarios

```gherkin
Scenario: 792a5a1a The flag wins over the environment
  Given ENGRAM_CHUNKS_DIR is set to /env/chunks
  When the user runs engram ingest --chunks-dir /flag/chunks --markdown notes.md
  Then notes.md is indexed under /flag/chunks

Scenario: 792a5a1b The environment variable is used without a flag
  Given ENGRAM_CHUNKS_DIR is set to /env/chunks
  When the user runs engram ingest --markdown notes.md
  Then notes.md is indexed under /env/chunks

Scenario: 792a5a1c The XDG data directory is the default
  Given neither --chunks-dir nor ENGRAM_CHUNKS_DIR is set and XDG_DATA_HOME is /data
  When the user runs engram ingest --markdown notes.md
  Then notes.md is indexed under /data/engram/chunks

Scenario: 792a5a1d The home directory is the last fallback
  Given neither --chunks-dir, ENGRAM_CHUNKS_DIR nor XDG_DATA_HOME is set
  When the user runs engram ingest --markdown notes.md
  Then notes.md is indexed under ~/.local/share/engram/chunks
```

## 792a5b Explicitly named sources

Requirement: [792a5b Explicitly named sources](#792a5b-explicitly-named-sources) `engram ingest` MUST index each file named by a repeatable `--transcript`
(a session transcript) or `--markdown` (a Markdown document), and each `.md` and `.jsonl` file under
each directory named by a repeatable `--pi-sessions`, skipping that directory's `jobs/` and
`projects/` subdirectories and its hidden subdirectories. It MUST ignore a `--pi-sessions` value that
is not a directory, and MUST fail with an error naming the file when a file named by `--transcript`
or `--markdown` cannot be read.

### 792a5b1 UAT Scenarios

```gherkin
Scenario: 792a5b1a A named transcript and document are indexed
  Given a session transcript s.jsonl and a Markdown document d.md
  When the user runs engram ingest --transcript s.jsonl --markdown d.md
  Then the chunk index holds chunks of both files, and stdout has one "ingest <path>: N chunks" line for each

Scenario: 792a5b1b A Pi sessions directory is indexed without its scratch
  Given a directory p holding a.jsonl, jobs/b.jsonl, projects/c.jsonl and .hidden/d.jsonl
  When the user runs engram ingest --pi-sessions p
  Then a.jsonl is indexed and none of the other three files is

Scenario: 792a5b1c A Pi sessions value that is not a directory is ignored
  Given a regular file f
  When the user runs engram ingest --pi-sessions f
  Then the command succeeds and indexes nothing

Scenario Outline: 792a5b1d An unreadable named file fails the command
  Given no file exists at <file>
  When the user runs engram ingest <flag> <file>
  Then the command fails with an error naming <file>

  Examples:
    | flag         | file          |
    | --markdown   | missing.md    |
    | --transcript | missing.jsonl |
```

## 792a5c Swept directories

Requirement: [792a5c Swept directories](#792a5c-swept-directories) `engram ingest --sweep <dir>`, repeatable, MUST index every `.md` and `.jsonl`
file under the directory, skipping hidden subdirectories, subdirectories named `node_modules`,
`vendor`, `.git`, `.hg`, `.jj`, `dist`, `build`, `target`, `.venv`, `venv`, `__pycache__`, `.next`,
`.cache` or `.idea`, and any file inside the chunk index directory. A swept file that cannot be read
MUST be skipped with a `skip <path>: ...` line on stdout while the rest of the run continues.

Rationale: a named file is a deliberate request, so failing to read it is an error. A swept file can
vanish between the walk and the read, for example a session that a harness deletes, and losing one
file must not stop the sweep. Skipping the index directory keeps a sweep root that contains the
index from indexing the index itself.

### 792a5c1 UAT Scenarios

```gherkin
Scenario: 792a5c1a A swept directory's documents and transcripts are indexed
  Given a directory r holding a.md, sub/b.jsonl, node_modules/c.md, .git/d.md and notes.txt
  When the user runs engram ingest --sweep r
  Then a.md and sub/b.jsonl are indexed, and node_modules/c.md, .git/d.md and notes.txt are not

Scenario: 792a5c1b A swept file that vanishes is skipped
  Given a swept directory whose file gone.md cannot be read when the sweep reaches it
  When the user runs engram ingest --sweep <dir>
  Then stdout has a "skip <dir>/gone.md:" line, the directory's other files are indexed, and the command succeeds

Scenario: 792a5c1c The chunk index directory is never indexed
  Given a chunk index directory inside the swept directory r
  When the user runs engram ingest --sweep r --chunks-dir r/chunks
  Then no file under r/chunks is indexed
```

## 792a5d Automatic sweep roots

Requirement: [792a5d Automatic sweep roots](#792a5d-automatic-sweep-roots) `engram ingest --auto` MUST sweep, as in [792a5c Swept directories](#792a5c-swept-directories), the repository root (the
nearest ancestor of the working directory, itself included, that contains `.git`, `.hg` or `.jj`, or
the working directory when none does), every `.claude` and every `.pi` directory on the working
directory's ancestor chain up to the filesystem root, and the session-log directory
(`ENGRAM_TRANSCRIPT_DIR` when set, else `~/.claude/projects`) when it exists. Inside a `.claude`
directory it MUST also skip the subdirectories `projects`, `jobs`, `plugins`, `cache`, `todos`,
`shell-snapshots`, `file-history`, `history`, `ide`, `statsig`, `session-env`, `debug` and
`worktrees`, and inside a `.pi` directory the subdirectories `jobs` and `projects`.

Rationale: the automatic sweep covers what an agent's memory can learn from: the project's own documents, project and
user harness configuration, and every recorded conversation of every project. Inside `.claude`, harness state and
third-party plugin content are not user memory, and `projects/` holds every project's transcripts, which arrive
through the session-log root instead. `jobs/` is agent-harness scratch that can hold whole snapshot copies of the
vault, and indexing it would fill the index with duplicates.

### 792a5d1 UAT Scenarios

```gherkin
Scenario: 792a5d1a The repository, its ancestors and the session logs are swept
  Given a working directory /r/sub inside a git repository /r, a /r/.claude directory, a /r/sub/.pi directory, and session logs under ~/.claude/projects
  When the user runs engram ingest --auto from /r/sub
  Then Markdown under /r, the files of /r/.claude and /r/sub/.pi, and the session transcripts are indexed

Scenario: 792a5d1b Without a repository marker the working directory is the root
  Given a working directory /w with no .git, .hg or .jj in it or any ancestor
  When the user runs engram ingest --auto from /w
  Then Markdown under /w is indexed

Scenario: 792a5d1c Harness scratch and state are not swept
  Given an ancestor .claude directory holding notes.md, jobs/snap.md, projects/p/s.jsonl and plugins/x.md
  When the user runs engram ingest --auto
  Then notes.md is indexed and jobs/snap.md, projects/p/s.jsonl and plugins/x.md are not

Scenario: 792a5d1d ENGRAM_TRANSCRIPT_DIR names the session logs
  Given ENGRAM_TRANSCRIPT_DIR is set to /logs, which holds a.jsonl
  When the user runs engram ingest --auto
  Then /logs/a.jsonl is indexed, and no transcript under ~/.claude/projects is
```

## 792a5e Repository sweep configuration

Requirement: [792a5e Repository sweep configuration](#792a5e-repository-sweep-configuration) `engram ingest --auto` MUST read `.engram/sweep.json` at the repository root
when it exists, letting each key it sets replace that setting's default: `repo_markdown`,
`ancestor_claude_dirs`, `ancestor_pi_dirs` and `session_logs` (each root on or off),
`extra_roots` (more directories to sweep), `exclude_dirs`, `claude_exclude_dirs`,
`include_hidden_dirs`, `non_persistent_prefixes` and `non_persistent_path_prefixes`. A setting the
file omits MUST keep its default, and a file that is not valid JSON MUST fail the command with an
error naming the file.

Rationale: the sweep is declarative and inspectable. Its defaults are compiled in, and a repository
tunes them by editing data rather than code.

### 792a5e1 UAT Scenarios

```gherkin
Scenario: 792a5e1a A repository adds a root and turns one off
  Given a repository whose .engram/sweep.json is {"session_logs": false, "extra_roots": ["/x"]}
  When the user runs engram ingest --auto inside it
  Then /x and the repository are swept, and no session log is indexed

Scenario: 792a5e1b A malformed sweep file is refused
  Given a repository whose .engram/sweep.json is not valid JSON
  When the user runs engram ingest --auto inside it
  Then the command fails with an error naming .engram/sweep.json, and nothing is indexed
```

## 792a5f Throwaway workspaces are not swept

Requirement: [792a5f Throwaway workspaces are not swept](#792a5f-throwaway-workspaces-are-not-swept) `engram ingest --auto` MUST NOT index a session-log project directory whose name
starts with `-private-tmp-`, `-tmp-`, `-var-folders-` or `-private-var-folders-`, and MUST NOT sweep a
repository, `.claude`, `.pi` or session-log root whose own path is, or is under, `/tmp`,
`/private/tmp`, `/var/folders` or `/private/var/folders`. Files named by `--transcript`, `--markdown`
or `--pi-sessions`, directories named by `--sweep`, and `extra_roots` MUST be indexed wherever they
are.

Rationale: eval and test runs work in throwaway directories, and indexing them would bloat the real
index permanently. The project-directory names are the slugified working directories under which
Claude Code records sessions, so a session run in `/private/tmp/...` lands in a directory named
`-private-tmp-...`. A root whose own path is throwaway, such as a repository root that resolves to
`/tmp/eval-pool/warm0` because that directory has no repository marker, is dropped whole, because a
name check only reaches a root's subdirectories. Deliberate test ingestion uses an explicit source
or an isolated index (`ENGRAM_CHUNKS_DIR`, `ENGRAM_VAULT_PATH`).

### 792a5f1 UAT Scenarios

```gherkin
Scenario: 792a5f1a Sessions recorded in a throwaway directory are skipped
  Given session logs under ~/.claude/projects/-private-tmp-eval-1/ and ~/.claude/projects/-Users-u-repo/
  When the user runs engram ingest --auto
  Then the -Users-u-repo sessions are indexed and the -private-tmp-eval-1 sessions are not

Scenario: 792a5f1b A throwaway working directory is not swept
  Given a working directory /tmp/test-pool/run123 with no repository marker
  When the user runs engram ingest --auto from it
  Then no file under /tmp/test-pool/run123 is indexed

Scenario: 792a5f1c An ancestor directory under the macOS temp root is dropped
  Given a working directory under /private/var/folders/xy/ with a .claude directory on its ancestor chain there
  When the user runs engram ingest --auto from it
  Then no file of that .claude directory is indexed

Scenario: 792a5f1d Explicit sources are indexed anywhere
  Given a transcript /tmp/explicit-session.jsonl and a directory /tmp/sweep-me holding a.md
  When the user runs engram ingest --transcript /tmp/explicit-session.jsonl --sweep /tmp/sweep-me
  Then both files are indexed
```

## 792a5g Transcript content

Requirement: [792a5g Transcript content](#792a5g-transcript-content) `engram ingest` MUST index a session transcript as its user and assistant text,
each turn's text prefixed `USER: ` or `ASSISTANT: `, with each tool call whose result is recorded as
plain text shown as one `[tool] <name>(<arguments>) → exit <0|1> | <first output line>` line, and
MUST leave out every other
transcript entry, system reminders, injected skill bodies, local-command output, task notifications,
and slash-command invocations other than their arguments.

Rationale: a transcript row carries far more harness bookkeeping than conversation. Chunks are
evidence an agent reads back, so they keep what was said and done, compressed to a line per tool
call, and drop what the harness injected.

### 792a5g1 UAT Scenarios

```gherkin
Scenario: 792a5g1a A conversation is indexed as text with tool summaries
  Given a transcript where the user asks to run the tests, the assistant runs a Bash tool that prints "ok", and a system reminder is injected
  When the user runs engram ingest --transcript <file> and reads the chunk with engram show-chunk
  Then the chunk text has a USER: line, an ASSISTANT: line and a "[tool] Bash(command=...) → exit 0 | ok" line, and no system-reminder text
```

## 792a5h Chunk boundaries and IDs

Requirement: [792a5h Chunk boundaries and IDs](#792a5h-chunk-boundaries-and-ids) `engram ingest` MUST cut a Markdown document into one chunk per heading section,
ignoring headings inside fenced code, with the heading's text as the anchor and `preamble` for text
before the first heading. It MUST cut a transcript into chunks of consecutive turns of about 500
characters, anchored `turn-<N>` by the first user turn they hold, or `preamble` for text before the
first user turn. It MUST split any section or turn group longer than 1500 characters into pieces that share its anchor, packing whole non-blank lines
in order and starting a new piece when the next line would take the piece past 1500 characters (a
single line longer than 1500 characters stays whole), and MUST NOT index a chunk shorter than 40
characters.

### 792a5h1 UAT Scenarios

```gherkin
Scenario: 792a5h1a Markdown is chunked by heading
  Given d.md holding text before any heading, a "## Setup" section and a "## Usage" section, each over 40 characters
  When the user runs engram ingest --markdown d.md
  Then the index holds the chunks d.md#preamble, d.md#Setup and d.md#Usage

Scenario: 792a5h1b A transcript is chunked by turns
  Given s.jsonl holding many short user and assistant turns
  When the user runs engram ingest --transcript s.jsonl
  Then each chunk holds consecutive turns, and their IDs are s.jsonl#turn-1, then the turn numbers at which later chunks start

Scenario: 792a5h1c Short sections are not indexed
  Given d.md whose only section is "## A" followed by "tiny"
  When the user runs engram ingest --markdown d.md
  Then no chunk of d.md is indexed

Scenario: 792a5h1d A long section is split under one anchor
  Given d.md whose "## Big" section is three paragraphs of about 940 characters each
  When the user runs engram ingest --markdown d.md
  Then the index holds several chunks d.md#Big, each at most 1500 characters long
```

## 792a5i Incremental re-indexing

Requirement: [792a5i Incremental re-indexing](#792a5i-incremental-re-indexing) `engram ingest` MUST NOT read or re-embed a source whose modification time and
size match its last ingest, MUST re-chunk a source whose content changed and embed only the chunks
whose text is not already in that source's index, and MUST print
`ingest <path>: <N> chunks (<R> reused, <E> embedded)` for each source it re-chunks, where `<N>` is
the number of chunks the source's index now holds, and nothing for a source it does not.

Rationale: the automatic sweep runs at the start of every learn, over every session and document
the user has, so an unchanged source must cost only a stat. Reusing the vectors of unchanged chunk
text means editing one section of a long document re-embeds only that section.

### 792a5i1 UAT Scenarios

```gherkin
Scenario: 792a5i1a An unchanged source costs nothing
  Given a source that was ingested and has not changed since
  When the user runs engram ingest again with the same source
  Then the source is not read, nothing is embedded, and stdout has no line for it

Scenario: 792a5i1b A changed source embeds only its new text
  Given a Markdown document with sections A and B that was ingested, after which section B was edited
  When the user runs engram ingest again with the document
  Then stdout reads "ingest <path>: 3 chunks (1 reused, 1 embedded)", the third chunk being B's earlier text (792a5j)
```

## 792a5j A source's chunks are never dropped

Requirement: [792a5j A source's chunks are never dropped](#792a5j-a-sources-chunks-are-never-dropped) When a source's content changes, `engram ingest` MUST keep every chunk the source
yielded on earlier ingests, alongside the chunks of its new content.

Rationale: an embedded chunk is the memory; its source is only where it came from. Keeping earlier
chunks means rewriting or truncating a document never erases what the index learned from it. The
cost is that a document edited many times keeps every earlier version of its edited sections.

### 792a5j1 UAT Scenarios

```gherkin
Scenario: 792a5j1a Edited-away text stays searchable
  Given a document that was ingested with the paragraph "use flock for the manifest"
  When the paragraph is replaced by other text and the user runs engram ingest again
  Then the index holds both the old paragraph's chunk and the new text's chunk
```

## 792a5k Sources without chunks leave no index file

Requirement: [792a5k Sources without chunks leave no index file](#792a5k-sources-without-chunks-leave-no-index-file) `engram ingest` MUST NOT write an index file for a source that yields no
chunks.

Rationale: every query opens every index file, so an empty file costs a read on every query and adds
nothing to any result.

### 792a5k1 UAT Scenarios

```gherkin
Scenario: 792a5k1a An empty transcript writes no index file
  Given an empty transcript e.jsonl
  When the user runs engram ingest --transcript e.jsonl
  Then stdout reads "ingest e.jsonl: 0 chunks (0 reused, 0 embedded)", and no index file exists for e.jsonl
```

## 792a5l Identical content is indexed once

Requirement: [792a5l Identical content is indexed once](#792a5l-identical-content-is-indexed-once) When sources of the same kind (transcript or Markdown) have byte-identical
content, `engram ingest` MUST index only one of them, chosen by precedence: a source named by
`--transcript`, `--markdown` or `--pi-sessions`; then one under the repository root; then one under
an ancestor `.claude` or `.pi` directory, nearest the working directory first; then any other; and
among equals the path with the fewest separators, then the lexicographically first path. It MUST
index a transcript and a Markdown document separately even when their bytes are identical.

Rationale: the same content is often reachable from several swept paths, such as a project file copied into an
ancestor `.claude` directory, a session copied into a worktree, or a vault note snapshotted into a scratch directory.
A list of excluded paths could not reach most of it, because the copies sit in legitimately swept roots. The
precedence is a fixed function of the copies found, so the same content always resolves to the same copy whatever
order the sweep finds them in. A transcript is chunked differently from a Markdown document with the same bytes, so
treating them as copies would lose one of the two chunkings.

### 792a5l1 UAT Scenarios

```gherkin
Scenario: 792a5l1a A repository file outranks its copy in an ancestor .claude directory
  Given byte-identical files /r/doc.md in the repository and /r/.claude/doc.md
  When the user runs engram ingest --auto from /r
  Then /r/doc.md is indexed and /r/.claude/doc.md is not

Scenario: 792a5l1b A named file outranks the repository copy
  Given byte-identical files /x/doc.md and /r/doc.md in the repository
  When the user runs engram ingest --auto --markdown /x/doc.md from /r
  Then /x/doc.md is indexed and /r/doc.md is not

Scenario: 792a5l1c The nearer ancestor directory wins
  Given byte-identical files in /r/a/.pi and in /r/.claude, and the working directory /r/a
  When the user runs engram ingest --auto from /r/a
  Then the /r/a/.pi copy is indexed and the /r/.claude copy is not

Scenario: 792a5l1d A transcript and a document with the same bytes are both indexed
  Given a.jsonl and a.md with byte-identical content
  When the user runs engram ingest --transcript a.jsonl --markdown a.md
  Then both files have chunks in the index
```

## 792a5m A later, preferred copy replaces an indexed one

Requirement: [792a5m A later, preferred copy replaces an indexed one](#792a5m-a-later-preferred-copy-replaces-an-indexed-one) When an already-indexed source becomes a copy of a newly found source that takes
precedence over it ([792a5l Identical content is indexed once](#792a5l-identical-content-is-indexed-once)), `engram ingest` MUST index the preferred source and remove the other
source's chunks, but only when every chunk the other source holds is also among the preferred
source's chunks. Otherwise it MUST keep the other source's chunks and print
``ingest: refusing to evict <path>: canonical <path> does not verifiably cover its records (needs review — run `engram prune --duplicates` once resolved)``.
When the preferred copy stops being found, the remaining copy MUST be indexed again.

Rationale: byte-identical content today does not mean identical chunks, because a source keeps every chunk of its
earlier versions ([792a5j A source's chunks are never dropped](#792a5j-a-sources-chunks-are-never-dropped)). Two files
equal now can hold different histories, and removing one on the strength of the other would destroy chunks that exist
nowhere else.

### 792a5m1 UAT Scenarios

```gherkin
Scenario: 792a5m1a A copy indexed first gives way to the preferred one
  Given /r/.claude/doc.md was indexed while /r/doc.md did not exist
  When /r/doc.md is created with the same bytes and the user runs engram ingest --auto from /r
  Then /r/doc.md is indexed and /r/.claude/doc.md no longer has chunks in the index

Scenario: 792a5m1b A copy with a history of its own is kept
  Given /r/.claude/doc.md was indexed with an earlier version, then edited to equal a new /r/doc.md
  When the user runs engram ingest --auto from /r
  Then /r/.claude/doc.md's chunks are kept, and stdout has a "refusing to evict /r/.claude/doc.md" line

Scenario: 792a5m1c The remaining copy is indexed when the preferred one goes away
  Given /r/doc.md is indexed and /r/.claude/doc.md is recorded as its copy
  When /r/doc.md is deleted and the user runs engram ingest --auto from /r
  Then /r/.claude/doc.md is indexed
```

## 792a5n Stray vault-note copies are not indexed

Requirement: [792a5n Stray vault-note copies are not indexed](#792a5n-stray-vault-note-copies-are-not-indexed) `engram ingest` MUST NOT index a swept `.md` file that has a sibling `.vec.json`
file and lies outside the vault ([792a2a Vault location](./vault-identity.md#792a2a-vault-location)), and MUST index such a file when it lies inside the vault or
is named by `--markdown`.

Rationale: a vault note always has a `.vec.json` sidecar beside it, so such a file outside the vault is a copy of a
vault note, for example in a snapshot or a backup. A copy of a note amended since it was taken no longer has the same
bytes as the live note, so identical-content deduplication cannot catch it.

### 792a5n1 UAT Scenarios

```gherkin
Scenario: 792a5n1a A vault-note copy outside the vault is skipped
  Given a swept directory holding 1.2026-10-09.x.md with 1.2026-10-09.x.vec.json beside it, and plain.md with no sidecar, outside the vault
  When the user runs engram ingest --sweep <dir>
  Then plain.md is indexed and 1.2026-10-09.x.md is not
```

## 792a5o Concurrent index writers never lose an update

Requirement: [792a5o Concurrent index writers never lose an update](#792a5o-concurrent-index-writers-never-lose-an-update) `engram ingest` and `engram prune` runs against one chunk index at the same time
MUST NOT lose either one's change to the index's record of sources.

Rationale: concurrent ingest and prune runs each read the record of sources, and the later write would overwrite the
earlier one's change.

### 792a5o1 UAT Scenarios

```gherkin
Scenario: 792a5o1a An ingest and a prune at the same time both take effect
  Given a chunk index that records a source whose file is gone
  When engram ingest --transcript new.jsonl and engram prune start at the same time
  Then afterwards the index records new.jsonl and no longer records the gone source
```

## 792a5p Prune detaches vanished sources

Requirement: [792a5p Prune detaches vanished sources](#792a5p-prune-detaches-vanished-sources) `engram prune` MUST stop recording each indexed source whose file no longer
exists, MUST keep that source's chunks in the index and searchable, and MUST print
`prune: detached <N> source(s) — embedded chunks preserved (still searchable)`, or
`prune: no dead sources` when every recorded source exists, or `prune: no manifest, nothing to prune`
when the index records no sources. With `--dry-run` it MUST print the same detach line prefixed
`[dry-run] ` and change nothing.

Rationale: the embedded chunk is the asset and the source file only its provenance. A user can delete
ingested files, for example a directory of restored transcripts, to reclaim disk without losing what
was learned from them.

### 792a5p1 UAT Scenarios

```gherkin
Scenario: 792a5p1a A deleted source is detached and stays searchable
  Given a transcript that was ingested and then deleted
  When the user runs engram prune and then engram query with a phrase matching it
  Then prune prints "prune: detached 1 source(s) — embedded chunks preserved (still searchable)", and the query returns its chunks with no re-indexing

Scenario: 792a5p1b A source that still exists is kept
  Given an ingested source whose file still exists
  When the user runs engram prune
  Then it prints "prune: no dead sources", and the source is still recorded

Scenario: 792a5p1c An empty index has nothing to prune
  Given a chunk index directory with nothing ingested
  When the user runs engram prune
  Then it prints "prune: no manifest, nothing to prune" and succeeds

Scenario: 792a5p1d A dry run changes nothing
  Given an ingested source that was then deleted
  When the user runs engram prune --dry-run
  Then it prints "[dry-run] prune: detached 1 source(s) — embedded chunks preserved (still searchable)", and the source is still recorded
```

## 792a5q Prune removes empty index files

Requirement: [792a5q Prune removes empty index files](#792a5q-prune-removes-empty-index-files) `engram prune --empty` MUST remove every empty index file in the chunk index and
print `prune: removed <N> empty chunk-index file(s) of <M> scanned`. With `--dry-run` it MUST print
the same line prefixed `[dry-run] ` and remove nothing.

Rationale: indexes built before [792a5k Sources without chunks leave no index file](#792a5k-sources-without-chunks-leave-no-index-file) left an empty file for each source with no chunks. Removing
them changes no search result, since they hold no chunks.

### 792a5q1 UAT Scenarios

```gherkin
Scenario: 792a5q1a Empty index files are removed
  Given a chunk index with three index files, one of them empty
  When the user runs engram prune --empty
  Then the empty file is gone, the other two remain, and it prints "prune: removed 1 empty chunk-index file(s) of 3 scanned"

Scenario: 792a5q1b A dry run removes nothing
  Given a chunk index with one empty index file
  When the user runs engram prune --empty --dry-run
  Then the file remains, and it prints "[dry-run] prune: removed 1 empty chunk-index file(s) of 1 scanned"
```

## 792a5r Prune removes duplicate index files

Requirement: [792a5r Prune removes duplicate index files](#792a5r-prune-removes-duplicate-index-files) `engram prune --duplicates` MUST group the recorded sources that are not already
recorded as copies by content and kind ([792a5l Identical content is indexed once](#792a5l-identical-content-is-indexed-once)), keep one source of each group, choosing the path
with the fewest separators and then the lexicographically first path, and remove every other
source's chunks and record, but only when the kept source has chunks in the index that include all
of the removed source's chunks. It MUST then print
`prune: removed <N> duplicate(s), retained <M> canonical(s)`, a
`prune: refused <K> removal(s): canonical has no index file (zero-chunk sources; nothing to lose)`
line when no source of a refused group has chunks, and a `refusing to remove <path>` line for each
refusal whose group does have chunks elsewhere. A removal that fails MUST print
`[FAIL] <path>: <error>` on stderr without stopping the others, and the command MUST then fail saying
how many of how many removals failed. With `--dry-run` it MUST report the same counts, each line
prefixed `[dry-run] `, and change nothing.

Rationale: ingest-time deduplication ([792a5l Identical content is indexed
once](#792a5l-identical-content-is-indexed-once)) prevents new copies but does nothing for copies indexed before it
existed, so this mode cleans up that backlog. It runs only when asked, because it removes index files from the user's
memory store. It knows only the paths in the index's record, not how each was found, so it cannot apply ingest's
precedence; a later `engram ingest --auto` may then replace the copy it kept with a preferred one ([792a5m A later,
preferred copy replaces an indexed one](#792a5m-a-later-preferred-copy-replaces-an-indexed-one)), once per group.

### 792a5r1 UAT Scenarios

```gherkin
Scenario: 792a5r1a A duplicate group is collapsed
  Given an index where /a/doc.md and /a/b/doc.md are both indexed with identical content and chunks
  When the user runs engram prune --duplicates
  Then /a/doc.md is kept, /a/b/doc.md no longer has chunks or a record, and it prints "prune: removed 1 duplicate(s), retained 1 canonical(s)"

Scenario: 792a5r1b A duplicate with chunks of its own is refused
  Given two indexed copies of one content, where the copy that would be removed holds a chunk the kept copy lacks
  When the user runs engram prune --duplicates
  Then both copies keep their chunks, and stdout has a "refusing to remove" line naming the copy

Scenario: 792a5r1c A dry run removes nothing
  Given an index with one removable duplicate
  When the user runs engram prune --duplicates --dry-run
  Then it prints "[dry-run] prune: removed 1 duplicate(s), retained 1 canonical(s)", and both copies remain

Scenario: 792a5r1d A second run removes nothing more
  Given an index on which engram prune --duplicates has just run
  When the user runs engram prune --duplicates again
  Then it prints "prune: removed 0 duplicate(s), retained 0 canonical(s)", and nothing changes

Scenario: 792a5r1e A failed removal is reported and fails the run
  Given an index with two removable duplicates, one of whose index files cannot be removed
  When the user runs engram prune --duplicates
  Then stderr has a "[FAIL]" line naming that file, the other duplicate is removed, and the command fails saying 1 of 2 removals failed
```
