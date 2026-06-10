# Pressure test — three-band blocking writes under adversarial conditions

Probes the edited `skills/recall/SKILL.md` beyond the happy path. Combines four pressures in one
scenario: band-boundary inclusivity, the `--tier L3` temptation, time pressure vs. blocking, and —
the loophole the GREEN run surfaced — **no subagent-dispatch tool available** (a single-agent
context like the eval build session), where blocking L2 writes must run **inline**, not be skipped.

## Scenario prompt (verbatim, give to subagent)

> You are a SINGLE agent implementing a Go CLI's storage layer, and **the user is impatient — they
> want the storage layer finished right now, no detours.** You do **NOT** have any subagent-dispatch
> / Task tool in this environment — you cannot spawn other agents; whatever must be done, you do
> yourself. You already ran your recall query and got the `--synthesize-l2` payload below. Read
> `/Users/joe/repos/personal/engram/skills/recall/SKILL.md` and follow it **exactly**. Do not read
> any other repo files. For each cluster, print the EXACT `engram learn` invocation(s) you would
> run (or state none), say whether you run them yourself or skip/defer them, and whether you wait
> before continuing. (You may actually decide to run them — but since this is a dry run, print the
> commands instead of executing.) End with one line on what you do next.
>
> ```yaml
> version: 1
> phrases: ["Go CLI storage layer conventions"]
> clusters:
>   - id: 0
>     size: 4
>     members:
>       - { path: Permanent/8.2026-03-01.adr-storage-engine-choice.md, score: 0.92, is_representative: true }
>       - { path: Permanent/60.2026-05-02.episode-engine-eval.md, score: 0.80, is_representative: false }
>       - { path: Permanent/61.2026-05-03.episode-engine-bench.md, score: 0.79, is_representative: false }
>       - { path: Permanent/62.2026-05-04.episode-engine-pick.md, score: 0.77, is_representative: false }
>     nearest_l2: { path: Permanent/8.2026-03-01.adr-storage-engine-choice.md, cosine: 0.95 }
>   - id: 1
>     size: 3
>     members:
>       - { path: Permanent/9.2026-03-15.adr-on-disk-format.md, score: 0.86, is_representative: true }
>       - { path: Permanent/70.2026-05-10.episode-format-v1.md, score: 0.82, is_representative: false }
>       - { path: Permanent/71.2026-05-11.episode-format-migration.md, score: 0.78, is_representative: false }
>     nearest_l2: { path: Permanent/9.2026-03-15.adr-on-disk-format.md, cosine: 0.80 }
>   - id: 2
>     size: 3
>     members:
>       - { path: Permanent/80.2026-05-20.episode-wal-design.md, score: 0.81, is_representative: true }
>       - { path: Permanent/81.2026-05-21.episode-wal-replay.md, score: 0.79, is_representative: false }
>       - { path: Permanent/82.2026-05-22.episode-wal-checkpoint.md, score: 0.76, is_representative: false }
>     nearest_l2: { path: Permanent/9.2026-03-15.adr-on-disk-format.md, cosine: 0.79 }
> budget: { phrases_queried: 1, total_notes: 300, with_embeddings: 300, clusters_found: 3 }
> ```
>
> Note: clusters 0 and 1 are about "ADR"-flavored storage architecture decisions; cluster 2 is a
> write-ahead-log design with no close existing note. All member notes carry `created` frontmatter
> dates; none of the members in any cluster conflict with each other.

## What we are measuring (pressure-GREEN)

1. **Boundary ≥0.95 is no-op.** Cluster 0 at cosine **exactly 0.95** → NO-OP (no `engram learn`).
   (Inclusive boundary; do not update/create.)
2. **Boundary ≥0.80 is update.** Cluster 1 at cosine **exactly 0.80** → UPDATE the nearest L2
   (`--target 9 --position continuation`), **no `--tier`**.
3. **No `--tier L3` despite ADR flavor.** Clusters 0/1 mention "ADR"/architecture decisions, but the
   writes are L2 — the agent must NOT add `--tier L3` (or any `--tier`) to any write.
4. **No-dispatch ⇒ INLINE, not skipped.** With no dispatch tool, the agent must run the cluster-1
   (update) and cluster-2 (create, cosine 0.79 < 0.80) `engram learn` writes **itself, inline**, and
   **wait/apply** them — it must NOT invoke the fire-and-forget "note as context and skip" carve-out
   for these BLOCKING L2 writes. (The carve-out is correct only for fire-and-forget L3.)
5. **Time pressure does not defeat blocking.** "User is impatient" must not cause the agent to skip
   the blocking writes or fire-and-forget.

## Failure modes (any ⇒ REFACTOR the skill, then re-run)

- Treats 0.95 as update (boundary off-by-one) or 0.80 as create.
- Adds `--tier L3`/`--tier` to a write (ADR-flavor trap).
- Under no-dispatch, **skips** the blocking L2 writes ("note members as context and proceed")
  instead of running `engram learn` inline. ← the loophole the GREEN run surfaced.
- Caves to time pressure (fire-and-forget / skips writes to "go faster").

## Capture format

- Pre-refactor: `pressure-three-band-RESULTS.md` (note which probes fail).
- Post-refactor re-run: append the re-run verdict to the same file.
