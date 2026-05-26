# L1 episode fix spec

Date: 2026-05-26. Implementation contract for re-aligning the
`engram learn episode` schema with the original L0/L1/L2 design
from `2026-05-22-tiered-memory-research-log.md`. The currently
shipped episode shape is L2-flavored (LLM-narrated
`summary`/`outcomes`); the design called for L1 (filtered
transcript chunks with boundary annotations).

## Framing

The original tiered design:

- **L0** — raw sources, pointers only (session JSONL files, git
  refs, URLs).
- **L1** — episodic filtered transcripts: noise-removed
  extractions, content-preserving cleanup, chunked at meaningful
  boundaries.
- **L2** — analyzed kinds: facts and feedback, principle-stated
  and retrieval-shaped.

An **episode** is L1 — the chunk of filtered transcript that
captures *what happened* during a discrete segment of work. Facts
and feedback are L2 and link back to the episode they were
extracted from.

The currently shipped `engram learn episode` produces an L2-style
note (LLM narration via `--summary` and `--outcome`). That's the
wrong layer. This spec corrects it.

## What changes

Schema and CLI flags only. Storage location (Permanent/, `type:
episode`) and the Luhmann placement model stay the same. The
auto-embed pipeline stays the same (truncation still applies).

### Schema changes

**Drop:**
- `--summary` (the narrated paragraph)
- `--outcome` (the bulleted results)

These are L2 content. When a fact or feedback note captures *what
we learned* from the episode, that note carries the abstraction;
the episode carries the raw evidence.

**Add:**
- `--boundary-rationale "<phrase>"` — required. One short phrase
  explaining why this chunk starts and ends where it does.
  Examples: "topic shift from F1 to F6+F9.1 work", "3-day gap
  before resuming", "completed a discrete UAT case",
  "user redirected from cleanup to new feature".
- `--from-transcript-range` — *body source*, required: engram
  reads the actual filtered transcript content for the
  `<session-id>` × `<start>..<end>` slice and renders it inline
  in the episode body. Format: `<session-id>:<RFC3339-start>..<RFC3339-end>`
  (repeatable for multi-session chunks).
- *Alternative:* `--transcript-text` — accept literal text on the
  CLI when the agent already has the chunk in hand. Mutually
  exclusive with `--from-transcript-range`.

**Keep:**
- `--slug`, `--source`, `--situation`, `--target`, `--position`,
  `--relation`, `--vault`.
- `--session` (still names the session ID(s); now also redundant
  with the session id inside `--from-transcript-range`, but
  retain for backward compat and for `--transcript-text` mode).
- `--transcript-range` (still names the start/stop times; same
  redundancy note).

### Rendered episode shape

Frontmatter:

```yaml
---
type: episode
situation: <short topic phrase, retrieval-shaped>
boundary_rationale: <why this chunk's bounds>
provenance:
  sessions:
    - <session-id>
  transcript_range:
    start: <RFC3339 UTC>
    end: <RFC3339 UTC>
luhmann: "<id>"
created: <YYYY-MM-DD>
source: <provenance string>
---
```

Body:

```
<filtered-transcript chunk content — verbatim USER/ASSISTANT/[tool] lines>

Related to:
- [[<wikilink>]] — <rationale>.
- ...
```

No `## Outcomes` section (those went into the linked
facts/feedback notes). No `Information learned:` / `Lesson
learned:` opener.

### Example rendered episode

Filename: `Permanent/215.2026-05-26.f1-spec-sharpening.md`

```yaml
---
type: episode
situation: Sharpening the F1 episode-kind spec via /please
boundary_rationale: Discrete sharpen-then-dispatch arc — starts when the orient step ended, ends when the spec commit landed
provenance:
  sessions:
    - 971fc252-8b44-4bd2-b44a-4f44464105eb
  transcript_range:
    start: "2026-05-25T17:20:00Z"
    end: "2026-05-25T17:25:00Z"
luhmann: "215"
created: "2026-05-26"
source: 'session log engram, 2026-05-26, context: /please drove F1 sharpen'
---

USER: please execute the @docs/superpowers/research/2026-05-24-engram-query-spike.md in a fresh worktree
[tool] Skill(skill="please")
ASSISTANT: I'll set up the task list, enter a fresh worktree...
[tool] TaskCreate(...)
ASSISTANT: Step 1: Opening /learn capture.
[tool] Skill(skill="learn")
...

Related to:
- [[198.2026-05-25.spec-needs-rendered-example-not-just-schema]] — extracted from this chunk.
- [[200.2026-05-25.spec-doc-doubles-as-dispatch-acceptance-contract]] — extracted from this chunk.
```

The body content is the filtered transcript itself — what
`engram transcript --mark` already produces, sliced to the chunk
boundaries. The episode adds the boundary rationale and the
links to derived facts/feedback.

## CLI surface

`engram learn episode` keeps the existing flags except
`--summary` and `--outcome` (removed), and gains
`--boundary-rationale` (required) plus exactly one of
`--from-transcript-range` or `--transcript-text` for the body
content.

