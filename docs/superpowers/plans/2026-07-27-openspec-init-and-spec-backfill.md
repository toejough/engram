# OpenSpec Init + Spec Backfill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Initialize OpenSpec (Fission-AI, v1.6.0) in the engram workspace with Claude Code integration, backfill one behavior spec per shipped capability in `docs/FEATURES.md`, and make `openspec/specs/` the primary behavior record (FEATURES.md slims to a pointer).

**Ask (verbatim, no GitHub issue — delivered via `/please` 2026-07-27):** "init openspec in this workspace and interact with it to create the relevant docs for our existing features"

**Architecture:** `openspec init --tools claude` scaffolds `openspec/` (config.yaml, specs/, changes/archive/) and the `/opsx:` Claude Code commands. Backfill writes main specs directly under `openspec/specs/<id>/spec.md` (no change proposals — we are recording existing behavior, the same backfill pattern used in pi-vim-mode 2026-07-27). Every spec is derived from **code and skill sources, not doc prose**, with FEATURES/ADR/LEDGER used as provenance pointers that get re-verified. The docs scrub then retargets the "what's shipped" surface to the spec tree.

**Tech Stack:** OpenSpec CLI 1.6.0 (installed at `/opt/homebrew/bin/openspec`); markdown specs; no Go changes, no engram binary changes, no SKILL.md edits.

## Decisions (user-ratified 2026-07-27)

- Tool integration: **claude only** (`--tools claude`).
- Backfill scope: **the FEATURES.md surface** — one spec per shipped capability (19 entries; the closing "Validated goals" section is a rollup, not a capability, and gets no spec).
- Doc authority: **specs become primary**. FEATURES.md slims to a pointer (retaining the Validated-goals rollup); ROADMAP/README/LEDGER references retarget.
- Consistency-edit set (ratified in the Gate A round, 2026-07-27): Task 8 ALSO updates docs/GLOSSARY.md (OpenSpec entry), CLAUDE.md (Directory Structure `openspec/` line), and docs/README.md:21 (the design-doc graduation rule, which becomes misleading once specs are primary).

*Author's mitigation (not a user decision):* the drift risk between the two doc surfaces, raised as a challenge during orientation, is mitigated by (a) code-derived specs, (b) cross-references instead of duplication, (c) the docs-index row pointing at the spec tree.

## Global Constraints

