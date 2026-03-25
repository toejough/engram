# Extraction Prompt Quality Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve extraction prompt to reject tasks/common knowledge, generalize before storing, and generate context-specific keywords.

**Architecture:** Three additions to the `systemPrompt()` string in `internal/extract/extract.go`. No pipeline changes.

**Tech Stack:** Go (string constant modification)

**Spec:** `docs/superpowers/specs/2026-03-25-extraction-prompt-quality-design.md`

---

### Task 1: Add task-vs-principle and common-knowledge filters to quality gate

**Files:**
- Modify: `internal/extract/extract.go:252-264` (QUALITY GATE section of systemPrompt)
- Test: `internal/extract/extract_test.go`

- [ ] **Step 1: Write the failing test**

Add a test that verifies the prompt contains the new rejection criteria:

```go
func TestSystemPrompt_RejectsTasksAndCommonKnowledge(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	prompt := systemPrompt()
	g.Expect(prompt).To(gomega.ContainSubstring("one-time tasks or completed actions"))
	g.Expect(prompt).To(gomega.ContainSubstring("common knowledge any competent developer already knows"))
}
```

Note: `systemPrompt` is unexported, so this test is in `package extract` (whitebox). Check if `extract_test.go` already uses whitebox or blackbox pattern.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestSystemPrompt_RejectsTasksAndCommonKnowledge -v`
Expected: FAIL — prompt doesn't contain those strings yet

- [ ] **Step 3: Add the rejection criteria to systemPrompt()**

In `internal/extract/extract.go`, add to the QUALITY GATE list (after the existing bullet about ephemeral context):

```
- one-time tasks or completed actions (e.g., "remove the --data-dir flag," "file an issue about X,"
  "clean up the hooks"). If the user said "do X" and X has a completion state, it is a task, not a
  reusable principle. Do not extract it.
- common knowledge any competent developer already knows (e.g., "test both branches of a boolean,"
  "handle errors," "use descriptive names"). If the principle would appear in an introductory
  course or tutorial, the model already knows it — skip it.
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestSystemPrompt_RejectsTasksAndCommonKnowledge -v`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: All passing (minus pre-existing hooks failures)

- [ ] **Step 6: Commit**

```
git add internal/extract/extract.go internal/extract/extract_test.go
```
Message: `feat: add task-vs-principle and common-knowledge filters to extraction prompt (#379)`

---

### Task 2: Add generalization guidance

**Files:**
- Modify: `internal/extract/extract.go` (add GENERALIZE section after EXTRACT section)
- Test: `internal/extract/extract_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSystemPrompt_GeneralizationGuidance(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	prompt := systemPrompt()
	g.Expect(prompt).To(gomega.ContainSubstring("GENERALIZE"))
	g.Expect(prompt).To(gomega.ContainSubstring("most transferable level"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Expected: FAIL

- [ ] **Step 3: Add GENERALIZE section**

Add after the EXTRACT section (before TIER CLASSIFICATION):

```
GENERALIZE — before storing, restate each learning at its most transferable level:
- Strip project-specific details (file names, variable names, tool names) unless they ARE the point.
- Ask: "What is the underlying principle that makes this correct?" State that, not the specific instance.
- Example: "persist surfacing queries in irrelevant_queries field" → "capture diagnostic context at
  the point of observation for later analysis, not after the fact."
- If the generalized form is identical to an existing well-known principle, score generalizability
  lower or reject entirely.
```

- [ ] **Step 4: Run test to verify it passes**

Expected: PASS

- [ ] **Step 5: Commit**

```
git add internal/extract/extract.go internal/extract/extract_test.go
```
Message: `feat: add generalization guidance to extraction prompt (#380)`

---

### Task 3: Add context-specific keyword guidance

**Files:**
- Modify: `internal/extract/extract.go` (add keyword guidance before JSON example)
- Test: `internal/extract/extract_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSystemPrompt_KeywordGuidance(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	prompt := systemPrompt()
	g.Expect(prompt).To(gomega.ContainSubstring("SITUATION where this principle is needed"))
	g.Expect(prompt).To(gomega.ContainSubstring("activity-level terms"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Expected: FAIL

- [ ] **Step 3: Add keyword guidance**

Add a paragraph before the JSON example in systemPrompt():

```
KEYWORD QUALITY — keywords should match the SITUATION where this principle is needed, not just
the subject area. Bad: "git log", "boolean", "testing", "UI" — domain terms that match too broadly.
Good: "post-migration verification", "parallel-agent id collision", "algorithm-exposed controls" —
activity-level terms describing when someone would need this memory. Ask: "What would someone be
doing when they need this memory?" Use those activity-level terms.
```

- [ ] **Step 4: Run test to verify it passes**

Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: All passing

- [ ] **Step 6: Commit**

```
git add internal/extract/extract.go internal/extract/extract_test.go
```
Message: `feat: add context-specific keyword guidance to extraction prompt (#381)`

---

### Task 4: Close issues and final verification

- [ ] **Step 1: Run full test suite**

Run: `targ check-full`
Expected: All passing

- [ ] **Step 2: Close issues**

```bash
gh issue close 379 --comment "Fixed in extraction prompt: added task-vs-principle filter"
gh issue close 380 --comment "Fixed in extraction prompt: added generalization guidance"
gh issue close 381 --comment "Fixed in extraction prompt: added keyword quality guidance"
```
