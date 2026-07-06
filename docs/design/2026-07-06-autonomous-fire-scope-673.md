# Plan — #673: autonomous learn-fire captures self-discovered kinds (option a)

**Status:** planning · **Cycle:** 2026-07-06 · **Retire at close** (docs-restructure convention).

## Problem (from #673)

`skills/learn/tests/baseline-autonomous-trigger.md:17` and its README index row
(`skills/learn/tests/README.md:12`) describe autonomous (no-user-prompt) self-fire as
crystallizing "ONLY corrections and explicit save-requests." That rationale predates kind 3
(self-discovered reversals, 2026-07-04) and kind 4b (self-validated bets, 2026-07-06, #668) —
both self-discovered, needing no user turn. The fixture asserts a stale two-kind taxonomy.

**Decision (option a):** autonomous fire captures the self-discovered kinds. Drop the narrowing;
add a positive autonomous case so the fixture locks the behavior instead of passing vacuously.

## Finding that reshapes the work (verified, per note 70 red-baseline-can-falsify-the-premise)

**The skill is already correct; the defect is fixture-only.** `skills/learn/SKILL.md` already
takes position (a):

- Frontmatter: "Also use at session end when a conclusion, design, or finding you presented was
  later overturned or corrected. Also use when a specific approach was confirmed to work…"
- Step 2 kind 3: "Nobody needs to have SAID the correction — self-discovered reversals qualify."
- Step 2 kind 4b: self-validated bet.
- **No autonomous narrowing anywhere in SKILL.md.**

A repo-wide grep (`rg` over skills/guidance/docs/commands) confirms the two-kind narrowing lives
in exactly **two sites**: `baseline-autonomous-trigger.md` (whole fixture) and `README.md:12`.
FEATURES.md, GLOSSARY.md, c1/c2 already reflect the four-kind taxonomy. So there is **no
behavioral RED on the skill** — the work is (1) fix the fixture that mis-grades a correct agent,
and (2) add a *discriminating* positive case. The stale line even actively teaches the wrong
behavior ("crystallizes ONLY corrections and explicit save-requests").

## Approach — writing-skills TDD via the guards harness (headless, not subagents)

The edit is to a test fixture, so writing-skills rigor applies as **validate the fixture encodes
true, discriminating behavior**, not "flip skill behavior." Two measured arms via the existing
`dev/eval/guards` harness (`wlearn` skill variant + scenario prompt + headless `claude -p
--output-format json`, fresh process):

- **Method = headless `claude -p`, NOT subagents** (note
  `feedback_headless_not_subagents_for_insession_guidance_revalidation`): subagents inherit this
  session's context — where kinds 3/4b have been discussed at length — so a two-kind RED control
  would leak the treatment and falsely GREEN. Fresh OS processes + **fictional domains** are
  mandatory.
- Match the repo's measured-rep convention (baseline-confirmed-approach.md records "6/6 reps"):
  ~5 reps/arm.

### Scenario (fictional domain, autonomous fire — no user prompt at fire time)

A session just finished autonomous plan work; learn self-fires. The session contained:

1. **User correction** (kind 1) — e.g. "don't parse the manifest by hand — call `pkgctl
   resolve`." (write both arms)
2. **Self-discovered reversal** (kind 3) — the agent committed "approach A (in-memory index)" to
   its plan, then while implementing discovered A couldn't hold the working set and switched to a
   streaming approach B. Nobody said anything. (**discriminating**: GREEN writes, RED skips)
3. **Self-validated bet** (kind 4b) — unsure whether to shard by tenant or by region, the agent
   bet on tenant-sharding, and the load test then passed at target p99, confirming it.
   (**discriminating**: GREEN writes, RED skips)
4. **Plain self-discovered fact** (no kind) — "the deploy tool exits 0 even when a sub-step warns;
   check stderr." Neither reversal nor bet. (**guard**: NOT written either arm)
5. **Typo fix** in a comment. (**guard**: NOT written either arm)

### Arms & expected

- **RED arm** — two-kind `wlearn` variant (corrections + save-requests only): writes item 1 only.
  Items 2 & 3 **skipped** (variant has no kind 3/4b) → proves items 2/3 discriminate.
- **GREEN arm** — current four-kind `wlearn` (= `dev/eval/guards/candidate/learn.md`, unchanged):
  writes items 1 (kind 1), 2 (kind 3), 3 (kind 4b); **skips** items 4 & 5 → proves the current
  skill captures self-discovered kinds autonomously, and the plain-fact/typo guards hold.

### Empirical decision gate (note 70)

- If the **GREEN arm writes items 2 & 3** (expected) → the current SKILL.md already captures
  self-discovered kinds autonomously → **fixture-only fix, no SKILL.md change.**
- If the GREEN arm **under-captures** (agent skips 2/3 because "no user is present") → add a
  minimal clarifying line to SKILL.md Step 2 stating autonomous fire still applies kinds 3/4b;
  that would be a real behavioral edit with its own RED/GREEN. **Let the measurement decide — do
  not add skill text speculatively.**

## Exact edits (verbatim anchors — note 170; grep-uniqueness verified at plan-write time)

### Edit 1 — `skills/learn/tests/README.md:12` (index row, grep count = 1)

Current:
```
| `baseline-autonomous-trigger.md` | On autonomous (no-user-prompt) self-fire, only explicit user corrections get crystallized as `engram learn feedback`; self-discovered facts and one-off trivial fixes are NOT written — left to the chunk index — and no three-gate logic runs. | Step 2 (autonomous scan scope) |
```
Replace with (names the four kinds; states self-discovered 3/4b DO fire autonomously; keeps the
plain-fact/typo guard):
```
| `baseline-autonomous-trigger.md` | On autonomous (no-user-prompt) self-fire, all four Step-2 kinds apply — including the self-discovered ones (kind 3 reversals, kind 4b validated bets), which need no user turn; a plain discovered fact (no kind) and one-off trivial fixes stay unwritten, left to the chunk index. | Step 2 (autonomous scan scope) |
```

### Edit 2 — `skills/learn/tests/baseline-autonomous-trigger.md` (whole fixture rewrite)

Restructure to mirror `baseline-confirmed-approach.md`: scenario → expected WRITE/guard →
RED-vs-GREEN with measured rep counts. Line-17 rationale corrected: item 4 (plain fact) is not
written because **it is none of the four kinds** (not because autonomous mode is two-kind).
"Failure modes that must FAIL" corrected so **NOT writing the reversal/bet is a failure** (the
current text inverts this). Full replacement authored in step 4 after the reps fix the counts.

## Requirements carried from memory

- **R1** (note 26): writing-skills TDD — baseline RED + edit + pressure tests before complete.
- **R2** (note 70): verify the premise empirically (done for the skill; GREEN arm decides the
  SKILL.md question). No speculative skill text.
- **R3** (note 170): verbatim anchors + grep-uniqueness=1 + complete replacement for every edit.
- **R4**: keep item-4 (plain fact) unwritten — correct only its *rationale*.

## Verification

- Reps: RED writes {1}, GREEN writes {1,2,3}, both skip {4,5}, across ~5 reps/arm. Record counts
  in the fixture.
- `targ check-full` clean (no code touched, but run to be safe).
- `rg` confirms zero remaining two-kind-narrowing sites after edits.

## Rollback

Pure-doc/test change; `git checkout` the two files. No binary/vault impact.

## Out of scope

The hybrid (3 fires autonomously, 4b needs a user) — considered, rejected: 4b's guard is the
point, and autonomous work is 4b's prime habitat. User chose (a).
