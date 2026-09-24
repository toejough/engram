# recall task: why two scenarios, why this domain, and the Channel-2 placement problem (tasks 1.1-1.4)

## Task 1.1: drift check against the current real skill

The four existing eval-fixture runbook notes at
`dev/eval/cumulative/runbook_vs_skill/phase2/encodings/shim/recall-learn/vault/` (dated
2026-09-14) were re-read against the CURRENT `agent-instructions/skills/recall/SKILL.md` (344
lines). Findings (not fixed here -- task 2's conversion inherits them):

1. **Stale write-memory wikilink target.** Notes 2 (`recall-note-synthesis-and-coverage`) and 3
   (`recall-activation-and-closing-synthesis`) wikilink the OLD local fixture placeholder
   `[[8.2026-09-14.write-memory-compose-execute-verify]]` at every write handoff (the Absent row,
   Step 4's persist, and the QA-capture follow-on). The current skill instead names write-memory's
   REAL promoted basename directly, `1053.2026-09-22.write-memory-compose-execute-verify`, per an
   earlier session's repointing (design.md's Context section, confirmed at `recall/SKILL.md:192,
   284,307`) -- this is design D2's "no fixture-placeholder detour" already applied live. The
   fixture notes also drop the skill's clarifying phrase at each handoff, "in Bash (a shell
   command -- write-memory is not a Skill tool)" -- added after write-memory itself stopped being
   a Skill and became a runbook.
2. **Missing `explore_allocated` reporting field.** The current skill's Step 2 documents a newer
   query response field, `explore_allocated` (a term-to-delivered-count map, always present,
   degrading to `{}` on missing/unreadable centroid data) -- note 6 (`recall-from-unified-memory`)
   describes explore-sampling but never mentions this field at all.
3. **Missing lazy-chunks verification clause.** The current skill's Channel-1 chunk bullet says
   "Under `--lazy-chunks` (recall's default invocation -- confirm via `budget.lazy_chunks: true`)"
   -- note 6 drops the "confirm via `budget.lazy_chunks: true`" clause entirely.
4. **Dropped C-number taxonomy labels.** Note 1 (`recall-glance-vs-deep-mode`) states the same
   escalation numbers as the skill (0/5 vs 4/5) but drops the skill's `(#661 full-bars)` citation
   and the `(C3)`/`(C4i)`/`(C6)` labels on glance's validated-equivalent cases -- same substance,
   less traceable to the underlying experiment codes other specs/notes cite.

None of these are fixed in this task (1.1 is read-only per the task instructions); they're a
concrete punch list for whoever builds `recall-glance`/`recall-deep` in task 2.

## Task 1.2: one representative task, two scenarios, as two directories

**Decision (design.md's Open Question): two separate task directories sharing a domain, not one
task with a branch, and not fully independent domains.** `fixtures/recall/glance/` and
`fixtures/recall/escalation/` are self-contained fixture tasks (each has its own
`task.json`/`task-prompt.txt`/`steps.json`/`done_when_checks.sh`/`init_fixture_repo.sh`/
`vault-template/`, mirroring every other `fixtures/<name>/` task in this harness), because:

- The two scenarios' `steps.json`/`done_when_checks.sh` shapes are fundamentally different (a: no
  write is ever correct; b: a write via write-memory's real runbook is the ONLY correct outcome)
  -- forcing both into one `task-prompt.txt`/`steps.json` pair would need a branch-detection
  mechanism this harness's `evaluate_steps`/`done_when_checks.sh` model doesn't have (both are
  fixed per task, not conditional on what the trial did).
- `probe_phase2.discover_tasks` (confirmed by reading it, not assumed) scans only the IMMEDIATE
  children of `fixtures/` for the four required files -- it does not recurse. So `fixtures/recall/
  glance/` and `fixtures/recall/escalation/` are **not yet auto-discovered as `--task` names** the
  way `fixtures/gitignore/` and `fixtures/gitignore-nested/` are (two independent top-level
  directories, the closest existing precedent for "two scenarios, one underlying skill"). **This
  is a known, documented gap for whoever runs task 1.5 / builds section 3**: either move both
  scenario directories up to top-level siblings (`fixtures/recall-glance/`,
  `fixtures/recall-escalation/`, exactly mirroring gitignore/gitignore-nested), or extend
  `discover_tasks` to recurse one level under a container directory. Not resolved here because
  task 1.5's paid run is explicitly out of scope for this pass; `done_when_checks.sh` was invoked
  directly by absolute path for all task 1.4 validation below, so the harness-discovery gap did
  not block hand-validation.
- Both scenarios share ONE fictional domain (a community garden co-op) and, where content isn't
  scenario-specific, near-identical seed-note shapes, keeping the pair cheap to build/maintain as
  a matched set while giving each its own deterministic, non-branching scoring path.

**Domain: Sprucebank Community Garden Co-op** -- distinct from write-memory's 3D-printer farm,
learn's home-brewing club, and curate's beekeeping vault (note 996's contamination guard).

## Scenario (a): `fixtures/recall/glance/` -- glance-only, no write

**The ask:** "Quick glance-check for me, nothing urgent -- Plot 14's tomato leaves have dark spots
that keep spreading across a few of the lower leaves. What does our records say is going on, and
what's the fix?"

**Vault (6 notes):** note 1 (fact, OLDER) claims the dark spotting is nutrient/soil splash-back,
fixed by switching to soaker hoses. Note 2 (feedback, NEWER) explicitly CORRECTS note 1 -- "assumed
X... but Y instead... it's actually Z, not X" -- the same reversal-cue shape recall's own Step 2.5B
names ("no longer", "replaced by", "use X not Y"). This is a deliberately targeted test of Step
2.5A (read the candidate content) + 2.5B (apply the recency weight to resolve the conflict) --
narrower and more surgical than a generic diagnostic-accuracy task, since recall's mechanics (not
tomato pathology) are what's under test. Four distractor notes (blossom-end rot, squash mildew,
compost schedule, tool-shed log) are near-topic but never the answer.

