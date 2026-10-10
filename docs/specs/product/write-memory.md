# Write Memory - Product Spec

`write-memory` is the skill that turns a decision to remember something into a vault note. A parent skill (recall or learn) decides what to capture and why, then hands the worker a structured request; the worker composes the matching `engram learn` command, runs it, checks the result and reports the written note path. The judgment stays with the parent and the mechanics stay with the worker. The shipped procedure is `agent-instructions/skills/write-memory/SKILL.md`; the vault runbook note that mirrors it is registered by the skill-runbook-registration capability ([skill-runbook-registration](./skill-runbook-registration.md)). What each note kind contains and how the binary writes it are note-writing's ([note-writing](./note-writing.md)); amending a note is note-revision's ([note-revision](./note-revision.md)); what makes a good capture belongs to learn and recall.

Rationale: the atomic-skills exploration (2026-07-04) evaluated splitting the skills into shared behavioral atoms. It extracted exactly one, the write seam, as a worker invoked as the parent's next whole action, never a mid-procedure reference fetch. It did not extract a read-memory atom, because recall's read, judge and write pipeline is sequentially cohesive. An earlier reference-card form of the same atom was dropped, and its "0/27 mid-procedure dereference" measurement binds nothing, because the instrument was invalid. The worker form's fire rates were measured with a fixed, marker-gated harness: the handoff fired at its seams, no boundary was violated and no spurious fire occurred in six non-handoff trials (`dev/eval/LEDGER.md#write-memory-worker-fire-rates`, 2026-07-04). The skills-to-runbooks conversion of this worker was proposed and then reversed; the skill file is the shipped form.

## 792a13a The worker runs only on a handoff and never re-judges it

