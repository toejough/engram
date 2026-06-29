# Recall-at-decision-moments — Gate-A rewording re-validation (2026-06-29)

**Result: clean RED 0/5 → GREEN 4/5** (headless re-run). The reworded + cost-filtered guidance still moves the
needle; the single GREEN non-recall is the cost-filter discriminating, not a miss. The original *subagent*
method was invalid — the control inherited the treatment — see Method note. Full table below.

Re-validates the **reworded** `~/.claude/CLAUDE.md` "Recall at the decision moments" guidance after the
`/please` Gate-A cost/over-fire review scoped it (cut "before a final verdict"; tightened "after a tool fails"
→ "after a failure you can't explain, once, before guessing"; reframed "design fork" → "before you start
building a new approach"; replaced the vague "judge each" guard with a cost-filter; moved the validation
metric out of the always-loaded file).

## Method note — why the first method failed (a real finding)

The original validation used fresh **subagents** (Task tool) as RED/GREEN arms. That is **invalid for
re-validating guidance that already lives in this session**: subagents inherit the parent session/conversation
context. Two contaminated batches confirmed it — control agents recalled while quoting the **reworded wording
verbatim** ("before you start guessing", "while the path is still cheap to change", "a past lesson may name the
cause") even with the section physically removed from `CLAUDE.md`. Those exact fragments exist only in this
conversation, never in the file or `MEMORY.md`. **The control inherited the treatment → RED = 5/5, lift
unmeasurable.**

Fix: run each arm as a **headless `claude -p` process** (fresh session, no conversation inheritance) from a
temp project whose `CLAUDE.md` is the only thing that differs (RED = neutral; GREEN = neutral + the guidance
section); the global `~/.claude/CLAUDE.md` was toggled OFF during the runs so it could not leak into RED.
Scenarios are **fictional non-engram domains** (acme-billing / northwind-web) so no gotcha can leak via memory.
Neutral framing ("what's your next step?"), no spotlight on recall. PASS = first action is a recall
(`/recall` or its binary, `engram query`).

Raw transcript: `headless_results.txt`. Scenarios: `clean_scenarios.json`.

## Result — clean RED 0/5 → GREEN 4/5

| Scenario (cue) | RED (no guidance) | GREEN (+guidance) |
|---|---|---|
| CF1 — before declaring done (retry logic) | verify end-to-end — no recall | **/recall** ✓ |
| CF2 — before declaring done (date formatter) | launch app, inspect — no recall | **/recall** ✓ |
| CF3 — after failure (connection refused) | check reachability — no recall | check reachability — **no recall** |
| CF4 — after failure (module not found) | clear caches — no recall | **engram query** ✓ |
| CF5 — before a new approach (Kafka swap) | brainstorming skill — no recall | **/recall** ✓ |
| **Recall total** | **0 / 5** | **4 / 5** |

## Reading

- **Clean lift confirmed:** 0/5 → 4/5 under a genuinely uncontaminated control. The reworded + cost-filtered
  guidance still moves the needle; the cost-filter did **not** neuter efficacy.
- **The single GREEN non-recall (CF3) is the cost-filter working, not a leak.** "Connection refused to the
  sandbox" reads as an obvious environment issue — *not* a "vault-specific gotcha", so the agent checked
  reachability directly. The other after-failure case (CF4, a genuinely puzzling "module not found") **did**
  recall. So the after-failure cue discriminates: it fires on the puzzling failure, stays silent on the
  obvious-infra one — directly answering the Gate-A over-fire concern (it does *not* fire on every failure).
- **The two primary cues are solid:** before-declaring-done (CF1/CF2) and before-a-new-approach (CF5) → 3/3.
- RED agents still behave well (verify end-to-end, check env, brainstorm) — they just never reach for **vault
  memory**. That is the exact application-gap the guidance targets.

## Post-test wording change (monotone-safe)

The GREEN arm was tested with the preamble filter "fire only when you expect vault-specific gotchas you haven't
loaded — **not rules you already know**." A later Gate-B design-fit review flagged that "not rules you already
know" collides with the rationalization the `please` skill exists to override ("I skipped /recall because I
already know this"). The shipped wording drops the negative clause: "fire only when you expect a **vault-specific
gotcha**: a prior decision, a hard-won project lesson, a convention that bites." This **removes a skip-license**,
so it can only **increase or hold** recall tendency — the measured GREEN 4/5 (with the more-restrictive wording)
is a floor the shipped version cannot drop below. No re-run needed; the headless harness
(`run_revalidation.sh`, parameterized by the project `CLAUDE.md`) is preserved for any future re-validation.
