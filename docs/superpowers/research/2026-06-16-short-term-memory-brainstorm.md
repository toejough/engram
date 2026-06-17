# Short-term memory for engram — ten approaches

Date: 2026-06-16. Brainstormed via `/please`, then expanded and adversarially
evaluated via multi-agent workflows: 8 idea lenses → distill → diversity critic →
per-approach evaluation + feasibility verification against the code, then a second
verification pass on the approaches added during review. Every approach below
carries a verdict checked against file:line.

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
[P] (authorship) becomes a side effect of [R] (recency) over the transcript.

So every approach below serves one goal: **get recent transcript-event content
into the recall payload.**

## Diagnosis (verified against the code)

The symptom is **purely a ranking problem**, and the verification pass narrowed
it to exactly that — two things people *assume* are also broken are not:

- **`engram query` has zero recency.** Chunk scoring is pure cosine
  (`scoreChunks`, `internal/cli/query_chunks.go:194-226`); notes use
  `max(situation, body)` cosine (`query.go:486-495`). Nothing decays or orders by
  time. This is the whole gap.
- **Capture is NOT the gap** *(corrected during review)*. An earlier critic
  called capture-freshness the dominant omission — that the prior session's
  closing turns might not be indexed yet. A deeper check refuted it: the `recall`
  skill runs `engram ingest --auto` as its **Step 0.5, before the query**, which
  re-reads the prior session's `.jsonl` (newer mtime) and indexes the closing
  turns first. So "I'll file #644" **is** in the index when the fresh agent
  retrieves — it just doesn't rank. The only genuine capture loss is a hard-kill
  before the `.jsonl` flushes, which no hook fixes either.
- **Provenance is NOT a separate gap.** The narration is captured verbatim (raw
  chunks, no episode summarization in the active path). Once recency surfaces it,
  authorship is recovered for free — exactly the operator's framing.
- **Time is latent, not stored.** `chunk.Record` is `{Source, Anchor,
  ContentHash, Text, Vector}` (`internal/chunk/index.go:13-25`) — no time/session
  field. But the transcript `Source` path is the session `.jsonl` (filename =
  session UUID), the `Anchor` is `turn-N`, and the ingest manifest
  (`map[string]manifestEntry`, `internal/cli/ingest.go:123`; keyed per source at
  `:258`) holds a **per-source** `MtimeUnixNano`. The chunk-query path does **not**
  read the manifest today.

