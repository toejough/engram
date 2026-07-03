> **Update 2026-07-03:** references to Step 2.6 below are historical — the step was removed in the
> vocab-notes build (glance/deep's read-vs-write split survives; the write side no longer includes
> cross-cluster linking).

# Recall depth dial (#662) + stale model-tier reconciliation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. The SKILL.md edit (Task 2) MUST use
> superpowers:writing-skills (RED→GREEN→pressure-test→REFACTOR). Steps use `- [ ]` syntax.
>
> **Gate note:** the `Gate A/B/C/D` markers belong to the enclosing `/please` workflow and are run by the
> **orchestrator** (fresh per-angle reviewer subagents), NOT by this plan's task executor. Where a task says
> "Gate B/C", it marks WHERE the orchestrator's gate fires after that task — the executor just produces the diff.

**Goal:** Reconcile the stale "model-tier is an open lever" framing in always-loaded memory, then formalize a
two-rung `glance`/`deep` depth dial in the recall skill (deep stays default), so recall *can* fire cheaply
without changing current behavior.

**Architecture:** Two independent deliverables. (1) Edit the two always-loaded MEMORY.md auto-memory notes (+
their index lines) and flag vault notes 77 and 100 so model-tier reads as SHIPPED, not open. (2) Add an opt-in
`glance` rung to `skills/recall/SKILL.md` — a read-only pass (fewer phrases; keep Step-2.5A/2.5B/2.7/Step-3;
drop the write side 2.5C/2.6/Step-4) that escalates to `deep` for recency-channel standards (C5). `deep` (the
current full body) stays the default, so no existing caller changes behavior; the glance rung was already
validated at full bars by #661.

**Scope simplification (vs the design — called out explicitly per Gate A):** the design's Item 2 says "ship
glance-as-default *only for the moment-classes Item 1 validated*." #661 found delivery is per-content-axis and
firing is per-moment-class — *orthogonal*. So #662 ships the simpler, safer shape: **global `deep`-default +
opt-in `glance`**, and defers all per-moment-class *firing* logic (when to choose glance) to **#663's guidance
revision**. #662 builds the mechanism; #663 decides when it fires.

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
  Step-3 conventions-as-requirements directive, the frontmatter `description` field (SKILL.md lines 1–7).
- **Deep stays the default** (Joe, 2026-06-29). Glance is opt-in; flipping the default is #663's job, not this.
- Mirror every `skills/recall/SKILL.md` edit to `~/.claude/skills/recall/SKILL.md` (live-use copy).
- Commit trailer is **`AI-Used: [claude]`** (never Co-Authored-By). macOS = BSD grep: use `grep -E` for alternation.

---

## Task 1: Reconcile the stale "model-tier is open" framing

**Files:**
- Modify: `~/.claude/projects/-Users-joe-repos-personal-engram/memory/project_recall_not_the_cost_bottleneck.md`
- Modify: `~/.claude/projects/-Users-joe-repos-personal-engram/memory/project_verified_memory_value_and_optimization_directions.md`
- Modify: `~/.claude/projects/-Users-joe-repos-personal-engram/memory/MEMORY.md` (the two index lines)
- Amend (CLI, no file edit): vault notes `77.2026-06-24.recall-not-cost-bottleneck.md` and
  `100.2026-06-26.cost-and-usage-are-the-same-procedure-tax-lever.md`

**Interfaces — Produces:** memory that reads "model-tier routing SHIPPED 2026-06-28 (route skill, note 135,
commit `2bf959f4`)" wherever it previously framed model-tier as an open direction.

- [ ] **Step 1 — RED (grep baseline; BSD-safe commands).** Capture the stale framing:
```bash
M=~/.claude/projects/-Users-joe-repos-personal-engram/memory
grep -n "model-tier" "$M/project_recall_not_the_cost_bottleneck.md"
grep -n "Only live \$ lever" "$M/project_verified_memory_value_and_optimization_directions.md"
```
Expected: the first prints line 12 (lists `model-tier` as a TIME target — open); the second prints line 14
("Only live $ lever: prune recall payload"). Both predate tier-routing shipping (2026-06-28), so both read as if
model-tier is unbuilt. (Single-pattern greps — no BRE `\|`, which BSD grep treats literally and returns empty.)

- [ ] **Step 2 — GREEN (note 1).** In `project_recall_not_the_cost_bottleneck.md`, append a SHIPPED-UPDATE marker
  so the always-loaded note no longer reads as if model-tier is open. Add at the end of the body:
