# Engram Deep Clean — Dead Code, Refactor, Docs, Memory

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or executing-plans. Code steps follow RED→GREEN→REFACTOR. `targ check-full` MUST be green before each commit. Skill edits (if any) MUST use superpowers:writing-skills.

**Goal:** Remove dead/unreachable code, collapse the binary's query surface to the single path that actually ships (`--synthesize-l2`), refactor what remains for clarity/conciseness/consistency, scrub stale documentation and add docs for the modern (recall-v2) functionality, and fix stale memory — so nothing describes functionality the system no longer has.

**Origin:** A C6 design miss traced to stale docs/memory describing a 3-hop wikilink subgraph as the recall mechanism. Three audits (dead-code reachability, doc staleness, memory staleness) produced the targets below. **User product decisions:** (1) CUT the default `engram query` subgraph/hubs/`--tier`/`--synthesis` path — `--synthesize-l2` becomes the sole query behavior; (2) DELETE the never-wired OpenCode transcript backend and correct the docs.

**Tech Stack:** Go (`internal/`, `cmd/`), `targ` toolchain, Python eval harness (`dev/eval/`), markdown docs, auto-memory store.

## Global Constraints

- `targ check-full` green before every commit (`deadcode`, `lint-full`, `reorder-decls-check`, `check-nils-for-fail`, coverage, `check-uncommitted` all pass).
- **Never break the shipping `/recall` contract:** `skills/recall/SKILL.md` invokes `engram query --synthesize-l2 --phrase ...`. That invocation MUST keep producing the same two-channel + `candidate_l2s` payload. Verify before/after with a real run.
- **Preserve `engram check`, `engram amend`:** they use `vaultgraph.ScanVault`/`UnresolvedTargets`/`Note`. Do not remove vaultgraph symbols those depend on.
- Deletions are driven by the compiler + `targ deadcode` + tests: after each removal, anything left unreferenced is also removed; nothing referenced is touched.
- Name-agnostic, DI-everywhere, `make([]T,0,n)`, wrapped errors, `t.Parallel()`, lines <120 — the repo's standing rules.
- **Verify, don't guess:** read the symbol and its callers before deleting; if a "dead" symbol turns out to have a live caller, stop and surface it.
- **Every commit ends with the trailer `AI-Used: [claude]`** (NOT Co-Authored-By). Commit-message examples below omit it for brevity; the executor MUST append it.
- **Phase ordering is binding:** A.1 → A.2 → A.3 → A.4 → B.1 → B.2 → C → D → E → F. A.4 deletes `luhmann.Less`, whose only live callers (`vaultgraph/selector.go:78`, `vaultgraph.go:68` `basenameLess`) are removed in A.1 — running A.4 first will not compile.
- **Phase D is the actual fix**, not an optional epilogue: the C6 miss was caused by stale docs/memory, so Phases C–D are co-equal with the deletions, not lower priority.

---

## Phase A — Delete confirmed-dead, test-only code (no product change)

Each task: delete the symbol(s)/file(s) AND their now-orphaned tests; run `targ deadcode` + `targ check-full`; commit. These have **zero production callers** (audit-confirmed).

### Task A.1: vaultgraph dead files + symbols
**Delete:** `internal/vaultgraph/{components,follow,recent,selector}.go` (whole files) + their tests; `StartingPoints` and `basenameLess` (`vaultgraph.go:23,58`).
**Keep:** `ScanVault`, `Graph`, `Note`, `UnresolvedTargets`, `UnresolvedLink`, `LetterLess` (used by query-load / amend / check / luhmann).
- [ ] Grep each deleted symbol repo-wide → confirm only test/self references. Delete files + tests. `targ deadcode` + `targ check-full` green. Commit `refactor(vaultgraph): remove unused subgraph traversal files`.

### Task A.2: cluster.BestMatch
**Delete:** `internal/cluster/match.go` `BestMatch` (+ test). c3 docs already note this path was removed.
- [ ] Confirm no production caller; delete; check-full; commit `refactor(cluster): drop unused BestMatch`.

### Task A.3: transcript range + OpenCode backends (Decision 2: delete)
**Delete:** `internal/transcript/range.go` (whole: `RangeReader`/`CompositeRangeReader`/`JSONLRangeReader`/`extractTimestamp`) and `internal/transcript/opencode.go` (whole — ADR-0007 composite/opencode backend, never wired; production `ingest.go` uses only `transcript.NewJSONLReader`) + their tests.
**Keep:** `NewJSONLReader`, `JSONLReader`, `ReadResult`, `ReadFrom` (production ingest path).
- [ ] Confirm `ingest.go`/`cli` never construct the composite/opencode/range readers. Delete; check-full; commit `refactor(transcript): remove unwired range/opencode backends`.

