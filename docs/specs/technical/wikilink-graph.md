# Wikilink Graph - Technical Spec

## 792b13a Orientation

- Ownership: Repository component.
- Responsibility: the vault seen as a set of linked notes. It scans the notes at the vault root,
  keying each by its basename without `.md` and recording its leading Luhmann ID and the link
  targets in its text, and it resolves a reference a user gives for a note to the note it names.
  It builds the directed link graph, reads a note's in-degree for the [vault-counter](./vault-counter.md), and lists the
  links that resolve to no note for [vault-health](../product/vault-health.md).
- Current realization (informational): `internal/vaultgraph` (`ScanVault`, `ParseBasename`,
  `ParseWikilinks`, `BuildGraph`, `UnresolvedTargets`); `normalizeNoteRef` in
  `internal/cli/note_ref.go`; `findNote` in `internal/cli/resituate.go`. `BFSWithCap`,
  `InDegreeIn` and `UndirectedNeighbors` have no production caller.
- Qualification: responsibility, scanning and reference resolution are stated without naming any
  command; ownership, callers decide what to do with a note once found, while this component
  decides what a note's key is and which note a reference names; independent verification, the
  scan and the lookup are tested directly over a fake vault listing and a note list; stability, the
  basename key is the link form notes and Obsidian share, and it survives any refactoring;
  necessity, every command that reads or revises a named note finds it here.
- Supported capabilities: [vault-identity](../product/vault-identity.md), [note-revision](../product/note-revision.md), [memory-query](../product/memory-query.md), [vault-counting](../product/vault-counting.md)
  [vault-counting](../product/vault-counting.md), [vault-health](../product/vault-health.md),
  [vault-migration](../product/vault-migration.md).

Rationale: navigation lives in authored relations, the wikilinks in note text, rather than in a
separate graph store that could drift from the notes, so the graph is derived from the notes each
time and is always current. A note is keyed by its basename because that is what a wikilink names
and what Obsidian resolves.

## 792b13c Behavior Requirements

### 792b13c1 Vault scan

