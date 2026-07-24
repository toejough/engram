# Plan: backlog bookkeeping — close #658, correct 5 stale issue bodies, reconcile ROADMAP rows

Cycle: /please, 2026-07-24. Origin: a 12-agent cross-check of all 23 open issues against
`dev/eval/LEDGER.md`, the vault, git history, and FEATURES/ROADMAP (run wf_7f46432a-ead).
Joe approved this as item 1 of the resulting briefing.

## Ask (verbatim scope)

Close #658 as already-shipped; correct the stale bodies of #683, #648, #701, #656, #637;
reconcile `docs/ROADMAP.md`'s rows to match, starting with the rank-2 #658 row.

**Out of scope — and deliberately so:** closing #701, #656, or #637. Their bodies are stale
enough that closing is arguable, but a disposition is Joe's call (vault note 335), so this
cycle corrects the record and proposes dispositions at close-out. No issue is closed here
except #658, which Joe named explicitly.

## Verified stale claims (each re-checked against the working tree, not the triage summary)

| Issue | Body claim | Tree reality | Verifying command |
| --- | --- | --- | --- |
| #658 | recall's $ is bundled into `build_cost`; no `recall_cost` field | AC option (b) shipped 2026-06-26 — `dev/eval/cumulative/harness.py:134,177,661` bill recall on its own call and emit `recall_cost` | `grep -n "recall_cost\|include_recall" dev/eval/cumulative/harness.py` |
| #683 | drop unused params of `buildTestFrontmatterWithLegacyVocab`; oracle asserts "`vocab:` removed" | helper deleted; `WriteVocabAssignment` touches only `tags:` (commit 44a62922, #681) | `grep -rn "buildTestFrontmatterWithLegacyVocab" --include='*.go' .` → no hits |
| #648 | tune `activationCosineCutoff`; blocked on #642 and #646 (both OPEN) | constant deleted (commit efaa2fcd); #642 closed 2026-07-18, #646 closed 2026-07-19 | `grep -rn "activationCosineCutoff" --include='*.go' .` → no hits |
| #701 | branched Luhmann IDs "lost somewhere, not sure where" | lost in commit f620bfaf (2026-06-12, recall/learn v2 rewrite); documented as intended in `agent-instructions/skills/learn/tests/baseline-clean-write.md:22,31,37`. The crosslink half meets ROADMAP:162-164's standing constraint on new edge types | `git show f620bfaf --stat`; `sed -n '160,166p' docs/ROADMAP.md` |
| #656 | needs the re-entry gate; blocked on #654 | recall Step 3.5 shipped via #655 (`agent-instructions/skills/recall/SKILL.md`, 10 `Re-entry` mentions), measured 93% fire-rate; #654 closed; #677 closed the enforcement residual won't-do | `grep -c "Re-entry" agent-instructions/skills/recall/SKILL.md` → 10 |
| #637 | callers must jam subject/behavior/time into one string and hope | answered by #639's repeatable `--phrase` flag — now the production recall path (8 `--phrase` mentions in the recall skill) | `grep -c '\--phrase' agent-instructions/skills/recall/SKILL.md` → 8 |

## Doc-surface enumeration grep (author-run: `#<issue-number>` across docs/, README.md, CLAUDE.md, LEDGER)

| File:line | Ref | Disposition |
| --- | --- | --- |
| `docs/ROADMAP.md:84` | #658 NOW rank 2 | **remove** — issue closes as shipped; renumber ranks 3–9 → 2–8 |
| `docs/ROADMAP.md:91` | #648 NOW rank 9 | **update** — row's scope note covers #646's shift but not that `activationCosineCutoff` no longer exists; state the remaining lever is the note half-life alone |
| `docs/ROADMAP.md:96` | #648 NEXT-band pointer | **update** — the "see NOW rank 9" cross-reference breaks under renumbering |
| `docs/ROADMAP.md:101` | #656 GATED row | **update** — names #654 as the gate without recording that the mechanism shipped at 93% and #677 closed the residual |
| `docs/ROADMAP.md:152` | #637 PARKED row | **update** — "no forcing function" is right but omits that #639 already answered the body's stated pain |
| `docs/ROADMAP.md:89` | #683 in rank-7 batch | **keep** — the row names the item and its low value, both still true; the re-spec lives in the issue body. Re-checked for newly-misleading omission (note 383): none — the row makes no claim about the test's contents |
| #701 | — | **N/A** — no doc references it anywhere (grep returns nothing outside GitHub) |
| `dev/eval/LEDGER.md`, `docs/FEATURES.md`, `docs/architecture/adr.md`, README, CLAUDE.md | — | **keep** — grep found no rows for these six numbers outside ROADMAP; LEDGER rows are vintage-stamped records regardless |

## Units

**Unit 1 — #658 close.** RED: list the body's live claims contradicted by the tree (above).
GREEN: post a close comment citing the three 2026-06-26 commits + the `harness.py` line
numbers, close the issue, remove the ROADMAP rank-2 row and renumber. VERIFY: `gh issue view
658` shows closed; ROADMAP ranks are 1..8 with no gaps.

**Unit 2 — five correction banners.** RED: per issue, the verified stale-claim list above is
the failing baseline. GREEN: prepend a dated `> **CORRECTION (2026-07-24):**` blockquote to
each body naming what changed, with the verifying evidence, and what the issue reduces to now.
Do NOT rewrite the original body text — the banner supersedes it (matching #693's and #644's
existing correction-banner convention). VERIFY: re-read each body; every listed stale claim is
addressed by the banner.

**Unit 3 — ROADMAP row reconciliation.** GREEN: apply the four `update` dispositions above.
VERIFY: `grep -n "#6\{0,1\}[0-9]\{2\}" docs/ROADMAP.md` rows for these issues read true against
the tree; band/rank rules at ROADMAP:41-53 still hold (no row moves band in this cycle —
band changes are dispositions, and dispositions are Joe's).

## Gates

Gate A (this plan): ask-alignment, code-alignment, docs/diagrams-alignment, clarity/standards.
Gate B: design-fit after Unit 3's edits (the only multi-part refactor). Gate C: ROADMAP (the
only doc touched). Gate D: all six issue-comment/banner texts + commit messages — every number
and provenance claim grepped from the record (note 248).

Commits: one for the ROADMAP reconciliation (issue-body edits live on GitHub, not in git),
`AI-Used: [claude]`, ff-only main.

## Proposed dispositions for Joe at close-out (NOT acted on here)

- #656, #637, #701: their premises are answered or collide with settled constraints. Candidates
  for closing, but that is Joe's call.
