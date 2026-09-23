## Context

`route-skill-to-runbook`, `please-skill-to-runbook`, and `curate-skill-to-runbook` (all archived) established the pattern this change follows: promote a validated runbook body via `engram learn runbook`, gate retirement on a shim-only eval within one trial of the skill row, verify retrieval in the real vault, confirm `shim.md` is imported, then delete the `SKILL.md` and update every live reference. `runbook-lexical-triggers` (archived) added `triggers:` and `engram query --text` so a runbook can surface on a literal cue, whole-word matched. `guidance-runbook-follow-frame`'s existing requirements (`openspec/specs/guidance-runbook-follow-frame/spec.md`) define the obligations once a runbook is "matched": announce it, restate its steps, treat `done_when` as the completion bar, read `red_flags`, and — the requirement at line 109 — fetch and follow a *wikilinked runbook inside another runbook's body* transitively.

`write-memory` breaks the pattern in one specific way: **it has no trigger surface of its own.** A user never says "write-memory" or "/write-memory" — the skill's own description states it "requires a handoff; do not fire on your own judgment." It is reached exactly one way: `recall` or `learn`, having already judged what to capture, invoke it as their next action (`agent-instructions/skills/recall/SKILL.md:192,284,307`; `agent-instructions/skills/learn/SKILL.md:119,127,137-138,189-190,199-200,225`, nine call sites total — 3 in recall + 6 in learn, verified line-by-line against the real files; `recall/SKILL.md:293,339` and `learn/SKILL.md:25,98,231-232` are prose mentions of write-memory, deliberately NOT invocation sites and not touched). The existing follow-frame requirement at line 109 covers a *runbook* wikilinking another runbook (please → route). It does not cover a *skill* (still `SKILL.md`, not yet a runbook) naming a runbook as its next action — because `recall` and `learn` are staying skills in this change; only `write-memory` converts. This is the one piece of new design.

The eval-fixture runbook `write-memory-compose-execute-verify` at `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/shim/recall-learn/vault/8.2026-09-14.write-memory-compose-execute-verify.md` (107 lines/931 words) already exists and was spot-checked faithful to the real skill during exploration. No `write-memory` eval task exists yet in `dev/eval/cumulative/runbook_vs_skill/phase2/`.

## Goals / Non-Goals

**Goals:**
- Promote write-memory's validated runbook body to the production vault, retiring `agent-instructions/skills/write-memory/SKILL.md`.
- Define how a *skill* (not a matched runbook) reaches and follows a runbook it names as its next action, since write-memory has no trigger of its own.
- Build a write-memory eval task and skill-arm baseline, using the same D8 bar the prior three conversions used.
- Update `recall`/`learn`'s own text to name/wikilink the write-memory runbook instead of "invoke the write-memory skill."