Requirement: [792a13a The worker runs only on a handoff and never re-judges it](#792a13a-the-worker-runs-only-on-a-handoff-and-never-re-judges-it) The write-memory worker MUST run only when a parent skill hands it a write, MUST execute that write without re-judging whether it deserves to be made, and MUST NOT write anything the parent did not hand off.

Rationale: the skill's `description:` names only parent-instructed invocation, so it never competes for autonomous firing. The worker has no trigger of its own and no user's words reach it directly.

### 792a13a1 UAT Scenarios

```gherkin
Scenario: 792a13a1a A complete handoff is executed
  Given recall or learn hands write-memory a complete handoff of kind feedback
  When the worker runs
  Then it composes the engram learn feedback command from the handed-off fields, runs it and reports the written note path

Scenario: 792a13a1b The parent's decision is not re-litigated
  Given a handoff whose content the worker could consider minor or redundant
  When the worker runs
  Then it still composes and runs the write and does not decline, reword or merge it

Scenario: 792a13a1c No handoff, no fire
  Given a session in which something seems worth remembering but no parent skill has handed anything off
  When the session continues
  Then the worker is not invoked and writes nothing
```

## 792a13b Parent skills invoke the worker by name

Requirement: [792a13b Parent skills invoke the worker by name](#792a13b-parent-skills-invoke-the-worker-by-name) A parent skill (recall or learn) MUST name write-memory as its next action at each write site and invoke it natively as a skill, and MUST NOT rely on `engram query` surfacing it by trigger or similarity.

Rationale: write-memory is reached only from a parent's own text, so a name in that text is the dependable route, as the please skill names route.

### 792a13b1 UAT Scenarios

```gherkin
Scenario: 792a13b1a A write site names the worker
  Given recall or learn reaches a write site in its instructions
  When the parent proceeds
  Then its next action is to invoke the write-memory skill with the handoff

Scenario: 792a13b1b A session without the skill follows the runbook note
  Given a session that received only the write-memory runbook note over a vault-graph edge
  When a parent reaches a write site
  Then the note is followed through the runbook shim's follow-frame, and the same handoff contract applies
```

## 792a13c The handoff and its missing fields

Requirement: [792a13c The handoff and its missing fields](#792a13c-the-handoff-and-its-missing-fields) The worker MUST accept a handoff of a kind (`fact`, `feedback`, `qa` or `runbook`), the content fields of that kind and a human-readable source, with optional chunk-sources, tags, supersedes, placement (position and target), red flags and triggers. When a required field is missing it MUST ask the in-session parent for it and MUST NOT invent content on the parent's behalf.

### 792a13c1 UAT Scenarios

```gherkin
Scenario: 792a13c1a A missing field is requested
  Given a feedback handoff with no action field
  When the worker runs
  Then it asks the parent for the action and runs no command until it is supplied

Scenario: 792a13c1b Content is never invented
  Given a fact handoff with a situation, a subject and a predicate but no object
  When the worker runs
  Then it does not supply an object of its own
```

## 792a13d Fact and feedback notes

Requirement: [792a13d Fact and feedback notes](#792a13d-fact-and-feedback-notes) The worker MUST compose a feedback handoff as `engram learn feedback` with `--slug`, `--position`, `--source`, `--situation`, `--behavior`, `--impact` and `--action`, and a fact handoff as `engram learn fact` with `--slug`, `--position`, `--source`, `--situation`, `--subject`, `--predicate` and `--object`, each followed by one `--tag` per handed-off tag. The situation MUST be the retrieval-shaped phrase the parent supplied.

### 792a13d1 UAT Scenarios

```gherkin
Scenario: 792a13d1a Feedback composition
  Given a feedback handoff with situation, behavior, impact, action and source
  When the worker composes the command
  Then it is engram learn feedback --slug <kebab-slug> --position top --source="<source>" --situation="<situation>" --behavior="<behavior>" --impact="<impact>" --action="<action>"

Scenario: 792a13d1b Fact composition
  Given a fact handoff with situation, subject, predicate, object and source
  When the worker composes the command
  Then it is engram learn fact --slug <kebab-slug> --position top --source="<source>" --situation="<situation>" --subject="<subject>" --predicate="<predicate>" --object="<object>"
```

## 792a13e Runbook notes and their red flags

Requirement: [792a13e Runbook notes and their red flags](#792a13e-runbook-notes-and-their-red-flags) The worker MUST compose a runbook handoff as `engram learn runbook` with `--slug`, `--position`, `--source`, `--situation`, `--done-when` and `--body`, with one `--red-flag` per handed-off red flag in the handed-off order, and with no `--red-flag` when the handoff carries none.

### 792a13e1 UAT Scenarios

```gherkin
Scenario: 792a13e1a A runbook handoff is executed
  Given learn hands off a confirmed runbook with slug, situation, done-when, body, source, position and two red flags
  When the worker runs
  Then it runs engram learn runbook with one --red-flag for each, in order, and reports the written note path

Scenario: 792a13e1b No red flags, no flag
  Given a runbook handoff with no red flags
  When the worker composes the command
  Then the command has no --red-flag, and no error or empty flag results
```

## 792a13f Question-and-answer notes

Requirement: [792a13f Question-and-answer notes](#792a13f-question-and-answer-notes) The worker MUST compose a qa handoff as `engram learn qa` with `--slug`, `--question`, `--answer`, `--certainty` (`high`, `medium` or `low`) and `--source`, with one `--contributors` per handed-off contributor basename, copying the answer as handed off.

### 792a13f1 UAT Scenarios

```gherkin
Scenario: 792a13f1a QA composition
  Given a qa handoff with slug, question, answer, contributors, certainty and source
  When the worker composes the command
  Then it is engram learn qa with --slug, --question, --answer, --contributors, --certainty and --source, the answer copied rather than re-derived
```

## 792a13g Placement

Requirement: [792a13g Placement](#792a13g-placement) The worker MUST pass `--position top` when the handoff carries no placement, MUST pass the handed-off `--position` (`top`, `continuation` or `sibling`) when it carries one, and MUST pass `--target <id>` only when the position is `continuation` or `sibling`, never alongside `top`. Placement applies to fact, feedback and runbook handoffs.

### 792a13g1 UAT Scenarios

```gherkin
Scenario: 792a13g1a No placement means top
  Given a fact handoff with no position or target
  When the worker composes the command
  Then it has --position top and no --target

Scenario: 792a13g1b A continuation carries its target
  Given a feedback handoff with position continuation and target 12.3
  When the worker composes the command
  Then it has --position continuation --target 12.3

Scenario: 792a13g1c Top never carries a target
  Given a handoff with position top and a stray target
  When the worker composes the command
  Then it has --position top and no --target
```

## 792a13h Free-text flags are written with the value touching the equals sign

Requirement: [792a13h Free-text flags are written with the value touching the equals sign](#792a13h-free-text-flags-are-written-with-the-value-touching-the-equals-sign) The worker MUST write every free-text flag value (for example `--situation`, `--source`, `--subject`, `--behavior`) as `--flag="value"`, with the quote touching the `=` and no space.

Rationale: a value that begins with `-` is otherwise read as a new flag, and the command fails.

### 792a13h1 UAT Scenarios

```gherkin
Scenario: 792a13h1a A value beginning with a dash survives
  Given a feedback handoff whose behavior begins with "-"
  When the worker composes the command
  Then the flag is written --behavior="-..." and the command runs without a misread flag
```

## 792a13i Provenance, tags, supersession and contributors

Requirement: [792a13i Provenance, tags, supersession and contributors](#792a13i-provenance-tags-supersession-and-contributors) The worker MUST append one `--chunk-source <source#anchor>` per handed-off chunk ID, one `--tag <family>/<value>` (or a bare `--tag <family>`) per handed-off tag for fact, feedback and runbook handoffs, exactly as provided, and `--supersedes="<basename>|<type>|<claim>"` for each handed-off supersession (type `updates`, `narrows` or `refutes`). It MUST NOT invent tags and MUST NOT write the `vocab/` namespace.

Rationale: the binary assigns `vocab/` terms itself and writes the `Supersedes:` body line when `--supersedes` is passed. Handed-off categorical tags travel in the same `tags` list as vocabulary terms but are not vocabulary.

### 792a13i1 UAT Scenarios

```gherkin
Scenario: 792a13i1a Chunk sources become flags
  Given a handoff with chunk-sources s#a1 and s#a2
  When the worker composes the command
  Then the command carries --chunk-source s#a1 --chunk-source s#a2

Scenario: 792a13i1b Tags are passed through exactly
  Given a feedback handoff with tags work-kind/rename and outcome/pass
  When the worker composes the command
  Then it carries --tag work-kind/rename --tag outcome/pass and no tag of the worker's own

Scenario: 792a13i1c A supersession is passed as given
  Given a handoff that supersedes note 12.3-old with type refutes and a claim
  When the worker composes the command
  Then it carries --supersedes="12.3-old|refutes|<claim>"
```

## 792a13j A qa note takes no tags

Requirement: [792a13j A qa note takes no tags](#792a13j-a-qa-note-takes-no-tags) When a qa handoff carries tags, the worker MUST drop them and MUST append the exact line `tags dropped: qa takes no tag flags` to its output, even when the output is only the command.

### 792a13j1 UAT Scenarios

```gherkin
Scenario: 792a13j1a QA tags are dropped and reported
  Given a qa handoff with the tag work-kind/review
  When the worker runs
  Then the command has no --tag and the output includes tags dropped: qa takes no tag flags
```

## 792a13k Kinds are never mixed, and the binary's own notation is not hand-authored

Requirement: [792a13k Kinds are never mixed, and the binary's own notation is not hand-authored](#792a13k-kinds-are-never-mixed-and-the-binarys-own-notation-is-not-hand-authored) The worker MUST NOT mix fact flags (`--subject`, `--predicate`, `--object`), feedback flags (`--behavior`, `--impact`, `--action`) and runbook flags (`--done-when`, `--body`, `--red-flag`, `--trigger`) in one command, and MUST NOT hand-author a `vocab/<term>` tag or a `Supersedes:` backlink. An inline `[[basename]]` citation MAY be typed directly into a handed-off `--body` or `--object` when the handoff content calls for citing a note.

### 792a13k1 UAT Scenarios

```gherkin
Scenario: 792a13k1a One kind's flags per command
  Given a fact handoff
  When the worker composes the command
  Then it has no feedback or runbook flag

Scenario: 792a13k1b A cited note is written inline
  Given a runbook handoff whose body cites [[12.3-old]]
  When the worker composes the command
  Then the --body carries [[12.3-old]] as typed
```

## 792a13l Runbook triggers are passed through

Requirement: [792a13l Runbook triggers are passed through](#792a13l-runbook-triggers-are-passed-through) The worker MUST append one `--trigger="<cue>"` per handed-off trigger, in order, after any `--red-flag`, and MUST emit no `--trigger` when the handoff carries none.

### 792a13l1 UAT Scenarios

```gherkin
Scenario: 792a13l1a Triggers become flags
  Given a runbook handoff with triggers /please and take this end-to-end
  When the worker composes the command
  Then the command ends with --trigger="/please" --trigger="take this end-to-end", after any --red-flag

Scenario: 792a13l1b No triggers, no flag
  Given a runbook handoff with no triggers
  When the worker composes the command
  Then the command has no --trigger
```

## 792a13m Trigger cues are specific, author-chosen strings

Requirement: [792a13m Trigger cues are specific, author-chosen strings](#792a13m-trigger-cues-are-specific-author-chosen-strings) The worker MUST treat a trigger as a cue word or phrase that the user, or an engram notice, would literally write (a slash form, a multi-word phrase, or a single distinctive word of at least three characters), matched case-insensitively, with whitespace collapsed, and as a whole word at letter and digit edges. When a handed-off single word is common in ordinary prose, the worker MUST ask the parent for a distinctive cue instead of passing it through, unless the handoff states that over-firing is a deliberate, accepted choice, in which case the worker MUST pass it through.

Rationale: a trigger hit only makes the runbook a candidate, and the agent still judges it against the runbook's own `situation`, so over-firing is tolerable and triggers never replace a well-written situation. A text-only skill test (12 fresh runs per arm) found the old skill dropped triggers silently and never questioned a lone common word, while the revised skill passed the trigger scenarios 12 of 12 (`dev/eval/LEDGER.md#write-memory-triggers-contract`, 2026-09-20).

### 792a13m1 UAT Scenarios

```gherkin
Scenario: 792a13m1a A distinctive bare word is composed without asking
  Given a runbook handoff with triggers curate, /curate and pending offers
  When the worker composes the command
  Then it carries --trigger="curate" --trigger="/curate" --trigger="pending offers" and asks for nothing

Scenario: 792a13m1b A very common bare word is questioned
  Given a runbook handoff with the trigger fix and no statement that over-firing is deliberate
  When the worker runs
  Then it asks the parent for a distinctive cue and does not emit --trigger="fix"

Scenario: 792a13m1c A deliberate over-fire is passed through
  Given a runbook handoff with the trigger please and a statement that over-firing is a deliberate, accepted choice
  When the worker composes the command
  Then it carries --trigger="please"
```

## 792a13n The write is verified and reported

Requirement: [792a13n The write is verified and reported](#792a13n-the-write-is-verified-and-reported) The worker MUST run the composed command and, when it succeeds, MUST report the printed note path or paths to the parent in one line. When the command fails with a CLI error, the worker MUST read the error, correct exactly the problem it names, and retry at most twice. When the command still fails after the retries, it MUST report the exact command and the CLI error verbatim, and MUST NOT silently skip a handed-off write.

### 792a13n1 UAT Scenarios

```gherkin
Scenario: 792a13n1a Success is reported in one line
  Given a handoff whose command succeeds
  When the worker finishes
  Then it reports the printed note path to the parent in one line

Scenario: 792a13n1b A named error is corrected and retried
  Given a command that fails because a flag is misspelled
  When the worker reads the error
  Then it fixes that flag and retries, at most twice

Scenario: 792a13n1c Persistent failure is reported verbatim
  Given a command that still fails after two retries
  When the worker stops
  Then it reports the exact command and the CLI error verbatim, and the write is not silently skipped
```
