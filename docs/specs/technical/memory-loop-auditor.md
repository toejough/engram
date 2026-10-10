# Memory Loop Auditor - Technical Spec

## 792b39a Orientation

- Ownership: Repository component (dev tooling).
- Responsibility: reads a real agent transcript in context and records, moment by moment, where the loop of recall, action and
  learning broke. It splits the transcript into windows, has a model detect the moments in each (failures, corrections, rework,
  successes and dispatches), has a model judge each against the scorecard, checks point-in-time whether a relevant memory
  existed in the vault as it was at that moment, reads back what a forked subagent inherited, and derives draft outcome
  records for a person to act on. It never writes a note.
- Current realization (informational): `dev/eval/audit/audit_moments.py` (`split_into_windows`, `detect_moments`,
  `judge_moment`, `audit_transcript`, the point-in-time check helpers, `_build_engram_query_argv`,
  `_run_engram_query_at_moment`, `_resolve_fork_parent_context`) and `dev/eval/audit/generate_outcome_records.py`. The mechanical
  extractor (`extract.py`), the full-corpus runner (`run_audit.py`), the scorecard (`build_scorecard.py`) and the parent-capture
  and re-judging scripts of the same directory are not specified here.
- Qualification: responsibility, the windows, the detection, the record, the existence check and the outcome records are stated
  without naming a function; ownership, it owns how a transcript becomes records, while the model call it makes, the vault
  and the chunk index it searches, and the engram binary it queries are others'; independent verification, every requirement is
  tested with a stub in place of the model and a scratch vault and chunk index, so no test makes a model call; stability, the
  contract survives a change of model or of window size; necessity, without it the maintainer has no measurement of where the
  memory loop loses what it should have kept.
- Supported capabilities: [memory-effectiveness-evaluation](../product/memory-effectiveness-evaluation.md).

Rationale: engram is meant to close the loop that lets a fixed-weight model learn from experience, and the audit is
the step that looks back and labels what happened a win or a loss. It is a standing instrument, built Python-first
under `dev/eval`, with one run mode built (observe, over real transcripts); replay and mine, which re-score a
candidate change against recorded moments, are planned. Its questions are the loss stages of the loop, asked for
failures and successes alike and for dispatches as a moment of their own: whether a memory existed, whether a search
fired and targeted the right thing, whether the memory surfaced and was applied, and, on the write side, whether a
learn fired and wrote a good note. Credit is judged-only: the audit emits records and nothing in the vault changes
unjudged, so there is no automatic decay, promotion or prune.

## 792b39c Behavior Requirements

### 792b39c1 Windows

