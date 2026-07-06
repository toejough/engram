# Evidence-Based Route Rubric Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **Every task that edits `skills/route/SKILL.md` MUST go through `superpowers:writing-skills` TDD (RED baseline → GREEN edit → pressure tests) — CLAUDE.md mandates it; vault note 26.**

**Goal:** Turn the `route` skill's tier-selection from a hard-coded task-character table into a memory-based rubric: default every unit to the cheapest/fastest tier, escalate on failure, record each dispatch's outcome, and let recalled evidence set the starting tier next time.

**Architecture:** A closed evidence loop expressed entirely as skill *behavior* (no binary change in this increment): **route records → `/learn` crystallizes tier-performance lessons → `/recall` surfaces them → route applies them.** The prescriptive task-character table demotes to a labeled *cold-start prior* (unproven hypotheses that recorded evidence overwrites). Adopts, from the LinkedIn post, the two mechanics that survive our harness-agnostic doctrine — spec-handoff quality (A) and fail→rewrite-spec→escalate (B) — plus a dispatch audit record (D, reframed as the evidence substrate). All tier language stays roster-relative (cheap/mid/deep); no model names as rules.

**Tech Stack:** Markdown skill text (`skills/route/SKILL.md`); `superpowers:writing-skills` for the TDD cycle; engram's existing `/learn` + `/recall` for the memory loop; `docs/` (FEATURES.md, architecture/adr.md) for documentation; `gh issue create` for deferred infra.

## Global Constraints

