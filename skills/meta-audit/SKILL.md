---
name: meta-audit
description: Analyze correction patterns and propose skill/CLAUDE.md improvements
context: fork
model: opus
skills: ownership-rules
user-invocable: true
---

# Meta Audit Skill

Analyze patterns in manual corrections to propose structural improvements to skills and CLAUDE.md.

**Model:** Opus — this requires meta-reasoning about patterns across multiple corrections and proposing structural changes. The most cognitively demanding skill.

## Purpose

Review corrections.jsonl for repeated patterns. When the same type of correction appears 2+ times, propose specific CLAUDE.md additions and skill edits so the correction never needs to happen again.

The goal is self-improvement: the system should learn from operator corrections and encode that learning permanently.

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Path to corrections.jsonl
- Project directory
- Current CLAUDE.md content (or path)
- List of skill files to consider for edits
- Trigger reason (threshold reached or project completion)

## Process

1. **Read corrections log** - Parse corrections.jsonl entries
2. **Identify patterns** - Group corrections by:
   - Type (same kind of mistake)
   - Skill (which skill was corrected)
   - Domain (requirements, design, architecture, implementation, testing)
   - Frequency (how many times)
3. **Filter for actionable patterns** - Only propose for patterns appearing 2+ times
4. **Draft CLAUDE.md additions** - Write exact text to add, specifying:
   - Which section it belongs in
   - The exact wording
   - Why this prevents the pattern
5. **Draft skill edits** - For each affected skill, write:
   - Which skill file to edit
   - What to add/change
   - Why this prevents the pattern
6. **Validate proposals** - Ensure:
   - Proposals don't contradict existing CLAUDE.md rules
   - Proposals fit the existing taxonomy (don't invent new sections)
   - Proposals are specific, not vague ("add error handling" is bad; "check for nil before accessing .Name field in state transitions" is good)

## Rules

1. **Only propose for 2+ occurrences** - A single correction could be a fluke
2. **Be specific** - Exact text, exact location, exact rationale
3. **Fit existing structure** - Don't invent new CLAUDE.md sections; use existing taxonomy
4. **Consolidate** - Check if content belongs in an existing rule before creating new ones
5. **Test against history** - Would this proposal have prevented all N occurrences?
6. **Don't over-engineer** - Simple rules that prevent real patterns, not elaborate frameworks

## What NOT to Propose

- Rules that only apply to one specific project (unless it's a universal pattern)
- Changes that contradict existing CLAUDE.md rules
- Vague guidelines ("be more careful") instead of specific checks
- Rules for things that happened only once
- Changes to tool behavior (only change prompts/rules)

## Structured Result

```
Status: success | blocked
Summary: Analyzed N corrections. Found M patterns. Proposing X CLAUDE.md additions, Y skill edits.
Patterns found:
  - pattern: <description>
    frequency: N occurrences
    affected_skill: <skill name>
    domain: <requirements|design|architecture|implementation|testing>
    examples:
      - <correction 1 summary>
      - <correction 2 summary>
Proposals:
  claude_md_additions:
    - section: "Lessons Learned > Code & Debugging"
      text: |
        **<Title>**: <Rule text>
      rationale: <why this prevents the pattern>
      prevents: [correction IDs]
  skill_edits:
    - skill: <skill name>
      file: <path>
      change: <what to add/modify>
      rationale: <why>
      prevents: [correction IDs]
No proposals needed: [patterns that don't meet threshold or already have rules]
```

## Result Format

See [shared/RESULT.md](../shared/RESULT.md) for the complete schema.

```toml
[status]
success = true

[outputs]
files_modified = []

[[decisions]]
context = "Process choice"
choice = "Follow established convention"
reason = "Consistency with existing patterns"
alternatives = []

[[learnings]]
content = "Captured from execution"
```
