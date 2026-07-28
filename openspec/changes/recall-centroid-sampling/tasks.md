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

- [ ] 4.1 Audit `agent-instructions/skills/recall/SKILL.md` for nomination-field references; update if needed (via `superpowers:writing-skills`)
- [ ] 4.2 Update GLOSSARY.md / architecture docs mentioning tag nomination; scrub stale references

## 5. Close-out

- [ ] 5.1 `targ test` + `targ check-full` green
- [ ] 5.2 `openspec validate` both delta specs; sync on archive