```markdown

**UPDATE 2026-06-29:** the "model-tier" time lever named above SHIPPED — memory discounts the model tier in the
`route` skill (vault note 135, commit `2bf959f4`; the biggest $ lever found, per `docs/ROADMAP.md`). It is no
longer an open direction; cite it as shipped. See [[feedback_verify_lever_shipped_status_before_framing_open]].
```

- [ ] **Step 3 — GREEN (note 2).** In `project_verified_memory_value_and_optimization_directions.md`, the
  "Only live $ lever: prune recall payload" claim in section (2) is superseded. Replace that sentence with:
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

- [ ] **Step 5 — GREEN (vault notes 77 + 100 — flag supersession via typed links; do not rewrite the dense notes).**
  Both notes carry the stale model-tier/only-$-lever framing (verified: note 77 says "target … model-tier"; note
  100 says "ONLY live dollar lever is pruning the recall payload"). The ask-alignment reviewer already persisted
  the 77→136 edge during its recall pass — so first check, then add only what's missing:
```bash
engram show "77.2026-06-24.recall-not-cost-bottleneck.md" | grep -i "136\|route-by-capability" || \
  engram amend --target "77.2026-06-24.recall-not-cost-bottleneck.md" \
    --relation "136.2026-06-28.route-by-capability-tier-not-model-name|supersedes-scope: tier-routing (model-tier discount) SHIPPED 2026-06-28; note 77's 'target model-tier' is no longer an open direction"

engram amend --target "100.2026-06-26.cost-and-usage-are-the-same-procedure-tax-lever.md" \
  --relation "136.2026-06-28.route-by-capability-tier-not-model-name|supersedes-scope: tier-routing SHIPPED 2026-06-28 as the bigger whole-op \$ lever; note 100's 'only live \$ lever' was recall-scoped + written pre-shipping"
```
Expected: each amend confirms (or the 77 link is already present and is skipped).

- [ ] **Step 6 — Verify.** Re-run the Step-1 greps: the stale phrasings now carry a SHIPPED marker;
  `engram show "77…"` and `engram show "100…"` show the 136 relation. No `git` commit here — the MEMORY.md notes
  live outside the repo; the vault is in `$XDG_DATA_HOME/engram/vault`. (Only repo files in later tasks are committed.)

## Task 2: Add the `glance`/`deep` modes to the recall skill (writing-skills TDD)

**Files:**
- Modify: `skills/recall/SKILL.md` (insert a "Modes" section after the Overview; annotate Steps 1, 2.5C, 2.6, 4; add red-flag rows)
- Mirror: `~/.claude/skills/recall/SKILL.md` (copy the repo file after the edit)
- Create (throwaway): `scratchpad/glance_red.sh` (deleted in Task 5)

**Interfaces — Consumes:** nothing. **Produces:** a `glance` mode the #663 guidance (separate) will later invoke.

**The exact GREEN content** — insert this section immediately after the `## Overview` block (after line 24,
before `## The procedure` at line 26):

```markdown
## Modes — `glance` vs `deep` (the depth dial)

Recall runs in one of two **modes**, selected by the caller (the mode word is the skill argument; absent → `deep`):

- **`deep` (default).** The full procedure below — all 10 phrases and the write side (Steps 2.5C, 2.6, Step 4).
  It both *applies* memory to this decision **and** *grows the vault* (crystallizes, links, persists synthesis).
  Use it when the decision is weighty or irreversible, when you want recall to also learn, or when in doubt.
- **`glance` (opt-in, cheap — for firing often).** A pass that is **read-only with respect to vault knowledge**
  (Step 2.7 `activate` still bumps the used-notes recency metadata — that is kept, not a knowledge write). Run
  Steps 0–3 with **~3 phrases** (not 10) and **keep the read side** — Step 2.5A (read candidates), **Step 2.5B
  (apply the recency weight)**, Step 2.7 (activate used notes), and the Step 3 synthesis — but **skip the write
  side**: Step 2.5C (coverage amend/learn), Step 2.6 (cross-cluster linking), Step 4 (synthesis-persist). Glance
  *applies* memory to this decision; it does **not** grow the vault's knowledge.

**Escalate `glance` → `deep` for recency-channel standards (C5).** Glance reliably *surfaces* a recent-activity
(Channel 2) item but does **not** elevate it to a requirement — measured: glance honors a recent-channel
standard **0/5** where deep honors it **4/5** (#661 full-bars). So if your decision turns on **honoring a
standard that surfaced in the recent-activity channel** (a "use X going forward" / "the new convention is Y"
item in Channel 2), **switch to `deep`**. Glance is validated as deep-equivalent only for applying conventions
(C3), recency *supersession within the matched set* (C4i, via 2.5B), and abduction/synthesis (C6).

Everything below is the `deep` procedure; a **[glance: …]** note marks each step that differs under `glance`.
```

