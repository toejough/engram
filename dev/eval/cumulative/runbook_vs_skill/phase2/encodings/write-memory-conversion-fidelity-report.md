# Conversion-Fidelity Report: `write-memory/SKILL.md` -> WM-R runbook

OpenSpec change `recall-learn-writememory-to-runbook`, tasks 2.1 (plan) and 2.5 (final). Format mirrors
`please-conversion-fidelity-report.md` / `curate-conversion-fidelity-report.md`.

- **Source**: `agent-instructions/skills/write-memory/SKILL.md` (140 lines, 6,943 bytes; frozen byte-identical
  copy at `encodings/taskWriteMemory/WM-S/skills/write-memory/SKILL.md`, re-verified identical this session).
- **Encoding**: `encodings/taskWriteMemory/WM-R/vault/6a.2026-09-22.write-memory-compose-execute-verify.md`,
  `type: runbook`, one note (the whole skill fits one retrieved item, design D1 — no sub-runbooks), with a
  `.vec.json` sidecar. Built with the real `engram learn runbook` in a scratch vault (`--position top`
  required explicitly; the CLI does not default it), then the luhmann id set to `6a` and re-embedded
  (`engram embed apply --vault <dir> --all`; `engram embed status`: total 1, with-embeddings 1, stale 0,
  broken 0, incompatible 0). No `vocab.centroids.json` shipped in the carrier dir (would overwrite the seed
  vault's own on merge, per curate's precedent). `engram show 6a.2026-09-22.write-memory-compose-execute-verify
  --vault <fixture>` resolves and prints no `EARLIER RED_FLAGS OMITTED` marker.

| Field | Value |
|---|---|
| `situation` | composing, executing, and verifying a vault write that a parent skill has already decided to make -- recall or learn judged what to write and why; this is only the compose/execute/verify/report of that decision, never a fresh judgment about whether or what to write |
| `done_when` | the composed engram learn (or amend) command has run, its success has been verified -- or, after up to 2 retries, the exact failing command and CLI error have been reported verbatim -- and the written note path(s) have been reported back to the parent that handed off the write |
| `triggers` | **none** (design D2 — write-memory has no trigger of its own; no `--trigger` flag was passed to `engram learn runbook`, so no `triggers:` key appears in the note at all — confirmed by reading the built file) |
| `red_flags` bytes | 736 of 1200 (the exact `    - ...` block `capRedFlagsForPreview` measures, computed directly from the raw frontmatter with the same scan rule as `internal/cli/redflags_truncation.go`; corroborated by `engram show` printing no omission marker) |
| body bytes | 6,685 (measured as the bytes after the closing `---\n` of the frontmatter) |

## Luhmann numbering (`6a`, not a plain top-level id)

