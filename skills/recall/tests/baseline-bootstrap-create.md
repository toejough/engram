# Baseline scenario — bootstrap: empty candidate_l2s must produce CREATE

The agent-judged model (Step 2.5) nominates existing L2s via `candidate_l2s: [{path, cosine}]`
per cluster. When a vault has **zero L2 notes**, `candidate_l2s` is empty (or absent) for every
cluster. Per the SKILL.md "Absent" criterion: **no candidate addresses the situation → CREATE**.
An empty `candidate_l2s` is definitionally the absent case.

This scenario tests that the agent:
1. Does not skip clusters because `candidate_l2s` is empty — empty IS the absent outcome.
2. Issues `engram learn fact|feedback --position top` for each such cluster.
3. Continues to apply the no-op rule correctly when `candidate_l2s` does contain a covered candidate.

## Scenario prompt (verbatim, give to subagent)

> You are a SINGLE agent (no subagent-dispatch / Task tool — do everything yourself) building a
> Go CLI's storage layer. You already ran your recall query and got the recall query payload
> below. Read `/Users/joe/repos/personal/engram/skills/recall/SKILL.md` and follow it **exactly**.
> Do not read other repo files. For each cluster, print the EXACT `engram` invocation(s) you
> would run (or state none) and whether you wait. (Dry run — print commands, don't execute.)
> End with one line on what you do next.
>
> ```yaml
> version: 1
> phrases: ["Go CLI storage layer conventions"]
> clusters:
>   - id: 0
>     size: 1
>     members:
>       - { path: Permanent/3.2026-06-01.storage-build-notes.md, score: 0.88, kind: chunk,
>           is_representative: true }
>     candidate_l2s: []
>   - id: 1
>     size: 2
>     members:
>       - { path: Permanent/4.2026-06-02.atomic-writes-notes.md, score: 0.85, kind: chunk,
>           is_representative: true }
>       - { path: Permanent/5.2026-06-03.fsync-notes.md, score: 0.80, kind: chunk,
>           is_representative: false }
>     candidate_l2s: []
>   - id: 2
>     size: 1
>     members:
>       - { path: Permanent/6.2026-05-01.filestore-interface.md, score: 0.90, kind: fact,
>           is_representative: true,
>           content: "filestore interface: use an injected FS interface, never os.Open directly" }
>     candidate_l2s:
>       - { path: Permanent/6.2026-05-01.filestore-interface.md, cosine: 0.97 }
>   - id: 3
>     size: 2
>     members:
>       - { path: Permanent/7.2026-03-15.storage-format.md, score: 0.86, kind: fact,
>           is_representative: true,
>           content: "on-disk format: use newline-delimited JSON for append efficiency" }
>       - { path: Permanent/8.2026-05-20.format-migration-notes.md, score: 0.81, kind: chunk,
>           is_representative: false }
>     candidate_l2s:
>       - { path: Permanent/7.2026-03-15.storage-format.md, cosine: 0.85 }
>       - { path: Permanent/9.2026-04-01.storage-overview.md, cosine: 0.71 }
>       - { path: Permanent/10.2026-02-10.storage-adr.md, cosine: 0.58 }
> budget: { phrases_queried: 1, total_notes: 10, with_embeddings: 10, clusters_found: 4 }
> ```
>
> All member notes carry `created` frontmatter dates; none conflict within any cluster.

## What we are measuring (GREEN — only achievable with the agent-judged model)

1. **Cluster 0 (empty `candidate_l2s`) → CREATE.** `engram learn fact|feedback --position top
   --relation "3|…" --source "…"`, no `--tier`. NOT skipped for small size or empty candidates.
2. **Cluster 1 (empty `candidate_l2s`) → CREATE.** Same — `--relation` for both members.
3. **Cluster 2 (candidate at cosine 0.97, content read, fully covers the principle) → COVERED.**
   The agent reads the candidate content from the `items[]` field (already in payload), judges it
   covers the cluster's principle with no material omission, and issues
   `engram amend --target <candidate> --activate` (link-enrich only). No rewrite.
4. **Cluster 3 (three candidates) → agent reads all three via `engram show`, judges coverage.**
   The top candidate at 0.85 covers the storage-format principle; agent issues
   `engram amend --target <candidate-at-0.85> --activate --chunk-source ...` (covered, link-enrich).
5. **No `size >= 3` precondition.** All clusters processed regardless of size.
6. **Empty `candidate_l2s` → absent outcome, not skip.** Explicitly stated.
7. **Single-agent (no dispatch) → writes run inline**, wait before continuing.

## Failure modes (the RED baseline)

- Skips clusters 0/1 because `candidate_l2s` is empty ("no L2 to band against").
- Issues no writes on empty-candidate clusters ("nothing to update or create").
- Skips cluster 2's coverage read because the note is already in the payload.
- Issues more than one write per cluster.

## Capture format

- RED (agent that skips on empty candidates): `baseline-bootstrap-create-RED-results.md`
- GREEN (agent-judged model): `baseline-bootstrap-create-GREEN-results.md`