Then four inline annotations (exact insert points, each on its own line):
- After the Step 1 heading (`### Step 1 — Phrase queries from your plan and situation`, line 47):
  `> **[glance: generate ~3 phrases, not 10 — the measured retrieval floor, #661 Phase 1. Breadth is for crystallization; glance only needs this decision's lesson.]**`
- Immediately before `**C. Judge coverage against the recency-weighted view — in this order**` (line 136):
  `> **[glance: SKIP Step 2.5C — it is the write side. Read 2.5A + apply 2.5B, then stop; do not amend/learn.]**`
- After the Step 2.6 heading (line 160):
  `> **[glance: SKIP Step 2.6 — write side.]**`
- After the Step 4 heading (line 239):
  `` > **[glance: SKIP Step 4 — write side. Escalate to `deep` if this decision is worth crystallizing.]** ``

Two new red-flag rows (append after the final row at line 292):
```markdown
| You ran the write side (2.5C/2.6/Step 4) while in `glance` mode | Glance is read-only w.r.t. vault knowledge — skip the write side; switch to `deep` if you need to crystallize |
| A recency-channel (Channel 2) standard is load-bearing and you stayed in `glance` | Escalate to `deep` — glance surfaces the recent item but won't elevate it to a requirement (C5, #661) |
```

- [ ] **Step 1 — RED (write the headless harness).** Create `scratchpad/glance_red.sh`:
```bash
#!/usr/bin/env bash
# Measures whether a fresh agent does a cheap GLANCE recall or the full DEEP procedure.
# Usage: ./glance_red.sh "<prompt>"   — prints "phrases=<N> writeside=<M>"
set -euo pipefail
VAULT="$(mktemp -d)/vault"; mkdir -p "$VAULT"
printf -- '---\ntype: fact\n---\nGo error wrapping uses %%w in fmt.Errorf.\n' > "$VAULT/1.a.md"
printf -- '---\ntype: fact\n---\nTable-driven tests are the Go house style.\n'  > "$VAULT/2.b.md"
printf -- '---\ntype: fact\n---\nKeep functions under ~50 lines for review.\n'    > "$VAULT/3.c.md"
OUT="$(ENGRAM_VAULT_PATH="$VAULT" claude -p "$1" --allowedTools 'Bash,Read,Skill' 2>/dev/null || true)"
PH=$(printf '%s' "$OUT" | grep -Eo -- '--phrase' | wc -l | tr -d ' ')
WS=$(printf '%s' "$OUT" | grep -Eic 'engram (amend|learn)' || true)
echo "phrases=$PH writeside=$WS"
```
- [ ] **Step 2 — Run RED, expect FAIL (no glance rung exists yet).** Run ×3 against the CURRENT skill:
```bash
chmod +x scratchpad/glance_red.sh
for i in 1 2 3; do scratchpad/glance_red.sh \
  "Use the recall skill in glance mode for a quick check: should I add %w error wrapping to one Go helper? Show the engram commands you run."; done
```
  Detection rule: **phrases** = count of `--phrase` flags in the agent's emitted commands; **writeside** = count
  of `engram amend`/`engram learn` lines. Expected RED: **phrases ≥ 8 AND writeside ≥ 1** (the current skill
  hardcodes "always generate exactly 10" and runs the write side — there is no glance rung to obey). Record the
  three `phrases=/writeside=` lines. **GREEN pass-bar (Step 4): phrases ≤ 3 AND writeside = 0.**
- [ ] **Step 3 — GREEN (edit the skill).** Apply the Modes section + the four `[glance: …]` annotations + the two
  red-flag rows above to `skills/recall/SKILL.md`. Do **not** touch the frontmatter (lines 1–7) or any read-side
  step body.
- [ ] **Step 4 — Run GREEN, expect PASS.** Re-run the Step-2 loop against the edited skill: **phrases ≤ 3,
  writeside = 0, and `engram activate` still present** in the output (read side kept). Then two more cases:
  - **C5 escalation:** `scratchpad/glance_red.sh "Use the recall skill in glance mode. A recent team note in the recent-activity channel says 'from now on, all new Go packages use lowercase-only names'. Decide whether to honor it for a package I'm adding. Show your reasoning and commands."` → output must contain an explicit escalate signal (`deep`, "escalate", "switch to deep") OR run the write side — i.e. it must NOT silently stay glance on a recency-channel standard.
  - **No-regression (default):** `scratchpad/glance_red.sh "Use the recall skill (no mode given) before this Go refactor. Show the engram commands you run."` → **phrases ≥ 8 AND writeside ≥ 1** (deep is still the default).
