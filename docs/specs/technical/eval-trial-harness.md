# Eval Trial Harness - Technical Spec

## 792b38a Orientation

- Ownership: Repository component (dev tooling).
- Responsibility: gives an eval trial a configuration that carries the skills it claims and an environment that keeps it off
  the maintainer's real engram state, guards the real vault against a leak and tells a genuine one from the orchestrating
  session's own writes, and decides from a trial's transcript what the agent did: which steps of a runbook it followed,
  whether it stopped to ask, and whether a probed note surfaced.
- Current realization (informational): `dev/eval/isolation.py` (`install_skill`, `isolated_env`, `engram_env`,
  `assert_isolated`, `assert_engram_isolated`, `vault_fingerprint`, `assert_no_trial_leak`), the warm-configuration
  builders `dev/eval/traps/wrun.py` (`build_warm_cfg`), `dev/eval/cumulative/matrix.py` and
  `dev/eval/cumulative/runbook_vs_skill/probe.py` (`build_cfg_template`), the phase 2 probe
  `dev/eval/cumulative/runbook_vs_skill/phase2/probe_phase2.py` with its fixtures, and `dev/eval/traps/retrieval_probe.py`
  (`rank_in_payload`). The Go harness behind `targ eval` (`dev/eval/*.go`) is not specified here.
- Qualification: responsibility, the installed skills, the isolated environment, the leak guard and the scoring are stated
  without naming a function; ownership, it owns what an eval trial is given and how its transcript is read, while the
  engram binary the trial runs, the vault and the query payload it reads are other components'; independent verification,
  every requirement is tested over a temporary directory, a synthetic payload or a synthetic transcript, with no model
  call; stability, the contract survives a new harness, because each one builds its environment through one primitive;
  necessity, a trial that reached the real vault or ran without its skills would make every number the maintainer reads
  wrong without any sign of it.
- Supported capabilities: [memory-effectiveness-evaluation](../product/memory-effectiveness-evaluation.md).

