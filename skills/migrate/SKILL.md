---
name: migrate
description: Use when upgrading engram memories from the pre-`cfd5fb5` (2026-04-17) flat format to the current split feedback/facts layout. Triggers on "/migrate", "migrate engram memories", "upgrade engram memories", or when you detect legacy-shaped files (flat TOML with `confidence`, `surfaced_count`, `followed_count`, `keywords`, `concepts`, `principle`, `anti_pattern`, or `project_scoped` fields) under `~/.local/share/engram/`.
---

# Migrate legacy engram memories

Walks through classifying and rewriting each legacy memory into the current shape. Requires human judgement for feedback-vs-fact classification and for spotting hindsight-biased situations. There is intentionally no CLI for this — bad classifications and hindsight-biased situations poison recall for every future session.

"Legacy" here means memory files written by engram before commit `cfd5fb5` (2026-04-17). The current shape is described in the main README's [Memory format](../../README.md#memory-format) section.

## When to use

- User says `/migrate`, "migrate memories", or asks about upgrading engram memories.
- You find files in `~/.local/share/engram/memories.v1-backup/` or `~/.local/share/engram/memories/` that have any of: `confidence`, `surfaced_count`, `followed_count`, `not_followed_count`, `irrelevant_count`, `keywords`, `concepts`, `principle`, `anti_pattern`, `project_scoped`, or a top-level `title`.
- New feedback/fact files are being created but legacy files still exist alongside.

## When NOT to use

- Files already have `schema_version = 2` and a `[content]` sub-table — they are in the current format. Skip.
- User is authoring a *new* memory — use `/remember` or `/learn`, not this.
- You haven't read the file. Read first, then classify. Do not batch-migrate on filename alone.

## Core principle

**Current-format situations must describe the task an agent is about to do, not the problem they hit.** Many legacy situations are hindsight-biased ("after I forgot to X", "when we discovered Y broke"). Rewrite every legacy situation to match the query a future agent would run, or the memory will never surface at the right time.

## Flow

### Step 1: Locate legacy data

Check both likely locations:

```bash
ls ~/.local/share/engram/memories.v1-backup/ 2>/dev/null
ls ~/.local/share/engram/memories/ 2>/dev/null
```

A file is legacy if it has any of these at the top level: `confidence`, `surfaced_count`, `followed_count`, `not_followed_count`, `irrelevant_count`, `keywords`, `concepts`, `principle`, `anti_pattern`, `project_scoped`, `title`.

A file is already in the current format if it has `schema_version = 2` and a `[content]` sub-table. Skip those.

### Step 2: Read each file fully before classifying

Do not guess from the filename. Read the full file. Identify which legacy fields are present.

### Step 3: Classify — feedback or fact

**Feedback** if the memory describes a behavior to apply or avoid:
- Has `behavior` + `impact` + `action` (direct match — always feedback)
- Or has `principle` + `anti_pattern` (behavioral rule — feedback)
- Or the `content` / `title` reads as "do X when Y" or "don't do X"

**Fact** if the memory describes a declarative truth about the project or environment:
- Has a clear subject–predicate–object shape even if not labeled
- Reads as "X is Y" or "project Z uses tool W"
- Describes configuration, conventions, API contracts, tool choices

**If unclear, STOP and ask the user.** Do not batch-classify ambiguous cases. A fact misfiled as feedback (or vice versa) will not surface for the right query.

### Step 4: Rewrite the situation

Read the legacy `situation` field. Ask: "does this describe the task the agent would be starting, or the problem the agent already hit?"

| Hindsight-biased (legacy) | Task-shaped (current) |
|---------------------------|------------------------|
| "After I forgot to use DI" | "writing new code in `internal/` with I/O dependencies" |
| "When we discovered tests were flaky" | "writing or reviewing async tests in the engram repo" |
| "After breaking the build with raw go test" | "running tests or checks in the engram repo" |

If the legacy situation is already task-shaped, keep it. If hindsight-biased, rewrite before continuing.

### Step 5: Build the new file

**Feedback template:**
```toml
schema_version = 2
type = "feedback"
source = "<see source rules below>"
situation = "<task-shaped situation from Step 4>"
created_at = "<preserve legacy created_at verbatim>"
updated_at = "<now, RFC3339 UTC, e.g. 2026-04-17T14:30:00Z>"

[content]
behavior = "<legacy behavior, OR derive from legacy anti_pattern>"
impact   = "<legacy impact, OR infer consequence from legacy content>"
action   = "<legacy action, OR derive from legacy principle>"
```

**Fact template:**
```toml
schema_version = 2
type = "fact"
source = "<see source rules below>"
situation = "<task-shaped situation from Step 4>"
created_at = "<preserve legacy created_at verbatim>"
updated_at = "<now, RFC3339 UTC>"

[content]
subject   = "<what the memory is about>"
predicate = "<relationship verb: uses, requires, has, defaults to, ...>"
object    = "<the declarative content>"
```

