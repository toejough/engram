# Land #657 — O2 inline candidate content (the #662-blocking cut)

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (or `executing-plans`).
> The SKILL.md edit (Task 2) MUST use `superpowers:writing-skills` (RED→GREEN→pressure-test→REFACTOR). Steps
> use `- [ ]`.

**Goal:** emit candidate-note content inline in `candidate_l2s` so recall Step 2.5 stops making the
per-candidate `engram show` round-trips — the agent-side tax that dominates deep-recall wall-time. Issue #657.

**Architecture:** `engram query` already loads each matched note's body once for scoring
(`rankCandidates`, query.go:1297) and keeps it in `matchedMember.content`. `candidate_l2s` are nominated from a
cluster's own matched note-members (`clusterNoteIndexFromMembers`, query.go:727) but the index currently drops
`content`. O2 threads that already-loaded content into the candidate entries — a pure render change, no new I/O.

**Tech Stack:** Go (`internal/cli/query.go`), gomega black-box query tests (`internal/cli/query_unified_test.go`),
`targ` build system, the recall SKILL.md, the C3/C4i/C5/C6 trap gate (`dev/eval/traps/gate.py`).

## Scope (verified against the code in orientation + Gate A)

- **O2 [THE deliverable].** Add `content` to `candidate_l2s`, sourced from `matchedMember.content`.
- **L2 — DONE.** SKILL.md:116–117 already skips empty-`candidate_l2s` clusters. The equivalence is exact:
  query.go nominates `candidate_l2s` *only* from a cluster's note members (`clusterNoteIndexFromMembers`), so
  **empty candidate_l2s ⟺ no note members** — #657's two-condition L2 reduces to the skill's one condition. No
  edit (writing-skills Iron Law: don't author against a passing baseline); **verify behaviorally** (Task 3).
- **L3a (batch ingest sweep), O1 (chunk content-budget): DEFERRED, tracked as open #657 remainder.** Not the
  show-round-trip win; L3a is learn-side (≠ the ROADMAP "dedupe double sweep" item — different scope, keep it
  under #657), O1 is a marginal render tweak on an already-capped budget (note 79). **#657 stays OPEN** after
  this plan (it lands the O2 cut the #662 dial needs, not the whole issue).
- **Gate correction (Gate A):** **C7 `recheck.py` is NOT a usable gate** — it has no runnable driver AND it
  stubs the `engram` binary (`write_stub_bin`), so a real-binary render change is invisible to it; and its
  fixture has never gone RED (1 fixture, paper gate per #654's own bars). The real regression gate is
  **`gate.py --tier smoke`** (runs the real binary via the warm harnesses). Do **not** claim C7 certifies O2 or
  close #654 off this.
- **Stale:** #657's "recall is 350s" is dated (recall-only ~94–190s); the O2 win is the removed `show`
  round-trips, not the headline seconds (post a one-line #657 comment).

## Task 1: O2 binary — `candidate_l2s` carry inline content

**Files:** Modify `internal/cli/query.go`; Test `internal/cli/query_unified_test.go`.
**Interfaces — Produces:** `queryCandidateNote.Content string \`yaml:"content,omitempty"\`` (consumed by Task 2).

- [ ] **Step 1 — RED test.** In `query_unified_test.go`, extend the candidate struct parsed by
  `TestRunQuery_ChunkClustersCarryCandidateL2s` (~:15) with `Content string \`yaml:"content"\``, and add a
  gomega assertion that a candidate_l2 for a planted note carries `Content` equal to that note's body.
- [ ] **Step 2 — Run it, expect FAIL.** `targ test` → fails (field absent / empty content).
- [ ] **Step 3 — GREEN.** Add `Content string \`yaml:"content,omitempty"\`` to `queryCandidateNote` (:275). Add
  `content []string` to `candidateNoteIndex` (:192) and populate it from `matchSet.members[i].content` in
  `clusterNoteIndexFromMembers` (:727). Emit `Content` in `topKCandidateNotes` (:1657–1690), aligned to the
  ranked `paths`. **Source content from the matched member, NOT items[]** (items[] is `--project`-filtered;
  matchSet is not — sourcing from the member is correct in all cases).
- [ ] **Step 4 — Run it, expect PASS.** `targ test`.
- [ ] **Step 5 — Verify + full check.** `targ check-full` GREEN (lint + coverage). `go install ./cmd/engram`
  then `engram query --phrase ... ` on a seeded vault → confirm `candidate_l2s` entries render `content:`.
- [ ] **Step 6 — REFACTOR + Gate B** (design-fit on the diff: the content-join reads as part of the cluster
  assembly, not bolted on).
- [ ] **Step 7 — Commit.** `git add internal/cli/query.go internal/cli/query_unified_test.go && git commit`.

## Task 2: O2 skill — read candidate content inline (writing-skills TDD)

**Files:** `skills/recall/SKILL.md` (gated); `~/.claude/skills/recall/SKILL.md` (live-use mirror — NOT
gate-required; the harness copies the repo skill).
**Interfaces — Consumes:** Task 1's `candidate_l2s[].content`.

- [ ] **Step 1 — RED (writing-skills).** Baseline harness: a fresh subagent given a recall payload (with
  candidate_l2s now carrying content) + the *current* skill, neutrally prompted to do Step 2.5; count
  candidate `engram show` calls. Current skill (SKILL.md:122 "Run `engram show <path>` on every entry in
  `candidate_l2s`") → agent makes ~5 redundant `show` calls. Record the count (pass bar: 0).
- [ ] **Step 2 — GREEN.** Edit SKILL.md:122 → *"`candidate_l2s` entries carry their `content` inline — read it
  directly; **no `engram show` calls for candidates**."* (Keep the chunk-member `show-chunk` path.)
- [ ] **Step 3 — Re-run baseline, expect 0 candidate `show` calls.**
- [ ] **Step 4 — Pressure-test** (writing-skills): a fresh agent under "be thorough / the payload might be
  truncated" pressure must still not `show` candidates. Close any rationalization loophole found.
- [ ] **Step 5 — REFACTOR + Gate B.**
- [ ] **Step 6 — Commit** (repo skill; mirror the edit to `~/.claude/...` for live use).

## Task 3: Doc + diagram scrub (note 64 — part of the change, not optional)

O2 changes two documented contracts: the `candidate_l2s` **shape** (`{path,cosine}`→`{path,cosine,content}`)
and the **Step-2.5 call sequence** (no candidate `show`). Scrub every doc naming the old shape/round-trip:

- [ ] SKILL.md:77, :114 (shape `[{path, cosine}]` → add `content`); :276 red-flag (drop the `engram show`
  mechanism, keep "read candidate content first"). **L2 behavioral verify** here too (a fresh agent skips a
  chunk-only cluster, writes nothing).
- [ ] `docs/architecture/c1-system-context.md`:89 (shape), :154 (delete the candidate `engram show` sequence
  message; keep `show-chunk`).
- [ ] `docs/architecture/c2-containers.md`:97/163 (shape), :99/165 + :184 flowchart (candidate `show`).
- [ ] `docs/architecture/c3-components.md`:138 (payload contract shape).
- [ ] `README.md`:131 (shape); :85 CLI ref (drop "Used by /recall to read candidate notes" from `engram show`).
- [ ] `docs/GLOSSARY.md`:157 (canonical schema `[{path, cosine}]` → add `content`).
- [ ] `docs/ROADMAP.md`:145 — **un-park** "inline candidate_l2 content" and move it to the Shipped payload-cuts
  list (matching how lazy-chunks/recent-fill were recorded).
- [ ] **Gate C** over every touched doc.
- [ ] **Commit.**

## Task 4: Regression gate + close-out

- [ ] **BEFORE** (baseline, run first): `cd dev/eval/traps && python3 gate.py --tier smoke` → GREEN. (C7 is
  NOT run as an O2 gate — it stubs the binary.)
- [ ] **AFTER**: `go install ./cmd/engram` (so the gate tests the NEW binary), then `python3 gate.py --tier
  smoke` → must stay GREEN. O2 delivers identical content, so no capability regression is expected; the gate
  proves it on the real binary + real skill.
- [ ] Post a #657 comment: O2 landed (with the gate evidence); L2 confirmed done; L3a/O1 remain (so **#657
  stays open**); note the stale "350s" figure.

## Global constraints
- `targ` for all Go test/lint (`targ test`, `targ check-full`) — never `go test` directly.
- writing-skills TDD (RED→GREEN→**pressure-test**→REFACTOR) for the SKILL.md edit — no exceptions.
- `go install ./cmd/engram` before any AFTER trap-gate run (else it tests the old binary).
- Source candidate content from `matchedMember.content`, never an `items[]` lookup (`--project` correctness).
