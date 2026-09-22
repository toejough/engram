Conventions: TDD (failing test, `targ test` RED, implement, GREEN, refactor); gomega + rapid;
`t.Parallel()` everywhere; no `os.*` in `internal/`; `targ check-full` before ticking section 3.

## 1. Matcher (TDD)

- [x] 1.1 RED: `internal/cli/query_triggers_test.go` (new, table of 30 cases: bare word hit/miss, 'accurate'/'curated'/'curates'/'pleased'/'displease', slash forms, start/end of text, later occurrence, unicode/digit neighbors, hyphen/underscore boundaries, whitespace collapse) and `query_trigger_property_test.go` property `HitsIffNormalizedWholeWord` with an independent regexp oracle; `targ test` failed on the 9 miss cases (evidence: "Expected [note] to be empty" for accurate/curated/curates/pleased/displease/inaccurate)
- [x] 1.2 GREEN: `containsWholeWord`/`boundaryOK`/`isWordRune` in `internal/cli/query_triggers.go`; `matchTriggers` calls it; `targ test` green
- [x] 1.3 Fix the one existing test the new rule legitimately changed: `serve_test.go` `TestServeQuery_TextCappedAtTwoKB` "cue at start of long text" had the cue glued to `xxxx` (now a right-boundary miss); a space separates them
- [x] 1.4 REFACTOR: `targ reorder-decls` applied; no lint suppressions

## 2. Real-binary smoke (scratch vault, non-repo cwd)

- [x] 2.1 `go install ./cmd/engram`; scratch vault under the session scratchpad; `engram learn runbook … --trigger curate`
- [x] 2.2 Results: `--text "please curate the offers"` -> hit (`provenances: [trigger]`); `"Curate!"` -> hit; `"/curate"` -> hit; `"that is accurate"` -> `items: []`; `"curated offers"` -> `items: []`

## 3. Quality gates

- [x] 3.1 `targ check-full`: all checks pass except check-uncommitted while uncommitted (final run after commit recorded in the report)
- [x] 3.2 `openspec validate runbook-trigger-whole-word --strict` and `openspec validate --all --strict`

## 4. Docs

- [x] 4.1 Update the match-rule docs per the Enumeration below (GLOSSARY, c3-components K6, README, adr.md ADR-0026 amendment)
- [x] 4.2 DONE (GREEN 25 runs vs control 15 runs, all scenarios improved or unchanged; SKILL.md + specs updated). Was: DEFERRED (needs paid tests): edit `agent-instructions/skills/write-memory/SKILL.md` authoring rule ("never a lone common word") to admit deliberate whole-word bare words, and update the main-spec requirement "Trigger cues SHALL be specific, author-chosen strings" (currently "a slash form or a multi-word phrase") to match. SKILL.md edits require `superpowers:writing-skills` RED->GREEN with behavioral runs; the precedent (LEDGER `write-memory-triggers-contract`) used 12 paid `claude -p` runs per arm. Not run; cost reported to Joe. Until then the SKILL.md rule stays stricter than the matcher (safe direction), and GLOSSARY `trigger` states the intended rule.
- [x] 4.3 After Joe's review: archive this change (not done here), then re-add bare-word triggers (`curate`) to runbook notes as desired — archived 2026-09-21: delta merged into `runbook-lexical-triggers` and `write-memory-worker` main specs, change moved to `openspec/changes/archive/2026-09-21-runbook-trigger-whole-word/`; re-adding bare-word triggers to runbook notes is a separate follow-up, not performed here

## Enumeration

Grep: `substring|case-insensitive|trigger hit|lone common|whitespace runs` (and `trigger`) across `docs/`, `agent-instructions/`, `openspec/specs/`, `README.md`, `dev/eval/*.md`.

| File | Disposition |
|---|---|
| `docs/GLOSSARY.md` (`trigger hit`, `trigger`) | update: whole-word rule; authoring rule admits deliberate bare words, still not very common words |
| `docs/GLOSSARY.md` (`provenances`, L352, L429) | leave: names the `trigger` role, no match rule |
| `docs/architecture/c3-components.md` K6 row | update: `matchTriggers` = whole-word |
| `docs/architecture/c3-components.md` payload lines (~L139-142) | leave: no match rule stated |
| `docs/architecture/adr.md` ADR-0026 | update: "case-insensitive substring" -> whole-word, dated amendment pointing at this change |
| `README.md` `engram query` row | update: whole-word wording with example |
| `README.md` L54 (please matched by triggers) | leave: lists triggers only |
| `agent-instructions/skills/write-memory/SKILL.md` L82-90 | update (task 4.2, done) |
| `openspec/specs/runbook-lexical-triggers/spec.md` | update via this change's MODIFIED delta (match requirement); "Trigger cues SHALL be specific" requirement deferred with 4.2 |
| `openspec/specs/recall-runbook-surfacing/spec.md` | leave: says only "trigger hit", no match rule |
| `openspec/specs/vault-merged-recall/spec.md`, `recall-payload-cuts/spec.md` | leave: trigger hits ordering/limit only |
| `openspec/specs/write-memory-worker/spec.md` L134 | leave: `--trigger` composition, no match rule |
| `dev/eval/LEDGER.md` L112/L116 | leave: dated measurements of prior behavior; not restated as current |
| `dev/eval/.../Please-R/vault/6.*please-drive*.md` fixture | leave: triggers `/please`, `please`, `take this end-to-end` unaffected except `please` no longer hits "pleased" |
