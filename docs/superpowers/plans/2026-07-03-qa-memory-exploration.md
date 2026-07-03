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
| D4 | Q and A are SEPARATE notes with an explicit Q→A edge (full-basename wikilink + frontmatter). Joe chose this OVER the plan's containment lean (Gate A escalation, 2026-07-03): the question is a first-class graph node, accepting the file-count/split-embedding cost. | "separate Q and A notes" |
| D5 | Past Q&As get a recall path via a SEPARATE CHANNEL (future build): incoming ask matched against Q-note embeddings in their own space — like the chunk space — top-1–2 appended to the payload as a distinct section. Additive tokens only; Q/A notes NEVER compete in the main matched set. Obsidian-only was offered and declined. | "a future similar question should surface the past answer" |

**What this delivers for Joe (the point, up front):** every substantive answer becomes a Q-note
+ A-note pair in the graph, the A-note wikilinked to the notes that actually fed it. Over time:
(a) a future similar question surfaces the past answer through the Q-channel (D5); (b)
`engram usage report` shows which notes actually get USED — repeatedly-cited notes are proven
keepers, never-cited notes are prune candidates — a graded signal where activation is binary;
(c) reasoning invented during an answer is crystallized before it evaporates.

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

Applied at FOUR mandatory exclusion points in the query pipeline (Gate A found the fourth):
- **Point A** (query.go:435, 1084): pre-clustering filter — vocab notes never enter any cluster.
- **Point B** (query.go:846): matched-set floor/cap (`!isVocabKind(item.note.content)`) — vocab
  notes do not receive a floor guarantee and are excluded from the matched set.
- **Point C** (query_nominations.go:95): tag-nomination gate — vocab notes are never nominated.
- **Point D** (query_nominations.go:337): the TermIndex BUILDER — without this point, a QA note
  carrying auto-assigned `vocab:` terms (the template mandates them) enters the TermIndex and
  gets nominated into `candidate_l2s` despite Points A–C.

QA kinds require the same exclusion at ALL FOUR points. The minimal implementation is:

```go
func isExcludedKind(content string) bool {
    kind := kindFromContent(content)
    return kind == typeVocab || kind == typeVocabIndex || kind == typeQAQuestion || kind == typeQAAnswer
}
```

replacing EVERY `isVocabKind` call site in the query pipeline (grep for all of them — the four
points above are the census as of 6fbe0612; a build must re-grep). `isVocabKindFilename` is a SEPARATE
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

### Dim A — Q&A node shape (SETTLED by Joe as D4: split notes)

| Option | Shape | Status |
|---|---|---|
| A1 — Unified note | One note per pair; Q in frontmatter, A in body (containment) | Was the plan's lean; **OVERRIDDEN by Joe (D4, Gate A escalation)** — containment demotes the question to metadata; Joe wants it as a first-class graph node |
| A2 — Split Q/A | Separate Q note + A note, explicit Q→A edge | **SETTLED (D4)** |

**Design consequence that makes A2 shine (the qanchor inversion):** note 153's finding —
question-shaped wording loses retrieval — applies to notes competing with CONTENT notes in one
cosine space. In a dedicated Q-space (D5's channel matches incoming asks against Q-note
embeddings ONLY), question wording is the *matching asset*: like matches like. The Q note's
embedding is purely the question; the A note's embedding is purely the answer. P1's value arm
measures exactly this (paraphrased-question → right Q note).

**Templates (both kinds excluded at all four seam points; filename prefix `qa.` for the scan
loops):**

Q note — `qa.<date>.<slug>.q.md`:
```yaml
type: qa-question
date: "YYYY-MM-DD"
answered_by: qa.<date>.<slug>.a   # FULL basename, no .md
```
Body: the verbatim question text, then a machine-written edge line:
`Answered by: [[qa.<date>.<slug>.a]]`

A note — `qa.<date>.<slug>.a.md`:
```yaml
type: qa-answer
date: "YYYY-MM-DD"
answers: qa.<date>.<slug>.q       # inverse edge, FULL basename
certainty: high|medium|low
contributors: [100.2026-06-26.cost-and-usage-are-the-same-procedure-tax-lever, 145.2026-06-30.recall-value-gate-not-holdable-by-wording-naming-primes]  # FULL basenames, no .md
vocab: [...]  # auto-assigned at write time
```
Body: the answer text, then machine-written lines:
```
Answers: [[qa.<date>.<slug>.q]]
Contributors: [[100.2026-06-26.cost-and-usage-are-the-same-procedure-tax-lever]], [[145.2026-06-30.recall-value-gate-not-holdable-by-wording-naming-primes]]
```

