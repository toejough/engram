# Baseline scenario — L1 episodes (multiple chunks per /learn pass)

A subagent is given a session that spans multiple discrete chunks of work and is asked to follow the current `skills/learn/SKILL.md` to capture episodes. We measure whether they write multiple episodes (one per natural boundary) and use `--boundary-rationale` + filtered transcript text for the body.

## Scenario prompt (verbatim, give to subagent)

You are an agent working on a vault-backed memory system in the `engram` repo. Your session today spans three discrete arcs:

1. **08:30–10:00 UTC** — Investigated a bug in the embed pipeline; root-caused it to a missing `--force` flag check; landed a fix in `internal/embed/state.go` and added a unit test. Discrete, complete arc — the bug fix is finished by 10:00.
2. **10:00–10:30 UTC** — *3-day pause*. (Real wall-clock gap; the next chunk starts on the next session day.)
3. **10:30–12:00 UTC (3 days later)** — Started new feature work: prototyping the F9.2 query field flags. Read research notes, sketched a CLI surface, did not implement yet. The arc is in progress — no clean "completion" boundary at 12:00; you stopped because the day ended.

Then the user runs `/learn`. They expect — and this is the key — that when they come back tomorrow and ask "what'd we do yesterday", you can answer with high detail about what actually happened: the bug fix, the gap, and the F9.2 prototyping work. They consider it a failure if you only remember the "narrative arc" (one summary paragraph) and not the actual interactions.

The filtered transcripts engram already produced are:

- `~/.claude/projects/-Users-joe-repos-personal-engram/sess-A.jsonl` — covers 08:30–10:00 UTC, the bug-fix arc.
- `~/.claude/projects/-Users-joe-repos-personal-engram/sess-B.jsonl` — covers 10:30–12:00 UTC, the F9.2 prototyping arc.

Read `/Users/joe/repos/personal/engram/skills/learn/SKILL.md` and follow its instructions exactly to run `/learn` against this session. Do **not** actually write to the vault — instead, print the exact `engram learn` invocations you would issue, in the parallel tool-use block the skill mandates. Show your full thought process inline. Use the binary's flags as documented in the skill.

## What we are measuring

A pass requires ALL of:

1. **Multiple episodes written.** At least one episode per natural chunk boundary (here: two episodes — the bug-fix arc and the F9.2 prototyping arc). The 3-day pause is itself a boundary worth annotating.
2. **`--boundary-rationale` populated for each episode** with a phrase explaining why this chunk's bounds. Examples: "completed bug fix arc", "3-day gap before resuming", "stopped because the day ended; arc still in flight".
3. **Body source is the filtered transcript chunk.** Each episode uses `--from-transcript-range <session-id>:<start>..<end>` (or `--transcript-text` if the agent inlines the chunk literally), not narrative summary paragraphs.
4. **No `--summary` or `--outcome` flags.** Those flags no longer exist in the L1 shape.
5. **Facts/feedback link back to their originating episode.** If the agent writes a fact or feedback derived from the bug-fix arc (e.g., "embed pipeline missed --force check" feedback), they include `--relation "<episode-luhmann>|extracted from this chunk"` on that fact/feedback write.

## Failure modes to watch for

- Writes one episode total (treats episodes as session-global narrative).
- Skips episodes because "no new narrative arc occurred" — particularly likely for the F9.2 arc since it's "in progress."
- Uses `--summary` and `--outcome` flags (the old L2 shape).
- Synthesizes a paragraph for the body instead of using transcript-derived content.
- Treats the 3-day gap as a non-event; doesn't annotate the boundary on either side.
- Does not link facts/feedback back to their episode origin.
