# Short-term memory for engram — ten approaches

Date: 2026-06-16. Brainstormed via `/please`.

## The trigger

On another machine: the operator created several GitHub issues, cleared
context, then ran `/please` to resolve them. The fresh agent ran `/learn` and
`/recall`, looked the issues up, and *remarked with surprise at how well
written they were* — having entirely forgotten it was the author, minutes
earlier. The ask: build short-term memory good enough that this can't happen.

## Diagnosis (verified against the code, not guessed)

The symptom is not one gap but three, and they compound:

1. **No recency anywhere in retrieval.** `engram query` scores by pure cosine
   (`internal/cli/query_chunks.go` `scoreChunks`; `query.go` `bestVector` =
   `max(situation, body)`). Nothing decays, boosts, or orders by time. "What
   happened five minutes ago" and "what happened in May" compete on topic
   similarity alone.
2. **Chunks carry no temporal or session identity.** A `chunk.Record` is
   `{Source, Anchor, ContentHash, Text, Vector}` — no timestamp, no session id
   (`internal/chunk/index.go`). The info is *latently* present (the transcript
   source filename is the session UUID; the anchor is `turn-N`; the manifest
   holds source mtime) but it is never extracted into a rankable signal.
3. **The issues were never engram memory at all.** `engram ingest` indexes only
   transcripts and markdown (`internal/cli/ingest.go`). GitHub issues are not a
   source. The agent "remembered" the issues via `gh`, with zero linkage to its
   own prior session — so engram could not have supplied authorship continuity
   even in principle.

Notes (unlike chunks) *do* carry a `created` stamp — but date-only
(`YYYY-MM-DD`, no sub-day resolution) — and a `source` string
(`internal/cli/learn.go`). `source` is unconstrained free text (the
`"claude"`/`"opencode"` values are a *transcript-harness* label elsewhere, not a
constraint on this field), so authorship *could* be stamped there with no schema
change. Nothing records per-agent/session identity today, and recall surfaces
neither field.

So the symptom leans on two problems the ten approaches are tagged against. The
tags are a comparison lens, **not a filter** — an approach that solves the
symptom some other way is just as welcome; the tags only show which problem each
leans toward:

- **[R] Recency / continuity** — the just-prior session should be foregrounded
  over the whole corpus.
- **[P] Authorship / provenance** — the agent should *know it* produced these
  artifacts, this session or last, rather than meet them as a stranger's work.

Prior research already named the underlying miss: engram has **no
working-memory analog** (`docs/superpowers/research/2026-05-22-human-memory-literature-summary.md`)
— effective retrieval combines recency + importance + relevance, not relevance
alone (the Generative Agents memory stream, Park 2023, does exactly this; the
literature summary makes the parallel point via ACT-R activation, Anderson), and
a bounded, ordered, decaying working-memory window is the missing container
(Baddeley).

---

## The ten approaches

### 1. Recency-weighted query scoring  · [R]
Add a time signal to every indexed item and fold it into the score:
`final = cosine × decay(age)` (exponential, Generative-Agents style). Requires
stamping chunks with a timestamp — derivable cheaply at ingest from the
transcript row time / source mtime, no LLM. Notes have a date-only `created`.
The time signal has two flavors worth A/B-ing under one direction: **write-age
decay** (how long ago it was created) or **access-activation** (ACT-R: boost
recently-*used* items by last-access + frequency). Age-decay is stateless and
fits the immutable, content-hash-keyed index; activation is more adaptive but
needs a mutable per-item access log — friction against the current rebuild-whole
index.
- **Fit:** small, surgical change in `scoreChunks` + the chunk schema (age-decay
  flavor). Pure computation — squarely "binary for computation."
- **Risk:** a global decay constant is a blunt instrument; tuning needs the eval
  harness, and over-weighting recency hurts genuine topic matches.

