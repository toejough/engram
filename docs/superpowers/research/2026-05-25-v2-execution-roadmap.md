# v2 execution roadmap

Date: 2026-05-25. Execution-status companion to the design
artifacts in this directory.

## Quick orientation for a new agent

Engram is a memory CLI for LLM agents (Go binary + a few skills).
v2 is the embeddings-driven retrieval rewrite spec'd in the
2026-05-22-…research-log + the 2026-05-24-…spike spec. The spike
landed; this roadmap tracks what remains.

**Start here:**
- Design trace (why each choice): `2026-05-22-tiered-memory-research-log.md`
  → see the **Current focus list** at the top (F1–F9, all
  resolved) and the per-item resolution sections.
- Spike spec (the foundation, now executed):
  `2026-05-24-engram-query-spike.md`
- MOC migration procedure (executed):
  `2026-05-24-moc-migration-procedure.md`
- Project conventions: root `CLAUDE.md` + `.claude/rules/go.md`.
  Build via `targ` only (`targ test`, `targ check-full`,
  `targ build`).
- Worktree rules: review before merge; ff-only merges; rebase on
  main before merge.

## Where we are (as of 2026-05-25)

| Phase | Commits | Status |
|---|---|---|
| Spike (UAT 13: Arctic-xs fell back to MiniLM-L6 per spec; pipeline shipped) | 12 commits (now squashed into main via ff-merge) ending at `e311c0be` | ✅ Done |
| Post-spike review fix (all-incompatible sidecar guard) | `a2e95f1b` | ✅ Done |
| Spec refinement: strip wikilinks from `items.content` | `e311c0be` | ✅ Done |
| F4 — MOC migration (per-MOC: 25 MOCs → 39 facts/feedback) | Multiple `engram learn` writes to `~/.local/share/engram/vault/` | ✅ Done |
| F4 cleanup (167 inbound wikilink rewrites; 74 cross-MOC related-to bullets; orphan sidecars) | Vault edits only; no repo commits | ✅ Done |
| Drop `engram learn moc` | `d34b24d5` | ✅ Done |
| Vault bootstrap + L1 diagram MOC cleanup | `da469c36` | ✅ Done |

Repo state: main at `da469c36`, 22 commits ahead of origin/main,
working tree clean. Vault state: 480 notes total, all embedded,
0 missing/stale/incompatible. `MOCs/` directory empty;
`_legacy/MOCs/` holds 25 original MOC `.md` + `.vec.json` pairs.

## What's next, in order

| # | Item | Notes |
|---|---|---|
| **1** | **F1 — episode kind** | Add `engram learn episode` + episode schema + SKILL.md update. See "Immediate next step" below. |
| 2 | F6 + F9.1 — subgraph clustering at query time | Add 3-hop link expansion + auto-k-means clustering + hub identification to `engram query`. Expand payload per F7's resolved shape (clusters section + richer items.provenances). |
| 3 | Updated `/recall` SKILL.md | Replace cascade logic with `engram query` invocation; add the synthesis-gate per-cluster discipline (write fact/feedback via `/learn` when a cluster has a binding principle). |

Items 1–3 are the only remaining v2 work. F4 is the largest item by token-cost; it's done. F1 is medium; F6+F9.1 is medium-large; the SKILL update is small-medium.

## Outstanding loose ends (low-priority cleanup)

These were flagged during F4 work but deliberately deferred:

- **Read-side MOC infrastructure** in `internal/vaultgraph/` (`IsMOC` field on `Note`, `pathOf` MOC branch, `ListIDs` scanning `MOCs/`, scanner reading `MOCs/`). Untouched — much larger change. Defer unless a concrete reason to retire.
- **Historical doc references** to `engram learn moc` in `docs/superpowers/research/*` and `docs/plans/*`. Kept as historical context.
- **Merged worktree branch** `worktree-engram-query-spike` still exists in `git branch` output. Worktree directory removed. Can be deleted with `git branch -d worktree-engram-query-spike` if you want a tidy branch list.