Two traps recur across the approaches (see "Cross-cutting" below): the
**live-session trap** ("newest source by mtime" is usually the *current* post-clear
session, not the prior one) and **untuned constants** (the eval harness that would
tune any weight is mid-rework, issue #642).

Prior research already named the underlying miss: engram has **no working-memory
analog** (`docs/superpowers/research/2026-05-22-human-memory-literature-summary.md`)
— effective retrieval combines recency + importance + relevance, not relevance
alone (Generative Agents, Park 2023; ACT-R activation, Anderson), and a bounded,
ordered, decaying working-memory window is the missing container (Baddeley).

[R] = recency/continuity; [P] = authorship/provenance (a lens, not a filter).

---

## The ten approaches

Grouped by **where recency lives** — ten distinct axes. Each is tagged
**CONTENDER** (worth building) or **PARK** (genuinely distinct but weak/redundant/
expensive on the verified read — kept so the contenders have something to be
evaluated against).

### Query-scoring — re-rank an existing cosine pool

**1. Recency-decay re-rank · [R] · CONTENDER · cost low · feasible-with-caveat**
Widen the cosine pass to a pool, then re-score `final = cosine × exp(−λ·age)
[× (1 + β·turnFrac)]` and re-sort. `age` from manifest `MtimeUnixNano` keyed by
`record.Source`; `turnFrac` from the `turn-N` anchor. No schema change, no
re-ingest. *Subsumes a discrete session-membership boost* (add a flat bonus to
all chunks of the newest session) as a tuning variant — same lever, different
curve.
- *Caveats:* the manifest read is **new wiring** on the chunk-query path (it
  reads only index files today); `turnFrac` needs a per-source max-`N` pass (no
  field carries it) and merged-turn chunks stamp the *first* turn's anchor, so
  position is approximate; `λ`/`β` are untuned (#642). Must exclude the live
  session (see Cross-cutting).

### Retrieval-structure — change the shape, not the score

**2. Guaranteed recent inclusion (quota or band) · [R][P] · CONTENDER · cost low-med · feasible-with-caveat**
Put the newest session's tail into the payload *unconditionally*, by either a
reserved **quota** (⌈limit·0.25⌉ slots inside the budget — evicts cosine results)
or a separate additive **band** (out of budget — evicts nothing). The
quota-vs-band choice is the one real sub-decision; both select by mtime + `turn-N`
regardless of cosine, so the tail lands even on a cold-topic query.
- *Caveats:* same new manifest wiring; the quota form trades away a cosine slot
  (a hard floor that hurts genuinely on-topic recalls); must exclude the live
  session or it surfaces the agent's own just-started empty turns.

**3. Cursor replay from the learn marker · [R][P] · CONTENDER · cost low-med · feasible-with-caveat**
Use the per-project learn marker (`last-learn-at`, RFC3339Nano) as a *semantic
cursor* and replay the transcript slice `[marker, now]` as a leading "since last
session" block — **bypassing cosine and the chunk index entirely** (`transcript.
ReadFrom` reduces each turn to `USER:`/`ASSISTANT:` plaintext, noise lines
removed, chronological). Can surface a session even if it was never swept.
- *Caveats:* **marker-freshness defeats it** — if the prior session ended with
  `/learn` (this workflow's own close step), the marker ≈ now and the window is
  empty; it's silent exactly when learn was diligent. The byte budget keeps the
  **oldest** rows and drops the recent tail (walks chronologically forward,
  `transcript.go:203-239`) — so the read must be reversed to keep the tail.
  Couples a learn-side `$XDG_STATE_HOME` artifact into the query path.

### Data-model — add the missing primitive

**4. Index time/session field + recency filter · [R][P] · CONTENDER (heavier) · cost low-med · feasible-with-caveat**
Add one coarse `bucket`/timestamp field to `chunk.Record` at ingest and a
`--since` filter; recall runs a recency-scoped pass (filter to the newest window,
then cosine) beside the topical pass. Recency as a **filter**, keeping pure-cosine
ranking intact — the cleanest "make recency first-class" option.
- *Caveats:* schema change + re-ingest (vectors reused by content hash, so I/O not
  GPU). Threading is harder than "one facet": `ReadResult` exposes only a single
  `LastTimestamp` (per-row times exist but are unexported); merged-turn chunks
  need a per-chunk timestamp-reduction policy. A coarse bucket usually spans
  several same-period sessions; the `Source` UUID is a sharper key it trades away.

### Content-shape — select by what the turn *is*

**5. Speech-act selection · [P] (no [R] of its own) · PARK · cost low · feasible, weak signal**
Up-weight chunks whose text is an assistant-authored commitment/completion
declarative ("I'll…", "I filed…", "done") — the turns that literally carry
authorship. ~10 lines: a lexical boost in `scoreChunks`, since `strip.go` keeps
the `ASSISTANT: ` prefix in chunk `Text`.
- *Why PARK:* turn-merging (`chunk.go:83-94`) puts both roles in most chunks, so
  "contains `ASSISTANT:`" fires on nearly everything — a weak selector. Verb
  regexes are fragile (miss paraphrases); the property is semantic, which vector
  search already handles; and it has **no recency** of its own, so a months-old
  "I'll fix X" ranks equal to a fresh one. Only useful bolted onto #1.

### Write-artifact — persist a recent-events note

**6. Rolling first-person recap note at close · [R][P] · PARK · cost low-med · feasible, redundant**
At close, write a compact first-person "what I just did" note via `engram learn
fact`/`feedback` (no mandatory `## Summary`, unlike the dead episode path — so
this is the *feasible* cousin of reviving episodes).
- *Why PARK:* surfaced by phrase-overlap, **not recency** (same limit as
  episodes); **high redundancy** — `ingest --auto` already chunks the same
  transcript, so it duplicates the index; diary-shaped phrasing is retrieval-
  hostile (future queries are task-shaped); and authoring it costs an LLM turn at
  close — exactly the cost the learn skill deleted episodes to avoid.

### Capture-timing — guarantee the tail is indexed

**7. Close-side capture flush · [R] · PARK · cost low · feasible, marginal**
A `SessionEnd`/`Stop` hook runs `engram ingest --auto` at close so the tail is
indexed before the next session.
- *Why PARK:* the gap it targets **self-heals** — recall's Step 0.5 already
  re-indexes the prior session before every query (see Diagnosis). It buys only
  *earlier* availability, not correctness; on `Stop` it fires every turn (model-
  load latency + the 15s hook timeout risks a silent skip); and it does **not**
  fix the one real loss (hard-kill before flush). `ingest --auto` is idempotent
  (separate manifest, no marker contention), so it's safe — just low-value.

### Harness-boot — inject before the query

**8. SessionStart boot-hook re-injection · [R][P] · CONTENDER · cost low · feasible-with-caveat**
A Claude Code `SessionStart` hook calls `engram transcript` for the project with a
recent window and returns the stripped tail as `additionalContext` — the clear
wipes the in-context conversation, the hook reconstitutes the tail from the
on-disk `.jsonl` **before the user speaks**. **Zero Go change**; bypasses ranking
entirely.
- *Highest fidelity for surfacing the agent's own narration* (in a Claude Code
  deployment) — `strip.go:194-204` emits `ASSISTANT: <text>` verbatim, so the
  agent boots already holding "I'll file #644".
