---
name: learn
description: Use at the end of every user request once meaningful work has happened — feature implemented, bug fixed, plan step closed, decision made, direction changed, discussion resolved. Also fires on explicit cues ("remember this", "save that for later", "/learn", "write up what we just did"). Default to firing; skip only for trivial edits (typo, rename, one-liner) where no lesson could plausibly exist. Captures lessons from the current session as permanent agent-memory vault notes that pass the Recurs + Activity-and-Domain + Knowledge gates.
---

# Learn

## Overview

Capture lessons from this session into the agent-memory vault as **permanent notes** (and **MOCs** when a real framing paragraph emerges across notes). One stage — no fleeting tier, no escape hatch. A candidate either passes all three gates and is written, or fails and is dropped.

This vault is your (the LLM's) persistent memory. You write everything; the human curates by directing what gets worked on. **Don't draft and ask for review** — you decide what becomes permanent and write it.

Style reference: https://obsidian.rocks/getting-started-with-zettelkasten-in-obsidian/. Source method: https://zettelkasten.de/introduction/.

## Vault paths

- Vault root: `/Users/joe/repos/personal/agent-memory/`
- Permanents: `<vault>/Permanent/`
- MOCs: `<vault>/MOCs/`

No `Fleeting/` directory. No `Main Index.md`. No log file. Chronology lives in filenames; navigation lives in MOCs and link context.

## Trigger modes

- **End of user request (primary)** — when a user request is complete, the skill self-fires to sweep the just-completed work. "Complete" means meaningful work has finished: feature shipped, bug fixed, plan step closed, decision made, direction changed, discussion resolved. Same gate sequence and write discipline. No user prompt before write. **Default to firing.**
- **Explicit cue** — `/learn`, "remember this", "save that for later", "write up what we just did". Input grain is determined from context: single observation when a specific moment is flagged; session-batch sweep when invoked at the end of a chunk of work.

**Do not fire on micro-tasks** (one-line edits, single-file moves, trivial renames, typo fixes, pure lookups). The threshold is "a chunk of work that *could plausibly* produce lessons" — not "anything ended." When unsure between firing and not firing, fire.

## The three gates

For each candidate, run gates in order. **A single failure drops the candidate.** No retries; no escape hatches. (You may reframe the situation once and re-run gates — see Gate 2.)

### Gate 1 — Recurs

Strip the situation to **activity + domain**. If it names:

- this project (engram / traced / etc.), its internals, or its architecture
- phase numbers, issue IDs, commit hashes, dates
- one-time events ("user said X today"), diary entries, status snapshots

…the candidate fails Recurs. An agent working on an unrelated project (web app, game, data pipeline) should plausibly hit the same situation.

### Gate 2 — Activity-and-domain framing

The `situation` field describes what an agent would be embarking on, framed as it would be queried **before** the lesson is known. No hindsight; no diagnosis-as-situation.

| Bad (bakes in hindsight) | Good (activity + domain) |
|---|---|
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |
| "When debugging the failing test" | "When writing tests that interact with the filesystem" |

If the candidate fails this gate, you may reframe the situation **once** and re-run all three gates. If still failing, drop.

### Gate 3 — Knowledge bar

From zettelkasten.de: *"Information is dead and contextless; knowledge adds relevance and context. Translate information into knowledge by enriching it with applicability."* A candidate that merely describes what happened is information; it converts only when restateable as a principle with applicability beyond the originating event.

No word counts. No graduation rates. No "useful 2 years out." Just: can this be stated as a transferable principle?

## Workflow

### 1. Identify candidates

Scan the in-context conversation (default) or session logs (when source isn't loaded) for:

- **User corrections** — the user told you to do something differently
- **Failed approaches** — something was tried and didn't work
- **Discovered facts** — new knowledge about tools, idioms, conventions, gotchas
- **Recurring patterns** — behaviors that should be codified

### 2. Apply the three gates

For each candidate, run **Recurs → Activity-and-Domain → Knowledge** in order. Fail at any step → drop. Single-failure reasons are useful in the final report.

### 3. Decide disposition per survivor

- **New permanent** — one candidate → one new top-level permanent
- **Continuation** — write a new permanent as a continuation under an existing one (`--target <id> --position continuation`, e.g. existing `1` → new `1a`). Covers both sharpening the parent's wording with another instance and adding claims that elaborate it; the distinction is a body-content judgment, not a separate disposition.
- **Split** — one candidate bundles multiple principles → multiple permanents

Continuation is always a fresh file under the parent's lineage — never an in-place edit. `engram learn` has no edit operation; reinforcing or elaborating an existing principle IS a new continuation note, which preserves the time-shape of the thinking.

### 4. Decide Luhmann position per write

For each write, find the most-related existing note. Choose `--position`:

- `continuation` — extends the related note's lineage (`1a` → `1a1`)
- `sibling` — parallel branch at the same level (`1a` → `1b`)
- `top` — brand new top-level thought (`5`, `6`, ...)

The binary computes the actual ID under a vault lock. **You do not compute the ID yourself.**

`--position` controls Luhmann placement. **`--relation` is a separate, repeatable flag** that supplies the `Related to:` bullets — see step 5.

### 5. Draft body in LLM voice

**Every `engram learn` invocation MUST include `--source`.** It is a required flag; the binary errors out when it is missing. The provenance string takes one of these forms:

- For feedback/fact derived from session activity: `session log <project>, <YYYY-MM-DD HH:MM UTC>, context: <short description>`
- For MOCs synthesized from cluster analysis: `constructed from cluster analysis, <YYYY-MM-DD>`
- For end-to-end smoke or test runs: a short label naming the run (e.g. `smoke test`)

**All body content is supplied via flags. Stdin is not read.**

- `--relation <wikilink-target>|<rationale>` — repeatable; each instance adds one `Related to:` bullet. The pipe `|` separates the wikilink target from its per-link rationale. Example: `--relation "1a.foo|same shape, different domain"`.
- `--framing "..."` — MOC only; the framing paragraph(s) that form the MOC body. Do NOT auto-list constituents; backlinks already do that.

The `Lesson learned: ...` / `Information learned: ...` opener line is auto-generated from `--situation` and `--action`/`--subject`/`--predicate`/`--object`. **Do not duplicate it in any flag.**

**Feedback:**

```
engram learn feedback \
  --slug <kebab-case-tag> \
  --vault /Users/joe/repos/personal/agent-memory \
  --target <luhmann-id-of-related-note-or-empty> \
  --position <top|continuation|sibling> \
  --source "session log <project>, <YYYY-MM-DD HH:MM UTC>, context: ..." \
  --situation "..." --behavior "..." --impact "..." --action "..." \
  --relation "<wikilink>|<rationale>" \
  --relation "<wikilink>|<rationale>"
```

**Fact:**

```
engram learn fact \
  --slug <kebab-case-tag> \
  --vault /Users/joe/repos/personal/agent-memory \
  --target <id-or-empty> \
  --position <top|continuation|sibling> \
  --source "..." \
  --situation "..." --subject "..." --predicate "..." --object "..." \
  --relation "<wikilink>|<rationale>"
```

**MOC** (judgement-based, no count threshold):

```
engram learn moc \
  --slug <kebab-case-tag> \
  --vault /Users/joe/repos/personal/agent-memory \
  --target <id-or-empty> \
  --position <top|continuation|sibling> \
  --source "constructed from cluster analysis, <YYYY-MM-DD>" \
  --topic "<theme name>" \
  --framing "<framing paragraph(s)>" \
  --relation "<wikilink>|<rationale>"
```

### 6. Contradictions

If a new permanent contradicts an existing one, write the new permanent with a `Related to:` bullet whose rationale names the discrepancy. Surface in the final report. Don't smooth.

### 7. Write — one parallel tool-use block

**Hard rule: all `engram learn` invocations for a single /learn pass go in a single parallel tool-use block.** Serial writes cost a tool roundtrip each (~15–20s); batching collapses that.

### 8. Report

Per pass:
- Candidates considered
- Gates passed / failed (with gate name and one-line reason)
- Permanents written (with Luhmann IDs)
- MOCs written or updated
- Contradictions surfaced

## Quality bars

- **Atomicity** — one idea per permanent.
- **Autonomy** — permanents are understandable without context. Strip "this case", "the incident", "we did X" framing.
- **Knowledge, not information** — the principle has applicability beyond the originating event.
- **LLM voice** — translate raw material into your own synthesis. Verbatim user quotes get rephrased on writing.
- **Per-link rationale** — every `Related to:` bullet explains why the connection exists. No bare wikilinks.
- **Heterarchy** — a permanent can belong to multiple MOCs; one `Related to:` bullet per MOC with its own rationale.
- **Surface contradictions** — link them with rationale naming the discrepancy.

## Common mistakes

| Mistake | Fix |
|---|---|
| Writing a note whose situation names "engram", "Task 8", "promote.go" | Fail at Recurs gate; drop |
| Hindsight-baked situation ("When fixing the bug in X") | Fail at Activity+Domain gate; reframe to pre-lesson query phrasing |
| Writing "we observed X" without stating it as a principle | Fail at Knowledge gate; either restate as principle or drop |
| Drafting and asking for human voice rewrite | You're the writer. Just write. |
| Writing files directly with the filesystem | Use `engram learn {feedback|fact|moc}` — handles ID assignment under lock |
| Computing the Luhmann ID yourself | Pass `--target` and `--position`; binary computes the ID |
| Putting a `Lesson learned:`/`Information learned:` opener inside `--framing` or any flag | The opener is auto-generated; never repeat it. Body bullets go in `--relation`, framing in `--framing`. |
| Piping body content via stdin | Stdin is ignored. All body content goes through `--relation` and `--framing` flags. |
| Auto-listing MOC constituents in body | Backlinks already do this — MOC body is framing prose only |
| Bare wikilinks without rationale | Every `Related to:` bullet must include per-link rationale |
| Serial `engram learn` calls across tool turns | One message, N parallel tool calls |
| Auto-firing on a one-line micro-task | Only autonomous-trigger on chunks that plausibly produce lessons; when unsure, don't fire |
| Creating a MOC because the cluster crossed a count threshold | Judgement, not count — a real framing paragraph must emerge |
| Putting an H1 title or `Luhmann-ID · date` line in the body | Filename is the display name; `luhmann` and `created` live in frontmatter |
| Smoothing over contradictions | Write `Related to:` bullets that name the discrepancy |
