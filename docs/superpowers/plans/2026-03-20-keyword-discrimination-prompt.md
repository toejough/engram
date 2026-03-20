# Keyword Discrimination Prompt Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the LLM classification prompt to generate discriminating keywords that won't match unrelated domains, reducing cross-context false-positive surfacing.

**Architecture:** Single string change in `internal/classify/classify.go` `systemPrompt()`. The JSON schema example for `keywords` gets guidance text. One test verifies the prompt contains the new guidance.

**Tech Stack:** Go, gomega, targ

---

## File Structure

| File | Change | Responsibility |
|------|--------|---------------|
| `internal/classify/classify.go` | Modify | Update system prompt keywords guidance |
| `internal/classify/classify_test.go` | Modify | Test prompt contains discrimination guidance |

---

### Task 1: Update keyword generation prompt

**Files:**
- Modify: `internal/classify/classify.go:354` (keywords line in system prompt)
- Modify: `internal/classify/classify_test.go`

- [ ] **Step 1: Write failing test — prompt contains keyword discrimination guidance**

Find existing tests for `systemPrompt` in `internal/classify/classify_test.go`. If none exist, add one:

```go
func TestSystemPrompt_ContainsKeywordDiscriminationGuidance(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    prompt := classify.SystemPromptForTest(false)
    g.Expect(prompt).To(ContainSubstring("unique to this specific domain"))
}
```

Note: `systemPrompt` is unexported. Either:
- (A) Export it as `SystemPromptForTest` (test-only export pattern)
- (B) Test it indirectly by checking the prompt is passed to the LLM mock
- (C) Add an exported `SystemPrompt()` function since the prompt content is the public contract

Check which pattern the codebase uses. If there's already a test that asserts on the prompt content, add the assertion there.

- [ ] **Step 2: Run test — verify it fails**

Expected: FAIL — current prompt doesn't mention discrimination.

- [ ] **Step 3: Update the system prompt**

In `internal/classify/classify.go`, in the `systemPrompt()` function, change the keywords line in the JSON schema from:

```
  "keywords": ["searchable", "keywords"],
```

to:

```
  "keywords": ["specific", "discriminating", "terms"],
```

And add guidance text above the JSON schema (after the tier descriptions, before "Return ONLY a JSON object"):

```
Keyword selection rules:
- Choose keywords UNIQUE to this specific domain or tool — terms that would NOT match
  unrelated projects or contexts (e.g., "nozzle-temperature" not "settings",
  "stl-mesh" not "file", "targ-check-full" not "check").
- Avoid generic programming terms: test, error, build, function, check, run, fix, add,
  update, config, setup, debug, log, data, file, code, project, tool, command.
- Include the specific tool, library, or domain name (e.g., "gomega", "targ", "toml").
- 3-7 keywords per memory. Fewer specific keywords > many generic ones.
```

- [ ] **Step 4: Run tests — verify pass**

Run: `targ test`

- [ ] **Step 5: Run targ check-full**

Run: `targ check-full`
Expected: all checks pass

- [ ] **Step 6: Commit**

```bash
git commit -m "feat(classify): improve keyword generation prompt for discrimination (#344)"
```

- [ ] **Step 7: Rebuild binary**

```bash
go build -o ~/.claude/engram/bin/engram ./cmd/engram/
```
