# Knowledge-Elicitation Frameworks — Briefing Gap + B Headroom Probe

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Realize the value of the frameworks exploration via the two concrete gaps, per Joe's disposition "Briefing gap now + probe B": (1) add a prior-work/failure element to the briefing convention (free); (2) run a cheap warm-vs-warm retrieval headroom probe for a recall artifacts-angle BEFORE any win-nucleus change.

**Architecture:** No shared-taxonomy artifact is built — memory (note 73: engram's value is the memory, not machinery; anti-YAGNI) reversed that lean; the taxonomy stays a conceptual lens. Two independent units: a **convention update** (amend the briefing note + its CLAUDE.md mirror — no code, no gate beyond doc review) and a **retrieval measurement** (pure `engram query` comparisons, zero API spend — no skill change this cycle). The recall SKILL.md 10q is NOT touched this cycle; whether to add an artifacts angle is a follow-up gated on the probe's result.

**Tech Stack:** engram vault notes (`engram amend`); the CLAUDE.md memory mirror (markdown); `engram query` for the probe (Python/bash harness, no API).

## Global Constraints

- **Do NOT touch the win-nucleus** (note 100): Step-3 conventions directive, Step-2.5B recency-weight, Step-2 matched-note retrieval, frontmatter description. This cycle does NOT edit `skills/recall/SKILL.md` at all.
- **Any recall-path addition must PROVE warm-vs-warm headroom before shipping** (note 73 — "usually there is none"). The probe is that proof; a null result PARKS proposal B.
- **The 10-phrase breadth stays load-bearing** (notes 72/100) — the probe measures whether an artifacts angle ADDS coverage, never whether to cut existing angles.
- **Preserve the systems/artifacts-independent distinction** (note 261).
- **The probe is a retrieval comparison, not an applied eval** (note 104: a free retrieval probe bounds expensive evals) — it uses `engram query` only, no `claude -p` trials.

---

## File Structure

- Vault note `261.2026-07-13.briefing-format-systems-and-artifacts-independent.md` — AMEND to add a prior-work/failure briefing element (8 parts).
- `~/.claude/projects/-Users-joe-repos-personal-engram/memory/feedback_issue_briefing_six_part_format.md` + `MEMORY.md` — mirror the amended format.
- `$CLAUDE_JOB_DIR/tmp/b-headroom-probe.{py,json}` — the probe harness + result (NOT committed; recorded in the LEDGER).
- `dev/eval/LEDGER.md` — record the probe finding.
- `docs/superpowers/plans/2026-07-13-frameworks-briefing-and-probe.md` — this plan.

---

## Task 1: Briefing gap — add a prior-work/failure element (convention, free)

**Files:** vault note 261 (amend); CLAUDE.md mirror `feedback_issue_briefing_six_part_format.md` + `MEMORY.md`.

**Interfaces:** the briefing goes from 7 parts to 8, inserting a **prior-work/failure-modes** element after "current states": 1 problem; 2 systems+relationships; 3 artifacts+relationships; 4 current states (verified live); **5 prior work / failure modes bearing on this decision (what's been tried, what closed it, known pitfalls — the decision-time analogue of recall's Step-3.5 re-entry check)**; 6 solution per option; 7 before/after (differs AND same) per option; 8 how it solves. Parts 6–8 per option.

- [ ] **Step 1 (RED analogue):** state the gap concretely — the current 7-part briefing (note 261) has NO element that surfaces prior attempts / known failure modes at decision time, so a checkpoint can present an option whose prior-closure isn't shown (the #690 template-assist checkpoint is the worked example: the "already a template" fact surfaced only at Gate A, not in the briefing). Write this as the amend's justification.
- [ ] **Step 2 (GREEN):** `engram amend --target 261.2026-07-13.briefing-format-systems-and-artifacts-independent --behavior/--impact/--action` re-synthesized to the 8-part format above, keeping the systems/artifacts-independent distinction and the remote-control delivery rule intact.
- [ ] **Step 3:** update the CLAUDE.md mirror `feedback_issue_briefing_six_part_format.md` (body + description) and the `MEMORY.md` index line to the 8-part format.
- [ ] **Step 4 (verify):** re-read both the vault note and the mirror; confirm the 8 parts are coherent, the prior-work element is distinct from current-states (states = what IS; prior-work = what was TRIED), and no existing element was lost. Gate C covers the doc review.

---

## Task 2: B headroom probe — does an artifacts retrieval angle surface memory the 10 angles miss?

**Files:** `$CLAUDE_JOB_DIR/tmp/b-headroom-probe.py` (harness); `b-headroom-probe.json` (result).

**Interfaces:** pure `engram query` comparison, zero API spend. For each fixture situation with a concrete artifact central to it, compare the memory surfaced by the current 10 angle-phrases vs. the same 10 PLUS one artifacts-keyed phrase.

- [ ] **Step 1: Build the fixture set (≥6 realistic artifact-central situations).** Each names a concrete engram file, e.g.: editing `recall_time.py`'s segmenter; modifying `skills/recall/SKILL.md` Step 1; debugging `internal/cli/query.go` clustering; changing `dev/eval/traps/gate.py`; touching `internal/embed` sidecars; editing the `route` SKILL. For each, author the 10 topic/situation angle-phrases (as the recall skill would) AND one artifacts-keyed phrase (the bare file/path + "prior lessons about <file>").
- [ ] **Step 2: Run the comparison.** For each fixture: (a) `engram query` with the 10 angle-phrases → record the surfaced NOTES (kind fact/feedback) and top chunks; (b) `engram query` with the 10 + the artifacts phrase → record the same. Compute what the artifacts phrase UNIQUELY adds (notes/high-relevance chunks present in (b) but not (a)'s top-N).
- [ ] **Step 3: Classify each unique add.** Is it a file-specific LESSON the topic angles missed (real headroom), or just raw file-mention chunks with no actionable lesson (weak/no headroom)? Count fixtures with ≥1 unique actionable add.
- [ ] **Step 4: Verdict + record.** Headroom EXISTS iff a majority of fixtures gain ≥1 unique actionable note the 10 angles missed. Write the finding to `dev/eval/LEDGER.md` (`b-artifacts-angle-headroom`). If headroom → recommend a follow-up gated skill change (add the artifacts angle via writing-skills TDD + trap + coverage) as a filed issue, NOT built this cycle. If null → PARK B (note 73 confirmed: no headroom), file nothing.
- [ ] **Step 5: CHECKPOINT** — present the probe result (labeled table, units) in the 8-part briefing; Joe disposes B's fate (file the follow-up, or park).

---

## Self-Review

**Spec coverage:** Joe's disposition "Briefing gap now + probe B" → Task 1 (briefing, C) + Task 2 (headroom probe, B). "Taxonomy = lens not built" → Architecture (no taxonomy artifact). Covered.

**Nucleus safety:** this cycle does NOT edit `skills/recall/SKILL.md`; the artifacts angle is only PROBED, its build deferred behind the probe + a future gate. Note 73's headroom requirement is the probe's whole purpose.

**Placeholder scan:** Task 1 is a concrete note amend + mirror update; Task 2 is a concrete `engram query` comparison with a fixture set and a majority-of-fixtures verdict rule. No TBDs. The probe fixtures are named by real engram files.

**Anti-over-engineering:** no taxonomy artifact, no premature skill change — the two units are the minimal realization of the disposed value, per note 73.
