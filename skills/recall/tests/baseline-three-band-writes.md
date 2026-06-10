# Baseline scenario — three-band blocking L2 synthesis writes

A subagent is given a `/recall` task whose `engram query` has already returned a
`--synthesize-l2` payload with three clusters — one in each `nearest_l2.cosine` band. Under the
**updated** SKILL.md (Phase 3 of the lazy-L2 plan) the agent must apply the three-band rule and
**block** on the writes, then use the freshly-minted L2s. Under the **current** SKILL.md there is no
`nearest_l2` band rule at all (Step 3 reads `--tier L2`; Step 3a dispatches *fire-and-forget*
synthesis subagents gated only on cluster size + theme, with no cosine bands and no blocking), so
the baseline run should NOT produce the three-band blocking behavior.

## Scenario prompt (verbatim, give to subagent)

> You are an agent in a vault-backed memory system, in the middle of a build task: "implement a Go
> CLI's storage layer." You already ran your recall query and got back the payload below. Read
> `/Users/joe/repos/personal/engram/skills/recall/SKILL.md` and follow its instructions **exactly**
> for what to do with these clusters. **Do NOT actually run `engram learn`** — instead, for each
> cluster, print the EXACT `engram learn` invocation(s) you would issue (or state explicitly that
> you would issue none), and state whether you would wait for those writes before continuing your
> build task or fire them and move on. Then state, in one line, what you do next.
>
> Payload (`engram query --synthesize-l2` output):
>
> ```yaml
> version: 1
> phrases: ["Go CLI storage layer conventions"]
> clusters:
>   - id: 0
>     phrase: "Go CLI storage layer conventions"
>     size: 3
>     members:
>       - { path: Permanent/12a.2026-05-01.filestore-interface.md, score: 0.91, is_representative: true }
>       - { path: Permanent/30.2026-05-10.episode-store-wiring.md, score: 0.78, is_representative: false }
>       - { path: Permanent/31.2026-05-12.episode-store-tests.md, score: 0.74, is_representative: false }
>     nearest_l2: { path: Permanent/12a.2026-05-01.filestore-interface.md, cosine: 0.97 }
>   - id: 1
>     phrase: "Go CLI storage layer conventions"
>     size: 3
>     members:
>       - { path: Permanent/12.2026-04-02.storage-atomicity.md, score: 0.88, is_representative: true }
>       - { path: Permanent/40.2026-05-20.episode-atomic-rename.md, score: 0.81, is_representative: false }
>       - { path: Permanent/41.2026-06-01.episode-fsync-before-rename.md, score: 0.79, is_representative: false }
>     nearest_l2: { path: Permanent/12.2026-04-02.storage-atomicity.md, cosine: 0.86 }
>   - id: 2
>     phrase: "Go CLI storage layer conventions"
>     size: 3
>     members:
>       - { path: Permanent/50.2026-05-25.episode-concurrent-writes.md, score: 0.83, is_representative: true }
>       - { path: Permanent/51.2026-05-26.episode-file-locking.md, score: 0.80, is_representative: false }
>       - { path: Permanent/52.2026-05-27.episode-lock-timeout.md, score: 0.77, is_representative: false }
>     nearest_l2: { path: Permanent/12.2026-04-02.storage-atomicity.md, cosine: 0.42 }
> budget: { phrases_queried: 1, total_notes: 200, with_embeddings: 200, clusters_found: 3 }
> ```
>
> Member-note `created` dates (read these from the notes' frontmatter — they diverge in cluster 1):
> cluster 1 members: `12.…` created 2026-04-02 (says "fsync optional"); `40.…` created 2026-05-20;
> `41.…` created 2026-06-01 (says "fsync REQUIRED before rename" — newer, contradicts 12's stance).

## What we are measuring

A pass (GREEN, only achievable under the updated skill) requires ALL of:

1. **Cluster 0 (`nearest_l2.cosine` 0.97 ≥ 0.95) → NO-OP.** The agent issues **no** `engram learn`
   for cluster 0 (an existing L2 already covers it).
2. **Cluster 1 (0.86, in 0.80–0.95) → UPDATE.** The agent issues `engram learn fact|feedback` that
   **targets the nearest L2's Luhmann id** (`--target 12 --position continuation`), with **no
   `--tier` flag** (absence = L2).
3. **Cluster 2 (0.42 < 0.80) → CREATE.** The agent issues `engram learn fact|feedback` with
   `--position top` and `--relation` to each cluster-2 member, **no `--tier`**.
4. **Writes are BLOCKING.** The agent states it **waits** for the cluster-1/cluster-2 writes to
   finish and will **apply** the resulting L2s to its current build task — NOT "dispatch and move
   on / fire-and-forget."
5. **Recency-bias on divergence.** For cluster 1, the agent's synthesis prefers the
   **more-recently-created** member (`41.…`, 2026-06-01, "fsync REQUIRED") over the older,
   conflicting `12.…` (2026-04-02, "fsync optional").

## Failure modes to watch for (expected in the RED baseline)

- Agent runs `engram query --tier L2` (the current Step 3) and never consumes `nearest_l2` bands.
- Agent applies no cosine bands — treats all 3 clusters the same (or only the size/theme gate).
- Agent fires synthesis subagents **fire-and-forget** and continues without waiting.
- Agent writes an L2 for cluster 0 (no no-op band) or adds `--tier` to the writes.
- Agent ignores `created` dates / does not prefer the newer member on cluster-1 divergence.

## Capture format

- RED (subagent run against the CURRENT, unedited SKILL.md): `baseline-three-band-RED-results.md`
- GREEN (subagent run against the EDITED SKILL.md): `baseline-three-band-GREEN-results.md`
