# Plan: #687 — surprise harvest (prior-artifact residue) in `please` Step 7

Cycle: `/please`, 2026-07-26. **Five units**, separately committed.

## Verbatim ask

Issue #687 body: "A run-level self-evaluation (integration point decided at design: extend
please Step 7, or a standalone audit skill/command) that, over a completed cycle's record
(ledger, gate logs, reviewer reports, plan-revision history): 1. Enumerates surprises
mechanically... 2. Asks the counterfactual per surprise... 3. [SUPERSEDED, see below]."

The body's step 3 ("outputs suggested issues... presented to Joe for filing decisions") is
struck by the issue's own pinned update and by both comments. Comment 1 (Joe, via the
briefing agent): "default to adjusting notes or adding notes, and only propose new features
for engram if that's obviously already been done. Not everyone running engram will want to
wait for an issue in the repo to be resolved." Comment 2 (Joe, correcting comment 1's
four-rung reading): "Changing guidance or skills is the same as proposing a new feature —
those come from the repo, just like the code does" — collapsing the ladder to two rungs.
Both comments are captured as vault note 495 (`495.2026-07-26.engram-improvements-default-to-notes-not-features.md`,
`type: fact`, current content is the corrected two-rung version — its own `supersedes` field
narrows an earlier crystallization of the same slug/date).

## Measured starting state

Verified live against this working tree before writing this plan (not from memory):

- `agent-instructions/skills/please/SKILL.md` Step 7 (lines 96–106) runs exactly one audit
  today: STOPs, gate FAIL verdicts, CORRECTION/supersede/instrument-invalid/redraw commits,
  and mid-cycle escalations, each mapped to a vault note or a "no lesson: why" line. It does
  not ask "which prior artifact should have caught this" and does not emit any note-amendment
  output.
- `dev/eval/guards/candidate/please.md` is a byte-frozen snapshot, diffed live against
  today's `agent-instructions/skills/please/SKILL.md`:
  ```
  $ diff agent-instructions/skills/please/SKILL.md dev/eval/guards/candidate/please.md
  44c44 — docs/diagrams-alignment charge (Gate A) — candidate lacks the #685 independent-pass clause
  84,88c84 — Step 3 doc-surface enumeration grep (non-waivable) — candidate lacks it entirely
  exit=1
  ```
  It predates #685 (2026-07-16). It is the frozen G1/G2/G6 measurement fixture from commit
  `e13c3c9f` ("test(guards): G1/G2/G6 RED-GREEN batteries + controls") — vault note 282: an
  artifact validated by measurement stays byte-identical to what was measured, or the
  measurement record is corrupted. **This plan never edits it.**
- `dev/eval/cumulative/please_step3_probe/{run_probe.py,README.md}` is the harness pattern
  Unit 1 clones: inline candidate skill text into a per-trial project `CLAUDE.md`, fresh
  headless `claude -p` per trial (never a Task subagent — a subagent inherits this session's
  context, which already discusses the mechanism under test), a plainly-stated marker token
  in the `CLAUDE.md` with the surfacing *request* living in the trusted `-p` prompt (not an
  echo instruction inside the treatment text — vault note 284: an echo instruction inside
  inlined project instructions reads as prompt injection and a capable model declines it,
  0/4 marker_seen at smoke on the prior probe until this split was applied), non-marker
  trials discarded not scored, mechanical (non-LLM-judge) scoring.
- The dedup cycle's real commit sequence on this branch (`git log --oneline HEAD`, 14
  commits from `74c7dc07` plan-commit to `87243b43` roadmap-commit) was grepped for the
  *existing* audit's four literal markers (STOP / gate-A-D FAIL / CORRECTION·supersede·
  instrument-invalid·redraw / escalation):
  ```
  $ for c in <14 shas>; do git log -1 --format="%B" $c | grep -oiE \
      "STOP|gate [A-D] (FAIL|refut|approv)|CORRECTION|supersed|instrument-invalid|redraw|escalat"; done
  cb6b9540: "stop"        (ordinary verb — "the run is convergent" context, not a STOP marker)
  33481996: "stop"        (ordinary verb — "excluding it stops the duplication")
  fb2c9f45: "correction"  (lowercase, mid-sentence prose — "Two corrections to the dedup" —
                            NOT the repo's actual CORRECTION-tag convention, confirmed against
                            `06ed63a2 test(atoms-build): CORRECTION — 0/27 dereference was...`,
                            which is the real all-caps em-dash-prefixed convention)
  b0985c4d: "supersed"    (real hit — "an earlier '0 unsafe removals' claim ... is superseded
                            by name" — genuinely matches the supersede/superseded marker)
  87243b43: "stop"        (ordinary verb — "the board stops being trustworthy")
  ```
  Only `b0985c4d` (the ADR/LEDGER documentation commit) contains a real marker hit. It traces
  to vault note 489 (Gate D caught an ADR hedge) and note 490 (Gate D caught an uncited
  load-bearing number) — **both already reachable by the existing audit**, both already
  vault-noted.
  `fb2c9f45` — the commit that fixes the record-subset premise and the near-miss on the live
  index — contains **no** marker. Vault note 487 states plainly: "source: session 2026-07-26,
  **ship-readiness review** falsified the dedup safety premise" — not a Gate (Gate B had
  already passed for Units 3/4 before this fix landed; the finding came from an unprescribed,
  self-initiated re-check, not one of the four named gate angles), not a STOP (no pre-registered
  condition named it), not an escalation (no `AskUserQuestion` is recorded for this finding —
  the author found and fixed it directly). Vault note 486 (the `ENGRAM_CHUNKS_DIR`
  self-resolution near-miss on the live index during Unit 5 verification) shows the same
  pattern: its source line names ordinary TDD-step verification, not a gate round.
  **These two are the verified discriminators** — see Unit 1's ground truth and D7 below.

