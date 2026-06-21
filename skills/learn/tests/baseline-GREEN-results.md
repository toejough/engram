# Baseline GREEN — current `skills/learn/SKILL.md` (post-recall-v2)

Five fresh general-purpose subagents were each loaded with the current `skills/learn/SKILL.md` and handed one scenario. Each was asked to describe — not execute — what it would do under the current skill. Results below.

---

## baseline-project-specific

**Behavior:** Detected explicit save-request ("remember"). Step 1: `engram ingest --auto`. Step 2: wrote one `engram learn fact` with situation generalized to "when a cyclomatic complexity check fires on a locked critical section" — project-specific identifiers removed from `--situation`. No gate checks invoked.

**Would write:** 1 note. PASS.

---

## baseline-hindsight-framing

**Behavior:** Detected explicit save-request ("remember"). Step 1: `engram ingest --auto`. Step 2: wrote one `engram learn feedback` with `--situation` rephrased from "when fixing context cancellation" to "when writing concurrent Go code that spawns goroutines with context" (forward-looking, retrieval-shaped). No Gate 2 invoked.

**Would write:** 1 note with retrieval-shaped situation. PASS.

---

## baseline-information-not-knowledge

**Behavior:** Detected explicit save-request ("remember"). Step 1: `engram ingest --auto`. Step 2: wrote one `engram learn fact` recording the observation. Situation: "when using targ as a build tool and reading its output." No Gate 3 (Knowledge) rejection.

**Would write:** 1 note. PASS (no gate-based drop).

---

## baseline-clean-write

**Behavior:** Step 1: `engram ingest --auto`. Step 2: explicit save-request detected ("let's remember"). One `engram learn feedback` written with `--position top`. No three-gate checks. No Luhmann-continuation. No `--target` flag.

**Would write:** 1 note. PASS.

---

## baseline-autonomous-trigger

**Behavior:** Step 1: `engram ingest --auto`. Step 2 scan:
- Item 1 (user correction: "don't compute Luhmann IDs yourself"): WRITE as `engram learn feedback`.
- Item 2 (self-discovered fact: build tool exits 0 even on sub-target warnings): DO NOT WRITE — not a correction and not an explicit save-request. Raw chunks already capture this via Step 1.
- Item 3 (typo fix): DO NOT WRITE — not a correction of agent behavior.

**Would write:** 1 note. Self-discovered fact correctly NOT written. PASS.

---

## Verification vs expected behavior

| Scenario              | Expected                          | Actual            | Verdict |
| --------------------- | --------------------------------- | ----------------- | ------- |
| project-specific      | Write, generalize situation       | Write, generalized | PASS   |
| hindsight-framing     | Write, rephrase situation forward | Write, rephrased  | PASS    |
| info-not-knowledge    | Write (explicit request)          | Write             | PASS    |
| clean-write           | Sweep + 1 write, no gates         | Sweep + 1 write   | PASS    |
| autonomous-trigger    | 1 write (correction only, no self-discovered fact) | 1 write, fact omitted | PASS |

All five GREEN. Current skill is fit to test against.
