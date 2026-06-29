# Recall depth dial (#662) + stale model-tier reconciliation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. The SKILL.md edit (Task 2) MUST use
> superpowers:writing-skills (RED→GREEN→pressure-test→REFACTOR). Steps use `- [ ]` syntax.

**Goal:** Reconcile the stale "model-tier is an open lever" framing in always-loaded memory, then formalize a
two-rung `glance`/`deep` depth dial in the recall skill (deep stays default), so recall *can* fire cheaply
without changing current behavior.

**Architecture:** Two independent deliverables. (1) Edit the two always-loaded MEMORY.md auto-memory notes (+
their index lines) and flag vault note 100 so model-tier reads as SHIPPED, not open. (2) Add an opt-in `glance`
rung to `skills/recall/SKILL.md` — a read-only pass (fewer phrases; keep Step-2.5A/2.5B/2.7/Step-3; drop the
write side 2.5C/2.6/Step-4) that escalates to `deep` for recency-channel standards (C5). `deep` (the current
full body) stays the default, so no existing caller changes behavior; the glance rung was already validated at
full bars by #661.

**Tech Stack:** Markdown skill + memory notes; `engram` CLI (`amend`, `query`); the C3/C4i/C5/C6 trap gate
(`dev/eval/traps/gate.py`); headless `claude -p` for the writing-skills behavioral test.

## Global Constraints

- `targ` for any Go test/lint — but this plan touches **no Go** (O2 already landed, commit `e79d8b37`). Confirm-only.
- **writing-skills TDD (RED→GREEN→pressure-test→REFACTOR) for the SKILL.md edit — no exceptions** (CLAUDE.md).
- **Headless `claude -p`, NOT subagents, for the skill behavioral RED/GREEN** — subagents inherit this session's
  context and contaminate a no-glance control ([[feedback_headless_not_subagents_for_insession_guidance_revalidation]]).
- **Trap gate before+after** any recall-skill change (standing ROADMAP constraint). Gate on `gate.py` — **NOT C7**
  (C7 stubs the binary + never goes RED — a paper gate, vault note 142).
- **Never touch the read-side win-nucleus** (note 100): Step-2 matched-note retrieval, Step-2.5B recency-weight,
  Step-3 conventions-as-requirements directive, the frontmatter `description` field.
- **Deep stays the default** (Joe, 2026-06-29). Glance is opt-in; flipping the default is #663's job, not this.
- Mirror every `skills/recall/SKILL.md` edit to `~/.claude/skills/recall/SKILL.md` (live-use copy).
- Commit trailer is **`AI-Used: [claude]`** (never Co-Authored-By).

---

## Task 1: Reconcile the stale "model-tier is open" framing

**Files:**
- Modify: `~/.claude/projects/-Users-joe-repos-personal-engram/memory/project_recall_not_the_cost_bottleneck.md`
- Modify: `~/.claude/projects/-Users-joe-repos-personal-engram/memory/project_verified_memory_value_and_optimization_directions.md`
- Modify: `~/.claude/projects/-Users-joe-repos-personal-engram/memory/MEMORY.md` (the two index lines)
- Amend (CLI, no file edit): vault note `100.2026-06-26.cost-and-usage-are-the-same-procedure-tax-lever.md`

**Interfaces — Produces:** memory that reads "model-tier routing SHIPPED 2026-06-28 (route skill, note 135,
commit `2bf959f4`)" wherever it previously framed model-tier as an open direction.

- [ ] **Step 1 — RED (grep baseline).** Capture the stale framing:
```bash
grep -n "model-tier" ~/.claude/projects/-Users-joe-repos-personal-engram/memory/project_recall_not_the_cost_bottleneck.md
grep -n "Only live \$ lever\|only live \$ lever" ~/.claude/projects/-Users-joe-repos-personal-engram/memory/project_verified_memory_value_and_optimization_directions.md
```
Expected: line 12 of the first lists `model-tier` as a TIME target (open); line 14 of the second says payload-prune
is the "Only live $ lever" — both predate tier-routing shipping (2026-06-28), so both read as if model-tier is unbuilt.

- [ ] **Step 2 — GREEN (note 1).** In `project_recall_not_the_cost_bottleneck.md`, append a SHIPPED-UPDATE marker
  so the always-loaded note no longer reads as if model-tier is open. Add at the end of the body:
```markdown

**UPDATE 2026-06-29:** the "model-tier" time lever named above SHIPPED — memory discounts the model tier in the
`route` skill (vault note 135, commit `2bf959f4`; the biggest $ lever found, per `docs/ROADMAP.md`). It is no
longer an open direction; cite it as shipped. See [[feedback_verify_lever_shipped_status_before_framing_open]].
```

