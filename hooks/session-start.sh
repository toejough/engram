#!/usr/bin/env bash
set -euo pipefail

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
ENGRAM_DATA="${ENGRAM_HOME}/data"

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

# UC-2: Surface relevant memories at session start
SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode session-start --data-dir "$ENGRAM_DATA" --format json) || true

# UC-28: Refresh signal queue (safety net if stop hook didn't complete)
"$ENGRAM_BIN" signal-detect --data-dir "$ENGRAM_DATA" 2>/dev/null || true

# UC-28: Surface pending maintenance/promotion signals
SIGNAL_OUTPUT=$("$ENGRAM_BIN" signal-surface --data-dir "$ENGRAM_DATA" --format json 2>/dev/null) || true

# P6f: Surface pending graduation signals
GRAD_OUTPUT=$("$ENGRAM_BIN" graduate-surface --data-dir "$ENGRAM_DATA" --format json 2>/dev/null) || true

# UC-14: Restore session context (project-specific path)
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
CONTEXT_FILE="${ENGRAM_DATA}/projects/${PROJECT_SLUG}/session-context.md"
SESSION_CONTEXT=""
if [[ -f "$CONTEXT_FILE" ]]; then
    # Extract summary (skip HTML comment on first line)
    SESSION_CONTEXT=$(tail -n +3 "$CONTEXT_FILE")
fi

# Static guidance for mid-turn message capture (issue #54)
MIDTURN_NOTE="[engram] Mid-turn user messages (delivered via system-reminder) bypass engram hooks. If you receive a mid-turn correction or instruction, capture it by running: ~/.claude/engram/bin/engram correct --message '<the user message>' --data-dir ~/.claude/engram/data"

SIGNAL_CTX=""
SIGNAL_COUNT=0
if [[ -n "$SIGNAL_OUTPUT" ]]; then
    SIGNAL_CTX=$(echo "$SIGNAL_OUTPUT" | jq -r '.context // empty' 2>/dev/null) || true
    SIGNAL_COUNT=$(echo "$SIGNAL_CTX" | jq -r '.signals | length' 2>/dev/null) || SIGNAL_COUNT=0
fi

GRAD_CTX=""
GRAD_COUNT=0
if [[ -n "$GRAD_OUTPUT" ]]; then
    GRAD_CTX=$(echo "$GRAD_OUTPUT" | jq -r '.context // empty' 2>/dev/null) || true
    GRAD_COUNT=$(echo "$GRAD_OUTPUT" | jq -r '.entries | length' 2>/dev/null) || GRAD_COUNT=0
fi