Requirement: [792b39c1 Windows](#792b39c1-windows) The memory-loop-auditor MUST return a transcript file of at most `max_kb` (150 KB) as a single window, and MUST split a
larger file into windows of at most `max_kb` at line boundaries, never inside a line, with the single-window threshold the
same number and not a larger one. Each window MUST carry its lines exactly as they are in the file, its path, its first and
last line locations counted from one in the file, its index among the windows and their total, and whether it is a single line larger than the cap. A single line larger than the cap MUST be detected in the
window that holds it.

Supports: [792a23k1a A subtle correction is detected](../product/memory-effectiveness-evaluation.md#792a23k1-uat-scenarios), [792a23k1b A success moment is detected](../product/memory-effectiveness-evaluation.md#792a23k1-uat-scenarios).

Rationale: a transcript that qualifies for a single window but exceeds the cap produces a prompt that `claude -p`
rejects instantly and deterministically (`prompt_too_long`); pinning the threshold to the cap closes the gap between
"too big to read whole" and "not big enough to split".

#### 792b39c1a Scenarios

```gherkin
Scenario: 792b39c1a1 A small file is one window
  Given a transcript at or just under the cap
  When it is split
  Then it is one window

Scenario: 792b39c1a2 The lines are preserved as they are
  Given a transcript
  When it is split
  Then each window's lines equal the file's lines unchanged

Scenario: 792b39c1a3 A window carries its fields
  Given a window
  When it is read
  Then it holds its path, its first and last line locations, its lines, its index and the total, and its oversized-line flag

Scenario: 792b39c1a4 Locations count from one
  Given a transcript
  When it is split
  Then the first window starts at line 1 and the locations are contiguous across windows

Scenario: 792b39c1a5 A larger file is split at line boundaries
  Given a transcript larger than the cap, including one in the former gap between the cap and the old threshold
  When it is split
  Then it becomes several windows of at most the cap and no line is cut

Scenario: 792b39c1a6 An oversized line is detected
  Given a transcript with a line larger than the cap in a middle window
  When it is split
  Then the oversized line is detected in the window that holds it
```

### 792b39c2 Detecting moments

Requirement: [792b39c2 Detecting moments](#792b39c2-detecting-moments) The memory-loop-auditor MUST ask a model to read a window, with its lines numbered by their absolute location in the
file, and to return the moments it finds as failures, corrections, rework, successes and dispatches, each with its location, and
MUST flag generously. It MUST read the window with the faster model first and re-read it with the stronger one, merging the
candidates by location, when the first returns fewer than the low-yield threshold (2). It MUST parse the model's JSON reply
tolerating a code fence around it, and a reply whose content is not JSON MUST yield no candidates. At the level of the `claude -p` response, a reply flagged as an error at a cost under 0.02 USD, and a reply that is not JSON, MUST be treated as transient failures and retried on the backoff sequence, and a flagged error that cost more MUST be returned as it is. It MUST raise `ExhaustedRetriesError` when every attempt of the
backoff sequence fails, so that a sustained outage never looks like a run of clean zero-moment windows, and a window that
genuinely yields no candidate MUST NOT raise. It MUST raise `PromptTooLongError` at once, without retrying, when the model
answers `prompt_too_long`, and MUST still retry a genuine transient error. A stub in place of the model MUST make no subprocess call.

Supports: [792a23k1a A subtle correction is detected](../product/memory-effectiveness-evaluation.md#792a23k1-uat-scenarios), [792a23k1b A success moment is detected](../product/memory-effectiveness-evaluation.md#792a23k1-uat-scenarios).

Rationale: detection that matched keywords would miss most subtle corrections ([792a23k The audit detects moments by
reading them](../product/memory-effectiveness-evaluation.md#792a23k-the-audit-detects-moments-by-reading-them)). A
sustained auth or API outage would otherwise produce many windows of clean-looking zero-moment results, and a
deterministic failure such as an oversized prompt would burn a backoff for nothing, so each failure has its own
exception.

#### 792b39c2a Scenarios

```gherkin
Scenario: 792b39c2a1 Moments are detected from a window
  Given a window and a model stub that returns candidates of each moment type
  When the moments are detected
  Then each candidate carries its type and absolute location

Scenario: 792b39c2a2 A low-yield window is re-read with the stronger model
  Given a window for which the first model returns fewer candidates than the threshold
  When the transcript is audited
  Then the stronger model reads it too and the candidates are merged by location

Scenario: 792b39c2a3 Lines are numbered by absolute location
  Given a window that starts mid-file
  When the detection and judgment prompts are built
  Then each line carries its absolute number and the numbers round-trip to the file's lines

Scenario: 792b39c2a4 A reply parses whether JSON, an error flag or garbage
  Given a model reply that is valid JSON, one flagged is_error, and one that is not JSON
  When each is parsed
  Then the first yields its result and cost, the second is retried until a good reply, and the third is retried and raises ExhaustedRetriesError when every attempt fails

Scenario: 792b39c2a5 Exhausted retries raise
  Given a model call that fails on every attempt
  When moments are detected, and again when the call is made by the retry helper
  Then ExhaustedRetriesError is raised and is distinguishable from a genuine empty result

Scenario: 792b39c2a6 A genuine zero is not an error
  Given a model that genuinely finds no candidate
  When moments are detected
  Then no error is raised

Scenario: 792b39c2a7 A prompt that is too long fails at once
  Given a model answer of prompt_too_long
  When moments are detected, with the real helper and with a stub
  Then PromptTooLongError is raised without a retry

Scenario: 792b39c2a8 A transient error still retries
  Given a transient model failure
  When the call is made
  Then it retries as before

Scenario: 792b39c2a9 A stub makes no subprocess call
  Given a stub in place of the model
  When moments are detected and judged
  Then no subprocess is run
```

### 792b39c3 The moment record

Requirement: [792b39c3 The moment record](#792b39c3-the-moment-record) The memory-loop-auditor MUST ask a model to judge each candidate moment and MUST return a record that carries the
moment's stable identifier (derived from its transcript and location), its type, description, timestamp (read from the window), location,
transcript, repository, harness, role, evidence pointer and instrument version (2), the finding-side answers, the writing-side answers, the handoff answers and the
failure category, joined with the structural events the extractor found. A question that the model answers "not applicable"
MUST leave its field null, carry a one-sentence rationale and be stamped with `judged_not_applicable` as its null reason, for
every conditional field of the finding side, the writing side and the failure category, and the model MUST be asked for a rationale
for every answered question, a malformed rationale object becoming an empty one. The judgment prompt MUST give rework moments the
categorization guidance, give success moments the capture-reachability variant and keep runbook 824's wording for failures. The fields that
need the search phrases MUST be null when there are none. A reply with a retry budget spent MUST raise `ExhaustedRetriesError`, a reply
of `prompt_too_long` MUST raise `PromptTooLongError`, and the whole pipeline MUST run with stubs and make no subprocess call.

Supports: [792a23l1a A complete moment record](../product/memory-effectiveness-evaluation.md#792a23l1-uat-scenarios), [792a23k1b A success moment is detected](../product/memory-effectiveness-evaluation.md#792a23k1-uat-scenarios).

Rationale: a bare null is ambiguous between "not asked" and "not applicable", so the conditional answers are stated,
with a reason, and the record says which instrument produced it so that before-and-after comparisons declare their
sides.

#### 792b39c3a Scenarios

```gherkin
Scenario: 792b39c3a1 A candidate is judged into a record
  Given a candidate and a model stub that returns a judgment
  When the moment is judged
  Then a record is returned

Scenario: 792b39c3a2 A record has every field of the scorecard
  Given a judged moment
  When the record is read
  Then it holds every top-level and every finding-side, writing-side and handoff field of the scorecard

Scenario: 792b39c3a3 Deferred fields are null without search phrases
  Given a judgment that supplies no search phrases
  When the record is built
  Then the fields that need them are null

Scenario: 792b39c3a4 The moment identifier is stable
  Given the same transcript and location
  When the identifier is derived twice
  Then it is the same

Scenario: 792b39c3a5 The timestamp is read from the window
  Given a window whose lines carry timestamps
  When a moment is judged
  Then the record's timestamp is the one at its location

Scenario: 792b39c3a6 A not-applicable answer is stamped
  Given a judgment that answers not applicable for followed, for conditional finding and writing fields, and for the failure category
  When the record is built
  Then each field is null with a judged_not_applicable reason and a rationale, and real answers leave no null reason

Scenario: 792b39c3a7 The prompt asks for not-applicable and rationales
  Given the judgment prompt
  When it is read
  Then it offers not_applicable for the conditional fields with the ban on bare nulls and asks for a rationale object, and malformed rationales become an empty object

Scenario: 792b39c3a8 The record carries the instrument version
  Given a judged moment
  When the record is read
  Then it carries instrument_version 2 and the module constant is 2

Scenario: 792b39c3a9 The prompt is scoped by moment type
  Given a rework moment, a success moment and a failure moment
  When their prompts are built
  Then rework gets the categorization guidance, success the capture-reachability variant, and failure keeps runbook 824's wording

Scenario: 792b39c3a10 The record reflects the extracted events
  Given extracted structural events for the transcript
  When a moment is judged
  Then the record's structural fields come from them

Scenario: 792b39c3a11 The whole pipeline runs on stubs
  Given the fixture transcripts and stubs for detection and judgment
  When the transcript is audited
  Then it returns JSON-serializable records, makes no subprocess call, and works for the main-session fixture

Scenario: 792b39c3a12 Judgment failures raise their own errors
  Given a judgment call that exhausts its retries, and another that answers prompt_too_long
  When the moment is judged, with the real helper and with a stub
  Then ExhaustedRetriesError and PromptTooLongError are raised respectively
```

### 792b39c4 Existence as of the moment's time

Requirement: [792b39c4 Existence as of the moment's time](#792b39c4-existence-as-of-the-moments-time) The memory-loop-auditor MUST judge whether a relevant memory existed by running `engram query` in the sandbox of
[792b38c3 The isolated environment of a trial](./eval-trial-harness.md#792b38c3-the-isolated-environment-of-a-trial) against the vault as of the moment's timestamp T. When the vault's git history has a commit at or before T and the flat-vault
migration's commit (2026-06-12) is an ancestor of it, it MUST query a detached worktree at that commit, remove the worktree afterwards even when the query fails, prune a dangling registration, and
label the answer DERIVED. Otherwise (the ESTIMATE branch) it MUST NOT run `engram query`: it MUST take the notes of the live vault whose creation date is at or before T, a note with no creation date included, report that a memory existed when any of them contains any of the moment's search phrases (case-insensitively), and label the answer ESTIMATE. That branch records no top five and no grade. It MUST query a copy of the chunk index that holds only the records ingested at or before T, excluding records with no
ingestion date, and label the chunk answer DERIVED. When it ran the query, it MUST capture the top five items on the record and grade whether any is
relevant to the moment (`relevant_top1`, `relevant_in_top5` or `nothing_relevant`) with a follow-up call, skipping that call when
no item came back and leaving the grade null for a reply that is not one of the three. It MUST NOT write to the live vault.

Supports: [792a23m1a An exact answer via git checkout](../product/memory-effectiveness-evaluation.md#792a23m1-uat-scenarios), [792a23m1b An approximate answer without history](../product/memory-effectiveness-evaluation.md#792a23m1-uat-scenarios), [792a23m1c The live vault is untouched](../product/memory-effectiveness-evaluation.md#792a23m1-uat-scenarios), [792a23m1d Chunks are always judged by ingestion date](../product/memory-effectiveness-evaluation.md#792a23m1-uat-scenarios).

Rationale: a memory written after a moment cannot have been missed at it. A worktree is a read-only operation on the
vault's git registry, so the live working tree is never touched. A relevance grade, rather than counting any returned
item, is what makes a returned-but-irrelevant item count as a miss.

#### 792b39c4a Scenarios

```gherkin
Scenario: 792b39c4a1 An empty scratch vault returns nothing
  Given an isolated engram environment over an empty vault and chunk index
  When the query runs
  Then it returns zero results, which proves the sandbox

Scenario: 792b39c4a2 The live vault is unchanged
  Given a point-in-time check run against a live vault
  When it completes
  Then the vault's fingerprint is unchanged

Scenario: 792b39c4a3 The worktree is removed when the query fails
  Given a worktree checkout and an engram query that fails
  When the check ends
  Then the worktree directory and its registration are gone

Scenario: 792b39c4a4 Chunks are filtered by ingestion date
  Given chunk records ingested before and after T and records with no ingestion date
  When the chunk index is filtered
  Then only the records ingested at or before T remain and the legacy records are excluded

Scenario: 792b39c4a5 Derived and estimate are told apart by the migration boundary
  Given a timestamp at or after the flat-vault migration and another before it
  When the existence check chooses its branch
  Then the first uses a git checkout and is DERIVED, and the second uses creation dates and is ESTIMATE

Scenario: 792b39c4a6 The estimate branch still has real content
  Given a real timestamp before the migration
  When the existence check runs
  Then the ESTIMATE branch returns the notes created by then

Scenario: 792b39c4a7 The check runs with the moment's search phrases
  Given search phrases written for the moment
  When the point-in-time check runs
  Then the query is made with those phrases against the vault of that time

Scenario: 792b39c4a8 The top five are captured and graded
  Given a query that returns items, one that returns none, and a grading reply that is not one of the three grades
  When the existence check is recorded
  Then the top five items and their grade are on the record, no grading call is made for the empty result, and an invalid grade leaves the grade null
```

### 792b39c5 The existence check's query

Requirement: [792b39c5 The existence check's query](#792b39c5-the-existence-checks-query) The memory-loop-auditor MUST build its `engram query --lazy-chunks` invocation with each phrase as one `--phrase=<value>`
argument, so that no value, one beginning with `--` included, can be read as a new flag, and the diagnostic rerun of the window
sampler MUST use the same builder. It MUST run the query with a 60-second timeout in the isolated environment and return
`items`, the raw output and an `error`: a failure to run, a timeout or a missing binary as an `error` with `items` null, a
non-zero exit as an `error` that contains both the standard output and the standard error, and an empty result as an empty list with no error. A judged moment MUST record a query error and an empty result
distinctly.

Supports: [792a23n1a A phrase starting with -- does not break the invocation](../product/memory-effectiveness-evaluation.md#792a23n1-uat-scenarios), [792a23n1b A real query failure's error text is preserved](../product/memory-effectiveness-evaluation.md#792a23n1-uat-scenarios).

Rationale: `engram`'s own errors print to standard output and a phrase beginning with `--` is parsed as a flag, so the
check would record a failure with an empty error; there is one builder so the copies cannot drift apart.

#### 792b39c5a Scenarios

```gherkin
Scenario: 792b39c5a1 A phrase is one argument
  Given search phrases
  When the argument vector is built
  Then each phrase is a single --phrase=<value> argument

Scenario: 792b39c5a2 A phrase starting with -- cannot break the invocation
  Given a phrase that begins with --
  When the vector is built, and the sampler's diagnostic rerun is built
  Then the phrase is one equals-joined argument in both

Scenario: 792b39c5a3 A real error on standard output is preserved
  Given an engram query that exits non-zero and writes its message to standard output
  When the check returns
  Then its error field contains that message

Scenario: 792b39c5a4 The result shape tells failure from emptiness
  Given a failing subprocess, a timeout and a successful query with no results
  When each runs
  Then the first two return items null with an error and the third an empty list with no error

Scenario: 792b39c5a5 A judged moment records a query error and an empty result
  Given a moment whose existence query errors, and another whose query returns nothing
  When each is judged
  Then the first records the error and the second records an empty result
```

### 792b39c6 What a forked subagent inherited

Requirement: [792b39c6 What a forked subagent inherited](#792b39c6-what-a-forked-subagent-inherited) The memory-loop-auditor MUST, for a forked subagent, read the fork-context line that starts its transcript to
find the parent transcript and the point in it the fork branched from, find the last recall that happened at or before that
point, and read back the vault notes it matched, skipping a note whose file is gone and keeping the rest. It MUST report why it
cannot resolve (not a fork transcript, the parent transcript is gone, the branch point is not in the parent) and why the notes are
empty (no recall before the fork point). The judgment prompt of a fork MUST include the inherited recall content, and the prompt of
any other role, and the prompt of a fork whose context cannot be resolved, MUST include no inherited-context section.

Supports: [792a23o1a An injected memory counts as found for the subagent](../product/memory-effectiveness-evaluation.md#792a23o1-uat-scenarios).

Rationale: a fork inherits its parent's whole conversation, so the memories recalled there count as found for it, but the model that
judges the fork is shown only the fork's own window, which holds none of it.

#### 792b39c6a Scenarios

```gherkin
Scenario: 792b39c6a1 The last recall before the fork point is found
  Given a fork whose parent transcript holds several recalls
  When the context is resolved
  Then the last recall at or before the branch point supplies the notes

Scenario: 792b39c6a2 An early branch point has no recall
  Given a fork that branched before any recall
  When the context is resolved
  Then it is available with no notes and says there was no recall before the fork point

Scenario: 792b39c6a3 A missing parent transcript is reported
  Given a fork whose parent transcript is gone
  When the context is resolved
  Then it is unavailable with that reason

Scenario: 792b39c6a4 A transcript that is not a fork is reported
  Given a transcript that does not start with a fork-context line
  When the context is resolved
  Then it is unavailable as not a fork transcript

Scenario: 792b39c6a5 A deleted note is skipped and the rest are kept
  Given a recall that matched a note whose file is gone and another that exists
  When the context is resolved
  Then only the existing note is returned

Scenario: 792b39c6a6 A fork's prompt includes the inherited recall
  Given a fork role with resolvable context
  When its judgment prompt is built
  Then it includes the inherited recall content

Scenario: 792b39c6a7 A non-fork prompt has no such section
  Given any other role
  When its judgment prompt is built
  Then it has no inherited-context section

Scenario: 792b39c6a8 An unresolvable fork has no dangling section
  Given a fork whose context cannot be resolved
  When its judgment prompt is built
  Then it has no inherited-context section
```

### 792b39c7 A dispatch prompt's memories

Requirement: [792b39c7 A dispatch prompt's memories](#792b39c7-a-dispatch-prompts-memories) The memory-loop-auditor MUST, for a fresh, workflow or Pi subagent whose dispatching parent's dispatch event is given, scan the
dispatch prompt for vault notes and record the ones it finds in `memories_in_dispatch_prompt`, count them as a search that ran
(injected) and as surfaced for the judgment, and MUST leave `memories_orchestrator_had_but_left_out` null: the orchestrator-side gap is
measured outside the per-moment record.

Supports: [792a23o1a An injected memory counts as found for the subagent](../product/memory-effectiveness-evaluation.md#792a23o1-uat-scenarios), [792a23o1b A dispatch moment measures the handoff on the orchestrator's side](../product/memory-effectiveness-evaluation.md#792a23o1-uat-scenarios).

Rationale: a memory in the prompt counts as found for the subagent, and what the orchestrator left out is measured
from its side, at the dispatch, so the per-transcript runner, which cannot see the parent, leaves the field to the
full-corpus work.

#### 792b39c7a Scenarios

```gherkin
Scenario: 792b39c7a1 A fresh subagent's injected memories count as found
  Given a fresh subagent and its parent's dispatch event whose prompt holds vault note content
  When a later moment is judged
  Then memories_in_dispatch_prompt lists them, the search is recorded as injected and the memory as surfaced, and the left-out field is null
```

### 792b39c8 Outcome records

Requirement: [792b39c8 Outcome records](#792b39c8-outcome-records) The memory-loop-auditor MUST derive draft outcome records from finished moment records and the extractor's events, as plain data files and
deterministically, with each record `{moment_id, note_ref, situation_text, outcome, ts, evidence{transcript, anchor}}`. For a success, failure or rework moment whose
`followed` answer is set and whose search ran, it MUST emit one record per distinct note found (a fork's inherited recall for an injected search, otherwise the last
recall at or before the moment) tagged `applied-helped` when followed is yes, `applied-hurt` when contradicted, and `injected-unused` for an
injected search or `surfaced-ignored` otherwise when it was ignored, partly followed, skipped or mis-ordered, or one record with a null note reference when no note resolves. For a failure or rework
moment it MUST emit an `absent-needed` record when no memory existed, or one existed and did not surface. It MUST make no LLM call and MUST NOT write, amend, activate, supersede or remove any vault note. The command-line entry point reads `results/moments.jsonl` and `results/transcript-events.jsonl` and writes `results/outcome-records.jsonl`, and it refuses (by assertion) a corpus other than the 150 moments and 568 transcript-event records it was written for; the derivation function itself takes any moment records and events.

Supports: [792a23p1a Output only, no side effects](../product/memory-effectiveness-evaluation.md#792a23p1-uat-scenarios).

Rationale: acting on the records takes a human judgment, so the audit labels and a person acts. Determinism means the same inputs
reproduce the same file byte for byte.

#### 792b39c8a Scenarios

```gherkin
Scenario: 792b39c8a1 A followed memory is applied-helped and a contradicted one applied-hurt
  Given a success moment with followed yes and a failure moment with followed contradicted, each with a resolved recalled note
  When the records are derived
  Then they are tagged applied-helped and applied-hurt, one per distinct note

Scenario: 792b39c8a2 An ignored memory is injected-unused or surfaced-ignored
  Given an ignored memory after an injected search and another after a recall
  When the records are derived
  Then the first is injected-unused and the second surfaced-ignored

Scenario: 792b39c8a3 A search that never ran gives no Rule 1 record
  Given a moment whose search_ran is none
  When the records are derived
  Then no applied, ignored or unused record is emitted for it

Scenario: 792b39c8a4 An absent or unsurfaced memory is absent-needed
  Given a failure moment with no existing memory, and another whose memory existed and did not surface
  When the records are derived
  Then each yields an absent-needed record naming DERIVED or ESTIMATE

Scenario: 792b39c8a5 The derivation is deterministic and writes no note
  Given the same moment records and events
  When the records are derived twice
  Then the files are identical, no model is called and no vault note changes
```