**Non-Goals:**
- Converting `recall` or `learn` themselves. They stay `SKILL.md` files in this change; converting them is separately-scoped future work (see proposal.md's Deferred section) — `recall`'s glance/deep split is a decided but unbuilt design (two runbook notes, `recall-glance` and `recall-deep`), and `learn`'s conversion is closer in shape to `please`/`curate` (one procedure, several lesson-kind branches) than to a worker.
- Retiring `agent-instructions/guidance/{recall,delegate,learn}.md` or resolving the `shim.md`/`recall.md` firing-cue redundancy — both wait on `recall`/`learn` themselves being real runbooks.
- Changing write-memory's actual behavior (the handoff contract, the per-kind compose rules, the verify/report step). Only the carrier changes.
- A new schema field or query mechanism. `--trigger`/`--text`/whole-word matching already ship; this change adds no new lexical mechanism, since write-memory deliberately has no trigger.

## Decisions

**D1. One runbook note; no sub-runbooks.** Write-memory's compose/execute/verify/report procedure fits in one query item (the starting-material eval fixture is 931 words / 107 lines; the built runbook note itself is ~8,213 bytes total / ~6,685 bytes body — well under any truncation concern). Unlike `please` (three sub-runbooks) or the coming `recall` conversion (28-row red-flags table forcing splits), write-memory's small red-flags set (flag-mixing across kinds, `--tags` vs `--tag`, qa-takes-no-tags, wikilink-not-plain-text) fits comfortably under the 1200-byte cap in one note.

**D2. No `triggers:` field on the write-memory runbook.** Write-memory is never invoked by a user's own words; giving it triggers would let it surface on a stray mention of "compose a vault write" with no parent judgment behind it, contradicting its own "do not fire on your own judgment" rule. It is reached by name only (D3).

**D3. `recall`/`learn` name the write-memory runbook by basename/wikilink; a new follow-frame requirement covers a skill (not a matched runbook) reaching a runbook by name.** Concretely: every call site in `recall/SKILL.md` and `learn/SKILL.md` that currently reads "invoke the **write-memory** skill with this handoff" changes to name the promoted runbook's basename (e.g. "fetch and follow `[[<basename>]]` with this handoff") — the same wikilink syntax `please`'s conversion used for its route reference, which the route eval confirmed agents follow when the syntax is stated plainly and directly above the field/action.
Add a requirement to `guidance-runbook-follow-frame` (delta in this change): when a *skill's own instructions* (not a query-matched runbook) name a specific runbook by basename or `[[wikilink]]` as the agent's next action, the agent SHALL fetch it (`engram show <basename>`) and apply the same follow-frame obligations (announce, restate steps, treat `done_when` as the bar, read `red_flags`) as if it had been matched by query. This generalizes the existing line-109 requirement (runbook→runbook) to skill→runbook, since `recall`/`learn` remain skills in this change. Alternative considered: leave `recall`/`learn`'s prose as the only instruction, relying on the model to "just follow good practice" — rejected; every prior conversion (`please`→route, `curate`) needed an explicit stated requirement plus an eval to confirm agents actually fetch and read a named runbook rather than treating the mention as a citation.

**D4. Build the eval task in scope, mirroring `please`'s decision.** No write-memory eval exists. Because write-memory has no independent trigger, the eval cannot be "give the agent an ask and see if the runbook surfaces" (please/curate's pattern) — it must instead: (a) run a `recall`- or `learn`-shaped task with `recall`/`learn` kept as skills, (b) verify the handoff reaches write-memory correctly, (c) score whether the runbook is fetched-by-name, followed, and the resulting vault write matches what the skill-arm baseline (write-memory as a skill) produces. Bar: shim-only run (write-memory as runbook, `recall`/`learn` as skills) within one trial of the skill-only row (write-memory as skill, `recall`/`learn` as skills) on write-composition correctness (right kind, right flags, right content) and on "runbook fetched and followed" as an explicit validity gate — a run that produces a correct write only because the agent skipped the runbook and composed the command from memory does not count as D8-passing, even if the end state matches, because it would leave the retirement unvalidated against real deviation risk (mirrors the route eval's own wikilink-fidelity finding).

**D5. Promote via `engram learn runbook`, never by hand,** per vault note 958 and every prior conversion — the note is rebuilt from real fields (`situation`, `body`, `done_when`, `red_flags`) so the sidecar embeds correctly, not copied from the fixture file.

**D6. Retirement gate is the same three-part bar** please/curate used: (a) D4's eval bar met, (b) retrieval-by-name verified against the real vault (`engram show <basename>` on the promoted runbook succeeds and its content matches the fixture; there is no query-retrieval check to run since write-memory has no trigger — the "retrieval" being verified is that `recall`/`learn`'s updated wikilink text resolves to a real basename), (c) `shim.md` confirmed imported in the real `CLAUDE.md`(s) — already true today (verified during the `please`/`curate` conversions this session), so (c) is a re-confirmation, not new work.

## Risks / Trade-offs

