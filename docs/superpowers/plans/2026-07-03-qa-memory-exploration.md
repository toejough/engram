# Q&A memory exploration — investigation plan

> **For agentic workers:** design + 2–3 cheap probes ask (~$5–15 total). Deliverable is
> `docs/design/2026-07-03-qa-memory-proposals.md` — NO production build this round. No production
> changes, no live-vault writes, copy-vault only in all probe scripts.

**Ask (Joe, 2026-07-03, condensed):** think through Q&A-shaped memory: (1) remember
question+answer pairs as vault nodes; (2) link Q→A and answer→contributing-notes; (3) use
accumulated contribution links as a graded usage signal (repeatedly-cited notes are worth
more — scales better than binary activation); (4) crystallize answer-reasoning not already in
a note; (5) all serving a compounding knowledge graph. Answer whether we already acted on
"persisted reasoning helps lesser models" (PARTIALLY — see Facts §1 below).

## Three settled decisions (Joe, 2026-07-03 — recorded as SETTLED, no gate re-litigates)

| # | Decision | Joe's rationale |
|---|---|---|
| D1 | Usage signal feeds RETENTION + TRIAGE first (pruning/keep decisions, human inspection, Obsidian degree-as-usefulness). Ranking changes deferred — future delivery A/B may sketch but must not build. | "retention/triage first — don't optimize recall ranking before we know the signal is real" |
| D2 | Capture fires at SUBSTANTIVE-ANSWER moments: deep recall Step 4, /please closes, and ad-hoc answers that cite notes or crystallize novel reasoning. OBSERVABLE bar: answer cites ≥1 vault note OR crystallizes a new note. No agent-judged "was this valuable" gate. | vault note 145: observable predicates; agent-judged gates fail |
| D3 | This round: design + 2–3 cheap probes (~$5–15 total). Full delivery A/B is out of scope. | "design before building" |

## Facts (all labeled; all code-verified unless noted)

### Fact 1 — Did we act on "persisted reasoning helps lesser models"? (PARTIALLY)

The finding appears in two vault notes:

- **Note 76 (2026-06-23):** persist synthesis for inspection AND reuse by less capable models —
  three values independent of strong-model accuracy: inspection (auditable), correction (only
  fixable if a note exists), and weaker-model reuse (a model that can't re-derive recalls the
  pre-computed conclusion directly).
- **Note 135 (2026-06-28, measured):** sonnet + recalled memory FULLY MATCHES opus across C3/C4i/C6
  — memory democratizes reasoning across model tiers; route memory-backed reasoning to cheaper tier.

