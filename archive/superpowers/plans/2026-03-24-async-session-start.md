# Async SessionStart Hook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the SessionStart hook non-blocking by splitting it into sync (static messages) and async (maintain + build), surfacing results at the next tool-use hook.

**Architecture:** New sync hook emits static context. Existing session-start.sh becomes async, writes maintain results to a pending file. Pre/PostToolUse hooks consume the file via shell variables that merge with their normal output.

**Tech Stack:** Bash, jq, Claude Code hooks API

**Spec:** `docs/superpowers/specs/2026-03-24-async-session-start-design.md`

---

### File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `hooks/session-start-sync.sh` | Create | Fast sync hook — static `/recall` reminder + mid-turn note |
| `hooks/session-start.sh` | Rewrite | Async hook — build, maintain, write pending file |
| `hooks/pre-tool-use.sh` | Modify | Add pending file consumption + merge with surfacing |
| `hooks/post-tool-use.sh` | Modify | Add pending file consumption + merge with advisory/surfacing |
| `hooks/hooks.json` | Modify | Two SessionStart entries (sync + async) |
| `internal/hooks/hooks_test.go` | Modify | New structural tests + update broken existing tests |

---

### Task 1: Create session-start-sync.sh

**Files:**
- Create: `hooks/session-start-sync.sh`
- Modify: `internal/hooks/hooks_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/hooks/hooks_test.go`:

```go
// TestT370_SessionStartSyncEmitsStaticContext verifies session-start-sync.sh
// emits the /recall reminder and mid-turn note without any build or maintain calls (#370).
func TestT370_SessionStartSyncEmitsStaticContext(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "session-start-sync.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// Must emit static context.
	g.Expect(script).To(ContainSubstring("/recall"))
	g.Expect(script).To(ContainSubstring("Mid-turn user messages"))
	g.Expect(script).To(ContainSubstring("systemMessage"))
	g.Expect(script).To(ContainSubstring("additionalContext"))
	g.Expect(script).To(ContainSubstring("set -euo pipefail"))

	// Must NOT contain slow operations.
	g.Expect(script).NotTo(ContainSubstring("go build"))
	g.Expect(script).NotTo(ContainSubstring("engram maintain"))
	g.Expect(script).NotTo(ContainSubstring("NEEDS_BUILD"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestT370_SessionStartSyncEmitsStaticContext -v`
Expected: FAIL — file does not exist

- [ ] **Step 3: Create session-start-sync.sh**

```bash
#!/usr/bin/env bash
set -euo pipefail

# Fast sync SessionStart hook — emits static context only (#370).
# Slow work (build, maintain) runs in the async session-start.sh hook.

SYSTEM_MSG="[engram] Say /recall to load context from previous sessions, or /recall <query> to search session history."
ADDITIONAL_CTX="[engram] Mid-turn user messages (delivered via system-reminder) bypass engram hooks. If you receive a mid-turn correction or instruction, capture it by running: ~/.claude/engram/bin/engram correct --message '<the user message>'"

jq -n \
    --arg sys "$SYSTEM_MSG" \
    --arg add "$ADDITIONAL_CTX" \
    '{systemMessage: $sys, additionalContext: $add}'
```