### 2. Explicit working-memory window (new container)  · [R][P]
A bounded, ordered, recency-decaying scratchpad *separate* from the vault: the
last N events of the current + prior session, **always injected verbatim** at
recall, never queried or similarity-filtered. This is the Baddeley working-memory
analog the research flagged as missing. (Contrast #6, which *queries* a recent
store and merges by score; here the recent window bypasses ranking entirely.)
"What just happened" is in every payload by construction, regardless of cosine.
- **Fit:** the most theoretically complete answer; matches the literature
  directly.
- **Risk:** largest new surface — a new store, its lifecycle, its eviction, and
  a recall-payload redesign. Highest blast radius.

### 3. Rolling session-recap note at learn time  · [R][P]
Every `/learn` writes/updates a single first-person "what I just did" note
(rich provenance: issues created, files touched, decisions, open threads).
Recall always surfaces the most recent K recaps. Reuses existing note infra
entirely.
- **Fit:** cheap, no schema change, no binary change beyond a learn convention;
  the first-person voice ("I filed #644, #642…") directly dissolves the
  authorship surprise.
- **Risk:** depends on the agent writing the recap honestly each time; quality
  is only as good as the closing `/learn`.

### 4. Authorship / provenance stamping + recall labeling  · [P]
Stamp every note (and chunk) with agent + session identity, and have recall
explicitly tag items: "← you wrote this, this session." Surgical fix for the
exact symptom ("forgot it authored them"). For notes this needs **no schema
change** — the free-form `source` field already accepts an identity string; the
work is a `learn`/`recall` convention plus chunk-side identity.
- **Fit:** directly targets [P]; minimal-to-no schema change for notes.
- **Risk:** identity is fuzzy across machines/harnesses; "you" is ill-defined
  when the prior session was a different model on a different host. Solves the
  surprise without solving continuity of *content*.

### 5. Ingest GitHub issues (and PRs) as a source  · [R][P]
Teach `engram ingest` to pull `gh issue`/`gh pr` activity into the chunk index
with author + timestamp. Targets the literal root cause: the issues were
invisible to engram.
- **Fit:** closes gap #3 precisely; the artifact the operator actually created
  becomes memory.
- **Risk:** narrow — fixes issues but not the general recency gap; adds a `gh`
  dependency and network I/O to ingest (currently pure-FS, zero-LLM). Couples
  engram to GitHub.

### 6. Two-store CLS model: fast recent + slow semantic  · [R]
Split memory into a fast episodic "recent" store (last session/day,
recency-ranked, aggressively archived) and the slow semantic vault
(similarity-ranked). Recall queries both and merges. Mirrors Complementary
Learning Systems (the research's "exactly two stores, not four").
- **Fit:** principled, scales, matches biology cited in the research.
- **Risk:** a larger architectural commitment than #1/#2. Distinct from #2 (that
  one *injects* a recent window unconditionally; this one keeps recent memory in
  the retrieval path and merges by score), but it may be over-engineering
  relative to the immediate symptom (YAGNI flag).

### 7. Recall surfaces a "since last session" block (skill-only)  · [R]
No binary change: the `recall` skill runs a second, *time-ordered* pass — list
the most-recently-created notes and the most-recent transcript turns — and
injects them as a distinct "here's what you just did" block, beside the cosine
results. Could be a thin `engram recent` flag if listing needs the binary.
- **Fit:** cheapest possible; ships today; reversible; tests the hypothesis that
  *foregrounding* recent work (not better ranking) is what's missing.
- **Risk:** a bolt-on, not a model; doesn't help non-`/recall` paths; "recent"
  by note `created` ignores chunk recency.