- **Harness-agnostic — no model names as rules.** Route by capability *tier* (cheap / mid / deep), roster-relative. Model names appear only as "current roster instantiation," never as a rule. (Vault note 136; verbatim from existing skill.)
- **Skill edits require `superpowers:writing-skills` TDD** — RED baseline showing the failure on the old text, GREEN edit, pressure tests against fresh rationalization loopholes, before the edit is complete. (CLAUDE.md; vault note 26.)
- **No bespoke infra this increment.** The evidence loop uses existing `/learn` + `/recall` machinery only. Structured ledgers / new `engram` subcommands are deferred to issues. (Vault notes 78, 80: don't build split infra unless the delegated part is a large share of op cost.)
- **Dispatch-record outcome comes from an actual review/gate verdict, never a subagent's self-report.** (Vault notes 148, 162: subagent reports — especially cost — confabulate; verify against ground truth.)
- **Model-tier routing is already SHIPPED** (vault note 136, commit `2bf959f4`); this plan *evolves* it (cheapest-first + evidence), it does not introduce it. Do not frame tier-routing as new. (Vault note 143.)
- **Line length under 120 chars; match the existing skill's prose voice and table style.**

---

## File Structure

- **`skills/route/SKILL.md`** (modify — the deliverable): rewrite the doctrine to cheapest-first + evidence-based, add spec-handoff and dispatch-record sections, refine the escalation rule, demote the rubric table to cold-start priors, add the evidence-loop section, update red flags.
- **`skills/route/tests/README.md`** (modify): update the baseline index to name the new locked behaviors (cheapest-first default, escalation ladder, dispatch record, evidence loop) and how each is RED-tested.
- **`skills/route/tests/evidence-rubric-RED-GREEN.md`** (create, **ephemeral**): the fresh RED/GREEN evidence for this cycle (writing-skills artifact). Per route's own `tests/README.md` convention, RED/GREEN files are transient — the durable lock is the **skill text itself** plus the tests index; `git log` recovers this file after cleanup (exactly as the deleted `memory-discount-RED-GREEN.md` is). No LEDGER row: this is a behavioral skill change, not a measured eval claim.
- **`docs/FEATURES.md`** (modify): update the "Memory tier discount (route)" section to describe the broader evidence-based-rubric behavior; fix/point the ADR + validation references.
- **`docs/architecture/adr.md`** (modify): add an ADR recording the decision to make the route rubric evidence-based/cheapest-first (FEATURES.md already references ADR-000x for routing; keep the chain honest).
- **`CLAUDE.md`** (modify only if it describes route's doctrine — verify first; likely just the one-line skill summary, which may need a touch).
- Deferred (issues, not files): structured routing-evidence ledger; periodic rubric-refit; parallel-builders pattern (post idea C, parked); harness cost/duration telemetry capture.

---

## Task 1: RED baseline — capture the old hard-coded-rubric behavior

**Files:**
- Create: `skills/route/tests/evidence-rubric-RED-GREEN.md`
- Reference (read only): `skills/route/SKILL.md`

**Interfaces:**
- Produces: a written RED scenario + observed old-text behavior that Task 3's GREEN must flip. No code interface.

- [ ] **Step 1: Invoke `superpowers:writing-skills`** and follow its RED-first discipline. Announce it.

- [ ] **Step 2: Author the RED scenario.** A batch of ~6 dispatch decisions where the OLD skill's task-character table forces an a-priori tier by *surface look* (e.g. "cross-cutting refactor → deep tier", "code review with context → mid tier") with **no recalled evidence and no attempt to start cheaper**. The failure the RED must expose, in the worker's own reasoning against the *current* text:
  - (a) it picks mid/deep tiers from the table's task-character label without first defaulting to cheapest;
  - (b) it has no step that *records* the dispatch outcome as evidence;
  - (c) it treats the table's tiers as prescriptions, not as overwritable priors.

- [ ] **Step 3: Run the RED baseline** by having a fresh-context agent route the ~6 units using ONLY the current `skills/route/SKILL.md`. Capture its tier picks and reasoning verbatim into `evidence-rubric-RED-GREEN.md` under a `## RED (old text)` heading.
  Expected: the agent over-provisions (picks mid/deep for surface-hard units) and never records outcome-evidence — the documented failure mode.

- [ ] **Step 4: Commit the RED artifact.**
```bash
git add skills/route/tests/evidence-rubric-RED-GREEN.md
git commit -m "test(route): RED baseline — old table over-provisions, no evidence loop"
```

---

## Task 2: Draft the new skill text (no write yet — draft in the plan-execution scratch)

**Files:**
- Reference: `skills/route/SKILL.md` (current)

**Interfaces:**
- Produces: the exact replacement prose for each section, ready for Task 3 to write. The canonical target content is specified below so the executor types it verbatim (adjusting only to match final voice).

The new `skills/route/SKILL.md` has these sections. **Keep** the frontmatter, the "Orchestration work vs object-level work" section, AND the "Two rules every dispatch obeys" section (2g) unchanged (they are correct and orthogonal). Replace the doctrine and the rubric table with the following.

### 2a — Opening doctrine (replace the intro paragraph)

> # Route — default to the cheapest tier, escalate on evidence, remember what works
>
> You are an orchestrator. You route, decompose, and synthesize; you do not do object-level work
> yourself. There is no inline escape — easy work is delegated to a cheap model, not skipped.
>
> **The rubric is memory, not a hard-coded table.** Every unit starts at the cheapest / fastest
> available tier. The *only* thing that raises the starting tier is **recalled evidence** that this
> kind of work has failed cheaper before. When a dispatch fails, you fix the spec and — if it fails
> again — escalate one tier. Every dispatch is recorded, and those records are what `/learn`
> crystallizes and `/recall` surfaces, so the starting tier for similar work reflects real evidence
> next time. Cold-start is cheapest-for-everything; as evidence accrues, the *effective* rubric warms
> up on its own — via recall, not by editing this file.

### 2b — Resolution rule (replace "Resolution:" and fold in escalation)

> ## How to pick a tier
>
> 1. **Recall first (you, the orchestrator).** Before dispatching, check recalled memory for
>    tier-performance evidence on *this kind of work* ("cheap tier sufficed for X" / "cheap failed on
>    Y — needs mid"). Recalled evidence sets the starting tier.
> 2. **Absent evidence, default to the cheapest / fastest tier.** Do not upgrade on a surface-look
>    hunch ("this looks hard"); an unproven guess is exactly what the evidence loop replaces. The
>    cold-start priors below are hypotheses, not prescriptions.
> 3. **Escalate on failure, spec-first:**
>    - **First fail** → the failure is usually a *spec* failure, not a model failure. Rewrite the
>      handoff (sharper files, acceptance checks, tighter do-NOT-touch), retry the **same** tier. The
>      builder never gets to guess twice off the same spec.
>    - **Second fail on the same unit** → escalate **one** tier and retry.
>    - Repeat until it passes or you reach the deep tier.
> 4. **Memory discounts the tier (a special case of evidence lowering it).** A unit whose needed
>    knowledge is recallable — a known convention, prior decision, crystallized diagnostic — drops one
>    tier (floored at cheap), because the model *applies* recalled knowledge instead of *deriving* it.
>    (Measured 2026-06-28 at the deep→mid boundary — vault note 135.)

### 2c — Spec-handoff quality (NEW section — post idea A)

> ## The handoff is the unlock
>
> A cheap-tier model matches an expensive one when the spec is exact; when it fails, suspect the spec
> before the model. Every dispatch MUST hand the subagent:
>
> - **Exact files and paths** to create/modify (not "the auth code" — `internal/foo/bar.go:20-45`).
> - **Acceptance checks** — the concrete, verifiable conditions that mean "done" (a command + its
>   expected output; a test that must pass; the property that must hold).
> - **Explicit do-NOT-touch bounds** — files, interfaces, and behaviors the unit must leave alone.
> - **The recall-first instruction** (see rules below).
>
> Vague handoffs are why cheap tiers "fail." Fix the handoff first.

### 2d — Dispatch record (NEW section — post idea D, reframed)

> ## Record every dispatch (the evidence)
>
> After each dispatch resolves, record one line of evidence. Build it from what you already know as
> orchestrator plus the review's verdict. No privileged telemetry needed — this keeps the loop
> harness-agnostic. A **work-kind** is your classification of the unit's shape and concept ("single-file
> refactor", "cross-cutting lint", "API integration") — kept consistent enough that the same kind
> reuses prior dispatches' evidence.
>
> | field | source |
> | --- | --- |
> | **work-kind** | your classification of the unit (see above) |
> | **tier used** | your routing decision (cheap / mid / deep) |
> | **model (roster @ dispatch)** | the concrete model the tier resolved to *at dispatch time* (e.g. `cheap (haiku)`). Provenance, not a routing rule — it lets a later audit ask "did swapping the cheap model change the failure rate on this work-kind?" |
> | **why** | source of the tier choice: "recalled evidence (kind K passed at tier T)", "memory-discount applied", or "cold-start default" |
> | **outcome** | **the review/gate verdict** — PASS/FAIL. Never the subagent's self-report (it confabulates — vault notes 148, 162). |
> | **escalation** | if it failed and escalated, the tier it finally passed at |
> | **duration** | observed wall-clock, best-effort |
> | **cost** | only if the harness exposes it; mark "n/a" otherwise — never invent it |
>
> Produce this record per-dispatch and collect the rows into a compact table in your user-facing
> report (the mini-report). The table is the audit trail for route's own tier guidance over time.

### 2e — The evidence loop (NEW section)

> ## The loop that improves the rubric
>
> 1. **Route** picks a tier (cheapest-first, or evidence-raised).
> 2. **Record** the dispatch outcome (above).
> 3. **`/learn`** crystallizes a tier-performance lesson when an outcome is a genuine signal — "cheap
>    tier sufficed for work-kind K" or "cheap failed N× on K, needs mid." One note per distinct
>    work-kind/tier finding.
> 4. **`/recall`** surfaces that lesson next time similar work is routed.
> 5. The **starting tier** now reflects evidence — the rubric improved without editing this file.
>
> Periodic consolidation: when strong repeated evidence accrues for a work-kind, fold it into the
> cold-start priors below via `superpowers:writing-skills` TDD, so a cold session starts warm. Until
> then, recall does the improving.

### 2f — Cold-start priors (REPLACE the prescriptive rubric table)

> ## Cold-start priors (unproven — evidence overwrites these)
>
> Absent recalled evidence, these are the *starting* tiers. Every entry is a hypothesis the dispatch
> record is meant to confirm or overturn — not a prescription. The posture is **cheapest for
> everything**; the only entry that starts above cheap must earn it with recorded evidence, and none
> does yet. There is **no cold-start exception for "looks hard"** — a surface-look guess is exactly
> what the loop replaces; genuinely deep work reaches the deep tier via two failed escalations, and
> the evidence loop then learns to start it there.
>
> | Work character | Cold-start tier | Status |
> | --- | --- | --- |
> | Everything, by default | cheap | the posture |
> | Memory-backed unit (answer is recallable) | one tier down, floored at cheap | **evidence-backed** (note 135) |
>
> Current roster: cheap = haiku, mid = sonnet, deep = opus. A new model re-fills a tier without
> changing this table.

### 2g — Two rules every dispatch obeys (KEEP, unchanged: recall-first + decompose-before-dispatch)

### 2h — Red flags (UPDATE)

> | Sign you're off | What to do |
> | --- | --- |
> | You picked mid/deep because the unit "looks hard" | Default cheapest; only recalled evidence raises the start. A surface hunch is the guess the loop replaces. |
> | You escalated a tier on the first failure | First fail → rewrite the spec, retry same tier. Escalate only on the second fail. |
> | You dispatched without exact files, acceptance checks, and do-NOT-touch bounds | Fix the handoff — a vague spec is why cheap tiers "fail." |
> | You skipped recording the dispatch outcome | The record IS the rubric; no record → no evidence → the rubric never improves. |
> | You wrote the outcome from the subagent's "done" claim | Outcome = the review verdict, a ground-truth artifact. Subagent reports confabulate (notes 148, 162). |
> | You invented a cost/duration number | Duration is best-effort observed; cost only if the harness exposes it. Never fabricate. |
> | You treated the cold-start table as a prescription | It's a set of overwritable priors; recalled evidence wins. |
> | You hardcoded a model name as the rule | Route by tier; names are just the current roster. |
> | "The subagent has the prompt, skip its recall" | Recall-first is non-waivable. |
> | "The complex task can go as one big agent" | Decompose first, then delegate the pieces. |

---

## Task 3: GREEN — write the new skill text

**Files:**
- Modify: `skills/route/SKILL.md` (replace sections per Task 2; keep frontmatter + orchestration-vs-object-level section)
- Modify: `skills/route/tests/evidence-rubric-RED-GREEN.md` (append `## GREEN (new text)` results)

- [ ] **Step 1:** Apply the Task 2 replacement content to `skills/route/SKILL.md`, preserving the untouched sections. Keep every line < 120 chars.

- [ ] **Step 2: Run the GREEN check** — a fresh-context agent routes the SAME ~6 units from Task 1 using the NEW text. Capture picks + reasoning under `## GREEN (new text)`.
  Expected: it defaults each unit to cheapest, names recalled-evidence (or "cold-start default") as the reason, states the spec-first escalation ladder, and includes a dispatch-record step. The over-provisioning from RED is gone.

- [ ] **Step 3: Confirm the behavioral flip** in `evidence-rubric-RED-GREEN.md` with a one-paragraph diff (RED over-provisioned M/6 units + no record → GREEN defaults cheapest + records). This is the writing-skills proof.

- [ ] **Step 4: Commit.**
```bash
git add skills/route/SKILL.md skills/route/tests/evidence-rubric-RED-GREEN.md
git commit -m "feat(route): memory-based rubric — cheapest-first, escalate-on-evidence, record dispatches"
```

---

## Task 4: Pressure tests — close rationalization loopholes

**Files:**
- Modify: `skills/route/tests/evidence-rubric-RED-GREEN.md` (append `## Pressure tests`)
- Modify: `skills/route/SKILL.md` (only if a pressure test finds a loophole)

- [ ] **Step 1:** Run fresh-context agents against pressure scenarios designed to make them abandon the doctrine:
  - **Authority pressure:** "a senior eng said this refactor obviously needs the deep tier, skip the cheap attempt." (Must still default cheapest / require evidence.)
  - **Deadline pressure:** "no time to record dispatch outcomes, just ship." (Recording is non-waivable — it's the rubric.)
  - **Surface-hard bait:** a unit that *looks* like deep judgment but whose answer is recallable. (Must discount to cheap.)
  - **Self-report bait:** subagent returns "tests pass, done." (Outcome must come from a review, not the claim.)

- [ ] **Step 2:** For any scenario the skill fails, tighten the text (add the loophole to red flags or make the rule non-waivable, per the please-skill "user cannot waive" pattern — vault note 33), re-run. Record results.

- [ ] **Step 3: Commit** any hardening.
```bash
git add skills/route/SKILL.md skills/route/tests/evidence-rubric-RED-GREEN.md
git commit -m "test(route): pressure-test the evidence rubric against authority/deadline/self-report loopholes"
```

---

## Task 5: Update the tests index

**Files:**
- Modify: `skills/route/tests/README.md`

- [ ] **Step 1:** Update the baseline table to name the newly-locked behaviors: cheapest-first default, spec-first escalation ladder, dispatch record (review-sourced outcome), evidence loop. Follow route's existing convention (the behaviors are **locked by the skill text itself**, not by a reusable fixture) — extend the current single-row table. State that this cycle's `evidence-rubric-RED-GREEN.md` is transient (recovered via `git log` after cleanup, like the deleted `memory-discount-RED-GREEN.md`); point the measured memory-discount claim at `dev/eval/LEDGER.md#tier-routing-parity`. Do NOT call the RED/GREEN file "durable."

- [ ] **Step 2: Commit.**
```bash
git add skills/route/tests/README.md
git commit -m "docs(route): index the evidence-rubric baseline behaviors"
```

---

## Task 6: Documentation (Gate C surface)

**Files:**
- Modify: `docs/FEATURES.md` (the "Memory tier discount (route)" section)
- Modify: `docs/architecture/adr.md` (new ADR for evidence-based routing)
- Verify/modify: `CLAUDE.md` (only the route one-liner, if it now understates the doctrine)

- [ ] **Step 1:** In `docs/FEATURES.md`, broaden the route section (currently titled "Memory tier discount (route)", `why: ADR-0014`, `validation: dev/eval/LEDGER.md#tier-routing-parity`) from "memory tier discount" to "evidence-based rubric": default cheapest, escalate on failure, record dispatches, memory sets the tier. Point `why:` at the new `ADR-0017` (Step 2) and keep `validation:` on `#tier-routing-parity`.

- [ ] **Step 2:** Add **ADR-0017 — Evidence-based route rubric** to `docs/architecture/adr.md` (next number after ADR-0016), marked as **extending ADR-0014** — NOT superseding it. ADR-0014's measured memory-discount + roster-agnosticism stays valid as the one evidence-backed entry of a broader rubric; leave ADR-0014's Status line untouched (the repo reserves "Superseded" for replaced/deleted decisions, and this doesn't replace 0014). ADR-0017 text: state "Extends ADR-0014 (memory-backed tier discount + roster-agnosticism) by embedding it in a broader evidence-based framework." *decision* — the route rubric is memory-based (cheapest-first + escalate-on-evidence + dispatch record) rather than a hard-coded task-character table; *why* — the old mid/deep tiers were asserted, not measured (only the deep→mid memory-discount boundary was measured — LEDGER `#tier-routing-parity`); the record→learn→recall loop makes routing self-correcting and harness-agnostic; *consequences* — needs a dispatch record and the learn/recall loop; deferred infra tracked in issues. Follow the exact ADR heading/format of ADR-0014..0016 in that file.

- [ ] **Step 3:** Grep `CLAUDE.md` and `docs/README.md` for route descriptions; update only if they now misstate the doctrine. Read first, edit minimally.

- [ ] **Step 4: Commit.**
```bash
git add docs/FEATURES.md docs/architecture/adr.md CLAUDE.md docs/README.md
git commit -m "docs: route rubric is evidence-based (ADR + FEATURES + skill summary)"
```

---

## Task 7: File deferred-infra issues

**Files:** none (uses `gh issue create` — vault: engram tracks issues via GitHub, not a local file)

- [ ] **Step 1:** File issues, each with context + acceptance criteria:
  - **Structured routing-evidence ledger** — an append-only record of dispatch outcomes + an `engram` query so route can ask "how has cheap tier done on work-kind K?" directly, instead of relying on free-text recall. (Increment-2 robustness for the loop.)
  - **Periodic route-rubric-refit** — a flow (analogous to vocab refit) that folds strong accrued evidence into the skill's cold-start priors via writing-skills TDD.
  - **Parallel-builders pattern (post idea C, parked)** — for ship-critical units, dispatch two builders in parallel and have a reviewer pick the winner; evaluate whether it earns its cost harness-agnostically.
  - **Harness cost/duration telemetry capture** — if/when the harness exposes per-subagent cost/tokens to the orchestrator, wire it into the dispatch record's cost field.

- [ ] **Step 2:** Record the issue numbers in the closing report.

---

## Self-Review

**Spec coverage:**
- Post idea A (spec-handoff quality) → Task 2c / Task 3.
- Post idea B (fail→rewrite-spec→escalate) → Task 2b / Task 3.
- Post idea D reframed (dispatch audit record) → Task 2d / Task 3.
- Post idea C (parallel builders) → **parked**, filed in Task 7 (not silently dropped — vault: deliver the full diverse set).
- User's core ask (memory-based rubric, cheapest-first, escalate, remember-to-improve) → Tasks 2a/2b/2e/2f (doctrine + loop + cold-start priors).
- Named roles → **not adopted** (user chose task-character framing); the table becomes cold-start priors instead.
- Harness-agnostic constraint → Global Constraints + tier-only language throughout; verified in red flags.
- writing-skills TDD → Tasks 1, 3, 4.
- Docs scrub → Task 6 (FEATURES, ADR, CLAUDE.md, README).

**Placeholder scan:** replacement prose is specified verbatim in Task 2; no TBD/TODO. RED/GREEN/pressure scenarios are concrete.

**Consistency:** "cold-start prior", "dispatch record", "evidence loop", "cheapest-first", "spec-first escalation" used consistently across tasks and the target text.

**Gate A resolutions folded in (2026-07-06):**
- *ask-alignment F1* — added a `model (roster @ dispatch)` provenance field to the dispatch record (2d), so longitudinal audits can attribute outcomes to concrete models even after a roster remap. It records a fact, not a routing rule — harness-agnosticism preserved.
- *ask-alignment F2* — **deleted the "irreducible deep judgment → deep" cold-start row** (2f). It was a hard-coded non-cheap default wearing a "prior" label, contradicting "everything defaults to cheapest" and the plan's own red flag. Cheapest-first + escalate + the evidence loop reach and learn the deep tier without it.
- *docs- & code-alignment* — ADR-0017 **extends** (not supersedes) ADR-0014; ADR-0014 Status untouched (Task 6 Step 2).
- *code-alignment F2/F3* — RED/GREEN file renamed `evidence-rubric-RED-GREEN.md` and treated as **transient** (locked-by-skill-text convention, git-recoverable), not "durable"; no LEDGER row (behavioral change, not a measured claim).
- *clarity/standards* — line-length fixes (2d/2e/2f), `why`-field relabel, `work-kind` defined in 2d, cold-start row-2 made self-contained.

**Confirmed clean by Gate A (stated so it isn't relitigated):** C parked correctly (not creep); no named roles; all file paths/anchors verified against the working tree (ADR-0016 is last, ADR-0017 correct; FEATURES/LEDGER anchors real; commit `2bf959f4` real); CLAUDE.md + GLOSSARY.md + docs/README.md survive unchanged (route one-liner still accurate).
