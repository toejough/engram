# Premortem: Cycle 5 Candidates

**Date:** 2026-03-18
**Scenario:** "It is 2026-04-01. Cycle 5 shipped. What went wrong?"

---

### Failure 1: Flush Pipeline Silently Drops Errors

**What went wrong:**
The flush command wraps learn → evaluate → context-update. Each step returns nil on missing API token (graceful skip). But a new failure mode — disk full, permission error, corrupted TOML — returns a real error. The flush pipeline stops at the first error (T-371), so evaluate failing means context-update never runs. The user loses session context because one step's transient I/O error blocked the rest. The `|| true` in stop.sh/pre-compact.sh swallows this entirely.

**Principle violated:** "Passing tests ≠ usable system." The integration test (T-337) uses a mock LLM and clean tempdir — never exercises disk-full or permission-denied paths.

**What would have caught it:**
A test where evaluate returns an error and verifies context-update still runs (fire-and-forget per step, not fail-fast).

**Remediation:**
Consider whether flush should be fail-fast (current) or fire-and-forget per step. File issue to evaluate the tradeoff. The `|| true` in shell scripts already makes the entire flush fire-and-forget at the shell level, so the Go-level fail-fast may be moot.

**Likelihood × Impact:** MEDIUM × MEDIUM

---

### Failure 2: TF-IDF Confidence Scores Are Never Acted On

**What went wrong:**
TF-IDF confidence is logged to stderr and included in MergePlan, but nothing reads those values to make decisions. The confidence score is purely informational. After 3 months, the stderr logs show confidence ranges of 0.3–0.9 but no one has used this data to tune cluster quality. The feature shipped but delivered zero value.

**Principle violated:** "Content quality > mechanical sophistication." Added a metric without a consumer.

**What would have caught it:**
Defining upfront what action confidence scores should trigger (e.g., skip low-confidence merges, alert user in dry-run).

**Remediation:**
This is by design for now — TF-IDF was explicitly scoped as observability-first (REQ-140 AC4). But if no action is ever taken on the data, consider removing it to reduce complexity.

**Likelihood × Impact:** HIGH × LOW

---

### Failure 3: Stale Spec Items Accumulate Between Cycles

**What went wrong:**
Cycle 4 code review found 20 stale spec items from prior deletions (audit, extractors, graduate-surface). The first deletion (UC-19 in #309) was incomplete — it caught UC/REQ but missed ARCH/DES/T items. This pattern repeats: every deletion leaves orphaned specs.

**Principle violated:** "Don't play whack-a-mole." The spec deletion was piecemeal across multiple commits.

**What would have caught it:**
An automated check that all spec `traces_to` references resolve to existing items. Similar to `traced verify` but checking referential integrity.

**Remediation:**
File issue for a `traced` verify check that catches dangling `traces_to` references. This would have flagged ARCH-42's reference to DES-21 after DES-21 was deleted.

**Likelihood × Impact:** HIGH × MEDIUM

---

## Priority Matrix

| Failure | Likelihood | Impact | Priority |
|---------|-----------|--------|----------|
| 1: Flush drops errors silently | Medium | Medium | P2 |
| 2: TF-IDF confidence unused | High | Low | P3 |
| 3: Stale specs accumulate | High | Medium | P1 |
