# Unified `/learn` Skill — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace four existing memory skills (`~/.claude/skills/capturing-fleeting-notes`, `~/.claude/skills/promoting-to-permanent-notes`, `skills/learn`, `skills/remember`) with one unified skill named `learn` that applies the Recurs + Activity-and-Domain + Knowledge gates at write time. Vault-only backend. Removes the orphaned `engram learn` subcommand and `--delete-fleeting` flag.

**Architecture:** Single skill at `skills/learn/SKILL.md` (symlinked to `~/.claude/skills/learn` for global activation). Writes vault notes only, via existing `engram promote {feedback|fact|moc}`. No `Fleeting/` directory; no fleeting tier. Two trigger modes (user-invoked, autonomous at task boundaries) sharing one gate sequence and write loop. Binary cleanup removes only clearly-orphan code: the `engram learn` subcommand wiring + `--delete-fleeting` flag plumbing. Other SBIA-backed subcommands (`recall`, `show`, `update`, `list`, `cycle`, `quick`, `reminder`) are out of scope.

**Tech Stack:** Go (1.x), `targ` build system, imptest+rapid+gomega test stack, Claude Code skill format (`SKILL.md` with YAML frontmatter), agent-memory vault (Obsidian-format markdown), `engram` CLI binary.

**Spec:** `docs/superpowers/specs/2026-05-10-unified-learn-skill-design.md`

---

## File Structure

**Created:**
- `skills/learn/SKILL.md` — full rewrite (replaces the existing in-repo learn skill)
- `skills/learn/tests/baseline-project-specific.md` — pressure-test scenario
- `skills/learn/tests/baseline-hindsight-framing.md` — pressure-test scenario
- `skills/learn/tests/baseline-information-not-knowledge.md` — pressure-test scenario
- `skills/learn/tests/baseline-clean-write.md` — pressure-test scenario (passes all gates)
- `skills/learn/tests/baseline-autonomous-trigger.md` — pressure-test scenario

**Modified:**
- `internal/cli/targets.go` — remove `targ.Group("learn", ...)` block; remove `CommonLearnArgs`, `LearnFactArgs`, `LearnFeedbackArgs` structs; remove `DeleteFleeting` field from `CommonPromoteArgs`
- `internal/cli/promote.go` — remove `DeleteFleeting` from `PromoteArgs`, `PromoteDeps`, the plumbing in `runPromote`, all three `runPromote*` callers
- `internal/cli/cli.go` — remove `osPromoteFS.DeleteFleeting` method and the `DeleteFleeting` wiring in `defaultPromoteDeps`
- `internal/cli/promote_test.go` — remove all `DeleteFleeting` field uses; delete `TestRunPromote_PropagatesDeleteFleetingError`
- `internal/cli/adapters_test.go` — delete `TestOsPromoteFS_DeleteFleeting_RemovesFile`
- `architecture/c4/c4-learn-skill.md` and `.json` — update to reflect unified vault-only skill (no SBIA writes)
- `architecture/c4/c3-skills.md` and `.json` — drop the `remember` skill node if present; consolidate `learn` to unified shape

**Deleted:**
- `skills/remember/SKILL.md` and the `skills/remember/` directory
- `internal/cli/learn.go`
- `internal/cli/learn_test.go`

**Filesystem ops (outside repo, no git):**
- `~/.claude/skills/capturing-fleeting-notes/` — delete
- `~/.claude/skills/promoting-to-permanent-notes/` — delete
- `~/.claude/skills/learn` — replace with symlink → `<repo>/skills/learn`

---

## Phase 1 — Author the new skill (TDD via writing-skills)

### Task 1: Write the five pressure-test scenarios

**Files:**
- Create: `skills/learn/tests/baseline-project-specific.md`
- Create: `skills/learn/tests/baseline-hindsight-framing.md`
- Create: `skills/learn/tests/baseline-information-not-knowledge.md`
- Create: `skills/learn/tests/baseline-clean-write.md`
- Create: `skills/learn/tests/baseline-autonomous-trigger.md`

- [ ] **Step 1: Create `skills/learn/tests/baseline-project-specific.md`**

```markdown
# Baseline pressure test — project-specific candidate (should fail Recurs)

## Scenario
The user says: "remember that the engram promote binary required us to extract writePromoteUnderLock when the cyclomatic complexity check fired on Task 8."

## Expected new-skill behavior
- Identify one candidate.
- Gate 1 (Recurs): FAIL. Situation names "engram promote", "writePromoteUnderLock", "Task 8" — project-specific.
- Drop the candidate. No `engram promote` call.
- Report names the gate failure with a one-line reason.

## Expected current-skill behavior (RED baseline)
The current `capturing-fleeting-notes` would write a fleeting; current `promoting-to-permanent-notes` would convert it to a permanent. Either path writes; neither rejects.
```

- [ ] **Step 2: Create `skills/learn/tests/baseline-hindsight-framing.md`**

