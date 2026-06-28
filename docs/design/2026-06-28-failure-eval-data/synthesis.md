# Synthesis pass (sonnet) over the 137 confirmed failures

Verbatim output of the synthesis agent that clustered the mined failures into patterns,
candidate new recall moments, trap-RED candidates, the ceiling, and the narrative.
Source-of-truth for `../2026-06-28-failure-eval-material.md`.

## 1. Patterns (13, summing to ~135)

| pattern | count | klass | coverage | subtle | role | flag | axis |
|---|---|---|---|---|---|---|---|
| incomplete-multi-part-execution | ~23 | APPLICATION | uncovered | 0.60 | both | mixed | C5 |
| unverified-claim-or-source | ~17 | APPLICATION | uncovered | 0.71 | both | mixed | new-C7 |
| explicit-constraint-violated | ~15 | APPLICATION | uncovered | 0.47 | main | mixed | C5 |
| premature-completion-declaration | ~14 | TRIGGER | uncovered | 0.64 | both | mixed | C5 |
| path-structure-existence-not-verified | ~12 | TRIGGER | uncovered | 0.75 | both | tactical | C3 |
| mismatch-absorbed-not-escalated | ~10 | APPLICATION | uncovered | 0.70 | both | mixed | C6 |
| silent-spec-deviation | ~9 | APPLICATION | uncovered | 0.78 | main | behavioral | C5 |
| tool-failure-not-escalated | ~8 | APPLICATION | uncovered | 0.38 | main | mixed | na |
| thinking-doing-gap | ~7 | APPLICATION | uncovered | 0.57 | main | behavioral | na |
| adversarial-role-collapse | ~6 | APPLICATION | uncovered | 0.83 | subagent | behavioral | C5 |
| tool-schema-or-invocation-error | ~6 | TRIGGER | uncovered | 0.33 | main | tactical | C3 |
| shallow-test-or-verification | ~5 | TRIGGER | partial | 0.80 | main | tactical | C5 |
| recall-retrieved-not-integrated | ~3 | APPLICATION | uncovered | 0.67 | subagent | behavioral | C5 |

(Per-pattern detail notes preserved in the parent deliverable's Section 2.)

## 2. Candidate new recall moments (7)

- **before-declaring-done** (~15) — cue: completion-summary language ("done", "all X added", "ready for"). Surface: run `targ check-full` not just `go vet`; verify every checklist item ran; tool-success ≠ file-modified; confirm output path is the target not /tmp. Partially hookable (pre-response keyword scan).
- **on-reading-multi-part-instruction** (~12) — cue: "X AND Y AND Z", "ALL", numbered required-items list. Surface: enumerate + verify each item independently. Deterministic scan feasible (noisy).
- **before-final-recommendation-verdict** (~12) — cue: "I recommend / accept / PASS / FAIL / conclusion". Surface: block≠accept-with-notes; severity-action consistency; null result invalidates the verdict. Hookable (pre-response keyword).
- **on-detecting-contradiction-with-prior** (~10) — cue: "wait, this shows / but the handoff says / that contradicts". Surface: contradiction → re-investigate + escalate, don't absorb. Recall-trigger on thinking patterns (noisy).
- **before-writing-code-or-first-edit** (~10) — cue: first Edit/Write in a task. Surface: verify path exists; check where similar symbols live; read the entry point; note any spec deviation. Fully deterministic PreToolUse on Edit/Write.
- **after-tool-failure-before-retry** (~8) — cue: non-zero exit / "permission denied" / "not found" / InputValidationError. Surface: after N=2 identical failures stop+escalate; pivot tools; scratchpad path; deferred tools need ToolSearch. **Fully deterministic PostToolUse — the single most hookable moment.**
- **after-search-before-synthesis** (~8) — cue: WebSearch/Grep returned, about to synthesize. Surface: WebFetch to verify content; negative claims need a targeted search. Deterministic PostToolUse.

## 3. Trap candidates (7)

See parent deliverable Section 3 for the full specs. Patterns turned into trap-RED candidates:
incomplete-multi-part-execution (tactical/C5), premature-completion-declaration (tactical/C5),
unverified-claim-or-source research-form (tactical/new-C7), path-structure-existence-not-verified
(tactical/C3), mismatch-absorbed-not-escalated (tactical/C6), tool-failure-not-escalated (mixed/na),
adversarial-role-collapse (behavioral/C5 — requires prior-turn approval-priming to fire, confirming
the behavioral classification).

## 4. Ceiling

Pure CAPTURE = 10/137 (7.3%). Genuinely unreachable by any memory (`mem=n`) ≈ 2/137 ≈ 1.5%
(silent-workaround-without-doc; hallucinated tool schema). Theoretically addressable
(APPLICATION 77 + TRIGGER 50 = 127/137 ≈ 93%) — but the dominant APPLICATION+uncovered combo
(~55 records) needs a new injection MOMENT, not a new memory.

## 5. Narrative

Dominant shape: 56% APPLICATION (knowledge present/available, not applied) × 77% uncovered (fails at a
mid-task cue current recall doesn't reach). Top 3 patterns (multi-part 23, constraint 15, premature-done
14) = 38% of all failures, uniformly APPLICATION+uncovered. Subagents = 66% (90/137): they do the
object-level work under a thin brief with no mid-task recall.

Why uncovered+APPLICATION: recall fires at 3 coarse moments (task-init, subagent recall-first, parent
brief); failures happen mid-task (before a verdict, after a tool fails, when a checklist item is dropped,
when a contradiction surfaces). These are structurally predictable + several are directly hookable.

Highest-value single change: a **before-declaring-done** checkpoint (covers ~27 uncovered records ≈ 26%)
+ an **after-tool-failure-before-retry** PostToolUse hook (8 records, fully deterministic). Both need
only new FIRING POINTS — the lessons already exist in the vault. adversarial-role-collapse is the
highest-consequence class but behavioral — needs a structural blocking gate, not a recalled tip.

Tactical vs behavioral: ~40% tactical (~55 records — cheaply evalable C5/C3/C6/C7 traps where a clean
model genuinely fails); ~60% behavioral (~82 — only fire under rich multi-turn priming; a clean toy
passes them; need an expensive rich-context harness, not stubs).
