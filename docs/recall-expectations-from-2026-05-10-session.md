# Expected recall surfaces for a new LLM — from 2026-05-10 session

> An evaluative artifact: predicting what an optimally-functioning recall system + skill triggering would surface for a fresh-context LLM dropped into each work-state of this session, so we can compare against what *actually* surfaces.

**Vault state at time of writing:** 65 permanents, 6 MOCs, 0 fleetings (clean post-promotion). Latest IDs added today: 1c1, 1f, 1g, 4h, 4i, 9o1, 9r, 9s, 10c1, 14, 14a, 15, 16, 16a.

## How to read this document

For each work-state, the new LLM is dropped in cold (no prior conversation context). The "Expected surfaces" rows describe what the system *should* fire automatically — not what the LLM should ask for. Then "Skills that should fire" lists what skill the system should trigger or the LLM should reach for first. Finally, "Reality check" notes what *actually* fired in this session vs. what was missed.

A passing system surfaces enough that the LLM doesn't repeat this session's mistakes. A failing system leaves the LLM rediscovering what's already in the vault.

---

## State A — "Should we redesign / consolidate two adjacent systems?"

**Trigger conditions:** user is comparing two systems they own, asking "which should win" or "should we merge these"; conversation has flavor of system-redesign or paradigm-choice.

**Expected surfaces (situation-keyed):**
- [[1c.assumptions-audit-explicit-step]] — name the load-bearing assumption first
- [[1c1.premise-verify-before-commit]] — when the framing is contested, premise-inversion changes the answer
- [[5.llm-rationalization-patterns]] (MOC) — the family of failures where the LLM commits early and rationalizes toward it
- [[1f.distortion-test-before-new-type]] — before adding a "third option / extra type", verify existing options can't carry the case
- [[4f.vault-is-not-the-universal-sink]] — right-home discipline; not everything belongs in one system
- [[8.situation-as-continuous-query]] — recall is exploration of features, not just a literal query

**Skills that should fire:**
- `brainstorming` (before any creative redesign work)
- `recall` for the user's named topics

**Reality check this session:**
- ✅ I invoked `recall` at user's request
- ❌ The premise-verify lesson (now 1c1) didn't exist yet, so the system couldn't fire it; I committed to engram-survives, then user reframed and the analysis flipped. The new permanent should fire next time.
- ❌ The distortion-test discipline (now 1f) didn't exist; I invented "principle" template before user corrected. Next time the system can prevent it.

---

## State B — "Iteratively designing a structured artifact format (note shape, schema, response template)"

**Trigger conditions:** user is working through what fields belong in a structured artifact; iterative editing of a worked example; questions about minimality.

**Expected surfaces:**
- [[14.drop-by-example-finds-minimal]] — the canonical method: take a worked example and iteratively drop content
- [[4i.structure-mechanical-relationships-prose]] — split between mechanical generation and prose-shaped content
- [[4h.moc-body-is-framing]] — let the graph tool carry membership; don't restate
- [[4a.no-tags-no-related-lists]] — don't duplicate what the link graph encodes
- [[9s.first-use-catches-naming]] — don't lock final names until first real use

**Skills that should fire:**
- `brainstorming` if the design space is genuinely new
- `writing-skills` if the artifact is a SKILL.md

**Reality check this session:**
- This work happened pre-promotion; none of these notes existed yet. The iterative-drop method was rediscovered organically through my own resistance + user pushback. Next session, the system should fire 14 the moment this pattern starts.

---

## State C — "Writing an implementation plan for a multi-file feature"

**Trigger conditions:** user invoked `writing-plans`; spec is settled; need to break work into bite-sized TDD tasks.

**Expected surfaces:**
- [[9m.plan-code-blocks-match-project-conventions]] — sample sibling files before writing plan code blocks; generic Go idioms become lint debt
- [[9q.cheap-checks-defer-expensive-debt]] — schedule `targ check-full` per task, not just `targ test`
- [[9o.holistic-final-review-catches-blind-spots]] — include a final cross-cutting review checkpoint
- [[9p.smoke-test-binary-not-unit-runX]] — include an end-to-end smoke task that runs the actual binary
- [[9o1.cross-cutting-finds-asymmetry]] — explicit instruction for the final review to look across functions for asymmetry
- [[15.tracked-concerns-via-issues]] — plan should anticipate that reviewers will surface non-spec concerns; carry them forward, file as issues post-completion

**Skills that should fire:**
- `writing-plans` (which I used)

