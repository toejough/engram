# Ingestion Source Discovery - Technical Spec

## 792b17a Orientation

- Ownership: Repository component, walking the local filesystem through the durable-file-store's ([durable-file-store](./durable-file-store.md))
  directory walk.
- Responsibility: turns one `engram ingest` invocation into the list of source files to index. It
  takes the named files, expands the named Pi session directories, resolves the sweep roots, both
  the `--sweep` directories and the declarative `--auto` roots with their per-repository
  configuration, walks each root under its exclusion rules, and tags every source with how it was
  found: the origin and, for an ancestor directory, its distance from the working directory.
- Current realization (informational): `gatherSources`, `explicitSources`, `piSessionSources`,
  `assembleSweepRoots`, `sweptSources`, `resolveAutoSpec`, `sessionDirFrom`, `sweepListerFrom`
  and `shouldSkipDir` in `internal/cli/ingest.go`; `sourceRefsFromRoot` and the origin constants in
  `internal/cli/ingest_dedup.go`; `SweepSpec`, `DefaultSweepSpec`, `LoadSweepSpec`,
  `ResolveSweepRoots` and `dropNonPersistentRoots` in `internal/cli/sweepspec.go`.
- Qualification: responsibility, finding the sources to index is stated without naming any
  function; ownership, the [chunk-index-store](./chunk-index-store.md) decides what to do with each source, while this
  component owns which files are sources and how each was found; independent verification, root
  resolution is tested as a pure function of the spec and an injected directory check, and
  gathering over fake listings; stability, the contract survives any change to the index;
  necessity, every source the index holds is one it found.
- Supported capabilities: [chunk-indexing](../product/chunk-indexing.md).

Rationale: the sweep is declarative and inspectable: what `--auto` covers is data, compiled in as
defaults and overridable per repository, so a developer tunes it without changing code. Two
throwaway-workspace rules exist because each reaches what the other cannot. Directory names catch
session-log project directories, which Claude Code names after the slugified working directory
(`-private-tmp-...`). Root paths catch a root that is itself throwaway, such as a repository root
that resolves to `/tmp/eval-pool/warm0` because it has no repository marker; the walk never checks
a root's own path against the exclusions, so such a root was once swept whole and permanently
indexed. Named files, `--sweep` and `extra_roots` are deliberate choices, so neither rule applies
to them.

## 792b17c Behavior Requirements

### 792b17c1 Named sources

