# Short-term memory for engram — ten approaches

Date: 2026-06-16. Brainstormed via `/please`, then expanded and adversarially
evaluated via a multi-agent workflow (8 idea lenses → distill → diversity critic
→ per-approach evaluation + feasibility verification against the code).

## The trigger

On another machine: the operator created several GitHub issues, cleared context,
then ran `/please` to resolve them. The fresh agent ran `/learn` and `/recall`,
looked the issues up, and *remarked with surprise at how well written they were*
— having entirely forgotten it was the author, minutes earlier.

## The refined target (operator's framing)

Provenance is **not** to be solved by a separate identity mechanism. It should
**fall out of recency**: if recall surfaces the *recent raw transcript events* —
where the agent's own first-person narration lives ("I'll file issue #644 for
X") — then re-reading that narration *is* the agent remembering it did the work.
[P] (authorship) becomes a side effect of [R] (recency) done over the transcript.

So every approach below serves one goal: **get recent transcript-event content
into the recall payload.** Approaches that bolt provenance on as separate
machinery are off-target by construction.

## Diagnosis (verified against the code)

- **`engram query` has zero recency.** Chunk scoring is pure cosine
  (`scoreChunks`, `internal/cli/query_chunks.go:194-226`); notes use
  `max(situation, body)` cosine (`query.go`). Nothing decays or orders by time.
- **Chunks carry no time/session field.** `chunk.Record` is exactly
  `{Source, Anchor, ContentHash, Text, Vector}` (`internal/chunk/index.go:13-25`).
  Time is *latent*: the transcript `Source` path is the session `.jsonl` (filename
  = session UUID), the `Anchor` is `turn-N`, and the ingest `manifest.json` holds a
  **per-source** `MtimeUnixNano` (`internal/cli/ingest.go:48`). None of it is
  extracted into a rankable signal, and the chunk-query path does **not** read the
  manifest today.
- **The learn workflow no longer writes episodes.** `engram learn episode` still
  exists in the binary, but the active path is `engram ingest --auto`, which
  chunks the **raw transcript verbatim** (stripped `USER:`/`ASSISTANT:` turns).
  That is *better* for this goal than episodes — episodes summarize the narration
  away; raw chunks keep it word-for-word.
- **Two latent traps the evaluation surfaced** (they recur across approaches):
  1. *"Newest source by mtime" is usually the **live** post-clear session* (its
     `.jsonl` is being appended right now), not the immediately-prior session we
     want. Any mtime-keyed selector needs to exclude the current session.
  2. *Capture-freshness:* `ingest --auto` runs lazily at the **next** recall and
     the manifest skips unchanged sources by mtime/size — so the prior session's
     **closing** turns (the most recall-critical ones) may not be in the index at
     all when the fresh agent retrieves. See "Precondition" below.

The two problems each approach is tagged against (a lens, **not** a filter):

- **[R] Recency / continuity** — the just-prior session foregrounded over the corpus.
- **[P] Authorship / provenance** — the agent knows *it* produced these artifacts.

Prior research already named the underlying miss: engram has **no working-memory
analog** (`docs/superpowers/research/2026-05-22-human-memory-literature-summary.md`)
— effective retrieval combines recency + importance + relevance, not relevance
alone (Generative Agents, Park 2023; ACT-R activation, Anderson), and a bounded,
ordered, decaying working-memory window is the missing container (Baddeley).

---

## The ten approaches

Grouped by **where recency lives**. Each entry carries the adversarial
feasibility verdict (verified against file:line).

### Query-scoring (re-rank an existing cosine pool)