**Reality check this session:**
- ✅ The plan included a final cross-cutting review and a smoke test (Tasks 9, 11)
- ❌ Plan code blocks used generic Go conventions; subagents had to fix lint debt across multiple tasks (9m would have fired this)
- ❌ I didn't preemptively schedule `targ check-full` per task; the implementers ran into the same debt-deferral pattern (9q would have fired this)
- ❌ I encoded the cross-cutting review as a checklist item but not the specific instruction to look for cross-function asymmetry (9o1 would have fired this)

---

## State D — "Executing a plan via subagent-driven dispatch"

**Trigger conditions:** invoked `subagent-driven-development`; sequence of fresh-context implementer subagents per task with two-stage review.

**Expected surfaces:**
- [[10c.implementer-reviewer-rhythm]] — the rhythm itself
- [[10c1.never-chase-lsp-post-commit]] — LSP fires stale diagnostics after every commit; verify via build tool
- [[10d.subagent-scope-defaults-to-task-local]] — subagents won't refactor outside their task without explicit license
- [[9c.red-shows-where-not-to-spend]] — RED carries information beyond "test fails"
- [[9r.red-audits-spec]] — RED can catch spec bugs; surface them via DONE_WITH_CONCERNS
- [[1g.ignored-own-concurrency-design]] — when using a freshly-built feature, audit whether your default invocation actually uses what the feature enables

**Skills that should fire:**
- `subagent-driven-development` (which I used)
- `requesting-code-review` per task

**Reality check this session:**
- ✅ I had a memory rule about stale LSP — it fired correctly across 7+ instances. Without it I'd have wasted significant time.
- ❌ I went sequential on the 10 promotions despite having implemented flock specifically to enable parallel — user had to remind me. The new permanent 1g now captures this.
- ✅ Subagent at Task 3 caught the spec's depth%2 inversion in RED and flagged it as a deviation — exactly the 9r pattern.

---

## State E — "Reviewing code per task: spec compliance then quality"

**Trigger conditions:** subagent reports DONE; controller dispatches review pair.

**Expected surfaces:**
- [[9o.holistic-final-review-catches-blind-spots]] — final pass at higher altitude catches blind spots
- [[9o1.cross-cutting-finds-asymmetry]] — specifically: cross-function asymmetries are invisible at per-task review level
- [[9p.smoke-test-binary-not-unit-runX]] — run the binary, don't just unit-test the runX
- [[15.tracked-concerns-via-issues]] — non-spec concerns get DONE_WITH_CONCERNS → carry-forward → post-completion issue filing

**Skills that should fire:**
- `requesting-code-review`
- `receiving-code-review` from implementer side

**Reality check this session:**
- ✅ Two-stage review caught issues per-task (e.g., implementer's depth%2 fix, the SiblingOfTopLevel inconsistency I'd embedded in the plan)
- ✅ Final cross-cutting review caught the MOC trailing-newline asymmetry that per-task reviews missed — exactly 9o1's case study

---

## State F — "First real use of a freshly-built tool"

**Trigger conditions:** previously implemented & merged a feature; now invoking it on real data for the first time.

**Expected surfaces:**
- [[9s.first-use-catches-naming]] — the first usage session is a naming review; flaws invisible during design surface here
- [[9p.smoke-test-binary-not-unit-runX]] — sibling pattern; real-world usage > inspection
- [[1g.ignored-own-concurrency-design]] — audit whether your default usage exercises the features you built
- [[16.name-the-op-not-workflow]] — if names feel awkward in non-canonical use cases, you named the workflow not the operation

**Skills that should fire:**
- None obvious; this is a usage moment, not a process moment

**Reality check this session:**
- ❌ I forgot the flock and ran sequential (1g now captures)
- ❌ The naming mismatch was invisible across 12 review-checkpointed tasks; only first-use surfaced it (9s now captures)
- → This state has the highest density of "lessons rediscovered" in this session. The system should fire most aggressively here.

---

## State G — "Designing CLI command names or skill names"

**Trigger conditions:** naming a new tool or renaming an existing one; brainstorming command/flag names.

**Expected surfaces:**
- [[16.name-the-op-not-workflow]] — name the operation, not the workflow that wraps it
- [[16a.optional-flag-bundles-coupled-ops]] — pattern for keeping a generally-named operation usable for its common workflow via optional flags
- [[9s.first-use-catches-naming]] — defer final names until first-use validation
- [[12.skill-authoring-craft]] (MOC) — the broader cluster

