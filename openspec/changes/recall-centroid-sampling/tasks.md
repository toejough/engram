# Tasks: recall-centroid-sampling

## 1. Sampling core (pure logic, TDD)

- [ ] 1.1 TDD: softmax allocation over query→centroid cosine with temperature constant; totals equal explore budget; property tests (rapid) for monotonicity and budget conservation
- [ ] 1.2 TDD: match-evidence bonus — bounded additive weight for clusters with members in the exploit half
- [ ] 1.3 TDD: centroid-proximal within-cluster selection (top-1 members, definition notes excluded, descending centroid similarity)
- [ ] 1.4 TDD: dedupe against exploit half + cross-cluster, with allocation-order backfill until budget met or candidates exhausted

## 2. Query-path integration

- [ ] 2.1 TDD: compose payload halves (exploit unchanged: floors/caps/local clustering); explore half from sampling core; missing/unreadable centroids → exploit-only + `explore_allocated: {}`
- [ ] 2.2 Remove `buildTagNominations` and `tag_nominations_added/dropped` budget fields; add `provenance: explore` + source term per note and `explore_allocated` to budget
- [ ] 2.3 Wire centroid loading through injected deps (reuse existing `vocab.centroids.json` reader; no new direct I/O)

## 3. Calibration + validation

- [ ] 3.1 Calibrate τ and the match-evidence bonus on the real vault; record method + values in dev/eval/LEDGER.md
- [ ] 3.2 Before/after payload comparison on the production vault (same query set through old nomination path and new sampling path); C3–C6-style trap check that previously-nominated load-bearing notes remain reachable
- [ ] 3.3 Verify with the installed binary + real query args from a non-vault cwd; inspect YAML output for provenance and budget fields

## 4. Skill + docs

- [ ] 4.1 Update `agent-instructions/skills/recall/SKILL.md` nomination references (lines 20, 110–111, 149–151 name tag nomination and the `tag_nominations_added/dropped` fields verbatim — a certain edit, not an audit; via `superpowers:writing-skills`)
- [ ] 4.2 Update GLOSSARY.md / architecture docs mentioning tag nomination per the disposition list below
- [ ] 4.3 Add missing spec deltas discovered by enumeration grep: `recall-two-channel-payload` (Channel 1 requirement + "Cross-cluster tag nomination" scenario name explore sampling instead) and `route-dispatch-evidence` (evidence aggregates surface "through tag nomination" → via vocab-tagged explore sampling)

### Doc-surface disposition list (enumeration grep: `tag.nomination|tag_nomination|nominat|explore_allocated`)

| File | Disposition | Reason |
| --- | --- | --- |
| `agent-instructions/skills/recall/SKILL.md` | update | names tag nomination + budget fields verbatim (4.1) |
| `agent-instructions/skills/recall/tests/baseline-bootstrap-create.md` | update | "nominates existing L2s via candidate_l2s" — reword to candidate surfacing |
| `agent-instructions/skills/route/SKILL.md` | N/A | "denominators" — false positive |
| `docs/architecture/adr.md` | update | ADR-0011 gets a supersession status note (historical body stays); new ADR entry for centroid sampling |
| `docs/architecture/c1-system-context.md` | update | query-flow prose names nomination + budget fields |
| `docs/architecture/c2-containers.md` | update | C2 row + query flow diagram name nomination, cap, `query_nominations.go` |
| `docs/architecture/c3-components.md` | update | K6 row, payload shape, sequence diagrams name nomination |
| `docs/architecture/memory-invariants.md` | keep | RETIRED L3-1 "nomination" = within-cluster candidate mechanism, unchanged |
| `docs/architecture/memory-system-rigor.md` | keep | same retired reference |
| `docs/GLOSSARY.md` | update | "vocab nomination" entry rewritten to explore sampling; def-note exclusion line updated |
| `docs/ROADMAP.md` | update | revive-trigger rows naming live nomination reworded; settled ADR-0011 row kept as history with supersession note |
| `README.md` | update | recall row + `engram query` usage text name nomination + budget fields |
| `openspec/specs/recall-query-timings/spec.md` | update | `nominate` stage description reworded to explore-sampling stage (stage key kept) |
| `openspec/specs/recall-two-channel-payload/spec.md` | update (via delta, 4.3) | Channel 1 requirement + scenario name tag nomination |
| `openspec/specs/route-dispatch-evidence/spec.md` | update (via delta, 4.3) | names tag nomination as the surfacing mechanism |
| `dev/eval/LEDGER.md` | keep | historical measured results; task 3.1/3.2 append new entries |
| `dev/eval/atoms*` sandbox/candidate texts | N/A | frozen eval fixtures — historical by design |
| `docs/superpowers/plans/*` | keep | historical plan documents |

### Execution requirements (from recall, 2026-07-28)

- Route go-tdd-feature executors at **mid (sonnet)**; verify artifacts after every executor report (grep tree + run gates) — note 600.
- `t.Parallel()` parent + subtests; declaration order; property tests mandatory for `internal/` (notes 300, 556).
- Contract change: every consumer of the removed budget fields swept (note 476) — the disposition list above is that sweep for docs.
- Before/after comparison (3.2): "before" arm runs a pre-treatment worktree binary, never the post-install tree (note 259).
- τ calibration (3.1): derive from the real vault's observed similarity spread (note 45).

## 5. Close-out

- [ ] 5.1 `targ test` + `targ check-full` green
- [ ] 5.2 `openspec validate` both delta specs; sync on archive