**Why no write is possible regardless of the coverage verdict:** under `glance`, Step 2.5C (the
write side of coverage judgment) is skipped UNCONDITIONALLY -- "SKIP Step 2.5C... continue to Step
2.7 (activate)" -- so whether the cluster judges Covered/Near/Absent never matters for whether a
write happens here. This means scenario (a) doesn't need an empty/absent cluster to guarantee no
write; any realistic near-topic cluster works, which let scenario (a) double as the 2.5A/B test
above.

**Scoring split (per write-memory/learn's own established division of labor):** `steps.json`
scores the sweep, the query's glance-scale phrase count (`not_pattern` disqualifies at 6+
`--phrase` flags in one call -- a loose proxy for "~3, not 10", since no exact-count signal type
exists in this harness), activation of the CORRECTING note (`blight-correction`, not the
superseded `splashback` note), and a `text_regex` requiring the final reply to name "blight" (the
only way "read + recency-weighted correctly" can show up as a transcript-observable signal for
Channel-1 note content, which is inline text, not a separate fetch call to score).
`done_when_checks.sh` is the SOLE authority for "no write occurred" -- exactly the six seed notes,
byte-identical to the template -- following write-memory/learn's own precedent of relying on the
vault-level exact-note-set check rather than a steps.json non-existence assertion (their own
TASK-RATIONALE: "No steps.json check for 'no unrequested QA pair' -- an extra write... is caught by
the exact-count check instead").

## Scenario (b): `fixtures/recall/escalation/` -- C5 escalation, write REQUIRED

**The ask:** "Quick glance-check for me, nothing urgent -- we're about to start the fall broccoli
batch in the greenhouse trays. Which starter mix should the crew use?" -- same "quick glance" framing
as (a), deliberately, so the test is whether the agent recognizes it must override that framing.

**Vault:** one topical note (fact) states the OLD convention -- peat-based starter mix. Four
distractor notes. Plus write-memory's REAL promoted runbook note
(`1053.2026-09-22.write-memory-compose-execute-verify`), copied byte-for-byte from the real vault
into `vault-template/` (`$ENGRAM_REAL_VAULT` override available) so `engram show 1053...` resolves
inside a fixture-only trial vault -- the identical mechanism `fixtures/learn/build_vault_template.sh`
already established and documented ("write-memory runbook is a REAL vault note copied into the
fixture").

**The new standard lives ONLY in Channel 2 (recent activity), never in a vault note.**
`init_fixture_repo.sh` commits `docs/co-op-notes-2026-09-22.md`, a meeting-notes doc, into the
trial repo. When the agent's own Step 0.5 sweep (`engram ingest --auto`) runs, the binary chunks
this file into the (per-trial, empty-until-then) chunk index with `IngestedAt` = the sweep's own
timestamp -- the newest possible ingest in a fresh trial, so it surfaces via
`buildRecentFillItems`'s "N newest chunks by IngestedAt" rule (`internal/cli/query.go`) regardless
of any phrase match, UNLESS it also independently matches a query phrase, in which case it is
promoted into Channel 1 and excluded from Channel 2's recent-fill (confirmed by reading
`buildRecentFillItems`: matched chunks are explicitly excluded from the recent-fill dedup).

**This last point is the central risk this task had to verify empirically, not assume (notes
1017a/955's "measure, don't guess" bar).** A first draft of the doc stated the new standard in its
own dedicated `## Sustainability committee update` heading -- chunked on its own, it scored 0.43-0.56
cosine against realistic query phrases (well above the query engine's `matchRelevanceFloor = 0.25`,
confirmed by reading `internal/cli/query.go`), landing it in Channel 1 as a normal cluster member --
which would let ordinary Step 2.5A/B coverage judgment resolve the "conflict" without ever
exercising the C5 escalation path this scenario exists to test. **Verified against the real
`engram ingest`/`engram query` binaries (no LLM call, no spend)** across several iterations: folding
the same sentence into a longer, topically mixed "Board business, various" paragraph (budget votes,
a bike rack, a rake, weatherstripping, parking restriping -- unrelated co-op business surrounding
the one purchasing-policy sentence) diluted the chunk's average embedding enough to drop it to
`score: 0` against three realistic query phrases (`getting the fall broccoli seedling trays going
in the greenhouse`, `quick glance check before answering, nothing urgent`, `which starter mix to
use for seedlings`) -- confirmed landing with `provenances: [recent]` only, absent from every
cluster's members. **Known limit:** this placement is embedding-model- and wording-dependent, not
structurally guaranteed by the fixture; if the runbook conversion (task 2) or a real agent's
phrasing meaningfully changes the realistic query-phrase set, re-verify Channel-1-vs-2 placement
with the real binary before trusting this scenario's escalation requirement (the same empirical
loop used here: ingest, query, grep `provenances`).

**Why the write lands at Step 4 (persist), not Step 2.5C:** Channel-2 items are never cluster
members, so they never appear in any cluster's `candidate_l2s` and never go through the ordinary
coverage table. The only path that can act on a Channel-2 standard is Step 3's glance-mode
instruction ("before synthesizing, check for a load-bearing Channel-2 standard... escalate to
`deep` now") followed by Step 4's persist, whose `--supersedes` mechanism is built exactly for "the
synthesis conclusion CORRECTS... an existing surfaced note." **Verified against the real binary**
that a `--supersedes` write does NOT mutate the target note's file at all (byte-diffed before and
after) -- the inverse link is graph-computed at read time, never written back -- so
`done_when_checks.sh` correctly requires the OLD peat note (and all other seed notes, and note
1053) to stay byte-identical to the template; only ONE new note may exist.

**Scoring:** `steps.json` scores the sweep, the query, the write-memory fetch (`engram show
1053...`), the compose (`engram learn fact|feedback` naming coir/coconut), that it carries
`--supersedes` and `--source`, and a `text_regex` on the final reply requiring the escalated
(correct) answer. `done_when_checks.sh` requires: `engram check` clean; every template note
byte-identical (including the old peat note and note 1053); exactly one new note; its BODY PROSE
(explicitly excluding the auto-generated `Supersedes:` backlink line, whose target basename
contains the substring "peat" purely as part of the old note's slug -- confirmed this was a real
false-pass risk during mutant validation, see below) names both coconut/coir and peat; and it cites
the old peat note's basename (with or without a trailing `.md` -- the flag's value is never
normalized by the binary, confirmed by testing both forms).