Requirement: [792b17c1 Named sources](#792b17c1-named-sources) The ingestion-source-discovery MUST make each `--transcript` and `--markdown`
path a source with the explicit origin, without checking that it exists, and for each `--pi-sessions`
value that is a directory MUST make each `.md` and `.jsonl` file under it, outside its `jobs`,
`projects` and hidden subdirectories, a source with the explicit origin. It MUST skip a
`--pi-sessions` value that is not a directory.

Supports: [792a5b1a A named transcript and document are indexed](../product/chunk-indexing.md#792a5b1-uat-scenarios), [792a5b1b A Pi sessions directory is indexed without its scratch](../product/chunk-indexing.md#792a5b1-uat-scenarios), [792a5b1c A Pi sessions value that is not a directory is ignored](../product/chunk-indexing.md#792a5b1-uat-scenarios), [792a5b1d An unreadable named file fails the command](../product/chunk-indexing.md#792a5b1-uat-scenarios), [792a5f1d Explicit sources are indexed anywhere](../product/chunk-indexing.md#792a5f1-uat-scenarios), [792a5l1b A named file outranks the repository copy](../product/chunk-indexing.md#792a5l1-uat-scenarios).

#### 792b17c1a Scenarios

```gherkin
Scenario: 792b17c1a1 Named files and Pi directories are gathered together
  Given --transcript /docs/markdown.md and --pi-sessions /.pi holding session.jsonl
  When the sources are gathered and ingested
  Then both files are indexed

Scenario: 792b17c1a2 Every session file of each Pi directory is a source
  Given --pi-sessions naming a directory that holds two .jsonl files
  When the sources are gathered and ingested
  Then both files are indexed

Scenario: 792b17c1a3 A Pi value that is not a directory is skipped
  Given --pi-sessions /.pi, a directory holding session1.jsonl, and /nonexistent/pi
  When the sources are gathered and ingested
  Then session1.jsonl is indexed and the run succeeds
```

### 792b17c2 Swept directories

Requirement: [792b17c2 Swept directories](#792b17c2-swept-directories) The ingestion-source-discovery MUST make each `--sweep` directory a root with
the manual-sweep origin, the default excluded directory names ([792b17c5 Sweep configuration](#792b17c5-sweep-configuration)) and hidden directories
pruned, and no throwaway-name or throwaway-path rule.

Supports: [792a5c1a A swept directory's documents and transcripts are indexed](../product/chunk-indexing.md#792a5c1-uat-scenarios), [792a5f1d Explicit sources are indexed anywhere](../product/chunk-indexing.md#792a5f1-uat-scenarios).

#### 792b17c2a Scenarios

```gherkin
Scenario: 792b17c2a1 A swept directory under a throwaway path is still swept
  Given --sweep /private/tmp/eval-ws, which holds s.jsonl
  When the sources are gathered and ingested
  Then s.jsonl is indexed
```

### 792b17c3 Automatic roots

Requirement: [792b17c3 Automatic roots](#792b17c3-automatic-roots) For `--auto`, the ingestion-source-discovery MUST resolve, in this order and
each only when its setting is on: the repository root (the nearest of the working directory and its
ancestors holding a `.git`, `.hg` or `.jj` directory, else the working directory) with the
repository origin; each `.claude` directory from the working directory up to the filesystem root,
nearest first, with the claude-ancestor origin; each `.pi` directory on the same chain with the
pi-ancestor origin; the session-log directory ([792b17c6 Session-log directory](#792b17c6-session-log-directory)), when it is a directory, with the
session-log origin; and each `extra_roots` entry with the manual-sweep origin. Every root MUST carry
`exclude_dirs` and prune hidden directories unless `include_hidden_dirs` is set; a `.claude` root
MUST also carry `claude_exclude_dirs`, a `.pi` root only `jobs` and `projects`, and only the
session-log root `non_persistent_prefixes`. Each ancestor root MUST record its depth: the number of
directory levels climbed from the working directory to reach the directory holding it.

Supports: [792a5d1a The repository, its ancestors and the session logs are swept](../product/chunk-indexing.md#792a5d1-uat-scenarios), [792a5d1b Without a repository marker the working directory is the root](../product/chunk-indexing.md#792a5d1-uat-scenarios), [792a5d1c Harness scratch and state are not swept](../product/chunk-indexing.md#792a5d1-uat-scenarios), [792a5d1d ENGRAM_TRANSCRIPT_DIR names the session logs](../product/chunk-indexing.md#792a5d1-uat-scenarios), [792a5e1a A repository adds a root and turns one off](../product/chunk-indexing.md#792a5e1-uat-scenarios), [792a5f1a Sessions recorded in a throwaway directory are skipped](../product/chunk-indexing.md#792a5f1-uat-scenarios), [792a5l1a A repository file outranks its copy in an ancestor .claude directory](../product/chunk-indexing.md#792a5l1-uat-scenarios), [792a5l1c The nearer ancestor directory wins](../product/chunk-indexing.md#792a5l1-uat-scenarios).

#### 792b17c3a Scenarios

```gherkin
Scenario: 792b17c3a1 The repository, its ancestors and the session logs are roots
  Given the working directory /home/dev/proj/sub, a /home/dev/proj/.git, .claude directories in /home/dev/proj and /home/dev, and a session-log directory
  When the roots are resolved with the defaults
  Then the roots include /home/dev/proj, both .claude directories and the session-log directory, the .claude roots exclude projects, plugins and cache, and the repository root does not exclude projects

Scenario: 792b17c3a2 Without a marker the working directory is the repository root
  Given the working directory /home/dev/scratch with no marker in it or any ancestor
  When the roots are resolved with the defaults
  Then /home/dev/scratch is a root

Scenario: 792b17c3a3 A .claude root excludes jobs
  Given a repository at /home/dev/proj with a .claude directory
  When the roots are resolved with the defaults
  Then the .claude root excludes jobs

Scenario: 792b17c3a4 Only the session-log root carries the throwaway names
  Given a repository and a session-log directory
  When the roots are resolved with the defaults
  Then the session-log root's name prefixes include -private-tmp- and the repository root has none

Scenario: 792b17c3a5 Settings turn roots off
  Given repo_markdown, ancestor_claude_dirs and session_logs off and extra_roots ["/opt/notes"]
  When the roots are resolved
  Then /opt/notes is the only root

Scenario: 792b17c3a6 Ancestor depth is the physical distance
  Given a .pi directory in the working directory and the nearest .claude directory one level up
  When the roots are resolved
  Then the .pi root has depth 0 and the .claude root depth 1
```

### 792b17c4 Throwaway roots

Requirement: [792b17c4 Throwaway roots](#792b17c4-throwaway-roots) The ingestion-source-discovery MUST drop every `--auto` root, before
`extra_roots` are added, whose own path equals or lies under a `non_persistent_path_prefixes` entry,
comparing whole path segments, so that `/tmpfoo` is not under `/tmp`.

Supports: [792a5f1b A throwaway working directory is not swept](../product/chunk-indexing.md#792a5f1-uat-scenarios), [792a5f1c An ancestor directory under the macOS temp root is dropped](../product/chunk-indexing.md#792a5f1-uat-scenarios).

#### 792b17c4a Scenarios

```gherkin
Scenario: 792b17c4a1 Roots under a throwaway path are dropped
  Given the working directory /private/tmp/work/proj with no marker and a .claude directory in it
  When the roots are resolved with the defaults
  Then no root lies under /private/tmp

Scenario: 792b17c4a2 Extra roots are kept under a throwaway path
  Given every root setting off and extra_roots ["/tmp/mydata"], with the default path prefixes
  When the roots are resolved
  Then /tmp/mydata is the only root
```

### 792b17c5 Sweep configuration

Requirement: [792b17c5 Sweep configuration](#792b17c5-sweep-configuration) The ingestion-source-discovery MUST default every root setting on,
`exclude_dirs` to `node_modules`, `vendor`, `.git`, `.hg`, `.jj`, `dist`, `build`, `target`,
`.venv`, `venv`, `__pycache__`, `.next`, `.cache` and `.idea`, `claude_exclude_dirs` to `projects`,
`jobs`, `plugins`, `cache`, `todos`, `shell-snapshots`, `file-history`, `history`, `ide`,
`statsig`, `session-env`, `debug` and `worktrees`, `non_persistent_prefixes` to `-private-tmp-`,
`-tmp-`, `-var-folders-` and `-private-var-folders-`, `non_persistent_path_prefixes` to `/tmp`,
`/private/tmp`, `/var/folders` and `/private/var/folders`, and `extra_roots` to none. For `--auto`
it MUST read `.engram/sweep.json` at the repository root, overlay each key the file sets on the
defaults, and return an error naming the file when it is not valid JSON; it MUST return an error
when the working directory cannot be found.

Supports: [792a5c1a A swept directory's documents and transcripts are indexed](../product/chunk-indexing.md#792a5c1-uat-scenarios), [792a5d1c Harness scratch and state are not swept](../product/chunk-indexing.md#792a5d1-uat-scenarios), [792a5e1a A repository adds a root and turns one off](../product/chunk-indexing.md#792a5e1-uat-scenarios), [792a5e1b A malformed sweep file is refused](../product/chunk-indexing.md#792a5e1-uat-scenarios), [792a5f1a Sessions recorded in a throwaway directory are skipped](../product/chunk-indexing.md#792a5f1-uat-scenarios), [792a5f1b A throwaway working directory is not swept](../product/chunk-indexing.md#792a5f1-uat-scenarios).

#### 792b17c5a Scenarios

```gherkin
Scenario: 792b17c5a1 The defaults sweep everything and exclude dependency trees
  Given no sweep configuration
  When the default settings are read
  Then repo_markdown, ancestor_claude_dirs and session_logs are on, exclude_dirs holds node_modules, vendor and .git, and non_persistent_prefixes holds -private-tmp-, -tmp- and -var-folders-

Scenario: 792b17c5a2 A configuration file overlays the defaults
  Given {"repo_markdown": false, "exclude_dirs": ["node_modules", "gen"]}
  When the configuration is loaded
  Then repo_markdown is off, exclude_dirs is exactly node_modules and gen, and session_logs keeps its default

Scenario: 792b17c5a3 The repository's file is read by --auto
  Given a repository /repo whose .engram/sweep.json is {"session_logs": false}
  When --auto sources are gathered and ingested from /repo
  Then /repo/README.md is indexed and no session log is

Scenario: 792b17c5a4 An invalid configuration file is an error
  Given a repository whose .engram/sweep.json is not valid JSON
  When --auto sources are gathered
  Then the error names .engram/sweep.json
```

### 792b17c6 Session-log directory

Requirement: [792b17c6 Session-log directory](#792b17c6-session-log-directory) The ingestion-source-discovery MUST take the session-log directory from
`ENGRAM_TRANSCRIPT_DIR` when it is non-empty, else `<home>/.claude/projects`, and MUST use no
session-log directory when the home directory cannot be found.

Supports: [792a5d1a The repository, its ancestors and the session logs are swept](../product/chunk-indexing.md#792a5d1-uat-scenarios), [792a5d1d ENGRAM_TRANSCRIPT_DIR names the session logs](../product/chunk-indexing.md#792a5d1-uat-scenarios).

Rationale: memory learns from every recorded conversation of every project, not only the current
one's. A headless or eval run sets `ENGRAM_TRANSCRIPT_DIR` so it sweeps only its own sessions.

#### 792b17c6a Scenarios

```gherkin
Scenario: 792b17c6a1 The default is every project's sessions
  Given no ENGRAM_TRANSCRIPT_DIR and the home directory /home/u
  When the session-log directory is found
  Then it is /home/u/.claude/projects, and it is empty when the home directory cannot be found

Scenario: 792b17c6a2 The environment names the directory
  Given ENGRAM_TRANSCRIPT_DIR is /custom/sessions
  When the session-log directory is found
  Then it is /custom/sessions
```

### 792b17c7 Walking a root

Requirement: [792b17c7 Walking a root](#792b17c7-walking-a-root) The ingestion-source-discovery MUST walk each root and return its files,
pruning every directory below the root whose name is in the root's excluded names, starts with one
of its name prefixes, or, when the root prunes hidden directories, starts with `.`, and MUST return
an error naming the root when the walk fails.

Supports: [792a5b1b A Pi sessions directory is indexed without its scratch](../product/chunk-indexing.md#792a5b1-uat-scenarios), [792a5c1a A swept directory's documents and transcripts are indexed](../product/chunk-indexing.md#792a5c1-uat-scenarios), [792a5d1c Harness scratch and state are not swept](../product/chunk-indexing.md#792a5d1-uat-scenarios), [792a5f1a Sessions recorded in a throwaway directory are skipped](../product/chunk-indexing.md#792a5f1-uat-scenarios).

#### 792b17c7a Scenarios

```gherkin
Scenario: 792b17c7a1 Excluded and hidden directories are pruned
  Given a directory holding docs/real.md, node_modules/pkg/dep.md and .layer-run/cfg/junk.md
  When it is walked as a root excluding node_modules and pruning hidden directories
  Then docs/real.md is returned and neither of the others is

Scenario: 792b17c7a2 A throwaway-named directory is pruned
  Given the name prefixes -private-tmp- and -tmp-
  When the directories -private-tmp-cummatrix-x, node_modules and -Users-joe-repos-engram are checked
  Then the first two are pruned and the third is kept

Scenario: 792b17c7a3 A failed walk is an error
  Given a Pi sessions directory whose walk fails
  When the sources are gathered
  Then gathering fails
```

### 792b17c8 Source files

Requirement: [792b17c8 Source files](#792b17c8-source-files) The ingestion-source-discovery MUST keep only `.md` and `.jsonl` files of a
walked root, MUST drop, for every root but a `--pi-sessions` directory, each file under the chunk
index directory, and MUST tag each source with its root's origin and depth, as explicit only when
the origin is explicit.

Supports: [792a5c1a A swept directory's documents and transcripts are indexed](../product/chunk-indexing.md#792a5c1-uat-scenarios), [792a5c1c The chunk index directory is never indexed](../product/chunk-indexing.md#792a5c1-uat-scenarios), [792a5l1a A repository file outranks its copy in an ancestor .claude directory](../product/chunk-indexing.md#792a5l1-uat-scenarios).

#### 792b17c8a Scenarios

```gherkin
Scenario: 792b17c8a1 Only documents and transcripts are sources
  Given a root holding a.md, b.jsonl and c.json
  When its sources are gathered
  Then a.md and b.jsonl are sources and c.json is not

Scenario: 792b17c8a2 The chunk index is never a source
  Given a swept root that contains the chunk index directory
  When its sources are gathered
  Then no file under the chunk index directory is a source

Scenario: 792b17c8a3 A swept root's files are found
  Given --sweep / over a filesystem holding /.pi/session.jsonl
  When the sources are gathered and ingested
  Then /.pi/session.jsonl is indexed
```