### Task A.4: dead session-finder + luhmann sorters
**Delete:** `transcript.SessionFinder`/`NewSessionFinder` + `osDirLister`/`ListJSONL` (`cli/cli.go:27-63`); `luhmann.SortIDs` (`luhmann.go:121`), `luhmann.Less` (`luhmann.go:49`) + tests.
**Keep:** `luhmann.LetterLess` (used at `cli/luhmann.go:75`).
- [ ] Confirm callers absent; delete; check-full; commit `refactor(cli,luhmann): remove unused session-finder and sort helpers`.

**Phase A exit:** `targ deadcode` clean of these; `targ check-full` green; `engram query --synthesize-l2` still produces an identical payload (spot-check on a seeded vault).

---

## Phase B — Cut the default `engram query` path (Decision 1: cut)

Collapse three query modes to one. `--synthesize-l2` behavior becomes the sole query path; bare `engram query` routes to the same synthesis behavior. Remove `--synthesis` and `--tier` and the entire per-phrase subgraph/hub machinery.

### Task B.1: Make synthesis the only query behavior
**Files:** `internal/cli/query.go` (the big one), `internal/cli/targets.go` (flag desc), `skills/recall/SKILL.md` (only if its invocation would break — it should NOT; `--synthesize-l2` stays accepted).

**Remove (the default per-phrase path + flags):** `runSinglePhraseQuery`, `aggregatePhraseSummaries`, `expandSubgraph`/`clusterSubgraph`/`addMatchedChunksToSubgraph`, `identifyHubs`/`hubReport`, `expandedSubgraph`/`subgraphMember`, the `subgraph*` consts (`subgraphCap`, `subgraphMaxHops`, `minSubgraphForClustering`, etc.), `runSynthesisQuery` + the `Synthesis` (`--synthesis`) flag, and `--tier`/`Tiers`/`applyTierFilter`/`gatherTierIndex`/`tierL2`. Remove the `dispatchSynthesisMode` branch indirection so `RunQuery` always runs the synthesize-l2 path. Drop now-empty `hubs`/`subgraph_size` payload fields from the YAML schema.
**Keep working:** `engram query --synthesize-l2 --phrase ...` → unchanged two-channel + `candidate_l2s` payload. `--project`, `--limit`, `--chunks-dir` stay.

- [ ] **Step 1 (RED):** Update `query_integration_test.go` `TestEngramQuery_F6F91_EndToEnd` to assert the synthesis payload (no `hubs`/`subgraph_size`) and to invoke the surviving form. Run → FAIL (old fields/flags still present).
- [ ] **Step 2 (GREEN):** Remove the default path + `--synthesis`/`--tier` + subgraph/hub symbols. Make `RunQuery` always synthesize. Keep `--synthesize-l2` accepted (default-on) so the recall skill is untouched.
- [ ] **Step 3:** `targ deadcode` + `targ check-full` green; run the real recall skill against a seeded isolated vault and confirm payload parity with pre-change `--synthesize-l2`.
- [ ] **Step 4: Commit** `refactor(query): make --synthesize-l2 the sole query path; drop subgraph/hubs/--tier/--synthesis`.

### Task B.2: Update the eval harness + scripts that used the default path
**Files:** `dev/eval/cumulative/skills_auto_l2/recall/SKILL.md` (bare `engram query`), `dev/eval/run-chain-stage.sh:41` (`--tier`), `dev/eval/cumulative/validate.py` (if it asserts removed flags).
- [ ] Repoint these to `engram query --synthesize-l2 --phrase ...`; drop `--tier`. `python dev/eval/cumulative/validate.py` green. Commit `fix(eval): use the synthesize-l2 query path`.

**Phase B exit:** only one query mode exists; `targ deadcode` clean; recall payload unchanged for the skill.

---

## Phase C — Memory scrub

