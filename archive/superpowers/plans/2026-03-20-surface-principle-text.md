# Surface Principle Text Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change memory surfacing output from `slug (surfaced N times, followed M%)` to `slug: principle text` across all surfacing modes.

**Architecture:** Modify `formatMemoryLine` to always include principle, remove `formatEffectivenessAnnotation` from surfacing output, update prompt mode inline formatting to match. All changes in `internal/surface/surface.go` + tests.

**Tech Stack:** Go, gomega, targ

---

### Task 1: Change formatMemoryLine to always include principle

**Files:**
- Modify: `internal/surface/surface.go:1241-1250` (formatMemoryLine)
- Modify: `internal/surface/surface.go:1220-1238` (formatEffectivenessAnnotation — delete or stop calling)
- Test: `internal/surface/surface_test.go`

- [ ] **Step 1: Write failing test — default level includes principle**

```go
// T-338a: Default enforcement level includes principle text.
func TestFormatMemoryLine_DefaultLevel_IncludesPrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	line := surface.FormatMemoryLine("targ-check-full", "Always run targ check-full before declaring done", "", "")
	g.Expect(line).To(Equal("  - targ-check-full: Always run targ check-full before declaring done\n"))
}
```

Note: `formatMemoryLine` is currently unexported. Export it as `FormatMemoryLine` for testability, or test via the rendering functions that call it. Check if there are existing tests that exercise it indirectly. If so, update those instead.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test-for-fail -- ./internal/surface/`
Expected: FAIL — current default format is `"  - slug(annotation)\n"` with no principle.

- [ ] **Step 3: Update formatMemoryLine to always include principle**

Change `formatMemoryLine` in `internal/surface/surface.go`:

```go
func formatMemoryLine(slug, principle, level, annotation string) string {
	switch level {
	case enforcementEmphasizedAdvisory:
		return fmt.Sprintf("  - IMPORTANT: **%s: %s**\n", slug, principle)
	case enforcementReminder:
		return fmt.Sprintf("  - REMINDER: %s: %s\n", slug, principle)
	default:
		return fmt.Sprintf("  - %s: %s\n", slug, principle)
	}
}
```

Key changes:
- Default level now includes `: principle` (was slug + annotation only)
- All levels drop the `annotation` parameter (effectiveness stats removed from output)
- Emphasized level includes principle (was slug only)

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test-for-fail -- ./internal/surface/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/surface/surface.go internal/surface/surface_test.go
git commit -m "feat(surface): formatMemoryLine always includes principle text (#338)"
```

---

### Task 2: Update prompt mode to include principle

**Files:**
- Modify: `internal/surface/surface.go:576-601` (runPrompt output formatting)
- Test: `internal/surface/surface_test.go`

- [ ] **Step 1: Find existing prompt-mode tests that assert the output format**

Search for tests that check for `"Relevant memories"` or the slug-only format in prompt mode output. These will need updating.

Run: `grep -n "Relevant memories\|filenameSlug\|promptMatch.*slug" internal/surface/surface_test.go`

- [ ] **Step 2: Update existing tests to expect `slug: principle` format**

For each test that asserts the prompt output format, change expected output from:
```
  - some-memory-slug (surfaced N times, followed M%)
```
to:
```
  - some-memory-slug: The principle text here
```

- [ ] **Step 3: Update runPrompt context and summary formatting**

In `runPrompt` (lines 576-601), change both the context and summary blocks:

```go
// Context block (lines 576-585):
_, _ = fmt.Fprintf(&buf, "<system-reminder source=\"engram\">\n")
_, _ = fmt.Fprintf(&buf, "[engram] Relevant memories:\n")

for _, match := range matches {
	_, _ = fmt.Fprintf(&buf, "  - %s: %s\n",
		filenameSlug(match.mem.FilePath), match.mem.Principle)
}

_, _ = fmt.Fprintf(&buf, "</system-reminder>\n")

// Summary block (lines 593-601):
_, _ = fmt.Fprintf(&summaryBuf, "[engram] %d relevant memories:\n", len(matches))

for _, match := range matches {
	_, _ = fmt.Fprintf(&summaryBuf, "  - %s: %s\n",
		filenameSlug(match.mem.FilePath), match.mem.Principle)
}
```

