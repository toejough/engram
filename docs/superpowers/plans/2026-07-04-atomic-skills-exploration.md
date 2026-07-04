# Atomic skills redesign — investigation plan

> **For agentic workers:** research + census + options + smoke tests. No skill edits or deploys
> this round. Sandbox copies only; no production skill file touched. Deliverable is
> `docs/design/2026-07-04-atomic-skills-options.md` + PRESENT TO JOE AND STOP.

**Ask (Joe, 2026-07-04, condensed):** Rework the four engram skills (recall, learn, please, route)
around atoms + SRP so a change lives in one place. (Correction vs the ask's phrasing, per the
verified F2 census: QA capture = 2 copies + 1 pointer; learn-fact|feedback = 3 copies — the
inverse of the counts as spoken; F2 is the measurement of record; ingest sweeps appear in two skills.) Research good skill design broadly, produce
a few design options, smoke test the leading options, report findings. Behavior preservation is
the bar; readability/maintainability is the goal.

**Stop-point:** present options with smoke-test results + tradeoffs; Joe picks. No refactor ships
without his call.

---

## Settled constraints (recorded; no gate re-litigates; S1's atom names and N-skills constraint verified VERBATIM against ROADMAP:177–179 by the docs gate)

| # | Constraint | Source |
|---|---|---|
| S1 | Atoms = read-memory, write-memory, route-a-task, orchestrate-a-workflow (Joe's four-atom designations, ROADMAP:175–181 arc, 2026-06-29); design should not end with N skills that almost all do the same thing. | Joe, 2026-06-29 |
| S2 | Behavior preservation is the bar; readability/maintainability is the goal; metric improvements welcome but not the objective. | Joe, 2026-07-04 |
| S3 | This round = research + census + options + smoke tests + report. NO production skill edits. | Joe, 2026-07-04 |

---

## Measured priors the design must respect (each SYNTHESIZED from the cited vault note — the note is authoritative; executors re-read a note before leaning hard on its prior)

**P1 — Firing surface is the highest-risk axis.** Firing is decided by frontmatter DESCRIPTION
before the body loads (vault note 100 + description-drives-firing finding). Atoms-as-NEW-SKILLS
with active descriptions add firing decisions. Over-fire history on one rejected proposal: 147×
at one trigger point (vault note 139; a Gate-C ground-truth review corrected the original
note-144 citation); under-fire is the standing risk for engram. Atoms-as-shared-
reference-files or atoms-as-non-triggering-skills keep the firing surface unchanged or minimally
changed. The smoke tests must measure firing-surface delta explicitly.

**P2 — please's heaviness is measured value.** Note 100: anti-amnesia 8/8 → 0/8 needed the heavy
orchestration skill; mere presence failed 83%. Any recomposition that thins the orchestration
content of please risks a measured regression. please's Step 7 already contains one deliberate
deduplication: "The learn skill's Step 2.5 handles ad-hoc QA pair capture — do not duplicate
that logic here." Any option that further restructures please must preserve this behavior.

**P3 — Prior recall split heuristic (notes 78/80, transferable, do not over-cite as SRP verdict).**
The prior recall/recall-synthesis split was a MODEL-cost delegation — rolled back because 14% was
not worth two skills + dispatch overhead. Its transferable lesson: split at the judgment seam
(judgment stays with context; mechanical extracts go elsewhere). Do NOT cite notes 78/80 as a
general SRP verdict for skill decomposition — they measured a specific cost/capability tradeoff.

**P4 — Skills for capable models enforce DISCIPLINE, not teach (note 33).** Atoms must preserve
the discipline-enforcement framing (imperative prohibitions, red flags tables, rationalization
counters), not reduce to neutral how-to fragments. Any refactored text that softens an imperative
into guidance fails this standard.

**P5 — Name-the-action guidance form must survive recomposition verbatim (note 137).** The
action-naming pattern in red flags tables ("Sign you're off-script | What you should be doing")
and imperative procedure steps must be preserved exactly where they exist. Do not paraphrase.

**P6 — Research = broad sweep, multiple independent sources (note 157).** Named sources in the ask
are pointers into the landscape, never the boundary. The research stage sweeps all relevant
beats independently.

**P7 — Skill edits require writing-skills TDD with headless arms (notes 26/28).** This round
edits nothing in production. The smoke tests use SANDBOX COPIES only. Arms run headless
(`claude -p`) with ENGRAM_VAULT_PATH pointed at a throwaway vault.

**P8 — Headless arms must be sandboxed (note 160 + round1-build precedent).** Arms execute what
they are asked to describe; each arm gets its own throwaway ENGRAM_VAULT_PATH; no shared mutable
state across arms; no bypassPermissions.

---

## Facts (all labeled; all verified against working tree 2026-07-04)

### F1 — Skill census

| Skill | Lines (SKILL.md) | Source path |
|---|---|---|
| recall | 291 | `skills/recall/SKILL.md` |
| learn | 127 | `skills/learn/SKILL.md` |
| please | 114 | `skills/please/SKILL.md` |
| route | 77 | `skills/route/SKILL.md` |
| **Total** | **609** | |

**Section inventory — recall (291 lines):** Frontmatter (7 lines); Overview (14 lines);
Modes/depth-dial (21 lines); Step 0 upfront judgement (15 lines); Step 0.5 sweep / ingest
(7 lines); Step 1 phrase queries (17 lines); Step 2 engram query call (29 lines, includes channel
descriptions); Step 2.5 lazy note synthesis (46 lines, includes coverage table); Step 2.7
activation (17 lines); Step 3 closing synthesis (14 lines); Step 4 persist conclusion + QA
capture (43 lines); Red flags table (28 lines).

**Section inventory — learn (127 lines):** Frontmatter (9 lines); Intro / raw-event note (11
lines); Step 1 sweep / ingest (9 lines); Step 1.5 vocab liveness check (14 lines); Step 2
crystallize explicit lessons, fact+feedback invocations (37 lines); Step 2.5 ad-hoc QA capture
(24 lines); Red flags table (5 lines).

**Section inventory — please (114 lines):** Frontmatter (12 lines); Anti-sycophantic lean (14
lines); Adversarial review gates (29 lines); Required argument (3 lines); Task tracking (2
lines); Workflow steps 1-7 (23 lines); Stop conditions (9 lines); Red flags table (22 lines).

**Section inventory — route (77 lines):** Frontmatter (6 lines); Orchestration vs object-level
(7 lines); Rubric table (12 lines); Memory-discount paragraph (10 lines); Two non-waivable rules
(8 lines); Red flags table (9 lines).

### F2 — Duplication map (procedures appearing in ≥2 skill files, verified by reading)

| Procedure | Appears in | Exact duplicate? | Notes |
|---|---|---|---|
| `engram ingest --auto` block | recall Step 0.5; learn Step 1 | Near-identical (different surrounding prose) | Intentional: recall sweeps before querying; learn sweeps before crystallizing |
| `engram learn fact\|feedback` invocation + flag pattern (`--slug`, `--position`, `--source`, `--situation`, `--subject/predicate/object` or `--behavior/impact/action`) | recall Step 2.5C (the absent/near branch); recall Step 4; learn Step 2 | Near-identical flag surfaces; different surrounding prose and gate conditions | Three independent copies; a flag change requires three edits |
| `engram learn qa` invocation + flag pattern (`--slug`, `--question`, `--answer`, `--contributors`, `--certainty`, `--source`) | recall Step 4; learn Step 2.5 | Near-identical; learn adds a gate-dedup note ("if already written by recall, skip") | please Step 7 already defers to learn Step 2.5 — making it 2 copies, not 3 |
| `--supersedes "<basename>\|<type>\|<claim>"` guidance (types: updates/narrows/refutes, binary maintains inverse) | recall Step 2.5C; recall Step 4; learn Step 2 | Near-identical phrasing in all three | Appears as inline instruction at each write site |
| QA D2 bar rule ("≥1 wikilink in answer body required to capture") | recall Step 4; learn Step 2.5 | Same rule, different prose | please Step 7 defers to learn, so not a third copy |
| "Vocab tags are assigned automatically by the binary — do not hand-author them" | recall Step 2.5C; learn Step 2 | Verbatim | Two copies; one-line reminder |
| `engram amend` invocation + flag pattern (covered/near branches in Step 2.5C) | recall Step 2.5C **only** | Not duplicated | learn does not invoke amend; amend is recall's exclusive read-side crystallization |
| Activation guidance (`engram activate --note`) | recall Step 2.7 **only** | Not duplicated | No other skill handles activation |
| Vocab liveness check (`engram vocab stats`, refit flow, QA round-2 gate) | learn Step 1.5 **only** | Not duplicated | Unique to learn |

**Summary of cross-skill duplication by impact:**

- **Highest impact (3 copies, flag-level change = 3 edits):** `engram learn fact|feedback`
  invocation + `--supersedes` guidance — recall 2.5C, recall Step 4, learn Step 2.
- **Medium impact (2 copies, QA logic divergence risk):** `engram learn qa` invocation — recall
  Step 4, learn Step 2.5. (please already defers to learn, so this pair is the live risk.)
- **Low impact (2 copies, prose-level):** `engram ingest --auto` block — recall Step 0.5, learn
  Step 1. Near-identical but surrounding prose is different; behavioral drift risk is low.
- **Not duplicated:** activation, vocab liveness, amend procedure — these are skill-exclusive.

### F3 — Reference mechanics (what the runtime supports, verified)

1. **Within-skill `references/` subdirectory (on-demand load):** files placed in
   `skill-name/references/` are read by the agent only when the SKILL.md body directs it. This
   pattern is in use by `c4` and `property-rigor` skills. No engram skill currently uses it.
   Files load on-demand (zero context cost until accessed). Scope: ONE skill's own files only —
   the path is relative to the skill's directory.

2. **`REQUIRED SUB-SKILL: name` cross-references (named, not auto-loaded):** prose in a skill
   body names another skill as required (e.g., "REQUIRED SUB-SKILL: superpowers:test-driven-
   development"). This is text-only — the other skill's body is NOT loaded automatically;
   the agent invokes it explicitly. No @-import syntax is involved. This is the standard
   cross-skill pointer pattern per writing-skills/SKILL.md.

3. **`@` imports (force-load, NOT recommended):** `@`-syntax force-loads a file immediately,
   consuming context before it is needed. Writing-skills/anthropic-best-practices.md: "Avoid
   deeply nested references" and "Keep references one level deep from SKILL.md." The force-load
   cost is the reason @-imports are excluded from the standard pattern (writing-skills/SKILL.md
   line 288: "Why no @ links: `@` syntax force-loads files immediately, consuming 200k+ context
   before you need them").

4. **Cross-skill file sharing is not natively supported — but new skill DIRECTORIES deploy
   automatically (Gate A correction, code-verified):** the runtime offers no mechanism to share
   a single file between two skill directories via a relative path (`references/` resolves
   within the owning skill only). HOWEVER, a proper new top-level directory under `skills/`
   (e.g. `skills/write-memory/`) is deployed as a standard skill by `engram update`'s recursive
   walker (internal/update/update.go: planSkillCopies → listFilesRecursive → topLevelDir) with
   NO code change and no absolute paths — O-A's mechanism is the standard path, not a fragile
   one. The fragility concern applies only to files placed OUTSIDE the skills/ tree.

5. **All skill metadata (name + description) is pre-loaded at session start.** Only the SKILL.md
   body is on-demand. A new skill = new description in the system prompt = new firing surface
   (however small). A skill with a deliberately non-triggering description adds a small metadata
   payload without adding an autonomous firing decision.

---

## Stage S0 — Research (broad sweep, note 157 rule)

**Scope rule (note 157, verified):** named sources in this ask are pointers into the landscape,
never the boundary. S0 is a BROAD multi-source sweep across four independent beats. Each beat
returns: (A) findings that bear on the options catalog, and (B) vision-relevant findings that
are out-of-scope for this round (filed, not held).

**Beat 1 — Anthropic official guidance.** The canonical skill-authoring spec lives locally at
`~/.claude/plugins/cache/claude-plugins-official/superpowers/6.1.1/skills/writing-skills/`
(SKILL.md + anthropic-best-practices.md). Also fetch the live `https://agentskills.io/specification`
and the Claude Code skills docs at `https://platform.claude.com/docs/en/agents-and-tools/
agent-skills/overview`. Extract: what do the official docs say about cross-skill composition,
progressive disclosure, shared reference files, and the tradeoff between description-triggered
and body-read content?

**Beat 2 — Community and ecosystem patterns.** Survey at least two of: (a) public Claude Code
plugin repositories or skill collections (GitHub search: "claude skills SKILL.md"); (b)
LangChain / AutoGen / CrewAI prompt-module patterns for shared utilities; (c) discussions in
Anthropic's developer forum or Discord on skill decomposition. Extract: what patterns have
practitioners converged on for sharing procedures across multiple prompt modules/skills?
If searches yield insufficient evidence because the ecosystem is too young, note that
explicitly and treat the ABSENCE as a valid finding (the pattern is uncharted); fall back to
Beat 3's software-engineering theory for guidance.

**Beat 3 — Software engineering lens (SRP/DRY applied to prompt artifacts).** Survey at least
one source applying classic SE decomposition principles (SRP, DRY, cohesion/coupling) to
prompt engineering or LLM instruction authoring. Candidates: "prompt design patterns" academic
papers, Anthropic's own meta-prompting guidance, practitioner essays on modular LLM prompting.
Extract: does the SE literature have a clear precedent for "shared procedure libraries" vs
"inline duplication" in prompt artifacts, and what does it say about the granularity of splits?

**Beat 4 — Failure modes of skill decomposition.** Survey AT LEAST TWO sources of evidence for what goes wrong when
skills are split. Candidates: the notes 78/80 prior-recall-split rollback (already in vault),
any community reports of skills that were decomposed and then merged back, discussions of
"micro-prompt" anti-patterns. Extract: what is the minimum unit of skill decomposition that
remains effective?

**Output:** one consolidated research summary (`docs/design/2026-07-04-atomic-skills-research.md`)
organized by beat, with a dedicated Section B per beat for vision-relevant-but-out-of-scope
findings. Research summary is committed at S0 completion. Options catalog (below) may be amended
based on S0 findings before options are written to the final deliverable.

---

## Options catalog (2-4, honest CONTENDER/PARK)

> **Note:** options are drafted based on the verified facts above. S0 research may amend.
> All options are evaluated against S1 (four-atom design), S2 (behavior preservation), P1
> (firing surface), P2 (please's heaviness), P4 (discipline-enforcement framing).

### O-A: Atom sub-skills with non-triggering descriptions — CONTENDER

**Shape:** Create 1-2 new skills containing the shared procedure text, with descriptions
explicitly marked as internal/non-triggering (e.g., "Internal shared write-memory procedure —
invoked explicitly by recall and learn skills; do not invoke independently"). Each parent skill
(recall, learn) replaces its duplicated blocks with an explicit "invoke the write-memory skill
for this step" instruction. The atom's SKILL.md contains the authoritative procedure text.

**Atom candidate — `write-memory`:** contains `engram learn fact|feedback` invocation, `engram
learn qa` invocation, `--supersedes` guidance, vocab-tags-automatic rule, QA D2 bar rule. This
covers the three-copy `learn fact|feedback` duplication and the two-copy `learn qa` duplication.

**What changes:**
- `recall/SKILL.md` Steps 2.5C and 4: replace the duplicated write blocks with "invoke the
  write-memory skill; follow its procedure for [absent/near/synthesis] cases"
- `learn/SKILL.md` Steps 2 and 2.5: replace the write invocations with "invoke the write-memory
  skill; follow its procedure for [correction/fact/qa-capture] cases"
- New `skills/write-memory/SKILL.md (repo source; deployed by engram update)`: authoritative write-procedure text
- `please/SKILL.md`: no change (already defers to /learn for all writes)
- `route/SKILL.md`: no change

**Future single-point update scenarios (worked mini-diffs):**
- QA-capture flag change — ONE file, `skills/write-memory/SKILL.md`:
  before: `engram learn qa --slug "<kebab>" --question "<verbatim>" ...`
  after:  `engram learn qa --slug-id "<kebab>" --question "<verbatim>" ...`
  recall Step 4 and learn Step 2.5 each read "apply the write-memory atom (invoke the
  write-memory skill)"; their own text is untouched and picks the change up at next invocation.
- learn-fact flag change — same shape, ONE file: the same atom file's fact|feedback block,
  e.g. `--source "<...>"` → `--source-ref "<...>"`, one edit; identical for every caller.

**Firing-surface delta:** write-memory's description adds ~80-150 chars to the system prompt
metadata. With a deliberately non-triggering description, it does NOT add an autonomous firing
decision — the description signals to the agent "do not invoke this skill on your own." This
is testable: does an agent that sees this description spontaneously invoke write-memory without
a parent skill instructing it? (Pre-registered negative hypothesis.)

**Token/procedure-tax delta:** parent skill bodies shrink by ~60-80 lines (combined). write-memory
SKILL.md adds ~70 lines. Net: roughly token-neutral at the body level; the metadata overhead
is tiny (~10-15 tokens for the description).

**Maintainability delta:** the three-copy write procedure becomes ONE authoritative source
(a flag change: 1 edit instead of 3; qa-capture: 1 instead of 2). UNKNOWN: does an agent
mid-skill reliably invoke the atom at the right step, or silently drop it? That is the
load-bearing smoke question — a skipped atom is within-skill under-fire, a silent step loss.

**Risks:**
- If the agent fails to invoke the atom at the right step (under-fire within the skill), the
  step is silently dropped. This must be verified in smoke tests.
- The "invoke write-memory at this step" instruction must be precise enough that the agent
  knows what invocation input to pass (the local context: covered/near/absent verdict, chunk
  sources, etc.).

**SRP alignment:** achieves SRP for write-memory. The three-copy violation is resolved. The
two-copy ingest sweep (recall 0.5, learn 1) is intentional (different surrounding context) and
may not warrant extraction.

### O-B: Prose cross-references only (no structural change) — CONTENDER

**Shape:** Keep all four skill files as-is in structure. Replace the worst duplications with
prose cross-references that tell the agent "apply the procedure from skill X, step Y." No new
skills. No new files.

**Primary target — QA capture (2-copy):** recall Step 4's QA block is replaced with: "For the
QA pair capture procedure, apply learn Step 2.5 verbatim. Do not re-derive the procedure or
re-emit the command — invoke `/learn` if needed, or execute the same logic learn Step 2.5
describes." This makes recall Step 4's QA block a pointer, not an independent copy.

**Secondary target — `--supersedes` guidance (3-copy):** add a sentence at the first occurrence
in each call site that says "the `--supersedes` rule is stated once in learn Step 2 — apply it
here exactly as written there." This keeps the invocation inline but consolidates the rule's
canonical statement.

**Not targeted — `engram learn fact|feedback` invocation (3-copy):** the flag invocations in
recall 2.5C, recall Step 4, and learn Step 2 remain as independent copies. The reason: these
invocations are tightly coupled to local context (the covered/near/absent verdict, the chunk
sources, the note kind decision) and removing the explicit invocation from recall's context
risks under-firing the write step. Cross-referencing a flag call is less legible than keeping
it inline.

**Not targeted — `engram ingest --auto` (2-copy):** the two occurrences serve different
surrounding logic; marking them as "intentional parallel" (a comment) is sufficient.

**What changes:**
- `recall/SKILL.md` Step 4: QA block replaced with a cross-reference to learn Step 2.5
- `recall/SKILL.md` Step 2.5C and Step 4: `--supersedes` paragraph replaced with a pointer to
  learn Step 2's canonical statement of the rule
- `learn/SKILL.md`: `--supersedes` paragraph in Step 2 gains a "canonical statement" label
- No new files or skills

**Future single-point update scenarios (worked mini-diffs):**
- QA-capture flag change — ONE file after restructure. learn Step 2.5 becomes the designated
  owner; recall Step 4's inline block is replaced once by a pointer:
  before (recall Step 4): the full 12-line `engram learn qa ...` invocation block
  after  (recall Step 4): "Write the qa pair per learn Step 2.5's procedure (single owner of
  qa-capture mechanics), contributors = the [[full-basename]] wikilinks in this synthesis."
  Then the future flag edit lands in learn Step 2.5 only.
- learn-fact flag change — honest O-B limit: the SAME one-line edit lands in THREE places
  (recall Step 2.5C, recall Step 4, learn Step 2 — e.g. `--source "<...>"` → `--source-ref
  "<...>"` ×3, identical text at each site). Still three files; the fact/feedback trio is not
  targeted by O-B.

**Firing-surface delta:** zero. No structural changes; no new skills.

**Token/procedure-tax delta:** parent skill bodies shrink by ~25-35 lines (combined) from the
replaced QA block and supersedes consolidation. No new files added.

**Maintainability delta:** qa-capture consolidates to a one-file edit; the fact/feedback
invocation trio — the LARGER duplication — remains 3 copies. O-B fixes the smaller half of
the problem.

**Risks:**
- Cross-references can drift: if learn Step 2.5 is renumbered or its QA procedure is restructured,
  recall's pointer breaks silently. Must add a comment at the target site: "CANONICAL — recall
  Step 4 cross-references this section; do not move without updating the pointer."
- Smoke test: verify that recall Step 4 still produces correct QA capture output when the text
  is a cross-reference instead of an independent copy. The agent must follow the pointer.

**SRP alignment:** PARTIAL. The QA duplication is resolved. The write-note duplication (3 copies)
is not. O-B is lower-risk than O-A (zero new skills, zero firing-surface delta) and
lower-value (the hardest duplication stays).

### O-C: Atom sub-skills with active (self-triggering) descriptions — PARK

**Shape:** same as O-A, but each atom skill has a description strong enough to trigger it
autonomously (e.g., "Use when writing a vault note with `engram learn` — handles fact, feedback,
and QA pair invocations").

**Why PARK:** P1 (measured prior). Firing surface is the highest-risk axis. A write-memory skill
with an active description that fires on "writing vault notes" would compete with learn's own
description for the same triggering condition. Joe's S1 constraint: "without ending up with N
skills that almost all do the same thing." An autonomous write-memory skill that fires on the
same conditions as learn is exactly that pattern. The over-fire history (147× at one trigger
point, note 139) makes this unacceptable without a full equivalence evaluation — which is larger
than the smoke test budget.

**If the S0 research produces strong evidence that a self-triggering atom can be isolated to a
precise, non-overlapping triggering condition that does not compete with learn/recall, revisit as
O-C′ and note the S0 evidence explicitly before implementing.**

### O-D: Per-skill `references/` factoring (intra-skill only) — PARK

**Shape:** extract each skill's own repeated sections into `skill-name/references/` subdirectory
files. E.g., recall gets `recall/references/write-note.md` with the write procedure; learn gets
`learn/references/write-note.md` with the same procedure.

**Why PARK:** does not achieve the stated SRP goal. Both `recall/references/write-note.md` and
`learn/references/write-note.md` exist as independent copies; a write-note flag change still
requires updating N files. This is a code-organization improvement within a single skill, not
a cross-skill deduplication. It is a valid refactoring step AFTER O-A or O-B has been chosen,
to organize each skill's remaining content — not a standalone solution.

---

## Smoke tests (pre-registered, ~$5-15 total)

Test the leading two options (O-A, O-B). For each, build the refactored skill text in a SANDBOX
COPY (at `/tmp/skills-sandbox/` or similar). Run headless equivalence batteries (`claude -p`)
comparing old skill text vs new skill text. Arms are sandboxed (ENGRAM_VAULT_PATH to a throwaway
directory; no production vault). Fixture mechanism (Gate A correction): each arm's CLAUDE.md
INLINES the full skill text under test (old or new) verbatim — no @import of sandbox paths.
Rationale: @imports of never-before-imported paths trigger a one-time interactive approval
dialog (note 151) that a headless arm cannot answer; the import would fail SILENTLY, both arms
would run against deployed skills, and the Scenario-4 control would falsely pass. Each inlined
text also carries a unique sentinel line (`SANDBOX-MARKER-<option>-<arm>`), and every arm's
prompt ends with "state the sandbox marker you see" — an arm that cannot echo its marker
proves the text did not load, and that arm is INVALID (harness bug, not a behavioral result).

### Scoring model (pre-registered, applies to every scenario)

Each scenario is scored against a FIXED behavioral CHECKPOINT (below), not arm-vs-arm output
matching (LLM prose varies; checkpoints don't). n = 3 fresh headless arms per text version
(old, new) per scenario. Scenario score = number of arms (0–3) hitting the checkpoint.

- **Option passes a scenario** iff new_score >= old_score AND no new arm produces the
  scenario's DISQUALIFIER unless an old arm also produced it.
- **Option passes its smoke** iff it passes ALL scenarios. A single scenario failure = the
  option FAILS, period — no post-hoc reinterpretation.
- **Improvement channel** (Joe: "if any metrics got better, I'd welcome that"): new_score >
  old_score is REPORTED as an observation per scenario; it never changes the pass bar.

### Pre-registered equivalence scenarios

**Scenario 1 — recall Step 2.5 coverage.** Fixture: authored inline in the harness (committed
before running) — a one-cluster payload whose single candidate_l2 addresses the situation but
omits one substantive claim the members evidence (a textbook NEAR). Prompt (verbatim):
"You are running the recall skill, deep mode, Step 2.5, cluster 0. Payload excerpt: <the
fixture YAML>. Judge coverage per the skill and state the EXACT engram command you would run,
flags included." CHECKPOINT: verdict NEAR + an `engram amend --target <candidate>` command
carrying content flags (--subject/--predicate/--object or --behavior/--impact/--action).
DISQUALIFIER: `engram learn` (new note) on this NEAR fixture, or amend with no content flags
(covered-style link-only amend).

**Scenario 2 — recall Step 4 QA capture.** Fixture: REUSE VERBATIM the corrected Task-7 GREEN
prompt from docs/superpowers/plans/2026-07-03-qa-memory-round1-build.md (the eval-checkpointing
synthesis citing [[159.2026-07-02.eval-runs-checkpoint-per-trial]]). CHECKPOINT: names/executes
`engram learn qa` with `--contributors 159.2026-07-02.eval-runs-checkpoint-per-trial`
(wikilink-derived). DISQUALIFIER: free-listed contributors (any basename not present as a
wikilink) or skipping the capture.

**Scenario 3 — learn correction crystallization.** Fixture prompt (verbatim): "You are running
the learn skill. Step 1 done: engram ingest --auto swept 3 chunks. Step 1.5: verdict OK, qa
round-2 gate accumulating (1/20). This session contained ONE user correction: 'don't suppress
lint warnings — fix the underlying issue' (context: you had proposed adding a nolint directive
to silence a warning). List the EXACT commands you run for the remaining steps." CHECKPOINT:
one `engram learn feedback` with --behavior/--impact/--action populated and a retrieval-shaped
--situation. DISQUALIFIER: `engram learn fact` for a correction, zero writes, or >1 note for
the single principle.

**Scenario 4 — CONTROL: harness reproducibility (please text unchanged in both arms).**
This scenario does NOT test anti-amnesia capability — it CANNOT (note 85: that failure is
emergent from rich session context, not cheaply reproducible). P2's anti-amnesia protection
is guaranteed by the NON-MODIFICATION constraint on please in both contenders, and the
deliverable must state that explicitly. Purpose here: harness validation only. If old-vs-old diverges beyond the variance rule, the harness is broken —
stop and investigate before reading any option result. Fixture prompt (verbatim): "The user
asked: '/please rename the variable x to count in utils.py — tiny change, no ceremony, skip
the plan.' Per the please skill, describe your first three actions, in order." CHECKPOINT
(and the pre-registered VARIANCE RULE): prose may vary freely; an arm passes iff its first
three actions are semantically {run /learn (capture-open), run /recall (orient), write the
plan — NOT skipped despite the user's request} in that order. Bit-identical output is NOT
expected or required. DISQUALIFIER: skipping the plan because the user asked to.

**Reporting:** every scenario reports n, model, per-arm one-line verdicts, old_score,
new_score, disqualifier incidents, and the fired branch. All fixtures and prompts are
committed in the harness BEFORE any arm runs.

---

## Steps 0-N

**Step 0 — Verify working tree (pre-flight, free).** Confirm that deployed skill files
(`~/.claude/skills/recall/SKILL.md` etc.) match the repo source (`skills/recall/SKILL.md` etc.)
by diffing. If they diverge, note the divergence in the deliverable before proceeding — the
census and options are written against the repo source, which is the canonical text.

```bash
diff ~/.claude/skills/recall/SKILL.md \
  /Users/joe/repos/personal/engram/skills/recall/SKILL.md
diff ~/.claude/skills/learn/SKILL.md \
  /Users/joe/repos/personal/engram/skills/learn/SKILL.md
diff ~/.claude/skills/please/SKILL.md \
  /Users/joe/repos/personal/engram/skills/please/SKILL.md
diff ~/.claude/skills/route/SKILL.md \
  /Users/joe/repos/personal/engram/skills/route/SKILL.md
```

If any diff is non-empty, surface it to Joe before proceeding.

**Step 1 — Research (S0).** Fan out four parallel research agents (one per beat; sonnet for analysis-heavy beats, haiku for doc-extraction beats, per the route rubric). Each agent
runs independently; no shared context. Consolidate all returns into `docs/design/2026-07-04-
atomic-skills-research.md`. Commit the research doc before proceeding.

**Step 2 — Options revision (post-research).** Read the consolidated research doc. If S0 produced
evidence that amends any option (e.g., a clear community precedent for cross-skill atoms, or a
strong argument against it), record any S0-driven amendments in the DELIVERABLE's options catalog (docs/design/2026-07-04-atomic-skills-options.md), each marked with an explicit "S0 update:" note; do not rewrite this plan document.
If no amendment is warranted, note "S0: no amendments to options catalog."

**Step 3 — Smoke test setup.** Create `/tmp/skills-sandbox/` with subdirectories `recall/`,
`learn/`, `write-memory/`. For each option under test:
- O-A sandbox: write the new recall SKILL.md (with atom invocation), new learn SKILL.md (with
  atom invocation), and the write-memory SKILL.md (the atom body).
- O-B sandbox: write the new recall SKILL.md (with cross-references) and the new learn SKILL.md
  (with canonical supersedes label). Old texts go to `recall-old/` and `learn-old/` for
  comparison.
- Per-arm vault isolation (P8, Gate A correction — arms WRITE notes, so a shared vault would
  leak state across arms): each arm gets its OWN throwaway vault at
  `/tmp/smoke-vault-<scenario>-<option>-<arm>/`, seeded fresh from the same 3-5 fixture notes;
  export ENGRAM_VAULT_PATH per arm. No vault is ever shared between arms.

**Step 4 — Run equivalence batteries.** For each scenario (1-4), run headless arms old vs new.
Minimum n=3 per arm per scenario. Record: pass/fail per scenario per option, exact failure mode
if failing, model used. Budget: ~$5-10 for four scenarios × two options × n=3.

**Step 5 — Compile deliverable.** Write `docs/design/2026-07-04-atomic-skills-options.md` with:

1. Research summary (by beat, with Section B out-of-scope findings).
2. Options table (all four options, CONTENDER/PARK, one-line rationale per decision).
   Report note: whichever option ships later owes GLOSSARY entries for 'atom' and
   'non-triggering description' (docs-gate recommendation; out of scope this round).
3. For each CONTENDER: worked examples for both the QA-capture-change scenario and the
   learn-fact-flag-change scenario (how many files to edit, what the edit looks like).
4. For each CONTENDER: smoke-test results (pass/fail per scenario, n, model, date).
5. Recommendation (if smoke tests distinguish): which option the results support, with caveats.
   If both CONTENDERs pass, present both with their tradeoffs explicitly — do not pick for Joe.

**Step 6 — PRESENT TO JOE AND STOP.** Do not begin any production skill edits. Do not merge
sandbox changes into the deployed skills. Present `docs/design/2026-07-04-atomic-skills-options.md`
to Joe and wait for his call.

---

## Deliverables

- `docs/design/2026-07-04-atomic-skills-research.md` — S0 consolidated research (committed at
  Step 1 completion)
- `docs/design/2026-07-04-atomic-skills-options.md` — options table + worked examples + smoke
  test results (committed at Step 5 completion)
- `docs/superpowers/plans/2026-07-04-atomic-skills-exploration.md` — this plan (committed now)

**Spend estimate (no cap; runs to completion):** S0 research ~$3-8 (four web-fetch + web-search
agents, haiku/sonnet); smoke tests ~$5-10 (headless arms, haiku/sonnet per the route rubric);
writing ~$0. Total ~$8-18.

---

## Unresolved questions (do not guess; surface to Joe if blocking)

**UQ1 — S1 roadmap atom semantics.** ROADMAP:175–181's "atoms = read-memory, write-memory, route-a-
task, orchestrate-a-workflow" — does Joe mean these as NEW skill files with their own frontmatter,
or as conceptual groupings that inform how existing skill bodies are organized? This matters for
whether O-A (atom sub-skills) is faithful to S1 or whether O-B (prose refactor) is closer. The
plan assumes both interpretations are worth testing (hence two CONTENDERs), but if Joe has a
strong preference, O-C or O-D may be revisited.

**UQ2 — please's step 7 explicit deduplication.** please Step 7 already says "The learn skill's
Step 2.5 handles ad-hoc QA pair capture — do not duplicate that logic here." Is this the intended
model for how skill cross-references should work (prose pointer, no new skill), which would favor
O-B? Or is it a gap to fix (please should explicitly invoke a write-memory atom), which would
favor O-A? The smoke tests treat please as unchanged; this question is for the post-options
conversation.

**UQ3 — ingest sweep intentional duplication.** recall Step 0.5 and learn Step 1 both run
`engram ingest --auto`. This appears intentional (different surrounding logic), but neither skill
documents it as deliberate. Should a comment ("intentional parallel — both skills sweep before
their respective operations") be added, or should this be extracted? The plan does not extract it
(rated low-impact); confirm if Joe wants it addressed.
