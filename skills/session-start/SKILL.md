---
name: session-start
description: |
  This skill is triggered automatically by the engram SessionStart hook.
  Use when you see "[engram]" system messages containing pending maintenance
  signals, graduation candidates, or memory management recommendations.
  Also use when the user asks about "memory management", "engram signals",
  "graduation", "memory maintenance", or "pending recommendations".
---

# Engram Session Start: Memory Management

When engram surfaces memory management data at session start, you MUST present it to the user before proceeding with any other task. Do not skip this step.

## What Gets Surfaced

The session start hook produces three categories of signals:

1. **Maintenance signals** — memories that need attention:
   - **Noise removal**: Rarely surfaced AND low effectiveness. Recommend deletion.
   - **Hidden gem broadening**: High effectiveness but rarely surfaced. Recommend broadening keywords so they fire more often.

2. **Graduation candidates** — memories ready for promotion to a higher tier:
   - **Skill promotion**: Memory used effectively across contexts, ready to become a skill.
   - **CLAUDE.md promotion**: Universal principle ready for always-loaded tier.
   - **Rules promotion**: File-scoped rule ready for `.claude/rules/`.

3. **Pending signals** — other maintenance actions queued by the signal pipeline.

## How to Present

Group and summarize. Do NOT dump raw signal data. Follow this format:

### For noise removal candidates:
- Group by theme (e.g., "stale project-specific memories", "duplicates of CLAUDE.md rules", "completed work items")
- State count per group and give 2-3 examples
- Recommend bulk action: "Remove all N noise candidates?"

### For hidden gems:
- List each with its current keywords and a suggestion for broader keywords
- Note any duplicates that should be consolidated first

### For graduation candidates:
- Identify duplicates (same principle, different wording) — recommend consolidating before promoting
- Group by destination (skill, CLAUDE.md, rules)
- For each, state the principle and recommended destination

## After Presenting

Wait for the user's decision. Do not act on signals without explicit approval. Valid user responses:
- "Remove all noise" / "Keep X, remove the rest"
- "Promote these" / "Skip promotions for now"
- "Broaden keywords on the hidden gems"
- "Let's deal with this later" (acknowledge and move on)

## Executing Decisions

- **Remove**: `engram maintain remove --id <memory-id> --data-dir ~/.claude/engram/data`
- **Broaden keywords**: Edit the memory's `.toml` file to add keywords
- **Graduate accept**: `engram graduate accept --id <id> --data-dir ~/.claude/engram/data`
- **Graduate dismiss**: `engram graduate dismiss --id <id> --data-dir ~/.claude/engram/data`
