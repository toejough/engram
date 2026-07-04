---
name: write-memory
description: >
  Internal shared write-memory procedure — invoked explicitly by recall and learn skills when
  writing vault notes or QA pairs. Do not invoke independently.
---

# Write Memory — Mechanical write procedures for recall and learn


This skill is invoked by other skills at their specific write steps. It contains ONLY the
mechanical flag content for vault write operations. Judgment about WHEN to write (verdicts,
gates, which-case branching) stays in the invoking skill.

## Feedback (correction notes)

When the invoking skill directs you to write a feedback note:

```bash
engram learn feedback --slug <kebab-slug> --position top \
  --source "<descriptive source string, e.g. 'session <date>, context: <one-line what-was-happening>'>" \
  --situation "<retrieval-shaped phrase: when does this apply>" \
  --behavior "<what was done>" \
  --impact "<why it was wrong/costly>" \
  --action "<what to do instead>"
```

If the invoking skill determined this note CORRECTS, narrows, or refutes an existing surfaced
note, also pass:
```
--supersedes "<basename>|<type>|<claim>"
```
Types: `updates|narrows|refutes`. The binary maintains the inverse automatically.

Vocab tags are **automatically** assigned by the binary on every write — do not hand-author them.

## Fact (explicit save-requests and new principles)

When the invoking skill directs you to write a fact note:

```bash
engram learn fact --slug <kebab-slug> --position top \
  --source "<descriptive source string, e.g. 'session <date>, context: <one-line what-was-happening>'>" \
  --situation "<retrieval-shaped phrase: when does this apply>" \
  --subject "<the thing>" \
  --predicate "<requires / must use / is>" \
  --object "<the standard or value>"
```

If the invoking skill determined this note CORRECTS, narrows, or refutes an existing surfaced
note, also pass `--supersedes "<basename>|<type>|<claim>"` (types: `updates|narrows|refutes`).
The binary maintains the inverse automatically.

Vocab tags are **automatically** assigned by the binary on every write — do not hand-author them.

### Chunk-source provenance (recall's absent case)

When the invoking skill provides chunk source IDs, pass one flag per chunk on the
`engram learn fact|feedback` call:

```
--chunk-source <source#anchor>
```

Repeatable; provenance only. Pass one flag per chunk ID the invoking skill provided; if the
invoking skill provided none, omit the flag.

## QA pair capture

**D2 bar (caller's responsibility to check):** ≥1 `[[full-basename]]` wikilink in the synthesis
body is required before invoking this procedure. If the synthesis body contains no wikilinks,
skip — the invoking skill handles this gate.

When the invoking skill directs you to write a QA pair:

```bash
engram learn qa \
  --slug "<kebab summary of the question>" \
  --question "<verbatim question that prompted this recall>" \
  --answer "<the synthesis conclusion you just wrote as the note body>" \
  --contributors "<full-basename-1>" \
  --contributors "<full-basename-2>" \
  ... (one --contributors per [[full-basename]] wikilink in the synthesis) \
  --certainty "<high|medium|low — match the certainty label on the synthesis note>" \
  --source "<context, e.g. 'recall Step 4, session <date>'>"
```

Contributors come ONLY from `[[full-basename]]` wikilinks in the written answer — never
free-listed. Do NOT pre-validate whether contributors exist in the vault; extract the wikilink
content verbatim and pass it to `--contributors`. Validation happens at write time.

## --supersedes syntax reference

Use `--supersedes "<basename>|<type>|<claim>"` where:
- `<basename>` is the full note basename (no .md extension)
- `<type>` is one of: `updates`, `narrows`, `refutes`
- `<claim>` is a brief description of what this note corrects/updates/refutes

The binary maintains the inverse link automatically. Do not hand-author wikilinks for structural linking.