### Task C.1: Delete + edit stale memories
**Files:** auto-memory at `/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/memory/`.
- [ ] DELETE `feedback_l1_episode_always_extracted.md`; remove its `MEMORY.md` index line.
- [ ] EDIT (strip L1/L2/L3-tier + episode framing, KEEP the core lesson) — update both file body and, if needed, the index hook:
  - `feedback_eval_dont_bypass_component_under_test.md` (drop the "exactly one episode" example → a current recall/learn-bypass example).
  - `feedback_check_metric_sensitivity_before_crowning_cheaper_arm.md` (generalize "eager wrote ~10 L2 / L1-only" → "eager vs lazy fact crystallization").
  - `feedback_scorer_vocabulary_bias.md` (drop "L2/L3 notes" → "memory notes").
  - `feedback_confirm_model_before_building.md` (drop the "L3 tier" anecdote framing → generalize).
- [ ] No git (auto-memory is outside the repo). Verify `MEMORY.md` index matches files on disk.

---

## Phase D — Documentation scrub + new docs

### Task D.1: Fix stale references
**Files + fixes (verified by the doc audit):**
- `README.md`: dual-vector sidecar shape (situation+body+last_used, not single `vector`); vault bootstrap creates `.obsidian/`+`.gitignore`+`README.md` (not `Permanent/`/`MOCs/`); graph caption (drop MOCs/Permanents/cascade); `engram query` line → describe `--synthesize-l2` two-channel + clusters + `candidate_l2s` (drop "3-hop subgraph + hubs").
- `docs/GLOSSARY.md`: `subagent` def (drop cascade); `--project` (drop "3-hop BFS" → bounded matched set).
- `docs/architecture/c1-system-context.md`: recall sequence + payload (drop hubs/anchor-from-hubs); **correct the OpenCode line** — engram does NOT read OpenCode sessions (backend deleted); state JSONL-only.
- `docs/architecture/adr.md`: drop `nearest_l3` from ADR-0004 channels; fix the `loadCompatibleSidecars` line cite.
- `docs/triage.md`: move `Permanent/`/`MOCs/` items to Decided/Retired; mark `StartingPoints` resolved (deleted).
- `skills/recall/SKILL.md`: the `nearest_l2` red-flag row references a field name that never shipped — fix or drop. **(SKILL.md edit → use superpowers:writing-skills.)**
- [ ] Apply; commit `docs: scrub stale recall-v2 references across docs + skills`.

### Task D.2: Add docs for modern functionality
- [ ] README + GLOSSARY (+ c-docs where structural): dual-vector sidecar (`situation_vector`/`body_vector`, `bestVector` axis selection); the `engram resituate`, `engram migrate-links`, `engram check` subcommands (add to README command table); note-recency decay / `LastUsed` (ACT-R). Commit `docs: document dual-vector sidecar, recency decay, and missing subcommands`.

---

## Phase E — Refactor remaining for clarity/conciseness/consistency

- [ ] After deletions, re-read the touched files (esp. `query.go`, now much smaller) and tighten: collapse now-trivial indirection, unify naming, drop comments that referenced removed paths, ensure the file reads as one coherent synthesis-only query. RED→GREEN guarded by existing tests + `targ check-full`. Gate B (design-fit) reviews this refactor. Commit `refactor(query): tighten synthesis-only query path`.

---

## Self-Review
- **Delete unused:** Phase A (confirmed-dead) + Phase B (decided cut). Every deletion compiler/deadcode/test-verified; load-bearing symbols (vaultgraph for amend/check, LetterLess, NewJSONLReader) explicitly preserved. ✓
- **Refactor remaining:** Phase E. ✓
- **Docs scrub + new docs:** Phase D (all audit findings) — incl. correcting the OpenCode claim to match the deletion. ✓
- **Memory lessons:** Phase C (scrub) + Step 7 closing `/learn` crystallizes the doc/code-freshness lessons. ✓
- **Guardrail honored:** the two load-bearing-not-dead paths were surfaced and decided by the user before inclusion. ✓
- **No skill broken:** `--synthesize-l2` stays valid; SKILL.md edits go through writing-skills. ✓

---

## Gate A resolutions (binding refinements)

Four adversarial reviewers (ask/code/docs/clarity) passed both user decisions and found no scope creep or unsafe deletion. Their refinements, all binding on the executor:

