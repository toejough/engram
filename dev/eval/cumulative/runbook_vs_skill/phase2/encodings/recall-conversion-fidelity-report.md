# Conversion-Fidelity Report: `recall/SKILL.md` -> Recall-R runbook set

OpenSpec change `recall-glance-skill-to-runbook`, tasks 2.1-2.7. Format mirrors
`please-conversion-fidelity-report.md` / `curate-conversion-fidelity-report.md` /
`write-memory-conversion-fidelity-report.md`.

**Revised after the fresh-context review (task 2.8, 2026-09-23):** row 6 (dispatched
cluster-synthesis subagents) was mis-summarized as "folded into body text" when it had actually
been fully dropped with no trace anywhere (verified: zero "subagent" hits across all four notes)
— restored as a `recall-core` red_flag rather than left dropped, since nothing in `shim.md`'s
follow-frame floor covers it. Row 15's restored-text claim ("k-means grouping is ground truth"
framing) did not actually exist in `recall-core`'s built body — restored for real this time, into
Step 2.5's intro paragraph, alongside two more small SKILL.md facts (Step 2.5's "never appears as
a candidate" / superseded-note ride-along sentences) and two Step-3 sentences (`recall-core`
SKILL.md:231-232) that a diff against the live `SKILL.md` showed missing. Two shipped `red_flags`
rows the placement table had omitted are added below. The body-only-row count is corrected to
6 rows (3, 9, 12, 13, 14, 15) — the old summary sentence's enumeration wrongly included row 6
(which is NOT body-only; it was fully dropped, now restored above) and excluded row 3 (which the
per-row table already tagged body-only but the summary sentence's count omitted). The `1053...`
wikilink-resolution claim about both fixture `vault-template/`s is corrected (only
`recall-escalation`'s ships it — correct and harmless, since neither `recall-core` nor
`recall-glance` ever wikilinks it). All edits to the built notes were made via `engram amend
--body/--red-flag` against the fixture vault (never hand-edited); `engram check`/`embed status`
re-confirmed clean afterward. See each item's placement-table row and the "Fresh review findings"
note at the end for detail.

**Fixture-tree only.** Nothing in `agent-instructions/skills/recall/` was touched. This report and the
notes it describes live entirely under `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskRecall/`.
Task 2.8 (fresh-context reviewer check) is now **done** (2026-09-23, this pass, findings applied — see
"Fresh review findings" below). Section 3 (retrieval/trigger decisions, D5) remains explicitly
**not done** — see "Deferred" at the end.

## Prerequisite: harness discovery gap (JOB 1, not part of tasks.md's section 2)

Before this section's fixtures could be scored at all, `probe_phase2.discover_tasks` needed the recall
eval's two scenario directories to be `--task`-discoverable. They were originally built (a prior
session pass, tasks 1.1-1.4) as `fixtures/recall/glance/` and `fixtures/recall/escalation/` — two levels
deep, invisible to `discover_tasks` (immediate-children-of-`fixtures/`-only, confirmed by reading it, not
assumed). Fixed by moving both to top-level siblings, `fixtures/recall-glance/` and
`fixtures/recall-escalation/`, exactly mirroring `fixtures/gitignore`/`fixtures/gitignore-nested` — zero
harness code changes. `TASK-RATIONALE.md` moved with `recall-glance` (curate/curate-signal precedent: the
primary scenario carries the shared rationale, the sibling doesn't duplicate it). TDD: a failing test
(`test_recall_glance_and_escalation_are_discovered_as_top_level_tasks`) was added first (RED: neither name
in `pp.TASKS`), the directories moved and `task.json`'s `vault_template` paths fixed, test now GREEN. Full
suite: 455 passed, 2 skipped (no regressions). `--task recall-glance --setup-only --arms S` and `--task
recall-escalation --setup-only --arms S` both confirmed building a clean trial vault, no spend.

## Source and encoding

- **Source**: `agent-instructions/skills/recall/SKILL.md` (344 lines, 27,327 bytes; frozen byte-identical
  copy at `encodings/taskRecall/Recall-S/skills/recall/SKILL.md`).
- **Encoding**: `encodings/taskRecall/Recall-R/vault/`, four `type: runbook` notes, each with a
  `.vec.json` sidecar. Built with the real `engram learn runbook` in a scratch vault (`--position top`,
  `--slug <name>`), which happened to assign luhmann ids 1-4 in creation order (core, write-extension,
  glance, deep) — not renumbered afterward, since the ids already matched the intended build order
  (migration plan step 6: sub-runbooks promoted before entry points) and `deep` (the skill's own default
  mode, "absent -> deep") sorting last matches `please`'s "the primary note sorts last" convention. No
  `vocab.centroids.json` or `.luhmann.lock` shipped in the carrier dir (would overwrite the seed vault's
  own on merge, per `curate`/`write-memory` precedent). `engram embed status --vault <fixture>`: total 4,
  with-embeddings 4, stale 0, broken 0, incompatible 0. `engram check --vault <fixture>`: PASS on
  graph-resolution/situation-presence/sidecar-schema; two expected WARN-only dangling links (the
  `write-memory` basename, absent from this note-only scratch vault but present in
  `recall-escalation`'s fixture `vault-template/` — `recall-glance`'s fixture does NOT ship it, and
  correctly so: neither `recall-core` nor `recall-glance` ever wikilinks `1053...` in the read-only
  path, only `recall-write-extension` does, which `recall-glance` never reaches; and the literal
  `[[full-basename]]` syntax placeholder in the write-extension body, same convention as `please`'s
  literal `[[note-basename]]` placeholder).

| Note | Role | Bytes | `red_flags` bytes (cap 1200) |
|---|---|---|---|
| `1.2026-09-23.recall-core` | shared sub-runbook (Steps 0, 0.5, 1, 2, 2.5A/B, 2.7, 3, 3.5) | 16,356 | 1,159 (14 rows) |
| `2.2026-09-23.recall-write-extension` | shared sub-runbook (Step 2.5C action + Step 4 persist) | 6,164 | 563 (6 rows) |
| `3.2026-09-23.recall-glance` | entry point (cheap, read-only rung) | 2,750 | 393 (3 rows) |
| `4.2026-09-23.recall-deep` | entry point (full rung, default mode) | 2,683 | 357 (3 rows) |

Bytes/rows re-measured after the task-2.8 fresh review restored row 6 into `recall-core`'s
`red_flags` (13 -> 14 rows, 1,028 -> 1,159 bytes, +131) and added ~474 bytes of small body-text
restorations (Fix 3, 5, 7 below) to `recall-core`'s body (15,751 -> 16,356 bytes total, +605). Still
well under the 1200-byte `red_flags` cap (41 bytes free) and far under `please`'s ~13-18KB
single-note precedent for body size.

`red_flags` bytes = the exact `    - ...\n` block `capRedFlagsForPreview` measures (`internal/cli/redflags_truncation.go`,
budget 1200), computed directly from each built note's raw frontmatter. Verified two ways: byte count of the raw
block, and `engram show <basename> --vault <fixture>` on every note prints no `EARLIER RED_FLAGS OMITTED` marker.
All four notes have substantial headroom (172-843 bytes free), unlike `please`'s top note (1163/1200, near the
cap) — `recall`'s 26-row table (not 28; measured directly, see below) split cleanly across four notes rather
than needing please's aggressive per-row merging.

