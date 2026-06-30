# Relax the decision-moment recall guidance's cost bar (#663) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans. This edits **global guidance**
> (`~/.claude/CLAUDE.md`), not a SKILL.md — so the writing-skills *Skill* does not apply, but its **headless
> RED/GREEN discipline does** (the precedent that shipped the current guidance — vault notes 138/139). Steps use
> `- [ ]`. Gate B/C/D markers are run by the `/please` orchestrator.

**Goal:** Point the three decision-moment cues at the cheap `/recall glance` rung (shipped in #662) and relax the
*cost* hesitation so it fires readily — while keeping the *value* gate (idiosyncratic-only) and the Gate-A
hardening intact.

**Architecture:** One surgical edit to the `~/.claude/CLAUDE.md` "Recall at the decision moments" section
(lines 25-33). The current filter "fire it only when you expect a vault-specific gotcha … each call costs real
minutes" *conflates* cost and value. #663 **separates** them: relax COST (glance is cheap → fire readily),
preserve VALUE (skip routine/non-idiosyncratic — memory is net-negative there even when cheap, notes 91/95/99).
Then scrub the ROADMAP and amend vault note 139 (its `sources` point at this section).

**Tech Stack:** Markdown guidance; the headless `claude -p` harness at
`docs/design/2026-06-29-recall-moments-revalidation-data/` (run_revalidation.sh + clean_scenarios.json);
`engram amend` (note 139).

## Global Constraints

- **Headless `claude -p`, fresh process, fictional domains — NEVER subagents** (subagents inherit session
  context and invalidate the control — note 138 / `feedback_headless_not_subagents…`).
- **Do NOT regress the guidance win-nucleus:** the explicit action-naming ("the action is recalling — not a
  substitute self-check" — note 137, 0/5→5/5), the 3 cues, the Gate-A hardening ("once, not per retry"; do NOT
  re-add the cut "before a final verdict" cue — note 139).
- **The value gate is load-bearing** (notes 91/95/99): relaxing cost must NOT extend firing to clearly
  non-idiosyncratic/routine decisions. That is the regression the value-gate test (Task 1 Step 5) guards.
- `~/.claude/CLAUDE.md` is a **global file edited in place** (NOT in the repo — no git commit for it). Only the
  ROADMAP scrub is a repo commit. Commit trailer `AI-Used: [claude]`. Use the **bare Luhmann id** for `engram
  amend --target` (per #664). `engram amend --behavior` is **feedback-only** — note 139 is a *fact*, so use
  `--relation` only.

---

## Task 1: Revise the guidance + headless RED/GREEN

**Files:**
- Modify (global, in place): `~/.claude/CLAUDE.md` lines 25-33.
- Use: `docs/design/2026-06-29-recall-moments-revalidation-data/{run_revalidation.sh, clean_scenarios.json}`.

**Exact GREEN content** — replace the current section (lines 25-33) with (jargon-free, in-scope escalation):

```markdown
## Recall at the decision moments, not only at the start

engram `/recall` surfaces vault memory you haven't loaded — most valuable right before a wrong call locks in.
The cheap **`/recall glance`** rung (read-only, no crystallization writes, ~3 phrases) makes this affordable —
fire it **readily** at the cues below, but only where memory plausibly helps: when you expect a **vault-specific
gotcha** (a prior decision, a hard-won lesson, a convention that bites). On routine, non-idiosyncratic decisions,
skip it — memory is net-negative there even when cheap. At these cues, **run `/recall glance` before you proceed**:

- **Before declaring work done** — **`/recall glance` first** (the action is recalling — not a substitute
  self-check or re-inspection), then verify; the vault names the gotchas you'd otherwise ship past.
- **After a failure you can't immediately explain** — **`/recall glance` once before you start guessing** (not on
  every retry) — a past lesson may name the cause.
- **Before you start building a new approach** — **`/recall glance` prior decisions and standards first**, while
  the path is still cheap to change.

Escalate to **`/recall deep`** when the decision is weighty or irreversible, or when `glance` flags a gap it
can't resolve.

These catch *application* gaps — the lesson existed, just unrecalled at the moment.
```

(The C5 recency-standard escalation lives **inside** the recall skill's Step-3 glance annotation — the global
guidance does not repeat it, avoiding undefined "recent-activity standard" jargon. Escalation here matches the
issue/design spec: weighty/irreversible + glance-flags-a-gap.)

- [ ] **Step 1 — Read + ground the harness.** Read `docs/design/2026-06-29-recall-moments-revalidation-data/run_revalidation.sh`
  and `clean_scenarios.json`. Confirmed structure: 5 fictional-domain cue scenarios CF1-CF5 (acme-billing Java /
  northwind-web TS; cues: before-declaring-done ×2, after-unexplained-failure ×2, before-building-new-approach ×1),
  each run via `claude -p` with a neutral "what is your next step? … STEP/ACTION two lines, no tools" wrapper.
  **The control variable is the GLOBAL `~/.claude/CLAUDE.md`** (clean_scenarios.json purpose: "physically remove
  the recall-moments section during the RED batch, restore it for GREEN") — NOT the project cwd. The runner's
  `BASE`/`red-proj`/`green-proj` are stale (a dead session); we drive the batches directly below.
  **Scoring:** fire = the ACTION line invokes `/recall` or `engram query` before the object-level step.
  **Rung (Step-4 criterion):** grep the ACTION line for the `glance` token (vs `deep`/bare `/recall`).
- [ ] **Step 2 — Back up + define scenarios.** `cp ~/.claude/CLAUDE.md ~/.claude/CLAUDE.md.663bak`. Use CF1-CF5
  (cue scenarios) for the flip test, and these 5 **routine, non-idiosyncratic** scenarios (no plausible vault
  gotcha) for the value-gate test:
  - **R1:** "You are in the acme-billing service. You need to rename a local variable `tmp` to `invoiceTotal` in one function. What is your next step?"
  - **R2:** "You are in northwind-web. You need to add a getter that returns the existing `this.name` field. What is your next step?"
  - **R3:** "You are in acme-billing. You need to reformat a JSON config file to 2-space indentation. What is your next step?"
  - **R4:** "You are in northwind-web. You need to add a `console.log` printing the current user id for debugging. What is your next step?"
  - **R5:** "You are in acme-billing. You need to bump a dependency's patch version in the build file. What is your next step?"
  Wrap each (CF and R) with the runner's PREFIX/SUFFIX (capable-agent + "respond ONLY STEP/ACTION, no tools").
- [ ] **Step 3 — RED batch (control: guidance stripped).** Delete the "## Recall at the decision moments" section
  (lines 25-33) from `~/.claude/CLAUDE.md`. Run CF1-CF5 via `claude -p` from a neutral cwd (e.g. `/tmp`). Expected:
  **~0/5 fire** (baseline — re-confirms note 139's clean RED). Record each ACTION line.
- [ ] **Step 4 — GREEN edit + flip test.** Write the **GREEN content above** into `~/.claude/CLAUDE.md` (lines 25
  onward). Re-run CF1-CF5. **Pass-bar: fires ≥4/5 AND the ACTION lines invoke `/recall glance`** (flip preserved
  — does not regress note 139's 4/5 — *and* the relaxation took: glance, not deep). Record ACTION lines + grep
  for `glance`.
- [ ] **Step 5 — Value-gate test (the regression guard).** With the GREEN guidance in place, run R1-R5.
  **Pass-bar: ≤1/5 fire** (the lowered cost bar must NOT induce recall on routine work — memory net-negative
  there, notes 91/95/99). Any fire must be defensible (the scenario had a hidden gotcha). Record ACTION lines.
- [ ] **Step 6 — REFACTOR if a bar fails (then Gate B).**
  - **Flip regressed (Step 4 <4/5):** the glance/value wording buried the action — restore action prominence
    (note 137). Concrete fix: lead each cue with the bare command, e.g. "**Before declaring work done, run
    `/recall glance` first** — recalling, not a self-check; then verify." Re-run Step 4.
  - **Value gate leaked (Step 5 >1/5):** strengthen the skip clause to lead the cost paragraph, e.g. "fire it
    readily at the cues below — **but only on idiosyncratic decisions; on routine work (renames, formatting,
    log lines) skip it** — memory is net-negative there even when cheap." Re-run Step 5.
  - The refactored guidance passes Gate B (design-fit) before the task is done.
- [ ] **Step 7 — Restore + tabulate.** Confirm `~/.claude/CLAUDE.md` final state = the GREEN guidance (the ship);
  `rm ~/.claude/CLAUDE.md.663bak`. Append a labeled results table to this plan file (rung = `none` when no fire):

  | arm | scenarios | fires/5 | rung | verdict |
  |---|---|---|---|---|
  | RED control (guidance stripped) | CF1-CF5 | _fill_ | none | baseline ~0/5 |
  | GREEN revised (cue/idiosyncratic) | CF1-CF5 | _fill_ | _glance?_ | flip preserved + glance used? |
  | GREEN revised (routine) | R1-R5 | _fill_ | _fill_ | value gate holds (≤1/5)? |

  Spend: ~15 `claude -p` runs ≈ $5-10 (no cap; report actual).

## Task 2: Doc + memory scrub (note 64)

**Files:**
- Modify: `docs/ROADMAP.md` — the Track-A "✅ SHIPPED — recall at the decision moments" entry (Finding: make
  clear the value-filter STILL applies, just at a lower cost threshold).
- Amend (CLI): vault note `139` (the bare Luhmann id; its `sources` point at the reworded CLAUDE.md section).

- [ ] **Step 1 — ROADMAP Track-A.** In the "✅ SHIPPED — recall at the decision moments" entry, replace the
  cost-filter clause — currently "each gated by a cost-filter ('fire only when you expect a vault-specific
  gotcha'), scoping firing to idiosyncratic unloaded content (the one regime where memory is a clean win — note
  99)" — with: *"each fires the cheap `/recall glance` rung (shipped #662): the **cost** bar is relaxed —
  fire readily — but the **value** filter is unchanged, scoping firing to idiosyncratic unloaded content (the one
  regime where memory is a clean win — note 99; net-negative on routine even when cheap). Cost-bar relaxed by
  #663 (2026-06-29), headless-validated."* (Surgical — do not rewrite the whole entry.)
- [ ] **Step 2 — ROADMAP Track-B (clean-up beyond the issue's literal scope — bookkeeping, not a regression
  guard).** In the depth-dial entry, mark "(3, #663)" as ✅ SHIPPED 2026-06-29 (guidance cost bar relaxed,
  value-gate preserved, headless-validated). The depth-dial arc (#661→#662→#663) is complete.
- [ ] **Step 3 — Amend note 139** (`--relation` only — 139 is a *fact*, `--behavior` is feedback-only; bare id
  per #664). Verify the flag first with `engram amend --help`, then:
```bash
engram amend --target "139" \
  --relation "140.2026-06-28.recall-depth-dial-relaxes-not-dissolves-overfire|update: #663 relaxed this guidance's COST bar — cues now fire the cheap /recall glance rung readily; the value gate (idiosyncratic-only) is unchanged"
```
  Note 139's `sources` (pointing at `~/.claude/CLAUDE.md#Recall at the decision moments`) stay valid — the
  section still exists, just with relaxed wording; the amend records the supersession, not a replacement. (If the
  exact 140 basename differs, resolve it from the Task-1 recall payload before running; do not invent it.)
- [ ] **Step 4 — Gate C** over the ROADMAP changes.

## Task 3: Close out #663

- [ ] **Step 1 — Commit** the ROADMAP changes (the `~/.claude/CLAUDE.md` edit is global, not committed). Scope
  `docs(663)`; trailer `AI-Used: [claude]`. Gate D over the message + the #663 comment first.
- [ ] **Step 2 — Comment + close #663**: guidance cost bar relaxed (glance fires readily, value gate preserved);
  the Task-1 results table (flip preserved + glance used; value gate holds); note 139 amended. Depth-dial arc
  (#661→#662→#663) complete.
- [ ] **Step 3 — Clean** any temp files (the `.663bak`, any scratch scenario files).

## Self-review (writing-plans checklist)
- **Coverage:** guidance edit + headless validation = Task 1; doc/memory scrub = Task 2; close-out = Task 3.
- **Scope honesty:** relaxes the COST bar only; the VALUE gate, action-naming, and Gate-A hardening are preserved
  (win-nucleus). C5 escalation stays inside the recall skill (not duplicated as guidance jargon). Track-B scrub
  is labeled clean-up beyond the issue's literal Track-A scope. The bounded-reach caveat (note 109) is recorded,
  not re-litigated (Joe chose #663 over payload-prune).
- **Validation correctness:** two headless bars — flip-preserved-with-glance (Step 4) AND value-gate-holds
  (Step 5) — via fresh `claude -p`, control = global-guidance-stripped, never subagents. Rung detected by
  grepping ACTION lines for `glance`.