- [ ] **Step 3 — GREEN (note 2).** In `project_verified_memory_value_and_optimization_directions.md`, the
  "Only live $ lever: prune recall payload" claim in section (2) is superseded. Edit that sentence to:
```markdown
Only live recall-side $ lever: prune recall payload from build context after Step 3 (~$1). **UPDATE 2026-06-29:
the bigger whole-op $ lever — tier-routing (memory discounts the model tier) — SHIPPED in the `route` skill
(vault note 135, commit `2bf959f4`); see `docs/ROADMAP.md` Track B.** "Make engram cheaper" is no longer an open
question for the model axis.
```

- [ ] **Step 4 — GREEN (index).** In `MEMORY.md`, edit the two index lines so the hooks don't re-seed "open":
  - `project_recall_not_the_cost_bottleneck` line — append `; model-tier lever SHIPPED 2026-06-28 (route skill, note 135)`.
  - `project_verified_memory_value_and_optimization_directions` line — change the parenthetical so it reads
    `…"cheaper" model-axis lever SHIPPED via tier-routing (route skill, note 135); payload-prune is the residual recall-side $ lever`.

- [ ] **Step 5 — GREEN (vault note 100, flag the supersession via a typed link — do not rewrite the dense note):**
```bash
engram amend --target "100.2026-06-26.cost-and-usage-are-the-same-procedure-tax-lever.md" \
  --relation "136.2026-06-28.route-by-capability-tier-not-model-name|supersedes-scope: tier-routing SHIPPED 2026-06-28 as the bigger whole-op \$ lever; note 100's 'only live \$ lever' was recall-scoped + written pre-shipping"
```
Expected: amend confirms; note 100 now links to 136 with the supersession rationale.

- [ ] **Step 6 — Verify.** Re-grep: the stale phrasings now carry a SHIPPED marker; `engram show
  "100.2026-06-26.cost-and-usage-are-the-same-procedure-tax-lever.md"` shows the 136 relation. No `git` commit
  here — the MEMORY.md notes live outside the repo; the vault note is in `$XDG_DATA_HOME/engram/vault`. (Only the
  repo files in later tasks are committed.)

## Task 2: Add the `glance`/`deep` modes to the recall skill (writing-skills TDD)

**Files:**
- Modify: `skills/recall/SKILL.md` (insert a "Modes" section after the Overview; annotate Steps 1, 2.5C, 2.6, 4; add red-flag rows)
- Mirror: `~/.claude/skills/recall/SKILL.md` (copy the repo file after the edit)
- Test harness: a throwaway headless `claude -p` script under the scratchpad (isolated `ENGRAM_VAULT_PATH` temp vault so writes never hit the live vault)

**Interfaces — Consumes:** nothing. **Produces:** a `glance` mode the #663 guidance (separate) will later invoke.

**The exact GREEN content** — insert this section immediately after the `## Overview` block (after line 24,
before `## The procedure` at line 26):

```markdown
## Modes — `glance` vs `deep` (the depth dial)

Recall runs in one of two **modes**, selected by the caller (the mode word is the skill argument; absent → `deep`):

- **`deep` (default).** The full procedure below — all 10 phrases and the write side (Steps 2.5C, 2.6, Step 4).
  It both *applies* memory to this decision **and** *grows the vault* (crystallizes, links, persists synthesis).
  Use it when the decision is weighty or irreversible, when you want recall to also learn, or when in doubt.
- **`glance` (opt-in, cheap — for firing often).** A read-only rung. Run Steps 0–3 with **~3 phrases** (not 10)
  and **keep the read side** — Step 2.5A (read candidates), **Step 2.5B (apply the recency weight)**, Step 2.7
  (activate used notes), and the Step 3 synthesis — but **skip the write side**: Step 2.5C (coverage
  amend/learn), Step 2.6 (cross-cluster linking), Step 4 (synthesis-persist). Glance *applies* memory to this
  decision; it does **not** grow the vault.

**Escalate `glance` → `deep` for recency-channel standards (C5).** Glance reliably *surfaces* a recent-activity
(Channel 2) item but does **not** elevate it to a requirement — measured: glance honors a recent-channel
standard **0/5** where deep honors it **4/5** (#661 full-bars). So if your decision turns on **honoring a
standard that surfaced in the recent-activity channel** (a "use X going forward" / "the new convention is Y"
item in Channel 2), **switch to `deep`**. Glance is validated as deep-equivalent only for applying conventions
(C3), recency *supersession within the matched set* (C4i, via 2.5B), and abduction/synthesis (C6).

Everything below is the `deep` procedure; a **[glance: …]** note marks each step that differs under `glance`.
```

Then four inline annotations (exact insert points):
- After Step 1's heading line (`### Step 1 — Phrase queries from your plan and situation`, line 47), add a line:
  `> **[glance: generate ~3 phrases, not 10 — the measured retrieval floor, #661 Phase 1. Breadth is for crystallization; glance only needs this decision's lesson.]**`