**Link form is pinned — G0 constraint (c2-containers.md):** the vault's known G0 defect is
that bare-id wikilinks (`[[100]]`) do NOT resolve in vaultgraph's basename resolver (census:
138/171 links orphaned). Contributor links therefore MUST be full note basenames (filename
minus `.md`), the form that resolves today — the `[[vocab.<term>]]` links shipped this week
are the working precedent. Enforcement is by construction: the `Contributors:` body line is
MACHINE-WRITTEN by the binary at capture time (exactly like the `Vocab:` line — single writer,
replace-whole idempotency, excluded from BodyText/ContentHash), never hand-typed by the agent,
so the resolving form is guaranteed rather than hoped for. Notes are never renamed in practice
(resituate rewrites content, not filenames), so full-basename links do not rot.

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
| Deep recall Step 4 | Writes a synthesis note via `engram learn fact\|feedback --chunk-source` | Extend `engram learn` with a repeatable `--contributors <full-basename>` flag; the binary machine-writes the `Contributors: [[<full-basename>]], ...` body line (Dim A form). Flag values are AUTO-EXTRACTED by the skill from the full-basename wikilinks already present in the written synthesis/answer text — the agent does not free-list (Fact 9) | learn SKILL.md (Step 4 block) — 1 TDD cycle |
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

**Lean: D-rt.** `InDegreeIn` is implemented (Fact 5) and derived-at-read-time is the correct
architectural choice: the count IS the graph state; persisting it creates a redundant copy that
can drift. Honest cost correction (Gate A): `BuildGraph` has ZERO production callers today —
`engram check` uses `ScanVault`/`UnresolvedTargets`, not `BuildGraph`. The usage-report build
wires `ScanVault → BuildGraph → InDegreeIn` for the first time (three existing library calls,
small but net-new wiring — not "already done"). Cost remains O(notes × links), negligible at
vault scale. The count restricts to the qa-ANSWER node set (contributors live on A notes, D4).

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

### P1 — Retrieval-pollution probe (cost: ~$0, estimated — local embedder + local queries only)

**Claim under test:** EMBEDDED synthetic QA nodes without the qa-kind exclusion pollute the
matched set (surface on cosine, displace substantive notes); with the three-point exclusion
(Fact 4) they cause zero baseline disturbance. Synthetic nodes MUST get sidecars (via
`ENGRAM_VAULT_PATH=$COPY_VAULT engram embed apply` — local model, $0) or the probe is a
tautology: sidecar-less files can never enter the cosine set, proving nothing (Gate A finding).