### Step 6: Pick `source`

The legacy format did not track source. Apply these rules in order:

1. If the legacy content contains a **direct quotation** of the user (e.g. `"User said: 'always use targ'"`, `"Remember that X"`) → `source = "human"`. Reported speech ("User requested X", "user wanted Y") does NOT count — that is agent paraphrase.
2. If legacy `confidence = "A"` (the "explicit instruction" tier) → `source = "human"`.
3. Otherwise → `source = "agent"`.

When in doubt, default to `"agent"`. Users correct misclassified `"human"` memories far less often than they correct misclassified `"agent"` ones.

### Step 7: Drop these legacy fields unconditionally

These have no equivalent in the current format. Do not carry them forward, do not try to encode them elsewhere:

- `surfaced_count`, `followed_count`, `not_followed_count`, `irrelevant_count` — outcome tracking was removed
- `project_scoped` — scope is now implicit in situation text
- `confidence` — replaced by `source` (see Step 6)
- `keywords`, `concepts` — retrieval now uses situation text + Haiku filtering
- `title` — content moved into `[content]` subject/predicate/object or behavior/impact/action

**Do not ask the user whether to keep these.** They are removed by design.

### Step 8: Write to the correct destination

- feedback → `~/.local/share/engram/memory/feedback/<slug>.toml`
- fact → `~/.local/share/engram/memory/facts/<slug>.toml`

Reuse the legacy filename if it is already kebab-case (lowercase letters, digits, hyphens only; ≤60 chars). Otherwise regenerate from the new situation: lowercase, non-alphanumerics → `-`, collapse repeats, trim to 60 chars.

### Step 9: Verify each migrated file

After writing, confirm it parses:

```bash
engram show --name <slug>
```

If it errors, fix the file before moving on. Do not continue to the next file with a broken one behind you.

If the `engram` binary is not on `$PATH`, build it first (`cd ~/repos/personal/engram && targ build`) and re-run. Do not skip verification.

### Step 10: Archive the legacy source

Only after all files are migrated and `engram show` succeeds on each:

```bash
mv ~/.local/share/engram/memories.v1-backup \
   ~/.local/share/engram/memories.legacy-migrated-$(date +%Y%m%d)
```

(Or the equivalent rename for whichever legacy directory you migrated from.)

Do not delete. Keep the dated archive until at least one real session has used `/recall` and `/prepare` against the migrated data and you have confirmed the memories surface as expected.

## Paths — always fully qualified

All paths in this skill are absolute. Never interpret `memory/feedback/` or `memory/facts/` as relative to the current directory — they are under `~/.local/share/engram/`.

## Quick reference

| Situation | Decision |
|-----------|----------|
| Legacy has `behavior` + `impact` + `action` | feedback |
| Legacy has `principle` + `anti_pattern` | feedback |
| Legacy reads "X is Y" / "project uses Z" | fact |
| Legacy ambiguous | STOP, ask user |
| Legacy situation is hindsight-biased | rewrite before writing the new file |
| Legacy `confidence = "A"` | `source = "human"` |
| Legacy confidence missing or B/C | `source = "agent"` |
| Legacy has `surfaced_count` etc. | drop, no current equivalent |
| Legacy has `project_scoped` | drop, no current equivalent |
| Legacy has `keywords` / `concepts` | drop, no current equivalent |

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Batch-classifying by filename | Read every file fully before deciding type |
| Preserving legacy `surfaced_count` / `followed_count` in a new field | Drop them — no current equivalent |
| Preserving legacy `project_scoped` as a tag | Drop — scope is implicit in situation text |
| Copying legacy `situation` verbatim when hindsight-biased | Rewrite to task shape (Step 4 table) |
| Defaulting `source = "human"` when legacy didn't track it | Default to `"agent"` unless rule 1 or 2 in Step 6 fires |
| Writing to `memory/feedback/` (bare path) | Use `~/.local/share/engram/memory/feedback/` — fully qualified |
| Skipping `engram show` verification | Always verify each file parses before moving on |
| Deleting legacy source after migration | Rename to `memories.legacy-migrated-<date>`, keep until verified in real sessions |

## Red flags — STOP

- Classifying a file without reading its full contents
- Assuming a file is already in the current format because the filename looks new
- Carrying forward any legacy outcome counter or `project_scoped` flag
- Rewriting situations from memory instead of reading the legacy text
- Moving files before `engram show` confirms they parse
- Deleting legacy sources before real-session verification
- Treating ambiguous feedback-vs-fact as "probably feedback"

All of these mean: pause, re-read the source file, or ask the user.
