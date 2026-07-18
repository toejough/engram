<!-- engram-owned: learn-firing guidance. Deployed by 'engram update --with-guidance' to ~/.claude/engram/learn.md; activate via '@~/.claude/engram/learn.md' in CLAUDE.md. Edit via writing-skills TDD. -->

## Capture at the correction moment — fire engram `/learn`, right then

When a review, a failing check, or the user **corrects your approach** mid-task, the confirmed lesson
is in hand for exactly one moment. Then you apply the fix, the turn ends, and it's gone — a future
session pays to re-discover it. So the action at a correction is not just *fix*. It is: **invoke the
engram `/learn` skill to crystallize the confirmed correction, then fix.**

**Fire the engram `/learn` skill specifically** — the `Skill` tool with `skill=learn`, which writes a
vault note (via `engram learn`). NOT a plain note file, NOT Claude Code's native project memory
(`memory/…`, `MEMORY.md`) — those are a different store engram can't recall from. If the only capture
you do is a native memory write, you have missed the cue.

**You do NOT need the full learn sweep mid-cycle.** This is a focused, single-note capture: crystallize
the one confirmed correction now (the `/learn` skill's mid-cycle fast path — skip its Step 1 sweep, go
straight to the crystallize step). Do NOT run `engram ingest --auto` mid-task — the sweep is a
cycle-close job; running it here is wasteful, can block on a large corpus, and losing the capture to a
hung sweep is the failure mode.

Fire at these cues:

- **A review or the user rejected your approach** — "no, do X not Y", "the convention here is Z",
  "that's wrong because W" — `/learn` the confirmed correction **before you apply the fix**. The
  fix-and-move-on reflex closes the turn and the lesson with it.
- **A build/test/check failed and revealed a non-obvious cause** — once you actually know the cause,
  `/learn` it so the next session doesn't re-hit it.
- **You caught your own approach being wrong mid-task** — a self-discovered reversal is the same cue;
  `/learn` the root cause at the moment you catch it.

**Even a one-line fix earns the note.** Small is not below the capture threshold — the value is in the
*rule you were just corrected on*, which is identical whether the fix is one line or ten. "It's a
trivial edit" is exactly the rationalization that loses the lesson.

Capture the **confirmed negative**, not a guess: "review rejected a `posixpath` join; the rule is the
`::` separator" is a fact worth persisting. Do NOT crystallize an unconfirmed hunch ("maybe it's X") —
that rots the vault. You know what was REJECTED with confidence long before the full right answer; the
rejection is the safely-capturable half, and it is enough.

These catch *capture* gaps — the lesson existed at the correction, and the only failure was never
firing `/learn` to write it down.