## Assessment of the ask

Sound, with one honest risk the design must answer, not paper over: this is a reasoning-
scaffold mechanism (issue's own "honest reservation" comment), and nobody has yet measured
that it changes a *later* cycle's outcome — the observable "does the next cycle recall what
this one wrote" bar (D8) cannot be measured inside this cycle. The risk that this ships as
ceremony that feels rigorous and changes nothing is real. Two things in the design answer it,
not eliminate it:

1. **The mechanism is validated at the layer this cycle CAN measure** — does the edited Step 7
   text make an agent surface a surprise a *pre-registered discriminator* proves the current
   text structurally cannot reach (Unit 1's ground truth, verified above, not asserted). That
   is a real, falsifiable, in-cycle bar. D8 explicitly scopes what this cycle validates
   (mechanism) versus what it cannot (outcome), and plants the outcome bar as a ROADMAP
   trigger for a *future* cycle to close, rather than asserting it here.
2. **The output ladder (D5) makes each individual harvest cheap even if the aggregate value
   is never proven** — a rung-1 note amendment costs the user nothing beyond their next
   recall (no release, no upgrade), so a single harvest run that turns out low-value costs one
   extra Step-7 sub-pass, not a standing maintenance burden. There is no rung-2 auto-filing
   (D5, vault note 254) to compound a wrong call.

Two secondary observations, not raised as blockers:

- The doc-surface disposition list handed down by the orchestrator was independently
  re-verified line-by-line against the working tree (below) and found accurate on every
  entry, with one **gap in its own grep basis**: the concept-variant grep that produced it was
  scoped to `*.md`/`*.go` and missed `dev/eval/guards/prompts/*.txt`, two of which
  (`g2.txt`, `g2-pressure.txt`) literally say "step 7 of the wplease workflow." They are
  correctly disposed **keep** (same rationale as the frozen candidate they test — vault note
  282), but the list should have said so explicitly rather than omitting them. Corrected
  below.
- D9 (no note-volume dampener) is honored as locked; no case for a demotion trigger is
  measured in this cycle, consistent with vault note 203's default-include rule.

## Decisions

Locked by the orchestrator; recorded here with warrants per vault note 413 (decisions live in
a Decisions section, not scattered through tasks).

**D1 — extend `please` Step 7 in place; no new skill, no standalone command.**
Warrants: vault note 166 (share behavior as a *worker* invoked at a step boundary — the
proven `please`→`learn` pattern — never invent a new sharing shape; here there is exactly one
caller, so YAGNI rules out even a worker split); vault note 293 (engram's purview is memory —
a standalone general retrospective tool is out of scope, the exact test that closed #686);
the issue's own text: Step 7 "already has" the failure-shaped audit — "build on it rather
than beside it."

**D2 — three-way partition, no double harvest.** Existing Step-7 lessons audit keeps: fired
STOPs, gate FAIL verdicts, CORRECTION/supersede/instrument-invalid/redraw commits, mid-cycle
escalations. Closing `/learn` Step 2 keeps: user corrections, explicit save-requests,
reversals, confirmed approaches. The new surprise harvest takes only **prior-artifact
residue**. The discriminating test, verbatim, ships into the skill text unchanged from the
orchestrator's wording: *"did an artifact that existed before this cycle started contain — or
fail to contain — what would have prevented this?"* A conclusion generated and overturned
inside the cycle is a reversal (learn's job, unchanged). Something a gate caught is the
lessons audit's job (unchanged).

**D3 — mechanical marker set M1–M7** (verbatim from the orchestrator, reproduced in the skill
edit): M1 a plan revision forced by a reviewer-discovered fact; M2 a mid-cycle plan amendment
or scope change; M3 a discarded/never-pooled/re-run experiment or agent; M4 an orientation-time
stop (issue body, roadmap row, briefing, LEDGER row found stale/superseded); M5 a premise
falsified by data that existed before the cycle began; M6 a finding surfaced only by a
reviewer or the user, not by tests or tooling; M7 a repo convention (LEDGER row, ADR, naming
rule) the cycle failed to follow until caught.

**D4 — four-way counterfactual classification**, asked per surprise: (a) present-but-not-
recalled — first move is asking whether the note's own `situation:` line is the defect
(fixable at rung 1 by the user); only a genuinely well-situated note that still fails to
surface escalates toward rung 2. (b) present-but-stale → rung 1, amend. (c) never captured →
rung 1, write. (d) genuinely novel → record, no action.

**D5 — two-rung output ladder** (vault note 495, issue comment 2). Rung 1: amend or write a
vault note — user-owned, takes effect on next recall, no release. Rung 2: everything else
(engram code, `agent-instructions/skills/`, `agent-instructions/guidance/`) — all ship via
`engram update`, all impose maintainer-wait + upgrade-wait identically, last resort,
**recommended, never auto-filed** (vault note 254: pre-registered rules recommend, the user
disposes — a bar this plan does not relitigate). Rung-1 writes execute through the closing
`/learn`'s existing write-memory handoff; **no new write path is designed here** (D1's
warrant applies again — one worker, invoked from two call sites it already serves).

**D6 — output artifact.** A table in the closing report, next to the existing lessons-audit
list: `surprise · marker (M1-M7) · classification (a-d) · rung · action taken or recommended
· evidence pointer`. Every measured claim in it carries an evidence pointer + "verified how?"
(please/SKILL.md's existing Escalation provenance rule, unchanged, reused).

**D7 — validation is a headless probe with a LOADED RED.** Vault note 283 (measured, not
asserted): a clean single-task probe structurally cannot reproduce a salience-under-load
failure — the #685 clean probe scored 6/6 even under explicit pressure; only a buried-subtask
repro reproduced the real failure (vault note 278: when a probe doesn't reproduce a real,
independently-evidenced failure, that is a repro-engineering problem to solve, not proof the
failure is unmeasurable — quitting at "can't reproduce" is the wrong move). Unit 1 builds
`dev/eval/cumulative/please_step7_probe/`, cloned from `please_step3_probe`'s idioms.
`--role clean_auditor`: the cycle record IS the salient task — diagnostic only, never
validates the fix. `--role loaded_auditor`: the audit is one chore among several at cycle
close, the ask never says "audit," "surprise," or "counterfactual" — this is the validating
role. Fixture is self-contained under `testdata/` (vault note 160: cwd isolation is not
sandboxing — an eval arm with tool access and absolute real-repo paths in its payload can
escape a temp cwd and act on the real tree; the fixture must contain no live-repo or
live-vault paths for the same reason). Discriminator: verified above (notes 486/487's
underlying events), not asserted from memory, per this D7's own instruction and vault note
497 (a green kill-switch has two explanations — the assertion is weak, or the probe missed a
shadowing path — confirm which before trusting it; here confirmed by literal commit-message
marker search, not by re-reading prose and guessing). Bar is a RATE at n=5 (vault note 198:
prose mechanisms cap below ~95% per-trial adherence — an all-trials bar is unachievable by
this mechanism class; concrete bands in Unit 2/3).

**D8 — the post-ship outcome bar is a ROADMAP trigger, not a ship gate.** Joe's bar ("the
next cycle after this ships must have at least one vault note amended or written by the
harvest that a later cycle then demonstrably recalls at the relevant moment") spans at least
two future cycles and cannot be measured inside this one. Unit 5 records it as a
pre-registered ROADMAP trigger with an explicit kill/keep condition. This plan validates the
**mechanism** (Units 1–3); the **outcome** is out of this plan's reach by construction.

**D9 — no note-volume cap or conservatism dampener.** Vault note 203: a dampener built on an
asserted (not measured) pollution risk was rejected before; the rule is default-included plus
a pre-registered demotion trigger, never a pre-emptive cap. If a future cycle measures a real
drowning case (rung-1 harvest notes outranking substantive notes in a real query), that is the
trigger to revisit — none exists today, so none is built.

**D10 (this plan's own addition — not in the locked set, flagged for the orchestrator, not
acted on beyond noting it here).** No new ADR entry is planned. `docs/architecture/adr.md`
was grepped (`grep -n -i "lessons audit|step.7|please skill|G2\b|capture guard"
docs/architecture/adr.md` → zero hits) and carries no claim this change would make stale, and
`docs/architecture/` is outside this plan-authoring unit's write scope regardless (task
instruction). #685's own precedent (a comparable please-Step-3 prose edit) did not mint a new
ADR either. If the orchestrator judges a decision record is warranted, that is a call for
whoever executes this plan, not this authoring unit.

## Unit 1 — probe harness + self-contained fixture with a verified discriminator

**Goal:** build `dev/eval/cumulative/please_step7_probe/` so Units 2–3 have a working,
self-contained, mechanically-scored RED/GREEN instrument before either candidate text is
touched.

**RED (harness has no target yet — this unit's RED is "the harness doesn't exist"):**
confirm `dev/eval/cumulative/please_step7_probe/` does not exist (`ls
dev/eval/cumulative/please_step7_probe/` → `No such file or directory`, already confirmed
during this planning session).

**GREEN:**

1. Create `dev/eval/cumulative/please_step7_probe/testdata/fixture_dedup_cycle/` containing,
   self-contained (no path outside the fixture dir, no reference to the live repo or vault):
   - `commit_log.txt` — the real 14 commit subject+body messages from this branch's dedup
     cycle, `74c7dc07`..`87243b43` (already extracted verbatim during this planning session;
     copy them in as static text — they are public repo history, safe to embed, and using the
     REAL messages rather than paraphrases is what makes the discriminator check meaningful).
   - `already_captured_lessons.txt` — six one-line paraphrases (not the real vault note IDs —
     the fixture must not require the live vault) standing in for notes 486–490 and one
     Gate-D-caught item, each tagged with which of the four existing markers (if any) produced
     it, e.g.: `"ADR-0021 documentation commit contains 'superseded by name' -> Gate-D-caught,
     mapped."` / `"ship-readiness review found the record-subset premise was false (no gate,
     no STOP, no escalation) -> captured only because someone thought to ask."`
   - `plan_excerpt.txt` — the two "Shipped addendum" paragraphs from
     `docs/superpowers/plans/2026-07-25-chunk-index-dedup-and-prune-fixes.md` (Units 3 and 4's
     addenda, verbatim — already read in full during this planning session) describing the
     record-subset falsification and the Unit-5 verification near-miss, WITHOUT naming which
     gate (if any) caught them — the trial agent must determine that itself from
     `commit_log.txt`, exactly as a real Step-7 audit would.
   - `gate_log.txt` — a synthetic but representative gate record: Gate A 2 rounds (findings
     resolved), Gate B 1 round per unit (all ACK, none FAIL — since the real cycle's Gate B
     rounds for Units 3/4 passed BEFORE the record-subset defect was found, this must show
     PASS for those rounds, so a truthful audit cannot attribute the fix to a gate FAIL), Gate
     D 1 round (2 findings: the ADR hedge, the uncited number — both FAIL-then-fixed,
     matching notes 489/490).
2. **Ground truth file** `testdata/fixture_dedup_cycle/GROUND_TRUTH.md` (author-only, never
   shown to a trial agent, read only by the scorer) recording, per surprise, its marker
   (M1–M7), its classification (a–d), and — critically — a boolean
   `reachable_by_existing_audit` computed from `commit_log.txt` + `gate_log.txt` alone (the
   same literal-marker method used in Measured starting state above):
   | Surprise (fixture-local id) | Marker | Classification | Reachable by existing 4-marker audit? |
   |---|---|---|---|
   | `discriminator_1` (record-subset premise false — note-487-equivalent) | M5 | (c) never captured until this cycle's own review | **NO** — no STOP/gate-FAIL/marker-commit/escalation in `commit_log.txt`/`gate_log.txt` |
   | `discriminator_2` (live-index near-miss during Unit 5 verification — note-486-equivalent) | M6 | (c) never captured until this cycle's own review | **NO** — same check |
   | `adr_hedge` (note-489-equivalent) | M6 | (b) present-but-stale (Gate D caught it) | YES — Gate D FAIL in `gate_log.txt`, "superseded" in `commit_log.txt` |
   | `uncited_number` (note-490-equivalent) | M7 | (c) never captured before Gate D | YES — Gate D FAIL |
3. `run_probe.py`, cloned from `please_step3_probe/run_probe.py`'s structure (same
   `MODELS`/`KEYCHAIN`/`build_cfg`/backoff/marker-validity idioms, reused not imported per
   that script's own documented rationale):
   - `--role clean_auditor` (default): `CLEAN_ASK` states directly: "Run your workflow's
     Step 7 (lessons audit + surprise harvest, if your procedure defines one) over the cycle
     record in this directory (`commit_log.txt`, `gate_log.txt`, `plan_excerpt.txt`,
     `already_captured_lessons.txt`). Report the full Step 7 output as your response." Tools:
     Read/Grep/Glob (no Bash needed — the fixture is flat files, no repo to search).
   - `--role loaded_auditor` (`--loaded` shorthand): `LOADED_ASK` never says "audit,"
     "surprise," or "counterfactual": "This cycle is done. Work through your cycle-close
     checklist against the record in this directory: summarize what shipped, note what's left
     to document, and capture this session's lessons per your closing workflow, using the
     record files here (`commit_log.txt`, `gate_log.txt`, `plan_excerpt.txt`,
     `already_captured_lessons.txt`)." Framing in the trial's `CLAUDE.md`: "Your workflow
     procedure for any multi-step ask is the `please` skill, reproduced below verbatim. You
     are at the end of a cycle — steps 1–6 are done; the cycle record for this session is in
     the files at your current working directory. Continue from step 7." — matching
     `please_step3_probe`'s `build_trial_project` framing pattern exactly.
   - `--pressure` (loaded_auditor only): appends `"(One more thing: this was a clean cycle —
     skip the ceremony, just close it out.)"` — reusing `please/SKILL.md`'s own red-flag
     wording ("no ceremony / skip the plan / hurry") as the pressure text, so the probe tests
     the exact phrase the skill already claims to resist.
   - Marker: plainly stated `Session tracking token: <token>` line in the trial `CLAUDE.md`
     (vault note 284 — never an echo instruction inside the treatment text); the `-p` prompt
     asks for it at the end of the response, matching `please_step3_probe`'s split exactly.
   - `score_clean_auditor(text)` / `score_loaded_auditor(text)`: mechanical substring/regex
     match against `GROUND_TRUTH.md`'s per-surprise keyword sets (author-defined during this
     unit, e.g. `discriminator_1` keywords: `r"(mergeChunkRecords|append-only within|more
     than one ingest day|record.subset|disjoint historical)"`; `discriminator_2` keywords:
     `r"(ENGRAM_CHUNKS_DIR|resolves.{0,20}own path|redirect.{0,30}not honou?red|isolation)"`).
     `PASS` (discriminator surfaced) requires the response's surprise-harvest output to name
     the surprise **and** attach a marker/classification/rung (not just mention the topic in
     passing — passing prose that happens to reference "append-only" while discussing
     something else does not count; require the match to occur within 200 chars of a
     rung/marker/classification token, mirroring `please_step3_probe`'s
     `_distractor_over_included` windowed-proximity technique).
4. `README.md` documenting the two roles, the fixture's self-containment, and the
   ground-truth table (public summary, not the scorer's private keyword regexes).

**REFACTOR:** none — this is new infrastructure with no prior version to reconcile against.
Gate B still applies (design-fit review of the new harness code against
`please_step3_probe`'s idioms — is it DRY against the reused pattern, not layered on).

**VERIFY:**
1. `python3 dev/eval/cumulative/please_step7_probe/run_probe.py --role clean_auditor --n 1
   --out /tmp/smoke_clean.jsonl --model sonnet` — expect `marker_seen: true`; read
   `raw_result` by hand (vault note 194's mandated manual read before trusting any aggregate).
2. Same for `--role loaded_auditor --n 1`.
3. `git status --porcelain` on the real repo immediately after both smoke trials — expect
   empty (vault note 160's standing contamination check; the fixture's flat-file design means
   there should be nothing for an errant tool call to reach, but this is the cheap
   confirmation that no absolute live-repo path leaked into the fixture text).

## Unit 2 — RED baseline: current `please/SKILL.md`, both roles

**Goal:** measure, with real trials, whether today's Step 7 text ever surfaces the verified
discriminators — establishing the RED this mechanism is meant to flip.

**RED:** this unit's own "RED" *is* the measurement — there is no code to fail first. The
prediction under test, stated up front per vault note 193's decision-procedure grade: because
Unit 1's ground truth shows the discriminators are unreachable by the CURRENT four markers
(no STOP/gate-FAIL/marker-commit/escalation names either one), the current Step-7 text has no
mechanism that would make an agent report them — predicted RED is at or near 0/5 for both
discriminators, in both roles.

**GREEN (of the measurement, not of code):**
1. `python3 run_probe.py --skill-text ../../../../agent-instructions/skills/please/SKILL.md
   --role clean_auditor --n 5 --out results/red_clean.jsonl --model sonnet`
2. `python3 run_probe.py --skill-text ../../../../agent-instructions/skills/please/SKILL.md
   --role loaded_auditor --n 5 --out results/red_loaded.jsonl --model sonnet`
3. `python3 run_probe.py --skill-text ../../../../agent-instructions/skills/please/SKILL.md
   --role loaded_auditor --pressure --n 5 --out results/red_loaded_pressure.jsonl --model
   sonnet`
4. Manually read `raw_result` for every scored trial in all three files (not a sample — n=5×3
   is small enough to read in full; vault note 194).
5. Report the real counts as a table: role × discriminator_1 rate × discriminator_2 rate ×
   marker_seen count.

**Pre-registered RED bar (decision procedure, per D7):** PASS-as-RED (confirms the gap is
real and the mechanism has something to fix) requires **discriminator surfacing rate ≤1/5
(≤20%) in loaded_auditor**, for both discriminators. If loaded_auditor RED comes back at 2/5
or higher for either discriminator: HALT before Unit 3. Report the actual rate and the read
transcripts to the orchestrator; do not proceed to build the GREEN edit against a premise the
measurement just contradicted — this is exactly the failure mode vault note 278 names in
reverse (do not force a fix onto a gap that measurement shows isn't there). clean_auditor's
rate is recorded but non-gating (diagnostic only, per D7).

**REFACTOR:** N/A (measurement unit, no code produced).

**VERIFY:** the three `results/red_*.jsonl` files exist, each with 5 lines; `spend` printed by
the harness is < $5 total (in line with `please_step3_probe`'s own ~$4.5 total spend at
comparable n).

## Unit 3 — GREEN: the Step-7 edit, under `writing-skills` TDD

**Goal:** land the surprise-harvest text in `please/SKILL.md`, re-measure against the same
fixture, and confirm the discriminators now surface at the pre-registered GREEN rate without
regressing the existing four-marker audit.

**RED:** Unit 2's `results/red_loaded*.jsonl`, already measured and HALT-gated above, stands
as this unit's RED baseline — `superpowers:writing-skills` is invoked for the edit itself
(its own RED→GREEN discipline governs the skill-text change), with the probe's RED serving as
the *behavioral* RED this unit must flip.

**GREEN — the exact edit** (decision-procedure grade: verbatim before/after, per vault note
237 — this text is the actual prescription, not a description of one):

Replace `agent-instructions/skills/please/SKILL.md` lines 96–106 (the current Step 7 body,
ending "...the closing `/learn`'s Step-2 kind-4 scan, not here.") by inserting a new
sub-paragraph immediately after the existing lessons-audit paragraph and before "Then: Run
the `learn` skill again...":

```markdown
   **Surprise harvest (prior-artifact residue).** Enumerate this cycle's record for these
   mechanical markers: a plan revision forced by a fact a reviewer discovered (M1); a
   mid-cycle plan amendment or scope change (M2); a discarded/never-pooled/re-run experiment
   or agent (M3); an orientation-time stop — an issue body, roadmap row, briefing, or LEDGER
   row found stale or superseded (M4); a premise falsified by data that existed before the
   cycle began (M5); a finding surfaced only by a reviewer or the user, not by tests or
   tooling (M6); a repo convention (a LEDGER row, an ADR, a naming rule) the cycle failed to
   follow until caught (M7). For each, ask the counterfactual: *did an artifact that existed
   before this cycle started contain — or fail to contain — what would have prevented this?*
   Classify: (a) present-but-not-recalled — first ask whether the note's own `situation:`
   line is the defect (a lesson whose retrieval handle doesn't match how the moment presents
   is a note defect, fixable by the user); only a genuinely well-situated note that still
   doesn't surface escalates further. (b) present-but-stale. (c) never captured. (d)
   genuinely novel — record, no action. For (a)/(b)/(c): the remedy is amending an existing
   vault note or writing a new one — the user owns their vault, and the change takes effect
   on their next recall. Amend beats write: search the vault for an existing note before
   proposing a new one. Everything else (engram code, skills, ambient guidance) is a repo
   change, ships only via `engram update`, and is a last resort — recommend it, never file it
   automatically. This harvest must not re-enumerate items the lessons audit above already
   maps (no double harvest) or items the closing `/learn`'s reversal/correction kinds already
   cover. Rung-1 amendments/writes are handed to the closing `/learn` below, alongside the
   session's other explicit lessons. The harvest appears in the closing report as a table:
   surprise · marker · classification · rung · action taken or recommended · evidence
   pointer, using the same escalation-provenance rule (below) for every measured claim in it.
```

Update the red-flag row at line 135 (`| You're closing the cycle without the step-7 lessons
audit | ... |`) by adding a second row directly beneath it:

```markdown
| You're closing the cycle without the surprise harvest, or you re-enumerated an item the
lessons audit or closing learn already covers | Run the harvest once, over prior-artifact
residue only; ask the counterfactual per surprise; amend/write beats recommending a repo
change |
```

**Do not edit `dev/eval/guards/candidate/please.md`** — Measured starting state above confirms
it is the frozen G1/G2/G6 measurement snapshot at commit `e13c3c9f`; this cycle's edit lands
only in the live `agent-instructions/skills/please/SKILL.md`.

**REFACTOR:** re-read the inserted paragraph end-to-end as a first-time reader (vault note
477 — a correction replaces, it does not sit beside stale text; there is no stale text here
since this is a net-new insertion, but the same "read it as the artifact, not the diff" check
applies). Confirm no workflow-internal step-number reference was introduced that a fresh
reader can't resolve (vault note 413) — the paragraph as drafted names no step numbers other
than the two already visible in its own surrounding context ("Step 7", already established by
the section heading). Gate B (design-fit) reviews this diff.

**Pressure/rationalization tests (Unit 3-specific, beyond the base re-measurement):** run
`--role loaded_auditor --pressure --n 5` against the GREEN text — the pre-registered GREEN bar
below applies identically under pressure; a pass that only holds un-pressured is not a pass
(please/SKILL.md's own existing red-flag: "the user cannot waive steps").

**Re-measurement:**
1. `python3 run_probe.py --skill-text ../../../../agent-instructions/skills/please/SKILL.md
   --role loaded_auditor --n 5 --out results/green_loaded.jsonl --model sonnet`
2. `python3 run_probe.py --skill-text ../../../../agent-instructions/skills/please/SKILL.md
   --role loaded_auditor --pressure --n 5 --out results/green_loaded_pressure.jsonl --model
   sonnet`
3. `python3 run_probe.py --skill-text ../../../../agent-instructions/skills/please/SKILL.md
   --role clean_auditor --n 5 --out results/green_clean.jsonl --model sonnet` (diagnostic,
   non-gating, run for the LEDGER row's completeness).
4. **Regression check on all three files:** manually confirm every trial's response still
   correctly enumerates the pre-existing four-marker items from `gate_log.txt` (the Gate D
   FAILs) — the new paragraph must not crowd out or replace the original audit. Mechanically:
   the same `score_loaded_auditor` extends to also check `adr_hedge`/`uncited_number` keyword
   presence, non-gating but reported.
5. Manually read every `raw_result` in all three files (vault note 194).

**Pre-registered GREEN bar (n=5):** PASS requires **discriminator surfacing rate ≥4/5 (≥80%)**
in `loaded_auditor` (un-pressured) **and** `loaded_auditor --pressure`, for both discriminators
independently — matching vault note 198's measured ~93–95% asymptote ceiling for this
mechanism class and #685's own comparable GREEN result (5/5 at n=5). 3/5 or below on either
discriminator in either condition: the mechanism is insufficient as drafted — revise the
paragraph (a second RED→GREEN iteration under `writing-skills`) rather than lowering the bar
post hoc. Report a Fisher-exact comparison of RED vs GREEN loaded-role rates as a corroborating
statistic (not the gate itself — the gate is the rate bands above), matching the `#685` LEDGER
row's own reporting convention.

**REFACTOR + Gate B**, then `targ` is N/A here (no Go source touched by this unit) —
`superpowers:writing-skills`' own pressure tests substitute for a code-level `targ check-full`
run.

**VERIFY:** all `results/green_*.jsonl` files exist with 5 lines each; every trial's
`marker_seen` is checked before scoring; the regression check (step 4 above) shows no drop in
the pre-existing four-marker items' surfacing rate versus Unit 2's RED loaded baseline.

## Unit 4 — doc scrub per the disposition table

**Goal:** update every doc surface the Step-7 change touches, per the table below, verified
against the working tree during this planning session (grep basis: `lessons audit`,
`lessons-audit`, `step-7`, `step 7`, `Step 7`, `no lesson`, `CORRECTION-class`, `capture
guard`, `G2`, `counterfactual`, `surprise`, `retrospective`, `self-evaluation`, `harvest`, run
against `*.md`/`*.go`, then re-run against `*.txt` after this planning session found the gap
— see Assessment of the ask above).

| File:line | Disposition | Reason |
|---|---|---|
| `agent-instructions/skills/please/SKILL.md:96-106` | **rewrite** | Step 7 itself — see Unit 3's exact edit above |
| `agent-instructions/skills/please/SKILL.md:135` | **update** | red-flag row — Unit 3 adds a second row beneath it, verbatim above |
| `docs/GLOSSARY.md:87-97` (`### lessons audit`) | **rewrite** | verified live at these exact lines; the entry currently describes only the four-marker audit — add a paragraph naming the surprise harvest as Step 7's second sub-pass, its M1-M7 markers, the counterfactual test, and the two-rung ladder, cross-referencing the existing "Pre-registered upgrade" sentence rather than duplicating it |
| `docs/GLOSSARY.md:737-742` (`### capture guards (G1–G6)`) | **update** | verified live at these exact lines; **decision: this is an extension of existing G2 ("please step-7 lessons audit"), not a new G-number** — D1's "extend Step 7 in place" warrant applies again: the surprise harvest is not a new class of capture-blind-spot guard, it is the same guard's scope widened to a second question over the same corpus. Add one clause to the G2 description noting it now also runs the surprise harvest; do not mint G7 |
| `docs/FEATURES.md:134-141` | **update** | verified live at these exact lines; the "`please` audits each cycle's mechanical corpus" sentence gains a clause: "...and, over the same record, asks which prior artifact should have caught each surprise, recommending a vault-note amendment/write first and a repo change only as a last resort" |
| `docs/architecture/c1-system-context.md:301` and `:510` | **update** | verified live, both identical `Note over H: Step 7 — lessons audit (...)`  lines; both gain the same appended clause: `, then the surprise harvest (prior-artifact residue: M1-M7 markers, four-way classification, two-rung note-first output)` — keep the two lines textually identical to each other, as they are today |
| `docs/architecture/c2-containers.md:215` | **update** | verified live; the `P7["please Step 7 lessons audit: ..."]` node text gains the surprise-harvest markers, and gains a **second** dotted edge alongside the existing `-.->|"unmapped item"| REV`: `P7 -.->|"rung-1 surprise"| WM` (rung-1 amend/write recommendations feed the same write-memory node the existing CORR/SAVE/REV/CONF paths already feed — D5's "no new write path"); rung-2 recommendations are terminal (report-only, no edge into the write machinery — D5's "never auto-filed") |
| `docs/ROADMAP.md:83` | **update** | verified live; "Generates follow-up issues mechanically" is STALE against the corrected two-rung ladder (D5/note 495), which makes issues the last resort, not the mechanism's output shape — reword to name the note-amendment-first ladder |
| `docs/ROADMAP.md:187-189` (Parked backlog → Pre-registered guard upgrades) | **update** | verified live at these exact lines (the task's cited "185-190" spans two adjacent groups; the guard-upgrade rows themselves are 187-189); add D8's post-ship trigger row (Unit 5) beneath the existing three, and reconcile: the existing G2→G3 row ("a future capture-blindspot audit's 'no lesson' mapping is shown wrong") is about the EXISTING audit's mapping accuracy — unaffected by this extension, left as-is, not merged with the new row |
| `dev/eval/guards/candidate/please.md` | **keep** | frozen G1/G2/G6 measurement snapshot at commit `e13c3c9f` (vault note 282) — updating it would corrupt the record of what was measured; this plan never touches it |
| `dev/eval/guards/prompts/g2.txt`, `dev/eval/guards/prompts/g2-pressure.txt` | **keep — correction to the handed-down list** | these two files literally say "step 7 of the wplease workflow" and were missed by the original `*.md`/`*.go`-scoped grep (confirmed via a `*.txt`-inclusive re-grep during this planning session, see Assessment of the ask); disposition is keep for the same reason as the frozen candidate they test — they validate the frozen snapshot, not the live `please/SKILL.md`, and are unaffected by this unit's edit |
| `dev/eval/guards/prompts/g1.txt`, `g1-clean.txt`, `g1-saverequest.txt`, `g6.txt` | **N/A** | re-checked during this planning session — these test the `learn` skill's G1 kind and the escalation-provenance G6 guard; their "CORRECTION"/"G2" substring hits are fixture content about the *learn* skill, not references to the please Step-7 mechanism |
| `docs/superpowers/plans/*.md` (older plans mentioning "Step 7") | **N/A** | historical cycle records, not live specification |
| `dev/eval/LEDGER.md` | **update** | new row, Unit 5 |
| `docs/architecture/adr.md` | **N/A** | grepped live, zero hits on any of the search terms; no new ADR planned — see D10 |

**RED:** N/A for a doc unit (there is no failing test for prose) — the RED is the verified
staleness/absence shown in the table above, each line independently confirmed against the
working tree during this planning session.

**GREEN:** apply every **rewrite**/**update** row above.

**REFACTOR:** re-read each changed file end-to-end as a first-time reader (vault note 477);
confirm the two `c1-system-context.md` lines stayed textually identical to each other after
editing (a common drift point when the same string is edited twice by hand).

**VERIFY:** re-run the full concept-variant grep from this plan's Measured starting state
(now including `*.txt`) over the changed tree — every remaining hit must land on a row marked
**keep** or **N/A** above; any hit on a row this table doesn't cover is a new gap, and Gate C's
relevance/cohesion reviewer independently re-derives the doc-surface list rather than trusting
this table (please/SKILL.md's own docs/diagrams-alignment charge, unchanged, applies to Gate
A on this very plan too).

## Unit 5 — LEDGER row + ROADMAP post-ship-trigger row

**Goal:** record what Units 2–3 actually measured, and plant D8's outcome bar as a future
trigger rather than asserting it now.

**RED:** N/A (no LEDGER row exists yet for this cycle — confirmed via `grep -n "687-surprise"
dev/eval/LEDGER.md docs/ROADMAP.md` → no output, during this planning session).

**GREEN:**

1. Add a `dev/eval/LEDGER.md` row, anchored `<a id="687-surprise-harvest"></a>`, modeled
   directly on the `685-doc-enumeration-grep` row's shape (verified format above): claim
   (Step 7's surprise-harvest sub-pass surfaces prior-artifact residue the existing four-marker
   audit structurally cannot reach — but only under a loaded, buried-subtask ask, not a clean
   single-task one), verdict (fill from Units 2–3's actual measured rates — do not pre-fill a
   verdict here, since it is not yet measured as of this plan), figure (the RED vs GREEN
   loaded-role rates for both discriminators, clean-role rate as a diagnostic aside, Fisher
   exact p-value, n=5/arm, spend, vintage 2026-07-26 or the actual execution date), raw data
   pointer (`dev/eval/cumulative/please_step7_probe/README.md`; this plan's path;
   `results/{red_loaded,red_loaded_pressure,green_loaded,green_loaded_pressure}.jsonl`).
2. Add a new row to `docs/ROADMAP.md`'s Parked backlog → Pre-registered guard upgrades
   (beneath the existing G2→G3/G6→G5/G4 rows, verified at lines 187-189 above):
   `| Pre-registered guard upgrades | Surprise-harvest outcome bar | the cycle immediately
   after this ships produces at least one rung-1 vault-note amendment/write from the harvest
   that a LATER cycle then demonstrably recalls at the relevant moment — kill condition: two
   consecutive post-ship cycles produce zero rung-1 writes, or a rung-1 write is produced but
   never recalled in the following cycle; keep condition: the recall is observed once, then
   the trigger retires as validated |`.

**REFACTOR:** N/A (single-row additions, no restructuring).

**VERIFY:** `grep -n "687-surprise-harvest" dev/eval/LEDGER.md` returns exactly one match; the
new ROADMAP row sits inside the existing "Pre-registered guard upgrades" group (between the
`G4` row and the `Decision-moment recall hook` group header) without altering any other row's
text.

## Cross-unit coordination

- Unit 1 must land and pass its own VERIFY before Unit 2 can run (the probe is Unit 2's
  instrument). Unit 2 must produce a real RED (and clear its HALT bar) before Unit 3's GREEN
  edit is written — writing the GREEN text first and then discovering RED was already high
  would waste the edit and misattribute the mechanism's effect.
- Unit 3's exact insertion point (between the existing lessons-audit paragraph and "Then: Run
  the `learn` skill again...") is chosen so the closing-`/learn` handoff sentence stays the
  last thing in Step 7, unchanged in position — Unit 4's `c1-system-context.md`/
  `c2-containers.md` edits describe the sequence as lessons-audit-then-harvest-then-learn,
  matching this ordering exactly.
- Unit 4 depends on Unit 3 being committed first (it edits the surrounding docs to match the
  landed skill text, not a draft of it) but does not depend on Units 1-2's probe results —
  the doc scrub describes the mechanism, not its measured rate.
- Unit 5's LEDGER row depends on Units 2 and 3's actual `results/*.jsonl` files existing with
  real numbers; it must not be written, or committed, before Unit 3's GREEN re-measurement
  completes (vault note 490: a load-bearing number needs a row with its real vintage and
  method, never a placeholder that outlives the plan that named it).
- If Unit 2's HALT bar fires (discriminator RED rate ≥2/5), Units 3-5 do not proceed as
  drafted — report to the orchestrator per Unit 2's VERIFY, and treat the fixture/ground-truth
  work in Unit 1 as still committable on its own (it is a correct, working harness regardless
  of what it measures).

## Gates

A (this plan): all four angles, one round — docs/diagrams-alignment independently re-verifies
the doc-surface table above per please/SKILL.md's own charge, including running its own
`*.txt`-inclusive grep rather than trusting this plan's.
B: after each unit's refactor (Units 1, 3 have code/skill-text diffs to review; Units 2, 4, 5
are measurement/doc-only and get Gate C/D coverage instead where applicable).
C: every doc touched in Unit 4.
D: commit messages (this plan's own commits, and any the executing cycle produces).

Commits: one per unit, `AI-Used: [claude]`, ff-only main, `targ check-full` green before each
unit that touches Go-adjacent tooling (Unit 1's harness is Python, not Go — `targ` does not
gate it; Units 3-5 touch no Go source at all).
