# Split `recall` into a host (opus) skill + a haiku retrieval sub-skill

> Implements Lever 1 the right way (note 78): split at the judgment seam, not bespoke infra, not a
> wholesale delegate. SKILL.md changes → **`superpowers:writing-skills` TDD** (project rule).

**Goal:** Move the expensive, low-judgment half of `recall` (sweep, query, read the ~49K payload,
coverage curation) onto a **haiku** sub-skill the host dispatches, while the host (opus) keeps the
reasoning that needs its plan context. Recall procedure is the time/$ cost (note 77); this moves the
payload read + mechanical steps to a ~5× cheaper model without putting the host's reasoning on it.

## The seam (locked unless gate A refutes)

| recall step | model | rationale |
|---|---|---|
| 0 print plan; 1 generate 10 phrases | **host / opus** | plan-coupled, cheap; keeps the host's context off haiku |
| 0.5 sweep; 2 query; **read the 49K payload**; 2.5 coverage judgment + crystallize/amend writes | **haiku sub-skill** | the mechanical bulk + the expensive read; coverage ≈ recency-conflict judgment, which haiku passed (C4 0 mis-rankings) |
| 2.6 precision gate; 3 plan-synthesis; 4 persist conclusion; 2.7 activate | **host / opus** | heaviest reasoning (2.6 hub test) + plan reasoning; operates over the SMALL returned note set, not the payload |

**Why 2.5 on haiku (the cost/quality bet):** savings scale with how much judgment moves off opus. If
opus keeps coverage judgment it must re-read candidate content → erodes the win. Putting 2.5 on haiku
keeps opus off the 49K payload entirely (max savings); the risk is crystallization *write quality*.
**Gated:** the eval spot-checks haiku-written notes; if quality drops, move 2.5 crystallization to the
host in v2 (haiku still retrieves + reads).

## Return contract (sub-skill → host)

The sub-skill returns, **verbatim, not paraphrased** (note 78):
1. The surfaced lessons (fact/feedback content) + chunk evidence — the material the host reasons over.
2. The surfaced **note members** (basename + content) across clusters — so the host can run 2.6.
3. The notes it **wrote/amended** in 2.5 (basenames + what changed) — for the host's activation + audit.
4. The coverage verdicts (covered/near basenames) — host activates these in 2.7.

No editorializing: the host must reason over real material, not haiku's summary of it.

## Files

- Modify: `skills/recall/SKILL.md` → the **host** skill (steps 0, 1, dispatch, 2.6, 3, 4, 2.7).
- Create: `skills/recall-retrieve/SKILL.md` → the **haiku sub-skill** (steps 0.5, 2, read, 2.5).
  Auto-synced by `engram update` (`planSkillCopies` lists all files under `skills/` — no code change).
- Test scaffolding under `skills/recall/tests/` (baseline + behavioral, per writing-skills).

## Wiring

The host skill's dispatch step: **invoke a subagent with `model: haiku`** (the Agent/Task tool) whose
prompt is "run the `recall-retrieve` skill with these 10 phrases; return the digest per its contract."
The subagent runs the sub-skill on haiku and returns. Because the split lives in the **skill**, every
caller (please, evals, ad-hoc) auto-delegates retrieval to haiku when the host runs on opus — no
harness change.

## Execute (writing-skills TDD)

- [ ] **RED — baseline behavior test.** Dispatch a fresh agent the current `recall` skill on a scenario;
  observe it does the WHOLE procedure itself (no model split, opus reads the 49K payload). Document.
- [ ] **GREEN — split.** Rewrite `recall/SKILL.md` as the host (delegating retrieval); write
  `recall-retrieve/SKILL.md` as the haiku sub-skill. Verify a fresh host agent dispatches a haiku
  subagent for retrieval and keeps 2.6/3/4 itself.
- [ ] **REFACTOR / pressure-test.** Close loopholes: host must NOT re-read the payload itself; sub-skill
  must NOT do plan-synthesis or editorialize (returns raw); the contract is honored under pressure.
  Gate B on the diff.

## Measure (the real L1 test)

- [ ] Deploy the split (`engram update`); run C3/C4/C5 warm on **opus host** (the skill auto-delegates
  retrieval to haiku), n=5 → n=10 if it holds.
- [ ] Compare to the whole-opus baseline: **cost must drop** (retrieval on haiku) AND **quality must
  hold** (C4 0 mis-rankings, C5 surfaced/honored, C3 applied) within the noise floor.
- [ ] **Note-quality spot-check:** read a sample of haiku-written 2.5 notes; if crystallization degraded,
  record it and recommend moving 2.5 to the host (v2). Report both axes; do not crown on cost alone.

## Caveats

- The cost win is real only if the host stays off the 49K payload — verify the host never re-reads it.
- Crystallization quality on haiku is the bet; the spot-check + C3-applied gate it.
- v1 keeps 2.6/3/4 on opus (the heaviest reasoning) — the conservative reasoning placement.