**1. Recency-decay re-rank · [R] · cost: low · durable-core · feasible-with-caveat**
Widen the cosine pass to a candidate pool, then re-score
`final = cosine × exp(−λ·age) × (1 + β·turnFrac)` and re-sort. `age` comes from
`manifest.json` `MtimeUnixNano` keyed by `record.Source`; `turnFrac` from the
`turn-N` anchor (newest turns → ~1.0). One half-life const, one tail const, no
schema change, no re-ingest. *The single continuous score-blend* — a strongly
on-topic old chunk can still win.
- *Caveats:* `turnFrac` needs a per-source max-`N` pass (no field carries it);
  the chunker merges consecutive turns and stamps the *first* turn's anchor, so
  position is approximate; the consts are untuned and the eval harness that would
  tune them is mid-rework (#642). A multiplicative blend can still leave
  mediocre-cosine narration below the truncation cut.

**2. Recency-quota lane · [R][P] · cost: low-med · durable-core · feasible-with-caveat**
Don't blend — *partition the budget*: reserve `⌈limit·0.25⌉` slots for the newest
chunks (by mtime, then descending `turn-N`), fill the rest from pure cosine,
dedup. *Cannot be tuned into irrelevance* — guarantees the prior tail even on a
cold-topic query.
- *Caveats:* the chunk-query path reads only index files today, so the manifest
  read is **genuinely new wiring** (not "free" as first sketched); it **evicts** a
  cosine slot rather than adding capacity (a hard floor that hurts genuinely
  on-topic queries); and "newest by mtime" can be the live session — needs a
  current-session exclusion.

### Retrieval-structure (change the shape, not the score)

**3. Working-memory channel · [R][P] · cost: med · durable-core · feasible-with-caveat**
A *second, query-independent* channel selects the newest session's last-N turns
and appends them as their own additive band (`recency:true`) that **never evicts
a cosine result** (the structural difference from #2's in-budget quota). Highest
fidelity of the score/index family.
- *Caveats:* same "newest-by-mtime = live session" mis-targeting (must skip the
  live transcript to reach the prior one); promotes `manifest.json` from an
  ingest-only staleness cache to a query-time input.

**4. Since-last-marker cursor replay · [R][P] · cost: low-med · durable-core · feasible-with-caveat**
Use the per-project learn marker (`last-learn-at`, RFC3339Nano) as a *semantic
cursor* and replay the stripped transcript slice `[marker, now]` as a leading
"since last session" block — **bypassing cosine and the chunk index entirely**
(`transcript.ReadFrom` already strips + orders chronologically). Can surface a
session that was never even ingested. **Immune to the capture-freshness trap.**
- *Caveats:* *marker-freshness defeats it* — if the prior session ended with
  `/learn` (this very workflow's close step), the marker ≈ now and the window is
  empty; it's silent exactly when learn was diligent. The byte budget truncates
  the **oldest** rows first, which can drop the recent tail unless the read is
  reversed. Couples a learn-side `$XDG_STATE_HOME` artifact into the query path.

**5. Session-id additive boost · [R][P] · cost: low · durable-core · feasibility NOT independently verified**
Add a bounded additive bonus to every chunk whose `Source` UUID matches the
newest session, floating that whole session up in one pass. Discrete
session-membership (vs #1's continuous curve, #2's hard quota).
- *Caveats:* **off-by-one is fatal as written** — "newest session UUID" is the
  *current* post-clear session, not the prior one; must target second-newest /
  exclude the live session. The bonus is an untuned magic number. *(The
  adversarial verifier for this one died on an API overload; the targeting flaw
  is from the evaluator and is corroborated by the identical bug flagged in #2/#3.)*

### Index / data-model (add the missing primitive)

**6. Coarse time-bucket facet + filter · [R][P] · cost: low-med · durable-core · feasible-with-caveat**
Add one low-cardinality `bucket` field (e.g. `YYYY-MM-DDTHH`) to `chunk.Record`
at ingest and a `--since` filter; recall runs a recency-scoped pass (filter to the
newest bucket, then cosine) alongside the topical pass. Recency as a **filter**,
not a weight/slot/band — keeps pure cosine ranking intact.
- *Caveats:* schema change + re-ingest (vectors reused by hash, so I/O not GPU).
  Threading is harder than "one facet": `ReadResult` exposes only a single
  `LastTimestamp`, and merged-turn chunks span turns with different times — needs
  a per-chunk reduction policy. A coarse bucket usually spans several
  same-period sessions (incl. the live one); the `Source` UUID is a sharper key
  the bucket trades away.

**7. Embed a temporal/session token into the vector · [R] · cost: high · situational · feasible, LOW fidelity**
Prepend a recency/session token to chunk text before embedding so recent chunks
*cosine-cluster*; a recency-flavored recall phrase pulls that cluster. Recency in
**geometry**, not a field.
- *Caveats (severe):* a time-varying prefix **breaks the hash-keyed vector
  reuse** (`ingest.go:392-404`) — every sweep becomes a full re-embed. A short
  token is semantically swamped by ~500 chars of topical text in MiniLM-L6;
  surfacing is probabilistic, unverifiable, and still only per-session granular.
  Weakest mechanism in the set.

### Graph-topology

**8. Session-spine + `continues`-edges · [R][P] · cost: high · situational · feasible-with-caveat**
Add a synthetic per-session "spine" node linking all its chunks, chain spines
with mtime-ordered `continues` edges, force-seed + hub-boost the latest spine so
the existing 3-hop BFS walks back into the prior session.
- *Caveats:* the headline selling point is **false** — the BFS/hub machinery
  operates on `vaultgraph.Note`, **never on chunks**; there is no chunk graph to
  reuse, so it must be built from scratch (new node type, new edge type,
  manifest-on-query). Highest cost/blast-radius way to get recent turns in; a
  plain mtime sort achieves the same surfacing for a fraction of the change.

### Write-synthesis

**9. Revive the dormant `learn episode` as a verbatim handoff · [R][P] · cost: low-med · situational · INFEASIBLE (as a recency mechanism)**
Re-enable `engram learn episode` at close, body sourced verbatim from the recent
transcript window; episodes are notes, so recall ranks them.
- *Why it fails:* (a) recall has **no recency** — an episode is surfaced by
  *phrase overlap* like any note, so the one thing required (recent foregrounded)
  is exactly what it does **not** deliver without also touching the ranker; (b) the
  whole-session body collapses into **one** 384-dim vector that
  `HugotEmbedder.Embed` **truncates** — the "#644" turn can fall past the char cap
  and never reach the vector, while `ingest --auto` already vectors that turn
  individually; (c) a `## Summary` is **mandatory** (an LLM turn at every close),
  reintroducing exactly the cost the learn skill deleted episodes to avoid, and
  overturning documented doctrine. Dominated by the free per-turn chunks.

### Skill / harness layer

**10. SessionStart hook re-injects the prior tail · [R][P] · cost: low · ship-now · feasible-with-caveat**
A Claude Code `SessionStart` hook resolves the project slug from cwd, calls
`engram transcript` with a recent (`--from`/marker-bounded) window, and returns
the stripped turn dump as `additionalContext`. The clear wipes the in-context
conversation; the hook reconstitutes the tail from the on-disk `.jsonl` **before
the user speaks**. **Zero Go change.** **Immune to capture-freshness** (reads the
raw transcript, not the index).
- *Highest fidelity of any approach* for surfacing the agent's own narration —
  `strip.go` emits `ASSISTANT: <text>` verbatim; the agent boots already holding
  "I'll file #644".
- *Caveats:* harness-coupled to Claude Code's hook contract (no OpenCode / non-hook
  harness); the hook **must be read-only** (`--from`/marker, never `--mark`, which
  hard-fails on a fresh project) and needs a one-shot/dedupe guard — all in
  untested shell outside `targ test`. *(Note: the feared marker contention with
  `ingest --auto` is a non-issue — `ingest` uses a separate `manifest.json` and
  never touches the learn marker.)*

---

## Comparison

| # | Approach | Where recency lives | Cost | Fidelity | Feasibility |
|---|----------|---------------------|------|----------|-------------|
| 1 | Recency-decay re-rank | query score | low | med-high | with caveat |
| 2 | Recency-quota lane | query budget | low-med | high (evicts cosine) | with caveat |
| 3 | Working-memory channel | separate band | med | high | with caveat |
| 4 | Since-marker cursor replay | raw transcript @ query | low-med | highest (in-engram) | with caveat |
| 5 | Session-id boost | query score | low | med | targeting bug; unverified |
| 6 | Time-bucket filter | new chunk field | low-med | high-med | with caveat |
| 7 | Embed temporal token | vector geometry | high | low | low payoff |
| 8 | Session-spine graph | graph topology | high | med-high *if* built | reuse claim false |
| 9 | Revive episode | write artifact | low-med | low | infeasible (no recency) |
| 10 | SessionStart hook | harness boot | low | highest overall | with caveat |

## Precondition that sits above all ten (completeness critic)

**Capture-freshness / freshness-at-close.** Approaches #1, #2, #3, #5, #6, #7, #8
all re-rank/filter/graph over the **chunk index** — but `ingest --auto` runs
lazily at the *next* recall and skips unchanged sources, so the prior session's
**closing** turns may not be indexed when the fresh agent retrieves. If the
narration isn't captured, no retrieval lever can surface it. The clean fix is a
**close-side flush** (a `Stop`/`SessionEnd`/`PreCompact` hook that ingests the
just-ended session's tail before context is lost). Notably, **#4 and #10 dodge
this entirely** by replaying the raw transcript instead of the index — a strong
point in their favor.

## Axes the ten do not cover (for the operator to consider swapping in)

The completeness critic flagged these distinct levers, none represented above:
- **Phrase-side:** have the `recall` skill seed a recency-anchored query phrase
  from the agent's own first sentence — pull the prior tail via ordinary cosine,
  zero binary change.
- **Speech-act selection:** up-weight first-person *commitment/completion* turns
  ("I'll…", "I filed…", "done") — the exact carriers of authorship — rather than
  selecting by time/source alone. Most directly serves "P falls out of R".
- **Within-session injection decay:** surface recent events strongly at the first
  post-clear turn, fade as the new session accrues its own context (avoid
  re-injecting the same tail every recall).
- **Open-vs-completed:** the motivating case ("I'll file #644") is a *commitment*
  that may be undone by the clear; distinguish open commitments from finished work.
- **Multi-session threading:** reason about a work-thread spanning several short
  post-clear sessions, not just the single newest `.jsonl`.

## Recommendation (a position to react to, not a final pick)

The adversarial pass reshaped my earlier read. All ten stand for evaluation; this
is where I'd put weight:

- **Ship-now / highest fidelity: #10 (SessionStart hook).** Zero Go, immune to
  capture-freshness, verbatim first-person narration at boot. The cost is
  harness-coupling (Claude Code only) and untested shell guards. Best
  single bet to *validate the hypothesis* that re-reading recent narration fixes
  the symptom — before committing to any schema or ranker change.
- **Best in-engram, harness-agnostic: #4 (cursor replay)**, paired with a
  **close-side flush** to neutralize its marker-freshness blind spot. Together
  they make "replay what just happened" reliable inside engram.
- **Durable retrieval core: #1 or #2.** Once the hypothesis holds, fold recency
  into ranking so it helps *every* recall, not just the boot. Both need the
  manifest plumbing and a current-session exclusion; #2 guarantees inclusion but
  evicts cosine, #1 is gentler but can be tuned into irrelevance.
- **Park / drop: #7, #8, #9.** High cost or dominated; #9 is infeasible as a
  recency mechanism without also changing the ranker.

Whatever is chosen should be **measured, not asserted** — via the memory eval
harness (`docs/superpowers/specs/2026-05-29-memory-eval-harness-design.md`),
which is itself mid-rework (issue #642); clearing those blockers is a prerequisite
to gating any of these empirically.

> Per the operator's working style, this artifact keeps a diagnosis and a
> recommendation rather than presenting a neutral survey — engram's "verify, don't
> guess" first principle requires grounding the pitches in the real code, and a
> position is more useful than a menu. The ten options remain independently
> evaluable; the recommendation is a starting argument, not a foreclosure.