# Count signals by type for the short summary line
NOISE_COUNT=0
HIDDEN_GEM_COUNT=0
SKILL_PROMO_COUNT=0
TRIAGE_DETAILS=""
if [[ "$SIGNAL_COUNT" -gt 0 ]]; then
    NOISE_COUNT=$(echo "$SIGNAL_CTX" | jq '[.signals[] | select(.signal == "noise_removal")] | length' 2>/dev/null) || NOISE_COUNT=0
    HIDDEN_GEM_COUNT=$(echo "$SIGNAL_CTX" | jq '[.signals[] | select(.signal == "hidden_gem_broadening")] | length' 2>/dev/null) || HIDDEN_GEM_COUNT=0
    SKILL_PROMO_COUNT=$(echo "$SIGNAL_CTX" | jq '[.signals[] | select(.signal == "memory_to_skill")] | length' 2>/dev/null) || SKILL_PROMO_COUNT=0

    # Build full details for additionalContext (Claude sees this if user says "triage")
    NOISE_DETAIL=$(echo "$SIGNAL_CTX" | jq -r '
        [.signals[] | select(.signal == "noise_removal")] |
        if length == 0 then empty else
            "## Noise Removal (\(length) memories)\nRarely surfaced AND low effectiveness — candidates for deletion.\n" +
            (to_entries | map(
                "  \(.key + 1). **\(.value.title)** (id: \(.value.source_id | split("/") | last | rtrimstr(".toml")))"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    HIDDEN_GEM_DETAIL=$(echo "$SIGNAL_CTX" | jq -r '
        [.signals[] | select(.signal == "hidden_gem_broadening")] |
        if length == 0 then empty else
            "## Hidden Gems (\(length) memories)\nHigh effectiveness but rarely surfaced — keywords need broadening.\n" +
            (to_entries | map(
                "  \(.key + 1). **\(.value.title)** (id: \(.value.source_id | split("/") | last | rtrimstr(".toml")))\n     Keywords: \(.value.keywords)"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    SKILL_PROMO_DETAIL=$(echo "$SIGNAL_CTX" | jq -r '
        [.signals[] | select(.signal == "memory_to_skill")] |
        if length == 0 then empty else
            "## Skill Promotion (\(length) memories)\nUsed effectively and frequently — candidates for promotion to skill.\n" +
            (to_entries | map(
                "  \(.key + 1). **\(.value.title)** (id: \(.value.source_id | split("/") | last | rtrimstr(".toml")))\n     Principle: \(.value.principle // "n/a")"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    for detail in "$NOISE_DETAIL" "$HIDDEN_GEM_DETAIL" "$SKILL_PROMO_DETAIL"; do
        if [[ -n "$detail" ]]; then
            TRIAGE_DETAILS="${TRIAGE_DETAILS}
${detail}
"
        fi
    done
fi

GRAD_DETAIL=""
if [[ "$GRAD_COUNT" -gt 0 ]]; then
    GRAD_DETAIL=$(echo "$GRAD_CTX" | jq -r '
        [.entries[]] |
        group_by(.recommendation) |
        map(
            "### \(.[0].recommendation) (\(length) candidates)\n" +
            (to_entries | map(
                "  \(.key + 1). **\(.value.title // .value.memory_path | split("/") | last | rtrimstr(".toml"))** (id: \(.value.id))\n     Recommendation: \(.value.recommendation)"
            ) | join("\n"))
        ) | "## Graduation Candidates (\([.[] | length] | add))\n" + join("\n\n")
    ' 2>/dev/null) || true
    if [[ -n "$GRAD_DETAIL" ]]; then
        TRIAGE_DETAILS="${TRIAGE_DETAILS}
${GRAD_DETAIL}
"
    fi
fi

# Build short summary for systemMessage (user sees this in terminal)
# Full details go in additionalContext for Claude to reference on request
DIRECTIVE=""
TRIAGE_CTX=""
if [[ "$SIGNAL_COUNT" -gt 0 ]] || [[ "$GRAD_COUNT" -gt 0 ]]; then
    # Build a compact counts line
    COUNTS=""
    [[ "$NOISE_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${NOISE_COUNT} noise"
    [[ "$HIDDEN_GEM_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${HIDDEN_GEM_COUNT} hidden gems"
    [[ "$SKILL_PROMO_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${SKILL_PROMO_COUNT} skill promotions"
    [[ "$GRAD_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${GRAD_COUNT} graduation candidates"

    DIRECTIVE="[engram] Memory triage: ${COUNTS} pending. Say \"triage\" to review, or ignore to proceed."

    # Full details + commands go in additionalContext
    TRIAGE_CTX="[engram] Memory triage details (present interactively if user says 'triage'):
${TRIAGE_DETAILS}
Commands: remove = engram maintain remove --id <id> --data-dir ~/.claude/engram/data | graduate accept/dismiss = engram graduate accept/dismiss --id <id> --data-dir ~/.claude/engram/data
Present one category at a time. Ask what the user wants to do with each before moving to the next."
fi

# Assemble output
ADDITIONAL_CTX="$MIDTURN_NOTE"
if [[ -n "$SESSION_CONTEXT" ]]; then
    ADDITIONAL_CTX="${ADDITIONAL_CTX}
[engram] Previous session context:
${SESSION_CONTEXT}"
fi
if [[ -n "$TRIAGE_CTX" ]]; then
    ADDITIONAL_CTX="${ADDITIONAL_CTX}
${TRIAGE_CTX}"
fi

if [[ -n "$SURFACE_OUTPUT" ]]; then
    SURFACE_MSG=$(echo "$SURFACE_OUTPUT" | jq -r '.summary // empty' 2>/dev/null) || true
    SURFACE_CTX=$(echo "$SURFACE_OUTPUT" | jq -r '.context // empty' 2>/dev/null) || true
    if [[ -n "$DIRECTIVE" ]]; then
        SYSTEM_MSG="${DIRECTIVE}
${SURFACE_MSG}"
    else
        SYSTEM_MSG="$SURFACE_MSG"
    fi
    jq -n \
        --arg sys "$SYSTEM_MSG" \
        --arg add "${SURFACE_CTX}
${ADDITIONAL_CTX}" \
        '{systemMessage: $sys, additionalContext: $add}'
elif [[ -n "$DIRECTIVE" ]]; then
    jq -n \
        --arg sys "$DIRECTIVE" \
        --arg add "$ADDITIONAL_CTX" \
        '{systemMessage: $sys, additionalContext: $add}'
else
    jq -n \
        --arg add "$ADDITIONAL_CTX" \
        '{additionalContext: $add}'
fi
