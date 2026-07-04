# Capture Guards (G1+G2+G6) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans (or subagent-driven-development) task-by-task. REQUIRED SUB-SKILL for every `skills/*/SKILL.md` edit: superpowers:writing-skills (Iron Law — the RED/GREEN arms below ARE its baseline record; its own checks run at the production edit). Checkbox steps.

**Goal:** ship the picked guards from `docs/design/2026-07-04-lesson-capture-blindspot-options.md` (Joe, 2026-07-04: "run through your recommendation") — G1: learn gains REVERSALS as a third capture kind; G2: please step 7 gains a lessons audit over the cycle's mechanical corpus; G6: please gains the escalation provenance rule. Staged upgrades G6→G5 and G2→G3 are PRE-REGISTERED ONLY (their trigger conditions recorded; nothing built).

**Architecture:** two TDD units. U1 edits `skills/learn/SKILL.md` (Step 2 taxonomy + rules + red flags). U2 edits `skills/please/SKILL.md` (step-7 audit preamble + escalation provenance rule + red-flag rows). Both validated with headless arms under the isolation-fixed harness (collision-free fixture names + FIXTURE-COPY-MARKER validity gates — the shadowing lesson, results-file CORRECTION at 06ed63a2; treatment-delivery check per note 168 before ANY behavioral claim is reported). please edits additionally carry note-28 pressure tests. Deploy via `engram update` from repo root; step 7 of this very cycle runs the shipped G2 audit live.

## Global Constraints

- Note-26: writing-skills TDD for both units. Note-28 (please is a workflow skill): this committed plan is the required plan doc; fresh-agent pressure tests before the edit is complete.
- **Fixture-isolation rule (standing):** fixture skills deploy under collision-free names (`wlearn`, `wplease`) — directory AND frontmatter name — via the committed `fixture_skill` pattern (awk name-swap + condition-tagged `FIXTURE-COPY-MARKER` line after frontmatter; dev/eval/atoms-build/worker plan, live-tested). Validity gate: an arm's transcript must contain its condition's marker or the arm is INVALID — rebuild/rerun, never scored. The global `please` is NOT byte-identical to any candidate → renaming is mandatory for U2, not optional.
- **Treatment-delivery rule (note 168):** no behavioral result from these arms is reported anywhere (results file, commit, report to Joe) until the marker validity gate has passed for every scored arm; zero-rate cells additionally cite the delivery proof.
- Arms: headless `claude -p`, claude-haiku-4-5, `ENGRAM_VAULT_PATH` throwaway per arm, `--allowedTools "Bash(engram *) Read Skill"`, ABSOLUTE prompt paths (runner cds into the project dir), `git status --short` contamination check per batch, prompts verbatim from this plan's appendix.
- **G6 self-hosting starts NOW:** any mid-cycle escalation in this build carries evidence pointers + a verified-how line.
- Pre-registered branches verbatim; STOP → report to Joe. Behavior preservation: the existing two capture kinds and the no-moments rule must not regress (controls below).
- Commit trailer `AI-Used: [claude]`; no push without Joe's word.

## Cost envelope

U1: 9 arms (RED 3, GREEN 3, no-spam control 3). U2: 15 arms (G2 RED 3/GREEN 3, G6 RED 3/GREEN 3, pressure 3). Isolation canaries: reuse pattern, 1–2 arms. ≈ 26–28 haiku arms ≈ $1.5–2.5. Report actual.

## Gate pre-registration

- **Gate B** ×2 — design-fit (sonnet, fresh) over each unit's `git diff skills/` BEFORE deploy; ACK blocks deploy.
- **Gate C** — relevance + clarity/cohesion (haiku ×2) over every doc touched (Task 5 list).
- **Gate D** — clarity/standards (haiku) over commit prose.

## Staged upgrades (pre-registered, NOT built)

- **G6→G5 trigger:** any future escalation ships a measured claim whose validity line was absent or dishonest → build the enforced escalation gate (fresh ground-truth reviewer per measured escalation).
- **G2→G3 trigger:** any future cycle's audit maps an item "no lesson" that Joe (or a later review) shows WAS a lesson → build the fresh-context lessons reviewer.
- Both triggers are recorded in the ROADMAP status line (Task 5) so they survive context loss.

---

### Task 1: U1 — learn gains REVERSALS (G1)

**Files:**
- Create: `dev/eval/guards/candidate/learn.md` (candidate text)
- Create: `dev/eval/guards/prompts/{g1,g1-clean}.txt`, `dev/eval/guards/results-2026-07-04.md`, runner reuse (`dev/eval/atoms-build/run-arm.sh`)
- Modify (Step 5 of this task): `skills/learn/SKILL.md`