- **Code truth:** every behavioral claim in a spec is verified against the named code/skill file before it is written. FEATURES.md/ADR/LEDGER prose is provenance to re-verify, never ground truth (this repo's CLAUDE.md prose has diverged from the binary before). Verification procedure, per claim: (1) grep/read the named file and confirm the symbol/flag/path exists; (2) read the surrounding code (or the SKILL.md/guidance text — for skill-implemented capabilities the skill text IS the implementation) far enough to confirm the claimed behavior; (3) for claims observable at the CLI (a flag, an output line), exercise the installed `engram` binary read-only when cheap to do so. A spec may name a file path or symbol only after step (1).
- **Status voice:** specs describe current shipped behavior in present tense. Anything committed-but-not-deployed or unvalidated (e.g. the surprise-harvest addition inside the write-memory capability) is stated with that exact status, dated — never in the shipped voice.
- **Commit gating (overrides any skill default):** executors NEVER commit. Stage nothing; leave the working tree for the orchestrator. Commits happen only at the workflow's designated points (this plan's commit at Step 3; implementation commits at Step 6 after Gate D), via the repo's commit skill with the `AI-Used: [claude]` trailer.
- **Scope walls:** no edits to `agent-instructions/` (specs *describe* skills; editing a SKILL.md would trigger the writing-skills TDD workflow — out of scope), no Go changes, no `dev/eval/LEDGER.md` content changes beyond the one wording line named in Task 8.
- **Validation gate:** `openspec validate --specs --strict --no-interactive` from the repo root; pass = output line `Totals: N passed, 0 failed (N items)`. Strict mode turns warnings (e.g. Purpose < 50 chars) into failures. Use verb-first commands (`openspec validate --specs`), not the deprecated `openspec spec validate`. (All "measured 2026-07-27" claims in this plan were measured by the plan author against scratch probe projects in the authoring session's scratchpad; the Gate A code-alignment reviewer independently reproduced every one against the real 1.6.0 binary.)
- **Spec file shape (measured against the real validator):**

  ```markdown
  # <Title> Specification

  ## Purpose
  <≥50 chars: what this capability does and why it exists. Include provenance
  pointers in prose: "Why: docs/architecture/adr.md ADR-NNNN. Validation:
  dev/eval/LEDGER.md#<anchor>." Keep pointers inside Purpose — extra top-level
  sections are unverified against the validator; do not invent them.>

  ## Requirements

  ### Requirement: <name>
  <One-sentence SHALL statement plus minimal context.>

  #### Scenario: <name>
  - **WHEN** <condition>
  - **THEN** <observable outcome>
  ```

  Exactly two top-level `##` sections: `## Purpose` and `## Requirements`. `### Requirement:` headings nest under Requirements, `#### Scenario:` headings nest under their Requirement. Do not add other `##` sections.

- **Live-tree re-derivation:** Task 1 Step 0 (pre-flight) re-lists FEATURES.md's `## ` headings. Any drift from the pinned inventory → STOP and re-scope with the orchestrator before writing specs.

## Spec inventory (pinned from docs/FEATURES.md as of commit 2822e484)

Spec ids are PINNED in this table — executors use them verbatim, never re-derive them from headings. Anchors in the last column were verified to exist at plan-authorship time; the executor MUST re-verify each (Step B) before claiming behavior from it.

| # | FEATURES.md entry | spec id | anchors (verified at authorship; re-verify at execution) |
|---|---|---|---|
| 1 | Matched-note floor | `recall-matched-note-floor` | `internal/cli/query.go` (`capWithNoteFloor`) |
| 2 | Payload cuts (--lazy-chunks, --recent-fill) | `recall-payload-cuts` | `internal/cli/query.go`, `internal/cli/query_chunks.go` |
| 3 | Query timing instrument (--timings) | `recall-query-timings` | `internal/cli/query.go` |
| 4 | Glance/deep recall dial | `recall-glance-deep-dial` | `agent-instructions/skills/recall/SKILL.md` (Modes section) |
| 5 | Unified two-channel recall payload | `recall-two-channel-payload` | `internal/cli/query.go`, `internal/cli/query_chunks.go` |
| 6 | Recall-at-decision-moments guidance (@import) | `guidance-recall-moments` | `agent-instructions/guidance/recall.md`, `internal/cli/update.go` |
| 7 | Delegate-object-level-work guidance (@import) | `guidance-delegate` | `agent-instructions/guidance/delegate.md` |
| 8 | Please Step 3 doc-surface enumeration grep + Gate A | `please-doc-enumeration-gate` | `agent-instructions/skills/please/SKILL.md` |
| 9 | Evidence-based route rubric | `route-evidence-rubric` | `agent-instructions/skills/route/SKILL.md` |
| 10 | Route dispatch evidence + aggregates (tags-based) | `route-dispatch-evidence` | `agent-instructions/skills/route/SKILL.md`, `internal/cli/learn.go` (`--tag`), `internal/cli/count.go` |
| 11 | Vocab lifecycle | `vault-vocab-lifecycle` | `internal/cli/vocab.go`, `vocab_commands.go`, `vocab_centroids.go`, `vocab_trigger.go`, `query_nominations.go` |
| 12 | Q&A memory round-1 (learn qa) | `learn-qa-capture` | `internal/cli/qa.go`, `internal/cli/learn.go` |
| 13 | Concurrency-safe vault writes | `vault-concurrency-safe-writes` | `internal/cli/locker.go` |
| 14 | Write-memory worker + capture guards | `write-memory-worker` | `agent-instructions/skills/write-memory/SKILL.md`, `learn/SKILL.md`, `please/SKILL.md` |
| 15 | Embed-on-write + dual-vector sidecars | `vault-embed-on-write` | `internal/embed/`, `internal/cli/embed.go` |
| 16 | Chunk-index dedup by content hash | `ingest-chunk-dedup` | `internal/cli/ingest_dedup.go`, `internal/cli/prune_duplicates.go` |
| 17 | Ingest auto-sweep with non-persistent-workspace skip | `ingest-auto-sweep` | `internal/cli/sweepspec.go`, `internal/cli/ingest.go`, `internal/cli/prune.go` |
| 18 | Prune preserves memory (detach on source deletion) | `ingest-prune-detach` | `internal/cli/prune.go` |
| 19 | Count / backlinks aggregation (engram count) | `cli-count` | `internal/cli/count.go` |

## Doc-surface enumeration (grep-derived 2026-07-27; reviewers re-verify)

Greps run: `FEATURES` (case-sensitive, all *.md), `what's shipped` (case-insensitive), `openspec` (case-insensitive — zero pre-existing hits repo-wide).

| file | disposition | detail |
|---|---|---|
| `docs/README.md:9` | update | "see what's shipped" row → `../openspec/specs/` (behavior specs, primary); FEATURES.md noted as pointer + mission rollup |
| `docs/FEATURES.md` | rewrite | slim to a pointer at `openspec/specs/` + retain the closing "Validated goals" rollup verbatim |
| `docs/ROADMAP.md:3` | update | header cite "Shipped work: `docs/FEATURES.md`" → `openspec/specs/` |
| `docs/ROADMAP.md:198` | update | inline cite "`docs/FEATURES.md` — Count / backlinks aggregation" → `openspec/specs/cli-count/spec.md` |
| `docs/ROADMAP.md:209` | update | inline cite "…Matched-note floor" → `openspec/specs/recall-matched-note-floor/spec.md` |
| `docs/ROADMAP.md:210` | update | inline cite "…Write-memory worker + capture guards" → `openspec/specs/write-memory-worker/spec.md` |
| `docs/ROADMAP.md:213` | update | inline cite "…Q&A memory round-1" → `openspec/specs/learn-qa-capture/spec.md` |
| `dev/eval/LEDGER.md:6` | update | "ROADMAP and FEATURES cite rows" → "ROADMAP and the openspec specs cite rows" |
| `docs/README.md:21` | update | design-doc graduation rule "conclusions graduate into FEATURES/ROADMAP/ADR" → name `openspec/specs/` as the destination for behavior conclusions (why → ADR, status → ROADMAP; FEATURES.md is no longer a graduation destination) |
| `docs/GLOSSARY.md` | update | add an "OpenSpec" entry (what it is, where specs live, that it is the primary behavior record) |
| `CLAUDE.md` (repo) | update | Directory Structure gains `openspec/` line (primary behavior specs; backfilled from FEATURES surface) |
| `docs/superpowers/plans/*.md` (6 historical files matching FEATURES; recount 2026-07-27 by Gate A) | N/A | past-cycle plan records — never scrubbed |
| `agent-instructions/`, `docs/architecture/`, `README.md` | N/A | zero FEATURES references (grep-verified 2026-07-27) |

---

### Task 1: Initialize OpenSpec with Claude Code integration

**Files:**
- Create (generated): `openspec/config.yaml`, `openspec/specs/` (empty), `openspec/changes/archive/` (empty), 6 skills under `.claude/skills/openspec-*/`, 6 commands under `.claude/commands/opsx/` (set verified on a probe init 2026-07-27; re-verify against the actual `git status --porcelain` output in Step 3)
- Modify: `openspec/config.yaml` (fill the context block)

**Interfaces:**
- Produces: an initialized OpenSpec root at the repo root; later tasks write `openspec/specs/<id>/spec.md` under it.

- [ ] **Step 0: Pre-flight**

Run from the repo root: `openspec --version` (expected: `1.6.0`) and `grep -c '^## ' docs/FEATURES.md` (expected: `20` — the 19 pinned capability entries plus the "Validated goals" rollup). Then `grep -n '^## ' docs/FEATURES.md` and diff the headings against the inventory table. Any drift → STOP and re-scope with the orchestrator.

- [ ] **Step 1: RED — prove no OpenSpec root exists**

Run: `test -d openspec && echo exists || echo absent` from the repo root
Expected: `absent` (there is no `openspec/` dir at HEAD). Note: `openspec list` prints `No active changes found.` even outside a root — it is NOT a usable existence probe (measured 2026-07-27).

- [ ] **Step 2: Initialize**

Run: `openspec init --tools claude`
Expected: "OpenSpec structure created" + "Config: openspec/config.yaml (schema: spec-driven)" (output shape measured on a probe 2026-07-27).

- [ ] **Step 3: GREEN — verify the root and integration**

Run: `openspec list --specs` (expected: `No specs found.` — measured) and `git status --porcelain` (enumerate exactly what init generated).
Expected generated set (verified by Gate A on a probe init 2026-07-27; re-verify against the actual output): `openspec/config.yaml`, empty `openspec/specs/` and `openspec/changes/archive/` dirs, 6 skills under `.claude/skills/openspec-*/`, 6 commands under `.claude/commands/opsx/` — no collision with the repo's tracked `.claude/skills/commit.md` or `.claude/commands/commit.md`. Note: git cannot commit empty directories, so `openspec/changes/archive/` will not appear in any commit — expected and harmless; the CLI degrades gracefully without it and recreates it whenever a new change is created (re-tested by Gate A). Record the actual file list in the execution log for the commit and Gate B.

- [ ] **Step 4: Fill `openspec/config.yaml` context**

Replace the commented context block with:

```yaml
schema: spec-driven

context: |
  Engram: persistent memory for LLM agents (Go, pure-Go no-CGO, DI-everywhere —
  no direct os/http/sql calls in internal/, all I/O through injected interfaces).
  Specs in openspec/specs/ are the PRIMARY behavior record, backfilled 2026-07-27
  from docs/FEATURES.md; why lives in docs/architecture/adr.md (ADR-0001..0021),
  measurements in dev/eval/LEDGER.md. Build/test/lint via targ (targ test,
  targ check-full), binary install via go install ./cmd/engram.
  Conventions: behavioral claims are verified against code before being written;
  status language only in the voice the evidence supports.
```

- [ ] **Step 5: Re-validate**

Run: `openspec validate --all --no-interactive`
Expected: `No items found to validate.` (measured 2026-07-27 on a fresh init).

### Tasks 2–7: Backfill specs by domain

Shared step template for every spec in a task (repeat per spec id; all six tasks are independent of each other and parallelizable — different directories, no shared files):

- [ ] **Step A: RED** — `openspec show <id> --type spec --no-interactive` → expected: `✖ Error: Spec '<id>' not found at <repo>/openspec/specs/<id>/spec.md` (shape measured 2026-07-27).
- [ ] **Step B: Read the verified anchors** for the spec id (inventory table above) plus the matching FEATURES.md entry and its ADR/LEDGER pointers. Re-verify every symbol/flag/behavior you are about to claim (grep/read the source). Where FEATURES prose and code disagree, the code wins and the discrepancy is reported to the orchestrator in the task summary.
- [ ] **Step C: Write** `openspec/specs/<id>/spec.md` in the exact shape from Global Constraints: Purpose ≥ 50 chars ending with `Why: docs/architecture/adr.md ADR-NNNN. Validation: dev/eval/LEDGER.md#<anchor>.` (or the entry's stated non-LEDGER validation, quoted accurately — e.g. "test-locked, not eval-measured"), then one `### Requirement:` per distinct SHALL-able behavior the sources evidence, each with at least one WHEN/THEN scenario grounded in actually-verified behavior. Requirement-count rule: ≥ 1; a single-requirement spec is valid; typical is 2–5; if you find more than 6 or cannot state any SHALL-able behavior, STOP and escalate to the orchestrator instead of padding or forcing.
- [ ] **Step D: GREEN** — `openspec validate <id> --type spec --strict --no-interactive` → expected: pass (0 failed).
- [ ] **Step E:** Do NOT commit (Global Constraints). Report the spec path + any code-vs-FEATURES discrepancies found.

**Task 2 — recall domain (5 specs):** `recall-matched-note-floor`, `recall-payload-cuts`, `recall-query-timings`, `recall-glance-deep-dial`, `recall-two-channel-payload`

**Task 3 — guidance/skills domain (5 specs):** `guidance-recall-moments`, `guidance-delegate`, `please-doc-enumeration-gate`, `route-evidence-rubric`, `route-dispatch-evidence`

**Task 4 — vault domain (3 specs):** `vault-vocab-lifecycle`, `vault-concurrency-safe-writes`, `vault-embed-on-write`

**Task 5 — learn domain (2 specs):** `learn-qa-capture`, `write-memory-worker`

**Task 6 — ingest domain (3 specs):** `ingest-chunk-dedup`, `ingest-auto-sweep`, `ingest-prune-detach`

**Task 7 — cli domain (1 spec):** `cli-count`

### Task 8: Docs scrub — make the spec tree primary

**Files:**
- Modify: `docs/FEATURES.md`, `docs/README.md`, `docs/ROADMAP.md`, `dev/eval/LEDGER.md`, `docs/GLOSSARY.md`, `CLAUDE.md`

Depends on Tasks 2–7 (cites spec ids that must exist).

- [ ] **Step 1: RED** — run the disposition greps and confirm the pre-state: `grep -n "docs/FEATURES.md" docs/ROADMAP.md` → 5 hits (lines 3, 198, 209, 210, 213); `grep -n "what's shipped" docs/README.md` → 1 hit (line 9); `grep -n "ROADMAP and FEATURES" dev/eval/LEDGER.md` → 1 hit (line 6); `grep -in "openspec" docs/GLOSSARY.md CLAUDE.md` → 0 hits.
- [ ] **Step 2:** Apply every `update`/`rewrite` row in the doc-surface table above. FEATURES.md's rewrite: replace everything between the title and the `## Validated goals` heading with one short pointer paragraph (specs = `openspec/specs/`, why = adr.md, measurements = LEDGER.md, per-capability validation anchors now live inside each spec's Purpose); keep the title and the "Validated goals (mission rollup — not a capability)" section verbatim.
- [ ] **Step 3: GREEN** — re-run the Step-1 greps; expected: `docs/FEATURES.md` cites in ROADMAP → 0 (all retargeted to `openspec/specs/...`), README row points at the spec tree, LEDGER line reads "the openspec specs", GLOSSARY + CLAUDE.md each gain exactly one OpenSpec reference block. Then verify every retargeted path exists (loop verified 2026-07-27 — prints nothing when all exist):

```bash
for id in recall-matched-note-floor recall-payload-cuts recall-query-timings \
  recall-glance-deep-dial recall-two-channel-payload guidance-recall-moments \
  guidance-delegate please-doc-enumeration-gate route-evidence-rubric \
  route-dispatch-evidence vault-vocab-lifecycle learn-qa-capture \
  write-memory-worker vault-concurrency-safe-writes vault-embed-on-write \
  ingest-chunk-dedup ingest-auto-sweep ingest-prune-detach cli-count; do
  test -f "openspec/specs/$id/spec.md" || echo "MISSING: $id"
done
```

Expected: no output.
- [ ] **Step 4:** Do NOT commit.

### Task 9: Full validation

- [ ] **Step 1:** `openspec validate --all --strict --no-interactive` from the repo root. Expected: `Totals: 19 passed, 0 failed (19 items)`.
- [ ] **Step 2:** `openspec list --specs` — expected: a `Specs:` block listing all 19 ids with their requirement counts (shape measured 2026-07-27: `  <id>  requirements N`).
- [ ] **Step 3:** Report the final file inventory (`git status --porcelain`) to the orchestrator for Gate B/C/D and the Step-6 commits.

(Plan-authoring measurement probes — `openspec-probe`, `openspec-probe2` — live in the authoring session's scratchpad and are cleaned up by the orchestrator at cycle close, not by plan executors; they are not execution prerequisites.)