The task fixture's `vault-template/` seeds occupy ids 1-6. `TASK-RATIONALE.md` (task 1.3) already validated
that, with only those 6 seeds present, the trial's two write-memory writes land at ids **7** and **8** —
`done_when_checks.sh` hardcodes exactly `note 7` / `note 8` by literal filename glob (`ls "$V"/7.*.md`).
`internal/cli/luhmann.go`'s `nextTopLevel` computes the next id as `max(existing single-segment top-level
ids) + 1`, scanning **every** note in the vault, not just the seeds. A plain top-level carrier id (the naive
choice, e.g. `9`, following curate's "one past the seeds" convention) would raise that max to 9 and shift
the agent's two writes to **10** and **11**, silently breaking `done_when_checks.sh`'s hardcoded 7/8 checks
once the carrier is merged into a trial vault for the R arm (curate never hit this because curation writes
no new notes, so its own top-level `10` was harmless).

`nextTopLevel` only counts ids where `ParseID` yields exactly one segment — a letter-suffixed id like `6a`
parses to two segments (`"6"`, `"a"`) and is skipped from the max computation. Using `6a` for the carrier
keeps `nextTopLevel(1..6) == 7` regardless of the carrier's presence, so the reserved write-site ids (7, 8)
stay correct. This is a judgment call, not something the task instructions specified — flagged for the
2.6 reviewer. Verified live: `probe_phase2.py --task write-memory --arms R --setup-only` reports
`carrier_basename=6a.2026-09-22.write-memory-compose-execute-verify` and `vault .md count after setup: 7`
(6 seeds + 1 carrier), and `--arms S --setup-only` still reports `vault .md count after setup: 6` — both
setups build clean with no vault-health failure.

`task.json` gained a `carrier_r_src: "encodings/taskWriteMemory/WM-R/vault"` entry (previously absent by
design, per `TASK-RATIONALE.md`, since the runbook didn't exist yet) — outside the literal 2.1-2.5 list but
required for the `--setup-only` R-arm check the task asked me to run; flagged as an addition.

## Section plan and disposition (2.1)

Tags: **carry** = verbatim (or near-verbatim) in the runbook body; **field** = moved to a structured
frontmatter field; **strengthened** = carried but made more explicit than the source; **drop** = not carried
(with reason).

| # | SKILL.md section (lines) | Tag | Where / what changed |
|---|---|---|---|
| S1 | Frontmatter `description` (1-8): "invoked by recall/learn... requires a handoff — do not fire on your own judgment" | field + drop | The "when/why" content becomes `situation` (reworded, process-shaped, names no eval-task domain — no printer farm, no tags). The "do not fire on your own judgment" clause is NOT a `triggers:` field (design D2) — this is the literal justification for D2: a `triggers:` field would let this runbook surface on a stray mention with no parent judgment behind it, contradicting this very sentence. The clause's behavioral content ("do not decide WHETHER to write") is also carried into the body intro (S2). |
| S2 | Title + intro paragraph (10-14) | carry | Verbatim, including the H1 title — `SKILL.md:10` already has the exact heading `# Write Memory — execute a handed-off vault write` in-body; it is carried unchanged as the runbook body's opening heading, not added (correction: an earlier draft of this report mistakenly claimed the H1 was added because "skills don't have one in-body," which is false here). |
| S3 | `## The handoff contract` (16-38) | near-verbatim | Content identical: kind, content fields, source, chunk-sources, tags, supersedes, position/target, red_flags, triggers, and the "ask, don't invent" closing rule. One diff: the heading itself was shortened from `## The handoff contract` to `## The handoff` (confirmed against the built note, line 22). |
| S4 | `## Compose` intro + kind=feedback block (40-50) | carry | Verbatim. |
| S5 | kind=fact block (52-60) | carry | Verbatim, including the position/target defaults paragraph (62-64 in the source). |
| S6 | kind=runbook block (66-77) | carry | Verbatim. |
| S7 | red-flag/trigger disposition + trigger-authoring rule (79-95) | near-verbatim | This is "the trigger-authoring rule text" named in the task description (how `--trigger` values are chosen when write-memory is asked to compose a DIFFERENT runbook note; unrelated to write-memory's own lack of triggers, S1/D2). One diff: the parenthetical example "(e.g. \"please\" on the please runbook, 2026-09-20)" (SKILL.md:92) lost its date in the runbook body, which reads "(e.g. \"please\" on the please runbook)" — confirmed against the built note. |
| S8 | kind=qa block (97-107) | carry | Verbatim. |
| S9 | "Append to any kind" bullets (109-117) | carry, strengthened | Chunk-source and supersedes bullets verbatim. The `--tag` bullet is strengthened (see next section) — the planted `--tag`/`--tags` divergence (`TASK-RATIONALE.md`) must be stated "at least as clearly" as the source; I made it more explicit than the source (which never contrasts against a plural form) rather than just matching it, since under-stating a divergence the eval is specifically built to test would weaken the runbook arm unfairly relative to the skill arm. |
| S10 | Rules: never mix kind flags; never hand-author vocab/Supersedes; wikilink citations are fine (119-130) | carry | Verbatim. This paragraph is the real-`SKILL.md` grounding for the wikilink-citation red_flag (below) — the rule already discusses `[[basename]]` prose citations at length; the red_flag makes the "you wrote plain text instead" failure mode explicit, the same design move please/curate made (SKILL.md never phrases it as a warning, but the underlying rule is stated). |
| S11 | `## Execute, verify, report` (132-140) | carry | Verbatim. |

**Nothing was dropped as shim-floor.** Unlike `please`/`curate`, `write-memory/SKILL.md` carries no generic
agentic-behavior content (anti-sycophantic lean, task tracking, announce/restate, N/A handling, …) that
overlaps `shim.md`'s follow-frame — it is a pure, task-specific compose/execute/verify procedure with no
orchestration overhead (matches proposal.md's own description: "no red-flags table, ... a pure worker ...
with no depth-dial or lesson-kind branching"). The entire body is carried.

## The `--tag`/`--tags` divergence (task 2.2's explicit requirement)

`SKILL.md` states the singular, repeatable form five times (lines 24, 49, 59, 76, and the "Append to any
kind" bullet at 112) but never contrasts it against a plural `--tags`. The runbook body keeps every one of
those five occurrences verbatim, **plus** bolds and expands the "Append to any kind" bullet to read: "one
`--tag <t>` per provided tag — a singular, repeatable flag, one `--tag` per tag. There is no plural `--tags`
list flag; the CLI does not accept it." — strictly more explicit than the source, never weaker, per the
task's instruction. This is also `red_flags` entry 2 (below), giving the divergence two independent
surfaces in the fetched runbook.

## `red_flags` (4 entries, 736 bytes): derivation from the real `SKILL.md`

| # | `red_flags` entry (paraphrased) | Grounded in `SKILL.md` at |
|---|---|---|
| 1 | Mixed fact/feedback/runbook flags in one command | Lines 119-122 ("Never mix fact flags ... feedback flags ... or runbook flags ... in one command") — verbatim rule, elevated to a red_flag. |
| 2 | Wrote `--tags` (plural) instead of singular repeatable `--tag` | Lines 24, 49, 59, 76, 112's consistent singular usage (never itself phrased as a warning in SKILL.md — this is proposal.md's own named divergence, confirmed against `internal/cli/targets.go`'s `Tags []string` bound to `targ:"flag,name=tag"` (singular, repeatable), not a plural list flag). |
| 3 | Silently dropped a qa handoff's tags instead of the exact line `tags dropped: qa takes no tag flags` | Lines 112-115, which name that exact literal string — carried verbatim into the body and restated as a red_flag. |
| 4 | Wrote a plain-text reference instead of an inline `[[basename]]` wikilink | Lines 123-130's wikilink-citation rule (design-added as a red_flag, matching please/curate's own W6/route precedent — SKILL.md discusses the rule but never frames it as a failure mode). |

These are exactly proposal.md's four named candidates (flag-mixing, `--tags` vs `--tag`, qa-takes-no-tags,
wikilink-not-plain-text) — nothing invented, nothing dropped. No 1200-byte pressure, as design D1 predicted
(736 of 1200, ample headroom for future additions).

## Call-site edits: `recall`/`learn` fixture copies (task 2.4)

Re-grepped `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskWriteMemory/recall-learn/{recall,learn}/SKILL.md`
directly (confirmed byte-identical to the live `agent-instructions/skills/{recall,learn}/SKILL.md` files
before editing, so these line numbers also match the live files today — that will drift as either evolves).
Found **9** literal "invoke the write-memory skill" (or equivalent) directives — not the "ten call sites"
figure in design.md's own prose, which appears to be an off-by-one in that document (3 in recall + 6 in
learn = 9, by my count and by design.md's own line list `recall:192,284,307` + `learn:119,127,138,189,199,225`).
This also reconciles against `tasks.md` 2.4, the binding instruction for this step: its own line list
(`recall/SKILL.md:192,284,307,339`; `learn/SKILL.md:25,119,127,138,189,199,225`) incorrectly named two
extra lines as call sites — `recall:339` and `learn:25` — which are prose mentions of write-memory, not
invocation directives (see the "left untouched" list below); both documents are corrected in the 2.6
review to the same 9-site list, with the two-line spans (137-138, 189-190, 199-200) stated explicitly.
All 9 are converted; every other mention of the bare word "write-memory" (naming the worker/process, not
directing an invocation) was left untouched, per the "carrier-language-only edit, not a rewrite" instruction:
`recall/SKILL.md:293` ("...in the write-memory handoff"), `recall/SKILL.md:339` (red-flags table row),
`learn/SKILL.md:25` ("hand off to write-memory as always"), `learn/SKILL.md:98` ("...in the write-memory
handoff..."), `learn/SKILL.md:231-232` ("...include the basenames in the write-memory handoff... if
write-memory reports a contributor rejection...") — all generic references to the worker/handoff, not
"invoke ... skill" directives.

**Judgment call, flagged for 2.6**: at each of `learn/SKILL.md`'s five bolded `**REQUIRED SUB-SKILL:**`
labels, I also changed the label itself to `**REQUIRED NEXT ACTION:**`, since "SKILL" in that label is
carrier language too (the whole point of the conversion is that write-memory is no longer a skill) and
leaving it unchanged would read as "REQUIRED SUB-SKILL: fetch and follow `[[wikilink]]`" — an internally
inconsistent sentence. The task's own example only named the "invoke the write-memory skill" phrase, so this
extra label edit is my own extension of the same principle, not something explicitly asked for.

| File:line (before edit) | Before | After |
|---|---|---|
| `recall/SKILL.md:192` | `Invoke the **write-memory** skill with this handoff — kind=fact or feedback ...` | `Fetch and follow` `` `[[6a.2026-09-22.write-memory-compose-execute-verify]]` `` `with this handoff — kind=fact or feedback ...` |
| `recall/SKILL.md:284` | `Hand ONE synthesis note per conclusion to the **write-memory** skill (kind=fact or feedback, per the conclusion's shape):` | `Fetch and follow` `` `[[6a.2026-09-22.write-memory-compose-execute-verify]]` ``, handing it ONE synthesis note per conclusion (kind=fact or feedback, per the conclusion's shape):` |
| `recall/SKILL.md:307` | `ALSO invoke the write-memory skill** with kind=qa ...` | `ALSO fetch and follow` `` `[[6a.2026-09-22.write-memory-compose-execute-verify]]` ``** with kind=qa ...` |
| `learn/SKILL.md:119` | `**REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — kind=feedback, ...` | `**REQUIRED NEXT ACTION:** fetch and follow` `` `[[6a.2026-09-22.write-memory-compose-execute-verify]]` `` `with this handoff — kind=feedback, ...` |
| `learn/SKILL.md:127` | `**REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — kind=fact, ...` | `**REQUIRED NEXT ACTION:** fetch and follow` `` `[[6a.2026-09-22.write-memory-compose-execute-verify]]` `` `with this handoff — kind=fact, ...` |
| `learn/SKILL.md:137-138` | `For each reversal, **REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — kind=feedback, ...` | `For each reversal, **REQUIRED NEXT ACTION:** fetch and follow` `` `[[6a.2026-09-22.write-memory-compose-execute-verify]]` `` `with this handoff — kind=feedback, ...` |
| `learn/SKILL.md:189-190` | `For a **runbook**, **REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — kind=runbook, ...` | `For a **runbook**, **REQUIRED NEXT ACTION:** fetch and follow` `` `[[6a.2026-09-22.write-memory-compose-execute-verify]]` `` `with this handoff — kind=runbook, ...` |
| `learn/SKILL.md:199-200` | `For **feedback**, **REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — kind=feedback, ...` | `For **feedback**, **REQUIRED NEXT ACTION:** fetch and follow` `` `[[6a.2026-09-22.write-memory-compose-execute-verify]]` `` `with this handoff — kind=feedback, ...` |
| `learn/SKILL.md:225` | `For each uncaptured substantive Q&A from this session, **invoke the write-memory skill** with this handoff — kind=qa, ...` | `For each uncaptured substantive Q&A from this session, **fetch and follow` `` `[[6a.2026-09-22.write-memory-compose-execute-verify]]` ``** with this handoff — kind=qa, ...` |

Wikilink syntax and "fetch and follow" phrasing mirror design.md D3's own prescribed wording ("fetch and
follow `[[<basename>]]` with this handoff") and the general `[[wikilink]]` convention already used
throughout the please/curate conversions and `shim.md` frame item 6. The real, currently-live
`agent-instructions/skills/{recall,learn,write-memory}/SKILL.md` files are **untouched** (`git status`
confirms no changes under `agent-instructions/skills/`); only the fixture copies under
`encodings/taskWriteMemory/` changed.

## Deliberately NOT changed

- `steps.json`, `done_when_checks.sh`, `task-prompt.txt`, `init_fixture_repo.sh`: unchanged (section 3 / task
  1.x scope).
- `WM-S/skills/write-memory/SKILL.md`: unchanged, re-verified byte-identical to the live skill.
- No fixture-specific hints (printer farm, tags, drafts) leaked into the runbook's `situation`/`done_when`/body.
- The five real-`SKILL.md` mentions of `[[full-basename]]`/`[[basename]]` as prose-citation syntax elsewhere
  in `recall`/`learn` (unrelated to write-memory call sites) were left alone.

## Open items for the 2.6 fresh-context reviewer

1. The `6a` luhmann-id choice (above) — confirm the reasoning and that it doesn't create some other
   downstream oddity I didn't anticipate (e.g., any code path that assumes single-segment top-level ids for
   carrier notes specifically).
2. The `**REQUIRED SUB-SKILL:**` → `**REQUIRED NEXT ACTION:**` label edits — confirm this is in scope for a
   "carrier-language-only" edit rather than over-reach.
3. The "ten call sites" vs. my count of 9 — re-verify no tenth site was missed (I re-grepped case-insensitively
   for "write-memory" in both fixture files and accounted for every hit; see the "left untouched" list above).
4. `task.json`'s new `carrier_r_src` entry — confirm this is acceptable to add now rather than waiting for
   section 3.

## 2026-09-22 — Task 3.4 iteration after task 3.2's D8 miss

Task 3.2 (n=3) failed the validity gate 0/3: no trial ran `engram show <basename>` before composing a
write. The transcripts (`results/3.2_write_memory_shim_only_sonnet5.md`) show the model reading
"fetch and follow `[[wikilink]]`" as a cue to invoke the **Skill tool** with `name: write-memory`
(a reflex carried from calling `Skill{recall}`/`Skill{learn}` moments earlier in the same session),
getting `Unknown skill: write-memory`, and then composing the `engram learn`/`amend` command from
memory/`--help` instead of recovering via `engram show`. Joe approved fixing both named levers
(option C):

- **All 9 call sites reworded** in `encodings/taskWriteMemory/recall-learn/{recall,learn}/SKILL.md`
  from "fetch and follow `[[<basename>]]`" to name the literal command, e.g.: "run `engram show
  6a.2026-09-22.write-memory-compose-execute-verify` in Bash (a shell command — write-memory is not a
  Skill tool) and follow what it prints." Carrier-language-only — every other word in each call site is
  unchanged (verified: `grep -c "engram show 6a.2026-09-22.write-memory-compose-execute-verify"` returns
  3 in `recall/SKILL.md` + 6 in `learn/SKILL.md` = 9; `grep "fetch and follow"` returns zero hits in
  either file after the edit).
- **A 5th `red_flags` entry added** to the WM-R runbook naming the trap explicitly: "You are about to
  invoke the Skill tool with name write-memory -- stop, it no longer exists as a skill; run `engram show
  6a.2026-09-22.write-memory-compose-execute-verify` in Bash instead and follow what it prints." Applied
  via `engram amend --target 6a.2026-09-22.write-memory-compose-execute-verify --red-flag ... (x5,
  replace-whole semantics — all 4 prior entries re-passed verbatim plus the new one)` against the fixture
  vault directly — never hand-edited, per vault note 958. New size: 5 entries, 955/1200 bytes (was 4
  entries, 736/1200) — `engram show` prints no `EARLIER RED_FLAGS OMITTED` marker. Sidecar rebuilt via
  `engram embed apply --vault <fixture> --all`; `engram embed status` reports total 1, with-embeddings 1,
  stale 0, broken 0, incompatible 0.
- Both edits are fixture-only; the live `agent-instructions/skills/{recall,learn}/SKILL.md` and the
  production vault are untouched pending a fresh D8 pass (task 3.4's re-run).
- Note: running `engram amend` directly against the `WM-R/vault` directory (as its own standalone vault)
  produced a `.luhmann.lock` (0 bytes) and a `vocab.centroids.json` (empty terms, 116 bytes) as a side
  effect — not present when this directory was first built (task 2.2). This matches existing precedent in
  other already-used R-arm carrier dirs (`encodings/taskPlease/Please-R/vault`,
  `encodings/taskRoute/Route-R/vault` both carry the same two files), so left in place rather than
  special-cased; flagging for visibility since task 2.2's own report explicitly noted the carrier
  originally shipped with neither file.
