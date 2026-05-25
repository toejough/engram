# F1 — episode kind spec

Date: 2026-05-25. Implementation contract for the third Permanent
kind (episodes), per F1 in the v2 execution roadmap.

## Storage

Permanent kind with `type: episode` in frontmatter. Same `Permanent/`
directory, same Luhmann numbering, same `.vec.json` sidecar, same
recall cascade. Heterogeneity is gated downstream by the `type:`
field; consumers already key on it.

Rejected alternative: separate `Episodes/` directory. Would
multiply scan paths, fork the Luhmann numbering, and force
`engram query` / `engram recall` / `engram embed` to grow a
second axis. The roadmap calls for "name the operation, not the
workflow" — episodes are the same operation (write a Permanent
note) with a different shape.

## Schema

Frontmatter:

```yaml
---
type: episode
situation: <narrative phrase — project/file/dates OK>
provenance:
  sessions:
    - <session-id or transcript filename>
  transcript_range:
    start: <RFC3339 UTC>
    end: <RFC3339 UTC>
luhmann: "<id>"
created: <YYYY-MM-DD>
source: <provenance string, same shape as fact/feedback>
---
```

Body:

```
<summary paragraph(s) — verbatim, no auto-prefix>

## Outcomes
- <outcome 1>
- <outcome 2>
- ...

Related to:
- [[<wikilink>]] — <rationale>.
- ...
```

Field semantics:

- `situation` — narrative phrase locating the work. Project names,
  dates, file paths OK. Not the retrieval-shaped "When …" phrase
  facts/feedback use.
- `summary` — paragraph(s) narrating what happened. LLM voice,
  first-person ("I did X") OK. Repeatable `--summary` flags
  concatenate as separate paragraphs.
- `outcomes` — bulleted list. Concrete results: decisions reached,
  artifacts produced, problems surfaced, follow-ups left. One
  line per outcome.
- `provenance.sessions` — at least one. Session ID (Claude Code
  UUID) or transcript filename. Multiple if the episode aggregates
  multiple sessions (rare; v2 default is one).
- `provenance.transcript_range` — start/end RFC3339 UTC. Bounds
  the source material for audit and re-derivation.
- `related` — wikilink|rationale entries, same shape as
  fact/feedback. Episodes link forward to facts/feedback spawned
  from them; facts/feedback may link back via their own
  `--relation`.