Make executable: `chmod +x hooks/session-start-sync.sh`

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestT370_SessionStartSyncEmitsStaticContext -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add hooks/session-start-sync.sh internal/hooks/hooks_test.go
git commit -m "feat(hooks): add sync session-start hook with static context (#370)"
```

---

### Task 2: Refactor session-start.sh to async (file output)

**Files:**
- Modify: `hooks/session-start.sh`
- Modify: `internal/hooks/hooks_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/hooks/hooks_test.go`:

```go
// TestT370_SessionStartAsyncWritesPendingFile verifies session-start.sh writes
// to pending-maintenance.json instead of stdout, uses atomic rename, and deletes
// stale files (#370).
func TestT370_SessionStartAsyncWritesPendingFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "session-start.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// Must write to pending file, not stdout.
	g.Expect(script).To(ContainSubstring("pending-maintenance.json"))
	// Must use atomic write (temp + mv).
	g.Expect(script).To(ContainSubstring(".tmp"))
	g.Expect(script).To(ContainSubstring("mv "))
	// Must delete stale pending file at start.
	g.Expect(script).To(ContainSubstring("rm -f"))
	// Must use atomic build (temp + mv).
	g.Expect(script).To(ContainSubstring("ENGRAM_BIN.tmp"))
	// Must still run maintain.
	g.Expect(script).To(ContainSubstring("engram maintain"), "async hook must run maintain")
	// Must NOT emit the old stdout JSON assembly.
	g.Expect(script).NotTo(ContainSubstring("{systemMessage: $sys"))
	// Must NOT contain /recall or mid-turn note (moved to sync hook).
	g.Expect(script).NotTo(ContainSubstring("Mid-turn user messages"))
	// Must reference #370.
	g.Expect(script).To(ContainSubstring("#370"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestT370_SessionStartAsyncWritesPendingFile -v`
Expected: FAIL — script still has old stdout output

- [ ] **Step 3: Replace session-start.sh**

Full replacement script:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Async SessionStart hook — build, maintain, write pending file (#370).
# Static context (recall reminder, mid-turn note) is in session-start-sync.sh.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
PENDING_FILE="${ENGRAM_HOME}/pending-maintenance.json"

# Delete stale pending file from a previous session (#370)
rm -f "$PENDING_FILE"

# Build if missing or stale (source newer than binary)
# Uses atomic temp+mv to avoid corrupting binary during concurrent reads (#370)
NEEDS_BUILD=false
if [[ ! -x "$ENGRAM_BIN" ]]; then
    NEEDS_BUILD=true
elif [[ -d "$PLUGIN_ROOT" ]]; then
    # Rebuild if any Go source file is newer than the binary
    if find "$PLUGIN_ROOT" -name '*.go' -newer "$ENGRAM_BIN" -print -quit 2>/dev/null | grep -q .; then
        NEEDS_BUILD=true
    fi
fi

if [[ "$NEEDS_BUILD" == "true" ]]; then
    mkdir -p "${ENGRAM_HOME}/bin"
    cd "$PLUGIN_ROOT"
    go build -o "$ENGRAM_BIN.tmp" ./cmd/engram/ 2>/dev/null || { echo "[engram] build failed — is Go installed?" >&2; exit 0; }
    mv "$ENGRAM_BIN.tmp" "$ENGRAM_BIN"
fi

# UC-27: Create global symlink so engram is on PATH (fire-and-forget)
SYMLINK_TARGET="$HOME/.local/bin/engram"
{
    mkdir -p "$HOME/.local/bin"
    if [[ -L "$SYMLINK_TARGET" ]]; then
        # Symlink exists — check if it points to our binary
        if [[ "$(readlink "$SYMLINK_TARGET")" != "$ENGRAM_BIN" ]]; then
            echo "[engram] warning: $SYMLINK_TARGET points to $(readlink "$SYMLINK_TARGET"), not overwriting" >&2
        fi
    elif [[ -e "$SYMLINK_TARGET" ]]; then
        # Regular file or directory — don't clobber
        echo "[engram] warning: $SYMLINK_TARGET exists and is not a symlink, not overwriting" >&2
    else
        ln -s "$ENGRAM_BIN" "$SYMLINK_TARGET"
    fi
} || true

# UC-28: Run maintenance classification (single source of truth for signals)
SIGNAL_OUTPUT=$("$ENGRAM_BIN" maintain 2>/dev/null) || true

# Parse maintain proposals (JSON array with quadrant, action, memory_path, diagnosis)
PROPOSAL_COUNT=0
NOISE_COUNT=0
HIDDEN_GEM_COUNT=0
LEECH_COUNT=0
TRIAGE_DETAILS=""
if [[ -n "$SIGNAL_OUTPUT" ]] && echo "$SIGNAL_OUTPUT" | jq -e 'type == "array" and length > 0' >/dev/null 2>&1; then
    PROPOSAL_COUNT=$(echo "$SIGNAL_OUTPUT" | jq 'length' 2>/dev/null) || PROPOSAL_COUNT=0
    NOISE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.quadrant == "Noise")] | length' 2>/dev/null) || NOISE_COUNT=0
    HIDDEN_GEM_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.quadrant == "Hidden Gem")] | length' 2>/dev/null) || HIDDEN_GEM_COUNT=0
    LEECH_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.quadrant == "Leech")] | length' 2>/dev/null) || LEECH_COUNT=0
    REFINE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "refine_keywords")] | length' 2>/dev/null) || REFINE_COUNT=0
    ESCALATION_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "escalation_escalate" or .action == "escalation_deescalate")] | length' 2>/dev/null) || ESCALATION_COUNT=0

    # Build full details for additionalContext (Claude sees this if user says "triage")
    NOISE_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
        [.[] | select(.quadrant == "Noise")] |
        if length == 0 then empty else
            "## Noise (\(length) memories)\nRarely surfaced AND low effectiveness — candidates for deletion.\n" +
            (to_entries | map(
                "  \(.key + 1). \(.value.memory_path | split("/") | last | rtrimstr(".toml")) — \(.value.diagnosis)"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    HIDDEN_GEM_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
        [.[] | select(.quadrant == "Hidden Gem")] |
        if length == 0 then empty else
            "## Hidden Gems (\(length) memories)\nHigh effectiveness but rarely surfaced — keywords need broadening.\n" +
            (to_entries | map(
                "  \(.key + 1). \(.value.memory_path | split("/") | last | rtrimstr(".toml")) — \(.value.diagnosis)"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    LEECH_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
        [.[] | select(.quadrant == "Leech")] |
        if length == 0 then empty else
            "## Leech (\(length) memories)\nFrequently surfaced but low effectiveness — need rewriting or escalation.\n" +
            (to_entries | map(
                "  \(.key + 1). \(.value.memory_path | split("/") | last | rtrimstr(".toml")) — \(.value.action): \(.value.diagnosis)"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    REFINE_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
        [.[] | select(.action == "refine_keywords")] |
        if length == 0 then empty else
            "## Refine Keywords (\(length) memories)\nSurfacing in wrong contexts — keywords are too generic.\n" +
            (to_entries | map(
                "  \(.key + 1). \(.value.memory_path | split("/") | last | rtrimstr(".toml")) — \(.value.diagnosis)"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    ESCALATION_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
        [.[] | select(.action == "escalation_escalate" or .action == "escalation_deescalate")] |
        if length == 0 then empty else
            "## Escalation (\(length) memories)\nEnforcement level changes recommended.\n" +
            (to_entries | map(
                "  \(.key + 1). \(.value.memory_path | split("/") | last | rtrimstr(".toml")) — \(.value.action): \(.value.diagnosis)"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    for detail in "$NOISE_DETAIL" "$HIDDEN_GEM_DETAIL" "$LEECH_DETAIL" "$REFINE_DETAIL" "$ESCALATION_DETAIL"; do
        if [[ -n "$detail" ]]; then
            TRIAGE_DETAILS="${TRIAGE_DETAILS}
${detail}
"
        fi
    done
fi

# Only write pending file if there are proposals (#370)
if [[ "$PROPOSAL_COUNT" -gt 0 ]]; then
    # Build compact counts line
    COUNTS=""
    [[ "$NOISE_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${NOISE_COUNT} noise"
    [[ "$HIDDEN_GEM_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${HIDDEN_GEM_COUNT} hidden gems"
    [[ "$LEECH_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${LEECH_COUNT} leech"
    [[ "$REFINE_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${REFINE_COUNT} refine keywords"
    [[ "$ESCALATION_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${ESCALATION_COUNT} escalation"
    DIRECTIVE="[engram] Memory triage: ${COUNTS} pending. Say \"triage\" to review, or ignore to proceed."

    TRIAGE_CTX="[engram] Memory triage details (present interactively if user says 'triage'):
${TRIAGE_DETAILS}
Use the engram:memory-triage skill for commands and presentation format.
Present one category at a time. Ask what the user wants to do with each before moving to the next."

    # Write to temp file, then atomic rename (#370)
    jq -n \
        --arg sys "$DIRECTIVE" \
        --arg ctx "$TRIAGE_CTX" \
        '{systemMessage: $sys, additionalContext: $ctx}' > "$PENDING_FILE.tmp"
    mv "$PENDING_FILE.tmp" "$PENDING_FILE"
fi
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestT370_SessionStartAsync -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add hooks/session-start.sh internal/hooks/hooks_test.go
git commit -m "refactor(hooks): make session-start.sh async with file output (#370)"
```

---

### Task 3: Add pending file check to pre-tool-use.sh

**Files:**
- Modify: `hooks/pre-tool-use.sh`
- Modify: `internal/hooks/hooks_test.go`

The `"strings"` import is needed starting from this task. Add it to the import block in `hooks_test.go`.

- [ ] **Step 1: Write the failing test**

Add `"strings"` to the import block in `internal/hooks/hooks_test.go`.

Add test:

```go
// TestT370_PreToolUsePendingCheck verifies pre-tool-use.sh checks for
// pending-maintenance.json after the engram filter but before the Bash-only
// exit (#370).
func TestT370_PreToolUsePendingCheck(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "pre-tool-use.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// Must check for pending file.
	g.Expect(script).To(ContainSubstring("pending-maintenance.json"))
	g.Expect(script).To(ContainSubstring("PENDING_SYS"))
	g.Expect(script).To(ContainSubstring("PENDING_CTX"))
	// Must use atomic consumption (mv).
	g.Expect(script).To(ContainSubstring("mv "))
	// Must reference #370.
	g.Expect(script).To(ContainSubstring("#370"))

	// Verify ordering: pending check must appear AFTER engram filter
	// but BEFORE the Bash-only exit.
	engramFilterIdx := strings.Index(script, "#352")
	pendingCheckIdx := strings.Index(script, "pending-maintenance.json")
	bashOnlyIdx := strings.Index(script, `"$TOOL_NAME" != "Bash"`)

	g.Expect(engramFilterIdx).To(BeNumerically(">", -1))
	g.Expect(pendingCheckIdx).To(BeNumerically(">", -1))
	g.Expect(bashOnlyIdx).To(BeNumerically(">", -1))
	g.Expect(pendingCheckIdx).To(BeNumerically(">", engramFilterIdx),
		"pending check must come after engram filter")
	g.Expect(pendingCheckIdx).To(BeNumerically("<", bashOnlyIdx),
		"pending check must come before Bash-only exit")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestT370_PreToolUsePendingCheck -v`
Expected: FAIL — no pending-maintenance.json reference in script

- [ ] **Step 3: Rewrite pre-tool-use.sh**

Full replacement for `hooks/pre-tool-use.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"

# Build if missing or stale (source newer than binary)
NEEDS_BUILD=false
if [[ ! -x "$ENGRAM_BIN" ]]; then
    NEEDS_BUILD=true
elif [[ -d "$PLUGIN_ROOT" ]]; then
    # Rebuild if any Go source file is newer than the binary
    if find "$PLUGIN_ROOT" -name '*.go' -newer "$ENGRAM_BIN" -print -quit 2>/dev/null | grep -q .; then
        NEEDS_BUILD=true
    fi
fi

if [[ "$NEEDS_BUILD" == "true" ]]; then
    mkdir -p "${ENGRAM_HOME}/bin"
    cd "$PLUGIN_ROOT"
    go build -o "$ENGRAM_BIN" ./cmd/engram/ 2>/dev/null || { echo "[engram] build failed — is Go installed?" >&2; exit 0; }
fi

# Read tool name and input from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // empty')"
TOOL_INPUT="$(echo "$STDIN_JSON" | jq -c '.tool_input // {}')"

# Don't surface memories for any engram CLI calls (#352, #369)
if [[ "$TOOL_NAME" == "Bash" ]]; then
    BASH_CMD="$(echo "$STDIN_JSON" | jq -r '.tool_input.command // empty')"
    # Normalize ~/... to $HOME/... so both path forms match (#369)
    BASH_CMD_NORMALIZED="${BASH_CMD//\~\//$HOME/}"
    if [[ "$BASH_CMD_NORMALIZED" == *"$ENGRAM_BIN"* ]]; then
        exit 0
    fi
fi

# Consume pending maintenance results from async SessionStart (#370)
PENDING_SYS=""
PENDING_CTX=""
PENDING_FILE="$ENGRAM_HOME/pending-maintenance.json"
PENDING_TMP="$PENDING_FILE.consuming.$$"
if [[ -f "$PENDING_FILE" ]] && mv "$PENDING_FILE" "$PENDING_TMP" 2>/dev/null; then
    PENDING_SYS="$(jq -r '.systemMessage // empty' "$PENDING_TMP")"
    PENDING_CTX="$(jq -r '.additionalContext // empty' "$PENDING_TMP")"
    rm -f "$PENDING_TMP"
fi

# Only surface memories for Bash tool calls — non-Bash tools produce near-random BM25 matches
if [[ "$TOOL_NAME" != "Bash" ]]; then
    # Emit pending content if available before exiting (#370)
    if [[ -n "$PENDING_SYS" || -n "$PENDING_CTX" ]]; then
        jq -n \
            --arg sys "$PENDING_SYS" \
            --arg ctx "$PENDING_CTX" \
            '{
                systemMessage: $sys,
                continue: true,
                suppressOutput: false,
                hookSpecificOutput: {
                    hookEventName: "PreToolUse",
                    permissionDecision: "allow",
                    permissionDecisionReason: "",
                    additionalContext: $ctx
                }
            }'
    fi
    exit 0
fi

# UC-2: Surface relevant memories before tool use
if [[ -n "$TOOL_NAME" ]]; then
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --format json) || true
    if [[ -n "$SURFACE_OUTPUT" ]]; then
        # Merge pending maintenance context with surfacing output (#370)
        SURFACE_SYS="$(echo "$SURFACE_OUTPUT" | jq -r '.summary // empty')"
        SURFACE_CTX="$(echo "$SURFACE_OUTPUT" | jq -r '.context // empty')"
        FINAL_SYS="${PENDING_SYS:+$PENDING_SYS
}$SURFACE_SYS"
        FINAL_CTX="${PENDING_CTX:+$PENDING_CTX
}$SURFACE_CTX"
        jq -n \
            --arg sys "$FINAL_SYS" \
            --arg ctx "$FINAL_CTX" \
            '{
                systemMessage: $sys,
                continue: true,
                suppressOutput: false,
                hookSpecificOutput: {
                    hookEventName: "PreToolUse",
                    permissionDecision: "allow",
                    permissionDecisionReason: "",
                    additionalContext: $ctx
                }
            }'
        exit 0
    fi
fi

# No surfacing output — emit pending content standalone if available (#370)
if [[ -n "$PENDING_SYS" || -n "$PENDING_CTX" ]]; then
    jq -n \
        --arg sys "$PENDING_SYS" \
        --arg ctx "$PENDING_CTX" \
        '{
            systemMessage: $sys,
            continue: true,
            suppressOutput: false,
            hookSpecificOutput: {
                hookEventName: "PreToolUse",
                permissionDecision: "allow",
                permissionDecisionReason: "",
                additionalContext: $ctx
            }
        }'
fi
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestT370_PreToolUsePendingCheck -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add hooks/pre-tool-use.sh internal/hooks/hooks_test.go
git commit -m "feat(hooks): add pending maintenance check to pre-tool-use (#370)"
```

---

### Task 4: Add pending file check to post-tool-use.sh

**Files:**
- Modify: `hooks/post-tool-use.sh`
- Modify: `internal/hooks/hooks_test.go`

- [ ] **Step 1: Write the failing test**

```go
// TestT370_PostToolUsePendingCheck verifies post-tool-use.sh checks for
// pending-maintenance.json after the engram filter but before all early-exit
// paths (Write/Edit advisory, Bash-only exit) (#370).
func TestT370_PostToolUsePendingCheck(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "hooks", "post-tool-use.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	// Must check for pending file.
	g.Expect(script).To(ContainSubstring("pending-maintenance.json"))
	g.Expect(script).To(ContainSubstring("PENDING_SYS"))
	g.Expect(script).To(ContainSubstring("PENDING_CTX"))
	// Must use atomic consumption (mv).
	g.Expect(script).To(ContainSubstring("mv "))
	// Must reference #370.
	g.Expect(script).To(ContainSubstring("#370"))

	// Verify ordering: pending check before Write/Edit advisory and Bash-only exit.
	engramFilterIdx := strings.Index(script, "#352")
	pendingCheckIdx := strings.Index(script, "pending-maintenance.json")
	advisoryIdx := strings.Index(script, "Write/Edit")
	bashOnlyIdx := strings.Index(script, `"$TOOL_NAME" != "Bash"`)

	g.Expect(pendingCheckIdx).To(BeNumerically(">", engramFilterIdx),
		"pending check must come after engram filter")
	g.Expect(pendingCheckIdx).To(BeNumerically("<", advisoryIdx),
		"pending check must come before Write/Edit advisory")
	g.Expect(pendingCheckIdx).To(BeNumerically("<", bashOnlyIdx),
		"pending check must come before Bash-only exit")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestT370_PostToolUsePendingCheck -v`
Expected: FAIL — no pending-maintenance.json reference in script

- [ ] **Step 3: Rewrite post-tool-use.sh**

Full replacement for `hooks/post-tool-use.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Read tool details from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // empty')"
TOOL_INPUT="$(echo "$STDIN_JSON" | jq -c '.tool_input // {}')"
TOOL_RESPONSE="$(echo "$STDIN_JSON" | jq -r '.tool_response // empty')"
FILE_PATH="$(echo "$STDIN_JSON" | jq -r '.tool_input.file_path // empty')"

ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"

# Don't surface memories for any engram CLI calls (#352, #369)
if [[ "$TOOL_NAME" == "Bash" ]]; then
    BASH_CMD="$(echo "$STDIN_JSON" | jq -r '.tool_input.command // empty')"
    # Normalize ~/... to $HOME/... so both path forms match (#369)
    BASH_CMD_NORMALIZED="${BASH_CMD//\~\//$HOME/}"
    if [[ "$BASH_CMD_NORMALIZED" == *"$ENGRAM_BIN"* ]]; then
        exit 0
    fi
fi

# Consume pending maintenance results from async SessionStart (#370)
PENDING_SYS=""
PENDING_CTX=""
PENDING_FILE="$ENGRAM_HOME/pending-maintenance.json"
PENDING_TMP="$PENDING_FILE.consuming.$$"
if [[ -f "$PENDING_FILE" ]] && mv "$PENDING_FILE" "$PENDING_TMP" 2>/dev/null; then
    PENDING_SYS="$(jq -r '.systemMessage // empty' "$PENDING_TMP")"
    PENDING_CTX="$(jq -r '.additionalContext // empty' "$PENDING_TMP")"
    rm -f "$PENDING_TMP"
fi

# Skill/command file advisory for Write/Edit
if [[ ("$TOOL_NAME" == "Write" || "$TOOL_NAME" == "Edit") && \
      ("$FILE_PATH" == */skills/* || "$FILE_PATH" == */.claude/commands/*) ]]; then
    ADVISORY_CTX="You just edited a skill/command file — did you pressure-test the changes? Verify it still triggers correctly and handles edge cases."
    MERGED_SYS="$PENDING_SYS"
    MERGED_CTX="${PENDING_CTX:+$PENDING_CTX
}$ADVISORY_CTX"
    jq -n \
        --arg sys "$MERGED_SYS" \
        --arg ctx "$MERGED_CTX" \
        '{
            continue: true,
            suppressOutput: false,
            systemMessage: (if $sys == "" then null else $sys end),
            hookSpecificOutput: {
                hookEventName: "PostToolUse",
                additionalContext: $ctx
            }
        }'
    exit 0
fi

# Only surface memories for Bash tool calls
if [[ "$TOOL_NAME" != "Bash" ]]; then
    # Emit pending content if available before exiting (#370)
    if [[ -n "$PENDING_SYS" || -n "$PENDING_CTX" ]]; then
        jq -n \
            --arg sys "$PENDING_SYS" \
            --arg ctx "$PENDING_CTX" \
            '{
                systemMessage: $sys,
                continue: true,
                suppressOutput: false,
                hookSpecificOutput: {
                    hookEventName: "PostToolUse",
                    additionalContext: $ctx
                }
            }'
    fi
    exit 0
fi

# Surface memories relevant to this tool call and its output
if [[ -x "$ENGRAM_BIN" ]]; then
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --tool-output "$TOOL_RESPONSE" \
        --format json 2>/dev/null) || SURFACE_OUTPUT=""
    if [[ -n "$SURFACE_OUTPUT" ]]; then
        # Merge pending maintenance context with surfacing output (#370)
        SURFACE_SYS="$(echo "$SURFACE_OUTPUT" | jq -r '.summary // empty')"
        SURFACE_CTX="$(echo "$SURFACE_OUTPUT" | jq -r '.context // empty')"
        FINAL_SYS="${PENDING_SYS:+$PENDING_SYS
}$SURFACE_SYS"
        FINAL_CTX="${PENDING_CTX:+$PENDING_CTX
}$SURFACE_CTX"
        jq -n \
            --arg sys "$FINAL_SYS" \
            --arg ctx "$FINAL_CTX" \
            '{
                systemMessage: $sys,
                continue: true,
                suppressOutput: false,
                hookSpecificOutput: {
                    hookEventName: "PostToolUse",
                    additionalContext: $ctx
                }
            }'
        exit 0
    fi
fi

# No surfacing output — emit pending content standalone if available (#370)
if [[ -n "$PENDING_SYS" || -n "$PENDING_CTX" ]]; then
    jq -n \
        --arg sys "$PENDING_SYS" \
        --arg ctx "$PENDING_CTX" \
        '{
            systemMessage: $sys,
            continue: true,
            suppressOutput: false,
            hookSpecificOutput: {
                hookEventName: "PostToolUse",
                additionalContext: $ctx
            }
        }'
fi
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestT370_PostToolUsePendingCheck -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add hooks/post-tool-use.sh internal/hooks/hooks_test.go
git commit -m "feat(hooks): add pending maintenance check to post-tool-use (#370)"
```

---

### Task 5: Update hooks.json and fix broken tests

**Files:**
- Modify: `hooks/hooks.json`
- Modify: `internal/hooks/hooks_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/hooks/hooks_test.go`:

```go
// TestT370_HooksJSONSessionStartSyncAsync verifies hooks.json has two
// SessionStart entries: sync (session-start-sync.sh) and async
// (session-start.sh with async: true) (#370).
func TestT370_HooksJSONSessionStartSyncAsync(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	hooksPath := filepath.Join(root, "hooks", "hooks.json")

	hooksData, err := os.ReadFile(hooksPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	type hookEntry struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
		Async   *bool  `json:"async"`
	}

	type hookGroup struct {
		Hooks []hookEntry `json:"hooks"`
	}

	type hooksFile struct {
		Hooks map[string][]hookGroup `json:"hooks"`
	}

	var parsed hooksFile

	parseErr := json.Unmarshal(hooksData, &parsed)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	entries := parsed.Hooks["SessionStart"]
	g.Expect(entries).To(HaveLen(2), "expected two SessionStart hook groups (sync + async)")

	if len(entries) < 2 {
		return
	}

	// First entry: sync (session-start-sync.sh, no async field).
	g.Expect(entries[0].Hooks).To(HaveLen(1))

	if len(entries[0].Hooks) > 0 {
		syncHook := entries[0].Hooks[0]
		g.Expect(syncHook.Command).To(ContainSubstring("session-start-sync.sh"))
		g.Expect(syncHook.Async).To(BeNil(), "sync hook must not have async field")
	}

	// Second entry: async (session-start.sh, async: true).
	g.Expect(entries[1].Hooks).To(HaveLen(1))

	if len(entries[1].Hooks) > 0 {
		asyncHook := entries[1].Hooks[0]
		g.Expect(asyncHook.Command).To(ContainSubstring("session-start.sh"))
		g.Expect(asyncHook.Command).NotTo(ContainSubstring("session-start-sync.sh"))
		g.Expect(asyncHook.Async).NotTo(BeNil(), "async hook must have async field")
		g.Expect(*asyncHook.Async).To(BeTrue(), "async hook must be async: true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestT370_HooksJSONSessionStartSyncAsync -v`
Expected: FAIL — hooks.json still has one SessionStart entry

- [ ] **Step 3: Update hooks.json**

Replace the SessionStart section with:

```json
"SessionStart": [
  {
    "hooks": [
      {
        "type": "command",
        "command": "${CLAUDE_PLUGIN_ROOT}/hooks/session-start-sync.sh",
        "timeout": 5
      }
    ]
  },
  {
    "hooks": [
      {
        "type": "command",
        "command": "${CLAUDE_PLUGIN_ROOT}/hooks/session-start.sh",
        "timeout": 120,
        "async": true
      }
    ]
  }
],
```

- [ ] **Step 4: Update TestT99_SessionStartCreationInSystemMessage**

The `/recall` and systemMessage/additionalContext output moved to sync hook. Update to point at `session-start-sync.sh`:

```go
func TestT99_SessionStartCreationInSystemMessage(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)
	// Sync hook now emits the systemMessage/additionalContext (#370).
	scriptPath := filepath.Join(root, "hooks", "session-start-sync.sh")

	content, err := os.ReadFile(scriptPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	script := string(content)

	g.Expect(script).To(ContainSubstring("{systemMessage: $sys"))
	g.Expect(script).To(ContainSubstring("additionalContext:"))
	g.Expect(script).To(ContainSubstring("/recall"))
}
```

- [ ] **Step 5: Update TestT43_SessionStartHookSurfaces**

This test checked session-start.sh for `/recall` — now in sync hook. Update to validate the async script's new behavior:

```go
func TestT43_SessionStartHookSurfaces(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	root := repoRoot(t)

	// Async session-start.sh still has build logic and maintain (#370).
	asyncPath := filepath.Join(root, "hooks", "session-start.sh")
	asyncContent, err := os.ReadFile(asyncPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	asyncScript := string(asyncContent)

	g.Expect(asyncScript).To(ContainSubstring("bin/engram"))
	g.Expect(asyncScript).To(ContainSubstring("CLAUDE_PLUGIN_ROOT"))
	g.Expect(asyncScript).To(ContainSubstring("set -euo pipefail"))
	g.Expect(asyncScript).To(ContainSubstring("maintain"))
}
```

- [ ] **Step 6: Run full test suite**

Run: `targ check-full`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add hooks/hooks.json internal/hooks/hooks_test.go
git commit -m "feat(hooks): split SessionStart into sync+async in hooks.json (#370)"
```

---

### Task 6: Manual verification and cleanup

- [ ] **Step 1: Run full check**

Run: `targ check-full`
Expected: ALL PASS

- [ ] **Step 2: Manual test — start a new session**

Update and reinstall the plugin (`/plugin`, `/reload-plugins`), then start a new Claude Code session. Verify:
1. Session starts immediately (no ~7.5s blocking)
2. `/recall` reminder appears in the system message
3. On first tool call, triage results appear (if maintain found proposals)

- [ ] **Step 3: Commit any final fixes**

If manual testing reveals issues, fix and commit.
