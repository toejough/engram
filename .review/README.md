# .review/

`events.jsonl` is the append-only event log behind this app's feedback
threads (see `openspec/changes/add-annotation-feedback/design.md` for the
full design rationale). This file exists so an agent asked to resolve a
comment can do so correctly from this directory alone, without needing
prior conversation context or a trip through the design doc or server
source.

## Open comments are an issue list, not a backlog to clear on your own

An open comment (no `resolution` event referencing its thread yet) is
feedback the user left for whoever reads it next. Treat the set of open
comments the way you'd treat an open issue tracker: when you notice one —
at the start of a session, or in the course of other work — re-present it
to the user (quote the comment, the file, and the anchored/snapshot text)
and offer options for next steps (address it now, defer it, explain why
it doesn't apply, etc.). Don't act on one and write a `resolution` event
on your own initiative just because the fix seems obvious — resolving is
the last step of a discussion the user should get to have, not something
to short-circuit.

This applies to the fix itself, not just the resolution write: don't jump
straight from "here's an open comment" to editing files, rebuilding,
redeploying, or any other side-effecting change without the user agreeing
to that plan first. Once the user has actually signed off on a next step,
carrying it out — including writing the `resolution` event per the rule
below — is the intended, expected use of this log.

## Reading

One JSON object per line. No other structure — read the whole file, group
by `thread_id`, and a thread's current state is whatever its events say
when read in order (there is no separate mutable "status" field to trust
instead).

Fields, by event `type`:

- **`comment`** (opens a thread) — `id`, `type`, `thread_id` (equals `id`
  for the opening comment), `file`, `created_at`, `anchor_commit`,
  `start_line`, `end_line`, `snapshot` (the highlighted text), `text` (the
  feedback itself).
- **`reply`** — `id`, `type`, `thread_id` (of the comment it replies to),
  `file`, `created_at`, `text`.
- **`resolution`** — `id`, `type`, `thread_id` (of the comment it
  resolves), `file`, `created_at`, `resolution_commit`, `summary`.

A comment's current anchor state (current / moved / resolved / orphaned)
is not stored — it's computed at read time by `GET /api/annotations`
(diffing `anchor_commit` against the working tree, and checking whether a
`resolution` event references the thread). Reading this file directly
gets you the raw events, not that computed state; call the endpoint
instead if you need it.

## Writing a resolution

Append one line — a single JSON object, `type: "resolution"`, with at
least:

```json
{"id": "<random hex>", "type": "resolution", "thread_id": "<the comment's thread_id>", "file": "<the comment's file>", "created_at": "<RFC3339 UTC>", "resolution_commit": "<commit that addresses it>", "summary": "<what was done>"}
```

**The write itself must be exactly one `write()` call appending the whole
line (with its trailing newline) to the file opened `O_APPEND`.** That's
the entire correctness mechanism here — there is no file lock. Concurrent
`O_APPEND` writers (this server's own HTTP handler, and an agent writing
directly to disk, as you are) never interleave as long as each writer's
line fits in one atomic write and stays under `PIPE_BUF` (4096 bytes on
Linux; the server itself refuses to append anything larger). Concretely:

- Open with `O_APPEND | O_CREAT | O_WRONLY`.
- Build the complete line first (JSON-encode, append `\n`), then issue one
  write of that whole buffer. Never write the object in pieces, and never
  read-modify-write the file to "update" anything — this log is append-only
  by design; a resolution is a new event, not an edit to the comment it
  resolves.
- Keep the line under 4096 bytes.

Shell equivalent, for reference (note the single redirect append, not an
editor or multi-step rewrite):

```sh
printf '%s\n' '{"id":"...","type":"resolution","thread_id":"...","file":"...","created_at":"...","resolution_commit":"...","summary":"..."}' >> .review/events.jsonl
```
