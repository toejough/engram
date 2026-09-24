## Context

`route-skill-to-runbook`, `please-skill-to-runbook`, `curate-skill-to-runbook`, `recall-learn-writememory-to-runbook`, and the in-progress `learn-skill-to-runbook` establish the pattern: promote a validated runbook body via `engram learn runbook`, gate retirement on a shim-only eval within one trial of the skill row (with an explicit validity gate), verify retrieval in the real vault, confirm `shim.md` is imported, delete the `SKILL.md`, update every live reference.

`agent-instructions/skills/recall/SKILL.md` measured: 344 lines, 27,327 bytes of body. Sections: Overview (1,386 B), Modes — glance vs deep (1,916 B), The procedure (19,262 B, one unified 9-step sequence — Steps 0, 0.5, 1, 2, 2.5, 2.7, 3, 3.5, 4 — annotated inline with `[glance: …]` notes marking where the read-only rung differs), Red flags (28 rows, 4,206 bytes — the largest red-flags source converted in this session, exceeding `please`'s 22-row table).

Per-step byte sizes within "The procedure": Step 0 (530 B), Step 0.5 Sweep (673 B), Step 1 Phrase queries (1,438 B), Step 2 Unified query (3,647 B), Step 2.5 Lazy note synthesis (4,783 B, the largest single step), Step 2.7 Activation (801 B), Step 3 Closing synthesis (1,918 B), Step 3.5 Re-entry (2,194 B), Step 4 Persist (3,227 B).

The skill's own text states the mode split precisely: `glance` runs Steps 0–3.5 plus Step 2.7 with ~3 phrases, keeping the read side of Step 2.5 (2.5A read candidates, 2.5B recency weight) and Step 2.7 activation, but skipping Step 2.5C (coverage amend/learn) and Step 4 (synthesis-persist) — the entire write side. `deep` runs everything, 10 phrases. Glance escalates to deep specifically for recency-channel (Channel 2) standards (measured: glance 0/5, deep 4/5, #661). Joe already decided (prior `/opsx:explore`) to encode this as **two separate runbook notes**, `recall-glance` and `recall-deep` — not one runbook with escalation logic in the body, not a schema tier field.

Only one main spec literally names the dead file path as a carrier: `recall-payload-cuts/spec.md:90` ("Step 2.5 of the recall procedure (`agent-instructions/skills/recall/SKILL.md`)"). The other `recall-*` specs (`recall-glance-deep-dial`, `recall-two-channel-payload`, `recall-centroid-sampling`, `recall-matched-note-floor`, `recall-query-timings`) describe behavior in carrier-agnostic terms ("recall is invoked with glance mode", "the query SHALL append...") and need no delta — confirmed by reading each, not assumed.

Both `recall`'s write sites already reference write-memory's real promoted runbook (`1053.2026-09-22.write-memory-compose-execute-verify`) at `recall/SKILL.md:192,284,307`, repointed in an earlier session change.

`recall` differs from every prior conversion in one structural way: every session's first action already runs an implicit recall-shaped query (`guidance-runbook-follow-frame`'s shim first-action requirement), so a `recall` runbook's surfacing path may lean on similarity rather than an explicit lexical trigger the way `please`/`curate`/`learn` do — route's own conversion (no `triggers:` at all, similarity-only) is the closer precedent here than please/curate.

No `recall` eval exists. Eval-fixture runbook notes already exist as starting material (not built fresh) at `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/shim/recall-learn/vault/`: `recall-glance-vs-deep-mode`, `recall-note-synthesis-and-coverage`, `recall-activation-and-closing-synthesis`, `recall-from-unified-memory` — spot-checked faithful to the real skill in an earlier session pass, but never scored against a skill baseline.

## Goals / Non-Goals

**Goals:**
- Promote `recall`'s validated runbook set (two entry points + shared sub-runbook(s)) to the production vault, retiring `agent-instructions/skills/recall/SKILL.md`.
- Encode glance/deep as two separate runbook notes per Joe's decision, without duplicating ~19KB of near-identical procedure text between them.
- Build a `recall` eval task and skill-arm baseline from scratch, reusing the existing eval-fixture runbook notes as starting material where faithful.
- Fix the one real carrier reference (`recall-payload-cuts/spec.md:90`).

