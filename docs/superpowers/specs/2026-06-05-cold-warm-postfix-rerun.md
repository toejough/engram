# Cold/warm convergence re-run — POST-FIX (2026-06-05)

Re-run of the `2026-05-30-cold-warm-todo-test.md` methodology against the
**fixed** engram system (Phase-8 fixes D1–D6 + FAIL set), to validate the
fixes deliver value. Same hidden 18-item todo spec; reviewer (me) drips
spec-free feedback a cluster at a time; executor = headless `claude -p`
(sonnet) iterating to **full 18/18** (the user asked for literal full
convergence, not the prior 80% bar).

## Setup (what makes this a valid value test)
- **Fixed binary** built from `fix/memory-rigor` (has check/migrate-links/
  migrate-episodes/resituate + `--synthesis`); isolated, not the stale global.
- **Fixed skills** (repo `skills/{recall,learn,please}`, with the §6a/§6b edits).
- **WARM vault** = curated `l2` (7 todo-derived Go-CLI architecture facts),
  re-embedded with the fixed binary, `engram check` clean. **COLD** = empty.
- **Recall forced to fire** (the M1 kill-shot fix): the build prompt explicitly
  invokes recall; verified `engram query` fired in both arms (the prior dud had
  zero memory invocation). Driver is recall→build→learn (NOT `/please`, which
  ran 49 min/build headless and is impractical for a multi-round loop).
- **Name-agnostic scoring** (the prior methodology's load-bearing lesson + vault
  note): score the *pattern* (any injected persistence interface = DI), never the
  vault's prescribed vocabulary (`Store`), else the scorer favors memory arms.

## Result — full convergence to 18/18

| round | WARM (memory) | COLD (no memory) |
|---|---|---|
| R1 (blind build) | **11/18** (arch 6/6, feat 5/12) — $3.05 | 2/18 (arch 2/6) — $0.90 |
| R2 | 15/18 — $1.38 | 4/18 — $1.36 |
| R3 | 17/18 — $1.29 | 9/18 — $1.65 |
| R4 | **18/18 ✓** — $0.83 | 10/18 — $2.25 |
| R5 | — | 16/18 — $2.26 |
| R6 | — | **18/18 ✓** — $1.02 |
| **to 18/18** | **4 rounds · 3 review turns · $6.56** | **6 rounds · 5 review turns · $9.44** |

All 12 builds across both arms compiled and passed their own tests at every round.

## Answers (cold vs memory, to literal 18/18)
- **Priming (round-1 spec-match):** memory **11/18 vs 2/18**, architecture **6/6 vs 2/6** — recall surfaced the conventions and the agent applied them autonomously, zero feedback. (Pre-fix this was ~equal because capture failed; post-fix the curated vault + fixed recall front-load strongly.)
- **Rounds to full convergence:** **4 vs 6** (memory 33% fewer).
- **Human-review interactions:** **3 vs 5** (memory 40% fewer) — the autonomy signal the prior methodology identified as memory's real value.
- **Cost:** **$6.56 vs $9.44** (memory ~30% cheaper). CORRECTED (post-antagonist): cold cost more because it did **~2.7× more output work** (206K vs 78K output tokens over 2 more rounds), NOT a resume-escalation tax (my original claim here was backwards — warm's round-1 actually had the higher cache-read cost). Cost is **downstream of the round count, not an independent metric.**
- **Sufficiency:** both reach 18/18 — memory is **not** necessary; its value is faster/cheaper/more-autonomous convergence, reproducing the prior (pre-fix) finding with the fixed mechanics.

## Honest caveats
- **n=1.** Cost/round counts carry single-trial noise (warm's $3.05 round-1 especially).
- **Residual circularity on architecture:** the l2 vault's 7 facts overlap the 6 architecture rubric items, so warm's architecture front-loading is partly the vault carrying those conventions. The **feature** front-loading (round-1 5/12 vs 0/12) and the rounds/cost metrics are less circular; the blind spec decouples the *rubric* from the vault (the spec is hidden + reviewer-authored), addressing the single-pass eval's worst circularity.
- **Validates the fixed mechanics end-to-end** (recall fires → surfaces relevant wisdom → agent applies → faster convergence). It does **not** isolate the D-fixes' improvement *over* the pre-fix system on a facts-only vault (l2 recall worked pre-fix too); the D-fixes' distinct value lives in episodes (D6) / synthesis (D5) / graph (G0), which a facts-only seed doesn't exercise.
- Memory **can override domain judgment** (seen in the earlier single-pass run: borrowed schema over domain fields) — recall output must stay advisory.

## Bottom line (revised after antagonistic review — the original headline OVERCLAIMED)
Honest version: a **curated, task-relevant facts vault + working recall front-loaded
conventions at round 1 and plausibly saved ~2 review rounds** (n=1). That is real but
**modest, narrow, and partly circular** — and it is **NOT** validation that the Phase-8
D-fixes deliver value. See the antagonistic review below.

## Antagonistic review — finding is inflated (2026-06-05)
A ruthless review, verifying every number against `/tmp/veval` on disk, found:

1. **The dollar figures are real** ($6.56 / $9.44, reconciled to 4 decimals from `modelUsage`), but the **story over them is inflated**, and the original cost mechanism ("cold's resume escalates") was **backwards** — cold simply did ~2.7× more output work; warm's round-1 carried the higher cache-read cost.
2. **The 4-metric ledger is ~1.5 independent signals.** `human-review turns = rounds − 1` exactly (no new info); cost is downstream of round count. "Wins on 4 metrics" = **"needed 2 fewer rounds," counted three times** — and that round-count is unverified (no scorecard persisted; live judge tallies with a known 16→17 reconciliation against a 2-round margin).
3. **Rubric == vault circularity.** The 7 seed notes map 1:1 onto the 6 architecture rubric items; warm's round-1 6/6-vs-2/6 is largely "did you receive the notes." ~44% of the round-1 lead is that circular slice.
4. **Reviewer-as-vault-holder (the deepest break).** Cold's R3 feedback *was* the 6 architecture vault notes read aloud — the reviewer holds the vault and fed cold the same conventions warm got from it, just later. This is **conventions-free-at-R1 vs the-same-conventions-from-a-human-at-R3**, not memory-vs-no-memory. Cold's "convergence" is the human spec-feeding the methodology bakes in.
5. **Does not isolate D1–D6.** A facts-only l2 vault would have helped the pre-fix system identically; episodes (D6) / synthesis (D5) / graph (G0) were never exercised. The weaker question actually answered — "does a curated facts vault + working recall front-load conventions?" — was never in doubt.
6. **Seed contamination for reuse:** warm-vault now holds 11 notes (not 7) — warm's `/learn` wrote an episode of this very build. This run was clean (recall fired before those writes) but the seed is now an answer key for any re-run.

**Verdict: STOP, do not scale.** n≥5 measures a confounded quantity precisely. A valid value-test of the D-fixes requires: decouple the rubric from the seed (criteria authored blind to vault contents; score off-rubric/transfer axes), persist a scorecard per round, pristine re-seed each run, hold reviewer feedback identical across arms, and **actually exercise episodes/synthesis/graph**.

**Surviving clean signal (thin):** with an empty vault, cold produced less spec-aligned round-1 output (14.5K vs 49.7K output tokens) — working recall *does* front-load conventions at round 1. Real, ~2 independent bits, partly negative (memory overrode domain judgment — borrowed the vault's tags schema over contact-domain fields).
