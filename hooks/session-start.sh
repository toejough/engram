#!/usr/bin/env bash
set -euo pipefail

# SessionStart hook — fast sync output + background maintain (#370).
# Emits static context (recall reminder, mid-turn note) synchronously,
# then forks build+maintain into the background to write a pending file
# consumed by PreToolUse/PostToolUse.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
PENDING_FILE="${ENGRAM_HOME}/pending-maintenance.json"
DEBUG_LOG="${ENGRAM_HOME}/async-hook-debug.log"

# --- Sync portion: emit static context immediately ---
SYSTEM_MSG="[engram] Say /recall to load context from previous sessions, or /recall <query> to search session history."
ADDITIONAL_CTX="[engram] Mid-turn user messages (delivered via system-reminder) bypass engram hooks. If you receive a mid-turn correction or instruction, capture it by running: ~/.claude/engram/bin/engram correct --message '<the user message>'"

jq -n \
    --arg sys "$SYSTEM_MSG" \
    --arg add "$ADDITIONAL_CTX" \
    '{systemMessage: $sys, additionalContext: $add}'

# --- Async portion: fork build+maintain into background ---
(
    # Redirect all output to debug log (temporary — remove after confirming fix)
    exec >> "$DEBUG_LOG" 2>&1
    echo "=== $(date -Iseconds) session-start.sh background fork ==="
    echo "CLAUDE_PLUGIN_ROOT=${CLAUDE_PLUGIN_ROOT:-<unset>}"
    echo "PLUGIN_ROOT=${PLUGIN_ROOT}"
    echo "ENGRAM_BIN=${ENGRAM_BIN}"
    echo "PWD=$(pwd)"

    # Delete stale pending file from a previous session
    rm -f "$PENDING_FILE"

    # Build if missing or stale (source newer than binary)
    # Uses atomic temp+mv to avoid corrupting binary during concurrent reads
    NEEDS_BUILD=false
    if [[ ! -x "$ENGRAM_BIN" ]]; then
        NEEDS_BUILD=true
    elif [[ -d "$PLUGIN_ROOT" ]]; then
        if find "$PLUGIN_ROOT" -name '*.go' -newer "$ENGRAM_BIN" -print -quit 2>/dev/null | grep -q .; then
            NEEDS_BUILD=true
        fi
    fi

    if [[ "$NEEDS_BUILD" == "true" ]]; then
        echo "Building engram binary..."
        mkdir -p "${ENGRAM_HOME}/bin"
        cd "$PLUGIN_ROOT"
        go build -o "$ENGRAM_BIN.tmp" ./cmd/engram/ || { echo "BUILD FAILED"; exit 0; }
        mv "$ENGRAM_BIN.tmp" "$ENGRAM_BIN"
        echo "Build complete"
    fi

    # UC-27: Create global symlink so engram is on PATH (fire-and-forget)
    SYMLINK_TARGET="$HOME/.local/bin/engram"
    {
        mkdir -p "$HOME/.local/bin"
        if [[ -L "$SYMLINK_TARGET" ]]; then
            if [[ "$(readlink "$SYMLINK_TARGET")" != "$ENGRAM_BIN" ]]; then
                echo "[engram] warning: $SYMLINK_TARGET points to $(readlink "$SYMLINK_TARGET"), not overwriting" >&2
            fi
        elif [[ -e "$SYMLINK_TARGET" ]]; then
            echo "[engram] warning: $SYMLINK_TARGET exists and is not a symlink, not overwriting" >&2
        else
            ln -s "$ENGRAM_BIN" "$SYMLINK_TARGET"
        fi
    } || true

    # UC-28: Run engram maintain — single source of truth for proposals
    echo "Running maintain..."
    SIGNAL_OUTPUT=$("$ENGRAM_BIN" maintain --data-dir "${ENGRAM_HOME}/data") || true
    echo "Maintain output length: ${#SIGNAL_OUTPUT}"

    # Parse maintain proposals (JSON array with id, action, target, field, value, rationale)
    PROPOSAL_COUNT=0
    TRIAGE_DETAILS=""
    if [[ -n "$SIGNAL_OUTPUT" ]] && echo "$SIGNAL_OUTPUT" | jq -e 'type == "array" and length > 0' >/dev/null 2>&1; then
        PROPOSAL_COUNT=$(echo "$SIGNAL_OUTPUT" | jq 'length' 2>/dev/null) || PROPOSAL_COUNT=0

        DELETE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "delete")] | length' 2>/dev/null) || DELETE_COUNT=0
        UPDATE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "update" and .target != "policy.toml")] | length' 2>/dev/null) || UPDATE_COUNT=0
        MERGE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "merge")] | length' 2>/dev/null) || MERGE_COUNT=0
        RECOMMEND_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "recommend")] | length' 2>/dev/null) || RECOMMEND_COUNT=0
        ADAPT_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "update" and .target == "policy.toml")] | length' 2>/dev/null) || ADAPT_COUNT=0

        # Build full details for additionalContext (Claude sees this if user says "triage")
        DELETE_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "delete")] |
            if length == 0 then empty else
                "## Delete (\(length) memories)\nFailing both effectiveness and irrelevance thresholds.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.target | split("/") | last | rtrimstr(".toml")) — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        UPDATE_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "update" and .target != "policy.toml")] |
            if length == 0 then empty else
                "## Rewrite (\(length) memories)\nFields need narrowing or clarification.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.target | split("/") | last | rtrimstr(".toml")) [\(.value.field)] — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        MERGE_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "merge")] |
            if length == 0 then empty else
                "## Consolidate (\(length) groups)\nSimilar memories that could be merged.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.target | split("/") | last | rtrimstr(".toml")) — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        RECOMMEND_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "recommend")] |
            if length == 0 then empty else
                "## Escalation (\(length) memories)\nConsider converting to rules/hooks/CLAUDE.md.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.target | split("/") | last | rtrimstr(".toml")) — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        ADAPT_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "update" and .target == "policy.toml")] |
            if length == 0 then empty else
                "## Parameter Tuning (\(length) proposals)\nSystem parameter adjustments suggested.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.field): \(.value.value) — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        for detail in "$DELETE_DETAIL" "$UPDATE_DETAIL" "$MERGE_DETAIL" "$RECOMMEND_DETAIL" "$ADAPT_DETAIL"; do
            if [[ -n "$detail" ]]; then
                TRIAGE_DETAILS="${TRIAGE_DETAILS}
