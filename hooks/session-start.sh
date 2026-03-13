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

# Pre-format signal summaries so the directive contains actual content, not formatting rules.
# Without this, Claude sees raw JSON + meta-instructions and tends to echo the instructions
# instead of parsing the data and presenting results.
NOISE_SUMMARY=""
HIDDEN_GEM_SUMMARY=""
SKILL_PROMO_SUMMARY=""
if [[ "$SIGNAL_COUNT" -gt 0 ]]; then
    NOISE_SUMMARY=$(echo "$SIGNAL_CTX" | jq -r '
        [.signals[] | select(.signal == "noise_removal")] |
        if length == 0 then empty else
            "## Noise Removal (\(length) memories)\nRarely surfaced AND low effectiveness — candidates for deletion.\n" +
            (to_entries | map(
                "  \(.key + 1). **\(.value.title)** (id: \(.value.source_id | split("/") | last | rtrimstr(".toml")))"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    HIDDEN_GEM_SUMMARY=$(echo "$SIGNAL_CTX" | jq -r '
        [.signals[] | select(.signal == "hidden_gem_broadening")] |
        if length == 0 then empty else
            "## Hidden Gems (\(length) memories)\nHigh effectiveness but rarely surfaced — keywords need broadening.\n" +
            (to_entries | map(
                "  \(.key + 1). **\(.value.title)** (id: \(.value.source_id | split("/") | last | rtrimstr(".toml")))\n     Keywords: \(.value.keywords)"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true

    SKILL_PROMO_SUMMARY=$(echo "$SIGNAL_CTX" | jq -r '
        [.signals[] | select(.signal == "memory_to_skill")] |
        if length == 0 then empty else
            "## Skill Promotion (\(length) memories)\nUsed effectively and frequently — candidates for promotion to skill.\n" +
            (to_entries | map(
                "  \(.key + 1). **\(.value.title)** (id: \(.value.source_id | split("/") | last | rtrimstr(".toml")))\n     Principle: \(.value.principle // "n/a")"
            ) | join("\n"))
        end
    ' 2>/dev/null) || true
fi

GRAD_SUMMARY=""
if [[ "$GRAD_COUNT" -gt 0 ]]; then
    GRAD_SUMMARY=$(echo "$GRAD_CTX" | jq -r '
        [.entries[]] |
        group_by(.recommendation) |
        map(
            "### \(.[0].recommendation) (\(length) candidates)\n" +
            (to_entries | map(
                "  \(.key + 1). **\(.value.title // .value.memory_path | split("/") | last | rtrimstr(".toml"))** (id: \(.value.id))\n     Recommendation: \(.value.recommendation)"
            ) | join("\n"))
        ) | "## Graduation Candidates (\([.[] | length] | add))\n" + join("\n\n")
    ' 2>/dev/null) || true
fi

# Build directive with pre-formatted content
DIRECTIVE=""
if [[ "$SIGNAL_COUNT" -gt 0 ]] || [[ "$GRAD_COUNT" -gt 0 ]]; then
    DIRECTIVE="[engram] ACTION REQUIRED: Present the following memory management summary to the user and ask what they want to do BEFORE working on anything else.
"
    if [[ -n "$NOISE_SUMMARY" ]]; then
        DIRECTIVE="${DIRECTIVE}
${NOISE_SUMMARY}
"
    fi
    if [[ -n "$HIDDEN_GEM_SUMMARY" ]]; then
        DIRECTIVE="${DIRECTIVE}
${HIDDEN_GEM_SUMMARY}
"
    fi
    if [[ -n "$SKILL_PROMO_SUMMARY" ]]; then
        DIRECTIVE="${DIRECTIVE}
${SKILL_PROMO_SUMMARY}
"
    fi
    if [[ -n "$GRAD_SUMMARY" ]]; then
        DIRECTIVE="${DIRECTIVE}
${GRAD_SUMMARY}
"
    fi
    DIRECTIVE="${DIRECTIVE}
Ask the user what to do. Options: remove noise (all or selectively), broaden hidden gem keywords, promote to skill, or deal with later.
Commands: remove = engram maintain remove --id <id> --data-dir ~/.claude/engram/data | graduate accept/dismiss = engram graduate accept/dismiss --id <id> --data-dir ~/.claude/engram/data"
fi

# Assemble output
ADDITIONAL_CTX="$MIDTURN_NOTE"
if [[ -n "$SESSION_CONTEXT" ]]; then
    ADDITIONAL_CTX="${ADDITIONAL_CTX}
[engram] Previous session context:
${SESSION_CONTEXT}"
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