What shipped:
- **Tier-routing via the route skill** (note 135; commit mentioned in note 135 "Next: wire into
  the route skill") — SHIPPED. The model-routing half of the finding is live.
- **Deep recall Step 4** (synthesis note with `--chunk-source` provenance links) — SHIPPED. Sound,
  non-trivial conclusions are persisted as vault notes hedged by inference mode, linked to their
  chunk inputs.

What was never built:
- **Q&A nodes** — no `type: qa` kind exists. Pairs are not stored.
- **Contributor links** (answer→contributing-notes provenance) — `--chunk-source` links to CHUNKS,
  not to notes. A "notes cited in this answer" edge does not exist.
- **Usage counting** — no count field anywhere in the sidecar or frontmatter. Only `last_used:
  YYYY-MM-DD` exists (binary activation bumps the date, not a counter).

**Verdict: PARTIALLY.** The reasoning-democratization lever (tier-routing) and the
synthesis-persistence mechanism (Step 4) both shipped. The Q&A pair storage, note-level
provenance, and usage accumulation are unbuilt — this exploration covers exactly those three.

### Fact 2 — Activation mechanics (code-verified)

`engram activate` (internal/cli/activate.go:36–39) bumps `Sidecar.LastUsed` to today's date.
`LastUsed` is `string` (YYYY-MM-DD) in the sidecar struct (internal/embed/embedder.go:81).
**No count field exists anywhere** — binary activation is the current full state of "usage
signal." `engram amend --activate` calls the same underlying `bumpLastUsed`; no new binary
behavior.

### Fact 3 — Learn/amend flag surface (code-verified: internal/cli/targets.go)

`CommonLearnArgs` (shared by `learn fact`, `learn feedback`, `amend`):
- `--supersedes <basename>|<type>|<claim>` (repeatable; types: updates/narrows/refutes)
- `--chunk-source <source#anchor>` (repeatable; chunk provenance)
- `--source` (required; human-readable provenance string)

No `--contributors` flag. No `--question` or `--answer` flag. No `type: qa` kind is recognized
by any subcommand. A new `engram learn qa` subcommand OR a `--kind qa` flag would be net-new
binary work.

### Fact 4 — Vocab-kind exclusion seam (code-verified; three-point, all required)

`isVocabKind(content string) bool` (internal/cli/vocab.go:165–168) checks the frontmatter
`type:` field for `"vocab"` or `"vocab-index"`. `isVocabKindFilename(name string) bool`
(vocab_commands.go:841–843) checks `strings.HasPrefix(name, "vocab.")`.

Applied at three mandatory exclusion points in the query pipeline:
- **Point A** (query.go:435, 1084): pre-clustering filter — vocab notes never enter any cluster.
- **Point B** (query.go:846): matched-set floor/cap (`!isVocabKind(item.note.content)`) — vocab
  notes do not receive a floor guarantee and are excluded from the matched set.
- **Point C** (query_nominations.go:95): tag-nomination gate — vocab notes are never nominated.

A `type: qa` QA kind requires the same three-point exclusion. The minimal implementation is:

```go
func isExcludedKind(content string) bool {
    kind := kindFromContent(content)
    return kind == typeVocab || kind == typeVocabIndex || kind == typeQA
}
```

replacing the two `isVocabKind` call sites at Points A/B/C. `isVocabKindFilename` is a SEPARATE
gate (vocab_commands.go scan loops); a `qa.` filename prefix convention would need the same.
**Alternatively**: no special filename prefix for QA notes (numbered like regular notes); only
the `type:` field gate is required. This is cheaper and sufficient — the filename check is only
needed for commands that scan the vault by file (vocab stats, centroids, trigger) which don't
need a QA-kind gate.

### Fact 5 — vaultgraph InDegree capability (code-verified: internal/vaultgraph/graph.go)

`Graph.InDegree(basename) int` — returns count of notes that wikilink TO basename (from any
source). Already implemented; returns 0 for unknown basenames.

`Graph.InDegreeIn(basename string, subset map[string]struct{}) int` — restricts to a subset of
source notes. **This is the key function for usage counting**: pass the set of all QA-node
basenames as the subset to count only contribution citations, not all wikilinks.

No new infrastructure needed. Usage counting = `InDegreeIn(noteName, qaNodeSet)` at stats time.

### Fact 6 — qanchor eval harness location (code-verified)

All harness files live at `dev/eval/traps/`:
- `qanchor_eval.py` — 3-stage eval (crystallize → headroom → deliver); the 10 qanchor corpus pairs
- `qanchor_retrieval_probe.py` — retrieval-channel FREE probe (no LLM; seeds temp vault, runs
  `engram query`, compares cosine scores)
- `qanchor_corpus.py` — the 10 Q+A+evidence PAIRS corpus (reusable)
- `qanchor_score.py` / `test_qanchor.py` — scoring logic (pure functions)
- `retrieval_probe.py` — general-purpose retrieval probe pattern (seed vault → query → rank check)

The 48-case miss population is at `dev/eval/links/misses_p1.json` + `bridges_p2.json` +
`p3_baselines.json` (from the link-value exploration S2; this is the population that powered
the vocab-notes slice-2 and slice-3 gates).

### Fact 7 — Vault scale (measured 2026-07-03)

168 total .md files: 142 non-vocab notes + 26 vocab term/index notes (measured:
`ls ~/.local/share/engram/vault/*.md | wc -l = 168`; vocab prefix count = 26).

### Fact 8 — Free-form relation edges retired (measured/settled 2026-07-02/03)

`--relation` flag and `migrateRelationLinks` REMOVED (vocab-notes build slice 1, commit
537d4a1a). 84 inventoried relation edges → 7 typed supersession edges kept, 76 dropped, 1
dangling (build results doc). Archive: `docs/design/artifacts/2026-07-02-retired-relation-rationales.md`.

**Distinction this plan must maintain:** contribution edges (one Q&A node → one cited note,
event-shaped, 1:1 per citation event) differ structurally from the retired topical links
(many-to-many topic associations, no event anchor, no consumer named). Contribution edges name
their consumers up front: usage counting (retention/triage, D1) + human inspection via Obsidian
in-degree. The retirement precedent does not apply here — the retired edges had no consumer.

### Fact 9 — Attribution confabulation risk (from vault notes 145/148/162)

Note 145: observable predicates gate (agents fail on agent-judged "was this valuable"). Notes
148/162: agents confabulate when free-listing attribution ("which notes contributed?").
**Implication:** contributors must be derived from notes CITED IN THE WRITTEN ANSWER/SYNTHESIS
text, not free-listed by the agent at close. The capture bar (D2: "cites ≥1 vault note" =
wikilinks appear in the answer text) makes attribution near-observable — an agent that writes
`[[note-basename]]` in an answer body has provided a durable, verifiable citation.

## Design dimensions (2–3 options + lean each)

### Dim A — Q&A node shape

The node stores question + answer + metadata as a single vault note with `type: qa`.

| Option | Shape | Lean |
|---|---|---|
| A1 — Unified note | One note per Q&A pair; frontmatter: `type: qa`, `question: "..."` (verbatim question text), `date: YYYY-MM-DD`, `certainty: high\|medium\|low`; body: the answer text + `[[contributor-basename]]` wikilinks at close | **CONTENDER** |
| A2 — Split Q/A | Separate Q note + A note linked by a `answers:` frontmatter field | PARK — doubles file count, complicates embedding (Q and A are complementary, not independent), adds link maintenance without clear retrieval benefit |

**Lean: A1.** The qanchor parked finding (note 153) shows Q-vocab loses retrieval — but that
applies to notes COMPETING in the matched set via cosine. QA nodes are EXCLUDED from the
matched set (vocab-kind seam precedent, Fact 4). Their retrieval value flows through
`[[contributor]]` wikilinks (ride-along; free from supersession machinery) and the human
Obsidian graph, NOT from cosine competition. A single note embeds the full Q+A context, making
the node semantically richer for future non-cosine use (e.g., LLM citation lookup).

**Frontmatter template:**
```yaml
type: qa
question: "<verbatim question text>"
date: "YYYY-MM-DD"
certainty: high|medium|low
contributors: [basename1, basename2]  # frontmatter channel (machine-readable)
vocab: [...]  # auto-assigned at write time
```

Body closes with `[[basename1]] [[basename2]]` (body channel; Obsidian in-degree + vaultgraph
InDegree).

### Dim B — Edge channels and inverse tracking

Contributors flow note→QA-node (forward: "this note contributed to this answer").

| Option | Channel | Inverse | Lean |
|---|---|---|---|
| B1 — Dual-channel | Frontmatter `contributors:` list + body wikilinks (vocab precedent: machine + human) | `InDegreeIn(note, qaNodeSet)` from vaultgraph body wikilinks — FREE, no new state | **CONTENDER** |
| B2 — Frontmatter only | `contributors:` list in frontmatter, no body wikilinks | Requires scanning all QA frontmatter at stats time (no graph leverage) | PARK — misses Obsidian degree-as-usefulness; vaultgraph's InDegreeIn is free if body wikilinks exist |

**Lean: B1.** Dual-channel is the vocab precedent (shipped 2026-07-02, slice 1) and costs zero
extra code — `InDegreeIn(noteName, qaNodeSet)` is already implemented (Fact 5). Obsidian shows
in-degree automatically from body wikilinks. The frontmatter list is the machine-readable
failsafe (readable without parsing body text). The binary does NOT need to maintain an explicit
`used_in:` inverse — vaultgraph computes it on demand from body wikilinks.

### Dim C — Capture mechanics per moment

Three D2-sanctioned capture moments, each implying SKILL.md edits (each edit = writing-skills
TDD, counted as implementation cost in the proposals table).

| Moment | Current behavior | Extension | SKILL.md edit required |
|---|---|---|---|
| Deep recall Step 4 | Writes a synthesis note via `engram learn fact\|feedback --chunk-source` | After writing the synthesis note, `engram amend --target <note> --contributors <cited-basenames>` OR extend `engram learn` with `--contributors` flag; add `[[contributors]]` to body | learn SKILL.md (Step 4 block) — 1 TDD cycle |
| /please closes | No capture hook today | Add a capture block at close: "if answer cites ≥1 [[note]], run `engram learn qa ...`" | please SKILL.md (close block) — 1 TDD cycle |
| Ad-hoc `engram learn qa` | Does not exist | New subcommand (net-new binary work; NOT this round) | N/A |

**Lean (this round):** describe all three; build none. The proposals doc sketches the binary
change needed for `engram learn qa` (the new subcommand / flag surface) so the next round can
scope it precisely. The skill edits (learn + please) are cheap and reversible; they are the
first candidates for the follow-up build.

**Cost estimate (skill edits only, deferred):** 2 SKILL.md edits × ~$1–2 each (RED/GREEN/pressure
test) = ~$2–4. Binary `--contributors` flag or new `learn qa` subcommand = $10–30 (DI, tests,
TDD). Total follow-up build: ~$12–34 (estimated; not metered).

### Dim D — Usage-count derivation

| Option | Mechanism | Lean |
|---|---|---|
| D-rt — Derived at read time | At stats/report time: build vaultgraph, call `InDegreeIn(noteName, qaNodeSet)` for every non-QA note; return sorted list. Zero new sidecar/frontmatter state. | **CONTENDER** |
| D-ps — Persisted counter | Add `contributor_count: N` to sidecar; bump on every new QA node that cites the note | PARK — new sidecar field, drift risk (QA node deleted → count stale), requires scan on every learn-qa write |

**Lean: D-rt.** `InDegreeIn` is free (Fact 5). Derived-at-read-time is the correct architectural
choice: the count IS the graph state; persisting it creates a redundant copy that can drift. The
only cost is building the vaultgraph once at stats time (~negligible, already done by
`engram check`).

**Consumer form (deferred build, describe now):** a new `engram usage report` command (or an
addition to `engram vocab stats`) that prints, per non-QA non-vocab note, its contribution
in-degree across all QA nodes. Output: sorted by in-degree descending, with the note's
luhmann ID, basename, and count. Feeds: triage (zero-count notes are prune candidates),
human inspection (Obsidian alternative: raw in-degree on the graph view), and future retention
rules (a follow-up after the signal is validated).

### Dim E — Retention/triage consumer (deferred; shape now)

| Option | Form | Lean |
|---|---|---|
| E1 — `engram usage report` | New command; prints per-note contribution in-degree, sorted | **CONTENDER (deferred)** — describes the consumer; does not build it this round |
| E2 — Integrated into vocab stats | Add a `top contributors` section to `engram vocab stats` output | PARK for now — vocab stats is already complex; keep concerns separate |
| E3 — Human Obsidian only | No binary consumer; Obsidian in-degree from body wikilinks is the entire signal | PARK — loses the sorted-by-count view that makes triage actionable; vaultgraph could compute it for free |

**Lean: E1 (deferred).** Describe the interface in proposals; build in the follow-on delivery
round after P3 (usage-distribution dry-run, below) validates the signal has spread.

## Probes (pinned; budgeted; ~$5–15 total; pass/fail stated before running)

### P1 — Retrieval-pollution probe (cost: ~$0–1, estimated; FREE — embedder only)

**Claim under test:** injecting N synthetic QA nodes into the vault without the qa-kind
exclusion WILL pollute the matched set (QA nodes surface on cosine, displace substantive notes).
With the exclusion at Points A/B/C (Fact 4), zero baseline disturbance.

**Pass/fail (pre-registered):**
- PASS: with `isExcludedKind` covering `type: qa`, the 48-case miss population + C3–C6 trap
  suite shows 0 new misses vs the baseline; QA nodes appear nowhere in `items[]`.
- FAIL: QA nodes surface in `items[]` OR a previously-surfaced note drops out of the top-5 → the
  qa-kind exclusion gate has a hole.
- PARTIAL-FAIL: some QA nodes surface via nomination (`tag_nominations_added`); check Point C
  exclusion in `query_nominations.go:95`.

**Harness:** `dev/eval/qa/p1_retrieval_pollution.sh` (committed before running). Safety guards
follow the Task-11 pattern (O2 build plan, 2026-07-03):

```bash
set -u
LIVE_VAULT="${ENGRAM_VAULT_PATH:-${XDG_DATA_HOME:-$HOME/.local/share}/engram/vault}"
WORK_DIR=$(mktemp -d)
COPY_VAULT="$WORK_DIR/qa-pollution-probe-vault"
cp -r "$LIVE_VAULT" "$COPY_VAULT"
export COPY_VAULT WORK_DIR

# Inject N=5 synthetic QA nodes (type: qa, contributors: [...real-note-basenames...])
# into $COPY_VAULT using a Python heredoc that writes raw .md files without the binary
# (no --kind qa flag exists yet; probe tests the FUTURE seam, not the current binary).

# Arm 1: WITHOUT exclusion (patch isExcludedKind to never exclude qa) — expect pollution
# Arm 2: WITH exclusion (patch isExcludedKind to include qa) — expect 0 disturbance

# Replay: for each of the 48 miss cases in dev/eval/links/misses_p1.json, run:
#   ENGRAM_VAULT_PATH="$COPY_VAULT" engram query --lazy-chunks --phrase "..." [x10]
# Check: no QA node in items[]; no baseline disturbance in top-5

# Arm 2 requires a PATCHED binary (the exclusion gate doesn't exist yet).
# Alternative without binary changes: inspect raw query output for type:qa in content.
# The free proxy: run Arm 2 WITHOUT the binary patch but verify no qa-type note surfaces
# by grepping items[].content for "type: qa" — valid because the qa nodes have no sidecar
# embedding yet (no --contributors flag = no learn-qa path = no sidecar = items[] can't
# score them). This makes P1 truly free (embedder never sees the synthetic nodes).
```

**Note on no-sidecar synthetic nodes:** synthetic `.md` files written by the probe without an
accompanying `.vec.json` sidecar will have no embedding and cannot enter the matched set.
P1 therefore measures: (a) that the future BINARY exclusion is designed correctly (describe the
gate, don't implement it in this probe); (b) that the RETRIEVAL PROBE BASELINE (48 cases) is
stable on the copy vault. Cost: zero LLM calls. Estimated: $0.

**Revised P1 scope:** since the binary change doesn't exist yet, P1 becomes a BASELINE
STABILITY check: run the 48-case miss population + C3–C6 trap suite on the copy vault with
5 synthetic QA .md files present (no sidecars → no scoring surface). Confirms the baseline
before any build. Pass = zero disturbance to today's 48-case results (or documents today's
delta as the new reference point). This is the right pre-build measurement.

### P2 — Attribution fidelity probe (cost: ~$3–8, estimated)

**Claim under test:** cite-derived attribution (extract `[[basename]]` wikilinks actually
written in the answer text) is more accurate and less confabulating than free-listed attribution
(agent enumerates "what notes did you use?" at close). Ground truth: human reading of each
session, identifying which vault notes were load-bearing.

**Pass/fail (pre-registered):**
- PASS: cite-derived confabulation rate < 20% (false positives: cited but not actually used) AND
  free-list confabulation rate > 30%. Separation of ≥15pp confirms cite-derived is the right
  channel.
- FAIL: both methods confabulate equally (both <20% or both >30%) → revise D2's capture bar.
- BORDERLINE: cite-derived recall (coverage of actually-used notes) < 50% → the bar "answer cites
  ≥1 vault note" misses too many contributors; consider enrichment step.

**Harness:** `dev/eval/qa/p2_attribution_fidelity.py`. Steps:
1. From session transcripts in `~/.claude/projects/-Users-joe-repos-personal-engram/`, select
   ~10 deep recall Step-4 synthesis events (where a vault note was written). These are
   identifiable by `engram learn fact|feedback` calls in transcript turns.
2. For each: extract (a) the notes cited in the synthesis body via `[[...]]` pattern, (b) have
   a fresh sonnet agent free-list which notes it would credit, (c) human reads the synthesis
   session context and judges ground truth (which notes were genuinely load-bearing).
3. Compute: confabulation rate (cited or free-listed but not in ground truth), recall rate
   (ground truth notes captured by each method).
4. Report as a table per step 3's pass/fail criteria.

**Cost estimate:** ~10 synthesis sessions × 1 sonnet call each (~$0.30/call) + human review
= ~$3–5 (estimated; not metered). Upper bound $8 if 20 sessions.

### P3 — Usage-distribution dry run (cost: ~$2–4, estimated; optional if budget exceeded)

**Claim under test:** if Q&A nodes existed today and captured the last N months of answered
questions, would the contribution in-degree signal have SPREAD (some notes cited 5+ times, most
0) or would it be flat (every note cited equally)? A flat distribution means the signal has no
discriminating power for triage. A spread distribution (power-law or Pareto) means it IS a
useful retention signal.

**Pass/fail (pre-registered):**
- PASS: top-10% of notes by would-be in-degree receive ≥3× the median in-degree — the signal
  has spread; the retention/triage consumer (E1) is worth building.
- FAIL: would-be in-degree distribution is flat (CV < 0.5) — the signal is uninformative; defer
  E1 until more Q&A nodes exist naturally.
- INFORMATIVE-NULL: fewer than 20 Q&A-eligible exchanges found in transcripts → usage
  distribution underpowered; note the floor and defer.

**Harness:** `dev/eval/qa/p3_usage_distribution.py`. Steps:
1. Scan `~/.claude/projects/-Users-joe-repos-personal-engram/` transcripts for turns where the
   agent wrote a vault note (`engram learn ...`) AND the surrounding context cited ≥1 vault note
   by name (`[[basename]]` pattern in ASSISTANT turns).
2. For each such turn: extract the cited basenames as "would-be contributors."
3. Count per-note would-be contribution citations across the corpus.
4. Report: top-10 notes by count, total count, CV, Pareto fraction.

**Cost estimate:** embedder-free transcript scan; one LLM pass to identify Q&A-eligible turns
from ambiguous cases → ~$2–4 (estimated). No vault writes.

## Steps (this round only — ends at PRESENT TO JOE)

**Step 0 — Commit this plan** to `docs/superpowers/plans/2026-07-03-qa-memory-exploration.md`.

**Step 1 — Run probes in parallel** (P1 baseline + P2 attribution fidelity; P3 optional):
- P1: `dev/eval/qa/p1_retrieval_pollution.sh` — baseline stability check (~free)
- P2: `dev/eval/qa/p2_attribution_fidelity.py` — ~$3–8
- P3 (if ≤$5 left in budget): `dev/eval/qa/p3_usage_distribution.py` — ~$2–4
- All probe scripts committed under `dev/eval/qa/` BEFORE running (no inline scripts).

**Step 2 — Interpret results against pre-registered pass/fail criteria.** Do NOT re-derive
criteria post-hoc.

**Step 3 — Write `docs/design/2026-07-03-qa-memory-proposals.md`** containing:
- Answer to "did we act on persisted reasoning?" (this plan's §Facts Fact 1)
- Probe results as labeled tables (columns: metric, units, arm A, arm B, delta)
- Design option table for each Dim A–E (option / contender-or-park / rationale / build cost estimate)
- Recommended build sequence (lean: skill edits first [learn+please Step 4], binary learn-qa
  subcommand second, usage-report third — each gated on prior round's validation)
- Follow-up A/B design sketch for ranking (D1-deferred): what would falsify "usage signal
  improves recall precision"? What arms, what n, what metric? (sketch only — not a build plan)

**Step 4 — PRESENT TO JOE AND STOP.** No build. No live-vault writes. No production changes.

## Constraints (non-negotiable)

- **No production changes** this round. No live-vault writes. No binary changes. No skill edits.
- **Copy-vault only** in all probe scripts. `set -u` + explicit `LIVE_VAULT` resolution +
  `COPY_VAULT` as a separate `mktemp -d` subtree. Pattern: Task-11 harness in
  `docs/superpowers/plans/2026-07-03-vocab-lifecycle-o2-build.md`.
- **All numbers labeled** measured / estimated / projected.
- **Three settled decisions recorded** above (D1/D2/D3) — no gate re-litigates them.
- **Q-shaped wording constraint** (note 153, measured: retrieval lost 10/10): QA nodes MUST be
  excluded from the cosine matched set via the three-point vocab-kind exclusion seam analog.
  Their retrieval value flows through wikilinks and Obsidian in-degree, NOT cosine competition.
- **No retrieval-time traversal or reranking** in any option this round (3 consecutive A/B
  negatives, note 73). Contribution edges are write-time data, not a new retrieval scaffold.
- **Free-form relation edges are retired** (2026-07-02; note the distinction: contribution edges
  name their consumer up front and are event-shaped, unlike the retired topical links).
- **Agent-judged gates prohibited** (note 145): all capture triggers must be observable predicates
  (wikilink exists in body text; note was written).
- **Attribution must be cite-derived** (notes 148/162): contributors = notes cited in written
  answer body, not free-listed by the agent at close. P2 validates this channel before any build.
- **Deliver the full option set** (all Dim options with honest CONTENDER/PARK ratings; never prune
  to the favorite before Joe sees the table).
- **Labeled tables with units** for all probe results; arms side by side.
- **Probe scripts committed before running** — no inline scripts in the proposals doc.
