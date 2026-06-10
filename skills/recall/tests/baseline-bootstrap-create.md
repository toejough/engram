# Baseline scenario — bootstrap: small clusters & absent nearest_l2 must CREATE

The lazy arm starts every app with a vault that has **zero L2s** (learn wrote L1 episodes only).
A `--synthesize-l2` recall over such a vault returns **small clusters (size 1–2)** with **no
`nearest_l2` field** (there is no existing L2 to point at). Per spec §2/§5 ("no member-exclusion and
**no minimum cluster size**"; demand is the relevance proof), these must still be banded — and an
**absent `nearest_l2` means there is definitionally no covering L2 → CREATE**.

The live opus A/B run (2026-06-10) showed the current skill **skips** these (it inherited a bogus
`size ≥ 3` precondition from the old L3 fire-and-forget gate, and has no rule for absent
`nearest_l2`), so arm B crystallized nothing and degenerated to L1-only memory. This scenario is the
failing baseline that authorizes the fix.

## Scenario prompt (verbatim, give to subagent)

> You are a SINGLE agent (no subagent-dispatch / Task tool — do everything yourself) building a Go
> CLI's storage layer. You already ran your recall query and got the `--synthesize-l2` payload
> below. Read `/Users/joe/repos/personal/engram/skills/recall/SKILL.md` and follow it **exactly**. Do
> not read other repo files. For each cluster, print the EXACT `engram learn` invocation(s) you would
> run (or state none) and whether you wait. (Dry run — print commands, don't execute.) End with one
> line on what you do next.
>
> ```yaml
> version: 1
> phrases: ["Go CLI storage layer conventions"]
> clusters:
>   - id: 0
>     size: 1
>     members:
>       - { path: Permanent/3.2026-06-01.episode-storage-build.md, score: 0.88, is_representative: true }
>     # no nearest_l2 — the vault has zero L2 notes
>   - id: 1
>     size: 2
>     members:
>       - { path: Permanent/4.2026-06-02.episode-atomic-writes.md, score: 0.85, is_representative: true }
>       - { path: Permanent/5.2026-06-03.episode-fsync.md, score: 0.80, is_representative: false }
>     # no nearest_l2 — still zero L2 notes
>   - id: 2
>     size: 1
>     members:
>       - { path: Permanent/12a.2026-05-01.filestore-interface.md, score: 0.90, is_representative: true }
>     nearest_l2: { path: Permanent/12a.2026-05-01.filestore-interface.md, cosine: 0.97 }
>   - id: 3
>     size: 2
>     members:
>       - { path: Permanent/9.2026-03-15.storage-format.md, score: 0.86, is_representative: true }
>       - { path: Permanent/40.2026-05-20.episode-format-v1.md, score: 0.81, is_representative: false }
>     nearest_l2: { path: Permanent/9.2026-03-15.storage-format.md, cosine: 0.85 }
> budget: { phrases_queried: 1, total_notes: 5, with_embeddings: 5, clusters_found: 4 }
> ```
>
> All member notes carry `created` frontmatter dates; none conflict.

## What we are measuring (GREEN — only achievable after the fix)

1. **Cluster 0 (size 1, NO `nearest_l2`) → CREATE.** A new L2 via `engram learn fact|feedback
   --position top --relation "<member>|…"`, **no `--tier`**. NOT skipped for size, NOT skipped for
   missing `nearest_l2`.
2. **Cluster 1 (size 2, NO `nearest_l2`) → CREATE.** Same — no size floor, absent band ⇒ create.
3. **Cluster 2 (size 1, `nearest_l2` 0.97 ≥ 0.95) → NO-OP.** A tiny cluster already covered by an
   existing L2 is still a no-op.
4. **Cluster 3 (size 2, `nearest_l2` 0.85) → UPDATE.** `--target 9 --position continuation`, no
   `--tier` — update works at small size too.
5. **No `size ≥ 3` precondition** anywhere; **absent `nearest_l2` is treated as CREATE**, not skip.
6. Single-agent (no dispatch) ⇒ writes run **inline** (per the established no-dispatch fallback), not
   skipped.

## Failure modes (the RED baseline)

- Skips clusters 0/1 because size < 3, or because they have no `nearest_l2` ("nothing to band on").
- Treats absent `nearest_l2` as "no-op / nothing to do".
- Only acts on clusters with `nearest_l2` present and/or size ≥ 3.

## Capture format

- RED (current skill): `baseline-bootstrap-create-RED-results.md`
- GREEN (fixed skill): `baseline-bootstrap-create-GREEN-results.md`