```markdown
# Baseline pressure test — hindsight-baked framing (should fail Activity+Domain)

## Scenario
The user says: "remember: when fixing context cancellation in concurrent code, always pass the parent context through to spawned goroutines."

## Expected new-skill behavior
- Identify one candidate.
- Gate 1 (Recurs): PASS — "concurrent Go code" is activity+domain.
- Gate 2 (Activity+Domain): FAIL — situation says "when fixing X", which bakes in hindsight. An agent embarking on concurrent-Go work would not query "when fixing context cancellation".
- Drop OR reframe and re-run. The skill must show the reframe attempt: "When writing concurrent Go code with context" → re-run gates.
- If reframed candidate passes Knowledge gate (it does: "pass parent context to spawned goroutines" is a transferable principle), write it.
- Report includes the reframe note.

## Expected current-skill behavior (RED baseline)
Current skills write the situation as-given, hindsight baked in.
```

- [ ] **Step 3: Create `skills/learn/tests/baseline-information-not-knowledge.md`**

```markdown
# Baseline pressure test — information not knowledge (should fail Knowledge)

## Scenario
The user says: "remember that we noticed the targ tool prints warnings in yellow."

## Expected new-skill behavior
- Identify one candidate.
- Gate 1 (Recurs): FAIL — names "targ" (project tool). Drop here.

## Variation: replace "targ" with "many CLI build tools" so Recurs passes
- Gate 1 (Recurs): PASS.
- Gate 2 (Activity+Domain): borderline; the situation is "using CLI build tools" — fine.
- Gate 3 (Knowledge): FAIL. "Tool prints warnings in yellow" is information, not a transferable principle. No action; no applicability beyond observation.
- Drop the candidate.

## Expected current-skill behavior (RED baseline)
Current skills capture this as a fleeting fact and likely promote.
```

- [ ] **Step 4: Create `skills/learn/tests/baseline-clean-write.md`**

```markdown
# Baseline pressure test — passes all three gates (should write)

## Scenario
After a long debugging session, the user says: "let's remember: when an LSP error appears after a commit, the commit may have already passed the build tool's checks — re-run the build tool before chasing the LSP error, since LSP often lags the post-commit state."

## Expected new-skill behavior
- Identify one candidate (feedback type — behavioral lesson).
- Gate 1 (Recurs): PASS — "developing in an IDE with LSP after committing" is activity+domain; no project naming.
- Gate 2 (Activity+Domain): PASS — situation phrased as agent would query before lesson known.
- Gate 3 (Knowledge): PASS — transferable principle with concrete action.
- Decide Luhmann position. Most-related existing note: `10c1.2026-05-10.never-chase-lsp-post-commit.md` (already in vault). This candidate merges or new-elaborates.
- If merge: fold into existing. If new-elaboration: write as continuation.
- Call `engram promote feedback` with full args; body on stdin includes `Related to:` bullet with rationale.
- Report: 1 candidate, 1 pass, 1 written.

## Expected current-skill behavior (RED baseline)
Capture writes a fleeting; promote later writes a permanent — two stages, ~doubled latency.
```

- [ ] **Step 5: Create `skills/learn/tests/baseline-autonomous-trigger.md`**

```markdown
# Baseline pressure test — autonomous trigger at task boundary

## Scenario
A coding agent has just finished implementing Phase 3 of a plan, all tests green, just committed. No user prompt. The skill self-fires.

The session contained:
1. A user correction: "don't compute Luhmann IDs yourself — pass --target and --relation."
2. A discovered fact: "the build tool exits 0 even when sub-targets warn; check stderr for the actual signal."
3. A trivial fix: a typo in a comment.

## Expected new-skill behavior
- Trigger fires because Phase 3 completion is a non-trivial chunk.
- Identify three candidates from session scan.
- Run gates:
  - #1: PASS all three gates (activity = "using a binary that manages IDs"; behavior + action concrete). Write.
  - #2: PASS all three gates (activity = "interpreting CLI tool exit codes"). Write.
  - #3: FAIL Knowledge gate (typo fix isn't a transferable principle). Drop.
- Two writes in one parallel tool-use block; one drop.
- No user prompt at any point.
- Report.

## Expected current-skill behavior (RED baseline)
No autonomous trigger exists in current skills.
```

- [ ] **Step 6: Commit**

```bash
git add skills/learn/tests/
git -c commit.gpgsign=false commit -m "$(cat <<'EOF'
test(learn): baseline pressure-test scenarios for unified skill

Five scenarios covering each gate failure mode, a clean write, and
the autonomous trigger path. Used as RED baseline before authoring
the new skill, then re-run as GREEN verification.

AI-Used: [claude]
EOF
)"
```

---

### Task 2: Run baseline tests against current skills (confirm RED)

**Files:**
- Read: `~/.claude/skills/capturing-fleeting-notes/SKILL.md`
- Read: `~/.claude/skills/promoting-to-permanent-notes/SKILL.md`
- Read: `skills/learn/SKILL.md` (current in-repo)
- Read: `skills/remember/SKILL.md`
- Read: each scenario file from Task 1

- [ ] **Step 1: For each of the five scenarios, dispatch a fresh general-purpose subagent**

For each scenario file, dispatch a fresh subagent with this prompt template:

```
You are operating under the following skill (read the contents of all four current skill files and treat them as your operating guide):

- `~/.claude/skills/capturing-fleeting-notes/SKILL.md`
- `~/.claude/skills/promoting-to-permanent-notes/SKILL.md`
- `<repo>/skills/learn/SKILL.md`
- `<repo>/skills/remember/SKILL.md`

The user has issued this scenario:

<paste scenario "Scenario" section verbatim>

Describe exactly what you would do: which skill applies, what notes/files you would write, what subcommands you would call. Do NOT execute the writes. Output the plan only.
```