**Skills that should fire:**
- None automatic; should surface during brainstorming

**Reality check this session:**
- This work happened pre-promotion of 16/16a/9s; they didn't exist yet. Next session, the system should fire all three when the user starts naming.

---

## State H — "Capturing review concerns as tracked work"

**Trigger conditions:** completing a feature; reviews surfaced concerns not in spec.

**Expected surfaces:**
- [[15.tracked-concerns-via-issues]] — the canonical pattern: DONE_WITH_CONCERNS → controller carry-forward → post-completion issue filing
- [[9o.holistic-final-review-catches-blind-spots]] — adjacent: where new concerns come from

**Skills that should fire:**
- `issue:issue-file` per concern

**Reality check this session:**
- ✅ I followed the pattern exactly: forward-concerns flagged in implementer reports, carried in subsequent task briefings, filed as #607–#611 at end. The new permanent 15 captures this so the system can fire it next time.

---

## Cross-cutting expectations — ALWAYS surface

A few notes are situation-agnostic enough that they should fire near the start of every session that involves writing or persisting anything:

- [[4g.want-to-remember-invoke-persistence]] — "I'll remember" is a no-op; capture or write it
- [[4f.vault-is-not-the-universal-sink]] — right-home check (CLAUDE.md? skill? code? memory?)
- [[1.subagent-driven-recovery]] — refusing to delegate is rationalizing context-greed
- [[5.llm-rationalization-patterns]] (MOC) — the umbrella

**Reality check:** these did surface via the early `/recall` invocation. The recall skill itself worked.

---

## What's MISSING from the vault that this session should produce

A few patterns from this session don't have a dedicated permanent yet, suggesting the next promotion pass should consider:

1. **"Use the feature you built" as a more general principle.** 1g captures the specific case (flock); the general case is broader — every feature you build has a slightly-uphill default-cost; you have to consciously choose it. Could become a sibling of 1g or a continuation.

2. **"Plan-quality cascades through subagent dispatch."** Tasks 8, 11 had spec ambiguities that subagents flagged. The pattern: a plan with a wrong assertion produces a chain of confused subagents. Plan-review should explicitly check for "would a subagent treating this verbatim get confused?" Not yet captured.

3. **"Engram-style structured memory + zettelkasten = the design we converged on."** This is the high-level architectural decision but it's spread across 4h, 4i, 14, 16. A MOC framing it as a synthesis is not yet written. When the cluster reaches 5+ permanents (it's at 4 today), an MOC may become writable.

---

## Summary table

| State | Notes that should fire | Did it? |
|---|---|---|
| A. Redesigning two systems | 1c, 1c1, 5, 1f, 4f, 8 | Partial — 1c1, 1f didn't exist yet |
| B. Iterative artifact design | 14, 4i, 4h, 4a, 9s | None existed yet |
| C. Writing implementation plan | 9m, 9q, 9o, 9p, 9o1, 15 | Some encoded; 9m, 9q, 15 missed |
| D. Subagent-driven execution | 10c, 10c1, 10d, 9c, 9r, 1g | 10c-shape worked; 1g would have helped |
| E. Per-task code review | 9o, 9o1, 9p, 15 | All worked correctly |
| F. First real use of tool | 9s, 9p, 1g, 16 | None existed; this state had highest miss-density |
| G. CLI/API naming | 16, 16a, 9s, 12 | None existed; surfaced via post-hoc retrospective |
| H. Review concerns → issues | 15, 9o | 15-shape worked even before the note existed |

---

## What "optimal" looks like next time

For a new LLM working on a similar feature:

1. **Pre-plan recall** surfaces State C notes (9m, 9q, 9o, 9p) so the plan includes those checkpoints from the start.
2. **Pre-execution recall** surfaces State D notes (10c, 10c1, 1g, 9r) so the orchestrator pattern is primed and stale-LSP doesn't waste cycles.
3. **Pre-review recall** surfaces State E notes (9o, 9o1, 9p, 15) so reviewers know what classes of issue to watch for.
4. **Pre-first-use recall** surfaces State F notes (9s, 9p, 1g) — *this is the highest-miss state* — so a new LLM doesn't repeat my "implemented flock then forgot to use it" mistake.
5. **Pre-naming recall** surfaces State G notes (16, 16a, 9s) so naming is deferred to first-use.

If those five surface points fire reliably, the LLM avoids most of this session's rediscovery work. The vault has the lessons; the test is whether retrieval surfaces them at the right moment.