**Code-alignment (deletion cascade / orphans — `targ deadcode`+`check-full` enforce, but named so the executor isn't surprised):**
- A.4: `SessionFinder`/`NewSessionFinder`/`DirLister`/`Find` and the `sourceOpencode` const live in `internal/transcript/transcript.go` (NOT only `cli/cli.go`, which has just `osDirLister`). Delete the orphans there too once `osDirLister` and `opencode.go` are gone.
- B.1: removing `--synthesis` orphans `runSynthesisQuery` (delete the body) AND `errQueryModeConflict` + the mode-conflict check in `validateQueryArgs` (delete them). Removing `--tier`/`Tiers` requires updating `collectClusterMembers(... tiers []string)` — drop the param (or pass nil) or GREEN won't compile.
- B.2 commits WITH or immediately after B.1 (same PR/sequence): `dev/eval/run-chain-stage.sh:41` uses `--tier` and breaks the moment B.1 lands.
- A.1 before A.4 (see Global Constraints ordering).

**Docs-alignment (Phase D additions — these were missed):**
- C1 mermaid diagrams: drop `hubs` from the payload line in BOTH the recall flow (`c1:146`, and the `:147` "surface anchor concepts from hubs" step) AND the please flow (`c1:253`).
- C3 component catalog: K6 drop `identifyHubs`; K8 drop the `--synthesis`/BFS-subgraph clause and the `:155` "The `--synthesis` path uses…" paragraph.
- GLOSSARY: DELETE the now-defunct entries `cascade`, `frontier`, `anchors`, `hub`, and the 3-hop-BFS sense of `subgraph`; KEEP `cluster` (the live `--synthesize-l2` concept).
- ADRs describing the OpenCode/composite backend (ADR-0007/0010): mark **Superseded** with a one-line note ("backend was never wired; removed in the deep clean; engram reads JSONL only") — do NOT rewrite the historical decision text.
- Bootstrap: vault-init code already creates `.obsidian/.gitignore/README.md` — this is **docs-only**, no code change.
- Phase E/D: fix the stale doc comment on `transcript.budgetSegmentLines` ("Shared by … OpencodeTranscriptReader").

**Ask-alignment (under-specified non-deletion phases):**
- Phase E concrete targets: `query.go` reads as a single synthesis-only file (no `dispatchSynthesisMode` indirection, no vestigial mode branches), comments referencing removed paths gone, naming/error-wrapping consistent. **Gate B charge:** "does `query.go` read as written single-mode from the start?"
- New **Phase F (closing `/learn`)** is a real task (below), not just a self-review mention.
- Phase C gets an exit checklist (below).
- D.2 begins with a survey: diff the README command table against `engram --help` output; the enumerated list (dual-vector sidecar, resituate/migrate-links/check, recency decay) is a FLOOR — add any other gaps found.

**Clarity (executor-specificity):**
- SKILL.md (D.1): the row to fix is `skills/recall/SKILL.md` red-flags table — the line reading "You used `nearest_l2` instead of `candidate_l2s`". `candidate_l2s` is the real shipped field, but the "never shipped" phrasing about `nearest_l2`/`nearest_l3` is the stale bit; reconcile the row to current field names (writing-skills TDD).
- Memory edits (C): grep these exact strings to locate the edit — `"exactly one episode"`, `"L1-only"` / `"~10 L2"`, `"L2/L3 notes"`, `"L3 tier"` — strip the tier/episode framing, keep each lesson's core.
- B.3 payload-parity (operational): before B.1, capture a golden — `engram query --synthesize-l2 --phrase "url validation" --phrase "dedup" > /tmp/golden-before.yaml` against a seeded isolated vault (`ENGRAM_VAULT_PATH`/`ENGRAM_CHUNKS_DIR` in a tempdir); after B.1, re-run and `diff` — the matched/recent/clusters/candidate_l2s content must be identical (only the removed `hubs`/`subgraph_size` fields may differ).
- README command table: it's the fenced command block at `README.md:~70-90` (the `engram <sub> …  description` lines); add `prune`-style rows for `resituate`/`migrate-links`/`check` in that same format.

## Phase F — Capture (closing /learn)

- [ ] Run the `/learn` sweep; crystallize, at minimum, these distilled lessons (feedback notes): (1) **stale docs/memory describing removed functionality cause real design misses** — verify the shipping mechanism against code before designing against it; (2) **doc/memory scrub is first-class after any architectural removal** — a removal isn't done until docs+memory+glossary+diagrams that named the old path are updated; (3) **this project's actual recall mechanism is `engram query --synthesize-l2` clustering + covered/near/absent crystallization** — there is no 3-hop subgraph / hubs / tiers / episodes on the recall path. Verify each lands.
