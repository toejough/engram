---
name: write-memory
description: >
  Executes a vault write handed off by another skill (recall, learn): composes the engram
  command from the provided fields, runs it, verifies the result, and reports the written
  note path. Requires a handoff — do not fire on your own judgment that something is worth
  remembering.
---

# Write Memory — execute a handed-off vault write

You were invoked by a parent skill that already made the judgment (what to write and why).
Your job is the write itself: compose, execute, verify, report. Do not re-litigate the
parent's judgment; do not decide WHETHER to write.

## The handoff contract

The parent provides:

- **kind** — `fact`, `feedback`, or `qa`
- **content fields** — by kind, per the blocks below
- **source** — human-readable provenance string
- optional **chunk-sources** — `<source#anchor>` chunk IDs (provenance)
- optional **tags** — categorical `<family>` or `<family>/<value>` strings (kebab-case;
  fact/feedback only), e.g. `work-kind/rename`, `tier/cheap`, `outcome/pass`
- optional **supersedes** — `<basename>|<type>|<claim>` (types: `updates|narrows|refutes`),
  when the parent determined this write corrects a surfaced note

If a required field is missing, ask for it from the in-session parent context — do not invent
content on the parent's behalf.

## Compose

kind=feedback:

```bash
engram learn feedback --slug <kebab-slug> --position top \
  --source "<source>" \
  --situation "<retrieval-shaped phrase: when does this apply>" \
  --behavior "<what was done>" --impact "<why it was wrong/costly>" --action "<what to do instead>" \
  [--tag <family>/<value> ...]
```

kind=fact:

```bash
engram learn fact --slug <kebab-slug> --position top \
  --source "<source>" \
  --situation "<retrieval-shaped phrase: when does this apply>" \
  --subject "<the thing>" --predicate "<requires / must use / is>" --object "<the standard or value>" \
  [--tag <family>/<value> ...]
```

kind=qa:

```bash
engram learn qa \
  --slug "<kebab summary of the question>" \
  --question "<verbatim question>" \
  --answer "<the answer body, copied — no re-derive>" \
  --contributors "<full-basename>" \
  --certainty "<high|medium|low>" \
  --source "<source>"
```

Append to any kind:

- one `--chunk-source <source#anchor>` per provided chunk ID
- one `--tag <t>` per provided tag (fact/feedback only — `engram learn qa` and `engram amend`
  take no `--tag`; a qa handoff carrying tags → drop them and say so: append the exact line
  `tags dropped: qa takes no tag flags` to whatever you output, even command-only output)
- `--supersedes "<basename>|<type>|<claim>"` if provided (repeatable)
- for qa: one `--contributors <full-basename>` per basename the parent provided

Rules:

- Never mix fact flags (`--subject/--predicate/--object`) with feedback flags
  (`--behavior/--impact/--action`) in one command.
- Never hand-author vocab tags or wikilinks — the binary assigns vocab automatically. Handed-off
  `--tag` categoricals are NOT vocab: pass them through exactly as provided; never invent tags.

## Execute, verify, report

Run the command. On success the CLI prints the written note path(s).

- CLI error → read it, fix exactly the named problem (missing/typo'd flag, bad value), retry.
  Max 2 retries.
- Success → report the printed note path(s) to the parent flow in one line.
- Still failing after retries → report the exact command and the CLI error verbatim. Never
  silently skip a handed-off write.
