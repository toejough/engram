# Conversion-Fidelity Report: `curate/SKILL.md` -> Curate-R runbook

OpenSpec change `curate-skill-to-runbook`, tasks 2.1-2.6. Format mirrors `please-conversion-fidelity-report.md`.

- **Source**: `agent-instructions/skills/curate/SKILL.md` (83 lines, 5,724 bytes; frozen byte-identical copy at
  `encodings/taskCurate/Curate-S/skills/curate/SKILL.md`, pinned by a test).
- **Encoding**: `encodings/taskCurate/Curate-R/vault/10.2026-09-21.curate-review-pending-offers.md`, `type: runbook`, one
  note (the 5.7 KB skill fits one retrieved item, so no sub-runbooks; design D1), with a `.vec.json` sidecar
  (`engram embed apply --vault <dir> --all`; `engram embed status`: total 1, stale 0, broken 0). Built with the real
  `engram learn runbook` in a scratch vault, then luhmann set to `10` (the seed vault holds 1-9) and re-embedded. No
  `vocab.centroids.json` is shipped (it would overwrite the seed vault's own on the carrier copy).

| Field | Value |
|---|---|
| `situation` | reviewing pending offers in a vault and deciding, for each, whether existing notes already cover it, nearly cover it, or do not |
| `triggers` | `curate`, `/curate`, `pending offers`, `pending offer` |
| `red_flags` bytes | 931 of 1200 (the exact `    - ...` block `capRedFlagsForPreview` measures; `engram show --vault <fixture>` prints no `EARLIER RED_FLAGS OMITTED` marker; pinned by `test_curate_r_red_flags_fit_the_redflags_preview_budget`) |
| body | 4,212 bytes |

The situation is process-shaped and names nothing from the eval task (no beekeeping, no offer number, no example);
pinned by `test_curate_r_triggers_and_process_shaped_situation`. `carrier_basename` = the only `.md` in the carrier dir.

## Section plan and disposition

Tags: **carry** = in the runbook body; **shim** = dropped because `shim.md` states it for every runbook; **field** =
moved to a structured frontmatter field; **adapted** = reworded for the new carrier.

| # | SKILL.md section | Tag | Where / what changed |
|---|---|---|---|
| S1 | Frontmatter `description` (four surfacing signals + explicit ask + host-local-only) | field + adapted | Becomes `situation` (process-shaped) and `triggers`. The four signals map: query payload flag `pending_offers: true` (a bare boolean, no text; NOT a firing signal any more, see D4 of the change), `engram update` notice and write-path nudge (both now print a trigger query, task 2a), "engram serve has just accepted a served write" (an observation, not a printed line; dropped as a firing signal). Host-local-only survives in the body. |
| S2 | Title + first paragraph (what a pending offer is; curation = recall Step 2.5 reasoning applied to an offer) | carry | Verbatim. |
| S3 | "Host-local only, on your own initiative" paragraph | carry, adapted | Server-path sentence verbatim. The last sentence changes from "Invoke this skill reactively, whenever you notice one of the three surfacing signals ..., or whenever explicitly asked" to "Run this runbook when you are asked to curate, review, or triage pending offers, or when engram itself tells you offers are waiting (the `engram update` report and the write-path nudge ...)"; the notice-driven route is described because it is now the automatic firing path. |
| S4 | Step 1 (query excludes offers; scan with `grep -l '^pending: true$' <vault>/*.md`; read each in full) | carry | Verbatim, including the grep (offers are excluded from query by design, so the runbook cannot rely on retrieval to list them). |
| S5 | Step 2 intro + covered/near/absent table (criteria + exact `engram amend` actions) | carry | Verbatim. |
| S6 | "Judge content, never a cosine/similarity score alone" | carry | Verbatim (also red flag 6). |
| S7 | "Never hand off to `write-memory`" | carry | Verbatim (also red flag 4). Kept: no shim analogue (it is curate-specific). |
| S8 | Step 3 (why covered and near both discard the offer) | carry | Verbatim: it is the rationale that separates near-and-discard from near-and-clear, the exact failure of the bare arm (0/3). |
| S9 | Step 4 (`engram check`, confirm no longer pending) | carry | Verbatim. |
| S10 | Red flags table (7 rows) | field | See placement table. |
| S11 | (implicit) skill-invocation language: "this skill", "invoke this skill" | adapted | "this skill" -> "this runbook" in S3; no other occurrence in the body. |
| S12 | (implicit) shim floor: announce, one todo per step, done_when bar, stop-and-ask, shortcut-is-a-cue | drop-as-shim | The SKILL.md carries none of these explicitly. `done_when` is NEW (the skill had none): it restates the end state the eval scores (every offer judged by content, per-outcome action, no `engram learn`, `engram check`, no pending left). |

### Red flags placement (7 rows -> 7 `red_flags` entries, 931 bytes)

Every row is curate-specific (none is the generic floor), so all seven stay in `red_flags`; the cap holds without dropping any.
Wording is tightened to `<sign> -- <action>` (`--` per please's convention) and shortened; meaning unchanged.

| # | Skill row (sign) | `red_flags` entry starts with |
|---|---|---|
| 1 | You rewrote the OFFER note's own content | "Rewriting the OFFER note's own content" |
| 2 | You cleared an offer's marker after folding or discarding its content elsewhere | "Clearing an offer's marker after its content was folded into or found covered by another note" |
| 3 | You ran `engram learn` for the absent case | "Running engram learn for the absent case" |
| 4 | You handed a judgment off to `write-memory` | "Handing a judgment to write-memory" |
| 5 | You used `engram query` to find pending offers | "Using engram query to find pending offers" |
| 6 | You applied a cosine threshold to decide | "Deciding covered/near/absent from a cosine or similarity score" |
| 7 | You curated from inside a served HTTP request | "Curating from inside a served HTTP request" |

### Deltas summary

| ID | Delta | Reason |
|---|---|---|
| C1 | `situation` + `triggers` replace the `description` | A runbook has no harness-read description; retrieval uses situation (semantic) and triggers (literal, whole-word). |
| C2 | Firing signals: three -> two printed notices, both carrying the trigger query | The query payload flag is a bare boolean; the served-write observation is not printed. |
| C3 | `done_when` added | Runbooks carry a completion bar; the skill had none. Derived from the skill's own outcomes and Step 4. |
| C4 | No wikilinks, no sub-runbooks, no syntax note | The runbook does not ask the agent to cite vault notes; the skill fits one note. |
| C5 | Red flags: table -> `red_flags` field | Structured field surfaced by `engram show`/`query` with the note. |
| C6 | Skill-invocation language -> runbook language | Carrier change only. |

## Behavior unchanged

Steps, per-outcome actions, the grep, and the rationale are byte-for-byte the skill's text (the body is the skill minus
the description, the red-flags table and the invocation sentence). The spec/skill disagreement about near and covered offers
(design D5 of the change) is NOT resolved here: the runbook follows the skill, which is what the eval scores.

## Open

- Task 2.7: fresh-context review of this report against `SKILL.md` for lost content is not done in this stage.