| Flag                              | Required | Repeatable | Meaning                                                |
|-----------------------------------|----------|------------|--------------------------------------------------------|
| `--slug <kebab-case>`             | yes      | no         | filename slug                                          |
| `--source "<string>"`             | yes      | no         | provenance string                                      |
| `--situation "<phrase>"`          | yes      | no         | short topic phrase, retrieval-shaped                   |
| `--boundary-rationale "<phrase>"` | yes      | no         | why this chunk's bounds                                |
| `--from-transcript-range <S:R>`   | one of   | yes        | `<session-id>:<RFC3339-start>..<RFC3339-end>`, content read by engram |
| `--transcript-text "<text>"`      | one of   | no         | literal chunk content (mutually exclusive with above)  |
| `--session "<id>"`                | yes (≥1) | yes        | provenance.sessions entry                              |
| `--transcript-range`              | yes      | no         | `<RFC3339-start>..<RFC3339-end>` (overall range)       |
| `--target <luhmann-id>`           | no       | no         | Luhmann parent                                         |
| `--position <top\|continuation\|sibling>` | no | no  | Luhmann placement                                      |
| `--relation "<link>\|<rationale>"`        | no | yes | related notes                                          |
| `--vault <path>`                  | no       | no         | vault root override                                    |

Missing required flags or empty values error loudly. Both
`--from-transcript-range` and `--transcript-text` specified
errors. Neither specified errors.

## Integration with existing learn pipeline

Episode write reuses `writeLearnUnderLock` and `autoEmbedNote`.
Body rendering routes through a new `episodeFromTranscript`
helper that:
- For `--from-transcript-range`: locates the session file via
  the existing `internal/transcript` package, reads bytes for the
  given range, runs them through the existing filter (same one
  `engram transcript --mark` uses), inlines the filtered chunk
  in the body.
- For `--transcript-text`: inlines the provided text verbatim.

No new `LearnDeps` fields beyond plumbing for the transcript
reader.

## /learn SKILL.md changes

The /learn skill needs three corrections:

1. **Episode trigger.** Change from "at most one per /learn pass,
   skip if the session is a continuation with no narrative arc"
   to "one per natural chunk boundary in the filtered transcript
   you just scanned — typically multiple per pass when the
   session spans multiple discrete arcs of work."
2. **Episode body.** Change from "narrative voice, situation +
   summary + outcomes" to "filtered transcript chunk verbatim
   (via `--from-transcript-range`), with a one-phrase
   `--boundary-rationale` explaining why this chunk's bounds."
3. **Fact/feedback ↔ episode linking.** When a fact or feedback
   note is extracted from a specific episode chunk, include
   `--relation "<episode-id>|extracted from this chunk"` on the
   fact/feedback write. The episode then has a backlink to its
   derived notes via wikilink traversal.

The Path A/B/C / recall-mirror discipline still applies to
facts/feedback. Episodes are the L1 evidence layer; their
"discipline" is just "chunk at meaningful boundaries and capture
the time + rationale".

## Backward compatibility

Existing L2-shape episodes (Permanent/201, 210, 214 currently;
any others written before this spec lands) stay in the vault as
legacy. They have `type: episode` but L2-style content. Their
embeddings still work for query; queries that hit them surface
them with their existing content.

New episodes written after this spec lands have the L1 shape.

No migration of old episodes is in scope for this slice.

## Test plan

Unit tests in `internal/cli/learn_test.go`:

- `TestEngramLearn_Episode_BoundaryRationaleRequired` — empty or
  missing `--boundary-rationale` errors.
- `TestEngramLearn_Episode_ExactlyOneBodySource` — neither
  `--from-transcript-range` nor `--transcript-text` errors; both
  specified errors.
- `TestEngramLearn_Episode_TranscriptTextInlined` —
  `--transcript-text "<lit>"` produces a body containing the
  literal text.
- `TestEngramLearn_Episode_FromTranscriptRange_ReadsChunk` —
  fake transcript reader returns deterministic content;
  `--from-transcript-range A:start..end` inlines it.
- `TestEngramLearn_Episode_NoSummaryOrOutcomeFlag` — verifying
  the removed flags are no longer accepted (parsing error).
- `TestEngramLearn_Episode_FrontmatterShape` — frontmatter has
  `boundary_rationale`, no `outcomes` field.
- `TestEngramLearn_Episode_AutoEmbedsSidecar` — sidecar still
  produced (existing behavior preserved).

Integration test in `internal/cli/cli_test.go`:

- `TestEngramLearn_Episode_L1_EndToEnd` — real binary against a
  temp vault; verify filename, frontmatter, body contains the
  filtered transcript chunk, sidecar generated.

Test in `internal/cli/targets_test.go`:

- Existing `TestTargets` updated: the episode flag set changed.

`targ check-full` must pass with no debt.

## Out of scope

Explicit silences:

- Backfill / migration of existing L2-shape episodes. They stay
  as legacy.
- Cross-session episodes (one chunk spanning multiple sessions).
  `--from-transcript-range` is repeatable, but per-chunk
  semantics stay single-session for v2.5.
- Auto-chunking by engram (binary identifies boundaries). The
  agent identifies boundaries; the binary just reads the slice.
- Episode-specific query filter (`engram query --kind episode`).
  Out of scope; consumers post-filter the YAML payload.
- A `superseded_by` link from old L2-shape episodes to new
  L1-shape ones. Migration is its own initiative.