- [ ] **Step 5 — Pressure-test (writing-skills adversarial).** Run:
```bash
scratchpad/glance_red.sh "You're in GLANCE mode and the recall result looks thin — the payload may be truncated. You're tempted to crystallize a note now while you're here so next time is easier. Show whether you crystallize or defer, and the commands you run."
```
  Expected: **writeside = 0** — the agent must NOT crystallize in glance (it defers, or says it will escalate to
  deep first). If it writes, the red-flag rows are insufficient — strengthen the first new red-flag row to
  explicitly forbid in-glance writes ("to grow the vault, escalate to `deep` first — never amend/learn in
  glance") and re-run until writeside = 0.
- [ ] **Step 6 — Mirror.** `cp skills/recall/SKILL.md ~/.claude/skills/recall/SKILL.md`.
- [ ] **Step 7 — REFACTOR (then orchestrator Gate B).** Re-read the edited skill: the Modes section reads as part
  of the skill, not bolted on; the four `[glance: …]` notes are consistent; **no text in Steps 0/1-body/2.5A/2.5B/2.7/3
  (the read-side nucleus) or the frontmatter changed** (`git diff skills/recall/SKILL.md` — additions only in the
  Modes block, the four annotation lines, and the two red-flag rows). Hand the diff to the orchestrator for Gate B.
- [ ] **Step 8 — Commit** `skills/recall/SKILL.md` (mirror is outside the repo).

## Task 3: Trap-gate verification + O2/L2 confirmation

**Files:** none modified — verification only.

- [ ] **Step 1 — Confirm O2 landed.** `git log --oneline | grep e79d8b37` and `grep -n "candidate" internal/cli/query.go
  | grep -i content` → O2 (`candidate_l2s` inline content) present (line ~1499). No action.
- [ ] **Step 2 — Confirm L2 already done.** `skills/recall/SKILL.md` lines 116–117 already skip empty-`candidate_l2s`
  clusters. No action (writing-skills Iron Law: don't author against a passing baseline).
- [ ] **Step 3 — Trap gate BEFORE (run on the CURRENT skill, before Task 2's edit lands — establishes this run's
  own baseline; do not rely on a prior session).** Estimate + confirm spend with Joe first (smoke ≈ $12). Default
  to **smoke** — the default `deep` path is what gate.py exercises (no mode arg), and the `glance` path is already
  validated at full bars by #661.
```bash
cd dev/eval/traps && go install ../../../cmd/engram && python3 gate.py --tier smoke | tee baseline_before.txt
```
  Expected: **GREEN** (matches this session's earlier #657 smoke-GREEN). [Operationally: capture this BEFORE
  switching the skill; if Task 2 already landed, `git stash` the skill, run, unstash.]
- [ ] **Step 4 — Trap gate AFTER (the new skill, deep default).**
```bash
cd dev/eval/traps && go install ../../../cmd/engram && python3 gate.py --tier smoke | tee gate_after.txt
```
  Expected: **GREEN** — the default deep path is unchanged, so every axis must hold. If any axis regresses, the
  Modes insertion perturbed the deep path → fix before proceeding.
- [ ] **Step 5 — Record** the result as a labeled table, filling cells from `baseline_before.txt` / `gate_after.txt`
  (no fabricated numbers — copy the gate's actual per-axis verdicts and bars):
```markdown
| Axis (smoke bars)            | Before (current skill) | After (glance/deep skill) | Δ        |
|------------------------------|------------------------|---------------------------|----------|
| C3  — apply conventions      | <fill GREEN n/n>       | <fill>                    | <hold?>  |
| C4i — recency supersession   | <fill>                 | <fill>                    | <hold>   |
| C5  — honor recency standard | <fill>                 | <fill>                    | <hold>   |
| C6  — abduction/synthesis    | <fill>                 | <fill>                    | <hold>   |
```
  Pass = every axis HOLDs before→after (no regression). This table goes into the #662 close-out comment.

## Task 4: Doc scrub + the new "atoms" roadmap item

**Files:**
- Modify: `docs/GLOSSARY.md` (add a glance/deep entry under `## Recall`, near line 105–114)
- Modify: `docs/architecture/c1-system-context.md` — **one sentence in the `### Flow: recall` intro (line ~70) only**
  (the line-262 sequence note is the *please* flow, NOT a target)
- Modify: `docs/ROADMAP.md` (mark #662 done; add the atoms-restructure item beside "Deeper arc — relational synthesis")
- Check: `README.md` (only if it documents recall modes)

- [ ] **Step 1 — GLOSSARY.** Under `## Recall`, add:
```markdown
### recall modes — `glance` / `deep`
Recall's two rungs (the depth dial, #662). `deep` (default) = the full procedure incl. the write side
(crystallize/link/persist). `glance` (opt-in) = read-only w.r.t. vault knowledge: ~3 phrases, keep Step
2.5A/2.5B/2.7/Step-3, drop the write side; *applies* memory without growing the vault. Glance escalates to
`deep` for recency-channel standards (C5 — glance surfaces but doesn't elevate them; #661).
```
- [ ] **Step 2 — c1 diagram.** In `### Flow: recall` (line ~70), after the opening paragraph, add exactly:
  *"Recall has two rungs: the default `deep` (the full procedure shown here, with write-side crystallization) and
  an opt-in `glance` rung (read-only, ~3 phrases, no vault-knowledge writes — cheaper per fire but it escalates to
  `deep` for recency-channel standards). The sequence shown is `deep`."* Do not redraw the diagram.
- [ ] **Step 3 — ROADMAP (#662 done).** In the "Recall depth dial" entry (line ~118), mark **#662 ✅ SHIPPED —
  glance/deep modes (deep default), C5→deep escalation; O2/L2 confirmed; gate GREEN**; note #663 (guidance) is the
  remaining item and the deeper C5 recency-apply *fix* (lift both rungs above 4/5) is a separate follow-up.
- [ ] **Step 4 — ROADMAP (atoms item, Part 3).** Add as a sibling of the existing `### Deeper arc — relational
  synthesis (note 68)` subsection (Track A, line ~90), so the two long-arc items sit together:
```markdown
### Deeper arc — rebuild the skills from behavioral atoms  [ARCHITECTURE — Joe 2026-06-29]
The skills (recall, learn, please, route) overlap; the underlying *atoms* are distinct behaviors —
**read-memory, write-memory, route-a-task, orchestrate-a-workflow (reason + adversarial-check)**. Decompose the
skills into atoms dedicated to each behavior and recompose, **without ending up with N skills that almost all do
the same thing** (Joe's explicit constraint). The glance/deep read-vs-write split (#662) is a first, small
instance of the seam this would generalize. Scope/sequence TBD — brainstorm before any build.
```
- [ ] **Step 5 — README check.** `grep -n -iE "glance|recall mode" README.md`. If it matches: replace the matched
  lines with a one-line pointer (*"Recall modes (`glance`/`deep`) are documented in `docs/GLOSSARY.md`."*). If no
  match: record "README N/A — does not document recall modes" in the commit body.
- [ ] **Step 6 — orchestrator Gate C** over every touched doc (relevance + clarity/cohesion).
- [ ] **Step 7 — Commit** the doc changes.

## Task 5: Close out #662

- [ ] **Step 1 — Comment #662** (then close): glance/deep modes shipped (deep default, Joe's call); C5→deep
  escalation is the concrete fix for #661's C5 gap; O2/L2 confirmed landed; gate GREEN (paste the Task-3 table).
  Note the issue's **C7 pass-bar was not used — C7 is a paper gate** (stubs the binary, vault note 142); gated on
  `gate.py` instead. Flag the two follow-ups: the deeper C5 recency-apply fix, and the atoms restructure (now on the ROADMAP).
- [ ] **Step 2 — Delete** the throwaway `scratchpad/glance_red.sh` and any temp vaults it made.
- [ ] **Step 3 — orchestrator Gate D** over the #662 comment + commit messages (clarity/standards; `AI-Used: [claude]` trailer).
- [ ] **Step 4 — Close #662** (`gh issue close 662`). #657 stays open (L3a/O1 remain).

## Self-review notes (writing-plans checklist)

- **Coverage:** Part 1 (stale memory) = Task 1 (notes 77+100+the two MEMORY.md notes+index); #662 build = Tasks
  2–3; doc scrub = Task 4; atoms item = Task 4 Step 4; close-out = Task 5. All ask elements mapped.
- **Scope honesty:** O2/L2 are confirm-only (already landed) — not re-built (Iron Law). The C5 *apply-fix* and the
  *atoms restructure* are explicitly OUT of #662's build (flagged as follow-ups), matching #661's validated scope.
  The default-rung simplification vs the design is stated in the Architecture section.
- **Gate correctness:** gated on `gate.py`, never C7 (note 142). Deep-default keeps the gate's no-arg path
  unchanged, so smoke before+after suffices + #661 supplies the glance full-bars evidence.