- [ ] **Step 2: Record each subagent's decision verbatim**

Capture each response into `skills/learn/tests/baseline-RED-results.md`:

```markdown
# Baseline RED results — current skills' behavior

## baseline-project-specific
<subagent response>

## baseline-hindsight-framing
<subagent response>

## baseline-information-not-knowledge
<subagent response>

## baseline-clean-write
<subagent response>

## baseline-autonomous-trigger
<subagent response>
```

- [ ] **Step 3: Verify the RED matches the spec's expected-current-skill-behavior**

For each scenario, check that the subagent did exactly what the scenario's "Expected current-skill behavior (RED baseline)" section predicted. If not, the test framing is wrong — fix the scenario file before proceeding.

- [ ] **Step 4: Commit RED results**

```bash
git add skills/learn/tests/baseline-RED-results.md
git -c commit.gpgsign=false commit -m "$(cat <<'EOF'
test(learn): record RED baseline against current four skills

Captures subagent decisions for each pressure-test scenario,
confirming the current skills fail in the expected ways (no Recurs
gate, hindsight-baked situations preserved, information promoted as
if it were knowledge, no autonomous trigger).

AI-Used: [claude]
EOF
)"
```

---

### Task 3: Write the new `skills/learn/SKILL.md` (full content)

**Files:**
- Modify: `skills/learn/SKILL.md` (overwrite — full replacement)

- [ ] **Step 1: Replace the entire file with the following content**

```markdown
---
name: learn
description: Use when the user says "remember this", "save that for later", "/learn", "write up what we just did", or after a discrete task completes (feature shipped, bug fixed, plan step closed, direction changed). Captures lessons from the current session as permanent agent-memory vault notes that pass the Recurs + Activity-and-Domain + Knowledge gates.
---

# Learn

## Overview

Capture lessons from this session into the agent-memory vault as **permanent notes** (and **MOCs** when a real framing paragraph emerges across notes). One stage — no fleeting tier, no escape hatch. A candidate either passes all three gates and is written, or fails and is dropped.

This vault is your (the LLM's) persistent memory. You write everything; the human curates by directing what gets worked on. **Don't draft and ask for review** — you decide what becomes permanent and write it.

Style reference: https://obsidian.rocks/getting-started-with-zettelkasten-in-obsidian/. Source method: https://zettelkasten.de/introduction/.

## Vault paths

- Vault root: `/Users/joe/repos/personal/agent-memory/`
- Permanents: `<vault>/Permanent/`
- MOCs: `<vault>/MOCs/`

No `Fleeting/` directory. No `Main Index.md`. No log file. Chronology lives in filenames; navigation lives in MOCs and link context.

## Trigger modes

- **User-invoked** — `/learn`, "remember this", "save that for later", "write up what we just did". Input grain is determined from context: single observation when the user flags a specific moment; session-batch sweep when invoked at the end of a chunk of work.
- **Autonomous at task boundaries** — after a discrete task completes (feature shipped, bug fixed, plan step closed, direction changed), the skill self-fires to sweep the just-completed work using the same gate sequence and write discipline. No user prompt before write.

**Do not auto-fire on micro-tasks** (one-line edits, single-file moves, trivial renames, typo fixes). The threshold is "a chunk of work that *could plausibly* produce lessons" — not "anything ended." When unsure, do not fire.

## The three gates

For each candidate, run gates in order. **A single failure drops the candidate.** No retries; no escape hatches. (You may reframe the situation once and re-run gates — see Gate 2.)

### Gate 1 — Recurs

Strip the situation to **activity + domain**. If it names:

- this project (engram / traced / etc.), its internals, or its architecture
- phase numbers, issue IDs, commit hashes, dates
- one-time events ("user said X today"), diary entries, status snapshots

…the candidate fails Recurs. An agent working on an unrelated project (web app, game, data pipeline) should plausibly hit the same situation.

### Gate 2 — Activity-and-domain framing

The `situation` field describes what an agent would be embarking on, framed as it would be queried **before** the lesson is known. No hindsight; no diagnosis-as-situation.

| Bad (bakes in hindsight) | Good (activity + domain) |
|---|---|
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |
| "When debugging the failing test" | "When writing tests that interact with the filesystem" |

If the candidate fails this gate, you may reframe the situation **once** and re-run all three gates. If still failing, drop.

### Gate 3 — Knowledge bar

From zettelkasten.de: *"Information is dead and contextless; knowledge adds relevance and context. Translate information into knowledge by enriching it with applicability."* A candidate that merely describes what happened is information; it converts only when restateable as a principle with applicability beyond the originating event.

No word counts. No graduation rates. No "useful 2 years out." Just: can this be stated as a transferable principle?

## Workflow

### 1. Identify candidates

Scan the in-context conversation (default) or session logs (when source isn't loaded) for:

- **User corrections** — the user told you to do something differently
- **Failed approaches** — something was tried and didn't work
- **Discovered facts** — new knowledge about tools, idioms, conventions, gotchas
- **Recurring patterns** — behaviors that should be codified

### 2. Apply the three gates

For each candidate, run **Recurs → Activity-and-Domain → Knowledge** in order. Fail at any step → drop. Single-failure reasons are useful in the final report.

### 3. Decide disposition per survivor

- **New permanent** — one candidate → one new permanent
- **Merge** — sharpens an existing permanent's wording or adds an example without new claims; fold into that note
- **Split** — one candidate bundles multiple principles → multiple permanents
- **New-elaboration** — if it adds claims the existing permanent doesn't make, write a new permanent as a continuation (e.g. existing `1` → new `1a`)

**Merge vs. new-elaboration:** if the candidate adds claims the existing permanent doesn't make, prefer new-elaboration. Editing a published, dated permanent erases the time-shape of the thinking.

### 4. Decide Luhmann position per write

For each write, find the most-related existing note. Choose the relation:

- `continuation` — extends the related note's lineage (`1a` → `1a1`)
- `sibling` — parallel branch at the same level (`1a` → `1b`)
- `top` — brand new top-level thought (`5`, `6`, ...)

The binary computes the actual ID under a vault lock. **You do not compute the ID yourself.**

### 5. Draft body in LLM voice

**Feedback:**

```
engram promote feedback \
  --slug <kebab-case-tag> \
  --vault /Users/joe/repos/personal/agent-memory \
  --target <luhmann-id-of-related-note-or-empty> \
  --relation <top|continuation|sibling> \
  --source "session log <project>, <YYYY-MM-DD HH:MM UTC>, context: ..." \
  --situation "..." --behavior "..." --impact "..." --action "..."
