#!/bin/bash
# install-skills.sh - Install/update projctl skills to ~/.claude/skills
# TASK-30: Create install-skills.sh script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
SKILLS_SRC="$REPO_DIR/skills"
SKILLS_DST="${HOME}/.claude/skills"
BACKUP_DIR="${HOME}/.claude/skills.backup.$(date +%Y%m%d%H%M%S)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

info() { echo -e "${GREEN}INFO${NC}: $1"; }
warn() { echo -e "${YELLOW}WARN${NC}: $1"; }
error() { echo -e "${RED}ERROR${NC}: $1"; exit 1; }

# Old skills to remove (being replaced by new architecture)
OLD_SKILLS=(
    "alignment-check"    # → alignment-producer + alignment-qa
    "architect-audit"    # → merged into arch-qa
    "architect-infer"    # → arch-infer-producer
    "architect-interview" # → arch-interview-producer
    "design-audit"       # → merged into design-qa
    "design-infer"       # → design-infer-producer
    "design-interview"   # → design-interview-producer
    "meta-audit"         # → merged into retro-producer
    "negotiate"          # → merged into QA escalate-phase capability
    "pm-audit"           # → merged into pm-qa
    "pm-infer"           # → pm-infer-producer
    "pm-interview"       # → pm-interview-producer
    "task-audit"         # → merged into tdd-qa
    "task-breakdown"     # → breakdown-producer
    "tdd-green"          # → tdd-green-producer
    "tdd-red"            # → tdd-red-producer
    "tdd-refactor"       # → tdd-refactor-producer
    "test-mapper"        # → obsolete (no TEST-NNN IDs)
)

# New skills to install
NEW_SKILLS=(
    # Shared resources
    "shared"

    # Core orchestrator
    "project"
    "commit"

    # PM phase
    "pm-interview-producer"
    "pm-infer-producer"
    "pm-qa"

    # Design phase
    "design-interview-producer"
    "design-infer-producer"
    "design-qa"

    # Architecture phase
    "arch-interview-producer"
    "arch-infer-producer"
    "arch-qa"

    # Breakdown phase
    "breakdown-producer"
    "breakdown-qa"

    # Documentation phase
    "doc-producer"
    "doc-qa"

    # TDD skills
    "tdd-producer"
    "tdd-red-producer"
    "tdd-red-infer-producer"
    "tdd-green-producer"
    "tdd-refactor-producer"
    "tdd-qa"
    "tdd-red-qa"
    "tdd-green-qa"
    "tdd-refactor-qa"

    # Support skills
    "alignment-producer"
    "alignment-qa"
    "retro-producer"
    "retro-qa"
    "summary-producer"
    "summary-qa"

    # Orchestration support
    "context-explorer"
    "context-qa"
    "parallel-looper"
    "consistency-checker"
    "intake-evaluator"
    "next-steps"
)

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Install or update projctl skills to ~/.claude/skills

Options:
    --dry-run     Show what would be done without making changes
    --force       Skip confirmation prompt
    --rollback    Restore from most recent backup
    --help        Show this help message

Backup Location:
    ~/.claude/skills.backup.<timestamp>

Rollback:
    $0 --rollback

    Or manually:
    rm -rf ~/.claude/skills
    mv ~/.claude/skills.backup.<timestamp> ~/.claude/skills
EOF
}

dry_run=false
force=false
rollback=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run) dry_run=true; shift ;;
        --force) force=true; shift ;;
        --rollback) rollback=true; shift ;;
        --help) usage; exit 0 ;;
        *) error "Unknown option: $1" ;;
    esac
done

# Rollback mode
if $rollback; then
    latest_backup=$(ls -dt "${HOME}/.claude/skills.backup."* 2>/dev/null | head -1)
    if [[ -z "$latest_backup" ]]; then
        error "No backup found to restore"
    fi

    info "Restoring from: $latest_backup"
    if $dry_run; then
        echo "Would run: rm -rf $SKILLS_DST"
        echo "Would run: mv $latest_backup $SKILLS_DST"
    else
        rm -rf "$SKILLS_DST"
        mv "$latest_backup" "$SKILLS_DST"
        info "Rollback complete"
    fi
    exit 0
fi

# Check source directory
if [[ ! -d "$SKILLS_SRC" ]]; then
    error "Skills source directory not found: $SKILLS_SRC"
fi

# Create destination if needed
if [[ ! -d "$SKILLS_DST" ]]; then
    if $dry_run; then
        echo "Would create: $SKILLS_DST"
    else
        mkdir -p "$SKILLS_DST"
        info "Created: $SKILLS_DST"
    fi
fi

# Show plan
echo "=== Skill Installation Plan ==="
echo ""
echo "Source: $SKILLS_SRC"
echo "Destination: $SKILLS_DST"
echo ""

echo "Skills to REMOVE (old architecture):"
for skill in "${OLD_SKILLS[@]}"; do
    if [[ -L "$SKILLS_DST/$skill" ]] || [[ -d "$SKILLS_DST/$skill" ]]; then
        echo "  - $skill"
    fi
done
echo ""

echo "Skills to INSTALL (new architecture):"
for skill in "${NEW_SKILLS[@]}"; do
    if [[ -d "$SKILLS_SRC/$skill" ]]; then
        echo "  + $skill"
    else
        warn "  ? $skill (not found in source)"
    fi
done
echo ""

# Confirmation
if ! $force && ! $dry_run; then
    read -p "Proceed with installation? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        info "Installation cancelled"
        exit 0
    fi
fi

# Backup existing
if [[ -d "$SKILLS_DST" ]] && [[ "$(ls -A "$SKILLS_DST")" ]]; then
    if $dry_run; then
        echo "Would backup: $SKILLS_DST -> $BACKUP_DIR"
    else
        cp -r "$SKILLS_DST" "$BACKUP_DIR"
        info "Backed up existing skills to: $BACKUP_DIR"
    fi
fi

# Remove old skills
info "Removing old skills..."
for skill in "${OLD_SKILLS[@]}"; do
    target="$SKILLS_DST/$skill"
    if [[ -L "$target" ]] || [[ -d "$target" ]]; then
        if $dry_run; then
            echo "Would remove: $target"
        else
            rm -rf "$target"
        fi
    fi
done

# Install new skills
info "Installing new skills..."
failed=()
for skill in "${NEW_SKILLS[@]}"; do
    src="$SKILLS_SRC/$skill"
    dst="$SKILLS_DST/$skill"

    if [[ ! -d "$src" ]]; then
        warn "Skipping missing skill: $skill"
        failed+=("$skill")
        continue
    fi

    if $dry_run; then
        echo "Would symlink: $dst -> $src"
    else
        # Remove existing if present
        [[ -L "$dst" ]] && rm "$dst"
        [[ -d "$dst" ]] && rm -rf "$dst"

        # Create symlink
        ln -s "$src" "$dst"
    fi
done

# Report
echo ""
if $dry_run; then
    info "Dry run complete. No changes made."
else
    info "Installation complete!"
    echo ""
    echo "Installed: $((${#NEW_SKILLS[@]} - ${#failed[@]})) skills"
    if [[ ${#failed[@]} -gt 0 ]]; then
        warn "Failed: ${#failed[@]} skills (${failed[*]})"
    fi
    echo ""
    echo "Backup: $BACKUP_DIR"
    echo ""
    echo "To rollback: $0 --rollback"
fi
