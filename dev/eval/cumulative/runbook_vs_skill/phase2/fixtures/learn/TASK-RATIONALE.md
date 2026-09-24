# learn task: why this ask, and how it forces the write-memory fetch-before-each-compose gate (tasks 1.1-1.3)

## The ask (`task-prompt.txt`)

One session, one task-prompt, that deterministically reaches learn's two easiest-to-force capture
kinds plus the write-memory handoff validity gate design.md D5 requires (proposal.md's own Goals:
"at least one of each: a correction-kind capture, a save-request-kind capture, and the
write-memory handoff"):

1. "Don't jump to raising the fermentation temperature... we made that exact mistake with batch #9
   last month... the fix is to check airlock/krausen activity... before touching the temperature at
   all" -- a stated correction (wrong-behavior + right-behavior + reason, the same shape as the
   skill's own kind-1 example: "don't suppress lint warnings -- fix the underlying issue") ->
   **kind=feedback**, `learn/SKILL.md:116-122`.
2. "Also: remember for next time, tagged under the sanitation category, that Campden tablets need a
   full 24 hours..." -- a verbatim explicit save-request, matching the skill's own unconditional
   rule ("An explicit save-request ALWAYS gets its note, immediately") -> **kind=fact**,
   `learn/SKILL.md:124-130`.

**Why a single-shot prompt can carry a "mid-task correction" at all:** like every task in this
harness, the trial is one autonomous `claude -p` turn (`probe_phase2.run_one_trial_phase2` calls
`p1.spawn_claude` exactly once per trial -- confirmed by reading the harness, not assumed) --
there is no live back-and-forth to script a real "user interrupts and corrects" moment. Write-
memory's own TASK-RATIONALE.md reached the same wall and deliberately avoided kind-1/kind-3
entirely for that reason. This task instead states the correction as already-having-happened
within the single prompt (the shape "don't do X — we already learned that's wrong, do Y instead,
because Z"), which is the same imperative shape as the skill's own written kind-1 example. This is
the pragmatic, honestly-documented substitute for a live correction, not a literal mid-conversation
interrupt -- flagged here the same way write-memory's "Known limit" section flagged its own
tag-flag divergence's residual gap.

**Why no `recall` involvement:** unlike write-memory's task (which needed recall's absent-cluster
judgment to reach one of its two write-memory call sites), `learn` fires directly off its own
description's trigger language ("remember this", "note for next time", "the moment ... a user
correction confirms a lesson") without needing a prior `recall` pass to decide anything is new.
Keeping `recall` entirely out of this fixture (not deployed at all, unlike write-memory's fixture
which deploys recall+learn unconditionally as fixed background) keeps the task's scoring surface
limited to what THIS conversion (`learn`) actually needs to prove, per design.md's own Non-Goal
("Converting `recall`... is its own future change").

## Domain: home-brewing club (fictional, new)

`vault-template/` is a small fictional home-brewing club vault -- distinct from curate's beekeeping
vault AND write-memory's 3D-printer-farm vault, per note 996's contamination guard and the task's
own instruction to pick a new domain. Six seed notes (`build_vault_template.sh`, real `engram
learn`, real sidecars/embeddings), each carrying a `<family>/<value>` categorical tag
(`process/mashing`, `ingredient/hops`, `ingredient/yeast`, `equipment/kegging`,
`sanitation/equipment`, `equipment/fermenter`) -- an established tagging convention for the agent's
own new writes to follow, exactly mirroring write-memory's `component/<x>` nudge. None of the six
covers an airlock/krausen false-stall reading or Campden-tablet dissipation timing, so both new
writes are genuinely new topics, not overlapping an existing note.

## The write-memory runbook is a REAL vault note copied into the fixture

Unlike write-memory's own conversion (which had to invent a temporary fixture basename because
write-memory wasn't yet promoted), `learn/SKILL.md`'s six "REQUIRED NEXT ACTION" call sites already
name the REAL production basename directly: `1053.2026-09-22.write-memory-compose-execute-verify`
(design D2 — confirmed by reading the live skill file, not assumed). For `engram show
1053...` to resolve inside a fixture-only trial vault (no real-vault copy for `vault_template`
tasks — confirmed by reading `probe_phase2.setup_trial_vault`), that exact note (md + sidecar) must
physically exist in `vault-template/`. `build_vault_template.sh` copies it byte-for-byte from the
real vault (`$HOME/.local/share/engram/vault`, overridable via `$ENGRAM_REAL_VAULT`) as a read-only
reference every arm carries unmodified -- verified in `done_when_checks.sh` (byte-identical check)
and every mutant list below.

**Discovered while validating (not assumed):** with note 1053 present, `engram learn ... --position
top` numbers new notes AFTER the vault's highest existing Luhmann id (1054, 1055, ...), not after
the fixture's own local max (6) the way write-memory/curate's notes 7/8 did. `done_when_checks.sh`
therefore identifies the two new notes by CONTENT (grep for `airlock|krausen` / `campden`) rather
than an assumed id — a hardcoded "7"/"8" would have been silently wrong here (caught by actually
running the hand-built ideal state through the real binary, per note 1017a/955's "verify against
the real artifact, don't assume the reading").

## Luhmann placement: always `top`, by design, not by accident

Neither write reads or cites any in-session note (learn does not perform an `engram query` over
the vault the way `recall` does), so per the skill's own placement rule ("No in-session candidate
note exists → always `position=top`; do not search the rest of the vault for a placement target"),
both writes are `--position top` with no `--target`. This task deliberately does NOT seed an
in-session continuation/sibling scenario (the task instructions left this as the author's call) —
testing `continuation`/`sibling` needs an in-session note to react to, which only `recall`-adjacent
tasks or a second learn-only fixture would naturally provide; out of scope for this representative
task per design.md's Non-Goals (only the four-kind scan and the write-memory handoff are the point
here).

## The write-memory fetch-before-EACH-compose validity gate (`steps.json`)

This is the primary thing this task must be able to score (proposal.md's own framing): does the
agent actually run `engram show 1053...` before EACH of the two composes, or does it fetch once and
reuse the content from memory for the second write? The skill's own text restates "REQUIRED NEXT
ACTION: run `engram show 1053...`" verbatim at BOTH the kind-1 (line 119) and kind-2 (line 127)
call sites — a literal-following agent re-reads and re-fetches at each site, regardless of whether
it does one learn invocation handling both kinds or two separate invocations (mid-cycle then
closing). `steps.json` steps 2/7 score this as TWO ordering-gated fetches, not one: step 7 requires
a fresh `engram show 1053...` occurrence strictly AFTER step 3's compose (`"after": 3`), so a
transcript that fetches once and composes twice fails step 7 (and, by `evaluate_steps`'s own
documented "unmatched referenced step -> dependent step is FALSE" rule, step 8 cascades to FALSE
too) — validated below against a synthetic transcript.

Step 1 (`engram ingest --auto`) is scored as "ran at least once, anywhere" rather than pinned to a
specific call site, because both a one-learn-invocation strategy (single closing sweep covering
both captures) and a two-invocation strategy (mid-cycle fast-path skip for the correction, then a
closing sweep for the save-request) are equally correct per the skill's own text — the task
instructions explicitly left this either/or, and requiring a SPECIFIC invocation pattern would
wrongly penalize a legitimate deviation.

Tag/source/situation checks (steps 4-6, 9-11) mirror write-memory's steps 4-6/8-10 granularity and
deliberately do NOT chain an `after` ordering onto the base compose match (matching write-memory's
own steps.json, which does not order 4-6 relative to 3 either) — they check flags on whichever
occurrence of the compose command matches, independent of the fetch-ordering gate that steps 7/8
already own.

## What is measured, and what is not

- `steps.json` (11 steps): sweep, first fetch, compose-1 (feedback/airlock, tag/source/situation),
  second fetch (ordered after compose-1), compose-2 (fact/campden, tag/source/situation).
- `done_when_checks.sh`: `engram check` clean; seed notes 1-6 and the write-memory reference note
  1053 byte-identical to the template; exactly 2 new notes; one names the airlock/krausen cause and
  carries a `<family>/<value>` tag; the other carries the Campden 24-hour rule and a
  `sanitation/<value>` tag. No steps.json check for "no unrequested QA pair" — an extra write of
  any kind (including a QA pair, note 1052's duplicate-guard fix) is caught by the exact-count
  check instead, the same way curate/write-memory rely on their own note-set checks rather than a
  separate steps.json assertion for "didn't write something extra."
- **Not yet scored** (a future task 2.x, once the runbook exists): the runbook-fetch gate as
  applied to arm R itself (the runbook won't say "run `engram show`" — it wikilinks write-memory
  directly), and the trigger/retrieval check (note 1039's real-agent-phrase requirement).

## Validation performed (no paid runs; note 1017a's bar)

- **Hand-built ideal vault**: ran the exact write-memory-runbook-prescribed compose commands
  (verbatim flags, per `1053.2026-09-22.write-memory-compose-execute-verify`'s own compose blocks)
  via the real `engram learn`/`engram show` binaries against a copy of `vault-template/`.
  `done_when_checks.sh` **PASSES** on this state.
- **11 single-defect mutants, all FAIL** `done_when_checks.sh` (meets the 8-mutant bar): no writes
  at all; only the airlock note written (Campden missing); only the Campden note written (airlock
  missing); the airlock finding folded into seed note 6 instead of written as a new note (the
  amend-instead-of-write failure mode); an extra unrequested third note; the airlock note missing
  its tag; the Campden note tagged the wrong family (`process/` instead of `sanitation/`); the
  Campden note missing its 24-hour content; the airlock note missing its airlock/krausen content;
  a corrupted sidecar (`engram check` fails); the write-memory reference note (1053) itself edited.
- **`steps.json`** (11 steps) validated against three synthetic transcripts via
  `probe_phase2.evaluate_steps` (no LLM call): the ideal transcript satisfies all 11 steps; a
  transcript that fetches the write-memory runbook once and composes both writes without a second
  fetch fails exactly steps 7 and 8 (nothing else); a transcript using `--tags` instead of `--tag`
  on both writes fails exactly steps 4 and 9 (nothing else). Structural validation (`n` sequential
  from 1, every `after` referencing an earlier `n`) also passes.

## Known limits

- The "correction" is stated as already-having-happened within a single prompt, not produced by a
  live mid-conversation interrupt (see above) — the same class of simplification write-memory's own
  task-rationale used for the opposite reason (it avoided kind-1 entirely; this task embraces the
  same one-shot constraint to reach it).
- A real CLI rejection of `--tags` as an unrecognized flag is self-correcting within a transcript
  (write-memory's runbook error-handling rule: read the error, fix, retry, max 2) — `done_when_
  checks.sh` (vault-level) cannot distinguish "used `--tag` correctly from the start" from "guessed
  `--tags`, got a parse error, and fixed it on retry"; only the transcript-level `steps.json` check
  (whether `--tags` ever appears at all) can, mirroring write-memory's own documented limit.
- Whichever family the agent picks for the airlock/krausen note ("process/fermentation" here) is
  accepted by design (task-prompt says "whichever process category fits", left open); only the
  Campden note's family is pinned (`sanitation/`), because the task-prompt names that category
  explicitly.