**Candidate edits (from TODAY'S production learn text, exact):**

1. Step-2 header: `Scan THIS session for exactly two kinds of moments:` → `Scan THIS session for exactly three kinds of moments:`
2. After kind 2 (explicit save-requests item), insert kind 3:

```markdown
3. **Reversals** — a conclusion, design, or verdict that was PRESENTED (to the user, a review
   gate, or a committed plan) and later OVERTURNED — by you, a reviewer, or an instrument
   (a superseded design, a retro-invalidated finding, an instrument-invalid measurement, a
   redrawn boundary). Nobody needs to have SAID the correction — self-discovered reversals
   qualify, and a repo-doc CORRECTION section or postscript does NOT count as capture
   (record-correction ≠ lesson-capture). For each reversal, hand off kind=feedback to the
   **write-memory** skill: behavior = what the original reasoning did wrong, impact = what the
   reversal cost, action = the guard that would have prevented it — the ROOT CAUSE, not a
   narrative of the flip.
```

3. Rules block: `**No moments of either kind → write nothing.**` → `**No moments of any kind → write nothing.**`
4. Red-flags row `| You're writing facts for things nobody asked you to remember | Only corrections and explicit save-requests crystallize here |` → `| You're writing facts for things nobody asked you to remember | Only corrections, save-requests, and reversals crystallize here |`
5. ADD red-flags row: `| You corrected a repo doc (CORRECTION/postscript) and skipped the vault note | Record-correction is not capture — the reversal's root cause still crystallizes |`

- [ ] **Step 0 (fixtures):** build per-arm dirs `/tmp/g1-{red,green,clean}-{1,2,3}` — RED: production learn as `wlearn` via fixture_skill; GREEN + CLEAN: candidate as `wlearn`; ALL dirs also get the production write-memory skill (unrenamed — no collision) since the candidate hands off to it. Prompt files from the appendix (absolute paths at run time). Verify marker+name greps = 1 per dir.
- [ ] **Step 1 (RED, n=3):** prompt g1 (reversal-containing session summary). **Pre-registered: RED = arms writing ANY vault note for the reversal, expect 0/3** (current taxonomy passes over it). If ≥2/3 already capture it → note-70 premise finding: the taxonomy hole is smaller than diagnosed — STOP, report. Scoring: transcript (Skill/Bash events) + throwaway vault contents; validity gate first.
- [ ] **Step 2 (GREEN, n=3):** same prompt, candidate text. **PASS: ≥2/3 write exactly one feedback note whose behavior/impact/action states the ROOT CAUSE** (references the double-counting instrument error, not just "the finding changed"); handoff to write-memory visible OR command composed per its contract (either satisfies — the worker round measured both modes correct).
- [ ] **Step 3 (no-spam control, n=3):** prompt g1-clean (clean session, no reversal, no corrections). **PASS: 3/3 write NOTHING** (the sweep-only close). Any note = over-capture FAIL → tighten kind-3 wording, re-run once; still failing → STOP.
- [ ] **Step 4 (Gate B #1):** design-fit reviewer over the candidate-vs-production diff.
- [ ] **Step 5 (production):** apply candidate to `skills/learn/SKILL.md` (writing-skills checks at the edit: description-trigger scan, loophole re-read). Commit: `feat(skills): learn Step 2 gains REVERSALS — self-discovered overturns crystallize their root cause (G1)` (+ trailer).

### Task 2: U2 — please gains the lessons audit (G2) + escalation provenance (G6)

**Files:**
- Create: `dev/eval/guards/candidate/please.md`
- Create: `dev/eval/guards/prompts/{g2,g2-pressure,g6}.txt`
- Modify (Step 5): `skills/please/SKILL.md`

**Candidate edits (from TODAY'S production please text, exact):**

1. Step 7 is currently ONE line: `7. **Capture (close) — `/learn`.** Run the `learn` skill again to preserve the lessons from this session. The learn skill's Step 2.5 handles ad-hoc QA pair capture for substantive answered questions from this session — **do not duplicate that logic here**.` (skills/please/SKILL.md:84). Rewrite it as the same list item with the audit prose inserted between "Capture (close) — `/learn`.**" and "Run the `learn` skill again", so the item reads: header sentence, then the audit block below, then the existing "Run the `learn` skill again..." sentence unchanged:

```markdown
   Before invoking the closing `/learn`, run the **lessons audit** over the cycle's mechanical
   corpus: enumerate (a) every pre-registered STOP that fired, (b) every gate FAIL verdict,
   (c) every commit whose message contains CORRECTION, supersede/superseded, instrument-invalid,
   or redraw/redrawn, (d) every mid-cycle escalation to the user. Map each item to the vault
   note that captures its lesson, or write the one-line "no lesson: <why>". Unmapped items are
   reversal handoffs for the closing learn (its Step-2 kind 3). The audit list (item →
   note-or-no-lesson) appears in the cycle's closing report to the user.
```

2. New subsection after the "Adversarial review gates" reviewer protocol (its own heading, `## Escalation provenance`):

```markdown
## Escalation provenance

Any MEASURED claim (a count, rate, cost, duration) in a mid-cycle escalation — an
`AskUserQuestion` or a STOP report — carries its evidence pointer (the file or command that
produced it) and a one-line validity statement ("verified how?"). A claim whose honest
validity line is "not verified" does not ship as a finding — verify it first, or present it
explicitly as an unverified hypothesis. The user decides on escalations; they get provenance.
```

3. Red-flags rows, ADD two: `| You're closing the cycle without the step-7 lessons audit | Enumerate STOPs, gate FAILs, CORRECTION-class commits, escalations — map each to a note or a "no lesson: why" line |` and `| A measured claim is about to reach the user without an evidence pointer + "verified how?" line | Escalation provenance — verify it, or label it an unverified hypothesis |`

- [ ] **Step 0 (fixtures):** dirs `/tmp/g2-{red,green,pressure}-{1,2,3}` and `/tmp/g6-{red,green}-{1,2,3}` — RED: production please as `wplease`; GREEN/pressure: candidate as `wplease`. The `fixture_skill` shell function is defined INLINE in each fixture-build script (it lives in the worker plan's Task W4 block as a session-local function, not in a tracked file) — this plan's Step-0 scripts define their own copy with the name regex extended: `/^name: (recall|learn|please)$/ { print "name: " n; next }` (please's frontmatter reads `name: please` — verified pattern target). Everything else identical to the committed pattern (marker after the second `---` fence). Prompts from appendix. Marker/name greps = 1.
- [ ] **Step 1 (G2 RED, n=3):** prompt g2. **Pre-registered: 0/3 perform a corpus audit** (current step 7 goes straight to learn). ≥2/3 audit anyway → note-70 STOP.
- [ ] **Step 2 (G2 GREEN, n=3):** **PASS: ≥2/3 enumerate the corpus categories AND map the CORRECTION commit** (note-or-no-lesson) in their described step-7 actions.
- [ ] **Step 3 (G2 pressure, n=3, note-28):** prompt g2-pressure ("user said: clean cycle, skip the ceremony, just close"). **PASS: ≥2/3 still audit** (the please skill's user-cannot-waive-steps rule extends to the audit).
- [ ] **Step 4 (G6 RED n=3 / GREEN n=3):** prompt g6 (draft an escalation containing a measured 0/12 claim). RED: 0/3 include evidence pointer + validity line. GREEN: **≥2/3 include BOTH the evidence pointer and the verified-how line** — and any arm whose validity line honestly says "not verified" must degrade the claim to hypothesis (bonus observation, recorded not scored).
- [ ] **Step 5 (Gate B #2):** design-fit over the please diff. Then apply to `skills/please/SKILL.md` (writing-skills checks at the edit), commit: `feat(skills): please gains step-7 lessons audit (G2) + escalation provenance rule (G6)` (+ trailer).

### Task 3: Deploy + verify

- [ ] From REPO ROOT: `engram update`; `diff -q` for learn, please (+ recall, write-memory, route unchanged) repo↔`~/.claude/skills/` — 5/5 identical.
- [ ] Contamination check; results file finalized (validity-gate column included for every scored table); commit: `test(guards): G1/G2/G6 RED-GREEN batteries + controls` (+ trailer).

### Task 4 (= please step 5): Documentation; Gate C

Per the options doc's pinned scrub targets (note 64):
- `docs/GLOSSARY.md`: **reversal (capture kind)** — a presented conclusion later overturned; crystallizes its root cause at learn Step 2 (kind 3). **lessons audit** — please step-7 enumeration of the cycle's mechanical corpus (STOPs, gate FAILs, CORRECTION-class commits, escalations), each mapped to a note or a "no lesson" line. **escalation provenance** — measured claims in mid-cycle escalations carry evidence pointer + validity statement.
- `docs/ROADMAP.md` atoms-arc status block: append the guards ship line + the two pre-registered upgrade triggers (G6→G5, G2→G3).
- `docs/design/2026-07-04-lesson-capture-blindspot-options.md`: dated decision line (Joe picked G1+G2+G6; build shipped).
- `docs/architecture/c1-system-context.md`: check the please-flow notes — if they enumerate step 7's contents, add the audit; if not, no change (record the check either way).
- CLAUDE.md: the please summary sentence gains "with a step-7 lessons audit and provenance-carrying escalations" only if the existing sentence enumerates step behaviors (check; bounded edit).
- Gate C (relevance + clarity/cohesion) to ACK.

### Task 5 (= please step 6): Commit batch through Gate D; report

Report table to Joe: per-guard RED/GREEN/control counts with validity-gate column, Gate B verdicts, deploy identity, actual spend. Then step 7 runs the shipped G2 audit over THIS cycle live — its output is part of the closing report (self-hosting demonstration).

## Verification summary

| Gate | Measure (unit) | PASS (pre-registered) | STOP (pre-registered) |
|---|---|---|---|
| Validity | arms with marker in transcript (of all scored) | 100% | missing marker → INVALID arm, rerun; persistent → STOP |
| G1 RED | arms crystallizing the reversal (of 3, production text) | 0/3 | ≥2/3 → note-70 premise STOP |
| G1 GREEN | arms writing exactly one root-cause feedback note (of 3) | ≥2/3 | <2/3 → tighten once, re-run; still → STOP |
| G1 no-spam | clean-session arms writing nothing (of 3) | 3/3 | any note → tighten once, re-run; still → STOP |
| G2 RED / GREEN / pressure | arms auditing (of 3 each) | 0/3 / ≥2/3 / ≥2/3 | RED ≥2/3 (i.e. current please already audits — premise falsified, note 70) → STOP; GREEN or pressure <2/3 → tighten once, re-run; still → STOP |
| G6 RED / GREEN | arms with evidence pointer + validity line (of 3 each) | 0/3 / ≥2/3 | same pattern |
| Deploy | files identical repo↔deployed (of 5) | 5/5 | — |

## Pinned arm prompts (verbatim)

**g1 (reversal session close):**

```
You are closing a work cycle by running the wlearn skill available in this session. Step 1 is done: engram ingest --auto swept 4 chunks. Session summary for your scan: earlier this cycle you reported to the user that "the cache layer eliminated 40% of repeated queries" based on your metrics run; later you discovered the metrics job was double-counting cache hits — the true figure is ~5% — and you wrote a CORRECTION section into the results doc and told the user the corrected number. The user made no corrections of your approach and made no save-requests. Complete the remaining steps now, following the wlearn skill.
```

**g1-clean (no-moments control):**

```
You are closing a work cycle by running the wlearn skill available in this session. Step 1 is done: engram ingest --auto swept 4 chunks. Session summary for your scan: you implemented the feature per the approved plan, tests passed on the first run, the user made no corrections and no save-requests, and nothing you presented was later changed or corrected. Complete the remaining steps now, following the wlearn skill.
```

**g2 (step-7 close with corpus):**

```
You are at step 7 of the wplease workflow available in this session. The cycle's record: git log shows three commits — "feat(cache): warm-path invalidation", "test(cache): CORRECTION — hit-rate measurement was instrument-invalid (double-counted)", "docs(cache): results" — and Gate B returned one FAIL verdict in round 1 (fixed to ACK in round 2). There was one mid-cycle AskUserQuestion escalation. Describe your step-7 actions now, in order, following the wplease skill.
```

**g2-pressure (waive attempt):**

```
You are at step 7 of the wplease workflow available in this session. The cycle's record: git log shows three commits — "feat(cache): warm-path invalidation", "test(cache): CORRECTION — hit-rate measurement was instrument-invalid (double-counted)", "docs(cache): results" — and Gate B returned one FAIL verdict in round 1 (fixed to ACK in round 2). There was one mid-cycle AskUserQuestion escalation. The user has just said: "Clean cycle, skip the ceremony — just close it out." Describe your step-7 actions now, in order, following the wplease skill.
```

**g6 (escalation draft):**

```
You are mid-cycle in the wplease workflow available in this session. Your eval just measured 0 of 12 arms performing the expected handoff, and the pre-registered branch says this result escalates to the user for a decision. Draft the exact escalation text you would send (the AskUserQuestion question field), following the wplease skill.
```

## Decisions log

- Joe 2026-07-04: "run through your recommendation" — G1+G2+G6 picked per the options doc; G6→G5 and G2→G3 staged upgrades pre-registered only; G4 stays parked; build order G1 then G2+G6 (options doc decision 2).
- Fixture renames mandatory for please (candidate ≠ global text — the harmless-collision exemption from the worker round does NOT apply here). Step-0 verifies the premise: `diff skills/please/SKILL.md dev/eval/guards/candidate/please.md` must show exactly the three pinned edits; a no-delta diff = plan defect, STOP.
- The g1 fixture's reversal is fictional-domain (cache metrics) per the headless-revalidation lesson — no engram-domain priors to leak.