**Non-Goals:**
- Retiring `agent-instructions/guidance/recall.md` or resolving its redundancy with the shim's re-entry moments — explicitly deferred as a fast-follow (proposal.md's Deferred section), buildable immediately once this change lands but not bundled into it.
- Retiring `guidance/delegate.md` or `guidance/learn.md` — separately deferred, tracked in the `learn` change.
- Changing `recall`'s actual behavior (the procedure, the escalation rule, the placement/coverage logic). Only the carrier changes.
- A new schema field or query mechanism for encoding depth — ruled out by Joe's decision (two notes, not a tier field).

## Decisions

**D1. Two thin triggered entry points (`recall-glance`, `recall-deep`) plus shared sub-runbook(s) for the common procedure content — not two full independent copies of the procedure.** The skill's own text is written as ONE sequence with inline `[glance: …]` differences, not two separate procedures; duplicating the full ~19KB into two notes would mean every future edit to shared steps (0, 0.5, 1, 2, 2.5A/B, 2.7, 3, 3.5) must be made twice and risks drift. Structure: a shared "core procedure" sub-runbook carrying Steps 0 through 3.5 (with the glance/deep branch points preserved as explicit body text, e.g. "if running as `recall-deep`, also do X"), and a "write extension" sub-runbook carrying Step 2.5C + Step 4 (the write side, wikilinking write-memory's real basename). `recall-glance`'s body wikilinks only the core procedure and states it stops there (plus the C5 escalation rule and a wikilink to `recall-deep` for that case). `recall-deep`'s body wikilinks the core procedure AND the write extension, and explains why/when glance should have escalated to it (per Joe's decision). Exact sub-runbook count and boundary (e.g., whether Step 2.5's read side (4,783 B) needs to be its own sub-runbook, given it's the single largest step) is a task-level decision, verified against real byte counts and the 1200-byte red_flags-per-note budget, not assumed here — mirrors `please`'s and `write-memory`'s own "measure, don't guess" precedent. Alternative considered: two fully independent runbooks (simplest to reason about, matches Joe's "two separate runbook notes" literally) — rejected as the primary shape because ~15KB of shared content would exist in two places; the two triggered entry points still satisfy Joe's decision (there are two runbook notes an agent can be matched to / choose between), while the shared sub-runbook avoids the duplication problem. If, once built, the split proves confusing in practice (e.g. eval trials struggle to follow the wikilink chain), fall back to two fully independent copies — flagged as an Open Question, not pre-decided against.

**D2. Both entry points wikilink write-memory's real basename directly; no fixture-placeholder detour.** Same approach as `learn`'s conversion — write-memory is already promoted, so there is no temporary basename to invent and later repoint (the exact failure mode that cost `write-memory`'s own conversion two of its three paid attempts).

**D3. `red_flags` split across notes, prioritized under the 1200-byte cap per note.** The 28-row/4,206-byte table is the largest converted so far. Rows specific to glance-only behavior (e.g. the C5 escalation condition) go on `recall-glance`; rows specific to the write side go on the write-extension sub-runbook or `recall-deep`; rows about the core procedure (phrasing, clustering, coverage judgment) go on the core sub-runbook. Exact row-by-row placement is a task-level activity against the real built notes, per every prior conversion's "measure, don't guess" precedent (D3 in `please`, D1/D3 in `write-memory` and `learn`).

**D4. Fix the one real carrier reference.** `recall-payload-cuts/spec.md:90` names the dead file path (`agent-instructions/skills/recall/SKILL.md`) inside a requirement body — a delta applies now (per the precedent this session set: requirement-body carrier claims get fixed via delta, not deferred to archive-time Purpose-only edits). The other `recall-*` specs describe behavior in carrier-agnostic terms and need no delta — confirmed by direct read, not assumed.

**D5. Trigger decision: evaluate similarity-only surfacing before adding explicit `triggers:`.** Unlike `please`/`curate`/`learn`, `recall`'s runbooks are the one thing every session's shim-mandated first query already searches for — the query IS a recall-shaped action by construction. `route`'s conversion (no `triggers:`, similarity-only) is the closer precedent than `please`/`curate`'s. During the retrieval-check task, score `recall-glance`/`recall-deep` against real agent first-query phrases with NO triggers first; only add `triggers:` (candidates: `/recall`, "recall glance", "recall deep") if similarity-only surfacing measurably fails to rank them competitively against `fact`/`feedback` notes on the phrases the shim's re-entry moments (endorsing an approach, declaring done, an unexplained failure, starting a new approach) would actually produce.

**D6. Eval built from scratch, reusing existing fixture notes as starting material where faithful, applying the write-memory cost lesson.** The four existing eval-fixture runbook notes are read and re-validated for fidelity against the CURRENT real skill (which may have drifted since they were written 2026-09-14) before being trusted as-is — do not assume they're still accurate. Skill-arm baseline first (paid, cost confirmed), then shim-only per entry point (glance and deep may need separate validation runs, since they're separately triggered/matched). **Before any paid re-run following a wording fix, verify via hash-check that the fix reached the live skill file the harness deploys** — the exact discipline that would have saved `write-memory`'s conversion $2.24.

**D7. Retirement gate is the same three-part bar** every prior conversion used, applied once for the whole `recall` conversion (both entry points and all sub-runbooks promoted together, not staged).

## Risks / Trade-offs

- **[Risk] The shared-sub-runbook structure (D1) is more complex than any prior conversion's split and could confuse an agent following the wikilink chain (glance → core; deep → core + write extension).** → Validated directly in the eval's validity gate: does the agent actually follow both links in the right order? If trials show confusion, D1's fallback (two fully independent copies) is available without re-deciding the "two entry points" choice itself.
- **[Risk] `red_flags` is the largest yet (28 rows) and may be genuinely hard to fit even split three ways.** → Task-level measurement against real byte counts; if still tight, prioritize by the same please/write-memory/learn method (session-frequency × failure severity), moving lower-priority rows into sub-runbook body text rather than `red_flags`.
- **[Risk] Dropping `triggers:` (D5) if similarity-only surfacing works could leave `recall` less discoverable than `please`/`curate`/`learn` for an explicit `/recall` request.** → D5's evaluation is empirical, not assumed either way; if similarity alone under-performs on the shim's actual re-entry phrasing, add `triggers:`.
- **[Trade-off] New eval spend on the largest, most structurally novel conversion yet, potentially needing separate glance/deep validation runs (more trials than prior conversions).** → Estimate and confirm cost before running, per project standard; consider validating glance first (cheaper, no write side) before spending on deep.
- **[Risk] Missed doc references after retirement, given `recall` is referenced more widely than any prior skill (it's the memory-lookup mechanism every other skill/runbook cites).** → Doc-surface enumeration plus an independent fresh-context reviewer before deletion, per every prior conversion's established pattern.

## Migration Plan

1. Re-validate the four existing eval-fixture runbook notes against the current real skill for fidelity (they predate this session and may have drifted); build the eval task and skill-arm baseline.
2. Decide the final sub-runbook split (D1) against measured byte counts; write `recall-glance`, `recall-deep`, and the shared sub-runbook(s) in the fixture vault, wikilinking write-memory's real basename directly (D2).
3. Fresh-context reviewer checks the fidelity report against the real `SKILL.md` for lost content.
4. Retrieval check: score both entry points against real agent phrases with no `triggers:` first (D5); decide whether to add triggers based on the result.
5. Run the shim-only eval per entry point against the baseline (D6); apply the hash-check discipline before any wording-fix re-run.
6. If D7 met: promote sub-runbook(s) first (so wikilinks resolve), then both entry points, via `engram learn runbook`.
7. Doc-surface enumeration; independent fresh-context review; perform every update/rewrite row (fix `recall-payload-cuts/spec.md:90` per D4).
8. Delete `agent-instructions/skills/recall/`; run `engram update`; `targ check-full`; commit, referencing #760 — this closes #760 fully once `learn`'s sibling change has also landed.
9. Archive the change; sync specs. File the `guidance/recall.md` retirement as a new, immediately-actionable issue (proposal.md's Deferred section).

Rollback: revert the retirement commit restores the skill; promoted runbook notes can be removed via `engram` note deletion.

## Open Questions

- Final sub-runbook count/boundary (D1) — resolved at task time against real byte counts and eval-validity-gate results, not decided here. If the shared-core structure proves confusing in the eval, fall back to two independent full copies.
- Whether `triggers:` are needed at all (D5) — resolved empirically during the retrieval-check task.
- Whether glance and deep need fully separate eval validation runs, or whether one combined task can exercise both (e.g. a scenario that starts at glance and is expected to escalate) — decide during task 1.x eval design.