```

Body content (`Related to:` bullets with per-link rationale) on stdin.

**Fact:**

```
engram promote fact \
  --slug <kebab-case-tag> \
  --vault /Users/joe/repos/personal/agent-memory \
  --target <id-or-empty> \
  --relation <top|continuation|sibling> \
  --source "..." \
  --situation "..." --subject "..." --predicate "..." --object "..."
```

Body (`Related to:` bullets) on stdin.

**MOC** (judgement-based, no count threshold):

```
engram promote moc \
  --slug <kebab-case-tag> \
  --vault /Users/joe/repos/personal/agent-memory \
  --target <id-or-empty> \
  --relation <top|continuation|sibling> \
  --source "constructed from cluster analysis, <YYYY-MM-DD>" \
  --topic "<theme name>"
```

Body (the framing paragraph(s) — no constituent list) on stdin.

### 6. Contradictions

If a new permanent contradicts an existing one, write the new permanent with a `Related to:` bullet whose rationale names the discrepancy. Surface in the final report. Don't smooth.

### 7. Write — one parallel tool-use block

**Hard rule: all `engram promote` invocations for a single /learn pass go in a single parallel tool-use block.** Serial writes cost a tool roundtrip each (~15–20s); batching collapses that.

### 8. Report

Per pass:
- Candidates considered
- Gates passed / failed (with gate name and one-line reason)
- Permanents written (with Luhmann IDs)
- MOCs written or updated
- Contradictions surfaced

## Quality bars

- **Atomicity** — one idea per permanent.
- **Autonomy** — permanents are understandable without context. Strip "this case", "the incident", "we did X" framing.
- **Knowledge, not information** — the principle has applicability beyond the originating event.
- **LLM voice** — translate raw material into your own synthesis. Verbatim user quotes get rephrased on writing.
- **Per-link rationale** — every `Related to:` bullet explains why the connection exists. No bare wikilinks.
- **Heterarchy** — a permanent can belong to multiple MOCs; one `Related to:` bullet per MOC with its own rationale.
- **Surface contradictions** — link them with rationale naming the discrepancy.

## Common mistakes

| Mistake | Fix |
|---|---|
| Writing a note whose situation names "engram", "Task 8", "promote.go" | Fail at Recurs gate; drop |
| Hindsight-baked situation ("When fixing the bug in X") | Fail at Activity+Domain gate; reframe to pre-lesson query phrasing |
| Writing "we observed X" without stating it as a principle | Fail at Knowledge gate; either restate as principle or drop |
| Drafting and asking for human voice rewrite | You're the writer. Just write. |
| Writing files directly with the filesystem | Use `engram promote {feedback|fact|moc}` — handles ID assignment under lock |
| Computing the Luhmann ID yourself | Pass `--target` and `--relation`; binary computes the ID |
| Auto-listing MOC constituents in body | Backlinks already do this — MOC body is framing prose only |
| Bare wikilinks without rationale | Every `Related to:` bullet must include per-link rationale |
| Serial `engram promote` calls across tool turns | One message, N parallel tool calls |
| Auto-firing on a one-line micro-task | Only autonomous-trigger on chunks that plausibly produce lessons; when unsure, don't fire |
| Creating a MOC because the cluster crossed a count threshold | Judgement, not count — a real framing paragraph must emerge |
| Putting an H1 title or `Luhmann-ID · date` line in the body | Filename is the display name; `luhmann` and `created` live in frontmatter |
| Smoothing over contradictions | Write `Related to:` bullets that name the discrepancy |
```

- [ ] **Step 2: Verify the file matches the spec**

Skim each section against `docs/superpowers/specs/2026-05-10-unified-learn-skill-design.md`. Confirm:
- Both trigger modes documented
- All three gates present in correct order
- All four disposition types (new / merge / split / new-elaboration)
- Vault-only backend (no SBIA references)
- No `Fleeting/` or `--delete-fleeting` references
- MOC creation is judgement-based with no threshold