Rationale: every harness here spawns `claude -p --permission-mode bypassPermissions` with the engram binary reachable, so unless
engram's state variables point at a per-trial directory a trial's `engram learn` writes to the real vault (#708) and its
`engram query` reads the real chunk index (#642). The environment is therefore built by one primitive that proves
the isolation before it returns and records the trial's marker as it does. The skill installation is likewise one
function, because three hand-mirrored copies let one of them silently skip its copy for a month (#749).

## 792b38c Behavior Requirements

### 792b38c1 Installing a skill into a trial configuration

Requirement: [792b38c1 Installing a skill into a trial configuration](#792b38c1-installing-a-skill-into-a-trial-configuration) The eval-trial-harness MUST, to install a skill into a trial configuration's `skills/` directory from a skills
root, raise an error that names the skill and the path checked when the skill's source directory does not exist, and one
that names the skill, the path and `SKILL.md` when the directory holds no `SKILL.md`. It MUST otherwise copy the skill tree to
the destination, overwriting the files already there (a file that exists only at the destination stays), and MUST raise when `SKILL.md` is not present at the destination afterwards, so that
it never installs fewer skills than it was asked to.

Supports: [792a23a1a Successful skill installation is verified](../product/memory-effectiveness-evaluation.md#792a23a1-uat-scenarios), [792a23a1b Post-copy destination content is confirmed](../product/memory-effectiveness-evaluation.md#792a23a1-uat-scenarios), [792a23b1a A missing skill source raises](../product/memory-effectiveness-evaluation.md#792a23b1-uat-scenarios), [792a23b1b An empty skill source raises](../product/memory-effectiveness-evaluation.md#792a23b1-uat-scenarios).

Rationale: a copy that silently does nothing because its source path is wrong looks the same as a successful copy of an empty
directory unless the source and the destination are each checked.

#### 792b38c1a Scenarios

```gherkin
Scenario: 792b38c1a1 A missing source raises
  Given a skills root that does not hold the skill
  When the skill is installed
  Then it raises an error that names the skill and the path checked

Scenario: 792b38c1a2 A source with no SKILL.md raises
  Given a skill directory that exists and holds no SKILL.md
  When the skill is installed
  Then it raises an error that names the skill and SKILL.md

Scenario: 792b38c1a3 The copy is verified at the destination
  Given a skill source with a SKILL.md
  When the skill is installed
  Then the destination holds the same SKILL.md, and an installation that leaves none raises
```

### 792b38c2 The warm-configuration builders

Requirement: [792b38c2 The warm-configuration builders](#792b38c2-the-warm-configuration-builders) Every function of the eval-trial-harness that builds a warm trial configuration MUST install each requested
skill through the one installation of [792b38c1 Installing a skill into a trial configuration](#792b38c1-installing-a-skill-into-a-trial-configuration), from `agent-instructions/skills/<skill>` under a repository root that it
computes from its own file's location and never from a hard-coded absolute path, so that a missing or empty source raises
through it. The cumulative harness's builder MUST install `recall` and `learn` for a warm configuration, and the probe harness's
builder MUST install the skills it is given, `recall` and `learn` by default and none for an empty list, and propagate the
choice to every configuration it copies. The cumulative and probe builders MUST skip rebuilding a configuration that is already built; the trap harness's builder rebuilds its configuration on every call.

Supports: [792a23a1c Every warm-configuration builder verifies the same way](../product/memory-effectiveness-evaluation.md#792a23a1-uat-scenarios), [792a23b1c Every builder raises on a missing source](../product/memory-effectiveness-evaluation.md#792a23b1-uat-scenarios), [792a23c1a The path resolves independent of the working directory](../product/memory-effectiveness-evaluation.md#792a23c1-uat-scenarios), [792a23c1b The cumulative harness resolves its path the same way](../product/memory-effectiveness-evaluation.md#792a23c1-uat-scenarios).

Rationale: shared installation is the fix for the drift of #749, and the repository root derived from `__file__` is what keeps the
source path right when the skills move again.

#### 792b38c2a Scenarios

```gherkin
Scenario: 792b38c2a1 The trap harness verifies a successful installation
  Given the recall and learn sources with their SKILL.md files
  When the trap harness builds a warm configuration
  Then both SKILL.md files exist under the destination's skills directory

Scenario: 792b38c2a2 The trap harness raises on a missing source
  Given a skill source that does not exist
  When the trap harness builds a warm configuration
  Then it raises an error naming the skill and the path, and no configuration claims success

Scenario: 792b38c2a3 The trap harness raises on an empty source
  Given a skill source with no SKILL.md
  When the trap harness builds a warm configuration
  Then it raises an error naming the skill and the source path

Scenario: 792b38c2a4 The cumulative harness installs the real skills
  Given the real agent-instructions skills
  When the cumulative harness builds a warm configuration
  Then recall and learn SKILL.md files exist under the destination's skills directory

Scenario: 792b38c2a5 The cumulative harness raises on a missing source
  Given a skill source that does not exist
  When the cumulative harness builds a warm configuration
  Then it raises an error naming the skill and the path instead of skipping the copy

Scenario: 792b38c2a6 The cumulative harness raises on an empty source
  Given a skill source with no SKILL.md
  When the cumulative harness builds a warm configuration
  Then it raises an error naming the skill and the source path

Scenario: 792b38c2a7 The cumulative pools are still built end to end
  Given a request for warm configuration pools
  When the pools are made
  Then each warm configuration is built with its skills

Scenario: 792b38c2a8 The probe harness keeps its default skills
  Given no skills argument
  When the probe harness builds a configuration
  Then it installs recall and learn

Scenario: 792b38c2a9 An empty skills list installs none, and a pool inherits it
  Given a skills list that is empty
  When the probe harness builds a configuration and a pool of copies
  Then no skill is installed in the template or in any copy

Scenario: 792b38c2a10 The probe harness raises on a missing source
  Given a skill source that does not exist
  When the probe harness builds a configuration
  Then it raises an error naming the skill and the path

Scenario: 792b38c2a11 The probe harness raises on an empty source
  Given a skill source with no SKILL.md
  When the probe harness builds a configuration
  Then it raises an error naming the skill and the source path

Scenario: 792b38c2a12 The repository root comes from the file's own location
  Given a builder
  When its source is read
  Then the skills root is computed from the file's own location and no absolute path is hard-coded
```

### 792b38c3 The isolated environment of a trial

Requirement: [792b38c3 The isolated environment of a trial](#792b38c3-the-isolated-environment-of-a-trial) The eval-trial-harness MUST build a trial's environment so that `ENGRAM_VAULT_PATH`, `ENGRAM_CHUNKS_DIR` and
`ENGRAM_TRANSCRIPT_DIR` point inside the per-trial directory (the transcript directory at the session directory of the trial's
working directory under the configuration's `projects/`, or at `projects/` when no working directory is given) and
`CLAUDE_CONFIG_DIR` at the trial's configuration, creating the vault and chunk directories and keeping every other entry of
the base environment. It MUST raise before it returns when a state variable is unset, when any of them resolves inside the
maintainer's engram data directory (`$XDG_DATA_HOME/engram`, else `~/.local/share/engram`), when `CLAUDE_CONFIG_DIR` is unset,
or when the working directory has an ancestor holding `.claude`, `.git`, `.hg` or `.jj`. A direct `engram` call's environment
MUST set the vault and the chunk directory (by default the vault's `.chunks` sibling) and be refused when either is the
maintainer's, without regard to the Claude variables. An environment that passes MUST register its trial directory, or its
vault, as a trial marker as it is built, safely when several are built at once.

Supports: [792a23d1a The orchestrating session's own writes do not trigger a leak report](../product/memory-effectiveness-evaluation.md#792a23d1-uat-scenarios), [792a23d1b A harness captures no trial evidence itself and still gets a correct guard](../product/memory-effectiveness-evaluation.md#792a23d1-uat-scenarios).

Rationale: a vault-only environment is the half-fix of #642: `engram query` keeps ranking against the maintainer's global chunk
index, so a probe measures his real memory competing with its own fixture. Walking the working directory's ancestors would
sweep operator-global sources into a per-trial index (vault note 296). The marker is recorded by the primitive that proves the
isolation, so no caller has to collect it (#750).

#### 792b38c3a Scenarios

```gherkin
Scenario: 792b38c3a1 Every state variable points into the trial
  Given a configuration directory, a trial directory and a working directory
  When the trial's environment is built
  Then CLAUDE_CONFIG_DIR, ENGRAM_VAULT_PATH, ENGRAM_CHUNKS_DIR and ENGRAM_TRANSCRIPT_DIR are set inside them

Scenario: 792b38c3a2 The vault and chunk directories exist
  Given a trial directory
  When the environment is built
  Then the vault and the chunk directories exist

Scenario: 792b38c3a3 The base environment is kept
  Given a base environment holding PATH and another entry
  When the environment is built
  Then both entries are preserved

Scenario: 792b38c3a4 An unset state variable is refused
  Given a built environment from which one of the three state variables is removed
  When the isolation is asserted
  Then it raises naming the variable

Scenario: 792b38c3a5 An unset configuration directory is refused
  Given a built environment without CLAUDE_CONFIG_DIR
  When the isolation is asserted
  Then it raises

Scenario: 792b38c3a6 A path inside the maintainer's data directory is refused
  Given an environment whose state variable resolves inside the maintainer's engram data directory
  When the isolation is asserted
  Then it raises

Scenario: 792b38c3a7 A working directory with a Claude or VCS ancestor is refused
  Given a working directory below a .claude directory, and another below a .git directory
  When the isolation is asserted
  Then each raises

Scenario: 792b38c3a8 A clean trial is accepted
  Given an isolated environment and a clean working directory
  When the isolation is asserted
  Then it passes

Scenario: 792b38c3a9 A direct engram call's environment is isolated
  Given a vault path and no chunk directory, and another with an explicit chunk directory
  When the engram environment is built
  Then the first defaults the chunk directory to the vault's .chunks sibling and the second uses its own, both creating it

Scenario: 792b38c3a10 The maintainer's real vault is refused for a direct call
  Given the maintainer's real vault as the vault path
  When the engram environment is built
  Then it raises

Scenario: 792b38c3a11 A direct call is judged without the Claude variables
  Given an environment with the engram variables isolated and no Claude variables
  When the engram isolation is asserted
  Then it passes

Scenario: 792b38c3a12 The data directory follows XDG
  Given XDG_DATA_HOME set, and then unset
  When the maintainer's data directory is resolved
  Then it is $XDG_DATA_HOME/engram, and otherwise ~/.local/share/engram

Scenario: 792b38c3a13 Building the environment registers the trial directory
  Given eight environments built at once from several threads
  When they are all built
  Then every trial directory is in the registry and none is dropped

Scenario: 792b38c3a14 A direct call's environment registers its vault
  Given a direct engram call's environment
  When it is built
  Then its vault is in the registry
```

### 792b38c4 The real-vault fingerprint

Requirement: [792b38c4 The real-vault fingerprint](#792b38c4-the-real-vault-fingerprint) The eval-trial-harness MUST fingerprint a vault as the number of its `.md` notes and the SHA-256 of their sorted
basenames, ignoring sidecars and every other file, and a vault directory that does not exist MUST have the empty fingerprint.
A harness's fingerprint of the maintainer's real vault MUST be taken before its trials run ([792b38c6 Every harness guards the real vault](#792b38c6-every-harness-guards-the-real-vault)).

Supports: [792a23f1c A same-filename in-place mutation is not caught](../product/memory-effectiveness-evaluation.md#792a23f1-uat-scenarios).

Rationale: the fingerprint is the cheap before-picture for the leak guard. Because it hashes names and not modification times, an
in-place edit of an existing note does not move it, a limit that [792a23f The leak guard's limits are stated](../product/memory-effectiveness-evaluation.md#792a23f-the-leak-guards-limits-are-stated) states.

#### 792b38c4a Scenarios

```gherkin
Scenario: 792b38c4a1 Notes are counted and sidecars ignored
  Given a vault with notes and sidecars
  When it is fingerprinted
  Then the count is of the .md notes and the hash is over their sorted basenames

Scenario: 792b38c4a2 A missing vault is empty
  Given a vault directory that does not exist
  When it is fingerprinted
  Then the fingerprint is that of an empty vault
```

### 792b38c5 The leak guard

Requirement: [792b38c5 The leak guard](#792b38c5-the-leak-guard) The eval-trial-harness MUST, given the fingerprint from before a run, return without a finding when the real vault's fingerprint is
unchanged and when no note was added, and otherwise read the notes that appeared and raise an error that
names the note and the marker when a note's content names the full path of a registered trial marker, or the marker's basename
when that basename is distinctive. A basename MUST be distinctive when its last segment after the final `-` is at least 8
characters, the length of the random suffix of `tempfile.mkdtemp`, whatever its characters, and a short deterministic name such as
`plumbing-0` or `history-rewrite-R-0` MUST NOT be. A note that names no registered marker MUST be treated as an orchestrator's own write and not raise.

Supports: [792a23d1a The orchestrating session's own writes do not trigger a leak report](../product/memory-effectiveness-evaluation.md#792a23d1-uat-scenarios), [792a23e1a A new real-vault note naming a trial is flagged](../product/memory-effectiveness-evaluation.md#792a23e1-uat-scenarios), [792a23e1b A note naming only a distinctive basename is still flagged](../product/memory-effectiveness-evaluation.md#792a23e1-uat-scenarios), [792a23e1c A note naming only a short deterministic basename is not flagged](../product/memory-effectiveness-evaluation.md#792a23e1-uat-scenarios), [792a23f1a A leak through an unmediated path is not caught](../product/memory-effectiveness-evaluation.md#792a23f1-uat-scenarios), [792a23f1b A leak whose content does not self-identify is not caught](../product/memory-effectiveness-evaluation.md#792a23f1-uat-scenarios), [792a23f1c A same-filename in-place mutation is not caught](../product/memory-effectiveness-evaluation.md#792a23f1-uat-scenarios), [792a23f1d A leak naming only a non-distinctive basename is not caught](../product/memory-effectiveness-evaluation.md#792a23f1-uat-scenarios).

Rationale: a whole-vault diff false-alarmed on the orchestrating session's legitimate writes (#750). The guard is single-layered
and stated as such: it misses a path that never builds its environment with the primitive, a leaked note that names no marker, a
non-distinctive basename and an in-place mutation, which the product scenarios of [792a23f The leak guard's limits are stated](../product/memory-effectiveness-evaluation.md#792a23f-the-leak-guards-limits-are-stated) state as accepted limits.

#### 792b38c5a Scenarios

```gherkin
Scenario: 792b38c5a1 A new note naming no registered marker is ignored
  Given a note added to the real vault that names no registered marker
  When the leak guard runs
  Then it does not raise

Scenario: 792b38c5a2 A new note naming a registered marker raises
  Given a note added to the real vault that names a registered trial marker's full path
  When the leak guard runs
  Then it raises naming the note and the marker

Scenario: 792b38c5a3 A short generic basename in passing is ignored
  Given a note that names only a registered marker's short deterministic basename
  When the leak guard runs
  Then it does not raise

Scenario: 792b38c5a4 A distinctive basename alone raises
  Given a note that names only a registered marker's distinctive basename
  When the leak guard runs
  Then it raises naming the note and the matched basename

Scenario: 792b38c5a5 Real mkdtemp suffixes are distinctive
  Given the random suffixes tempfile.mkdtemp produces, underscores included
  When each basename is classified
  Then each is distinctive

Scenario: 792b38c5a6 A fixed underscore-bearing suffix is judged by its last segment
  Given a basename whose last segment after the final dash is long and holds an underscore
  When it is classified
  Then it is distinctive

Scenario: 792b38c5a7 The harness's own deterministic names are not distinctive
  Given the harness's own trial directory names, such as plumbing-0, setup-only-0 and history-rewrite-R-0
  When each is classified
  Then none is distinctive

Scenario: 792b38c5a8 A leak the guard cannot see is not raised
  Given a leak through an unmediated path, a note that names no registered marker, a non-distinctive basename, or an in-place edit
  When the leak guard runs
  Then it does not raise
```

### 792b38c6 Every harness guards the real vault

Requirement: [792b38c6 Every harness guards the real vault](#792b38c6-every-harness-guards-the-real-vault) Each of the seven tier-A harnesses of [792b38c11 Every spawn site builds its environment through the primitive](#792b38c11-every-spawn-site-builds-its-environment-through-the-primitive) and each runbook-versus-skill probe MUST take the real vault's fingerprint before it runs its trials and call the leak guard after, and no harness MUST still compare a whole-vault fingerprint of its own: the retired `_real_vault_fingerprint` MUST be absent from the two files that used it. The older experiment harnesses (`dev/eval/traps/c4.py`, `c5.py`, `c6.py`, `cake.py`, `compound_eval.py`, `wrun.py` and their kin) isolate their environment by hand and take no fingerprint; they are outside this requirement.

Supports: [792a23d1c Every harness snapshots the real vault and asserts no leak](../product/memory-effectiveness-evaluation.md#792a23d1-uat-scenarios).

Rationale: a harness that forgets the guard is the failure the guard exists for, so a test over the harness sources makes the omission fail.

#### 792b38c6a Scenarios

```gherkin
Scenario: 792b38c6a1 Every harness snapshots the real vault
  Given each tier-A harness
  When its source is read
  Then it calls vault_fingerprint

Scenario: 792b38c6a2 Every harness asserts no leak
  Given each tier-A harness
  When its source is read
  Then it calls assert_no_trial_leak

Scenario: 792b38c6a3 The retired whole-vault comparison is gone
  Given the two files that used the retired fingerprint
  When their sources are read
  Then neither references _real_vault_fingerprint
```

### 792b38c7 Detecting a step in every form of its command

Requirement: [792b38c7 Detecting a step in every form of its command](#792b38c7-detecting-a-step-in-every-form-of-its-command) The eval-trial-harness MUST match the `history-rewrite` fixture's step 3 on a bash command that runs `git filter-branch`,
`git filter-repo` or `git-filter-repo`, with whitespace or a hyphen between `git` and the tool, and only when `git` is not
preceded by a word character or a hyphen, so that a command whose text merely contains the name, or ends a longer word with
`git`, does not match. A step gated behind it MUST score as followed only after it.

Supports: [792a23g1a The hyphenated git-filter-repo form is matched](../product/memory-effectiveness-evaluation.md#792a23g1-uat-scenarios), [792a23g1b The space-separated git filter-repo form still matches](../product/memory-effectiveness-evaluation.md#792a23g1-uat-scenarios), [792a23g1c An unrelated command is still not matched](../product/memory-effectiveness-evaluation.md#792a23g1-uat-scenarios), [792a23g1d A letter-adjacent substring is not matched](../product/memory-effectiveness-evaluation.md#792a23g1-uat-scenarios).

Rationale: a pattern for one valid form scores every success that used another as a miss and cascades the miss to every step behind it.

#### 792b38c7a Scenarios

```gherkin
Scenario: 792b38c7a1 The hyphenated form matches
  Given a bash command git-filter-repo --path secrets.env --invert-paths --refs main --force
  When step 3 is evaluated
  Then it matches, and a command with another word before it does not

Scenario: 792b38c7a2 The space-separated form still matches
  Given a bash command git filter-repo --path secrets.env --invert-paths --refs main --force
  When step 3 is evaluated
  Then it matches

Scenario: 792b38c7a3 Letter-adjacent substrings do not match
  Given the commands digit-filter-repo --some-arg and mygit-filter-repo --some-arg
  When step 3 is evaluated
  Then neither matches

Scenario: 792b38c7a4 A full trial with the hyphenated form scores every step as followed
  Given a synthetic trial that used the hyphenated form and then fetched, pushed and verified
  When it is scored
  Then steps 3 to 6 all score as followed
```

### 792b38c8 A question stop

Requirement: [792b38c8 A question stop](#792b38c8-a-question-stop) The eval-trial-harness MUST report `question_stop` as false when the trial's end-state check passes. Otherwise it MUST
report it as true exactly when the last assistant text that poses an ambiguity (the ambiguity wording, or a `?` anywhere) has
no mutating tool call after it, whether the transcript ends there or an `AskUserQuestion` or `ExitPlanMode` call follows, and
false when a mutation follows it or no ambiguity was ever posed. Both the live scoring and the re-scoring of a recorded trial MUST pass
the end-state result in, and the argument MUST be required so that no call site can skip it.

Supports: [792a23h1a A trailing follow-up question after full completion is not a stop](../product/memory-effectiveness-evaluation.md#792a23h1-uat-scenarios), [792a23h1b A genuine pre-completion stop is still detected](../product/memory-effectiveness-evaluation.md#792a23h1-uat-scenarios).

Rationale: if the final state already satisfies the checks, every mutation the task required has happened, so ambiguity text with no
mutation after it cannot have blocked one. An agent that raises an ambiguity and mutates anyway is a silent deviation and
never a stop.

#### 792b38c8a Scenarios

```gherkin
Scenario: 792b38c8a1 Ambiguity text ending the transcript is a stop
  Given an ambiguity text and no further calls
  When question_stop is detected with a failing end state
  Then it is true

Scenario: 792b38c8a2 A question mid-paragraph counts
  Given a text with a question in the middle of a paragraph and no mutation after it
  When question_stop is detected with a failing end state
  Then it is true

Scenario: 792b38c8a3 A later mutation means no stop
  Given an ambiguity text followed by a mutating call
  When question_stop is detected
  Then it is false

Scenario: 792b38c8a4 No ambiguity, no stop
  Given a transcript that never poses an ambiguity
  When question_stop is detected
  Then it is false

Scenario: 792b38c8a5 An AskUserQuestion after the ambiguity is a stop
  Given an ambiguity text followed by an AskUserQuestion call and no mutation
  When question_stop is detected with a failing end state
  Then it is true

Scenario: 792b38c8a6 The last ambiguity mention decides
  Given two ambiguity texts, a mutation between them and none after the second
  When question_stop is detected with a failing end state
  Then it is true

Scenario: 792b38c8a7 A trailing question after completion is not a stop
  Given an ambiguity text at the end of a transcript whose end-state check passes
  When question_stop is detected
  Then it is false

Scenario: 792b38c8a8 A failing end state with an ambiguity is a stop
  Given an ambiguity raised and a failing end-state check
  When question_stop is detected
  Then it is true

Scenario: 792b38c8a9 Live scoring wires the end state
  Given a trial scored live whose end state passes and which ends with a trailing question
  When it is scored
  Then question_stop is false

Scenario: 792b38c8a10 Re-scoring wires the end state
  Given a recorded trial re-scored whose end state passes and which ends with a trailing question
  When it is re-scored
  Then question_stop is false
```

### 792b38c9 A fixture's local follow-up target

Requirement: [792b38c9 A fixture's local follow-up target](#792b38c9-a-fixtures-local-follow-up-target) The `bisect-before-fix` fixture of the eval-trial-harness MUST contain `ISSUES.md` from its initial commit, MUST name `ISSUES.md` in its task prompt as
the place to file a follow-up issue for a pre-existing regression, and its step 8 MUST score a trial as having filed the issue when it
edits or writes `ISSUES.md` with a native tool after step 6, or appends to it with a bash redirect or `tee -a` anchored on the file,
and MUST NOT when step 6 never matched or the command only mentions the file.

Supports: [792a23i1a The bisect-before-fix fixture names a local issue-filing target](../product/memory-effectiveness-evaluation.md#792a23i1-uat-scenarios).

Rationale: without a local target a shim-following agent must stop and ask merely because the fixture omitted the means to comply, and the
runbook is not changed to work around a fixture gap.

#### 792b38c9a Scenarios

```gherkin
Scenario: 792b38c9a1 The prompt names the target
  Given the bisect-before-fix task prompt
  When it is read
  Then it names ISSUES.md as the place to file a follow-up issue

Scenario: 792b38c9a2 The fixture holds the file from the initial commit
  Given a trial repository set up from the fixture template
  When its initial commit is read
  Then it contains ISSUES.md

Scenario: 792b38c9a3 Filing through a native edit scores
  Given a trial that edits ISSUES.md with a native tool after step 6
  When step 8 is evaluated
  Then it scores as followed

Scenario: 792b38c9a4 Filing through a bash append or tee scores
  Given a trial that appends to ISSUES.md with a redirect, or with tee -a, after step 6
  When step 8 is evaluated
  Then it scores as followed

Scenario: 792b38c9a5 Step 8 is false when step 6 never matched
  Given a trial that files an issue and never fixed bar.py
  When step 8 is evaluated
  Then it does not score as followed

Scenario: 792b38c9a6 An unanchored append or a mention does not score
  Given a command that only mentions ISSUES.md or appends elsewhere
  When step 8 is evaluated
  Then it does not score as followed
```

### 792b38c10 The rank of a surfaced note

Requirement: [792b38c10 The rank of a surfaced note](#792b38c10-the-rank-of-a-surfaced-note) The eval-trial-harness MUST, to decide whether a target note surfaced in an `engram query` payload, look for the target
among the payload's top-level `items[]` by path or basename and report `surfaced` true with its one-based rank there, and `surfaced`
false when it is absent, and MUST NOT consult any cluster's `candidate_l2s`. A probe over a set of runbooks MUST do this for each
and report a batch outcome per runbook, a miss included, and MUST raise `RuntimeError` naming the runbook, the exit code and the error output when the query subprocess exits non-zero. A runbook that missed MAY then be escalated, one query per phrase.

Supports: [792a23j1a A target present in items() counts as surfaced](../product/memory-effectiveness-evaluation.md#792a23j1-uat-scenarios), [792a23j1b candidate_l2s absence does not count as a miss](../product/memory-effectiveness-evaluation.md#792a23j1-uat-scenarios).

Rationale: `candidate_l2s` is a capped subset chosen for synthesis ([792b24c10 Candidate notes](./query-engine.md#792b24c10-candidate-notes)), so reading it reports a correctly matched note as a miss.

#### 792b38c10a Scenarios

```gherkin
Scenario: 792b38c10a1 A target in items[] surfaces at its rank
  Given a payload whose items[] holds the target at rank 2 and whose candidate_l2s does not
  When the rank is read
  Then surfaced is true and the rank is 2

Scenario: 792b38c10a2 The old candidate_l2s logic would have missed it
  Given the same payload
  When the old candidate_l2s check is applied as a control
  Then it reports a miss, which shows the fixture discriminates old from new

Scenario: 792b38c10a3 A batched runbook probe reports success, miss and failure
  Given runbook probes whose query finds the runbook, does not, or whose subprocess fails
  When each is probed in a batch
  Then the first reports surfaced at its rank, the second reports surfaced false with no rank, and the third raises RuntimeError carrying the exit code and the error output

Scenario: 792b38c10a4 Escalation is per phrase and the corpus loads
  Given a runbook that a first phrase misses and a later phrase finds, and the runbook corpus
  When the probe runs
  Then it escalates per phrase and loads every runbook of the corpus
```

### 792b38c11 Every spawn site builds its environment through the primitive

Requirement: [792b38c11 Every spawn site builds its environment through the primitive](#792b38c11-every-spawn-site-builds-its-environment-through-the-primitive) Each of the seven tier-A sites of the eval-trial-harness that spawn a trial MUST build its environment with the isolated-environment
primitive of [792b38c3 The isolated environment of a trial](#792b38c3-the-isolated-environment-of-a-trial) and import it, and MUST NOT hand-roll `env = dict(os.environ)`. A tier-B Python site that sets a vault MUST also set the chunk directory, by building its direct-call environment with `engram_env`, and a tier-B shell site that sets a vault MUST set the chunk directory too. The Claude harness's own environment building MUST delegate to the isolation module. The older experiment harnesses (`dev/eval/traps/c4.py`, `c5.py`, `c6.py`, `cake.py`, `compound_eval.py`, `wrun.py` and their kin) build their environment by hand and set the vault and the chunk directory together; they are outside the two site lists.

Supports: [792a23d1b A harness captures no trial evidence itself and still gets a correct guard](../product/memory-effectiveness-evaluation.md#792a23d1-uat-scenarios).

Rationale: a site that sets only `CLAUDE_CONFIG_DIR` wrote seven real notes into the maintainer's vault during #687's measurement,
and a site that sets the vault alone leaves the chunk index at the maintainer's global one (#642). The sites share no entry point
to instrument, so the assertion is on the source text: a weak test of behavior and a strong one of the property that regressed.

#### 792b38c11a Scenarios

```gherkin
Scenario: 792b38c11a1 Tier A sites import the isolation module
  Given the seven tier-A spawn sites
  When their sources are read
  Then each imports isolation

Scenario: 792b38c11a2 Tier A sites call isolated_env
  Given the seven tier-A spawn sites
  When their sources are read
  Then each calls isolated_env

Scenario: 792b38c11a3 No tier A site hand-rolls its environment
  Given the seven tier-A spawn sites
  When their sources are read
  Then none holds env = dict(os.environ)

Scenario: 792b38c11a4 No Python site sets a vault without a chunk directory
  Given the tier-B Python sites
  When their sources are read
  Then none sets ENGRAM_VAULT_PATH without ENGRAM_CHUNKS_DIR

Scenario: 792b38c11a5 Tier B Python sites use engram_env
  Given the tier-B Python sites
  When their sources are read
  Then each builds its direct-call environment with engram_env

Scenario: 792b38c11a6 No shell site sets a vault without a chunk directory
  Given the tier-B shell sites
  When their sources are read
  Then none sets a vault without a chunk directory

Scenario: 792b38c11a7 The Claude harness delegates to the isolation module
  Given the cumulative harness's environment building
  When its source is read
  Then it delegates to the isolation module
```
