# Plan — #673: autonomous learn-fire captures self-discovered kinds (option a)

**Status:** planning · **Cycle:** 2026-07-06 · **Retire at close** (docs-restructure convention).

## Problem (from #673)

`skills/learn/tests/baseline-autonomous-trigger.md` and its README index row
(`skills/learn/tests/README.md:12`) describe autonomous (no-user-prompt) self-fire as crystallizing
"ONLY corrections and explicit save-requests." That rationale predates kind 3 (self-discovered
reversals, 2026-07-04) and kind 4b (self-validated bets, 2026-07-06, #668) — both self-discovered,
needing no user turn. The fixture asserts a stale two-kind taxonomy.

**Decision (option a):** autonomous fire captures the self-discovered kinds. Drop the narrowing;
add a positive autonomous case so the fixture locks the behavior instead of passing vacuously.

## Finding that reshapes the work (verified, per note 70 red-baseline-can-falsify-the-premise)

**The skill is already correct; the defect is fixture-only.** `skills/learn/SKILL.md` already
takes position (a): frontmatter "use at session end when a conclusion… was later overturned" +
"when a specific approach was confirmed to work"; kind 3 "Nobody needs to have SAID the correction
— self-discovered reversals qualify"; kind 4b self-validated bet; **no autonomous narrowing
anywhere in SKILL.md**. The narrowing / stale-phrasing lives in **three test-fixture sites**:
`baseline-autonomous-trigger.md` (the autonomous fixture, whole file), `README.md:12` (its index
row), and `baseline-information-not-knowledge.md:25` (a stale "either kind" quote of the skill rule,
in an unrelated save-request fixture — surfaced by Gate A). So there is **no behavioral RED on the
skill** — the work is fixture-consistency: fix the fixtures that mis-grade / mis-quote, and add a
*discriminating* positive case.

## Harness mechanism (verified against the tree — Gate A code-alignment)

There is **no turnkey guards runner** (`dev/eval/guards/` holds only inputs; the G-battery was run
via `dev/eval/atoms-build/run-arm.sh`, which `cd`s into a project dir carrying a `w`-prefixed
`.claude/skills/<name>/SKILL.md`). I use a **cleaner inline mechanism** that sidesteps the note-168
shadowing risk entirely:

- **Inline both skill variants directly in the scenario prompt** (`scratchpad/scenario-autonomous.txt`,
  `{{SKILL}}` placeholder) — the skill text is the sole variable.
- **GREEN source = the deployed `skills/learn/SKILL.md` Step 2, inlined** (NOT
  `dev/eval/guards/candidate/learn.md` — it differs at Step 1.5's doc pointer; the Step-2 four-kind
  text is verbatim-identical, but "current skill" means the deployed file).
- **RED variant = deployed SKILL.md with kind-3 and kind-4 blocks + the four→two enumerations
  removed** — a faithful pre-kind-3 two-kind skill; cut ONLY those, so the variable is isolated.
- **Delivery marker (note-168 gate):** each inlined variant carries a `FIXTURE-MARKER:` line
  (`wlearn-GREEN-4kind` / `wlearn-RED-2kind`); the prompt makes the agent echo it as line 1, proving
  the arm loaded its variant before scoring.
- **Isolation:** run each `claude -p` in a **fresh temp dir** (not the engram repo) and **disallow
  the Skill tool**, so the agent cannot invoke the deployed `learn` skill and must follow the inlined
  variant.
- **Scoring definition:** "writes item N" = the agent's decision that item N is handed to the
  **write-memory** sub-skill (kinds 1/3/4b emit a handoff; write-memory is not installed inline, so
  there is no on-disk note). The prompt is describe-only; score the emitted per-item WRITE/SKIP
  decision + final `WRITES:` tally.

## Procedure (explicit sequence)

1. **Build the two inlined variants** (GREEN = deployed SKILL.md; RED = two-kind cut), each stamped
   with its `FIXTURE-MARKER`.
2. **Run reps, headless** (`claude -p --output-format json`, fresh OS process in a temp dir, Skill
   tool disallowed — NOT subagents, which inherit this session's kind-3/4b discussion and would leak
   the treatment; note `feedback_headless_not_subagents_for_insession_guidance_revalidation`).
   **5 reps/arm** (leaner than the kind-4 precedent's 6; the discriminating signal is robust).
   Verify each rep's marker line matches its arm before scoring.
3. **Check the empirical gate** (see below).
4. **Author + apply the edits** (Edit 1 + Edit 2 + Edit 3), filling the measured rep counts into
   Edit 2.
5. **Verify:** re-read the touched files; `rg` shows zero remaining two-kind-narrowing sites;
   `targ check-full` clean.

### Scenario (fictional domain "GlyphForge", autonomous fire — no user prompt at fire time)

Session contained, in order: (1) an earlier **user correction**; (2) a **self-discovered reversal**
(kind 3) — committed approach A to the plan, found it wrong while building, switched to B, nobody
said so; (3) a **self-validated bet** (kind 4b) — bet on an approach, a benchmark then confirmed it;
(4) a **plain self-discovered fact** (no kind) — an observed gotcha, neither reversal nor bet; (5) a
**typo fix**. Full prompt: `scratchpad/scenario-autonomous.txt`.

### Arms & expected

- **RED arm** (two-kind variant): writes **{1}** only. Items 2 & 3 match no kind it knows → a
  self-discovered reversal and a validated bet fired autonomously **evaporate**. Proves items 2/3
  discriminate.
- **GREEN arm** (current four-kind): writes **{1, 2, 3}**; skips **{4, 5}**. Proves the current
  skill captures self-discovered kinds autonomously and the plain-fact/typo guards hold.

### Empirical decision gate (note 70 — surface, don't absorb)

- **Expected:** GREEN writes {1,2,3} → the current SKILL.md already captures self-discovered kinds
  autonomously → **fixture-only fix. No SKILL.md change. This is the default outcome.**
- **If GREEN under-captures** (agent skips 2/3 because "no user is present"): that is a **new,
  surprising finding about the skill's behavior — OUT of scope for #673** (a fixture-doc ask). Per
  note 70, **STOP and surface it to Joe** with the measured reps as a candidate follow-up
  (its own issue/cycle). Do **not** absorb a SKILL.md behavioral edit into this cycle. The fixture
  correction (below) still ships regardless — it is unconditional.

**Decoupling:** dropping the narrowing + adding the positive case is unconditional and is what the
user asked for; only the *numeric rep-count annotations* in Edit 2 depend on the harness run. A
flaky/slow harness must not stall the doc correction — if reps can't complete, ship the fixture with
counts marked "(pending)" and note it.

## Exact edits (verbatim anchors — note 170; grep-uniqueness verified at plan-write time)

### Edit 1 — `skills/learn/tests/README.md:12` (index row; grep count = 1)

Current:
```
| `baseline-autonomous-trigger.md` | On autonomous (no-user-prompt) self-fire, only explicit user corrections get crystallized as `engram learn feedback`; self-discovered facts and one-off trivial fixes are NOT written — left to the chunk index — and no three-gate logic runs. | Step 2 (autonomous scan scope) |
```
Replace with (canonical kind names per SKILL.md/GLOSSARY; states self-discovered 3/4b fire
autonomously; keeps the plain-fact/typo guard):
```
| `baseline-autonomous-trigger.md` | On autonomous (no-user-prompt) self-fire, all four Step-2 kinds apply — including the self-discovered ones (kind 3 reversals, kind 4b self-validated bets), which need no user turn; a plain discovered fact (matching no kind) and one-off trivial fixes stay unwritten, left to the chunk index. | Step 2 (autonomous scan scope) |
```

### Edit 2 — `skills/learn/tests/baseline-autonomous-trigger.md` (whole-file replacement)

The old fixture carries the two-kind framing in **six** places, all removed here: title (L1 "only
crystallize explicit moments"), Step-2 header (L15 "scan for explicit moments only"), item-2
rationale (L17 "crystallizes ONLY corrections and explicit save-requests"), report line (L21 "1
explicit correction crystallized"), the "Failure modes" section (L25 "Writing the self-discovered
fact… is out of scope"), and the "What changed from pre-v2" section (L31-33 "current skill does NOT
write self-discovered facts"). Full replacement (rep counts as `N` until step 4 fills them):

```
# Baseline pressure test — autonomous learn: capture self-discovered kinds too

Reusable RED/GREEN scenario input for autonomous (no-user-prompt) self-fire. Autonomous fire is NOT
narrowed to user-attributed kinds — all four Step-2 kinds apply, including the self-discovered ones
(kind 3 reversals, kind 4b self-validated bets), which need no user turn. A plain discovered fact
(matching no kind) and one-off trivial fixes stay unwritten, left to the chunk index.

## Scenario

A coding agent finished a cycle of AUTONOMOUS work (grinding an implementation plan), all tests
green, just committed. NO user prompt — the learn skill self-fires. The session contained, in order:

1. **User correction** (kind 1) — earlier the user said "don't hand-roll the kerning lookup — call
   `glyphd.KernPair()` like the rest of the pipeline."
2. **Self-discovered reversal** (kind 3) — the agent had committed "cache every glyph in one
   in-process LRU map" to its plan, discovered while implementing that the CJK working set thrashed
   it, and switched to a two-tier slab allocator that held. Nobody said the first approach was wrong.
3. **Self-validated bet** (kind 4b) — unsure whether to shard rasterization by font-family or
   glyph-range, the agent bet on glyph-range, implemented it, and the throughput benchmark then
   passed at the 4000 glyphs/sec target.
4. **Plain self-discovered fact** (no kind) — "the `glyphd build` command exits 0 even when a
   subfont fails to embed; grep stderr for `WARN embed`." Neither a reversal nor a bet.
5. **Typo fix** in a comment ("recieve" → "receive").

## Expected current-skill behavior (GREEN)

- **Step 1 — sweep:** `engram ingest --auto` runs first.
- **Step 2 — scan all four kinds (autonomous fire is not narrowed):**
  - Item 1 (user correction) → WRITE `feedback` (kind 1).
  - Item 2 (self-discovered reversal) → WRITE `feedback` (kind 3). Nobody SAID the correction —
    self-discovered reversals qualify; autonomous fire captures them.
  - Item 3 (self-validated bet) → WRITE `feedback` (kind 4b). A bet acted on and confirmed by an
    observable outcome (the benchmark passing); autonomous fire captures it.
  - Item 4 (plain discovered fact) → DO NOT WRITE. It is none of the four kinds — not a reversal
    (nothing was presented then overturned), not a bet (no uncertainty resolved by acting). The
    reason is "no kind matches," NOT "autonomous mode only does corrections"; it stays in the chunk
    index (Step 1).
  - Item 5 (typo fix) → DO NOT WRITE. One-off, not a behavior lesson.
- **Result:** 3 writes (items 1, 2, 3), 0 for the plain fact, 0 for the typo. No `engram transcript`
  calls, no episode notes, no session summarizing.

## RED (two-kind skill) vs GREEN (current four-kind skill)

- **RED:** the pre-kind-3 two-kind skill (corrections + save-requests only) writes ONLY item 1;
  items 2 & 3 match no kind it knows, so a self-discovered reversal and a self-validated bet fired
  autonomously EVAPORATE. (Measured 2026-07-06: item 1 written N/N; items 2 & 3 written 0/N.)
- **GREEN:** the current four-kind skill writes items 1, 2, 3 and skips items 4 & 5. (Measured
  2026-07-06: items 1+2+3 written N/N; items 4 & 5 written 0/N.)

Items 2 & 3 are the discriminating cases: they prove autonomous fire captures the self-discovered
kinds (option a, #673) and would catch a regression that re-narrowed autonomous mode to two kinds.

## Failure modes that must FAIL this test

- NOT writing item 2 (self-discovered reversal) or item 3 (self-validated bet) — the whole point of
  option (a): autonomous fire is not narrowed to user-attributed kinds.
- Writing item 4 (plain fact) or item 5 (typo) — over-capture; neither is one of the four kinds.
- Skipping item 1 (the genuine correction).
- Running `engram transcript`, writing episode notes, or applying removed three-gate logic.
```

### Edit 3 — `skills/learn/tests/baseline-information-not-knowledge.md:25` (stale rule quote; grep count = 1 in-file)

The fixture quotes the skill's write-nothing rule with pre-four-kind phrasing. Correct the quote to
match `SKILL.md:121` verbatim; leave the surrounding save-request-override discussion untouched.

Current fragment:
```
the skill says "No moments of either kind → write nothing" but the save-request is explicit
```
Replace with:
```
the skill says "No moments of any kind → write nothing" but the save-request is explicit
```

## Requirements carried from memory

- **R1** (note 26): writing-skills TDD — baseline RED + edit + pressure tests before complete.
- **R2** (note 70): verify the premise empirically; under-capture STOPs-and-surfaces, is not
  absorbed. No speculative skill text.
- **R3** (note 170): verbatim anchors + grep-uniqueness=1 + complete replacement for every edit
  (both edits' full replacement text is above).
- **R4**: keep item-4 (plain fact) unwritten — correct only its *rationale*.

## Rollback

Pure-doc/test change; `git checkout` the three touched fixture files. No binary/vault impact.

## Out of scope

- The hybrid (3 fires autonomously, 4b needs a user) — considered, rejected: 4b's guard is the
  point, and autonomous work is 4b's prime habitat. User chose (a).
- Any SKILL.md behavioral edit — the skill is already position (a); an under-capture surprise
  escalates to Joe, it is not fixed here.
