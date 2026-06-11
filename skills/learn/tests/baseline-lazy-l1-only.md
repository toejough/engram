# Baseline test — lazy learn writes L1 episodes only (defers L2 to recall)

**What this tests:** the `/learn` DEFAULT write behavior. Under the lazy method, `/learn` writes
**L1 episodes — one per work-arc — and does NOT write facts/feedback (L2) at learn time;** recall
crystallizes L2 on demand. The test also guards the episode behavior the user corrected: **L1-only
means *all* the episodes (one per arc), NOT "exactly one episode per build."**

## Scenario given to the agent

> You just finished a build session in `/tmp/notes-cli`. It had **two distinct work-arcs**:
> - **Arc 1** — implemented the storage layer: a `Store` interface injected into the command
>   handlers (DI), atomic file writes (temp-file + rename), and wrapped errors throughout.
> - **Arc 2** (after a context switch) — added a `--json` output mode and a `NO_COLOR`-aware
>   formatter.
>
> Several conventions recurred (you've applied DI, atomic writes, and error-wrapping on prior CLIs
> too). You are now running `/learn` in its DEFAULT mode. List exactly what notes you write: for
> each, give its **kind** (episode / fact / feedback), a one-line subject, and the total **count of
> each kind**.

## Pass / fail

- **PASS (lazy / spec-correct):** writes **2 episodes** (one per arc) and **ZERO facts/feedback**,
  explicitly noting that L2 (the DI / atomic / error-wrapping conventions) will be **crystallized at
  `/recall`**, not written here.
- **FAIL (eager — current behavior):** writes facts (e.g. one per recurring convention: DI, atomic,
  error-wrapping) and/or feedback at learn time. Any L2 write at learn time is a fail.
- **ALSO FAIL (the "exactly one episode" misimplementation):** collapses the two arcs into a single
  episode. L1-only must preserve one-episode-per-arc.