- *Caveats:* harness-coupled to Claude Code (no OpenCode / non-hook harness);
  the hook must be **read-only** (`--from`/marker, never `--mark`, which hard-fails
  on a fresh project) with a one-shot/dedupe guard — all in untested shell. *(The
  feared marker contention with `ingest --auto` is a non-issue — `ingest` uses a
  separate `manifest.json` and never touches the learn marker.)*

### Graph-topology

**9. Session-spine + `continues`-edges · [R][P] · PARK · cost high · feasible, over-built**
A synthetic per-session spine node links all its chunks; spines chain by
mtime-ordered `continues` edges; force-seed + hub-boost the latest spine so the
3-hop BFS walks back into the prior session.
- *Why PARK:* the headline reuse claim is **false** — the BFS/hub machinery
  operates on `vaultgraph.Note`, **never chunks** (`vaultgraph/graph.go`,
  `bfs.go`); there is no chunk graph to reuse, so it's built from scratch (new
  node type, edge type, manifest-on-query). Highest cost/blast-radius for an
  outcome a plain mtime sort achieves far cheaper.

### Vector-geometry

**10. Embedding-geometry recency token · [R] · PARK · cost high · feasible, low fidelity**
Prepend a recency/session token to chunk text before embedding so recent chunks
cosine-cluster, then pull that cluster with a recency-flavored phrase.
- *Why PARK:* a time-varying prefix **breaks the hash-keyed vector reuse**
  (`ingest.go:392-404`) — every sweep becomes a full re-embed. The token is
  semantically swamped by ~500 chars of topical text; surfacing is probabilistic
  and unverifiable; and the agent reads ranker output, not the turn itself, so it
  only *biases* selection rather than guaranteeing the narration appears.

---

## Comparison