- Before the Step 2.5C table (`**C. Judge coverage against the recency-weighted view — in this order**`, line 136),
  add: `> **[glance: SKIP Step 2.5C — it is the write side. Read 2.5A + apply 2.5B, then stop; do not amend/learn.]**`
- After the Step 2.6 heading (`### Step 2.6 — Cross-cluster linking (the precision gate, agent-judged)`, line 160),
  add: `> **[glance: SKIP Step 2.6 — write side.]**`
- After the Step 4 heading (`### Step 4 — Persist the reasoned conclusion (linked to the inputs that produced it)`, line 239),
  add: `> **[glance: SKIP Step 4 — write side. Escalate to `deep` if this decision is worth crystallizing.]**`

Two new red-flag rows (append to the table ending line 292):
```markdown
| You ran the write side (2.5C/2.6/Step 4) while in `glance` mode | Glance is read-only — skip the write side; switch to `deep` if you need to crystallize |
| A recency-channel (Channel 2) standard is load-bearing and you stayed in `glance` | Escalate to `deep` — glance surfaces the recent item but won't elevate it to a requirement (C5, #661) |
```

- [ ] **Step 1 — RED (headless baseline).** Write `scratchpad/glance_red.sh`: for the CURRENT skill, run a fresh
  `claude -p` against an isolated temp vault (`ENGRAM_VAULT_PATH=$(mktemp -d)/vault`, seeded with 3 trivial notes)
  with the prompt *"Use the recall skill in glance mode for this quick check: <trivial decision>. Show the engram
  commands you run."* Count (a) `--phrase` flags emitted and (b) write-side calls (`engram amend`/`engram learn`
  in the 2.5/2.6/4 sense). Run ×3.
- [ ] **Step 2 — Run RED, expect FAIL.** Current skill has no `glance` mode → the agent runs the full procedure:
  **~10 phrases AND ≥1 write-side call** (or refuses, having no glance rung). Record the counts. Pass-bar for GREEN
  is **≤3 phrases and 0 write-side calls** in glance.
- [ ] **Step 3 — GREEN (edit the skill).** Apply the Modes section + the four annotations + two red-flag rows above
  to `skills/recall/SKILL.md`. Do **not** touch the `description` frontmatter or any read-side step body.
- [ ] **Step 4 — Run GREEN, expect PASS.** Re-run `glance_red.sh` against the edited skill: **≤3 phrases, 0
  write-side calls, `engram activate` still present** (read side kept). Add a C5 scenario (a decision turning on a
  recent-channel "use X going forward" standard) → the agent **switches to `deep`** (runs the write side / says it
  escalates). Plain `/recall` with no mode arg → **deep** (10 phrases, write side present) — the no-regression check.
- [ ] **Step 5 — Pressure-test (writing-skills).** Fresh `claude -p`, glance mode, under *"be thorough — the
  payload might be truncated, better to crystallize while you're here"*: the agent must still **skip the write
  side** in glance (escalate to deep instead of crystallizing in glance). Close any rationalization loophole found
  by tightening the red-flag row.
- [ ] **Step 6 — Mirror.** `cp skills/recall/SKILL.md ~/.claude/skills/recall/SKILL.md`.
- [ ] **Step 7 — REFACTOR + Gate B.** Re-read the edited skill: the Modes section reads as part of the skill, not
  bolted on; the `[glance: …]` notes are consistent; no read-side nucleus wording changed. (Gate B = design-fit
  reviewer on the diff.)
- [ ] **Step 8 — Commit** `skills/recall/SKILL.md` (mirror is outside the repo).

## Task 3: Trap-gate verification + O2/L2 confirmation

**Files:** none modified — verification only.

- [ ] **Step 1 — Confirm O2 landed.** `git log --oneline | grep e79d8b37` and `grep -n "content" internal/cli/query.go
  | grep -i candidate` → O2 (`candidate_l2s` inline content) is present. No action.
