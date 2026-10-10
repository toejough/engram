# Transcript Reader - Technical Spec

## 792b15a Orientation

- Ownership: Repository component, at the boundary of the harnesses' session-transcript file
  format (Claude Code and Pi JSONL).
- Responsibility: turns one session transcript file into conversation text. It reads the file,
  keeps the user and assistant entries, drops harness-injected noise, shortens oversized content,
  compresses each tool call and its result to one line, and returns the text with the timestamp of
  the last entry it kept.
- Session stores: engram reads the harnesses' transcripts as files, not through a harness API.
  Claude Code writes `~/.claude/projects/<slug>/*.jsonl`; Pi writes session JSONL under `.pi`
  directories, which [chunk-indexing](../product/chunk-indexing.md) sweeps ([792a5d Automatic sweep roots](../product/chunk-indexing.md#792a5d-automatic-sweep-roots)) or names with `--pi-sessions` ([792a5b Explicitly named sources](../product/chunk-indexing.md#792a5b-explicitly-named-sources)). Only
  JSONL is read.
- Current realization (informational): `JSONLReader.ReadFrom` in `internal/transcript/transcript.go`,
  over `StripWithConfigIndexed` with `ToolSummaryMode` and the shared cleaning helpers in
  `internal/context/strip.go`, `stripconfig.go` and `toolsummary.go`; chunk ingestion wires it in
  `newIngestDeps` (`internal/cli/ingest.go`). `ReadFrom` also takes a start time and a byte budget,
  which ingestion leaves unset; `SegmentsFrom`, `Strip` and the `KeepToolCalls` mode have no
  production caller.
- Qualification: responsibility, turning a transcript into conversation text is stated without
  naming any function; ownership, the caller decides which files to read and what to do with the
  text, while this component owns the file format and what is kept; independent verification, it is
  tested directly over a fake file reader and on raw transcript lines; stability, the contract
  survives any change to how the cleaning is split across packages; necessity, every transcript
  chunk is made from its output.
- Supported capabilities: [chunk-indexing](../product/chunk-indexing.md), and memory-query's [792a7p When a chunk was first indexed](../product/memory-query.md#792a7p-when-a-chunk-was-first-indexed) (the last timestamp).

Rationale: Claude Code and Pi both write JSONL, so one reader serves both and stripping and
emitting do not depend on the harness.

## 792b15c Behavior Requirements

### 792b15c1 Reading a transcript

Requirement: [792b15c1 Reading a transcript](#792b15c1-reading-a-transcript) The transcript-reader MUST read the whole transcript file at the given path
and return its conversation text, and MUST return an error wrapping the read failure as
`reading transcript` when the file cannot be read.

Supports: [792a5b1d An unreadable named file fails the command](../product/chunk-indexing.md#792a5b1-uat-scenarios), [792a5g1a A conversation is indexed as text with tool summaries](../product/chunk-indexing.md#792a5g1-uat-scenarios).

#### 792b15c1a Scenarios

```gherkin
Scenario: 792b15c1a1 An unreadable transcript is an error
  Given a file reader that fails with "file not found"
  When a transcript is read
  Then the error says reading transcript and file not found
```

### 792b15c2 Conversation entries

Requirement: [792b15c2 Conversation entries](#792b15c2-conversation-entries) The transcript-reader MUST keep only entries whose top-level `type`, or else
whose `message.role`, is `user` or `assistant`, MUST render each kept entry's text, given as a string
or as text content blocks joined by spaces, as one line prefixed `USER: ` or `ASSISTANT: `, and MUST
drop entries that are not JSON, have another type, or have no text.

Supports: [792a5g1a A conversation is indexed as text with tool summaries](../product/chunk-indexing.md#792a5g1-uat-scenarios), [792a5h1b A transcript is chunked by turns](../product/chunk-indexing.md#792a5h1-uat-scenarios).

#### 792b15c2a Scenarios

```gherkin
Scenario: 792b15c2a1 User and assistant text become prefixed lines
  Given a user entry "plain question" and an assistant entry "plain answer", each with string content
  When the entries are read
  Then the text is "USER: plain question" then "ASSISTANT: plain answer"

Scenario: 792b15c2a2 Other entries are dropped
  Given a non-JSON line, a progress entry, a user entry with empty content, and a user entry "real question"
  When the entries are read
  Then the only line is "USER: real question"

Scenario: 792b15c2a3 An entry with another role is dropped
  Given a user entry "hello", an entry whose role is toolResult, and an assistant entry "response"
  When the transcript is read
  Then the text holds "USER: hello" and "ASSISTANT: response" and nothing of the toolResult entry
```

### 792b15c3 Harness noise

Requirement: [792b15c3 Harness noise](#792b15c3-harness-noise) The transcript-reader MUST drop text that starts with `<system-reminder`,
`Base directory for this skill:`, `<local-command-stdout>`, `<local-command-caveat>` or
`<task-notification>`, MUST replace text holding a `<command-name>` slash-command block with the
trimmed text of its `<command-args>` block, and MUST drop that text when the block has no
arguments.

Rationale: the harness writes these into the user's turns. They are not what the user said, and
an injected skill body would otherwise repeat the whole skill in every chunk of every session that
used it.

Supports: [792a5g1a A conversation is indexed as text with tool summaries](../product/chunk-indexing.md#792a5g1-uat-scenarios).

#### 792b15c3a Scenarios

```gherkin
Scenario: 792b15c3a1 A system reminder is dropped
  Given a user entry whose text is a <system-reminder> block, and a user entry "real question"
  When the entries are read
  Then the only line is "USER: real question"

Scenario: 792b15c3a2 Injected skill bodies, command output and task notifications are dropped
  Given user entries starting "Base directory for this skill:", "<local-command-stdout>", "<local-command-caveat>" and "<task-notification>"
  When the entries are read
  Then none of them yields a line

Scenario: 792b15c3a3 A slash command keeps only its arguments
  Given a user entry holding <command-name>/learn</command-name> and <command-args>remember X</command-args>, and one holding a <command-name> block with no arguments
  When the entries are read
  Then the only line is "USER: remember X"
```

### 792b15c4 Oversized content

Requirement: [792b15c4 Oversized content](#792b15c4-oversized-content) The transcript-reader MUST replace every run of 100 or more base64 characters
(letters, digits, `+`, `/`, `=`) in an entry with `[base64 removed]`, and MUST cut any text line
longer than 2000 characters to its first 2000 followed by `[truncated]`.

Supports: [792a5g1a A conversation is indexed as text with tool summaries](../product/chunk-indexing.md#792a5g1-uat-scenarios).

#### 792b15c4a Scenarios

```gherkin
Scenario: 792b15c4a1 Base64 data is replaced
  Given a user entry whose text holds a 200-character base64 run
  When the entry is read
  Then its line holds [base64 removed] in place of the run

Scenario: 792b15c4a2 An oversized line is cut
  Given a user entry whose text is 3000 characters long
  When the entry is read
  Then its line, prefix included, is cut to its first 2000 characters followed by [truncated]
```

### 792b15c5 Tool-call summaries

Requirement: [792b15c5 Tool-call summaries](#792b15c5-tool-call-summaries) The transcript-reader MUST render each `tool_use` block that a later
`tool_result` block with string content answers by ID as one line,
`[tool] <name>(<key>=<value>, ...) → exit <0|1> | <first non-empty result line>`, at the position of
the result, with the arguments sorted by key and string values quoted, exit 1 when the result is an
error, the argument list and the result line each cut to 120 characters followed by `[truncated]`,
and the ` | ...` part left out when the result is empty. It MUST drop a `tool_use` that no such
result answers, and every other `tool_result`.

Supports: [792a5g1a A conversation is indexed as text with tool summaries](../product/chunk-indexing.md#792a5g1-uat-scenarios).

Rationale: a tool call's full input and output are usually the bulk of a session. One line keeps
what was done and whether it worked, which is what an agent reading the chunk back needs. A
`tool_result` whose content is a list of blocks matches no summary, so its call is dropped.

#### 792b15c5a Scenarios

```gherkin
Scenario: 792b15c5a1 A tool call and its result become one line
  Given an assistant entry "Let me read" with a Read tool_use of file_path /src/main.go, and its tool_result "package main"
  When the entries are read
  Then the lines are "ASSISTANT: Let me read" and a line starting "[tool] Read(" holding file_path="/src/main.go", exit 0 and package main

Scenario: 792b15c5a2 An error result exits 1
  Given a Bash tool_use and its tool_result "Error: test failed" marked as an error
  When the entries are read
  Then the one line starts "[tool] Bash(" and holds exit 1 and Error: test failed

Scenario: 792b15c5a3 Only the first output line is kept
  Given a tool_result whose content is "first line", "second line" and "third line" on separate lines
  When the entries are read
  Then the summary line holds first line and not second line

Scenario: 792b15c5a4 Several calls keep their order among the text
  Given a user entry, an assistant entry with text and a tool_use, its result, two more tool calls, and a closing assistant entry
  When the entries are read
  Then the text lines and the [tool] lines appear in the order of the entries

Scenario: 792b15c5a5 An unanswered tool call is dropped
  Given an assistant entry "Let me check" with a tool_use that no tool_result answers, and an assistant entry "Done."
  When the entries are read
  Then the lines are "ASSISTANT: Let me check" and "ASSISTANT: Done."

Scenario: 792b15c5a6 Long arguments and output are cut
  Given a tool_use whose arguments, and a tool_result whose output, are each over 120 characters
  When the entries are read
  Then the summary line holds [truncated] for each
```

### 792b15c6 Last timestamp

Requirement: [792b15c6 Last timestamp](#792b15c6-last-timestamp) The transcript-reader MUST return, with the conversation text, the timestamp of the
last entry it kept, treating an entry whose timestamp is absent, null or unreadable, or that is not
JSON, as carrying the timestamp of the entry before it, and MUST return no timestamp when no entry
carries a readable one.

Supports: [792a7p1b A transcript chunk takes the session's last timestamp](../product/memory-query.md#792a7p1-uat-scenarios).

Rationale: the timestamp becomes every chunk's first-indexed time ([792b18c14 First-indexed time](./chunk-index-store.md#792b18c14-first-indexed-time)), which orders the
newest chunks of a query ([792a7o The newest chunks](../product/memory-query.md#792a7o-the-newest-chunks)), so a session with a few rows that lack a time must still report
when it ended.

#### 792b15c6a Scenarios

```gherkin
Scenario: 792b15c6a1 Unreadable timestamps are skipped
  Given a transcript whose first rows are not JSON, lack a timestamp or carry "not-a-date", and whose last row is stamped 2026-01-01T00:03:00Z
  When it is read
  Then the last timestamp is 2026-01-01T00:03:00Z

Scenario: 792b15c6a2 A null timestamp inherits the previous one
  Given a transcript whose first row has a null timestamp and whose second is stamped 2026-01-01T00:01:00Z
  When it is read
  Then both rows are kept, the last timestamp is 2026-01-01T00:01:00Z and the read is not partial
```
