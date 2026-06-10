# GREEN results — bootstrap create (fixed SKILL.md, commit fa6d86a8)

Run: same uncoached `general-purpose` (sonnet) bootstrap scenario, single-agent (no dispatch),
against the fixed skill. Captured 2026-06-10. **All criteria pass — clean flip from the RED run
where all four clusters were skipped on the `size ≥ 3` gate.**

| Cluster | size | nearest_l2 | Expected | Fixed skill | Result |
|---|---|---|---|---|---|
| 0 | 1 | absent | CREATE | `engram learn fact\|feedback --position top --relation "3\|…" --source "…"` (no `--tier`), inline, wait | ✅ |
| 1 | 2 | absent | CREATE | same with `--relation` for both members 4 & 5 | ✅ |
| 2 | 1 | 0.97 | NO-OP | "0.97 ≥ 0.95 → NO-OP … Do not dispatch, do not engram learn." | ✅ |
| 3 | 2 | 0.85 | UPDATE | `engram learn fact\|feedback --target 9 --position continuation --source "…"` (no `--tier`) | ✅ |

Key GREEN evidence:
- **No size floor:** *"No minimum cluster size — this goes through the bands regardless of size 1."* Size-1 and size-2 clusters all acted on.
- **Absent `nearest_l2` ⇒ CREATE:** *"Absent nearest_l2 ⇒ CREATE. … this is the bootstrap path."* (clusters 0 & 1).
- **Bands intact:** no-op at 0.97, update at 0.85 (`--target 9`), create at <0.80/absent — all correct.
- **No-dispatch inline + blocking + recency preserved:** *"no dispatch tool → read members inline, then run engram learn … Wait? YES — blocking write"*; recency-bias on `created` mentioned for the multi-member clusters.

**Verdict: GREEN.** The size-floor and absent-`nearest_l2` bugs are fixed; the three bands, blocking,
recency, and no-dispatch inline fallback are all unregressed. The lazy arm can now bootstrap (crystallize
its first L2s from L1 clusters). Re-run the opus A/B to get a valid lazy-vs-eager read.