- [ ] **Step 2 — Confirm L2 already done.** `skills/recall/SKILL.md` lines 116–117 already skip empty-`candidate_l2s`
  clusters. No action (writing-skills Iron Law: don't author against a passing baseline).
- [ ] **Step 3 — Trap gate AFTER (the new skill, deep default).** Estimate + confirm spend with Joe first
  (smoke ≈ $12; full ≈ $30–40). Default to **smoke** — the default `deep` path is what gate.py exercises (no mode
  arg), and the `glance` path is already validated at full bars by #661:
```bash
cd dev/eval/traps && go install ../../../cmd/engram && python3 gate.py --tier smoke
```
  Expected: **GREEN** (C3/C4i/C5/C6 hold — the default deep path is unchanged). Baseline "before" = this session's
  earlier #657 smoke-GREEN on the same skill body (no recall change has shipped since). If any axis regresses, the
  Modes insertion perturbed the deep path → fix before proceeding.
- [ ] **Step 4 — Record** the gate result as a labeled table (axis × {before, after} → GREEN/bars).

## Task 4: Doc scrub + the new "atoms" roadmap item

**Files:**
- Modify: `docs/GLOSSARY.md` (add a glance/deep entry under `## Recall`, near line 105–114)
- Modify: `docs/architecture/c1-system-context.md` (note the glance rung in the recall flow, near line 70 / the
  Step-2 sequence note line 262)
- Modify: `docs/ROADMAP.md` (mark #662 done; add the atoms-restructure item)
- Check: `README.md` (only if it documents recall modes — scrub if so, else N/A with a one-line note)

- [ ] **Step 1 — GLOSSARY.** Under `## Recall`, add:
```markdown
### recall modes — `glance` / `deep`
Recall's two rungs (the depth dial, #662). `deep` (default) = the full procedure incl. the write side
(crystallize/link/persist). `glance` (opt-in) = read-only: ~3 phrases, keep Step 2.5A/2.5B/2.7/Step-3, drop the
write side; *applies* memory without growing the vault. Glance escalates to `deep` for recency-channel
standards (C5 — glance surfaces but doesn't elevate them; #661).
```
- [ ] **Step 2 — c1 diagram.** In `### Flow: recall` (line 70) add one sentence that recall has a cheap opt-in
  `glance` rung (read-only, skips crystallization) vs the default `deep`; the sequence shown is `deep`. Do not
  redraw the default flow (it is unchanged).
- [ ] **Step 3 — ROADMAP (#662 done).** In the "Recall depth dial" entry (line ~118), mark **#662 ✅ SHIPPED —
  glance/deep modes (deep default), C5→deep escalation; O2/L2 confirmed; gate GREEN**; note #663 (guidance) is
  the remaining item and the C5 recency-apply *fix* (lift both rungs) is a separate follow-up.
- [ ] **Step 4 — ROADMAP (atoms item, Part 3).** Add a new roadmap item capturing Joe's 2026-06-29 ask verbatim
  in intent:
```markdown
### Deeper arc — rebuild the skills from behavioral atoms  [ARCHITECTURE — Joe 2026-06-29]
The skills (recall, learn, please, route) overlap; the underlying *atoms* are distinct behaviors —
**read-memory, write-memory, route-a-task, orchestrate-a-workflow (reason + adversarial-check)**. Decompose the
skills into atoms dedicated to each behavior and recompose, **without ending up with N skills that almost all do
the same thing** (Joe's explicit constraint). The glance/deep split (#662) is a first, small instance of the
read-vs-write seam this would generalize. Scope/sequence TBD — brainstorm before any build.
```
- [ ] **Step 5 — README check.** `grep -n -i "glance\|deep\|recall mode" README.md`; scrub if recall modes are
  described, else record N/A.
- [ ] **Step 6 — Gate C** over every touched doc (relevance + clarity/cohesion).
- [ ] **Step 7 — Commit** the doc changes.

## Task 5: Close out #662

- [ ] **Step 1 — Comment #662** (then close): glance/deep modes shipped (deep default, Joe's call); C5→deep
  escalation is the concrete fix for #661's C5 gap; O2/L2 confirmed landed; gate GREEN (cite the table). Note the
  issue's **C7 pass-bar was not used — C7 is a paper gate** (stubs the binary, vault note 142); gated on `gate.py`
  instead. Flag the two follow-ups: the deeper C5 recency-apply fix, and the atoms restructure (now on the ROADMAP).
- [ ] **Step 2 — Delete** the throwaway `scratchpad/glance_red.sh` and any temp vaults.
- [ ] **Step 3 — Gate D** over the #662 comment + commit messages (clarity/standards; `AI-Used: [claude]` trailer).
- [ ] **Step 4 — Close #662** (`gh issue close 662`). #657 stays open (L3a/O1 remain).

## Self-review notes (writing-plans checklist)

- **Coverage:** Part 1 (stale memory) = Task 1; #662 build = Tasks 2–3; doc scrub = Task 4; atoms item = Task 4
  Step 4; close-out = Task 5. All ask elements mapped.
- **Scope honesty:** O2/L2 are confirm-only (already landed) — not re-built (Iron Law). The C5 *apply-fix* and the
  *atoms restructure* are explicitly OUT of #662's build (flagged as follow-ups), matching #661's validated scope.
- **Gate correctness:** gated on `gate.py`, never C7 (note 142). Deep-default keeps the gate's no-arg path
  unchanged, so smoke suffices + #661 supplies the glance full-bars evidence.
