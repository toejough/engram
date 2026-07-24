# Plan: backlog bookkeeping — close #658, correct 5 stale issue bodies, reconcile ROADMAP rows

Cycle: /please, 2026-07-24. Origin: a 12-agent cross-check of all 23 open issues against
`dev/eval/LEDGER.md`, the vault, git history, and FEATURES/ROADMAP (run wf_7f46432a-ead).
Joe approved this as item 1 of the resulting briefing.

## Ask (verbatim)

> close #658 (shipped), and fix the roadmap/issue rows the triage proved stale (#683, #648,
> #701, #656, #637 bodies + the ROADMAP rank-2 row)

— approved by Joe as "Please do 1". The enumerated scope is five issue BODIES plus ONE ROADMAP
row (rank-2, #658). Everything else this cycle's grep surfaced is proposed, not folded — except
the three further ROADMAP rows Joe explicitly approved mid-cycle into Unit 4 (below), after
Gate A flagged them as unauthorized.

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
| `docs/ROADMAP.md:96` | #648 NEXT-band pointer | **update** — in scope: the "see NOW rank 9" cross-reference breaks under the authorized renumbering |
| `docs/ROADMAP.md:91` | #648 NOW rank 9 | **update (Unit 4)** — scope addition approved by Joe; omits that `activationCosineCutoff` no longer exists |
| `docs/ROADMAP.md:101` | #656 **LATER** row (band header at :98; the GATED band starts at :103 and does not contain #656) | **update (Unit 4)** — scope addition approved by Joe; names #654 as the gate, omits the shipped 93% mechanism and #677's won't-do |
| `docs/ROADMAP.md:152` | #637 **DEFERRED** row (band header at :145; the separate "Parked backlog" section at :166 does NOT contain #637) | **update (Unit 4)** — scope addition approved by Joe; omits that #639 answered the body's pain |
| `docs/ROADMAP.md:89` | #683 in rank-7 batch | **keep** — the row names the item and its low value, both still true; the re-spec lives in the issue body. Re-checked for newly-misleading omission (note 383): none — the row makes no claim about the test's contents |
| #701 | — | **N/A** — no doc references it anywhere (grep returns nothing outside GitHub) |
| `dev/eval/LEDGER.md`, `docs/FEATURES.md`, `docs/architecture/adr.md`, README, CLAUDE.md | — | **keep** — grep found no rows for these six numbers outside ROADMAP; LEDGER rows are vintage-stamped records regardless |

## Units

**Unit 1 — #658 close.** RED: the body's live claims contradicted by the tree (the #658 row of
the "Verified stale claims" table). GREEN: post a close comment citing these three commits —
`fb9bff24` (build_prompt `include_recall` flag + `recall_only_prompt` for the $METER split),
`d99f2fbd` (`split_costs` pure helper separating recall_cost from build_cost), `8425c9d0`
(split round-1 into recall-only + resumed build; billed recall_cost) — plus
`dev/eval/cumulative/harness.py:134`, `:177`, `:661`; then close the issue, remove the ROADMAP
rank-2 row, and renumber. VERIFY: `gh issue view 658` shows closed; ROADMAP NOW ranks read
1..8 with no gaps or duplicates.

**Unit 2 — five correction banners.** RED: for each issue, that issue's row in the "Verified
stale claims" table is the failing baseline. GREEN: prepend exactly this blockquote to the top
of each body, filling the three fields from that issue's table row:

```
> **Correction (2026-07-24):** <what changed, one sentence, naming the commit or mechanism>.
> Verifying: <the command or file:line from the table row, as evidence anyone can re-run>.
> This issue now reduces to: <what remains actionable, or "nothing — see the close-out
> proposal" if the premise is fully answered>.
```

Casing is `**Correction (date):**`, matching the repo's one existing instance of this convention
— #644's body, which opens `> **Correction (2026-07-07):**` above an untouched body. (#693's
banner is a different thing — a `STATUS: DEFERRED` disposition banner — and is not a precedent
for this format.) Do NOT edit, delete, or reflow any original body text; the banner supersedes
it. VERIFY: re-read each of the five bodies (#683, #648, #701, #656, #637) and
confirm every stale claim in that issue's table row is named by its banner, and that the
original text below is byte-unchanged.

**Unit 3 — ROADMAP: the authorized row only.** GREEN: remove the rank-2 #658 row, renumber
ranks 3–9 → 2–8, and repair `docs/ROADMAP.md:96`'s "see NOW rank 9" pointer, which the
renumbering itself breaks (repairing damage the authorized edit causes is in scope; changing
rows the ask did not name is not). VERIFY: NOW ranks are contiguous 1..8 with no gaps or
duplicates; no cross-reference to a rank number is left dangling; band/rank rules at
ROADMAP:41-53 still hold. No row moves band — band changes are dispositions, and dispositions
are Joe's.

**Unit 4 — three further ROADMAP rows (scope addition, APPROVED by Joe 2026-07-24).** The
doc-surface grep found three stale rows the original ask did not name. Gate A's ask-alignment
reviewer correctly flagged folding them in as unauthorized scope; they were put to Joe, who
chose "fold all three in." GREEN: `:91` (#648's row — state that `activationCosineCutoff` no
longer exists, so the remaining lever is the note half-life alone), `:101` (#656's row — record
that the mechanism shipped via #655 at a 93% fire-rate and #677 closed the enforcement
residual won't-do), `:152` (#637's row — record that #639's repeatable `--phrase` flag answered
the body's stated pain). VERIFY: each row reads true against the tree; no row changes band.

## Gates

Gate A (this plan): ask-alignment, code-alignment, docs/diagrams-alignment, clarity/standards.
Gate B: design-fit after Unit 3's and Unit 4's ROADMAP edits. Gate C: ROADMAP (the only doc
touched). Gate D: all six issue-comment/banner texts + commit messages — every number
and provenance claim grepped from the record (note 248).

Commits: one for the ROADMAP reconciliation (issue-body edits live on GitHub, not in git),
`AI-Used: [claude]`, ff-only main.

## Proposed dispositions for Joe at close-out (NOT acted on here)

- #656, #637, #701: their premises are answered or collide with settled constraints. Candidates
  for closing, but that is Joe's call.