Requirement: [792b13c1 Vault scan](#792b13c1-vault-scan) The wikilink-graph MUST scan a vault by listing the `.md` files at its root,
ignoring subdirectories and other files, and MUST return one note per file, keyed by its filename
without `.md`, with the Luhmann ID that leads the filename (empty when there is none) and the
targets of the links in its text. A failure to list the vault or to read a file MUST be returned,
naming the path.

Supports: [792a2c1a A Markdown file in a subdirectory is not a note](../product/vault-identity.md#792a2c1-uat-scenarios), [792a2d1a A fresh vault has no notes](../product/vault-identity.md#792a2d1-uat-scenarios), [792a2d1b A user's README is a note](../product/vault-identity.md#792a2d1-uat-scenarios), [792a4a1a Every reference form finds the note](../product/note-revision.md#792a4a1-uat-scenarios), [792a4a1b An unknown reference is refused](../product/note-revision.md#792a4a1-uat-scenarios), [792a8e1i Only notes at the vault root link](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1j An unreadable note fails the report](../product/vault-counting.md#792a8e1-uat-scenarios), [792a9a1e An unreadable note stops the check](../product/vault-health.md#792a9a1-uat-scenarios).

#### 792b13c1a Scenarios

```gherkin
Scenario: 792b13c1a1 Root notes are scanned with their keys and links
  Given a vault whose root holds 1a.2026-05-09.note.md, linking [[2.other]], and notes.md
  When the vault is scanned
  Then 1a.2026-05-09.note has Luhmann ID 1a and the link target 2.other, and notes has no Luhmann ID

Scenario: 792b13c1a2 An empty vault has no notes
  Given a vault whose root holds no Markdown file
  When the vault is scanned
  Then no note is returned and no error

Scenario: 792b13c1a3 Only Markdown files are notes
  Given a vault whose listing holds a file that does not end in .md
  When the vault is scanned
  Then that file is not a note

Scenario: 792b13c1a4 A listing failure is returned
  Given a vault whose listing fails
  When the vault is scanned
  Then the error is returned, naming the vault path

Scenario: 792b13c1a5 A read failure is returned
  Given a vault one of whose notes cannot be read
  When the vault is scanned
  Then the error is returned, naming that note's path
```

### 792b13c2 Note references

Requirement: [792b13c2 Note references](#792b13c2-note-references) The wikilink-graph MUST resolve a note reference by trimming surrounding
whitespace, removing enclosing `[[` and `]]` and any `|display` part, and dropping a trailing
`.md`, and then returning the vault-relative path of the first scanned note whose basename or
Luhmann ID equals the result. When no note matches, it MUST fail with a not-found error quoting the
reference as given.

Supports: [792a4a1a Every reference form finds the note](../product/note-revision.md#792a4a1-uat-scenarios), [792a4a1b An unknown reference is refused](../product/note-revision.md#792a4a1-uat-scenarios).

Rationale: an agent copies a note's name from wherever it saw it, as a wikilink from a note's text,
a filename from a listing, or a Luhmann ID, so each form names the same note. The error quotes what
the user typed, not its normalized form, so the user can see which reference failed.

#### 792b13c2a Scenarios

```gherkin
Scenario Outline: 792b13c2a1 Every reference form names the note
  Given a scanned note 1.linting with Luhmann ID 1
  When the reference <ref> is resolved
  Then the result is 1.linting.md

  Examples:
    | ref             |
    | 1.linting       |
    | 1               |
    | 1.linting.md    |
    | [[1.linting]]   |

Scenario: 792b13c2a2 An unknown reference quotes what was given
  Given a scanned note 1.linting
  When the reference 99.missing.md is resolved
  Then it fails as not found, quoting "99.missing.md"
```

### 792b13c3 Link targets

Requirement: [792b13c3 Link targets](#792b13c3-link-targets) The wikilink-graph MUST list the targets of the links in a text: the text between
`[[` and the next `]]` on one line, each target once, in order of first appearance, after a trailing
`.md` is dropped. It MUST leave out a link whose target is empty, a link that would span a line break,
and every link inside a fenced code block, whose fence is a run of three or more backticks or tildes
and ends at a line of the same character, at least as long, followed only by whitespace, or at the end
of the text when none follows. It MUST keep a `|display` part as part of the target, and MUST NOT
treat brackets inside a target as nesting.

Supports: [792a7x1a A note is printed with its links](../product/memory-query.md#792a7x1-uat-scenarios), [792a7x1e A fenced link is not listed](../product/memory-query.md#792a7x1-uat-scenarios), [792a7x1h Each link is listed once](../product/memory-query.md#792a7x1-uat-scenarios), [792a8e1f A link inside a fenced code block does not count](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1g A link ending in .md counts as the bare link](../product/vault-counting.md#792a8e1-uat-scenarios), [792a9b1c A legacy suffixed link resolves](../product/vault-health.md#792a9b1-uat-scenarios), [792a9c1b A missing note is reported once however it is spelled](../product/vault-health.md#792a9c1-uat-scenarios), [792a9c1d A link inside a fenced code block is not reported](../product/vault-health.md#792a9c1-uat-scenarios).

Rationale: Obsidian does not resolve a link inside a fenced code block, so verbatim transcript text
pasted into a note must not make graph edges. The legacy `.md`-suffixed form
resolves to the same note as the bare basename, so both are one target. A bare link that ends in the
three characters `.md` is always read as the suffixed form, because the text cannot say which was
meant.

#### 792b13c3a Scenarios

```gherkin
Scenario: 792b13c3a1 A link is a target
  Given the text "see [[9o.2026-05-09.holistic-final-review]] for context."
  When its link targets are listed
  Then the result is 9o.2026-05-09.holistic-final-review

Scenario: 792b13c3a2 Targets are listed once in order of appearance
  Given the text "ref [[B]] then [[A]] then [[B]] then [[C]]", across lines or on one
  When its link targets are listed
  Then the result is B, A, C

Scenario: 792b13c3a3 A .md suffix is dropped
  Given the text "see [[x]] and [[x.md]] and [[y.md]] and [[0.md]]"
  When its link targets are listed
  Then the result is x, y, 0

Scenario: 792b13c3a4 Empty links give nothing
  Given empty text, text without links, and the text "empty [[]] valid [[Real]]."
  When their link targets are listed
  Then the first two give none and the last gives Real

Scenario: 792b13c3a5 A link cannot span lines
  Given the text "partial [[start" followed by a line break and "end]] not a link"
  When its link targets are listed
  Then the result is empty

Scenario: 792b13c3a6 Brackets do not nest
  Given the text "[[outer[[inner]]"
  When its link targets are listed
  Then the result is the single target outer[[inner

Scenario: 792b13c3a7 A fenced link is skipped
  Given the text "outside [[y]]" then a fenced block holding "inside [[x]]" then "after [[z]]"
  When its link targets are listed
  Then the result is y, z, whichever fence characters surround the block

Scenario: 792b13c3a8 An unclosed fence runs to the end
  Given the text "outside [[y]]" then a fence opener then "inside [[x]]" with no closer
  When its link targets are listed
  Then the result is y

Scenario: 792b13c3a9 A shorter fence does not close a longer one
  Given a four-backtick block that holds a three-backtick line and links on either side of it
  When its link targets are listed
  Then every link between the four-backtick lines is skipped and the links outside are kept

Scenario: 792b13c3a10 A display part stays in the target
  Given the text "see [[4.gamma|the gamma note]]"
  When its link targets are listed
  Then the result is 4.gamma|the gamma note
```

### 792b13c4 Link graph

Requirement: [792b13c4 Link graph](#792b13c4-link-graph) The wikilink-graph MUST build, from scanned notes, a directed graph with one
node per note, keyed by its basename, and one edge from a note to each distinct target among its
link targets that is the basename of a scanned note. It MUST leave out a link whose target is not a
scanned note and a link from a note to itself, and MUST collapse repeated links to the same target
into one edge.

Supports: [792a8e1d A repeated link counts once](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1e A note's link to itself does not count](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1h A link to a missing note counts for no note](../product/vault-counting.md#792a8e1-uat-scenarios).

Rationale: a wikilink names a note by its basename, and Obsidian resolves it the same way. A link
to no note is not a backlink of any note, and a note linking to itself is not evidence that anything
else refers to it.

#### 792b13c4a Scenarios

```gherkin
Scenario: 792b13c4a1 No notes make an empty graph
  Given no notes
  When the graph is built
  Then it has no nodes and no edges

Scenario: 792b13c4a2 A link to a missing note is dropped
  Given a note A that links B and MISSING, and a note B
  When the graph is built
  Then A has an edge to B and none to MISSING, and MISSING has in-degree 0

Scenario: 792b13c4a3 A link to itself is dropped
  Given a note A that links A and B, and a note B
  When the graph is built
  Then A has an edge to B and none to A

Scenario: 792b13c4a4 Edges never outnumber links, and in-degrees sum to out-degrees
  Given any set of notes
  When the graph is built
  Then the number of edges is at most the number of link targets, and the in-degrees of all notes sum to the number of edges

Scenario: 792b13c4a5 No node has an edge to itself
  Given any set of notes
  When the graph is built
  Then no note has an edge to itself
```

### 792b13c5 In-degree

Requirement: [792b13c5 In-degree](#792b13c5-in-degree) The wikilink-graph MUST report a note's in-degree as the number of distinct
notes with an edge to it ([792b13c4 Link graph](#792b13c4-link-graph)), and 0 for a basename that is not a node.

Supports: [792a8e1a A note with incoming links](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1b A note with no incoming links](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1c A basename that is not a note](../product/vault-counting.md#792a8e1-uat-scenarios).

#### 792b13c5a Scenarios

```gherkin
Scenario: 792b13c5a1 In-degree counts the linking notes
  Given A and B link C, C links A, and D is linked by none
  When the in-degrees are read
  Then C has 2, A has 1, and B and D have 0

Scenario: 792b13c5a2 An unknown basename has in-degree 0
  Given a built graph
  When the in-degree of a basename that is not a node is read
  Then it is 0
```

### 792b13c6 Unresolved links

Requirement: [792b13c6 Unresolved links](#792b13c6-unresolved-links) The wikilink-graph MUST list, for scanned notes, each link target of a note that
is neither that note's own basename nor the basename of a scanned note, as the linking note's
basename and the target, in note order and then in link order.

Supports: [792a9b1a A bare-ID link fails](../product/vault-health.md#792a9b1-uat-scenarios), [792a9b1c A legacy suffixed link resolves](../product/vault-health.md#792a9b1-uat-scenarios), [792a9c1a A link to a missing note warns](../product/vault-health.md#792a9c1-uat-scenarios), [792a9c1b A missing note is reported once however it is spelled](../product/vault-health.md#792a9c1-uat-scenarios), [792a9c1c A note's link to itself is not reported](../product/vault-health.md#792a9c1-uat-scenarios), [792a9c1e A vault with no dangling link prints no G3 line](../product/vault-health.md#792a9c1-uat-scenarios).

Rationale: the graph drops a link to no note without a sound ([792b13c4 Link graph](#792b13c4-link-graph)), so this is the list of the
edges it dropped. A vault that writes links in a form the graph does not resolve, such as a bare
Luhmann ID against a basename key, loses most of its edges without it.

#### 792b13c6a Scenarios

```gherkin
Scenario: 792b13c6a1 A link in the wrong form is listed
  Given a note 105.2026-05-30.foo that links 105, and a note 1a.2026-05-30.bar that links 105.2026-05-30.foo
  When the unresolved links are listed
  Then only 105.2026-05-30.foo linking 105 is listed

Scenario: 792b13c6a2 Self-links and resolved links are not listed
  Given a note A that links A and B, and a note B
  When the unresolved links are listed
  Then none is listed

Scenario: 792b13c6a3 A legacy suffixed link resolves
  Given a note A whose text links B.md, gone.md and missing, and a note B
  When the links are parsed and the unresolved links are listed
  Then gone and missing are listed and B is not
```