- **[Risk] The eval can't reuse please/curate's "explicit ask, runbook surfaces or doesn't" measurement shape, since write-memory has no trigger.** → D4's design (score correct composition + a validity gate on runbook-fetched-and-followed, not surfacing) is the mitigation; if the validity gate can't be scored cleanly from a transcript (e.g. the agent silently skips `engram show` but still writes correctly by luck), the eval task construction (task 2.x) must include a fixture where the runbook's actual content differs subtly from what the agent would compose from memory alone (analogous to how the route eval's wikilink-vs-plaintext finding only surfaced because the runbook's real syntax mattered) — this is a task-design detail to resolve during implementation, not before.
- **[Risk] Nine call sites across `recall`/`learn` name write-memory; missing one leaves a stale "invoke the write-memory skill" instruction pointing at a deleted file.** → Task list enumerates all nine by line number; the doc-surface enumeration step (mirroring `please`/`curate`'s) plus a fresh-context reviewer catch any missed site before deletion, per this session's established pattern (both prior conversions' independent reviews found real gaps the author's own list missed).
- **[Trade-off] This change does not resolve the `shim.md`/`recall.md` redundancy** flagged in GitHub #760's own comment thread, even though it's directly adjacent. → Deliberately deferred (Non-Goals): resolving it before `recall` itself is a runbook risks removing `recall.md`'s firing cues while nothing replaces the procedure they gate, per the archived `runbook-shim-follow-frame` change's own D0/D2 deferral reasoning.
- **[Risk] New eval spend on a task with an unusual (worker, not trigger-fired) shape.** → Estimate and confirm cost before running, per project standard; expect roughly please/curate's per-stage range ($1–5) given a comparably small fixture.

## Migration Plan

1. Build the write-memory eval task, skill-arm baseline (`recall`/`learn` real skills, write-memory real skill) — confirm the baseline is a usable D8 reference before building anything else.
2. Write the write-memory runbook in the eval fixture vault from the existing `write-memory-compose-execute-verify` note, with `red_flags`, no `triggers`; write the conversion-fidelity report.
3. Update `recall`/`learn`'s nine call sites in the fixture's frozen skill copies to name the runbook by wikilink; add the new follow-frame requirement's delta.
4. Run the shim-only arm (write-memory as runbook, `recall`/`learn` as real skills) against the baseline; check D4's bar including the validity gate.
5. If met: promote to the production vault via `engram learn runbook`; update the real `recall`/`learn/SKILL.md` files' nine call sites; verify `engram show <basename>` resolves.
6. Doc-surface enumeration across CLAUDE.md/README/GLOSSARY/architecture docs; independent fresh-context review of the enumeration (per this session's established pattern); perform every update/rewrite row.
7. Delete `agent-instructions/skills/write-memory/`; run `engram update` and confirm deployed copies are gone; `targ check-full`; commit with `Fixes #760`'s write-memory slice only (issue stays open for `recall`/`learn`, or is split into a dedicated write-memory sub-issue if that's cleaner for tracking — decide at commit time).

Rollback: revert the retirement commit restores the skill; the promoted runbook can be removed via `engram` note deletion.

## Open Questions

- Should this close #760 partially (leave it open, noting write-memory done) or should a fresh issue be filed for the write-memory slice specifically, with #760 narrowed to just recall+learn? Decide before the retirement commit (task 6.x).
- Whether the write-memory eval's validity-gate fixture (risk above) needs a deliberately non-obvious runbook detail to force a real "did they read it" test, the way route's wikilink syntax did — resolve during eval task construction, not in this document.

## 2026-09-22 — Iteration after task 3.2's D8 miss (task 3.4)

Task 3.2 (n=3, shim-only) FAILED the validity gate 3/3: no trial ran `engram show <basename>` before
composing a write. Transcript evidence (`results/3.2_write_memory_shim_only_sonnet5.md`) showed the
model reading "fetch and follow `[[wikilink]]`" as an instruction to invoke a **Skill tool** named
`write-memory` — a habit carried over from calling `Skill{recall}`/`Skill{learn}` moments earlier in
the same transcript — and, on the resulting `Unknown skill: write-memory` error, falling back to
composing the `engram learn` command from memory/`--help` rather than recovering via `engram show`.
Joe approved fixing both of the report's named levers (option C) before re-running:

1. **Call-site wording (D3's prescribed phrasing was ambiguous in practice).** All 9 call sites in the
   fixture's frozen `recall`/`learn` copies (`encodings/taskWriteMemory/recall-learn/{recall,learn}/SKILL.md`)
   were reworded from "fetch and follow `[[6a.2026-09-22.write-memory-compose-execute-verify]]`" to name
   the literal shell command, e.g. "run `engram show 6a.2026-09-22.write-memory-compose-execute-verify` in
   Bash (a shell command — write-memory is not a Skill tool) and follow what it prints" — carrier-language-only,
   every other word unchanged. This does not amend D3 itself (the underlying obligation — a skill naming a
   runbook by basename must fetch and follow it — is unchanged); it corrects the call-site *wording* D3
   prescribed, which this eval showed collides with the model's `Skill{name}` reflex.
2. **New `red_flags` entry on the WM-R runbook** naming the trap directly: "You are about to invoke the
   Skill tool with name write-memory -- stop, it no longer exists as a skill; run `engram show
   6a.2026-09-22.write-memory-compose-execute-verify` in Bash instead and follow what it prints." Added via
   `engram amend --red-flag` (replace-whole semantics, all 5 entries re-passed) against the fixture vault,
   never hand-edited, per vault note 958. New total: 955/1200 bytes (was 736/1200 with 4 entries) — still
   comfortable headroom under D1's cap.

Both fixes are scoped to the fixture only; the real, live `agent-instructions/skills/{recall,learn}/SKILL.md`
and the production vault are untouched pending a fresh D8 pass (task 3.4).

## 2026-09-22 — D-fix: Step 2.5 self-referential QA over-capture (found via 3.4b/R-0)

**Root cause.** 3.4b's own passing run (D8 MET) still logged one real, reproducible defect
unrelated to write-memory's conversion: trial R-0 wrote an unrequested `engram learn qa` pair
(two extra files) after correctly writing the two notes the task required. Cause: the (pre-existing,
unrelated-to-this-change) `learn` skill's Step 2.5 defines "substantively answered" as "the answer
body contains ≥1 `[[wikilink]]` OR you crystallized a new vault note (Step 2) as the answer" — an
OR whose second disjunct is self-contradictory: it treats "I just wrote the Step 2 note that
answers this" as a TRIGGER for a separate QA-capture write, rather than as evidence the answer is
already captured. R-0's transcript confirms this literally: the agent quoted Step 2.5's old text,
then justified the extra `engram learn qa` write as satisfying "crystallized a new vault note
(Step 2) as the answer."

**Scope decision.** This bug lives in `learn`'s Step 2.5 (unrelated to write-memory's carrier
conversion, this change's actual subject), but is added here rather than as a separate change: it
was found mid-implementation of this change's own validation (3.4b), the fix is small (one Gate
clause), and this change already touches every `learn` write site. A separate change would
duplicate setup for one paragraph of prose.

**Fix.** Extend the existing "Gate — do not duplicate" clause in Step 2.5 (it already excludes
recall's Step 4 duplicate pairs) to also exclude the self-referential case: a question answered by
a Step 2 note the agent wrote THIS turn is already captured by that note (whatever wikilinks it
cites) — no separate QA pair. The wikilink-based disjunct is untouched and stays live on its own:
a question answered by citing an EXISTING prior note, with no new Step 2 note written for it this
turn, still gets its QA pair. See the new `learn-adhoc-qa-capture` delta spec for the normative
statement.

**RED evidence.** (a) The already-kept 3.4b/R-0 transcript
(`.../trials/write-memory-R-0/repo/82febb65-7bf0-47e6-a334-54d37987361c.jsonl`) shows the agent
received the OLD Step 2.5 text verbatim and cited exactly the self-referential disjunct to justify
the extra write — no new spend, confirms the bug is real in production wording. (b) A fresh
headless micro-test (`claude -p`, no tools, sonnet5, 3 reps) reproduced the same failure
independent of that specific transcript: given the OLD wording + a scenario mirroring R-0 (a
self-posed question answered by a new Step 2 note that also cites an existing wikilink), all 3/3
reps answered "YES, write a QA pair," citing the same disjunct.

**GREEN evidence.** Same micro-test harness, NEW wording: (1) the self-referential scenario now
gets "NO, do not write a QA pair" in 3/3 reps, citing the extended Gate clause; (2) a second
scenario (question answered by citing an EXISTING prior note, no new Step 2 note this turn) still
gets "YES, write a QA pair" in 3/3 reps, confirming the legitimate wikilink trigger is untouched.
Zero variance across all 9 runs (3 cells × 3 reps) — the wording is binding, not ambiguous.
No further live pressure-test re-run of the full write-memory fixture was performed (out of scope
cost-wise for a skill-wording fix already covered by 100%-convergent micro-tests); cost was
negligible (9 short, single-turn, mostly-tool-free `claude -p` calls under the logged-in session,
not a metered API key).
