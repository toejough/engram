# Hold List NDJSON Output Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change `engram hold list` output from TSV to NDJSON so jq pipelines in the lead skill work correctly.

**Architecture:** Replace the `fmt.Fprintf` TSV loop in `runHoldList` with a `json.NewEncoder` loop that encodes each `HoldRecord` directly. `HoldRecord` already has `json:"hold-id"` (kebab-case) tags matching the lead skill's `.["hold-id"]` jq query. No struct changes needed — `encoding/json` is already imported.

**Tech Stack:** Go standard library (`encoding/json`), gomega for test assertions, `targ` for build/test/check.

---

### Task 1: Write failing test for NDJSON output

**Files:**
- Modify: `internal/cli/cli_test.go` (after `TestRun_HoldList_FiltersCorrectly`, around line 1268)

- [ ] **Step 1: Write the failing test**

Add this test after `TestRun_HoldList_FiltersCorrectly`:

```go
func TestRun_HoldList_OutputsNDJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Acquire a hold with all fields populated.
	acquireErr := cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "lead",
		"--target", "exec-1",
		"--tag", "codesign-1",
	}, io.Discard, io.Discard, nil)
	g.Expect(acquireErr).NotTo(HaveOccurred())
	if acquireErr != nil {
		return
	}

	var stdout bytes.Buffer

	listErr := cli.Run([]string{
		"engram", "hold", "list",
		"--chat-file", chatFile,
	}, &stdout, io.Discard, nil)
	g.Expect(listErr).NotTo(HaveOccurred())

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	g.Expect(lines).To(HaveLen(1))

	// Each line must be valid JSON with a non-empty hold-id.
	var record map[string]any
	g.Expect(json.Unmarshal([]byte(lines[0]), &record)).To(Succeed())
	g.Expect(record["hold-id"]).To(BeAssignableToTypeOf(""))
	g.Expect(record["hold-id"]).NotTo(BeEmpty())
	g.Expect(record["holder"]).To(Equal("lead"))
	g.Expect(record["target"]).To(Equal("exec-1"))
	g.Expect(record["tag"]).To(Equal("codesign-1"))
}
```

- [ ] **Step 2: Ensure `encoding/json` is imported in the test file**

Check `internal/cli/cli_test.go` imports for `"encoding/json"`. If missing, add it. Run:

```bash
head -30 /Users/joe/repos/personal/engram/internal/cli/cli_test.go
```

If `"encoding/json"` is absent, add it to the import block alongside `"bytes"`, `"strings"`, etc.

- [ ] **Step 3: Run test to verify it fails**

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -A5 "FAIL\|TestRun_HoldList_OutputsNDJSON"
```

Expected: test fails with an error like `invalid character '\t' looking for beginning of value` (TSV is not valid JSON) or `unexpected end of JSON input`.

---

### Task 2: Implement NDJSON output in runHoldList

**Files:**
- Modify: `internal/cli/cli.go` lines 874–881 (the TSV output loop inside `runHoldList`)

- [ ] **Step 4: Replace TSV loop with NDJSON encoder**

In `internal/cli/cli.go`, replace the current output loop:

```go
// OLD — remove this:
for _, hold := range filterHolds(chat.ScanActiveHolds(messages), *holder, *target, *tag) {
	_, writeErr := fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
		hold.HoldID, hold.Holder, hold.Target, hold.Condition, hold.Tag,
	)
	if writeErr != nil {
		return fmt.Errorf("hold list: writing output: %w", writeErr)
	}
}
```

With:

```go
// NEW:
enc := json.NewEncoder(stdout)
for _, hold := range filterHolds(chat.ScanActiveHolds(messages), *holder, *target, *tag) {
	if encErr := enc.Encode(hold); encErr != nil {
		return fmt.Errorf("hold list: writing output: %w", encErr)
	}
}
```

Note: `encoding/json` is already imported in `cli.go` (line 8). No import change needed.

- [ ] **Step 5: Run full checks**

```bash
cd /Users/joe/repos/personal/engram && targ check-full
```

Expected: all 7 checks pass. `TestRun_HoldList_OutputsNDJSON` passes. Existing tests `TestRun_HoldList_FilterByTag` and `TestRun_HoldList_FiltersCorrectly` still pass (their `ContainSubstring` assertions match string values inside the JSON output).

- [ ] **Step 6: Verify NDJSON output manually**

```bash
cd /Users/joe/repos/personal/engram && targ build
engram hold acquire --holder lead --target exec-1 --tag codesign-1 2>/dev/null || true
engram hold list | head -3
```

Expected output (one JSON object per line, no trailing whitespace issues):
```
{"hold-id":"<uuid>","holder":"lead","target":"exec-1","acquired-ts":"...","tag":"codesign-1"}
```

Verify jq pipeline works:
```bash
engram hold list | jq -r '.["hold-id"]'
```

Expected: UUID printed, no errors.

- [ ] **Step 7: Commit**

```bash
cd /Users/joe/repos/personal/engram
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
fix(cli): change hold list output from TSV to NDJSON

TSV output silently broke jq pipelines in the lead skill's bulk-release
pattern. HoldRecord already has the correct kebab-case JSON tags, so
encoding it directly produces hold-id, holder, target, acquired-ts,
condition (omitempty), and tag (omitempty) per line.

Fixes #516

AI-Used: [claude]
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- ✅ TSV → NDJSON output change: Task 2 Step 4
- ✅ RED test first (TDD): Task 1 Steps 1–3
- ✅ Existing tests unbroken: Step 5 verifies ContainSubstring assertions still pass
- ✅ jq pipeline verified: Step 6
- ✅ Commit: Step 7

**Placeholder scan:** No TBD/TODO present. All code is complete.

**Type consistency:**
- `HoldRecord` used throughout — no aliasing. `json.NewEncoder` / `enc.Encode` consistent across steps.
- `map[string]any` used in test for flexible JSON field access — correct for `any`-typed values from `json.Unmarshal`.
- `record["hold-id"]` is kebab-case matching `HoldRecord`'s `json:"hold-id"` tag — consistent.