## Immediate next step: F1 — episode kind

**Plan (user-ratified 2026-05-25):** (b) sharpen the spec first,
then (a) dispatch to a subagent for implementation. Same pattern
as the `drop engram learn moc` dispatch that worked well.

### Why sharpen first

F1's sub-decisions were deferred to the v2 spec but never
written out concretely. The implementation needs:

- Storage location: **lean = same `Permanent/` dir with
  `kind: episode` in frontmatter.** Alternative: separate
  `Episodes/` dir.
- Schema fields: **lean = situation, summary, outcomes,
  provenance (sessions + transcript range), related.** Each
  field's semantics needs a one-line definition (especially
  `outcomes` and `provenance`).
- Rendered body shape: facts auto-generate `Information learned:
  …`. What does an episode's body render look like? Probably:
  a date line, the situation phrase, then the summary as a
  paragraph or paragraphs, with outcomes as a bulleted list and
  provenance in frontmatter only.
- Discipline (different from facts/feedback): episodes are
  narrative; project names OK; dates OK; first-person "I did X"
  framing OK. Need a discipline doc / SKILL.md section.
- `engram learn episode` flag set: `--slug`, `--source`,
  `--situation`, `--summary`, `--outcome` (repeatable),
  `--session` (repeatable), `--transcript-range`,
  `--relation` (repeatable, same shape as facts/feedback).
- Wikilinking: episodes link to facts/feedback they spawned via
  `--relation`. Optional: facts/feedback can `--relation` back
  to the originating episode.

### What the sharpened spec should produce

A short artifact (~50-100 lines) at
`docs/superpowers/research/2026-05-25-episode-kind-spec.md`
containing:

1. Storage decision (Permanent vs Episodes — recommend
   Permanent with `kind: episode`).
2. Schema with per-field semantics + an example rendered
   episode `.md` file.
3. Discipline doc — what's allowed (narrative, project names,
   dates) and what's forbidden (analysis dressed as narrative,
   speculation as fact). One paragraph.
4. CLI surface — flag list with descriptions.
5. SKILL.md additions — section title + brief structure (the
   actual prose goes in via `superpowers:writing-skills`).
6. Test plan — what unit tests + integration tests are needed.
7. Out of scope (deferred to later iterations).

### Then dispatch

Subagent prompt structure (template from the drop-learn-moc
dispatch, with episode-specific details substituted):

- Context: episode kind is being added per F1 in the v2 work order
- Pointer to the sharpened spec doc
- Project discipline (targ, DI, test patterns, AI-Used trailer)
- SKILL.md edit goes through `superpowers:writing-skills`
- Output: commit SHA, files changed, judgment calls flagged

## Reference docs (load order for a new agent)

1. This roadmap
2. `2026-05-22-tiered-memory-research-log.md` — design rationale
   if you need to know *why* something
3. The relevant spec for the work you're doing:
   - F1: `2026-05-25-episode-kind-spec.md` (to be written)
   - F6+F9.1: see the F6/F9.1 resolution in the research log
     (no separate spec yet — write one next)
   - `/recall` SKILL.md update: see F6+F9.1 resolution's
     "Synthesis expectations for the consuming skill" section
4. `CLAUDE.md` + `.claude/rules/go.md`
5. `docs/architecture/c1-system-context.md` if you need the
   system context

## How to verify you're on the right track

- After any code change: `targ check-full` must pass (8 checks)
- After any vault change: `engram embed status` should show
  total = with-embeddings, stale = 0, incompatible = 0, broken = 0
- After any SKILL.md edit: re-test the skill's invocation per
  `superpowers:writing-skills` discipline
- Worktree: only use one if multiple agents need conflict-free
  edits in parallel; for single-agent feature work, just work on
  main with normal commits