- [ ] **Step 3: Do NOT commit yet — Task 4 runs GREEN verification first**

---

### Task 4: Re-run pressure tests against new skill (confirm GREEN)

**Files:**
- Read: `skills/learn/SKILL.md` (the new content from Task 3)
- Read: each scenario file
- Create: `skills/learn/tests/baseline-GREEN-results.md`

- [ ] **Step 1: For each scenario, dispatch a fresh general-purpose subagent**

Prompt template:

```
You are operating under this skill — read it and treat as your operating guide:

<paste full content of skills/learn/SKILL.md>

The user has issued this scenario:

<paste scenario "Scenario" section verbatim>

Describe exactly what you would do: which gates pass/fail, what subcommands you would call with what args, what you would report. Do NOT execute the writes. Output the plan only.
```

- [ ] **Step 2: Record GREEN results**

Append each subagent response to `skills/learn/tests/baseline-GREEN-results.md`:

```markdown
# Baseline GREEN results — new unified skill

## baseline-project-specific
<subagent response>

## baseline-hindsight-framing
<subagent response>

## baseline-information-not-knowledge
<subagent response>

## baseline-clean-write
<subagent response>

## baseline-autonomous-trigger
<subagent response>
```

- [ ] **Step 3: Verify each GREEN result matches the scenario's "Expected new-skill behavior"**

For each scenario, check the subagent did exactly what the scenario predicted. If not, edit `skills/learn/SKILL.md` to fix the gap, then re-run that scenario. Loop until all five pass.

- [ ] **Step 4: Commit skill + GREEN results together**

```bash
git add skills/learn/SKILL.md skills/learn/tests/baseline-GREEN-results.md
git -c commit.gpgsign=false commit -m "$(cat <<'EOF'
feat(learn): unified /learn skill with three quality gates

Replaces the two-stage capture+promote flow with a single skill
that applies Recurs + Activity-and-Domain + Knowledge gates at
write time. Vault-only backend via engram promote. Includes
autonomous trigger at task boundaries.

Pressure-tested against five baseline scenarios; behavior matches
spec.

AI-Used: [claude]
EOF
)"
```

---

## Phase 2 — Migration (filesystem + global symlink)

### Task 5: Delete in-repo `skills/remember/`

**Files:**
- Delete: `skills/remember/SKILL.md` and the `skills/remember/` directory

- [ ] **Step 1: Confirm no other repo code references `skills/remember`**

```bash
grep -rn "skills/remember\|skills\\\\remember" --include="*.md" --include="*.go" --include="*.json" \
  --exclude-dir=archive --exclude-dir=docs/superpowers/plans \
  2>/dev/null
```

Expected: zero matches outside of historical plan docs and the design spec.

- [ ] **Step 2: Delete the directory**

```bash
git rm -r skills/remember/
```

- [ ] **Step 3: Verify deletion**

```bash
ls skills/remember/ 2>&1
```

Expected: `ls: skills/remember/: No such file or directory`

- [ ] **Step 4: Commit**

```bash
git -c commit.gpgsign=false commit -m "$(cat <<'EOF'
chore(skills): remove obsolete /remember skill

Subsumed by the new unified /learn skill.

AI-Used: [claude]
EOF
)"
```

---

### Task 6: Replace global symlinks

This task involves filesystem operations outside the repo. No git commit.

**Files (outside repo):**
- Delete: `~/.claude/skills/capturing-fleeting-notes/`
- Delete: `~/.claude/skills/promoting-to-permanent-notes/`
- Delete (or replace): `~/.claude/skills/learn` (currently a directory or symlink — verify before removing)
- Delete: `~/.claude/skills/remember` (if exists)
- Create: `~/.claude/skills/learn -> /Users/joe/repos/personal/engram-worktrees/opencode-plugin/skills/learn`

- [ ] **Step 1: Inventory current global state**

```bash
ls -la ~/.claude/skills/ | grep -E "(learn|remember|capturing-fleeting|promoting-to-perm)"
```

Record what's currently there (directory vs symlink, target if symlink).

- [ ] **Step 2: Delete the two obsolete global skills**

```bash
rm -rf ~/.claude/skills/capturing-fleeting-notes
rm -rf ~/.claude/skills/promoting-to-permanent-notes
```

- [ ] **Step 3: Remove any global `learn` and `remember` directories/symlinks**

```bash
[ -e ~/.claude/skills/learn ] && rm -rf ~/.claude/skills/learn
[ -e ~/.claude/skills/remember ] && rm -rf ~/.claude/skills/remember
```

- [ ] **Step 4: Create the new symlink**

```bash
ln -s /Users/joe/repos/personal/engram-worktrees/opencode-plugin/skills/learn \
      ~/.claude/skills/learn
```

- [ ] **Step 5: Verify the symlink resolves correctly**

```bash
ls -la ~/.claude/skills/learn
readlink ~/.claude/skills/learn
test -f ~/.claude/skills/learn/SKILL.md && echo "SKILL.md visible via symlink" || echo "FAIL: SKILL.md not visible"
```

Expected: symlink points to the repo path; `SKILL.md visible via symlink`.

- [ ] **Step 6: Verify the four obsolete skills are gone**

```bash
for skill in capturing-fleeting-notes promoting-to-permanent-notes remember; do
  [ -e ~/.claude/skills/$skill ] && echo "FAIL: $skill still present" || echo "OK: $skill removed"
done
```