${detail}
"
            fi
        done
    fi

    # Only write pending file if there are proposals
    if [[ "$PROPOSAL_COUNT" -gt 0 ]]; then
        # Build compact counts line
        COUNTS=""
        [[ "${DELETE_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${DELETE_COUNT} delete"
        [[ "${UPDATE_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${UPDATE_COUNT} rewrite"
        [[ "${MERGE_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${MERGE_COUNT} consolidate"
        [[ "${RECOMMEND_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${RECOMMEND_COUNT} escalation"
        [[ "${ADAPT_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${ADAPT_COUNT} parameter tuning"
        DIRECTIVE="[engram] Memory triage: ${COUNTS} pending. Say \"triage\" to review, or ignore to proceed."

        TRIAGE_CTX="[engram] Memory triage details (present interactively if user says 'triage'):
${TRIAGE_DETAILS}
Use the engram:memory-triage skill for commands and presentation format.
Present one category at a time. Ask what the user wants to do with each before moving to the next."

        # Write to temp file, then atomic rename
        jq -n \
            --arg directive "$DIRECTIVE" \
            --arg triage "$TRIAGE_CTX" \
            '{systemMessage: $directive, additionalContext: $triage}' > "$PENDING_FILE.tmp"
        mv "$PENDING_FILE.tmp" "$PENDING_FILE"
        echo "Wrote pending file: $(wc -c < "$PENDING_FILE") bytes"
    fi

    echo "=== done ==="
) & disown

exit 0
