## Context

The archived `runbook-shim-follow-frame` change converted `recall`, `learn`, `write-memory`, and
`route` from Claude Code skills into engram runbook vault notes — but only inside its own eval
fixture vaults, for a shim-only comparison harness (D7: "No engram skills in the trial config dir;
recall and learn present in the trial vault as runbooks ... Otherwise the test measures a hybrid that
will not exist"). None of the four has been promoted to the production vault: all four `SKILL.md`
files still ship live today, confirmed by direct inspection
(`agent-instructions/skills/{recall,learn,write-memory,route}/`), and no production vault note exists
for any of them. Promoting recall/learn/write-memory is #760's scope, not this change's — route is
the only one of the four this change touches.

`guidance-runbook-follow-frame` (`openspec/specs/guidance-runbook-follow-frame/spec.md`) is the
generic contract that archived change established: once a `kind: runbook` note matches a task, the
shim requires the agent to announce it, restate its steps as a plan, treat `done_when` as the
completion bar, read and react to `red_flags`, and follow wikilinked sub-runbooks — uniformly, for
any runbook, not route-specific. That contract is already implemented and unchanged by this proposal;
this change only adds a new runbook instance (route's) plus one `red_flags` entry to it.

`route`'s conversion was validated inside the archived change (task 4.4 of `runbook-shim-follow-
frame`) but never promoted. The validated fixture — `Route-R`
(`dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskRoute/Route-R/vault/1.2026-09-12.route-
dispatch-tier-selection.md`) — already carries the content fixes made across that eval's 6 rounds:

1. The live `agent-instructions/skills/route/SKILL.md`'s "Red flags — STOP and re-read" table (14
   rows) exists only as a markdown table in the skill body. The fixture moves the equivalent content
   into a structured `red_flags:` frontmatter list (verbatim-fused, condition + corrective reasoning
   per row) — the schema `guidance-runbook-follow-frame` already expects.
2. The live SKILL.md references "the write-memory skill" as bare prose in 4 places. The fixture
   wikilinks all 4 as `[[8.2026-09-14.write-memory-compose-execute-verify]]`, so the shim's
   transitive-follow requirement (spec line 77-84) actually fires on them.
3. The live SKILL.md's flag-naming is ambiguous ("there is no `--kind` flag; write-memory maps the
   fields below to the real `engram learn fact` flags"). The fixture names the forbidden guess and
   the real flag explicitly ("no `--tags` flag — the installed CLI rejects `--tags` outright ...; the
   real tag flag is `--tag <family>/<value>`").

Despite those fixes, the eval's own D8 acceptance bar (parity with route's skill-row behavior) was
never met for route across 6 rounds, final spend $27.04
(`openspec/changes/archive/2026-09-17-runbook-shim-follow-frame/tasks.md` task 4.4). Every round from
rerun 2 onward converged on the same single remaining defect: agents write the aggregate-evidence
field (`route-evidence-<work-kind>`'s `--object` text) as plain-text citations instead of the
`[[note-basename]]` wikilink syntax the runbook's own two aggregate-write templates ("Match" and "No
match" branches) already show, verbatim, directly above the field being filled in. Three independent
character-by-character re-reads of those templates (reruns 3, 4, 5) confirmed the wording is not
ambiguous. Per the eval's own decision rule ("if the runbook already says to wikilink and the agent
just didn't, report it, don't fix the checker"), no fix was attempted within that change — it closed
with route's D8 bar explicitly unmet, and filed the gap as follow-up (task 5.2 → GitHub issue #756 →
split into #757 for route specifically).

## Goals / Non-Goals

**Goals:**
- Promote `Route-R`'s validated body into route's production runbook, retiring
  `agent-instructions/skills/route/SKILL.md`.
- Close the one confirmed remaining defect blocking D8 for route (the wikilink-vs-plaintext
  aggregate-evidence miss) using a lever the archived eval never tried.
- Re-measure against the same D8 bar the archived eval used, so this change's own "done" claim rests
  on the same evidence standard as recall/learn/write-memory's promotions, not a lower one.

**Non-Goals:**
- Re-wording the aggregate-write templates themselves. Three independent re-reads in the archived
  eval already confirmed the `[[wikilink]]` syntax is unambiguous and correctly placed; spending
  more words on an already-clear template is a saturated lever with low expected yield, not the
  fix this change proposes.
- Converting `please`, `curate`, or the openspec (`opsx`) skills. Those are #758, #759, and #761
  respectively — separate changes.
- Changing `guidance-runbook-follow-frame`'s requirements. That spec already covers `red_flags`
  handling generically (Requirement: "The agent SHALL read a runbook's `red_flags` and stop on one
  firing"); this change adds one entry's worth of *content* to one runbook, not new *behavior*.
- Changing route's dispatch-evidence *behavior* — the tier-selection loop, the evidence-note/
  aggregate-write procedure, and the count-as-audit machinery are unchanged. `route-dispatch-
  evidence` and `route-evidence-rubric` do get small delta specs (see Capabilities in proposal.md),
  but only to update two places where their normative text says "the skill" / "the skill's
  red-flags table" — carrier-identity language, not behavior.

## Decisions

**D1. Promote by rebuilding the whole note via `engram learn runbook` / `engram amend`, never a
hand-edit.** Per vault note 958 ("rebuild the whole note when a fix rewrites retrieved content"), the
promoted runbook must be created through the real CLI (situation, steps/body, done_when, and the new
`red_flags` entry all passed as real fields) and re-embedded — not copied by hand from the fixture
file, which would leave the sidecar `.vec.json` stale or mismatched. The fixture file is the source
of *content*, not a file to symlink or cp into the vault.

**D2. Add exactly one `red_flags` entry; do not touch the aggregate-write template wording.**
Alternatives considered:
- *Reword the templates with heavier emphasis (bold, a warning callout, repeating the instruction)*
  — rejected. This is the lever the archived eval's own investigators repeatedly inspected and judged
  already-unambiguous; retrying it without new evidence that wording was the actual cause is the same
  displacement pattern vault note 122 warns against (re-litigating a settled finding on no new
  evidence).
- *Fix the checker (`done_when_checks.sh` Check 6) to also accept a plain-text citation* — rejected;
  explicitly rejected in the archived eval too ("report it, don't fix the checker" — the checker is
  verifying real production behavior other tooling depends on the wikilink for, e.g. `engram count`
  audits and the vault's wikilink graph).
- *Add a `red_flags` entry naming the exact condition* — accepted. `red_flags` is mechanically
  different from template wording: the shim treats it as a standing, actively-rechecked condition
  ("if a listed condition occurs while you work, stop" — `guidance-runbook-follow-frame` spec line
  68-76), not a passive example the agent reads once while composing a command. This is the one
  lever never exercised in 6 rounds of the archived eval.

**D3. Re-run the eval's existing route-only probe as the verification step, not a new harness.**
`dev/eval/cumulative/runbook_vs_skill/phase2/probe_phase2.py --task route --model sonnet5 --n 3
--arms R --shim-only --keep` (or its current equivalent) is the same instrument that produced every
prior D8 verdict for route; reusing it keeps this change's verification comparable to the archived
eval's numbers rather than introducing a new, less-comparable measurement.

**D4. Retire `SKILL.md` only after the promoted runbook is confirmed retrievable and matched, AND
`shim.md` is confirmed activated in the real target CLAUDE.md(s).** The retirement is a
**BREAKING** change (route stops being explicitly `Skill`-invokable; it becomes retrieval-dependent).
Sequencing the retirement after verification avoids a window where route is neither an invokable
skill nor a confirmed-working runbook — but "confirmed-working" per the D8 eval only proves the
runbook works inside the eval's own shim-only fixture config, not that any real session would ever
discover it. As of this session, `agent-instructions/guidance/shim.md` — the mechanism that
surfaces runbooks — is deployed but **not activated** in the real `~/.claude/CLAUDE.md` (only
`recall.md`/`delegate.md`/`learn.md` are imported) or any `~/.pi/agent/` equivalent. Retirement
therefore requires BOTH: (a) the D8 bar met, and (b) `shim.md` confirmed activated in the real
target CLAUDE.md(s) this will run under — not merely present in the eval's fixture config.

## Risks / Trade-offs

- **[Risk] The `red_flags` fix does not clear D8 — the miss may be a deeper agent-behavior gap the
  archived eval didn't fully characterize (e.g. models not weighting `red_flags` as strongly as
  inline template text when the two conflict in practice).** → Mitigation: this change's tasks
  explicitly record the re-run's outcome either way (task requires reporting the actual
  `followed_all`/`end_state` numbers, not assuming success); if D8 still isn't met, the retirement
  step (D4) does not proceed and the change reports the finding rather than forcing promotion.
- **[Risk] Retiring `SKILL.md` removes route's explicit `/route`-style invocability, relying entirely
  on retrieval to surface it.** → This was originally (and incorrectly) mitigated by claiming
  retrieval-only discovery already works for recall/learn/write-memory — false: as of this session
  none of the four skills have actually been retired (recall/learn/write-memory's promotion is
  #760's still-open scope). The real, complete mitigation is D4's two-part gate: retirement only
  proceeds once BOTH the D8 bar is met AND `shim.md` is confirmed activated in the real target
  CLAUDE.md(s) — a D8 pass alone, inside the eval's shim-only fixture, does not prove any real
  session would discover the runbook. This change's own D8 bar was NOT met, so retirement (task 5)
  correctly did not proceed — but the gate as originally written (D8 alone) would have retired
  `SKILL.md` and stranded route in every real session had D8 passed while `shim.md` remained
  unactivated. Avoided this round only because D8 failed, not because the gate checked for this —
  captured as vault note `1031a.2026-09-19.skillretirement-gate-needs-shim-activation-precondition`.
- **[Trade-off] This change re-spends eval budget on a already-6-round-deep investigation.** → The
  re-run is scoped to the R arm only, n=3, matching the cheapest prior round's shape (~$5), not a
  full re-run of all arms/rounds.

## Migration Plan

1. Promote `Route-R`'s content into the production vault via `engram learn runbook` (or `engram
   amend` if a route runbook already exists from a prior partial promotion — check first), including
   the new `red_flags` entry (D2). Re-embed; confirm `engram embed status` clean.
2. Verify retrieval: `engram query` with route-shaped phrases surfaces the promoted runbook as a
   top-ranked `kind: runbook` match (mirrors the archived eval's task 3.3 retrieval-check pattern).
3. Re-run the D8 probe (D3). Record `followed_all`/`end_state` against the skill row's 2/3 reference.
4. **If D8 is met** (within one trial of 2/3): retire `agent-instructions/skills/route/SKILL.md`
   (remove or replace with a pointer stub, matching how recall/learn/write-memory's directories were
   handled). Update any deployment manifest that lists route's skill directory.
5. **If D8 is not met**: do not retire SKILL.md. Report the outcome, leaving route on its skill form
   pending a design call (this is a valid, honest outcome for this change — not a blocker to closing
   it, since the promotion + one clean attempt at the known defect is the change's scope; a further
   fix is separately scoped work if needed).
6. No rollback machinery beyond normal git revert / `engram amend` back to prior content — this
   change touches one vault note and one skill directory, no schema or migration-scale surface.

## Open Questions

- Confirmed at design time: no route runbook exists yet in the production vault (only in the
  archived eval's fixture vaults) — step 1 uses `engram learn runbook`, not `engram amend`, unless a
  fresh check at implementation time finds otherwise.
- Should `SKILL.md`'s retirement leave a stub file (pointing agents/humans at the runbook) or be a
  clean removal? No precedent exists yet — recall/learn/write-memory (#760) are still unpromoted
  too, so this change sets the first precedent rather than following one. Default to a clean
  removal unless implementation surfaces a reason (e.g. Claude Code's skill-discovery UI needing an
  entry) to keep a stub; record whichever is chosen so #758/#759/#760 can follow it.
