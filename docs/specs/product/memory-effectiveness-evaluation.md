# Memory Effectiveness Evaluation - Product Spec

Joe, the maintainer, measures whether engram's memory helps an agent, and the tools he uses are dev tooling in this
repository and not part of the shipped command line: `targ eval <arm> [trials]` runs the Go harness under `dev/eval` for the
arms `nothing`, `skills-only` and `current-state` (the arm decides which skills and whether the binary is reachable), the
Python drivers under `dev/eval` run headless trials of a prompt, a skill and a vault (cold or warm) and score what the agent
did, and the memory-loop audit under `dev/eval/audit` reads real transcripts and records, moment by moment, where the loop of
recall, action and learning broke. This capability is what those tools promise to the person who relies on their numbers: a
trial measures what it says it measures, never reaches the maintainer's real vault, is scored as it behaved, and an audit
records what it saw without changing a note. It is a capability of the repository, not of an engram user's vault, which is
why the expected user is the maintainer.

## 792a23a A warm trial carries the skills it claims

Requirement: [792a23a A warm trial carries the skills it claims](#792a23a-a-warm-trial-carries-the-skills-it-claims) Any eval code that builds a warm trial configuration (one with skills and memory installed) MUST confirm,
for each skill it installs, that the skill's source directory exists and holds a `SKILL.md` before it installs the skill, and
MUST confirm that `SKILL.md` exists at the destination after the copy. It MUST NOT rely on the copy's own success: a copy
from a wrong source path is indistinguishable from a successful copy of an empty directory unless the source and the
destination are checked independently.

Rationale: a warm-config builder that silently skips its copy leaves warm trials running with no skills, to be read as
warm results. The check is shared (`isolation.install_skill`) so that there is one place left to get it wrong.

### 792a23a1 UAT Scenarios

```gherkin
Scenario: 792a23a1a Successful skill installation is verified
  Given the recall and learn skill sources exist with their SKILL.md files present
  When a warm trial configuration is built
  Then both skills' SKILL.md files exist under the configuration's skills directory after the call returns

Scenario: 792a23a1b Post-copy destination content is confirmed
  Given a warm trial configuration being built
  When the skill copy step completes
  Then SKILL.md is confirmed at the destination of each installed skill before the build returns successfully

Scenario: 792a23a1c Every warm-configuration builder verifies the same way
  Given the three Python warm-configuration builders of the trap, cumulative and probe harnesses
  When each is called with real skill sources present
  Then each leaves both skills' SKILL.md files under its skills directory
```

## 792a23b A missing or empty skill source stops the build

Requirement: [792a23b A missing or empty skill source stops the build](#792a23b-a-missing-or-empty-skill-source-stops-the-build) When a named skill's source directory does not exist, or exists and holds no `SKILL.md`, the construction of a
warm trial configuration MUST raise an error that names the skill and the path it checked. It MUST NOT log a warning and
continue, MUST NOT produce a configuration with fewer skills than requested, and MUST NOT leave a partial configuration that
claims success. This holds for every function that builds a warm configuration.

Rationale: a silent skip makes a warm arm indistinguishable from a cold one in the results, which is the failure the previous
requirement closes.

### 792a23b1 UAT Scenarios

```gherkin
Scenario: 792a23b1a A missing skill source raises
  Given a named skill whose source directory does not exist at the expected path
  When a warm trial configuration is built
  Then the call raises an error that names the skill and the path checked

Scenario: 792a23b1b An empty skill source raises
  Given a named skill whose source directory exists but holds no SKILL.md
  When a warm trial configuration is built
  Then the call raises an error that names the skill and the invalid source path

Scenario: 792a23b1c Every builder raises on a missing source
  Given the cumulative harness's and the probe harness's configuration builders and a skill source that is missing
  When each is called
  Then each raises an error naming the skill and the path, and does not skip the copy
```

## 792a23c Skill sources are found from the harness's own location

Requirement: [792a23c Skill sources are found from the harness's own location](#792a23c-skill-sources-are-found-from-the-harnesss-own-location) The eval harness MUST compute the repository root it uses to locate skill sources from the calling module's own
file location, and MUST NOT hard-code a machine- or session-specific absolute path for it. The skill sources are
`agent-instructions/skills/<skill>` relative to that root.

Rationale: a hard-coded absolute path is the fragility that lets a silent skip go unnoticed, in two independent
functions that share one contract.

### 792a23c1 UAT Scenarios

```gherkin
Scenario: 792a23c1a The path resolves independent of the working directory
  Given the eval harness computing the repository root that locates skill sources
  When the root is computed from a different working directory
  Then it is derived from the calling file's own location, with no hard-coded absolute path in the source

Scenario: 792a23c1b The cumulative harness resolves its path the same way
  Given the cumulative harness computing the path to the recall and learn skill sources
  When the path is computed
  Then it is agent-instructions/skills/<skill> under the root computed from the harness file's own location
```

## 792a23d A trial that reaches the real vault is detected from the trial's side

Requirement: [792a23d A trial that reaches the real vault is detected from the trial's side](#792a23d-a-trial-that-reaches-the-real-vault-is-detected-from-the-trials-side) Any harness that spawns isolated eval trials and guards against a trial reaching the maintainer's real vault MUST
detect a leak by checking trial-side evidence, the isolation state each trial's environment actually carried, and MUST NOT
use a before-and-after fingerprint of the whole real vault as its only mechanism. The evidence MUST be captured
automatically by the same primitive that builds a trial's isolated environment, at the moment it builds it, so that a harness
never has to collect, return or pass along each trial's environment itself, even when the trial is built deep inside a
worker function whose result the harness's top level never sees.

Rationale: a whole-vault diff cannot tell a trial's leak from the orchestrating session's own legitimate writes to the
real vault while a run is in progress (route evidence notes, recall-glance sidecars), so it false-alarms on them.

### 792a23d1 UAT Scenarios

```gherkin
Scenario: 792a23d1a The orchestrating session's own writes do not trigger a leak report
  Given a trial run in progress with isolated per-trial environments built through the standard primitive
  When the orchestrating session writes a note and its sidecar into the real vault, for example through a /route recall glance
  Then the run completes without reporting a leak, because the note names no trial directory or session identifier of the run

Scenario: 792a23d1b A harness captures no trial evidence itself and still gets a correct guard
  Given a harness that builds each trial's environment in a per-trial worker function and never returns it to its top-level run loop
  When the run completes and the harness invokes the leak guard
  Then the guard reflects every trial that ran, because each environment was recorded when it was built

Scenario: 792a23d1c Every harness snapshots the real vault and asserts no leak
  Given each tier-A harness and each runbook-versus-skill probe
  When its code is read
  Then it takes the real vault's fingerprint before the run and calls the leak guard after it, and no harness still holds the former whole-vault comparison
```

## 792a23e A real leak through the standard primitive is caught and fails loud

Requirement: [792a23e A real leak through the standard primitive is caught and fails loud](#792a23e-a-real-leak-through-the-standard-primitive-is-caught-and-fails-loud) When a note is newly present in the real vault after a run, and its content names a trial directory or session identifier of a
trial built through the standard isolation primitive, the leak guard MUST raise an error that names the note and the trial
marker. It MUST also raise when the content names only the trial marker's basename and that basename is distinctive, a
random suffix such as `tempfile.mkdtemp` makes, and not when it is a short deterministic name that a note could mention
in passing.

Rationale: replacing whole-vault diffing with trial-side detection must not weaken the guard against an actual leak,
and a note that quotes a trial's full path is the dominant real shape of one. A short basename such as `plumbing-0`
could false-match an orchestrator note that mentioned the fixture by name.

### 792a23e1 UAT Scenarios

```gherkin
Scenario: 792a23e1a A new real-vault note naming a trial is flagged
  Given a run whose trials were built through the standard isolation primitive
  When a note appears in the real vault that did not exist before the run and its content names a trial directory or session identifier of the run
  Then the guard raises, naming the note and the trial it references

Scenario: 792a23e1b A note naming only a distinctive basename is still flagged
  Given a trial marker whose basename ends in a random suffix
  When a note appears in the real vault whose content names only that basename and not the trial's full path
  Then the guard raises, naming the note and the matched basename

Scenario: 792a23e1c A note naming only a short deterministic basename is not flagged
  Given a trial marker whose basename is a fixed task name and a small index
  When a note appears in the real vault whose content mentions only that basename in passing
  Then the guard does not raise
```

## 792a23f The leak guard's limits are stated

Requirement: [792a23f The leak guard's limits are stated](#792a23f-the-leak-guards-limits-are-stated) The trial-side guard MUST NOT be described or relied on as a complete leak-proof: its detection is scoped to leaks that
a registered trial marker in the leaked note's own content traces, and the limits that follow MUST stay documented. A leak
through a path that never builds its environment with the standard primitive, a leaked note whose content names no
registered trial directory or session identifier, a leak that names only a non-distinctive basename, and an in-place content
mutation of an existing note that leaves the set of note basenames unchanged are not detected.

Rationale: stating the limits keeps a green guard from being read as a proof. The in-place mutation case is reachable
only through the unmediated path already named; the short-basename case is accepted in exchange for not false-matching
an orchestrator note.

### 792a23f1 UAT Scenarios

```gherkin
Scenario: 792a23f1a A leak through an unmediated path is not caught
  Given a code path that writes to the real vault without building its environment through the standard primitive
  When that path leaks into the real vault
  Then the guard does not detect it, and this is a documented limit

Scenario: 792a23f1b A leak whose content does not self-identify is not caught
  Given a trial built through the standard primitive that leaks a note whose content names no registered trial directory or session identifier
  When the guard runs
  Then it does not flag the note, and this is a documented limit

Scenario: 792a23f1c A same-filename in-place mutation is not caught
  Given a guard whose fingerprint is the set of the real vault's .md basenames
  When an existing note's content is mutated in place without changing the set of basenames
  Then the guard does not flag it, and this is a documented limit

Scenario: 792a23f1d A leak naming only a non-distinctive basename is not caught
  Given a trial marker with a short, deterministic basename
  When a leaked note's content names only that basename and never the trial's full path
  Then the guard does not flag it, and this is a documented limit
```

## 792a23g A step is detected in every valid form of its command

Requirement: [792a23g A step is detected in every valid form of its command](#792a23g-a-step-is-detected-in-every-valid-form-of-its-command) The step definitions of the `runbook_vs_skill` phase 2 probe's fixtures MUST recognize every commonly used invocation
form of the command a step is meant to detect, when a later step's detection depends on it, and MUST NOT match a command
that merely contains the command's name. A command text that happens to contain the literal substring immediately preceded
by a word character is not a real invocation and MUST NOT match.

Rationale: a step that matches one valid form of a command scores every genuine success that used another form as a miss, and
cascades the miss to every step gated behind it.

### 792a23g1 UAT Scenarios

```gherkin
Scenario: 792a23g1a The hyphenated git-filter-repo form is matched
  Given a trial's bash history that contains git-filter-repo --path secrets.env --invert-paths --refs main --force
  When the history-rewrite fixture is scored
  Then step 3 and the steps gated behind it score as followed

Scenario: 792a23g1b The space-separated git filter-repo form still matches
  Given a trial's bash history that contains git filter-repo --path secrets.env --invert-paths --refs main --force
  When the history-rewrite fixture is scored
  Then step 3 scores as followed, exactly as before

Scenario: 792a23g1c An unrelated command is still not matched
  Given a trial's bash history that contains cat notes-about-filter-repo.md
  When the history-rewrite fixture is scored
  Then step 3 does not score as followed

Scenario: 792a23g1d A letter-adjacent substring is not matched
  Given a trial's bash history that contains digit-filter-repo --some-arg or mygit-filter-repo --some-arg
  When the history-rewrite fixture is scored
  Then step 3 does not score as followed, because the pattern requires a left boundary before git
```

## 792a23h A question stop is judged in the order of the task's completion

Requirement: [792a23h A question stop is judged in the order of the task's completion](#792a23h-a-question-stop-is-judged-in-the-order-of-the-tasks-completion) The phase 2 probe MUST report a trial's `question_stop` as true only when the last ambiguity-posing text in its transcript
precedes the trial's task completion, and not merely when no mutating tool call follows that text. A genuinely separate
follow-up question that the trial asks after it has completed and verified its task, as its own end-state check shows, MUST
NOT be scored as a trial that stopped before finishing.

Rationale: otherwise a trial that finished the work and then asked a question is scored as a stop, which mixes a clean
success with a stop to ask.

### 792a23h1 UAT Scenarios

```gherkin
Scenario: 792a23h1a A trailing follow-up question after full completion is not a stop
  Given a trial that completes and verifies its task, its end-state check passing, and then asks a separate follow-up question with no further mutation
  When the trial is scored
  Then question_stop is false

Scenario: 792a23h1b A genuine pre-completion stop is still detected
  Given a trial that poses an ambiguity before completing, performs no further mutation, and whose end-state check does not pass
  When the trial is scored
  Then question_stop is true
```

## 792a23i A fixture gives a runbook's follow-up action a local target

Requirement: [792a23i A fixture gives a runbook's follow-up action a local target](#792a23i-a-fixture-gives-a-runbooks-follow-up-action-a-local-target) When a runbook that a trial follows prescribes an action that the fixture gives no local way to perform, such as filing a
follow-up issue in a fixture repository with no issue tracker, the fixture MUST provide a minimal local convention that
satisfies the action and the task prompt MUST name it. The runbook MUST NOT be changed to work around a fixture gap.

Rationale: otherwise a shim-following agent is forced to stop and ask merely because the fixture omitted the means to comply,
which the score would read as the agent's failure.

### 792a23i1 UAT Scenarios

```gherkin
Scenario: 792a23i1a The bisect-before-fix fixture names a local issue-filing target
  Given a trial that follows the runbook's step 6 and must file a follow-up issue for a pre-existing regression in the bisect-before-fix fixture
  When the fixture and its task prompt are read
  Then the prompt names ISSUES.md as the place to file it, and the fixture repository contains that file from its initial commit onward
```

## 792a23j A surfacing probe reads the rank of the item list

Requirement: [792a23j A surfacing probe reads the rank of the item list](#792a23j-a-surfacing-probe-reads-the-rank-of-the-item-list) A harness or script that decides whether a target note surfaced for an `engram query` MUST read the query payload's
top-level `items[]` list, matching on path or basename, and report the note's one-based rank there. It MUST NOT use
membership in a cluster's `candidate_l2s` as the surfacing signal, and absence from `candidate_l2s` MUST NOT count as a miss.

Rationale: `candidate_l2s` is a capped top-five, within-cluster subset chosen for synthesis ([792b24c10 Candidate notes](../technical/query-engine.md#792b24c10-candidate-notes)), so a note can be
correctly matched and present in `items[]` while absent from every cluster's list because its cluster holds more than five
members closer to the centroid.

### 792a23j1 UAT Scenarios

```gherkin
Scenario: 792a23j1a A target present in items[] counts as surfaced
  Given a probe that runs a query and a target note whose path appears in the payload's top-level items[]
  When the probe reads the payload
  Then it reports the note as surfaced, at its one-based rank within items[]

Scenario: 792a23j1b candidate_l2s absence does not count as a miss
  Given a target note present in items[] and absent from every cluster's candidate_l2s
  When the probe reads the payload
  Then it still reports the note as surfaced
```

## 792a23k The audit detects moments by reading them

Requirement: [792a23k The audit detects moments by reading them](#792a23k-the-audit-detects-moments-by-reading-them) An audit of the memory loop against real transcripts MUST detect moments, which are failures, corrections, rework,
successes (a case where a surfaced or injected memory visibly shaped what the agent did) and dispatches (a subagent launch),
by reading the transcript in context and MUST NOT match keywords or fixed phrases. It MUST flag generously and discard
non-matches afterwards. Its quality bar MUST be a miss rate, how many real moments it fails to catch, measured against
hand-labeled examples and agreed before any full-corpus run, and never how often it over-flags.

Rationale: most corrections are subtle enough that word-matching alone misses most of them, and the successes are the
reinforcement half of the loop.

### 792a23k1 UAT Scenarios

```gherkin
Scenario: 792a23k1a A subtle correction is detected
  Given a transcript window that contains a correction with no blunt signal word, such as a redirect, a quiet reversal or a reviewer rejection in its own idiom
  When the auditor reads the window
  Then it reports the correction as a detected moment

Scenario: 792a23k1b A success moment is detected
  Given a transcript window that shows the agent applying a recalled or prompt-injected memory and the action visibly succeeding
  When the auditor reads the window
  Then it reports a success moment that names the memory involved

Scenario: 792a23k1c The miss rate is agreed before the real run
  Given an audit about to run over the full sample
  When it starts
  Then a hand-labeled calibration sample of at least 10 to 20 windows, stratified by role and repository, exists, its measured miss rate is recorded, and the tolerance was committed before the full run began
```

## 792a23l Every audited moment carries a complete record

Requirement: [792a23l Every audited moment carries a complete record](#792a23l-every-audited-moment-carries-a-complete-record) Each detected moment's record MUST answer every question needed to see where the memory loop broke down for it. On the
finding side it MUST say whether a relevant memory existed, with its kind (chunk, fact, feedback, runbook or qa), whether it
was generic or specific to the situation, and whether that existence check was DERIVED or ESTIMATE; whether a search ran (a
quick glance, a full recall, an injected prompt or none); whether the search targeted the right thing, judged separately;
whether the memory surfaced, with its rank and whether an outdated note outranked its replacement; and whether it was
followed, recording how it deviated (ignored, partial, a step skipped, steps out of order or contradicted). On the writing side it
MUST say whether the moment was worth learning from, whether a learn step fired, whether a note was written and was well
targeted, correctly superseded an old note and was not a duplicate, and whether the used note's strength was updated, with
mismatches flagged. For a handoff between agents it MUST say which memories the subagent's prompt included and, for a
dispatch, which ones the orchestrator held and left out. It MUST carry a failure category of fixable,
nothing-could-have-caught-it, or found-but-not-followed (vault runbook 824), and the transcript, the location, the timestamp,
the repository, the role and a pointer to the evidence. Every answer MUST be a value or an explicit not-applicable. A rate
MUST NOT be computed before its counting unit (per moment, per dispatch or per transcript) is fixed, and every reported
number MUST be labeled DERIVED (computed from real records) or ESTIMATE (approximated because the exact data was not
available).

Rationale: the loop's funnel names the loss stages of the loop, and a record that skips one cannot show where the loss
happened. A rate with no stated unit is a number nobody can check.

### 792a23l1 UAT Scenarios

```gherkin
Scenario: 792a23l1a A complete moment record
  Given the auditor emitting a moment record
  When the record is read
  Then every question above has a value or an explicit not-applicable, and the record carries the transcript, location, timestamp, repository, role and a pointer to the evidence

Scenario: 792a23l1b A rate without a fixed counting unit is a violation
  Given a report that states a rate such as "X% of dispatches lacked a relevant note"
  When the report is read
  Then it names what is counted and labels the number DERIVED or ESTIMATE
```

## 792a23m Whether a memory existed is judged against the vault of that time

Requirement: [792a23m Whether a memory existed is judged against the vault of that time](#792a23m-whether-a-memory-existed-is-judged-against-the-vault-of-that-time) The audit MUST judge "was a relevant memory in the vault" by searching the vault as it was at the moment's own timestamp, with
search phrases the auditor writes from the moment itself, and the agent's own phrases at the time are judged separately. For
notes, a read-only checkout of the vault at the last commit at or before the timestamp MUST give a DERIVED answer, and where
git history does not exist or does not reach back that far, filtering notes by their creation date MUST give an ESTIMATE
that is labeled as such, with the report's limits saying that later edits to a note do not show in it. For chunks, filtering by ingestion
date MUST always give a DERIVED answer. The check MUST NOT write to the live vault, and every `engram` command it runs MUST run
under the eval harness's sandboxing (#708).

Rationale: a memory written after a moment cannot have been missed at it, and an exact answer is worth a checkout where history
allows one.

### 792a23m1 UAT Scenarios

```gherkin
Scenario: 792a23m1a An exact answer via git checkout
  Given a vault whose git history covers the moment's timestamp
  When the existence check runs
  Then it queries a detached, read-only checkout at the last commit at or before the timestamp, and the judgment is labeled DERIVED

Scenario: 792a23m1b An approximate answer without history
  Given a vault with no git history covering the timestamp
  When the existence check runs
  Then it filters notes by created at or before the timestamp, labels the judgment ESTIMATE, and the report's limits say that later edits to a note do not show in it

Scenario: 792a23m1c The live vault is untouched
  Given an audit run
  When it completes
  Then a before-and-after fingerprint of the live vault shows no change attributable to the audit

Scenario: 792a23m1d Chunks are always judged by ingestion date
  Given a chunk index that holds chunks ingested before and after the timestamp
  When the existence check runs
  Then only the chunks ingested at or before it are searched, and the answer is DERIVED
```

## 792a23n The existence check's query cannot be broken by a phrase and does not swallow a failure

Requirement: [792a23n The existence check's query cannot be broken by a phrase and does not swallow a failure](#792a23n-the-existence-checks-query-cannot-be-broken-by-a-phrase-and-does-not-swallow-a-failure) The existence check's `engram query` invocation, and any function that builds one from auditor-generated phrase or text
strings, MUST pass each value in a form `engram`'s flag parser cannot mistake for a new flag, whatever the value's content,
a value that itself begins with `--` included. When the invocation exits non-zero, the error information it returns MUST
contain the tool's actual error output, on whichever stream it was written, so that a failed check is diagnosable from its own
recorded error field.

Rationale: `engram`'s own errors print to standard output, and a phrase beginning with `--` is parsed as a flag, which
records a failure with an empty error.

### 792a23n1 UAT Scenarios

```gherkin
Scenario: 792a23n1a A phrase starting with -- does not break the invocation
  Given the existence check given a search phrase that begins with --, such as "--text verbatim scenario dropped from modified requirement"
  When it runs engram query
  Then the query runs successfully and the result reflects a real query outcome, not a flag-parsing error

Scenario: 792a23n1b A real query failure's error text is preserved
  Given engram query exits non-zero and writes its error message to standard output
  When the existence check returns
  Then its error field contains that message and is never empty or uninformative
```

## 792a23o Prompt-injected memories count as searched and surfaced

Requirement: [792a23o Prompt-injected memories count as searched and surfaced](#792a23o-prompt-injected-memories-count-as-searched-and-surfaced) For a subagent moment, any memory present in the dispatch prompt MUST count as found: it is recorded as "a search ran
(injected)" and "surfaced: yes", and a forked subagent, which inherits the whole parent conversation, MUST have everything in it
count as found. The gap on the finding side, memories the orchestrator surfaced and did not hand over, MUST be measured from the
orchestrator's side at the moment of dispatch with a structural check across the corpus, validated against a smaller
sample judged by a model that tells genuinely missing content from content paraphrased into the prompt.

Rationale: what matters for the subagent is whether it followed a memory it had, and what matters for the orchestrator
is what it left out.

### 792a23o1 UAT Scenarios

```gherkin
Scenario: 792a23o1a An injected memory counts as found for the subagent
  Given a subagent transcript whose dispatch prompt contains a vault note's content, and a later moment that note addresses
  When the auditor judges the moment
  Then it records search ran injected and surfaced yes, and measures whether the subagent followed the memory

Scenario: 792a23o1b A dispatch moment measures the handoff on the orchestrator's side
  Given an orchestrator that dispatches a fresh-context subagent while holding memories relevant to the unit
  When the auditor records the dispatch moment
  Then it records which of those memories the prompt included and which it left out, and the leftover ones count toward the finding-side gap per dispatch

Scenario: 792a23o1c Paraphrased content is told from missing content
  Given the structural check flags a note as missing from a dispatch prompt
  When the sampled model check runs
  Then it judges the content paraphrased in, genuinely missing, or ambiguous, and the structural rate is adjusted before the gap is reported
```

## 792a23p The audit emits credit records and never consumes them

Requirement: [792a23p The audit emits credit records and never consumes them](#792a23p-the-audit-emits-credit-records-and-never-consumes-them) The audit MUST emit draft outcome records, each `{moment_id, note_ref, situation_text, outcome, ts,
evidence{transcript, anchor}}` with `outcome` one of `applied-helped`, `applied-hurt`, `surfaced-ignored`, `injected-unused` and
`absent-needed`, as plain data files. It MUST NOT write, amend, activate, supersede or remove any vault note, because acting on the
records takes a human judgment and belongs to a separate, later capability.

Rationale: engram's learning loop is judged-only: nothing in the vault changes unjudged, so the audit labels what happened and a person
acts on it.

### 792a23p1 UAT Scenarios

```gherkin
Scenario: 792a23p1a Output only, no side effects
  Given a corpus that contains applied-hurt evidence for a note
  When an audit run finishes
  Then an outcome record for that note exists in the output files and the note itself is byte-identical to before the run
```

## 792a23q Audit results land additively in the ledger

Requirement: [792a23q Audit results land additively in the ledger](#792a23q-audit-results-land-additively-in-the-ledger) Audit findings MUST be recorded as `dev/eval/LEDGER.md` rows, one claim per row in the existing format, each pointing at its
raw data, and reflected in a coverage-map index that maps each memory function and kind to the rows backing it and the measured
real-world rate. Recording audit results MUST NOT rename, rewrite (beyond the format's own way of superseding a row) or remove any existing
row or anchor.

Rationale: other documents cite the ledger's anchors, so the ledger grows and is never rebuilt.

### 792a23q1 UAT Scenarios

```gherkin
Scenario: 792a23q1a Adding without disturbing what is there
  Given an audit run's findings to record
  When they are written to the ledger
  Then every pre-existing ledger anchor still resolves, and the new claims appear as new rows referenced from the coverage-map index
```