No auto-generated opener line (unlike facts'
`Information learned: …` or feedback's `Lesson learned: …`). The
summary IS the body.

Paragraph-length `summary` and multi-bullet `outcomes` live in the
body rather than frontmatter because YAML scalars handle neither
cleanly — same reason `Related to:` lives in the body for
fact/feedback notes. Frontmatter keeps only the keyed metadata
(situation phrase, provenance, Luhmann, source).

### Example rendered episode

Filename: `Permanent/198.2026-05-25.f1-episode-kind-spec-sharpening.md`

```markdown
---
type: episode
situation: Sharpening the F1 episode-kind spec for engram v2 before dispatching implementation
provenance:
  sessions:
    - 971fc252-8b44-4bd2-b44a-4f44464105eb
  transcript_range:
    start: 2026-05-25T22:00:00Z
    end: 2026-05-25T23:30:00Z
luhmann: "198"
created: "2026-05-25"
source: 'session log engram, 2026-05-25 23:00 UTC, context: /please drove sharpen-then-dispatch for F1 per the v2 roadmap'
---

Ran the seven-step /please workflow against "keep executing on the v2 roadmap". Opening /learn moved no marker (byte-cap continuation). Orient ran a recall cascade over 26 relevant notes; the spec-craft surfaces (subtraction, deliberate silences, property-ledger shape, voice/vocabulary independence) shaped the spec form directly. Wrote ~150 lines pinning storage, schema, discipline, CLI surface, SKILL.md additions, test plan, and out-of-scope.

## Outcomes
- Spec landed at docs/superpowers/research/2026-05-25-episode-kind-spec.md, committed before dispatch.
- Implementation dispatched to a subagent with the spec's test plan quoted as acceptance criteria.
- Out-of-scope explicitly enumerated to prevent scope creep on the implementation side.

Related to:
- [[157.2026-05-25.minimal-artifact-format-design-is-subtraction-not-accretion]] — applied: spec written by subtraction, with explicit out-of-scope section.
- [[174.2026-05-25.parent-subagent-split-is-an-economics-problem-not-coordination]] — applied: parent shapes the spec; subagent does the implementation drafting.
```

## Discipline

Two independent rules apply simultaneously:

- **Voice freedom.** Narrative tone, first-person, project names,
  dates, file paths, commit SHAs all OK. Episodes are the place
  for "I did X then Y because Z" framing that facts/feedback
  forbid.
- **Vocabulary fidelity.** Named decisions, session IDs, file
  paths, commit SHAs, error strings, and other vocabulary that
  came from source material stay verbatim. Do not paraphrase a
  session ID; do not re-name a decision.

Forbidden in episodes:

- Analysis dressed as narrative — if the content is a principle,
  write a fact; if it's "do X differently next time", write
  feedback; episodes are the narrative connecting tissue only.
- Speculation about future work — research belongs in
  `docs/superpowers/research/`, not in the vault.
- Other agents' actions narrated as if first-person — be explicit
  about who did what when an episode mentions parallel agents.

Path A/B/C and the recall-mirror test do NOT apply to episodes.
Those rules govern facts/feedback because facts/feedback are
retrieved by phrase-matching against future plans. Episodes are
retrieved through the situational stream (project context, time
range, related-note traversal), so their `situation` field is
narrative, not retrieval-shaped.

## CLI surface

`engram learn episode` with flags:

| Flag                       | Required | Repeatable | Meaning                                                   |
|----------------------------|----------|------------|-----------------------------------------------------------|
| `--slug <kebab-case>`      | yes      | no         | filename slug (same shape as fact/feedback)               |
| `--source "<string>"`      | yes      | no         | provenance string (same shape as fact/feedback)           |
| `--situation "<phrase>"`   | yes      | no         | narrative situation phrase                                |
| `--summary "<paragraph>"`  | yes      | yes        | summary body; each instance is one paragraph              |
| `--outcome "<bullet>"`     | yes (≥1) | yes        | outcomes list; each instance is one bullet                |
| `--session "<id>"`         | yes (≥1) | yes        | provenance.sessions entry                                 |
| `--transcript-range`       | yes      | no         | `<RFC3339-start>..<RFC3339-end>`                          |
| `--target <luhmann-id>`    | no       | no         | Luhmann parent (default: top)                             |
| `--position <top\|continuation\|sibling>` | no | no  | Luhmann placement (default: top)                          |
| `--relation "<link>\|<rationale>"`        | no | yes | related-note bullet (same shape as fact/feedback)         |
| `--vault <path>`           | no       | no         | vault root override (env: `ENGRAM_VAULT_PATH`)            |

Missing required flags emit a loud error, not a silent default
(per the "defaults and silent paths" surface rule). Empty
`--summary` or empty `--outcome` reject at parse time.
`--transcript-range` rejects when start ≥ end or either side is
unparseable.

### Integration with existing learn pipeline

`engram learn episode` reuses `writeLearnUnderLock` (vault lock,
Luhmann ID computation, file write) and `autoEmbedNote` (sidecar
generation) without modification. The implementation adds an
`episodeFields` / `episodeFrontmatterDoc` pair alongside the
existing `factFields` / `feedbackFields` and routes through
`assembleLearnContent` via a new `typeEpisode` case. No new
`LearnDeps` fields beyond what facts/feedback already plumb.

## SKILL.md additions

In `skills/learn/SKILL.md`:

1. **Intro** — replace "Two kinds of notes" with "Three kinds of
   notes" and add an Episodes bullet:
   > **Episode** — narrative arc of a session or work segment.
   > "What I did, in what order, with what outcomes." Project
   > names, dates, first-person framing all OK; vocabulary stays
   > verbatim. The note exists so future-you can answer "what was
   > I working on" and trace where facts/feedback came from.

2. **Episode-specific workflow section** — added after the
   facts/feedback workflow. Covers:
   - When to write (at most one per /learn pass; skip if the
     session is a continuation with no narrative arc of its own).
   - Voice + vocabulary discipline (cross-reference to this spec).
   - Why path A/B/C does NOT apply.
   - Example `engram learn episode` invocation.

3. **Related-section guidance** — note that fact/feedback writes
   in the same /learn pass MAY add a `--relation` back to the
   episode (forward references encouraged, not required;
   backlinks are computed from `--relation` at episode-write time).

The actual SKILL.md prose is written via
`superpowers:writing-skills`, which enforces TDD on trigger and
behavior changes.

## Test plan

Unit tests in `internal/cli/learn_test.go`:

- `TestEngramLearn_Episode_RenderingShape` — frontmatter + body
  assemble correctly for a minimal episode.
- `TestEngramLearn_Episode_ProvenanceRequired` — missing
  `--session` or `--transcript-range` errors; bad
  `--transcript-range` format errors.
- `TestEngramLearn_Episode_OutcomeRepeatable` — multiple
  `--outcome` bullets emit in order.
- `TestEngramLearn_Episode_SummaryParagraphs` — multiple
  `--summary` flags concatenate as separate paragraphs.
- `TestEngramLearn_Episode_LuhmannPlacement` — top, continuation,
  sibling all compute the right ID.
- `TestEngramLearn_Episode_AutoEmbedsSidecar` — episode write
  produces a `.vec.json` sidecar via the same auto-embed path.

CLI surface test in `internal/cli/targets_test.go`:

- `TestTargets` gains a `learn episode` registration check
  (length assertion + subcommand walk).

Integration test in `internal/cli/cli_test.go`:

- `TestEngramLearn_Episode_EndToEnd` — runs the binary against a
  temp vault, verifies filename, frontmatter, body, sidecar.

`targ check-full` must pass with no debt before declaring done
(no lint suppression, no coverage relaxation).

## Out of scope

Explicit silences (per "structured artifact silences are
deliberate design"):

- Episode backfill from past transcripts. Episodes start empty in
  v2 and accumulate forward.
- LLM-driven episode synthesis (auto-extracting facts/feedback at
  episode-write time). Deferred to F9.1 in v3.
- Episode-specific recall DSL. Episodes are retrieved through
  `engram query` semantic search and through forward/backward
  wikilink cascade in `engram recall`.
- Migration of `_legacy/MOCs/` content into episodes. Legacy is
  audit-only.
- Multi-session episodes (one episode spanning ≥2 sessions).
  Cross-session narrative goes in a separate fact/feedback.
- Episode pruning/expiry. Episodes are durable, same as
  facts/feedback.
- A `kind: episode` filter on `engram query`. Out of scope;
  consumers can post-filter the YAML payload.
