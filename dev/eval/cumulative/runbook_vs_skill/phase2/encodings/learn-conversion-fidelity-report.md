# Conversion-Fidelity Report: `learn/SKILL.md` -> Learn-R runbook set

OpenSpec change `learn-skill-to-runbook`, tasks 2.1-2.7. Format mirrors `please-conversion-fidelity-report.md`
and `write-memory-conversion-fidelity-report.md`. Task 2.8 (fresh-context review against the real SKILL.md)
is explicitly NOT done in this pass — a separate fresh-context reviewer does that next.

**2026-09-23 task-2.8 review fixes applied** (see `openspec/changes/learn-skill-to-runbook/design.md`'s
matching amendment note): (1) top runbook `red_flags` entry 3's self-contradictory "the only prescribed
exception" corrected to "the prescribed exception" via `engram amend --red-flag ...` (replace-whole), since
the mid-cycle fast path is a second, legitimate skip condition; (2) the byte counts below for note `1057`
are re-measured against the actual built file AFTER that fix and after the trigger add-back below — they no
longer match the pre-fix figures this report originally stated (the review caught those as wrong even before
the edits: it had said 13,908/1,118, actual pre-fix was 14,182/1,118 whole-file/red_flags with body 11,541 —
this report's original 13,908 was simply a measurement error, not tied to any edit); (3) `triggers:` on the
top runbook restored to `["/learn", "remember this", "note for next time", "write this down"]` (see
"Trigger scoring" section below, amended in place).

- **Source**: `agent-instructions/skills/learn/SKILL.md` (307 lines, 21,233 bytes of body post-frontmatter —
  re-measured directly, matches design.md's own figure). Already carries this session's Step 2.5
  self-referential QA duplicate-guard fix (spec `learn-adhoc-qa-capture`) — carried forward verbatim below,
  not regressed. The real file was NOT touched by this task.
- **Encoding** (`encodings/taskLearn/Learn-R/vault/`, all `type: runbook`, each with a real `.vec.json`
  sidecar built via `engram embed apply --vault <dir> --all`):

| Note | Role | Bytes (whole file) | `red_flags` bytes (cap 1200) |
|---|---|---|---|
| `1054.2026-09-23.learn-sweep-and-vocab-liveness` | sub-runbook 1/3 (Step 1 + Step 1.5) | 4933 | 476 |
| `1055.2026-09-23.learn-adhoc-qa-capture` | sub-runbook 2/3 (Step 2.5 QA capture) | 4244 | 884 |
| `1056.2026-09-23.learn-reparent-luhmann-batch-mode` | sub-runbook 3/3 (batch mode) | 3966 | 604 |
| `1057.2026-09-23.learn-crystallize-explicit-lessons` | top runbook (situation, Step 2 scan, done_when) | 14,242 | **1112** |

`red_flags` bytes = the exact `    - ...\n` block that `capRedFlagsForPreview` measures
(`internal/cli/redflags_truncation.go`, budget 1200 — truncation drops from the FRONT of the list and keeps
the newest/last entries, per that file's own comment; not relevant here since every note is already under
budget with margin). Verified three independent ways (task 2.4):

1. Byte-exact count of the raw `red_flags:` block using the SAME split logic the please/curate tests use
   (`fm.split("red_flags:\n",1)[1].split("\ntriggers:",1)[0].split("\nluhmann",1)[0]`) — top note: **1112
   bytes** (post task-2.8 fix; body 11,543 B, whole file 14,242 B), 88 bytes of margin under the 1200 cap.
2. Built a scratch vault = `fixtures/learn/vault-template/` (the real production `1053...write-memory`
   note plus 6 fictional seed notes) + the 4 new Learn-R notes, ran the real `engram embed apply --vault
   <scratch> --all`, then `engram show <basename>` on all four — **no `EARLIER RED_FLAGS OMITTED` marker on
   any note**.
3. `engram check --vault <scratch>`: PASS on G0 (graph-resolution), M5 (situation-presence), S1
   (sidecar-schema); the only finding is a WARN-level G3 (dangling links) for the four intentional
   *illustrative syntax placeholders* (`[[basename]]` in the REAL production 1053 note itself, `[[wikilink]]`
   and `[[full-basename]]`/`[[...]]` in the converted notes) — all four are VERBATIM carries of the source
   skill's own placeholder syntax examples, not omissions. `engram embed status --vault <scratch>`: total
   11, with-embeddings 11, stale 0, broken 0, incompatible 0.

**Numbering note.** The vault-template's highest pre-existing Luhmann id is `1053` (the real production
write-memory note, copied byte-for-byte — see `fixtures/learn/TASK-RATIONALE.md`). The four new notes take
`1054`-`1057`, with the TOP runbook at `1057` so it sorts LAST among the four in `add_carrier`'s
`sorted(os.listdir(src_dir))` scan (confirmed: `add_carrier(vault, "learn", "R")` returns
`1057.2026-09-23.learn-crystallize-explicit-lessons`) — same convention `please`/`curate` used. `1053` is
NOT copied into `Learn-R/vault/` (per design D2, it is the fixture's own vault-template baseline, already
physically present when the carrier is merged in by `setup_trial_vault`).
`fixtures/learn/task.json` now carries `"carrier_r_src": "encodings/taskLearn/Learn-R/vault"` (added this
task, mirroring please/curate's task.json wiring) — `add_carrier`/`setup_trial_vault` resolve the R arm for
task_key `"learn"` correctly (verified directly, no spend).

## Section plan and disposition (2.1)

Tags: **carry** = lives in the top runbook body; **sub** = moved into a sub-runbook; **drop-as-shim-floor**
= removed because `agent-instructions/guidance/shim.md`'s follow-frame or floor rules already cover it.

| # | SKILL.md section | Tag | Where / what changed |
|---|---|---|---|
| S1 | Frontmatter `description` (trigger/discrimination text) | carry, reframed | Becomes the top `situation:` almost verbatim (the description's own wording IS already situation-shaped: "after work begun with recall, or any work requiring more than one tool call...") — see D4 discussion below for why the description's quoted phrases ("remember this", "save that", "note for next time") did NOT all become `triggers:` entries. |
| S2 | H1 + "Two jobs, in order..." intro paragraph | carry | Verbatim in top body. |
| S3 | "Raw event memory is AUTOMATIC" callout | carry | Verbatim in top body. |
| S4 | "Mid-cycle capture (fast path)" callout | carry, reworded | "SKIP Step 1 (sweep) and Step 1.5 (vocab)" -> "SKIP the sweep-and-vocab opening sequence below" (prose reference, no bracket link in this callout — the actual wikilink to sub-runbook 1054 lives in the index list right after it, not inline in the callout, keeping the callout's urgency readable). |
| S5 | `## Step 1 — Sweep` (command + dedup/retroactive-cleanup note + skip rule) | sub (a) | Verbatim in `1054...`. |
| S6 | `## Step 1.5 — Vocab liveness check` (stats/refit/round-2-gate) | sub (a) | Verbatim in `1054...`, INCLUDING the "report to Joe" / `docs/ROADMAP.md` engram-repo-specific wording — kept verbatim, not genericized, since this is a real, project-specific instruction in the source, not a placeholder. |
| S7 | `## Step 2` intro (LESSONS-lines-as-scan-input paragraph, disposition/placement test) | carry | Verbatim in top body. |
| S8 | Kind 1 — Corrections | carry | Verbatim; write-memory's "run `engram show 1053...` in Bash ... and follow what it prints" -> "follow `[[1053.2026-09-22.write-memory-compose-execute-verify]]`" (D2: real basename, direct wikilink, no fixture placeholder). |
| S9 | Kind 2 — Explicit save-requests | carry | Same wikilink swap. |
| S10 | Kind 3 — Reversals | carry | Same wikilink swap. |
| S11 | Kind 4 — Confirmed approaches (4a/4b, completion-moment anchor, exemplars, runbook-vs-feedback split, both compose blocks) | carry | Same wikilink swap, twice (once for the `kind=runbook` compose, once for `kind=feedback`). |
| S12 | "Rules" bullet list (state the general principle, situation-as-retrieval-handle, one note per principle, supersedes, "no moments -> write nothing") | carry | Verbatim in top body. |
| S13 | `## Step 2.5 — Ad-hoc QA capture` (substantive-answer test, handoff, contributor-extraction rule, **the self-referential duplicate guard**) | sub (b) | Verbatim in `1055...`, wikilink swap. This is the highest-fidelity-risk section — see the dedicated check below. |
| S14 | `## Batch mode — Luhmann re-eval answers (--reparent-luhmann)` (trigger, disposition test, coverage requirement, JSON shape, save+report instructions) | sub (c) | Verbatim in `1056...`. No write-memory handoff exists in this section (batch mode writes a JSON file directly, not a vault note), so no wikilink swap needed here. |
| S15 | Red flags table (8 data rows) | split | See the row-by-row table below. |

## Step 2.5 duplicate-guard fidelity check (the highest-priority item in this task)

Spec `learn-adhoc-qa-capture` (already shipped in the real SKILL.md this session) has two requirements. Both
are present, unaltered in meaning, in `1055.2026-09-23.learn-adhoc-qa-capture.md`:

1. **"Learn SHALL treat a self-answered Step 2 note as already-captured, not as an additional QA-capture
   trigger."** Present verbatim in the sub-runbook body's "Gate — do not duplicate" paragraph: "if a question
   was answered by a Step 2 note YOU wrote this turn, that note already IS the capture ... do not
   additionally write a QA pair for the same question, even though 'crystallized a new vault note (Step 2)
   as the answer' above nominally qualifies it as substantively answered." Also restated as a `red_flags`
   entry on the sub-runbook itself, AND (per design D3, "the highest-cost row to lose") again as a top-runbook
   `red_flags` entry, since it is a previously-shipped-and-regressed defect.
2. **"A question answered by citing an EXISTING prior note SHALL still get a QA pair."** Present verbatim:
   "The wikilink-based trigger stays live on its own: a question answered by citing an EXISTING prior note,
   with no new Step 2 note written for it this turn, still gets its QA pair." Also added as its OWN
   `red_flags` entry on the sub-runbook (the under-fire mirror of requirement 1 — not a row in SKILL.md's own
   table, but the natural companion guard; matches the please precedent of adding new flags for
   validity-critical items not in the source table).

## Red-flags table: placement of all 8 data rows (D3) + additions

| # | SKILL.md row (short) | Placement | Reason |
|---|---|---|---|
| 1 | `engram transcript` anything | sub (a) `1054...` | Episode-workflow-reconstruction guard, per design D3 explicit placement. |
| 2 | Writing an episode/summarizing the session | sub (a) `1054...` | Same episode-workflow class. |
| 3 | Writing facts nobody asked to remember | TOP (added beyond D3's named 3, see below) | General Step-2 scope guard; Step 2 lives in the top runbook, no sub-runbook is a better fit; proposal.md's own row list did not explicitly place this one, so this is a task-level placement decision per D3's closing line ("exact row-by-row placement is a task-level activity"). |
| 4 | Confirmed-approach note for routine success / bare "thanks!" | TOP | Named explicitly in D3 ("over-capturing routine successes/bare 'thanks'"). |
| 5 | Skipped the sweep because "nothing changed" | TOP | Named explicitly in D3. |
| 6 | `--tier` flags or L3/ADR writing | sub (a) `1054...` | D3: "a batch-mode-adjacent historical footgun -> wherever it fits" — placed in the opening-sequence sub-runbook (a guard against reverting to the pre-ingest-era workflow, the same class as rows 1-2) rather than the batch-mode sub-runbook itself, since `--tier`/L3/ADR writing was never part of the reparent-Luhmann batch flow specifically. |
| 7 | Corrected a repo doc (CORRECTION/postscript), skipped the vault note | TOP (added beyond D3's named 3) | Named explicitly in proposal.md's row list ("record-correction-isn't-capture") but not given a specific sub-destination in design D3; it is Kind-3/Reversals discipline, and Kind 3 lives entirely in the top runbook's Step 2, so top is its only natural home. |
| 8 | QA self-referential duplicate guard | TOP + sub (b) `1055...` (full context) | Named explicitly in D3 as "the highest-cost row to lose" — kept on top per D3, AND carried in full in the QA sub-runbook where the complete procedure lives (see the dedicated check above). |
| new | Fetched write-memory's runbook once, reused it for a second compose instead of re-fetching before EACH write | TOP | Not a SKILL.md table row — added because `dev/eval/cumulative/runbook_vs_skill/phase2/fixtures/learn/TASK-RATIONALE.md` names this as "the primary thing this task must be able to score" (does the agent re-fetch write-memory's runbook before EACH compose, per `steps.json` steps 2/7's ordering gate). Mirrors the please/write-memory precedent of adding validity-critical flags not present in the source table. |
| new | Free-listed QA contributors instead of extracting from wikilinks | sub (b) `1055...` | Body-derived (SKILL.md's Step 2.5 prose, not its red-flags table); mirrors the "new" additions please/write-memory made from body rules. |
| new | Pre-validated a contributor basename before including it | sub (b) `1055...` | Body-derived, same section. |
| new | Wrote a duplicate QA pair for a pair already written earlier this session (e.g. by recall) | sub (b) `1055...` | Body-derived ("Gate — do not duplicate" paragraph's first sentence). |
| new | Under-skipped: dropped a QA pair for an existing-note citation because a new Step 2 note also touched the session | sub (b) `1055...` | The explicit under-fire mirror of the duplicate guard (requirement 2 above). |
| new | Skipped a reparent-batch candidate instead of giving it exactly one entry | sub (c) `1056...` | Body-derived (the "Coverage requirement" paragraph). |
| new | Judged reparent position from similarity score alone | sub (c) `1056...` | Body-derived (the disposition-test paragraph's own warning). |
| new | Invoked `engram update --reparent-luhmann` directly instead of leaving it to the user/acting agent | sub (c) `1056...` | Body-derived (batch mode's closing paragraph: "This skill does not invoke either command itself"). |
| new | Renamed/added fields in the reparent answers JSON | sub (c) `1056...` | Body-derived (the JSON shape's own "do not rename fields or add others"). |
| new | Deferred a REFIT_PENDING vocab verdict to the user instead of running it autonomously | sub (a) `1054...` | Body-derived (Step 1.5's "run the refit autonomously — do not defer to the user"). |

Top `red_flags` = 6 entries, **1112 bytes** (post task-2.8 fix removing the redundant "only" from entry 3;
was 1118; <=1200; re-measured after every edit, confirmed via `engram show` with no omitted marker). This is
3 more than D3's literal "keep exactly these two-plus-one" list — the two additions (row 3, row 7) are
proposal.md-named rows that design D3 left unplaced, and the top runbook is their only natural home (both
are Step-2-wide, and Step 2 has no sub-runbook of its own). Budget allowed it (1112 of 1200) without
crowding out the three D3-mandated entries.

## Shim-floor drops (mapped to `agent-instructions/guidance/shim.md`)

| Dropped content | shim.md rule | Coverage verdict |
|---|---|---|
| None of SKILL.md's 8 red-flag rows were dropped outright — all 8 were placed somewhere (see table above). | — | The shim floor's generic "reread a step, don't shortcut/substitute" rules do NOT cover learn-specific content (the four-kind scan discipline, the QA duplicate guard, the sweep-skip exception, the batch JSON shape) — none of it is a generic process-following rule, so nothing from the red-flags table was eligible for a pure shim-floor drop. This matches write-memory's own conversion (which also kept its full row set) more than please's (which had a 21-row table with genuine process-following overlap to drop). |
| "Report loudly" instruction phrasing for vocab refit | shim.md floor rule 3 ("the letter of a step is its spirit") | Kept verbatim anyway (S6) — the floor doesn't specify WHAT to report, only that the step's letter matters; SKILL.md's exact report string is content, not process-following boilerplate. |

## `red_flags`/`done_when`/`situation` structured-field hygiene (mirrors please's own test pattern)

Verified programmatically against all four notes (no `[[...]]` in any structured field except the literal
placeholder convention):

- `1054`, `1056`, `1057`: zero `[[...]]` occurrences anywhere in `situation`/`done_when`/`red_flags`.
- `1055`: initially had two illustrative `[[wikilink]]`/`[[full-basename]]` placeholders IN ITS OWN
  `done_when`/`red_flags` (my own paraphrase, not a SKILL.md quote) — caught and reworded to plain prose
  ("answered with at least one inline wikilink" / "extracting them only from the wikilinks actually present")
  before finalizing, since structured fields must never contain bracket syntax (please's own convention).
- Body text DOES retain SKILL.md's own verbatim `[[wikilink]]` / `[[full-basename]]` / `[[...]]` illustrative
  placeholders where the SOURCE file itself uses them (Step 2's runbook-compose block; Step 2.5's substantive-
  answer test) — these are correct-fidelity carries, not link targets, and `engram check` reports them as
  WARN-level dangling links exactly the way the REAL production `1053...write-memory` note's own
  `[[basename]]` placeholder already does (confirmed against the live vault-template copy — this is an
  existing, accepted pattern, not a defect introduced here).
- Top runbook (`1057`) body wikilinks resolve to exactly its three siblings (`1054`, `1055`, `1056`) plus
  the real external `1053...write-memory` note (present in the vault-template baseline, not in `Learn-R/`
  itself — this differs from please/curate's fully self-contained R vaults by design, per D2) — confirmed
  programmatically and via `engram check`.

## Trigger scoring (2.5) — candidates tested against the real whole-word matcher

Candidates (design D4): `/learn`, "remember this", "note for next time", "save that", "write this down".
Scored in a scratch vault copy (`fixtures/learn/vault-template/` + the 4 new notes) using the REAL shipped
matcher (`engram query --vault <scratch> --text "<probe>"`, checking for `provenances: [trigger]` on the top
note) — not a hand-simulation. Three rounds:

| Candidate | Genuine capture phrasing | Task's 3 generic over-fire probes ("I'll note that...", "remember that this function...", "save the file...") | Tailored over-fire probe (phrase used in a mundane technical sense) | Verdict |
|---|---|---|---|---|
| `/learn` | MATCH (`/learn`) | no match (any) | no match ("Let's learn from this mistake", "We need to learn the new API") | **KEEP** |
| "remember this" | MATCH ("Remember this: always tag sanitation notes.") | no match | **MATCH** ("I remember this bug from last time we deployed.") | DROP |
| "note for next time" | MATCH ("Note for next time: check airlock activity...") | no match | **MATCH** ("There's a sticky note for next time's meeting on the fridge."; "I jotted a quick note for next time I forget the command.") | DROP |
| "save that" | MATCH ("Save that -- we'll need it for the batch report.") | no match | **MATCH** ("We should save that file before closing the editor."; "Let's save that decision for the retro.") | DROP |
| "write this down" | MATCH ("Write this down: never amend pushed commits.") | no match | **MATCH** ("The compiler will write this down to a log file automatically."; "The daemon will write this down to disk every five minutes.") | DROP |

All four multi-word phrase candidates cleared the task's three GENERIC over-fire probes (which target the
bare words "remember"/"note"/"save" used incidentally, not the candidate phrases themselves) — but all four
failed a TAILORED over-fire probe built from each phrase's own plausible non-capture usage in an
engineering-adjacent sentence (recalling a bug, saving a file, a program writing to a log, a stray reminder
note unrelated to the vault). Per task instructions ("Keep only candidates that clear the bar (present for
genuine capture phrasings, **absent** for over-fire probes)"), an "absent" requirement is not met by any of
the four phrase candidates once tested against realistic non-capture usage of their own exact wording.

**Original decision (superseded below): `triggers:` on the top runbook = `["/learn"]` only.** This used the
design's own named escape hatch ("`/learn` alone is a safe floor if the others don't clear the bar"). The
four dropped phrases remain covered by ordinary situation-based semantic retrieval (the top runbook's
`situation:` still names "the user says remember this, save that, or note for next time" as a firing cue) —
dropping them from `triggers:` only removes the forced-candidate lexical fast path, not the runbook's
discoverability. `/learn` itself was verified over-fire-clean against "learn" used as a bare word in
unrelated sentences (it requires the literal `/learn` substring).

**2026-09-23 task-2.8 amendment — three phrases added back as a deliberate, recorded over-fire acceptance.**
A fresh-context reviewer found the four-phrase drop over-strict relative to this repo's own shipped
precedent. `runbook-lexical-triggers` (archived 2026-09-21, D9(c)) records: "Joe added the plain word
`please` as a third trigger ... This is a deliberate exception to the write-memory authoring rule against
generic single-word triggers. Over-firing is accepted: a hit is only a candidate, and the agent decides
relevance from the runbook's own applicability guard." `curate-skill-to-runbook` (archived 2026-09-21, D3)
made the same trade explicitly: "`curate` fires on 'curate a playlist'/'curate the docs' (a trigger hit is a
candidate only, the agent judges it against the situation; documented ... as the accepted trade for a
deliberate invocation word)". The same reasoning applies here, with an added, concrete cost-of-drop finding:
the reviewer's own reproduction confirmed that with only `["/learn"]`, `engram query --text "Remember this:
always tag sanitation notes."` — an unambiguous, textbook save-request, and this runbook's own Kind-2
example phrasing — returned `items: []`, i.e. the runbook did not surface at all for the skill's own
canonical example utterance. A missed capture is learn's highest-cost failure mode (worse than an occasional
unnecessary surface, since every trigger hit is still only a candidate the agent judges against `situation`
and the applicability guard before acting).

**Re-scored per-phrase, against this cost asymmetry:**
- **"remember this" — ADDED BACK.** It is literally the skill's OWN canonical Kind-2 example wording ("the
  user said 'remember this/that X'" — see `situation:` and Step 2's own Kind-2 text). Excluding engram's own
  documented example phrase from its own runbook's triggers was the most indefensible of the four drops.
  Over-fire accepted (tailored probe above still matches, e.g. "I remember this bug from last time we
  deployed") — recorded here as a deliberate trade, not hidden.
- **"note for next time" — ADDED BACK.** Its only over-fire hit in testing was the contrived tailored probe
  above (a sticky note on a fridge, a personal reminder unrelated to engram) — a scenario an actual coding-
  agent transcript essentially never produces. Over-fire accepted as a deliberate trade.
- **"write this down" — ADDED BACK.** Same reasoning: its over-fire hits (a compiler or daemon writing to a
  log) are contrived-probe artifacts, not realistic transcript content. Over-fire accepted as a deliberate
  trade.
- **"save that" — KEPT DROPPED.** This is the one cut that stays well-justified: it is a generic verb
  ("save") plus a generic demonstrative ("that") with no capture-specific signal on its own, unlike the other
  three phrases which each name a concrete, distinctive capture-intent action ("remember", "note ... next
  time", "write ... down"). Its over-fire probes ("save that file before closing the editor", "save that
  decision for the retro") are exactly the mundane, everyday usage the phrase's own grammar invites.

**Final decision: `triggers:` on the top runbook = `["/learn", "remember this", "note for next time", "write
this down"]`.** Amended via `engram amend --target 1057... --trigger "/learn" --trigger "remember this"
--trigger "note for next time" --trigger "write this down"` (replace-whole). Re-verified in a scratch vault
(`fixtures/learn/vault-template/` + the 4 Learn-R notes, real `engram query --text` calls, no spend): all
four generic over-fire probes from the original table above still return no match; all four tailored
over-fire probes still match (accepted, per the above); and `engram query --text "Remember this: always tag
sanitation notes."` now returns the top runbook with `provenances: [trigger]` — the missed-capture failure
mode is fixed. `engram embed status` on the scratch vault: total 11, with-embeddings 11, stale 0, broken 0.
`red_flags` is a separate structured field and is unaffected by this change (still 1112 B, confirmed via
`engram show` after the trigger amend).

## Batch-mode sub-runbook triggers (2.6)

**Decision: no `triggers:` field on `1056.2026-09-23.learn-reparent-luhmann-batch-mode`** (nor on the other
two sub-runbooks — no sub-runbook in this set, or in the please/curate/write-memory precedents, carries its
own `triggers:`; only the top runbook does). Reasoning, per the design's own Open Question: batch mode's
source text is explicit that it is "a deliberate, explicitly-invoked mode" reached only when "you were handed
a derive-phase candidate JSON payload from `engram update --reparent-luhmann`" — nobody types a trigger
phrase for it; the orchestrator (or `engram update` itself) hands the agent a payload directly, and the top
runbook's wikilink is the only intended discovery path. A lexical trigger would have nothing plausible to
fire on (there is no natural user phrase for "start Luhmann batch reparenting").

## 2026-09-23 task-2.8 review: two nit-level trade-offs recorded, no further file changes

**`--tier`/L3 red flag placement (FIX 5).** The reviewer's note describing this trade-off had the two notes'
roles swapped; verified directly against the built files: the `--tier`/L3/ADR-writing red flag actually
lives on **sub-runbook `1054` (Step 1 + Step 1.5)**, and the mid-cycle-fast-path SKIP instruction lives on
the **top runbook `1057`** (its "Mid-cycle capture (fast path)" callout: "SKIP the sweep-and-vocab opening
sequence below; go straight to Step 2"). The real trade-off, correctly stated: because `1054` is exactly
what the fast path skips, an agent taking the mid-cycle fast path for a single correction capture never
fetches `1054` and so never sees the `--tier`/L3 red flag at all during that path — a stricter gap than "it
fires at write time," since the flag is unreachable on that path, not merely late. Moving the row to the top
runbook (`1057`) would fix this, but `1057`'s red_flags budget is 1112/1200 (88 bytes margin after FIX 1) —
tight but the row's plain text ("`--tier` flags or L3/ADR writing -- tiers are not part of learn anymore",
~78 chars) would likely fit as a raw string; however, a `--tier`/L3 writing mistake is specifically an
artifact of the *old episode-based workflow* (`engram transcript`, `engram learn episode`), which is
inherently a cold-start/sweep-time concern, not a mid-cycle single-correction one — an agent already deep in
a mid-cycle correction capture is not the agent at risk of reaching for `--tier`/L3/ADR writing in the first
place. On that reasoning the current placement is left as-is (no file change); flagging the trade-off here
rather than silently accepting it, per the task's instruction.

**`situation:` field restoration (FIX 6).** The runbook's `situation:` had dropped two phrases from the real
skill's `description`: "at session end" (timing) and ", design, or finding" (the object list for "a
presented conclusion ... was later overturned"). Both restored via `engram amend --situation ...`: the
top runbook's `situation:` now reads "...or at session end when a presented conclusion, design, or finding
is later overturned, or a specific approach is confirmed..." — verified this reads naturally as a single
retrieval-shaped clause without breaking the existing structure, re-embedded (`engram embed apply`), and
re-confirmed clean (`engram embed status`: 0 stale/broken; `engram check` unchanged from before: only the
four verbatim-carried illustrative-placeholder dangling links, no new findings). Final whole-file size:
14,282 B (up from 14,242 B after FIX 1 + trigger add-back); `red_flags` bytes unaffected (still 1,112 B,
since `situation:` and `red_flags:` are independent fields).

## Deliberately NOT changed

- No fixture-specific hints in the runbooks (no mention of the home-brewing task domain, airlock/krausen,
  Campden tablets, or any `steps.json` scoring detail).
- The engram-repo-specific "report to Joe" / `docs/ROADMAP.md` wording in sub-runbook (a) is kept verbatim,
  not genericized — it is real content in the source skill, not a fixture artifact.
- `fixtures/learn/task.json` gained `"carrier_r_src"` (needed for `add_carrier`/`setup_trial_vault` to find
  the R arm at all — this is data wiring, not behavior, and required no paid run to verify).

## What was NOT done in this task (explicitly out of scope)

- Task 2.8 (fresh-context review) — not performed here by design.
- Section 3 (validation/retrieval-check against real baseline transcripts, D8 bar) and section 4/5
  (promotion, retirement) — untouched.
- No `.luhmann.lock` or `vocab.centroids.json` were generated inside `Learn-R/vault/` itself (unlike
  please/curate's committed R-vault directories, which happen to carry a 0-byte `.luhmann.lock` and a small
  `vocab.centroids.json`) — `engram embed apply --vault <dir> --all` run directly against `Learn-R/vault/`
  did not produce either file, and the merged-scratch-vault verification above confirms the carrier is fully
  functional without them (health checks clean, no truncation, all notes resolve). Flagging this difference
  explicitly for the fresh-context reviewer to double check it is not a functional gap.

## Points for the fresh-context reviewer (task 2.8) to specifically check

1. **Row placement judgment calls**: rows 3 and 7 (general over-capture; record-correction-isn't-capture)
   were placed on the TOP runbook by my own task-level judgment, not an explicit design D3 instruction —
   verify this reading of "exact row-by-row placement is a task-level activity" is reasonable.
2. **Trigger drop decision**: I dropped 4 of 5 candidate triggers based on tailored over-fire probes I
   constructed myself (not literally the three probes given in the task text). Verify the tailored probes are
   realistic for an actual engram-using coding-agent transcript (not contrived), and that dropping
   "note for next time" / "write this down" — which are arguably closer in spirit to genuine capture-intent
   than "remember this" / "save that" — was the right call rather than a partial keep.
3. **`--tier` row placement** (sub (a) vs elsewhere) — I read design D3's "wherever it fits" as sub (a)
   because rows 1/2 are the same "don't revert to the old pre-ingest workflow" class; confirm this reading.
4. **The `1053` external-link deviation from the please/curate self-contained-R-vault pattern** — confirm
   this is acceptable given design D2's explicit "no fixture placeholder" instruction, not an oversight.
