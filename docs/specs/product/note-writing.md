# Note Writing - Product Spec

`engram learn` writes new notes into the vault: `fact`, `feedback` and `runbook` notes, each placed
in the vault's Luhmann hierarchy, and `qa` pairs of a question note and an answer note. Each note's
`vault:` field takes the vault name of [792a2b Vault name](./vault-identity.md#792a2b-vault-name). Changing an existing note is the note-revision
capability ([note-revision](./note-revision.md)); deciding which lessons to capture, and how to phrase them, belongs to the [learn](./learn.md) and
[write-memory](./write-memory.md) skills.

## 792a3a Fact notes

Requirement: [792a3a Fact notes](#792a3a-fact-notes) `engram learn fact` MUST write one note named
`<luhmann>.<YYYY-MM-DD>.<slug>.md` at the vault root, whose frontmatter carries `type: fact`, the
situation, subject, predicate and object, the Luhmann ID, the creation date and the source, and
whose body reads `Information learned: when in <situation>, <subject> <predicate> <object>.` with
any leading `When ` dropped from the situation, and MUST print the note's path on stdout.

### 792a3a1 UAT Scenarios

```gherkin
Scenario: 792a3a1a A fact note is written
  Given an empty vault
  When the user runs engram learn fact --slug ctx-fact --position top --source "smoke test" --situation "concurrent Go code" --subject goroutines --predicate "leak when" --object "ctx is ignored"
  Then stdout prints the path of 1.<today>.ctx-fact.md at the vault root, whose frontmatter has type: fact and whose body reads "Information learned: when in concurrent Go code, goroutines leak when ctx is ignored."

Scenario: 792a3a1b A leading When is not doubled
  Given an empty vault
  When the user runs engram learn fact with --situation "When testing" --subject s --predicate p --object o
  Then the note's body reads "Information learned: when in testing, s p o."
```

## 792a3b Feedback notes

Requirement: [792a3b Feedback notes](#792a3b-feedback-notes) `engram learn feedback` MUST write one note with the same filename shape, whose
frontmatter carries `type: feedback`, the situation, behavior, impact and action, the Luhmann ID,
the creation date and the source, and whose body reads `Lesson learned: when <situation>, <action>.`
with any leading `When ` dropped from the situation, and MUST print the note's path on stdout.

### 792a3b1 UAT Scenarios

```gherkin
Scenario: 792a3b1a A feedback note is written
  Given an empty vault
  When the user runs engram learn feedback --slug ctx-rule --position top --source "smoke test" --situation "writing concurrent Go code" --behavior "ignoring ctx" --impact "leaks goroutines" --action "check ctx.Done()"
  Then stdout prints the path of 1.<today>.ctx-rule.md, whose frontmatter has type: feedback and whose body reads "Lesson learned: when writing concurrent Go code, check ctx.Done()."

Scenario: 792a3b1b A leading When is not doubled
  Given an empty vault
  When the user runs engram learn feedback with --situation "When x" --action a
  Then the note's body reads "Lesson learned: when x, a."
```

## 792a3c Runbook notes

Requirement: [792a3c Runbook notes](#792a3c-runbook-notes) `engram learn runbook` MUST write one note with the same filename shape, whose
frontmatter carries `type: runbook`, the situation, `done_when`, the Luhmann ID, the creation date
and the source, and whose body is the `--body` steps verbatim, and MUST print the note's path on
stdout.

Rationale: confirmed approaches used to live in a feedback note's single `action` sentence, with no
way to represent an ordered procedure or an ending condition. A runbook answers exactly three
questions: when to use it (`situation`, the retrieval handle), what the steps are (the body, which
may link the fact and feedback notes to consider), and what should be true when done (`done_when`,
which also makes runbooks scorable for later efficacy tracking). Joe chose the name `runbook` over
`procedure`, `function` and `strategy` (vault note 789). Runbooks use the fact and feedback filename
and Luhmann placement rather than the `qa` scheme: a top-level position needs no target, so "no
related note yet" never required a separate scheme, and `qa` is the one kind built to skip the
hierarchy and be partly excluded from retrieval.
A kind-prefixed flat filename was an earlier draft, corrected in design review.

### 792a3c1 UAT Scenarios

```gherkin
Scenario: 792a3c1a A runbook note is written
  Given an empty vault
  When the user runs engram learn runbook --slug rb --position top --source src --situation "releasing a build" --done-when "the tag is pushed" --body "1. Run the checks"
  Then stdout prints the path of 1.<today>.rb.md, whose frontmatter has type: runbook, situation and done_when, and whose body is "1. Run the checks"
```

## 792a3d Runbook red flags and triggers

Requirement: [792a3d Runbook red flags and triggers](#792a3d-runbook-red-flags-and-triggers) `engram learn runbook` MUST write each `--red-flag` value, in order, to a
`red_flags:` list and each `--trigger` value, in order, to a `triggers:` list after it, and MUST
write neither field when its flag is absent.

Rationale: a red flag names a failure mode specific to this procedure that a general "follow the
steps" rule would not catch; the general floor is not restated on each runbook. How triggers
surface a runbook belongs to the memory-query capability ([memory-query](./memory-query.md)).
`red_flags` is a structured field and not a body heading, because the follow frame treats it as a standing
condition to recheck and so it carries weight that template wording does not. Rewording or bolding body text
was tried and rejected: six rounds of the eval had already exercised that lever.

### 792a3d1 UAT Scenarios

```gherkin
Scenario: 792a3d1a Red flags are kept in order
  Given an empty vault
  When the user runs engram learn runbook with --red-flag "A" --red-flag "B"
  Then the note's red_flags: lists A then B

Scenario: 792a3d1b Triggers follow the red flags in order
  Given an empty vault
  When the user runs engram learn runbook with --red-flag "A" --trigger "cue one" --trigger "cue two"
  Then the note's triggers: lists "cue one" then "cue two", after red_flags:

Scenario: 792a3d1c Neither flag means neither field
  Given an empty vault
  When the user runs engram learn runbook with no --red-flag and no --trigger
  Then the note is written with no red_flags: and no triggers: field
```

## 792a3e No task-type or input fields

Requirement: [792a3e No task-type or input fields](#792a3e-no-task-type-or-input-fields) `engram learn runbook` MUST NOT accept a task-type or input-signature field.

Rationale: a `task_type` field with a query pre-filter was planned on the strength of benchmark gains
for a whole bundled system, none of which isolated the classification step, so it could not justify a
new ranking mechanism and its regression risk; the field was dropped with the filter. An input
signature was considered during the #728 adapter experiment and rejected.

### 792a3e1 UAT Scenarios

```gherkin
Scenario: 792a3e1a A task type is refused
  Given an existing vault
  When the user runs engram learn runbook with --task-type foo
  Then it fails naming --task-type as an undefined flag, and no note is written
```

## 792a3f Required content

Requirement: [792a3f Required content](#792a3f-required-content) `engram learn fact`, `feedback` and `runbook` MUST refuse to write a note whose
`--situation` is missing or blank, and `engram learn runbook` MUST also refuse one whose
`--done-when` is missing or blank, with an error naming the missing flag.

Rationale: the situation is the note's retrieval handle. Recall finds a note by a query phrased as
its situation, so a note without one cannot be recalled that way.

### 792a3f1 UAT Scenarios

```gherkin
Scenario: 792a3f1a A missing situation is refused
  Given an existing vault
  When the user runs engram learn fact without --situation
  Then it fails naming --situation, and no note is written

Scenario: 792a3f1b A blank situation is refused
  Given an existing vault
  When the user runs engram learn feedback with --situation "  "
  Then it fails saying --situation is required, and no note is written

Scenario: 792a3f1c A runbook without done-when is refused
  Given an existing vault
  When the user runs engram learn runbook without --done-when
  Then it fails naming --done-when, and no note is written
```

## 792a3g Argument validation

Requirement: [792a3g Argument validation](#792a3g-argument-validation) `engram learn fact`, `feedback` and `runbook` MUST require `--source` and a
`--slug` matching `[a-z0-9-]+`, and MUST refuse, before writing anything, a `--project` that is not
kebab-case, an `--issue` containing whitespace, a `--tag` that is not `<family>` or
`<family>/<value>` with kebab-case segments, a `--trigger` shorter than three characters after
trimming, a `--tier` other than `L1`, `L2` or `L3`, and a `--supersedes` that is not
`<note>|<type>|<claim>` with a type of `updates`, `narrows` or `refutes`.

### 792a3g1 UAT Scenarios

```gherkin
Scenario Outline: 792a3g1a An invalid argument is refused before any write
  Given an existing vault
  When the user runs engram learn <kind> with <argument>
  Then it fails with an error saying <problem>, and no note is written

  Examples:
    | kind    | argument               | problem                                         |
    | fact    | no --source            | --source is a missing required flag             |
    | fact    | no --slug              | slug is required                                |
    | fact    | --slug X_y             | slug must match [a-z0-9-]+                      |
    | fact    | --project A            | project slug must match [a-z0-9-]+              |
    | fact    | --issue "a b"          | issue must be non-empty with no whitespace      |
    | fact    | --tag "A B"            | tag must be <family> or <family>/<value>        |
    | runbook | --trigger ab           | trigger must be at least 3 characters           |
    | fact    | --tier L9              | tier must be L2, or L1 or L3                    |
    | fact    | --supersedes "a\|bogus\|c" | supersedes type must be updates, narrows or refutes |
```

## 792a3h Optional metadata

Requirement: [792a3h Optional metadata](#792a3h-optional-metadata) `engram learn fact`, `feedback` and `runbook` MUST write a given `--project` to
`project:`, `--issue` to `issue:`, each `--tag` to `tags:`, each `--chunk-source` to `sources:`, and
each `--supersedes` both to a `supersedes:` entry and to a body line
`Supersedes: [[<note>]] — <type>: <claim>`, and MUST omit each of these fields when its flag is
absent.

Rationale: `learn` records `--chunk-source` IDs without checking the chunk index, because the chunks
a note is written from may not be indexed yet when it is written. `engram amend`, which runs later,
checks them.

### 792a3h1 UAT Scenarios

```gherkin
Scenario: 792a3h1a Optional metadata is written
  Given an empty vault
  When the user runs engram learn runbook with --project proj --issue 12 --tag fam/val --chunk-source "a#b" --supersedes "1.x|updates|claim"
  Then the note carries project: proj, issue: "12", tags: [fam/val], sources: [a#b], a supersedes: entry for 1.x of type updates, and the body line "Supersedes: [[1.x]] — updates: claim"

Scenario: 792a3h1b Absent metadata writes no field
  Given an empty vault
  When the user runs engram learn fact with none of those flags
  Then the note has no project:, issue:, tags:, sources: or supersedes: field
```

## 792a3i Tier field

Requirement: [792a3i Tier field](#792a3i-tier-field) `engram learn fact`, `feedback` and `runbook` MUST write `tier: L2` when `--tier`
is absent, and the given value when `--tier` is `L1`, `L2` or `L3`.

Rationale: the three-tier model is retired; `L1` and `L3` are still accepted so that hand-edited notes
carrying them stay valid.

### 792a3i1 UAT Scenarios

```gherkin
Scenario: 792a3i1a The tier defaults to L2
  Given an empty vault
  When the user runs engram learn fact or feedback without --tier
  Then the note carries tier: L2

Scenario: 792a3i1b A legacy tier is written as given
  Given an empty vault
  When the user runs engram learn fact with --tier L3
  Then the note carries tier: L3
```

## 792a3j Luhmann placement

Requirement: [792a3j Luhmann placement](#792a3j-luhmann-placement) `engram learn fact`, `feedback` and `runbook` MUST give the new note, for
`--position top`, the top-level number one above the highest existing one (`1` when there is
none); for `--position continuation`, the next child ID of `--target` after its highest existing
child; and for `--position sibling`, the next child ID of the target's parent after its highest
existing child. A child of an ID ending in a number ends in the letter after the highest existing
letter child (`a` to `z`, then `aa`), and a child of an ID ending in a letter ends in one more than
the highest existing numeric child. A gap left by a missing ID is never refilled.

Rationale: runbooks take part in the hierarchy exactly as facts and feedback do, with no
kind-specific opt-out.

### 792a3j1 UAT Scenarios

```gherkin
Scenario: 792a3j1a Top takes the next top-level number
  Given a vault whose notes have IDs 1, 2 and 2a
  When the user runs engram learn fact --position top
  Then the new note's ID is 3

Scenario: 792a3j1b A continuation of a number takes the next letter
  Given a vault whose notes have IDs 1 and 1a
  When the user runs engram learn fact --position continuation --target 1
  Then the new note's ID is 1b

Scenario: 792a3j1c A continuation of a letter takes the next number
  Given a vault whose note has ID 1a
  When the user runs engram learn runbook --position continuation --target 1a
  Then the new note's ID is 1a1

Scenario: 792a3j1d A sibling takes the next child of the target's parent
  Given a vault whose notes have IDs 1, 1a and 1a1
  When the user runs engram learn feedback --position sibling --target 1a1
  Then the new note's ID is 1a2

Scenario: 792a3j1e Letters roll over from z to aa
  Given a vault whose notes have IDs 1 and 1a through 1z
  When the user runs engram learn fact --position continuation --target 1
  Then the new note's ID is 1aa
```

## 792a3k Placement refusals

Requirement: [792a3k Placement refusals](#792a3k-placement-refusals) `engram learn fact`, `feedback` and `runbook` MUST refuse, with no note written,
a missing or unknown `--position`, a `continuation` or `sibling` without `--target`, a `--target`
that is not a Luhmann ID, and a `sibling` of a top-level ID, which must use `--position top`
instead.

### 792a3k1 UAT Scenarios

```gherkin
Scenario: 792a3k1a A missing position is refused
  Given an existing vault
  When the user runs engram learn fact without --position
  Then it fails saying the position must be top, continuation or sibling, and no note is written

Scenario: 792a3k1b A continuation without a target is refused
  Given an existing vault
  When the user runs engram learn fact --position continuation without --target
  Then it fails saying a target is required, and no note is written

Scenario: 792a3k1c A sibling of a top-level note is refused
  Given a vault whose note has ID 1
  When the user runs engram learn fact --position sibling --target 1
  Then it fails saying a sibling of a top-level note requires --position top, and no note is written

Scenario: 792a3k1d A malformed target is refused
  Given an existing vault
  When the user runs engram learn fact --position continuation --target abc
  Then it fails saying the ID must start with a digit, and no note is written
```

## 792a3l Placement under an absent target

Requirement: [792a3l Placement under an absent target](#792a3l-placement-under-an-absent-target) `engram learn` MAY place a note under a `--target` that names no existing note.

## 792a3m Concurrent writes never collide

Requirement: [792a3m Concurrent writes never collide](#792a3m-concurrent-writes-never-collide) Two `engram learn` runs against one vault at the same time MUST give their notes
distinct Luhmann IDs and MUST NOT overwrite each other's note.

Rationale: before every vault writer took the vault lock, two concurrent writers corrupted vault and
index state in production. The guarantee matters most when many sub-recalls write the vault in
parallel.

### 792a3m1 UAT Scenarios

```gherkin
Scenario: 792a3m1a Concurrent learns get distinct IDs
  Given an existing vault
  When two engram learn fact --position top runs start at the same time
  Then two notes exist with different Luhmann IDs, each holding its own content
```

## 792a3n Embed on write

Requirement: [792a3n Embed on write](#792a3n-embed-on-write) Every note `engram learn` writes MUST have a sibling `.vec.json` embedding
sidecar, named after the note, by the time the command returns. When embedding the note fails, the
note MUST still be written, and the command MUST print a warning naming the note and succeed.

Rationale: each note carries its own sidecar, so there is no separate index to build or keep in
sync. A missing sidecar is recoverable later, because the note itself was written in one atomic
step.

### 792a3n1 UAT Scenarios

```gherkin
Scenario: 792a3n1a A learned note has its sidecar
  Given an empty vault
  When the user runs engram learn fact, feedback, runbook or qa
  Then every note written has a .vec.json sidecar beside it, and engram check reports no note missing a sidecar

Scenario: 792a3n1b A failed embedding keeps the note
  Given embedding fails for the note being learned
  When the user runs engram learn fact
  Then the note is written, stderr carries a warning naming it, and the command succeeds
```

## 792a3o Repository provenance

Requirement: [792a3o Repository provenance](#792a3o-repository-provenance) `engram learn fact`, `feedback` and `runbook` MUST stamp `repo:` with the
working directory's git `origin` remote URL, or, when there is no `origin` remote, with the basename
of the working directory's git top-level directory, and MUST omit `repo:` outside a git repository.

Rationale: the free-text `source:` field says what a note came from, but not which repository,
person or vault produced it. Structured, auto-detected provenance lets filtering, attribution and
parent exchange rely on a note's origin. Detection is self-reported by the writer's environment,
not verified. Being outside a git repository is not an error, so the field is simply omitted.

### 792a3o1 UAT Scenarios

```gherkin
Scenario: 792a3o1a The origin remote names the repo
  Given a working directory inside a git repository whose origin is git@example.com:o/r.git
  When the user runs engram learn fact
  Then the note's repo: is git@example.com:o/r.git

Scenario: 792a3o1b Without an origin the top-level directory names the repo
  Given a working directory inside a git repository named proj with no origin remote
  When the user runs engram learn fact
  Then the note's repo: is proj

Scenario: 792a3o1c Outside a git repository there is no repo
  Given a working directory outside any git repository
  When the user runs engram learn fact
  Then the note has no repo: field
```

## 792a3p User provenance

Requirement: [792a3p User provenance](#792a3p-user-provenance) `engram learn fact`, `feedback` and `runbook` MUST stamp `user:` with
`git config user.email`, or, when that resolves to nothing, with the OS username. When neither
resolves, the note MUST be written without a `user:` field, never with an empty one, and the command
MUST print one warning that user detection resolved empty.

### 792a3p1 UAT Scenarios

```gherkin
Scenario: 792a3p1a The git email names the user
  Given git config user.email resolves to a@example.com
  When the user runs engram learn fact
  Then the note's user: is a@example.com

Scenario: 792a3p1b The OS username is the fallback
  Given git config user.email resolves to nothing and the OS username is joe
  When the user runs engram learn fact
  Then the note's user: is joe

Scenario: 792a3p1c No detectable user
  Given neither git config user.email nor the OS username resolves
  When the user runs engram learn fact
  Then the note has no user: field, vault: is stamped as usual, and stderr carries one warning that user detection resolved empty
```

## 792a3q Question-and-answer pairs

Requirement: [792a3q Question-and-answer pairs](#792a3q-question-and-answer-pairs) `engram learn qa` MUST write two notes at the vault root,
`qa.<YYYY-MM-DD>.<slug>.q.md` and `qa.<YYYY-MM-DD>.<slug>.a.md`, and print both paths, the question's
first. The question note's frontmatter carries `type: qa-question`, `date`,
`answered_by: <answer basename>` and `source`, and its body is the question followed by the line
`Answered by: [[<answer basename>]]`. The answer note's frontmatter carries `type: qa-answer`,
`date`, `answers: <question basename>`, `certainty`, `contributors` when any are given, and
`source`, and its body is the answer, from `--answer` or the file named by `--answer-file`, followed
by the line `Answers: [[<question basename>]]` and, when contributors are given, the line
`Contributors: [[<first>]], [[<second>]]`.

Rationale: an answer is a reasoned conclusion with provenance, while a question is situational
wording that measurably loses retrieval against notes with content: re-anchoring notes to their
questions gave no delivery benefit and lost retrieval in 10 of 10 cases. So the pair is linked both
ways, the answer competes for retrieval, and the question is reachable only through its answer
([792a3t Question-and-answer retrieval](#792a3t-question-and-answer-retrieval)). An earlier design excluded both halves.

### 792a3q1 UAT Scenarios

```gherkin
Scenario: 792a3q1a A pair is captured
  Given a vault holding note 1.2026-10-09.a-fact
  When the user runs engram learn qa --slug why-x --question "Why X?" --answer "Because." --source src --contributors 1.2026-10-09.a-fact
  Then stdout prints the paths of qa.<today>.why-x.q.md and then qa.<today>.why-x.a.md, the question note ends "Answered by: [[qa.<today>.why-x.a]]", and the answer note ends "Answers: [[qa.<today>.why-x.q]]" and "Contributors: [[1.2026-10-09.a-fact]]"

Scenario: 792a3q1b A pair without contributors
  Given an existing vault
  When the user runs engram learn qa without --contributors
  Then the answer note has no contributors: field and no Contributors: line

Scenario: 792a3q1c The answer can come from a file
  Given a file holding the answer text
  When the user runs engram learn qa with --answer-file naming it
  Then the answer note's body is that file's text
```

## 792a3r Question-and-answer validation

Requirement: [792a3r Question-and-answer validation](#792a3r-question-and-answer-validation) `engram learn qa` MUST refuse, before writing either note, a missing or invalid
`--slug`, a missing or blank `--question`, anything other than exactly one of `--answer` and
`--answer-file`, a missing `--source`, a `--certainty` other than `high`, `medium` or `low`, and a
`--contributors` value that is not the basename of a note in the vault. An absent `--certainty`
MUST be written as `medium`.

### 792a3r1 UAT Scenarios

```gherkin
Scenario Outline: 792a3r1a An invalid pair is refused before any write
  Given an existing vault
  When the user runs engram learn qa with <argument>
  Then it fails with an error saying <problem>, and neither note is written

  Examples:
    | argument                       | problem                                      |
    | no --question                  | --question is required                       |
    | both --answer and --answer-file | exactly one of --answer or --answer-file is required |
    | neither --answer nor --answer-file | exactly one of --answer or --answer-file is required |
    | --certainty huge               | certainty must be high, medium, or low       |
    | --contributors nope            | contributor not found in vault               |
    | --slug X_y                     | slug must match [a-z0-9-]+                   |
    | no --source                    | --source is a missing required flag          |

Scenario: 792a3r1b Certainty defaults to medium
  Given an existing vault
  When the user runs engram learn qa without --certainty
  Then the answer note carries certainty: medium
```

## 792a3s Question-and-answer partial failure

Requirement: [792a3s Question-and-answer partial failure](#792a3s-question-and-answer-partial-failure) When the answer note cannot be written, `engram learn qa` MUST remove the
question note it has just written and fail with an error saying the answer note failed.

### 792a3s1 UAT Scenarios

```gherkin
Scenario: 792a3s1a A failed answer removes the question
  Given a vault that already holds a file named like today's answer note for slug why-x
  When the user runs engram learn qa --slug why-x
  Then it fails saying the answer note could not be written, and no question note for why-x remains
```

## 792a3t Question-and-answer retrieval

Requirement: [792a3t Question-and-answer retrieval](#792a3t-question-and-answer-retrieval) `engram query` MUST return answer notes as it returns any other note, and MUST
NOT return question notes.

### 792a3t1 UAT Scenarios

```gherkin
Scenario: 792a3t1a The answer is found and the question is not
  Given a vault holding a pair whose question and answer both match a phrase
  When the user runs engram query with that phrase
  Then the answer note can appear in the results, and the question note never does
```

## 792a3u Vocabulary tagging by kind

Requirement: [792a3u Vocabulary tagging by kind](#792a3u-vocabulary-tagging-by-kind) The vault's write-time vocabulary tagging MUST apply to every fact, feedback,
runbook and answer note `engram learn` writes, and MUST NOT apply to question notes.

Rationale: the answer is the half that competes for retrieval, so it is tagged like any other note;
the question carries no vocabulary, so it stays reachable only through its answer.

### 792a3u1 UAT Scenarios

```gherkin
Scenario: 792a3u1a A runbook is tagged
  Given a vault with a vocabulary term close to a runbook's text
  When the user runs engram learn runbook with that text
  Then the runbook carries the term's vocab tag

Scenario: 792a3u1b The answer is tagged and the question is not
  Given a vault with a vocabulary term close to both a question and its answer
  When the user runs engram learn qa with them
  Then the answer note carries the term's vocab tag, and the question note has no tags: field
```