## Validation performed (no paid runs; notes 1017a/955's bar)

- **Hand-built ideal end states, both scenarios, via the real `engram`/`engram learn`/`engram
  activate` binaries against a copy of each `vault-template/`.** Scenario (a)'s ideal state is the
  template itself, unmodified (glance never writes) -- `done_when_checks.sh` **PASSES**. Scenario
  (b)'s ideal state is the template plus one `engram learn fact --supersedes "..."` call with the
  documented content -- `done_when_checks.sh` **PASSES**.
- **9 single-defect mutants per scenario, all FAIL** (exceeds the 8-mutant bar):
  - Glance: extra unrequested note; the corrected note amended (activate-only); the superseded
    note amended (activate-only); a seed note deleted; corrupted sidecar (`engram check` fails);
    an unrelated seed note's content modified; an extra feedback note written; a seed note
    renamed; an entirely empty vault.
  - Escalation: no write at all (glance-stuck, the real failure mode this scenario targets); wrong
    content (no coir/peat); correct content but missing `--supersedes`; the old note amended in
    place instead of a new note written; an extra second note also written; the write-memory
    reference note (1053) edited; corrupted sidecar; the new note's body never actually names
    peat (caught ONLY after excluding the auto-generated `Supersedes:` line from the body search
    -- the first version of this check false-passed because the target basename's slug itself
    contains "peat", a concrete instance of the false-pass risk notes 1017a/955 warn about); an
    unrelated seed note modified.