Expected: all three show `OK`.

- [ ] **Step 7: No commit (filesystem-only changes)**

Note in the next commit's description that global symlinks were rewired.

---

### Task 7: Update references to old skill names

**Files (modify if matched):**
- `README.md`
- `architecture/c4/c4-learn-skill.md`
- `architecture/c4/c4-learn-skill.json`
- `architecture/c4/c3-skills.md`
- `architecture/c4/c3-skills.json`
- `docs/opencode-plugin-plan.md`
- Any other non-archive `.md` file referencing the four old skills

- [ ] **Step 1: Find all references in non-archive paths**

```bash
grep -rln "capturing-fleeting-notes\|promoting-to-permanent-notes\|skills/remember" \
  --include="*.md" --include="*.json" \
  --exclude-dir=archive --exclude-dir=docs/superpowers/plans \
  --exclude-dir=docs/superpowers/specs \
  2>/dev/null
```

(Historical plans and specs are read-only history; don't rewrite them.)

- [ ] **Step 2: For each matched file, read it and decide whether the reference is**

- **Stale** (e.g., README listing the four skills): update to mention the unified `learn` skill.
- **Architectural** (C4 diagrams): update node names/edges to match the new shape — single `learn` skill that writes vault notes only, no SBIA backend edges.
- **Historical** (archived plan/spec): leave alone.

- [ ] **Step 3: Edit the C4 diagrams**

For `architecture/c4/c4-learn-skill.md` and `.json`:
- Update the skill node to reflect the unified-skill scope (Recurs/Framing/Knowledge gates; vault-only backend).
- Remove edges to SBIA/`engram learn` if present.
- Add edges to `engram promote` (feedback/fact/moc subcommands).

For `architecture/c4/c3-skills.md` and `.json`:
- Remove any `remember` skill node and incident edges.
- Confirm `learn` node still exists; update its description if it referenced "feedback + fact via engram learn".

Use `mcp__pencil__*` tools only if the C4 files are `.pen` encrypted; otherwise standard Read/Edit applies. (Spot check: if a quick `head -1 <file>` shows readable markdown/JSON, treat as plaintext.)

- [ ] **Step 4: Update README.md if needed**

If `README.md` lists the four skills, replace the list with the unified `learn` (alongside `recall`, `prepare`, `migrate`, `c4`).

- [ ] **Step 5: Re-grep to confirm zero remaining references in non-archive paths**

```bash
grep -rln "capturing-fleeting-notes\|promoting-to-permanent-notes\|skills/remember" \
  --include="*.md" --include="*.json" \
  --exclude-dir=archive --exclude-dir=docs/superpowers/plans \
  --exclude-dir=docs/superpowers/specs \
  2>/dev/null
```

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add -A
git status --short
git -c commit.gpgsign=false commit -m "$(cat <<'EOF'
chore(docs,c4): update references for unified /learn skill

Removes references to obsolete capturing-fleeting-notes,
promoting-to-permanent-notes, and /remember skills across README,
C4 diagrams, and other docs. Global skill symlinks have been
rewired (filesystem-only; not tracked in git).

AI-Used: [claude]
EOF
)"
```

---

## Phase 3 — Binary cleanup (remove `engram learn` + `--delete-fleeting`)

### Task 8: Remove the `learn` subcommand wiring from `targets.go`

**Files:**
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Read the current `targ.Group("learn", ...)` block to know exactly what to remove**

```bash
sed -n '160,175p' internal/cli/targets.go
```

You should see something like:
```go
targ.Group("learn",
    targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
        errHandler(runLearnFeedback(ctx, a, stdout))
    }).Name("feedback").Description("Learn from behavioral feedback"),
    targ.Targ(func(ctx context.Context, a LearnFactArgs) {
        errHandler(runLearnFact(ctx, a, stdout))
    }).Name("fact").Description("Learn a factual statement"),
),
```

- [ ] **Step 2: Remove the entire `targ.Group("learn", ...)` block** (Edit tool, exact match on the block)

- [ ] **Step 3: Remove the now-orphan structs**

In the same file, remove:
- `CommonLearnArgs` struct
- `LearnFactArgs` struct
- `LearnFeedbackArgs` struct

- [ ] **Step 4: Run build to confirm no other code in `targets.go` references the removed types**

```bash
targ check-full 2>&1 | head -50
```

Expected: failures only in `internal/cli/learn.go` and `internal/cli/learn_test.go` (which still reference `runLearnFeedback`, `runLearnFact`, `LearnFeedbackArgs`, `LearnFactArgs`). These get deleted in Task 9.

- [ ] **Step 5: Do not commit yet — Task 9 follows immediately**

---

### Task 9: Delete `internal/cli/learn.go` and `internal/cli/learn_test.go`

**Files:**
- Delete: `internal/cli/learn.go`
- Delete: `internal/cli/learn_test.go`

- [ ] **Step 1: Delete both files**

```bash
git rm internal/cli/learn.go internal/cli/learn_test.go
```

- [ ] **Step 2: Run `targ check-full` to confirm clean**

```bash
targ check-full 2>&1 | tail -30
```

Expected: all green. If errors remain, they're either:
- A still-living reference to removed code (find it, remove it, retry)
- Test that relied on the SBIA store being reachable through `engram learn` paths (unlikely; check carefully — these tests may be in `internal/memory/` and need to stay if they cover other consumers like `engram recall`)

- [ ] **Step 3: Commit**

```bash
git add -A
git -c commit.gpgsign=false commit -m "$(cat <<'EOF'
refactor(cli): remove orphaned engram learn subcommand

