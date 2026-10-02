## ADDED Requirements

### Requirement: A step-detection regex SHALL match every valid real-world invocation form of its target command

The `runbook_vs_skill` phase2 probe's fixture `steps.json` step definitions SHALL recognize every commonly-used invocation form of the command a step is meant to detect, not only one syntactic form, when a gated later step's detection (`"after": N`) depends on it. A step that only matches one valid form of a command false-negatives every genuine success that used another valid form, and cascades that false negative to every step gated behind it.

#### Scenario: The hyphenated git-filter-repo form is matched

- **WHEN** a trial's bash history contains `git-filter-repo --path secrets.env --invert-paths --refs main --force`
- **THEN** the `history-rewrite` fixture's step 3 (and steps 4-6, gated behind it) score as followed

#### Scenario: The space-separated git filter-repo form still matches (no regression)

- **WHEN** a trial's bash history contains `git filter-repo --path secrets.env --invert-paths --refs main --force`
- **THEN** the `history-rewrite` fixture's step 3 scores as followed, exactly as before this change

#### Scenario: An unrelated command is still not matched

- **WHEN** a trial's bash history contains a command that merely contains the substring `filter-repo` without being a `git`/`git-filter-repo` invocation (e.g. `cat notes-about-filter-repo.md`)
- **THEN** the `history-rewrite` fixture's step 3 does not score as followed

#### Scenario: A broadened separator class does not create new false positives on a letter-adjacent substring

- **WHEN** a trial's bash history contains `digit-filter-repo --some-arg` or `mygit-filter-repo --some-arg` — a command whose text happens to contain the literal substring `git-filter-repo` immediately preceded by a word character, not a real git invocation
- **THEN** the `history-rewrite` fixture's step 3 does not score as followed, because the matched pattern requires a left boundary before `git` and not merely a hyphen-or-whitespace separator after it

### Requirement: `question_stop` detection SHALL be order-aware relative to task completion

The `runbook_vs_skill` phase2 probe's `detect_question_stop` SHALL only report a trial as a legitimate clarity stop when the last ambiguity-posing text in the transcript precedes the trial's task completion, not merely when no mutating tool call follows that text. A trailing, genuinely separate follow-up question asked after the trial has already completed and verified its task (per the trial's own end-state check) SHALL NOT be scored identically to a trial that stopped before finishing the task.

#### Scenario: A trailing follow-up question after full completion is not a stop

- **WHEN** a trial fully completes and verifies its task (its end-state check passes) and then, in its closing report, asks a separate follow-up question with no further mutation
- **THEN** `question_stop` is `False` for that trial

#### Scenario: A genuine pre-completion stop is still detected

- **WHEN** a trial poses an ambiguity before completing its task, performs no further mutation afterward, and its end-state check does not pass
- **THEN** `question_stop` is `True` for that trial

### Requirement: A fixture requiring a runbook-prescribed follow-up action SHALL provide a satisfiable local target for it

When a runbook a trial is following prescribes an action with no fixture-local way to perform it (for example, filing a follow-up issue when the fixture repo has no issue tracker), the fixture SHALL provide a minimal local convention satisfying that action, named in the task prompt, so a shim-following agent is never forced to stop and ask merely because the fixture omitted the means to comply. The runbook itself SHALL NOT be changed to work around a fixture gap.

#### Scenario: The bisect-before-fix fixture names a local issue-filing target

- **WHEN** a trial follows runbook 846's step 6 and needs to file a follow-up issue for a pre-existing regression in the `bisect-before-fix` fixture
- **THEN** the fixture's task prompt names `ISSUES.md` as the place to file it, and the fixture repo contains that file from its initial commit onward