**Four arms, single pre-registered criteria set:**
- **Arm 0 (re-baseline):** copy vault, no QA nodes, CURRENT binary → record fresh 48-case
  results (the recorded baselines predate this week's vault growth; re-baseline honestly).
- **Arm 1 (pollution measurement):** +5 embedded synthetic QA nodes, CURRENT binary (no
  exclusion exists) → MEASUREMENT, not a gate: report QA-node appearances in `items[]` and
  top-5 disturbances vs Arm 0. Expected >0; the magnitude quantifies the risk the exclusion
  prevents.
- **Arm 2 (the gate):** same vault, PROBE-ONLY WORKTREE-PATCHED binary — add the qa kinds to
  the exclusion at ALL FOUR points (Fact 4, incl. Point D query_nominations.go:337) in a git
  worktree, `go build` there, never merge (precedent: the vocab build's slice-3 "worktree
  binary on PATH" gates). PASS = QA notes appear NOWHERE in `items[]` (including
  `tag_nominations_added` — Points C/D) AND 0 top-5 disturbances vs Arm 0. FAIL = any QA
  surfacing or any displaced note → the seam design has a hole; name the leaking point.
- **Arm V (the value arm — D5's channel premise, ~$0):** for each of the 5 synthetic pairs,
  write 2 PARAPHRASES of its question (distinct wording, same intent — authored in the probe
  script, committed before running). Using the qanchor_retrieval_probe.py direct-cosine
  pattern (dev/eval/traps/), compute each paraphrase's cosine against ALL note embeddings in
  the copy vault (Q notes, A notes, content notes). PASS = ≥8/10 paraphrases rank their own
  Q note #1 among Q notes AND above every content note; FAIL = <6/10 → question-to-question
  matching is too weak to power the D5 channel, and the channel needs redesign before any
  build. This is the qanchor-inversion test: Q-wording should WIN in q-space.

**Harness:** `dev/eval/qa/p1_retrieval_pollution.sh` (committed before running; single shell
invocation, no cross-invocation env carry needed). Safety guards follow the Task-11 pattern:

```bash
set -u
LIVE_VAULT="${ENGRAM_VAULT_PATH:-${XDG_DATA_HOME:-$HOME/.local/share}/engram/vault}"
WORK_DIR=$(mktemp -d)
COPY_VAULT="$WORK_DIR/qa-pollution-probe-vault"
cp -r "$LIVE_VAULT" "$COPY_VAULT"
export COPY_VAULT WORK_DIR
[ -d "$COPY_VAULT" ] || { echo "COPY_VAULT missing — abort"; exit 1; }

# Inject N=5 synthetic Q/A PAIRS (10 files: qa.<date>.<slug>.q.md type: qa-question +
# qa.<date>.<slug>.a.md type: qa-answer, contributors as full basenames, machine-form body
# lines per Dim A) as raw .md files, then:
ENGRAM_VAULT_PATH="$COPY_VAULT" engram embed apply   # sidecars for the new nodes ($0, local)

# Arm 0/1: current binary. Arm 2: worktree-patched binary (probe-only; never merges):
#   git worktree add "$WORK_DIR/wt" && (patch Points A/B/C) && go build -o "$WORK_DIR/engram-qa" ./cmd/engram
# Replay all 48 cases in dev/eval/links/misses_p1.json per arm; diff items[] + top-5 vs Arm 0.
```

Executor verifies the exact `embed apply` flag surface before running (`engram embed --help`);
if apply skips new files without `--force`/`--stale` semantics, use whichever flag embeds the
5 new nodes — report the command used.

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

**Agent safety (note 160, measured 2026-07-03 — mandatory for P2 AND P3):** eval agents run
WITHOUT bypassPermissions, and transcript excerpts are payload-sanitized before injection —
strip or rewrite absolute paths into the real repo (`/Users/joe/repos/...`) so an agent cannot
follow them and execute tasks it was asked only to judge.

**Harness:** `dev/eval/qa/p2_attribution_fidelity.py`. Steps:
1. From session transcripts in `~/.claude/projects/-Users-joe-repos-personal-engram/`, select
   ~10 deep recall Step-4 synthesis events (where a vault note was written). These are
   identifiable by `engram learn fact|feedback` calls in transcript turns.
2. For each: extract (a) the notes cited in the synthesis body via `[[...]]` pattern, (b) have
   a fresh sonnet agent free-list which notes it would credit, (c) GROUND TRUTH — an opus judge
   reads the full session context with this pinned rubric: "a note is load-bearing iff removing
   its content would change the answer's substance; for each load-bearing note, name the
   specific claim it supplied." Output schema per case:
   `{case_id, load_bearing: [<full basenames>], claim_supplied: {<basename>: "<claim>"}}`.
   Ground truth is LABELED LLM-judged in all results; the orchestrator spot-checks 3 of the 10
   judgments against the raw transcripts before the rates are computed.
3. Compute: confabulation rate (cited or free-listed but not in ground truth), recall rate
   (ground truth notes captured by each method).
4. Report as a table per step 3's pass/fail criteria. NOTE (tier-specificity): the free-list arm
   uses sonnet because the PRODUCTION attributor would be a session agent of that tier — the
   probe measures the shipped contrast, not free-listing as an abstract method; label the
   finding tier-specific.

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

**Pre-sample gate (free, runs FIRST):** grep the transcript corpus for the eligibility pattern
and COUNT before spending: if <20 eligible moments exist, declare INFORMATIVE-NULL immediately
and defer P3 — do not spend the LLM pass on an underpowered corpus (Gate A finding).

**Harness:** `dev/eval/qa/p3_usage_distribution.py`. Steps:
1. Scan `~/.claude/projects/-Users-joe-repos-personal-engram/` transcripts for ASSISTANT turns
   containing an `engram learn ...` call AND, IN THE SAME TURN's text, ≥1 vault-note citation
   (full-basename `[[...]]` pattern or an explicit "note NNN" reference). Window = the same
   assistant turn only — no prior-turn or time-window context (pinned; Gate A finding).
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

- **No SHIPPED changes** this round: no live-vault writes, no merged binary changes, no skill
  edits. P1's Arm-2 worktree-patched binary is probe-only — built in a throwaway worktree, run
  against the copy vault, never merged (slice-3 worktree-binary precedent).
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
