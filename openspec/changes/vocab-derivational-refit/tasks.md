# Tasks: vocab-derivational-refit

## 1. Derivation core

- [x] 1.1 TDD: derivation function clustering non-definition note vectors via `internal/cluster` silhouette-auto-K — AutoK returns flat `Assignments []int` over bare vectors, so this function owns the note↔vector correlation and groups members per cluster index (pure logic, DI'd inputs)
- [x] 1.2 TDD: greedy centroid-cosine name matching against existing terms (threshold constant; unmatched-cluster and unmatched-term outputs)
- [x] 1.3 TDD: silhouette hysteresis (prefer previous K within epsilon) and min-K floor decision from design open questions

## 2. Apply path

- [x] 2.1 TDD: retirement flow — supersede unmatched derived terms' definition notes, strip their tags in the re-tag pass; `origin: proposed` terms exempt
- [x] 2.2 TDD: naming-request emission for unmatched clusters (structured output with centroid-nearest exemplar notes) and mint-on-answer path reusing `mintDefinitionNote`
- [x] 2.3 TDD: `vocab.centroids.json` gains `origin` per term + derivation metadata; atomic write via existing locked write path; readable by old binary (additive fields)
- [x] 2.4 TDD: stamp `origin: proposed` in `RunVocabPropose`'s centroid write path (provenance is NEW plumbing — nothing distinguishes proposed from minted terms today), and default existing/bootstrap terms to `origin: derived` on first derivation
- [x] 2.5 Rewire `RunVocabRefit` to the derivation flow; remove `--emit-request`/`--plan` and plan types; add `--dry-run` diff output

## 3. Triggers

- [x] 3.1 TDD: `evaluateVocabTriggers` growth-only for `refit_pending`; untagged/hub demoted to diagnostics in `vocab stats` output (no REFIT_PENDING verdict); verify `refitUntaggedRateMax`/`hubThreshold` constants stay referenced by the diagnostics path in the same pass (unused-const lint)

## 4. Skill + docs

- [ ] 4.1 Update `agent-instructions/skills/learn/SKILL.md` Step 1.5 to the run-refit-and-answer-naming flow (via `superpowers:writing-skills` RED→GREEN, pressure tests)
- [ ] 4.2 Update GLOSSARY.md, ADR (new decision entry), and c1 diagrams referencing the plan-based refit; scrub stale references

## 5. Verification

- [ ] 5.1 `targ test` + `targ check-full` green
- [ ] 5.2 Run `engram vocab refit --dry-run` against the real 574-note vault with the installed binary from a non-vault cwd; inspect matched/new/retired diff and K for sanity
- [ ] 5.3 Update `openspec/specs/vault-vocab-lifecycle/spec.md` sync via delta; validate change with `openspec validate`