| # | Approach | Where recency lives | Verdict | Cost | Fidelity |
|---|----------|---------------------|---------|------|----------|
| 1 | Recency-decay re-rank (+session-boost variant) | query score | CONTENDER | low | med-high |
| 2 | Guaranteed inclusion (quota or band) | retrieval shape | CONTENDER | low-med | high |
| 3 | Cursor replay from marker | raw transcript @ query | CONTENDER | low-med | highest (in-engram) |
| 4 | Time field + recency filter | new chunk field | CONTENDER (heavier) | low-med | high |
| 5 | Speech-act selection | content shape | PARK | low | low (turn-merge) |
| 6 | Rolling recap note | write artifact | PARK | low-med | low (redundant) |
| 7 | Close-side capture flush | capture timing | PARK | low | n/a (self-heals) |
| 8 | SessionStart boot-hook | harness boot | CONTENDER | low | highest (Claude Code) |
| 9 | Session-spine graph | graph topology | PARK | high | med *if* built |
| 10 | Embedding-geometry token | vector geometry | PARK | high | low |

## Cross-cutting findings

- **Live-session trap.** "Newest source by mtime" is usually the *current*
  post-clear session (its `.jsonl` is being appended now), not the prior one we
  want. Every mtime/session selector (#1's session-boost variant, #2, #4) must
  exclude the live session — else it surfaces the agent's own just-started turns.
- **Recency is the only gap.** Capture self-heals (Step 0.5) and provenance falls
  out of recency — so the work is squarely "make recent events rank/appear,"
  nothing more.

## Weaker levers considered, not numbered

- **Phrase-side recency-seed** (recall seeds a recency-anchored query phrase):
  verified **not genuinely distinct** — when concrete anchors are in context it's
  ordinary topical retrieval with a recency label; when they aren't, the generic
  phrase is cosine-useless. A real recency capability needs binary changes (i.e.
  it collapses into #1/#4).
- **Within-session injection decay** (fade the recent injection as the new session
  grows): a modifier on #2/#3/#8, not a standalone lever.
- **Open-vs-completed commitment selection** (surface unfinished commitments): a
  refinement on #5.
- **Multi-session work-thread** (the contiguous thread across several clears, not
  just the newest `.jsonl`): an extension, mostly of #3/#9.

## Recommendation (a position to react to, not a final pick)

The verification pass sharpened the problem to **recency in ranking** — capture
and provenance are already handled. So:

- **Ship-now / validate the hypothesis: #8 (SessionStart hook).** Zero Go, highest
  fidelity, reconstitutes the recent narration verbatim at boot *regardless of
  ranking*. Best first probe that "re-reading recent narration fixes the symptom."
  Cost: Claude-Code-coupled, untested shell.
- **Best in-engram, harness-agnostic: #3 (cursor replay).** Same "verbatim, bypass
  ranking" idea inside engram; reverse the read to keep the tail and accept the
  marker-freshness blind spot.
- **Durable retrieval core: #1 or #2.** Once validated, fold recency into ranking
  so *every* recall benefits, not just the boot. #1 is gentler (tunable, can be
  drowned); #2 guarantees inclusion (but evicts cosine). Both need live-session
  exclusion; #1 subsumes the session-boost variant.
- **Heavier bet: #4** if a filterable time field proves worth the schema change.
- **Park: #5, #6, #7, #9, #10** — genuinely distinct axes, but weak / redundant /
  self-healing / over-built on the verified read. Keep them as the set you
  evaluate the contenders against; don't build them first.

Whatever is chosen should be **measured, not asserted** — via the memory eval
harness (`docs/superpowers/specs/2026-05-29-memory-eval-harness-design.md`),
itself mid-rework (issue #642); clearing those blockers is a prerequisite to
gating any of these empirically.

> Per the operator's working style, this keeps a diagnosis and a recommendation
> rather than a neutral menu — engram's "verify, don't guess" first principle
> requires grounding the pitches in the real code, and the verification pass
> changed the answer (capture is not the gap; phrase-side and the two write/graph
> options are weaker than they first looked). The ten remain independently
> evaluable; the CONTENDER/PARK split is an argument, not a foreclosure.