- **`steps.json` validated against synthetic transcripts via `probe_phase2.evaluate_steps`** (no
  LLM call): both scenarios' ideal transcripts satisfy every step; a glance transcript using a
  10-phrase query fails step 2 (its dependent activation step, `after: 2`, correctly cascades to
  FALSE per `evaluate_steps`'s documented "unmatched referenced step -> dependent step is FALSE"
  rule); a glance transcript that activates the superseded note and answers "splash-back" fails
  steps 3-4; an escalation transcript that never escalates (stays on "peat") fails steps 3-7; an
  escalation transcript that fetches the runbook but never writes fails steps 4-6 while step 7
  (the reply itself, since this synthetic transcript still names "coconut-coir" in a truthful
  answer) and step 3 (the fetch) still pass. Structural validation (`n` sequential from 1, every
  signal a recognized kind, every `after` referencing an earlier `n`) also passes for both.

## What is measured, and what is not

- `steps.json`: sweep, query (+ glance-scale phrase count for scenario a), activation of the
  correct note (a), the write-memory fetch and compose with `--supersedes`/`--source` (b), and a
  `text_regex` proxy for "the reply reflects the correctly-judged content" in both.
- `done_when_checks.sh`: vault end-state -- for (a), zero writes, byte-identical seed set; for (b),
  every seed/reference note byte-identical, exactly one new note, correct content, correct
  supersedes link.
- **Not yet scored** (task 2/section 3, once the runbooks exist): the runbook-fetch/wikilink-chain
  validity gate for arm R (core sub-runbook fetched; write-extension fetched only when deep); the
  retrieval/trigger check (note 1039's real-agent-phrase requirement, design D5); and the
  discover_tasks harness-registration gap noted above.

## Harness-registration gap: RESOLVED

**Resolved in a later pass.** `discover_tasks` was confirmed (not just assumed) to scan only the
immediate children of `fixtures/` for the four required files -- it does not recurse, so a nested
`fixtures/recall/{glance,escalation}/` layout was invisible to `--task <name>`. Fix applied: moved
both scenario directories up to top-level siblings, `fixtures/recall-glance/` and
`fixtures/recall-escalation/`, exactly mirroring `fixtures/gitignore`/`fixtures/gitignore-nested`'s
existing "two independent top-level dirs sharing a domain" precedent -- zero changes to
`discover_tasks` or any other harness code. `task.json`'s `vault_template` path (and
`escalation/build_vault_template.sh`'s comment) were updated from the old
`fixtures/recall/{glance,escalation}/` paths to the new top-level ones. `TASK-RATIONALE.md` itself
moved to `fixtures/recall-glance/TASK-RATIONALE.md` (mirroring `curate`/`curate-signal`: the
primary scenario carries the shared rationale doc, the sibling scenario doesn't duplicate it).
Verified: `--task recall-glance --setup-only` and `--task recall-escalation --setup-only` each
build a clean trial vault with no spend; `pp.TASKS` now contains both names with `skill_src`
pointing at `encodings/taskRecall/Recall-S/skills/recall` and `vault_template` pointing at each
task's own `vault-template/`. All prose mentions of `fixtures/recall/glance/` and
`fixtures/recall/escalation/` earlier in this document describe the state AS BUILT during tasks
1.1-1.4, before this resolution, and are left as-is for that history — read them as referring to
the current `fixtures/recall-glance/` and `fixtures/recall-escalation/` paths.

## Known limits

- The Channel-1-vs-Channel-2 placement of the escalation scenario's new standard is empirically
  tuned against the specific doc wording and three specific query phrases used here, not
  structurally guaranteed -- re-verify with the real binary if either changes materially (see
  above).
- Scenario (a)'s "phrase count discipline" check is a loose ≤5-`--phrase` proxy for "~3, not 10"
  (no exact-count signal type exists in this harness's `steps.json` vocabulary); it distinguishes
  glance-scale from deep-scale queries but does not pin an exact phrase count.
