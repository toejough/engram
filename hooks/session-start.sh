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

# UC-28: Run engram maintain — single source of truth for signals
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
        --arg directive "$DIRECTIVE" \
        --arg triage "$TRIAGE_CTX" \
        '{systemMessage: $directive, additionalContext: $triage}' > "$PENDING_FILE.tmp"
    mv "$PENDING_FILE.tmp" "$PENDING_FILE"
fi
