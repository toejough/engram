# Learn - Product Spec

`learn` is the skill an agent runs to keep memory true at the end of work, and at the moments a lesson is confirmed. It has two jobs, in order: sweep the chunk index so every conversation and document is searchable, and write a vault note for every explicit lesson of the session. The explicit lessons are user corrections, explicit save-requests, presented conclusions that were later overturned, and approaches confirmed good. It writes nothing else. The sweep is chunk-indexing's ([chunk-indexing](./chunk-indexing.md)), the notes are written through [write-memory](./write-memory.md) and [note-writing](./note-writing.md), and the lessons that subagents return are lessons-contract's ([lessons-contract](./lessons-contract.md)). The shipped procedure is `agent-instructions/skills/learn/SKILL.md`, cited here by the skill's step numbers.

## 792a15a Learn writes only explicit lessons

Requirement: [792a15a Learn writes only explicit lessons](#792a15a-learn-writes-only-explicit-lessons) Learn MUST NOT write an episode, a session summary or a note for a fact that nobody asked to remember and that is not one of the four kinds of lesson ([792a15e The four kinds of explicit lesson](#792a15e-the-four-kinds-of-explicit-lesson)), and MUST NOT run `engram transcript` or `engram learn episode`, or write a tier or a decision-record note. A session with no correction, save-request, reversal or confirmed approach MUST end with the sweep and its report alone.

Rationale: raw event memory is automatic: `engram ingest` chunks and embeds session transcripts and markdown without summaries, episode notes or arc detection, so the agent's only writing job is crystallizing explicit lessons. The episode workflow was retired when the chunk layer replaced it.

### 792a15a1 UAT Scenarios

```gherkin
Scenario: 792a15a1a A session with no lesson writes nothing
  Given a session with no correction, save-request, reversal or confirmed approach
  When the agent runs learn
  Then it runs the sweep, reports its tally and writes no note

Scenario: 792a15a1b No episode or summary is written
  Given a long session
  When the agent runs learn
  Then it writes no episode or session-summary note and runs neither engram transcript nor engram learn episode
```

## 792a15b The sweep comes first

Requirement: [792a15b The sweep comes first](#792a15b-the-sweep-comes-first) Learn MUST begin with `engram ingest --auto` and report what it prints (one `ingest <path>: ...` line for each source it re-chunked) or, when it prints nothing because the corpus is unchanged, "memory index up to date", and MUST continue to the lessons when the sweep fails, surfacing the error. It MUST skip the sweep when one already ran earlier in the same session, unless this is the closing learn of a work cycle, which MUST always sweep.

Rationale: an unchanged corpus returns in seconds. The closing learn captures the session's tail for future sessions. A worry that something changed in another terminal is picked up by the next sweep, not by sweeping now on suspicion. Explicit lessons must not be lost to an ingest hiccup.

### 792a15b1 UAT Scenarios

```gherkin
Scenario: 792a15b1a The sweep runs and is reported
  Given no sweep has run in this session
  When the agent runs learn
  Then it runs engram ingest --auto and reports the per-source lines it prints, or "memory index up to date" when it prints nothing

Scenario: 792a15b1b A failed sweep does not block the lessons
  Given engram ingest --auto fails
  When the agent continues
  Then it surfaces the error and still crystallizes the session's lessons

Scenario: 792a15b1c An earlier sweep is not repeated
  Given a sweep already ran earlier in this session and this is not the closing learn
  When the agent runs learn
  Then it does not sweep again

Scenario: 792a15b1d The closing learn always sweeps
  Given a sweep already ran earlier in this session and this is the closing learn of a work cycle
  When the agent runs learn
  Then it sweeps again
```

## 792a15c A correction moment gets a single-note capture

Requirement: [792a15c A correction moment gets a single-note capture](#792a15c-a-correction-moment-gets-a-single-note-capture) When learn fires at a correction moment in the middle of a task (a review, a failing check, or the user rejecting the approach or naming a different convention), it MUST skip the sweep and the vocabulary check, go straight to crystallizing the one confirmed correction, and hand it to write-memory as always. It MUST NOT run `engram ingest --auto` in the middle of a task.

Rationale: the capture is a focused single note and not the opening or closing of a cycle. A sweep is a cycle-boundary job: it is wasteful in the middle of a task, can block on a large corpus, and losing the capture to a hung sweep is the failure mode. The closing learn still sweeps. The same fast path serves a subagent return ([lessons-contract](./lessons-contract.md)).

### 792a15c1 UAT Scenarios

```gherkin
Scenario: 792a15c1a A rejected approach is captured at once
  Given a reviewer rejected the agent's approach in the middle of a task
  When learn fires
  Then it writes one note for that correction through write-memory with no sweep and no vocabulary check

Scenario: 792a15c1b No mid-task sweep
  Given learn fires at a correction moment
  When the agent proceeds
  Then it does not run engram ingest --auto
```

## 792a15d The vocabulary check and its autonomous refit

Requirement: [792a15d The vocabulary check and its autonomous refit](#792a15d-the-vocabulary-check-and-its-autonomous-refit) Learn MUST run `engram vocab stats` before it crystallizes lessons in a full cycle. On `verdict: OK` it MUST do nothing more. On `verdict: REFIT_PENDING (<reason>)` it MUST run the refit itself without deferring to the user: it MUST run `engram vocab refit`; when that prints `vocab refit: no structure found; vocabulary unchanged` it MUST report exactly that and not claim an applied refit; when it prints a JSON payload of naming requests and exits without writing it MUST name every new cluster exactly once from its exemplars, write an answers file that echoes the payload's `fingerprint` verbatim, and re-run `engram vocab refit --names <file>`; when that re-run fails with a stale-names error it MUST discard those requests, re-run the refit from scratch and answer the new ones; and when a refit is applied it MUST report "Vocab refit applied: <version bump>. Triggered by: <reason>." When the stats output carries `qa round-2 gate: READY (...)`, learn MUST tell the user that the round-2 validation is due and which gates to schedule, and MUST NOT run that validation itself.

Rationale: the vault's own write path marks a refit pending when it has grown enough ([792a6s Refit trigger](./controlled-vocabulary.md#792a6s-refit-trigger)) and the learn skill acts on the verdict, so vocabulary drift is repaired without asking. `--dry-run` prints the derivation diff without writing, for a first refit over a long-drifted vault. The round-2 validation needs the user's oversight.

### 792a15d1 UAT Scenarios

```gherkin
Scenario: 792a15d1a A healthy vocabulary needs nothing
  Given engram vocab stats prints verdict: OK
  When learn reads it
  Then it takes no vocabulary action

Scenario: 792a15d1b A pending refit is run autonomously
  Given engram vocab stats prints verdict: REFIT_PENDING (growth: 52 notes, 20 days)
  When learn reads it
  Then it runs engram vocab refit without asking the user

Scenario: 792a15d1c No structure found is reported as such
  Given engram vocab refit prints "vocab refit: no structure found; vocabulary unchanged"
  When learn reports
  Then it says the refit ran with no structure found and the vocabulary unchanged, and does not claim an applied refit

Scenario: 792a15d1d New clusters are named and the refit re-run
  Given engram vocab refit prints naming requests and a fingerprint
  When learn answers
  Then it names every new cluster exactly once, echoes the fingerprint verbatim in the answers file and re-runs the refit with --names

Scenario: 792a15d1e A stale answer restarts the refit
  Given the re-run fails with a stale-names error
  When learn recovers
  Then it re-runs engram vocab refit from scratch and answers the regenerated requests

Scenario: 792a15d1f An applied refit is reported loudly
  Given a refit is applied
  When learn reports
  Then it states "Vocab refit applied: <version bump>. Triggered by: <reason>."

Scenario: 792a15d1g The QA round-2 gate is reported and not run
  Given the stats output carries qa round-2 gate: READY (20>=20)
  When learn reads it
  Then it tells the user the round-2 validation is due and does not run it
```

## 792a15e The four kinds of explicit lesson

Requirement: [792a15e The four kinds of explicit lesson](#792a15e-the-four-kinds-of-explicit-lesson) Learn MUST scan the session for exactly four kinds of moment and MUST hand each to write-memory. A correction (the user corrected the agent's approach or behavior) MUST be handed off as a feedback note carrying the situation, the behavior, its cost as the impact and what to do instead as the action. An explicit save-request ("remember this", "save that", "note for next time", "write this down") MUST be handed off immediately as a fact note carrying the situation, the subject, the predicate and the object. A reversal MUST be handed off as a feedback note whose situation says when the failure mode applies, whose behavior is what the original reasoning did wrong, whose impact is what the reversal cost and whose action is the guard that would have prevented it. A confirmed approach MUST be handed off as a feedback note, or as a runbook note when its shape calls for one ([792a15g A confirmed approach needs a bet or specific praise, and takes the kind its shape calls for](#792a15g-a-confirmed-approach-needs-a-bet-or-specific-praise-and-takes-the-kind-its-shape-calls-for)). Every handoff MUST carry a slug and a source of the form "session <date>, context: <one-line what-was-happening>", and the supersession details when the lesson corrects an existing vault note.

### 792a15e1 UAT Scenarios

```gherkin
Scenario: 792a15e1a A correction becomes a feedback note
  Given the user said "don't suppress lint warnings, fix the underlying issue"
  When learn crystallizes it
  Then write-memory receives kind feedback with situation, behavior, impact and action

Scenario: 792a15e1b A save-request becomes a fact note at once
  Given the user said "remember that releases are tagged from main"
  When learn runs
  Then write-memory receives kind fact with situation, subject, predicate and object, before anything else is done

Scenario: 792a15e1c A reversal becomes a feedback note about the root cause
  Given a design that was presented and later overturned by a reviewer
  When learn crystallizes it
  Then write-memory receives kind feedback whose behavior is what the original reasoning did wrong and whose action is the guard against it

Scenario: 792a15e1d A confirmed approach becomes a feedback note
  Given a specific approach confirmed by the user or by an observable outcome
  When learn crystallizes it
  Then write-memory receives kind feedback with the situation when it would apply again, what worked, the confirming evidence and a keep-doing action

Scenario: 792a15e1e A correction of an existing note carries a supersession
  Given a lesson that corrects, narrows or refutes an existing vault note
  When learn hands it off
  Then the handoff includes that note's basename, the type and the claim
```

## 792a15f Reversals and repo-doc corrections

Requirement: [792a15f Reversals and repo-doc corrections](#792a15f-reversals-and-repo-doc-corrections) Learn MUST capture a reversal as a vault note even when nobody said the correction aloud (a self-discovered reversal by the agent, a reviewer or an instrument qualifies), and MUST NOT treat a CORRECTION section or postscript in a repository document as that capture. Learn MUST write the note for the reversal's root cause and not a narrative of the flip.

Rationale: a record of a correction is not the capture of its lesson.

### 792a15f1 UAT Scenarios

```gherkin
Scenario: 792a15f1a A self-discovered reversal is captured
  Given the agent's presented design was overturned by a failing instrument with nobody saying so
  When learn runs
  Then it writes a reversal note for the root cause

Scenario: 792a15f1b A repo-doc correction does not count
  Given the agent added a CORRECTION section to a repository document
  When learn runs
  Then it still writes the vault note for the reversal
```

## 792a15g A confirmed approach needs a bet or specific praise, and takes the kind its shape calls for

Requirement: [792a15g A confirmed approach needs a bet or specific praise, and takes the kind its shape calls for](#792a15g-a-confirmed-approach-needs-a-bet-or-specific-praise-and-takes-the-kind-its-shape-calls-for) Learn MUST capture a confirmed approach only when either the user explicitly praised or thanked a specific behavior that can be restated as a reusable tactic, or the agent made a genuine guess or was uncertain about a plan, acted on it, and a session-recorded outcome (a test passed, the user confirmed it, an artifact functioned, a blocker cleared) then confirmed it. It MUST NOT capture a bare pleasantry that names no behavior, a routine success with no bet behind it, or an unconfirmed guess. It MUST hand off a reusable, ordered, multi-step procedure for a recurring task as a runbook note (with `done_when` and a body of numbered steps, which may link fact or feedback notes) and a single behavioral tweak with no step structure as a feedback note. When it writes a runbook and the user also says to do that going forward, it MUST NOT additionally write a fact note for a save-request on the same content.

Rationale: a routing by shape was added when the runbook kind was introduced: a runbook capture is itself the "remember to do it this way" note. The confirmed-approach kind is the positive mirror of a reversal and, in practice, the kind that fires least often, so the skill carries exemplars (a design assumption checked by a targeted code search, a suspected defect confirmed by a reproduction test, a vendor claim verified against evidence), but the bar does not move: a resolved uncertainty or an explicit specific confirmation, never "it worked".

### 792a15g1 UAT Scenarios

```gherkin
Scenario: 792a15g1a Specific praise is captured
  Given the user said "thanks for attaching the file, it is easier to read from my phone"
  When learn runs
  Then it captures "attach deliverable files instead of inlining them" as a confirmed approach

Scenario: 792a15g1b A bare thanks is not
  Given the user said "thanks!" and named no behavior
  When learn runs
  Then it writes no note

Scenario: 792a15g1c A self-validated bet is captured
  Given the agent was unsure an assumption held, ran a targeted search to check it, and the search confirmed it
  When learn runs
  Then it captures the check-before-you-commit method as a confirmed approach

Scenario: 792a15g1d A routine success is not
  Given work that went well with no uncertainty resolved
  When learn runs
  Then it writes no confirmed-approach note

Scenario: 792a15g1e A procedure becomes a runbook
  Given a confirmed sequence of run tests, tag the release, push the tag and update the changelog
  When learn hands it off
  Then write-memory receives kind runbook with a situation, done_when and numbered steps

Scenario: 792a15g1f A single tweak stays feedback
  Given a confirmed behavior with no step structure
  When learn hands it off
  Then write-memory receives kind feedback

Scenario: 792a15g1g A runbook is not duplicated as a save-request
  Given a confirmed runbook and the user also said "do that going forward"
  When learn runs
  Then it writes the runbook and no additional fact note for the same content
```

## 792a15h Each note is placed by an ordered test

Requirement: [792a15h Each note is placed by an ordered test](#792a15h-each-note-is-placed-by-an-ordered-test) Before each hand-off, learn MUST decide the note's Luhmann placement, considering only notes written or recalled earlier in the same session, and MUST pass the resulting position and target to write-memory. It MUST apply these tests in order: when the note develops one specific sub-point raised inside an in-session note, the placement is `continuation` with that note's ID as the target; otherwise, when it continues or extends the same overall thought at the same level, the placement is `sibling` with that note's ID as the target; otherwise the placement is `top` with no target. When no in-session candidate exists, it MUST choose `top` and MUST NOT search the rest of the vault for a target.

Rationale: an ID's position encodes where a note sits in the thought tree, which follows Niklas Luhmann's own practice and Sönke Ahrens, _How to Take Smart Notes_. The disposition step had been removed from the skill on 2026-06-12, so for months every capture was `top` and all 447 notes in the vault carried flat integer IDs, leaving the branching the ID scheme was named for unused. Placement is a judgment of content, which the skill makes and the binary does not.

### 792a15h1 UAT Scenarios

```gherkin
Scenario: 792a15h1a A sub-point is a continuation
  Given the new note develops one specific sub-point raised inside a note written earlier this session
  When learn decides placement
  Then the handoff carries position continuation and that note's ID as the target

Scenario: 792a15h1b The same thought at the same level is a sibling
  Given the new note extends the same overall thought as an in-session note without being a sub-point of it
  When learn decides placement
  Then the handoff carries position sibling and that note's ID as the target

Scenario: 792a15h1c An unrelated note is top-level
  Given the new note relates to no note written or recalled earlier this session
  When learn decides placement
  Then the handoff carries position top and no target

Scenario: 792a15h1d Continuation is checked before sibling
  Given a note that could be read as either a sub-point or a same-level continuation
  When learn decides placement
  Then it applies the continuation test first and takes the first test that matches

Scenario: 792a15h1e The vault is not searched
  Given no note was written or recalled earlier this session
  When learn decides placement
  Then it chooses top without searching the vault
```

## 792a15i A lesson states the general principle

Requirement: [792a15i A lesson states the general principle](#792a15i-a-lesson-states-the-general-principle) Learn MUST state the general principle and not the session-specific instance, MUST phrase `situation` as the retrieval handle a future task would be described by (for example "releasing a Go module"), and MUST write one note per distinct principle.

Rationale: capture is generic and actionable, because a future session recalls by situation similarity and not by remembering this one. This is a skill-authoring convention, gated by baselines and evals, and not an architecture decision.

### 792a15i1 UAT Scenarios

```gherkin
Scenario: 792a15i1a The principle, not the instance
  Given a correction made while fixing one specific test file
  When learn writes the note
  Then the note states the general rule and not the file's narrative

Scenario: 792a15i1b One note per principle
  Given a session with two distinct corrections
  When learn writes the notes
  Then there are two notes, one per principle
```

## 792a15j A write to a parent is reported as done

Requirement: [792a15j A write to a parent is reported as done](#792a15j-a-write-to-a-parent-is-reported-as-done) With `ENGRAM_PARENT` set, learn MUST treat the write as complete once the note is written locally, because the binary offers it upward without anything for the agent to run. When a write prints `engram: parent unreachable (retry after …); N offer(s) queued`, learn MUST report the note as saved and its offer as queued for the parent, MUST NOT re-run the write, MUST NOT call it failed, and MUST NOT set `ENGRAM_SERVER`.

Rationale: the note is in the local vault and the offer waits in the queue until a later engram command reaches the parent. `ENGRAM_SERVER`'s mode is gone and setting it is a hard error ([parent-sync](./parent-sync.md)).

### 792a15j1 UAT Scenarios

```gherkin
Scenario: 792a15j1a An unreachable parent is not a failure
  Given a write that printed "engram: parent unreachable (retry after 30s); 1 offer(s) queued"
  When learn reports
  Then it says the note is saved and the offer is queued, and does not retry the write

Scenario: 792a15j1b ENGRAM_SERVER is never set
  Given a write whose parent is unreachable
  When learn considers a workaround
  Then it does not set ENGRAM_SERVER
```

## 792a15k An answered question is captured once

Requirement: [792a15k An answered question is captured once](#792a15k-an-answered-question-is-captured-once) Learn MUST scan the session for a substantive answered question, one whose answer body contains at least one `[[full-basename]]` wikilink or whose answer is a new vault note it crystallized, and for each such question not yet captured MUST hand off to write-memory a note of kind qa with a slug, the verbatim question, the answer body copied, contributor basenames, a certainty and the source "ad-hoc capture, learn session <date>". Contributors MUST come only from the `[[full-basename]]` wikilinks in the written answer, extracted verbatim and never free-listed, and learn MUST NOT pre-validate that they exist; when write-memory reports a contributor rejection, learn MUST surface it. Learn MUST skip a question answered by generic advice or without a wikilink and without a new note. It MUST NOT write a pair that was already written this session (for example by recall's Step 4), and MUST NOT write a pair for a question answered by a Step 2 note written this turn, because that note is the capture. It MUST still write a pair for a question answered by citing an existing earlier note when no new note was written for it. It MUST write one pair per distinct answered question.

Rationale: the traceability bar (a wikilink or a new note) is the observable test of a substantive answer. The self-answered exclusion fixes a production bug found by the write-memory eval (task 3.4b, trial R-0): the old wording treated the agent's own just-written note as a trigger for a second write, producing an unrequested duplicate `engram learn qa` write whenever the agent posed and answered a diagnostic question through its own note in the same turn (`dev/eval/cumulative/runbook_vs_skill/phase2/results/3.4b_write_memory_shim_only_retry2_sonnet5.md`).

### 792a15k1 UAT Scenarios

```gherkin
Scenario: 792a15k1a A cited answer gets a pair
  Given a question answered with a body that cites [[a]] and [[b]]
  When learn captures it
  Then write-memory receives kind qa with the verbatim question, the copied answer and contributors a and b

Scenario: 792a15k1b Generic advice gets none
  Given a question answered with generic advice, no wikilink and no new note
  When learn scans
  Then no qa pair is written

Scenario: 792a15k1c A pair already written is not written again
  Given recall's Step 4 wrote the pair for this question earlier this session
  When learn scans
  Then it writes no second pair

Scenario: 792a15k1d The agent's own note is the capture
  Given a question posed and answered through a Step 2 note written this turn
  When learn scans
  Then it writes no qa pair for it, whatever wikilinks the note cites

Scenario: 792a15k1e A prior note still earns a pair
  Given a question answered by citing a note that existed before this turn, with no new Step 2 note for it
  When learn scans
  Then it writes the qa pair

Scenario: 792a15k1f A rejected contributor is surfaced
  Given write-memory reports that a contributor was rejected
  When learn receives the report
  Then it surfaces the rejection
```

## 792a15l The re-parent batch mode

Requirement: [792a15l The re-parent batch mode](#792a15l-the-re-parent-batch-mode) When learn is handed the derive-phase candidate payload of `engram update --reparent-luhmann` (candidates with `note`, `note_excerpt`, `target`, `target_excerpt` and `similarity`, plus an `instruction` and a `fingerprint`) and asked for its answers file, it MUST judge each candidate by the content of its two excerpts and not by the similarity score alone, applying the placement test of [792a15h Each note is placed by an ordered test](#792a15h-each-note-is-placed-by-an-ordered-test) (sub-point as `continuation`, same thought as `sibling`, otherwise `top`). It MUST write exactly one entry for every distinct `note`, choosing the one `target` and `position` it finds most correct when a note has several candidates. It MUST write the answers as JSON of the shape `{"reparenting": [{"note", "position", "target"}], "fingerprint"}` with the fingerprint echoed verbatim, and tell the user to preview with `engram update --reparent-luhmann --answers <file> --dry-run` and then re-run the command without `--dry-run`. It MUST NOT run either command itself and MUST NOT run this mode as part of an ordinary capture.

Rationale: whether one note is a sub-point of another is a judgment of content that an embedding cannot make: two notes can share vocabulary and address different problems. A `top` verdict still needs a full entry, and there is no option to skip a note.

### 792a15l1 UAT Scenarios

```gherkin
Scenario: 792a15l1a Every candidate note is answered once
  Given a payload in which one note is a candidate for two targets
  When learn writes the answers file
  Then the file has exactly one entry for that note, with the target it judged most correct

Scenario: 792a15l1b A high score does not decide
  Given a candidate with similarity 0.8 whose excerpts address different problems
  When learn judges it
  Then its entry is position top

Scenario: 792a15l1c The fingerprint is echoed and the next step is the user's
  Given a finished answers file
  When learn reports
  Then the fingerprint is the payload's verbatim, the user is told to run the command with --dry-run first, and learn runs neither command
```
