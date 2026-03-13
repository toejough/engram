---
name: memory-triage
description: |
  Use when the user asks about "memory management", "engram signals",
  "graduation", "memory maintenance", "pending recommendations", "triage
  memories", or "clean up memories". Provides formatting rules and commands
  for reviewing and acting on engram maintenance signals and graduation
  candidates.
---

# Engram Memory Triage

Review and act on engram maintenance signals and graduation candidates. This skill covers the presentation format and available commands for memory triage.

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