### 8. Temporal-edge continuity via the graph walk  · [R][P]
Continuity as *links*, not scores. At `/learn`, link each new note to the
immediately-prior session's notes with an explicit `continues`/`temporal`
relation. Recall already does a 3-hop BFS over wikilinks (`internal/cluster/`,
the recall flow's subgraph walk), so from any current-topic seed the walk
naturally pulls in the recent chain — and the chain is self-evidently *this
agent's* trail, addressing [P] too. Engram-native: reuses machinery that exists.
- **Fit:** orthogonal to ranking entirely; leverages the existing graph walk;
  no new store and no scoring change.
- **Risk:** only surfaces recent work when a topic seed connects to it; a
  cold-open with no on-topic seed won't reach the chain. Needs a reliable
  "previous session's notes" handle at learn time.

### 9. Context-clear handoff artifact  · [R][P]
At session end / before a context clear, the agent writes a structured handoff
(open threads, what I created, next steps); the next session's first recall
loads it. Fixes the problem exactly at the seam where it breaks — the clear
itself — like a human leaving themselves a note.
- **Fit:** precise to the failure mode; first-person handoff also addresses [P].
- **Risk:** depends on a reliable "before clear" hook; context clears are often
  abrupt/unhooked, so capture may not fire when it matters most.

### 10. Reflection / consolidation pass  · [R]
Periodically (or at closing `/learn`) synthesize recent raw events into a
higher-level "what I've been working on lately" note that recall surfaces.
Generative-Agents reflection; reuses the existing cluster-synthesis machinery.
- **Fit:** produces durable, queryable continuity narrative; leverages
  machinery that already exists.
- **Risk:** costs LLM calls; synthesis lag means the *most recent* events (the
  ones in the symptom) may not be consolidated yet — weakest exactly where the
  symptom bites.

---

## Comparison

| # | Approach | Problem | Cost / blast radius | Engram-fit | Fidelity to symptom |
|---|----------|---------|---------------------|------------|---------------------|
| 1 | Recency-weighted scoring (age-decay / activation) | R | Low–med (schema + scorer) | High | Med–high |
| 2 | Working-memory window | R P | High (new container) | High (theory) | High |
| 3 | Rolling session-recap note | R P | Low (learn convention) | High | High |
| 4 | Authorship stamping + labels | P | Low–med (schema) | Med | High for P, low for R |
| 5 | Ingest GitHub issues | R P | Med (gh dep, network) | Med (breaks zero-dep ingest) | High for the literal case |
| 6 | Two-store CLS | R | High (architecture) | High (theory) | High |
| 7 | "Since last session" block | R | Very low (skill-only) | High | Med |
| 8 | Temporal-edge continuity (graph walk) | R P | Low–med (link convention) | High (reuses BFS) | Med (needs on-topic seed) |
| 9 | Context-clear handoff | R P | Med (needs hook) | Med | High at the seam |
| 10 | Reflection pass | R | Med (LLM cost) | Med | Low–med (lag) |

## Recommendation (a position to react to, not a final pick)

All ten stand on their own for evaluation; this is my read on value-per-cost, to
push against — three tiers, run in sequence:

- **Ship-now probe (do first, in order):** start with **#7** (skill-only "since
  last session" block) — it ships today and *tests the core hypothesis*, that
  foregrounding recent work is what's missing, before any schema commitment. If
  the probe holds, add **#3** (rolling first-person recap) to also nail [P]. Run
  #7 first to validate cheaply; layer #3 once the hypothesis survives.
- **Durable core:** **#1** (recency-weighted scoring) is the smallest change
  that makes recency a first-class retrieval signal everywhere, and **#2** (the
  working-memory window) is the theoretically complete version if #1 proves
  recency matters but ranking-only isn't enough.
- **Root-cause patch:** **#5** (ingest issues) only if "the artifact was
  invisible to engram" recurs beyond this one anecdote — otherwise it's a
  narrow coupling that #3's recap covers in spirit.

Whatever is chosen should be measured, not asserted — "did short-term memory
improve" wants the memory eval harness
(`docs/superpowers/specs/2026-05-29-memory-eval-harness-design.md`). Note that
harness is mid-rework (the scenario/calibration layer is being redone for the
self-seeding cold-vs-warm design, issue #642); clearing those blockers is a
prerequisite to gating any of these empirically, not a step to assume done.

## Open question for the operator

Which problem is the real target — **[R]** continuity of *content* (don't
re-explore, don't lose the thread) or **[P]** continuity of *identity* (know I
authored this)? The symptom showed both; the cheapest strong answers differ by
which one you weight.