**Row-count correction:** design.md's Context section states "28-row/4,206-byte" red-flags table. Direct
measurement (line-counting `| ... |` data rows in `SKILL.md`'s Red Flags section, excluding the header and
separator rows) found **26 rows**, 4,206 bytes exactly — the byte count matches design.md; the row count does
not. Not a fidelity loss (no rows were skipped in the count), just a correction for whoever reads design.md next.
Section byte sizes (Overview 1,389; Modes 1,919; Step 0 534; Step 0.5 677; Step 1 1,442; Step 2 3,651; Step 2.5
4,787; Step 2.7 805; Step 3 1,922; Step 3.5 2,198; Step 4 3,231; Red flags 4,206) were re-measured directly and
match design.md's numbers within a few bytes (heading-line boundary rounding) — the design's byte-count
groundwork was accurate; only the row count needed a fix.

## Section plan and disposition (2.1)

Tags: **carry** = content preserved (possibly reworded/condensed); **sub** = moved into a sub-runbook;
**split** = divided across notes; **drop-shim-floor** = removed, covered by `shim.md`'s generic follow-frame.

| # | SKILL.md section | Tag | Where |
|---|---|---|---|
| S1 | Frontmatter `description` (trigger discrimination) | carry, reworded | Becomes each entry point's own `situation:` — NOT one shared trigger description, since `recall-glance` and `recall-deep` are selected differently (see "Deferred", D5 is task 3's job). No `triggers:` flag was passed to either entry point in this pass (matches `route`'s precedent of leaving trigger decisions for the retrieval-check task, not decided here). |
| S2 | Overview: two-layer memory, jobs-in-order list (1-6), binary auto-resolves vault/chunks | carry | Condensed into `recall-core`'s opening paragraph plus the `--vault`/`--chunks-dir` caveat. The jobs-in-order numbered list itself is dropped as a *list* (the six items are exactly the note's own Step 0/0.5/1/2/2.5/2.7/3/3.5 headings restated) — carrying both the list and the steps would duplicate; the steps ARE the list. |
| S3 | Modes: `deep` definition | carry, reworded | `recall-deep`'s opening paragraph. |
| S4 | Modes: `glance` definition (what it keeps: 2.5A, 2.5B, 2.7, Step 3, Step 3.5; what it skips: 2.5C, Step 4) | carry, reworded | `recall-glance`'s opening paragraph; ALSO restated as explicit branch text inside `recall-core`'s Step 2.5C-criterion paragraph and Step 1's phrase-count line (design D1's "branch points preserved as explicit body text"). |
| S5 | Modes: C5 escalation rule (glance 0/5, deep 4/5, #661) + "glance validated as deep-equivalent only for C3/C4i/C6" | carry, **verbatim numbers** | `recall-glance`'s "Escalate to `recall-deep`" section, per task 2.3's explicit requirement to state this verbatim. The C3/C4i/C6 labels (dropped in the pre-existing shim-fixture notes per TASK-RATIONALE.md's task-1.1 drift finding) are RESTORED here — more traceable to the underlying experiment codes other specs cite. |
| S6 | Step 0 (upfront judgement) | carry, verbatim | `recall-core`. |
| S7 | Step 0.5 (sweep) | carry, verbatim | `recall-core`. |
| S8 | Step 1 (10-phrase list) | carry, branch-annotated | `recall-core`; opening sentence states the deep-10/glance-~3 split inline (replacing SKILL.md's `[glance: ...]` blockquote-annotation style with plain prose, since a runbook body has no analogous "here's an aside for the other mode" convention already established elsewhere in this session's conversions — `please`/`write-memory` don't have an inline-annotation precedent to match, so plain prose was chosen). |
| S9 | Step 2 (unified query, two channels, `explore_allocated`) | carry, verbatim | `recall-core`. Includes the `explore_allocated` field (present in the CURRENT skill; TASK-RATIONALE.md's task-1.1 flagged it as missing from the old shim-fixture note 6 — this conversion re-adds it) and the `--lazy-chunks`/`budget.lazy_chunks: true` confirmation clause (also flagged missing in the old fixture, restored here). |
| S10 | Step 2.5A (read candidates) | carry, verbatim | `recall-core`. |
| S11 | Step 2.5B (recency weight) | carry, verbatim | `recall-core`. |
| S12 | Step 2.5C (coverage table: Covered/Near/Absent, CRITERION + ACTION columns) | **split** (see "Coverage-table split" below) | Criterion column -> `recall-core` (needed by both modes for Step 2.7's activation judgment); Action column -> `recall-write-extension` (deep only). |
| S13 | Step 2.7 (activation) | carry, verbatim | `recall-core` — kept in both modes per the skill's own text. |
| S14 | Step 3 (closing synthesis rules) | carry, verbatim + glance escalation-check branch | `recall-core`; the `[glance: before synthesizing, check for a load-bearing Channel 2 standard...]` annotation becomes an explicit "Under `recall-glance`, before synthesizing..." lead sentence. |
| S15 | Step 3.5 (re-entry query + `Re-entry:` line contract) | carry, verbatim | `recall-core` — runs in both modes per the skill's own text ("a query is a read; glance keeps it"). |
| S16 | Step 4 (persist: certainty-by-inference-mode, `--source` marking, supersedes, QA capture) | carry, verbatim | `recall-write-extension` — deep only. |
| S17 | Red Flags table (26 rows) | split | See placement table below. |

### Coverage-table split — the one genuine interpretive judgment call (flag for fresh review)

SKILL.md's own inline annotation says, verbatim, `[glance: SKIP Step 2.5C — it is the write side. Read
2.5A + apply 2.5B, do not amend/learn — continue to Step 2.7 (activate).]` — literally, glance skips the
ENTIRE 2.5C section, table included. But Step 2.7's own text (kept under glance) says to activate "the
candidates you judged Covered or Near **at the coverage table**" — a table glance was just told to skip
entirely. This is a genuine tension already present in the source, not introduced by conversion. Resolution
applied here: the CRITERION half of the table (what counts as Covered/Near/Absent) was kept in `recall-core`
(so glance can still judge it, purely to decide what to activate and how to frame Step 3's synthesis), and
only the ACTION half (the actual `engram amend`/write-memory commands) moved to `recall-write-extension`
(deep only). **A fresh reviewer should specifically check this interpretation against the real skill's
intent** — the alternative reading (glance skips the coverage judgment entirely, and Step 2.7's "Covered or
Near" clause only ever applies under deep) was rejected here because it would leave glance with no
principled activation rule at all, but it is a plausible alternative and was not tested against the real
skill's author's intent, only against internal consistency.

## Red Flags table: placement of all 26 rows (D3)

| # | SKILL.md row (short) | Placement | Reason |
|---|---|---|---|
| 1 | Never printed Step 0 | `recall-core` red_flag | Structural gate; every other conversion leads with this. |
| 2 | Skipped 0.5 sweep, no prior sweep | `recall-core` red_flag | |
| 3 | `--vault`/`--chunks-dir` on query | body only (`recall-core`'s opening line) | Already an emphatic bolded body instruction; budget went to higher-severity rows instead. |
| 4 | Separate query calls per phrase | `recall-core` red_flag | |
| 5 | Quoted chunks wholesale | `recall-core` red_flag | |
| 6 | Dispatched cluster-synthesis subagents (deprecated pattern) | `recall-core` red_flag (restored 2026-09-23, task 2.8 review) | Originally cut for budget and marked "dropped" in this table, but the summary below wrongly claimed it was "covered as body text" — a grep for "subagent" across all four notes found zero hits, so it had no trace anywhere. Not covered by `shim.md`'s follow-frame floor either (verified: the floor has no guard against reverting to a note-specific deprecated pattern). Default-restore rule applied: added as `recall-core`'s 14th `red_flags` row via `engram amend --red-flag` (1,159/1,200 bytes, still under budget). |
| 7 | Judged coverage before reading candidate content | `recall-core` red_flag | Skill calls this out with its own extra STOP paragraph — high severity. |
| 8 | Applied a cosine threshold instead of judging content | `recall-core` red_flag | |
| 9 | Candidate matching only superseded content marked covered | body only (`recall-core`'s Step 2.5B/C text states this directly: "near, not covered") | |
| 10 | Wrote two notes (fact AND feedback) for one cluster | `recall-write-extension` red_flag | Write-side action. |
| 11 | Called `engram learn --target` instead of `amend` | `recall-write-extension` red_flag | |
| 12 | High-cosine cluster activated without reading | body only (`recall-core`'s 2.5A "do not judge coverage before reading" already covers the same failure shape) | |
| 13 | Called `engram show` on a note already in `items[]` | body only (`recall-core`'s Step 2.5A: "no `engram show` calls for candidates") | |
| 14 | Assumed chunk content inline, skipped evidence | body only (`recall-core`'s Step 2.5A zero-note-cluster paragraph) | |
| 15 | Grouped chunks by eye instead of clusters | body only (`recall-core`'s Step 2.5 intro) | Restored 2026-09-23 (task 2.8 review): the original pass's claimed restoration text did not actually exist anywhere in the built note (verified: no "k-means", no "ground truth" hits before this fix). Now genuinely present, added via `engram amend --body`: "The binary's k-means clustering is ground truth — group by its clusters, not by eye." |
| 16 | Skipped Step 2.5 or read chunk-only as "nothing surfaces" | `recall-core` red_flag | |
| 17 | Activated every returned note | `recall-core` red_flag | |
| 18 | Activated recent-channel items | `recall-core` red_flag | |
| 19 | Skipped `engram activate` after drawing on notes | `recall-core` red_flag | |
| 20 | `--relation` / hand-authored vocab tag / `Supersedes` backlink | `recall-write-extension` red_flag | |
| 21 | Composed an `engram learn`/`amend` command at a write site | `recall-write-extension` red_flag | |
| 22 | Reply is a memory dump with no plan reference | `recall-core` red_flag | |
| 23 | Recommending a prerequisite/better test instead of the asked task | `recall-core` red_flag | |
| 24 | Wrote a recommendation line with no `Re-entry:` line | `recall-core` red_flag | |
| 25 | Ran the write side while in `glance` mode | `recall-glance` red_flag | Mode-violation; belongs on the entry point that owns the mode boundary, not the shared core. |
| 26 | Recency-channel standard load-bearing, stayed in `glance` | `recall-glance` red_flag | The C5 rule itself — the single most important flag in this whole conversion. |
| new | Invoking write-memory as a Skill tool instead of Bash | `recall-write-extension` red_flag | Not a SKILL.md table row, but explicit body prose in three places ("write-memory is not a Skill tool") — added as a flag since it is a real historical gotcha (write-memory used to be a Skill). |
| new | Generated 10 phrases under `recall-glance` | `recall-glance` red_flag | Mirrors row 4's "separate calls" flag but for the mode-specific phrase-count contract; not a SKILL.md row (the count contract is stated in Modes/Step 1 prose, not the red-flags table). |
| new | Skipped the write side after a sound conclusion under `recall-deep` | `recall-deep` red_flag | Derived from Step 4's own requirement; `recall-deep`'s entry point needed at least one flag of its own beyond mirroring glance's escalation trigger, and this is the direct converse failure mode (the entry point that should always reach the write side, not reaching it). |
| new | Ran `recall-deep` for a routine check `recall-glance` would cover | `recall-deep` red_flag | Not a SKILL.md row; states the cost trade-off the Modes section's "use deep when... in doubt" language implies but never turns into an explicit warning. |
| new | Rewrote a Covered candidate's content instead of provenance-enriching only | `recall-write-extension` red_flag | Shipped in the built note from the start (task 2.5); omitted from this placement table in the original pass — added here (task 2.8 review) so the table's accounting matches the note. |
| new | Generated only ~3 phrases under `recall-deep` instead of the full 10 | `recall-deep` red_flag | Shipped in the built note from the start (task 2.4); omitted from this placement table in the original pass — added here (task 2.8 review). Mirrors row-25-adjacent glance's "generated 10 phrases" flag, but for deep's own converse. |

**Body-only rows: 3, 9, 12, 13, 14, 15 (6 rows).** (Corrected 2026-09-23, task 2.8 review: the
original summary sentence below enumerated these as "6, 9, 12, 13, 14, 15" — row 6 was a
typo/miscategorization for row 3, which the per-row table above already tagged "body only" but the
summary sentence omitted. Row 6 itself is NOT body-only: it was fully dropped with no trace
anywhere in any note, and this review restored it above as an actual `recall-core` red_flag rather
than leaving it dropped or reclassifying it as body-only.) All 6 rows in the corrected set are
confirmed still genuinely present as instructions in `recall-core`'s body, not just implied, per
the "What a fresh reviewer should specifically check" section below.

These 6 body-only rows remain covered as **body text** in `recall-core` (see the table above) —
none were silently removed; each was judged lower-severity than the rows kept as actual
`red_flags` entries, per the please/write-memory "prioritize by frequency x severity, move the rest into
body text" method (design's Risk section explicitly sanctions this). Row 6 is the one exception:
it was not body text at all (a genuine drop), and this review restored it as a `red_flags` entry
instead of leaving it dropped or reclassifying it as body-only.

## Wikilink targets (built and verified via `engram show`/`engram check`)

- `recall-core` -> `recall-glance`, `recall-deep`, `recall-write-extension` (all resolve; it links to
  every other note in the set, since all three reference back into or out of it).
- `recall-write-extension` -> `recall-core`, `1053.2026-09-22.write-memory-compose-execute-verify` (real
  write-memory basename, per design D2 — no fixture-placeholder detour), plus the literal `[[full-basename]]`
  syntax-placeholder text (matches `please`'s convention).
- `recall-glance` -> `recall-core`, `recall-write-extension` (named in the "do NOT run" sentence, not a
  call to follow it — still a valid wikilink target per the shim's follow-frame, since the body says not
  to fetch it in the glance case), `recall-deep` (the escalation target).
- `recall-deep` -> `recall-core`, `recall-write-extension`, `recall-glance` (referenced in the "when you
  should have started here instead" section).

All four resolve by exact basename inside `encodings/taskRecall/Recall-R/vault/` (`engram check`: PASS
G0 graph-resolution). The `1053...` link resolves inside `recall-escalation`'s fixture
`vault-template/` (confirmed: it already ships `1053...md`). **Corrected 2026-09-23 (task 2.8
review):** it does NOT resolve inside `recall-glance`'s fixture `vault-template/` — that fixture does
not ship `1053...md` at all. This is correct and harmless, not a gap: `1053...` is wikilinked only
from `recall-write-extension` (the write side), which `recall-glance` never reaches; neither
`recall-core` nor `recall-glance` mentions or wikilinks it anywhere in the read-only path.

## Fixture-vault wiring

`fixtures/recall-glance/task.json` and `fixtures/recall-escalation/task.json` both gained
`"carrier_r_src": "encodings/taskRecall/Recall-R/vault"` — a SHARED carrier directory, mirroring
`curate`/`curate-signal`'s precedent of one R-arm carrier serving two sibling scenario tasks. Verified:
`--task recall-glance --setup-only --arms R` and `--task recall-escalation --setup-only --arms R` each
build a clean 10-note trial vault, no spend. **Fixed 2026-09-23 (task 2.8 review, harness FIX 1,
blocker):** `carrier_basename` used to report `4.2026-09-23.recall-deep` for BOTH tasks (the
last-sorted `.md` in the shared dir), which is wrong for `recall-glance` (its own entry point is
`3.2026-09-23.recall-glance`) and made `score_found_phase2`'s substring-match "found" check unable
to discriminate the two tasks — worse, since `recall-core`'s and `recall-glance`'s own body text
wikilinks `[[4.2026-09-23.recall-deep]]`, "found" could fire off ANY of the three notes surfacing,
not specifically the intended entry point. Fixed by adding an explicit `carrier_entry_basename`
field to each task's `task.json` (`recall-glance`: `3.2026-09-23.recall-glance`;
`recall-escalation`: `4.2026-09-23.recall-deep`), with `add_carrier` preferring it when present and
falling back to the old last-sorted rule otherwise (backward compatible with
please/curate/write-memory/learn's single-entry-point carriers, which set no such field). TDD:
`test_recall_glance_found_scoring_uses_its_own_entry_point_not_recall_deep` failed against the old
code (a query result containing only `3.2026-09-23.recall-glance.md`, not `4...recall-deep`, scored
`found=False`), now passes; full suite 460 passed / 2 skipped (no regressions).

## What a fresh reviewer should specifically check

1. **The coverage-table split** (see above) — the one place this conversion made an interpretive call
   about ambiguous source text rather than a mechanical carry.
2. **Six body-only red_flags rows** (3, 9, 12, 13, 14, 15 — corrected 2026-09-23, see above) — confirm
   each is still genuinely present as an instruction, not just implied. Row 6 (the true drop) is
   restored as an actual `recall-core` red_flag, not body-only; confirm its wording against
   `SKILL.md`'s real row 6 above.
3. **Luhmann numbering** (`deep` sorts last, not `glance`) — a deliberate choice (deep is the skill's own
   default mode) but the opposite of what a literal reading of design.md's "glance is discovered first"
   framing might suggest; not load-bearing yet since no `steps.json` scores it. Explicitly NOT
   addressed by this review pass (Joe: low priority, skip).
4. **No `triggers:` on any of the four notes** — deliberate (D5 is task 3's job, not decided here), but
   worth confirming nobody mistakes the absence for an oversight.

## Fresh review findings (task 2.8, 2026-09-23) — applied, not just noted

All items below were fixed in this pass, not merely flagged for a future one:

- **Harness FIX 1 (blocker):** `carrier_entry_basename` added; see "Fixture-vault wiring" above.
- **Report FIX 2:** row 6's drop/restore corrected (table row 6, summary sentence, and the "What a
  fresh reviewer should check" list above).
- **Report/note FIX 3:** row 15's restoration claim corrected and made real (table row 15, note body).
- **Report FIX 4:** the `1053...` both-fixtures claim corrected (see "Wikilink targets" above and the
  frontmatter-table note).
- **Note FIX 5:** the two missing SKILL.md:231-232 Step-3 sentences restored into `recall-core`'s body
  (via `engram amend --body`).
- **Report FIX 6:** body-only row count corrected (6, not the miscounted "6,9,12,13,14,15" enumeration
  that used "6" for "3"); the two shipped-but-omitted `red_flags` rows added to the placement table.
- **Note FIX 7:** the two small Step-2.5 facts (never-a-candidate rule; superseded-note ride-along)
  restored into `recall-core`'s body alongside FIX 3's sentence, in one combined addition — judged to
  fit naturally without bloating the note (body cap is not the 1200-byte constraint; that budget is
  `red_flags`-only, and `recall-core`'s body at 16,356 bytes remains well under `please`'s
  ~13-18KB-is-fine precedent).

All four notes re-verified after every edit: `engram check --vault <fixture>` PASS on
graph-resolution/situation-presence/sidecar-schema (same two expected WARN dangling links as before);
`engram embed status --vault <fixture>`: total 4, with-embeddings 4, stale 0, broken 0, incompatible 0.
Every edit went through `engram amend --body`/`--red-flag` against the fixture vault directly (never
hand-edited); the auto-created `.luhmann.lock` and empty `vocab.centroids.json` stub `amend` leaves
behind were deleted afterward to preserve the "no lock/vocab file shipped in the carrier dir"
convention documented above.

## Trial-vault resolution check (no-spend, 2026-09-23)

Built both trial vaults (`--task recall-glance --arms R --setup-only` and `--task recall-escalation
--arms R --setup-only`); in each, `engram show <basename> --vault <trial-vault>` resolved all four
notes (`1.2026-09-23.recall-core`, `2.2026-09-23.recall-write-extension`, `3.2026-09-23.recall-glance`,
`4.2026-09-23.recall-deep`) with exit 0 and content (situation/done_when/red_flags, row counts and
wording) matching this report exactly. `carrier_entry_basename` (FIX 1) confirmed correct per task:
`recall-glance`'s setup reported `carrier_basename=3.2026-09-23.recall-glance`, `recall-escalation`'s
reported `carrier_basename=4.2026-09-23.recall-deep`. No trigger/retrieval scoring was attempted (D5,
requires paid baseline transcripts per vault note 1039 — out of scope here).

## Deferred (explicitly out of scope for this pass, per task instructions)

- **Section 3** (retrieval check, trigger decision D5, shim-only validation, D7 gate) — not run; requires
  the harness fix above (done) plus real agent-phrase data this pass did not generate.
- **A preliminary no-spend similarity check per D5** — offered as optional in the task instructions ("if
  time permits, not required"); not run this pass, left for task 3.1's dedicated retrieval-check work.