Key changes:
- Replace `filenameSlug + annotation` with `filenameSlug: principle`
- Remove `formatEffectivenessAnnotation` calls

- [ ] **Step 4: Run tests**

Run: `targ test-for-fail -- ./internal/surface/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/surface/surface.go internal/surface/surface_test.go
git commit -m "feat(surface): prompt mode includes principle text (#338)"
```

---

### Task 3: Update tool mode — drop annotation from renderToolAdvisories

**Files:**
- Modify: `internal/surface/surface.go:339-382` (renderToolAdvisories)
- Test: `internal/surface/surface_test.go`

- [ ] **Step 1: Find existing tool-mode tests that assert output format**

Run: `grep -n "Tool call advisory\|tool advisory\|formatMemoryLine\|annotation" internal/surface/surface_test.go`

- [ ] **Step 2: Update existing tests to expect `slug: principle` format (no stats)**

Change expected tool output from:
```
  - some-slug (surfaced N times, followed M%)
```
to:
```
  - some-slug: principle text
```

- [ ] **Step 3: Remove annotation from renderToolAdvisories**

In `renderToolAdvisories` (line 362-373), stop computing annotation:

```go
for _, match := range candidates {
	toolMems = append(toolMems, match.mem)
	level := s.enforcementLevelFor(match.mem.FilePath)
	line := formatMemoryLine(
		filenameSlug(match.mem.FilePath),
		match.mem.Principle,
		level,
		"", // annotation removed
	)
	_, _ = fmt.Fprint(&summaryBuf, line)
	_, _ = fmt.Fprint(&contextBuf, line)
}
```

- [ ] **Step 4: Run tests**

Run: `targ test-for-fail -- ./internal/surface/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/surface/surface.go internal/surface/surface_test.go
git commit -m "feat(surface): tool mode drops stats annotation (#338)"
```

---

### Task 4: Delete formatEffectivenessAnnotation if no longer called

**Files:**
- Modify: `internal/surface/surface.go` (delete function if unused)

- [ ] **Step 1: Check if formatEffectivenessAnnotation is still called anywhere**

Run: `grep -n "formatEffectivenessAnnotation" internal/surface/surface.go`

If only the function definition remains (no callers), delete it.

- [ ] **Step 2: Run deadcode check**

Run: `targ deadcode`
Expected: no new dead code

- [ ] **Step 3: Commit if changes made**

```bash
git add internal/surface/surface.go
git commit -m "fix(surface): delete unused formatEffectivenessAnnotation (#338)"
```

---

### Task 5: Update session-start mode if needed

**Files:**
- Modify: `internal/surface/surface.go` (runSessionStart)
- Test: `internal/surface/surface_test.go`

- [ ] **Step 1: Check session-start output format**

Read `runSessionStart` to see if it uses `formatMemoryLine` or has its own formatting. If it already includes principle text (like precompact mode does), skip this task.

- [ ] **Step 2: If needed, update to `slug: principle` format**

Follow the same pattern as Task 2.

- [ ] **Step 3: Run full test suite + check-full**

Run: `targ check-full`
Expected: all checks pass (except check-uncommitted)

- [ ] **Step 4: Commit**

```bash
git add internal/surface/surface.go internal/surface/surface_test.go
git commit -m "feat(surface): session-start mode includes principle text (#338)"
```

---

### Task 6: Update hook script tests

**Files:**
- Modify: `internal/hooks/hooks_test.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Find tests that assert hook output format with stats**

Run: `grep -n "surfaced.*times\|followed.*%" internal/hooks/hooks_test.go internal/cli/cli_test.go`

- [ ] **Step 2: Update assertions to match new format**

Replace any assertions checking for `(surfaced N times, followed M%)` with assertions checking for principle text presence.

- [ ] **Step 3: Run full check-full**

Run: `targ check-full`
Expected: PASS (all 7 real checks)

- [ ] **Step 4: Final commit**

```bash
git add internal/hooks/hooks_test.go internal/cli/cli_test.go
git commit -m "test: update hook tests for principle-text format (#338)"
```