The /remember and old /learn skills (both removed) were the only
callers. Vault-only memory writes now flow through engram promote
via the unified /learn skill. The internal/memory SBIA store
remains for the engram recall and engram show subcommands, which
are still used by the prepare and migrate skills.

Removes:
- targ.Group("learn", ...) wiring in internal/cli/targets.go
- CommonLearnArgs, LearnFactArgs, LearnFeedbackArgs structs
- internal/cli/learn.go and learn_test.go

AI-Used: [claude]
EOF
)"
```

---

### Task 10: Remove `DeleteFleeting` from `CommonPromoteArgs` and `PromoteArgs`

**Files:**
- Modify: `internal/cli/targets.go`
- Modify: `internal/cli/promote.go`

- [ ] **Step 1: Read the current `CommonPromoteArgs` struct in `targets.go`**

```bash
grep -n "DeleteFleeting\|CommonPromoteArgs" internal/cli/targets.go
```

- [ ] **Step 2: Remove the `DeleteFleeting` field from `CommonPromoteArgs`** (Edit tool)

Remove the line:
```go
DeleteFleeting string `targ:"flag,name=delete-fleeting,desc=path to fleeting note to delete after success"`
```

- [ ] **Step 3: Read the `PromoteArgs` struct in `promote.go`**

```bash
sed -n '15,40p' internal/cli/promote.go
```

- [ ] **Step 4: Remove `DeleteFleeting string` from `PromoteArgs`** (Edit tool)

- [ ] **Step 5: Remove `DeleteFleeting func(path string) error` from `PromoteDeps`**

```bash
sed -n '40,55p' internal/cli/promote.go
```

Remove the `DeleteFleeting` field from the struct.

- [ ] **Step 6: Run targ check-full to see all sites referencing the removed fields**

```bash
targ check-full 2>&1 | tail -40
```

Expected: errors in `promote.go` (the plumbing in `runPromote` and the three `runPromote{Feedback,Fact,MOC}` callers), `cli.go` (the `osPromoteFS.DeleteFleeting` method and the wiring in `defaultPromoteDeps`), and `promote_test.go` + `adapters_test.go`.

- [ ] **Step 7: Do not commit yet — Tasks 11–14 follow**

---

### Task 11: Remove `DeleteFleeting` plumbing from `runPromote` in `promote.go`

**Files:**
- Modify: `internal/cli/promote.go`

- [ ] **Step 1: Find and remove the `if args.DeleteFleeting != ""` block**

```bash
sed -n '240,250p' internal/cli/promote.go
```

Should show:
```go
if args.DeleteFleeting != "" {
    delErr := deps.DeleteFleeting(args.DeleteFleeting)
    if delErr != nil {
        return fmt.Errorf("promote: deleting fleeting %s: %w", args.DeleteFleeting, delErr)
    }
}
```

Remove the entire block.

- [ ] **Step 2: Remove `DeleteFleeting: fs.DeleteFleeting` from the deps construction**

```bash
grep -n "DeleteFleeting: " internal/cli/promote.go
```

Remove each matching line in the `runPromote{Feedback,Fact,MOC}` deps-passthrough blocks (lines ~135, 264, 282, 300 per earlier grep). They look like:
```go
DeleteFleeting: fs.DeleteFleeting,
```
or
```go
DeleteFleeting: a.DeleteFleeting,
```

- [ ] **Step 3: Run targ check-full**

```bash
targ check-full 2>&1 | tail -30
```

Expected: errors remaining in `cli.go` (osPromoteFS method) and tests.

- [ ] **Step 4: Do not commit yet**

---

### Task 12: Remove `osPromoteFS.DeleteFleeting` in `cli.go`

**Files:**
- Modify: `internal/cli/cli.go`

- [ ] **Step 1: Find and remove the method**

```bash
sed -n '83,92p' internal/cli/cli.go
```

Should show something like:
```go
// DeleteFleeting removes the fleeting file at path.
func (*osPromoteFS) DeleteFleeting(path string) error {
    return os.Remove(path)
}
```

Remove the comment + method.

- [ ] **Step 2: Run targ check-full**

```bash
targ check-full 2>&1 | tail -30
```

Expected: errors remaining in `promote_test.go` and `adapters_test.go`.

- [ ] **Step 3: Do not commit yet**

---

### Task 13: Update `promote_test.go` — drop all `DeleteFleeting` references

**Files:**
- Modify: `internal/cli/promote_test.go`

- [ ] **Step 1: Find all references**

```bash
grep -n "DeleteFleeting\|delete-fleeting" internal/cli/promote_test.go
```

- [ ] **Step 2: Remove every `DeleteFleeting: ...` field assignment** in test fixtures (Edit tool, one per match)

- [ ] **Step 3: Delete the entire test `TestRunPromote_PropagatesDeleteFleetingError`**

This is the test specifically for the now-removed delete-fleeting behavior. Cut from function signature through its closing brace.

- [ ] **Step 4: Run the tests**

```bash
targ test 2>&1 | tail -20
```

Expected: all `internal/cli/promote_test.go` tests pass; `adapters_test.go` still failing on `TestOsPromoteFS_DeleteFleeting_RemovesFile`.

---

### Task 14: Delete `TestOsPromoteFS_DeleteFleeting_RemovesFile` in `adapters_test.go`

**Files:**
- Modify: `internal/cli/adapters_test.go`

- [ ] **Step 1: Find the test**

```bash
grep -n "TestOsPromoteFS_DeleteFleeting_RemovesFile" internal/cli/adapters_test.go
```

- [ ] **Step 2: Delete the entire test function** (from `func TestOsPromoteFS_DeleteFleeting_RemovesFile(t *testing.T) {` through its closing brace)

- [ ] **Step 3: Run full check**

```bash
targ check-full 2>&1 | tail -20
```

Expected: all green.

- [ ] **Step 4: Commit the entire `--delete-fleeting` removal as one logical change**

```bash
git add -A
git -c commit.gpgsign=false commit -m "$(cat <<'EOF'
refactor(cli): remove --delete-fleeting flag from engram promote

The new unified /learn skill writes vault notes directly without a
fleeting intermediate, so no fleeting paths exist to delete. Drops:
- DeleteFleeting field from CommonPromoteArgs and PromoteArgs
- DeleteFleeting func from PromoteDeps
- osPromoteFS.DeleteFleeting method
- The delete-on-success block in runPromote
- TestRunPromote_PropagatesDeleteFleetingError
- TestOsPromoteFS_DeleteFleeting_RemovesFile

AI-Used: [claude]
EOF
)"
```

---

## Phase 4 — Smoke test the end-to-end flow

### Task 15: Smoke-test the unified skill via a real invocation

**Files:**
- (no file changes; validation only)

- [ ] **Step 1: Confirm the binary builds clean**

```bash
targ check-full 2>&1 | tail -5
targ build 2>&1 | tail -5
```

Expected: both succeed.

- [ ] **Step 2: Confirm `engram learn` is gone**

```bash
engram learn feedback --help 2>&1 | head -5
```

Expected: error / "unknown subcommand learn".

- [ ] **Step 3: Confirm `engram promote` works without `--delete-fleeting`**

```bash
engram promote --help 2>&1 | grep -- "--delete-fleeting" && echo "FAIL: still present" || echo "OK: removed"
```

- [ ] **Step 4: Confirm the symlinked skill resolves**

```bash
test -f ~/.claude/skills/learn/SKILL.md && head -3 ~/.claude/skills/learn/SKILL.md
```

Expected: shows the frontmatter `---` / `name: learn` of the new skill.

- [ ] **Step 5: Dispatch a fresh general-purpose subagent with the new skill loaded, hand it `baseline-clean-write` scenario, and have it actually call `engram promote feedback` against a test vault (NOT the real `/Users/joe/repos/personal/agent-memory`)**

Set up a throwaway vault:
```bash
mkdir -p /tmp/learn-smoke/{Permanent,MOCs}
```

Subagent prompt:
```
You are operating under <repo>/skills/learn/SKILL.md (read it now).
Scenario: <paste baseline-clean-write scenario>
EXECUTE the writes against vault /tmp/learn-smoke (not the real vault).
Use --vault /tmp/learn-smoke for all engram promote calls.
```

- [ ] **Step 6: Verify the test vault contains a well-formed permanent**

```bash
ls /tmp/learn-smoke/Permanent/
cat /tmp/learn-smoke/Permanent/*.md
```

Expected: one `<Luhmann-ID>.YYYY-MM-DD.<slug>.md` file with frontmatter (`type: feedback`, `situation:`, `behavior:`, `impact:`, `action:`, `luhmann:`, `created:`, `source:`) and body line starting `Lesson learned:` plus `Related to:` bullets.

- [ ] **Step 7: Clean up the test vault**

```bash
rm -rf /tmp/learn-smoke
```

- [ ] **Step 8: No commit (smoke test only)**

If anything failed at this stage, return to the appropriate phase and iterate. Do not declare the work complete.

---

## Self-Review Notes (for the executing agent)

- **Spec coverage:** every section of the design spec maps to a task — gates (Tasks 1, 3, 4), trigger modes (Task 3), vault-only backend (Task 3), MOC judgement (Task 3), filename/Luhmann mechanics (Task 3), migration steps 1–5 of spec (Tasks 3, 5, 6, 7), binary cleanup (Tasks 8–14), testing (Tasks 1–4, 15).
- **Non-goals stay non-goals:** no vault-search-before-write, no source-attribution dichotomy, no Right-Home gate, no DUPLICATE/CONTRADICTION handling, no retroactive sweep of existing permanents, no removal of `engram update/list/cycle/quick/reminder` or `internal/memory/`.
- **TDD shape:** Phase 1 uses behavioral pressure-tests (RED → write → GREEN), the writing-skills idiom. Phase 3 cleanup is a sequence of remove-and-verify steps since removing-code isn't naturally testable via "write failing test first" — TDD here means *don't break existing tests*; commit only when `targ check-full` is green.
- **No subprocess test runs across the boundary:** Task 15 dispatches a subagent that writes to `/tmp/learn-smoke`, never the real vault. The real vault stays untouched by this plan.
