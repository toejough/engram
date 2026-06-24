# Split `recall` into a host (opus) skill + a haiku retrieval sub-skill

> Implements Lever 1 at the judgment seam (note 78), not bespoke infra. Two SKILL.md artifacts →
> **two `superpowers:writing-skills` RED→GREEN→REFACTOR cycles** (Iron Law per skill).

**Goal:** Move the expensive, low-judgment half of `recall` (sweep, query, read the ~49K payload,
coverage judgment + content writes) onto a **haiku** sub-skill the host dispatches, while the host
(opus) keeps the relation-gating + plan reasoning. Recall *procedure* is the cost (note 77); this
moves it to a ~5× cheaper model without putting the host's reasoning on it.

## The seam (decomposed; locked unless refuted)

| step | model | rationale |
|---|---|---|
| 0 print plan · 1 generate 10 phrases | **host/opus** | plan-coupled, cheap; keeps host context off haiku |
| 0.5 sweep · 2 query · **read the 49K payload** | **haiku** | the expensive read + pure mechanics |
| 2.5-A coverage judgment (covered/near/absent, recency-weighting) | **haiku** | ≈ recency-conflict judgment, which haiku passed (C4 0 mis-rankings) — **measured, not assumed** |
| 2.5-B note **content** writes (amend/learn subject/predicate/object; `--chunk-source` provenance only, **NO `--relation`**) | **haiku** | content crystallization; the relation gating is removed from here |
| 2.6 + within-cluster **relation** gating (all `--relation` edges, the hub test) | **host/opus** | the heaviest reasoning; SKILL.md:145 couples 2.5's relations to the 2.6 gate, so **all relations live on the host** — this is the clean cut |
| 3 plan-synthesis · 4 persist conclusion · 2.7 activate | **host/opus** | plan reasoning; operates over the SMALL returned note set |

**Why the relation move (resolves the 2.5↔2.6 coupling):** the current skill has 2.5 emit
`--relation` flags "for every note source that passes the Step 2.6 precision gate" — i.e. 2.5's
relations are *defined by* 2.6. So haiku must NOT write relations; it writes note **content** only, and
the host runs ALL relation gating (within- and cross-cluster) in an expanded 2.6 over the returned
note set. Cleaner seam, and it keeps the flood-prone link reasoning entirely on opus.

**Why coverage (2.5-A) on haiku is a *measured bet*, not an assumption:** coverage ≈ the
recency-conflict judgment haiku passed (C4). It *might* not generalize to comprehension-style coverage
— so the eval's note-quality spot-check (below) gates it; if it fails, 2.5-A moves to the host in v2.

## Return contract (sub-skill → host) — concrete format

The sub-skill's final message is a single fenced ```yaml block, **verbatim content, no editorializing**:

```yaml
surfaced_items:            # the material the host reasons over (Step 3) — full content, not summaries
  - path: <basename>
    kind: fact|feedback|chunk
    content: "<verbatim>"
note_members:              # surfaced NOTE members across clusters, for the host's 2.6 relation gating
  - basename: <luhmann-prefixed>
    content: "<verbatim>"
coverage:                  # 2.5-A verdicts, for the host's 2.7 activation
  covered: [<basename>, ...]
  near:    [<basename>, ...]
written_notes:             # 2.5-B content writes haiku made (for host audit + activation)
  - basename: <…>
    action: created|amended
    change: "<one-line>"
```

The host parses this; it must NOT need the 49K payload. If the host finds itself re-reading the
payload, the seam failed (see RED).

## Files

- Modify: `skills/recall/SKILL.md` → **host** (0, 1, dispatch, 2.6+relations, 3, 4, 2.7).
- Create: `skills/recall-retrieve/SKILL.md` → **haiku sub-skill** (0.5, 2, read, 2.5-A, 2.5-B).
- `skills/recall/tests/` → new baseline + pressure scenarios (below).
- Auto-synced by `engram update` (verified: `planSkillCopies` lists all files under `skills/` — no Go
  change). **Verify** the eval warm-cfg builder (`build_warm_cfg` in `dev/eval/traps/wrun.py`) copies
  `recall-retrieve` into the cfg, or delegation silently won't fire in the harness.

## Wiring (verified capability)

The host's dispatch step issues an **Agent-tool call with `subagent_type: general-purpose, model:
haiku`** and a directive prompt: "run the `recall-retrieve` skill with these 10 phrases; return its
YAML contract." The Agent tool's `model` override applies to **subagents** — the main-loop model lock
(route skill) does NOT apply to dispatched subagents, so this is a real capability, not a workaround.
Because the split lives in the skill, any caller running the deployed host skill on opus auto-delegates
retrieval to haiku — confirm the criterion harnesses invoke the real deployed skill (they build a warm
cfg from `skills/` and run `/recall`), not a stubbed/inline recall.

## Execute — TWO writing-skills cycles (Iron Law per skill)

**Cycle 1 — host `recall/SKILL.md`:**
- [ ] **RED (falsifiable).** Give a fresh opus agent the *draft host skill minus the binding language*
  on a real recall scenario; observe the two failures the skill must drive out: **(b)** the host
  **re-reads the 49K payload itself** instead of trusting the sub-skill's YAML, and **(c)** the host
  does retrieval inline rather than dispatching. Document the observed violations.
- [ ] **GREEN.** Write the host skill (dispatch + relation-gating + 3/4/2.7). Re-run: host dispatches
  haiku, never re-reads the payload, reasons only over the returned YAML.
- [ ] **REFACTOR + pressure.** ≥1 pressure scenario per failure: host under *cost/time pressure* "just
  peeks" at the payload to save a round-trip → skill must forbid it (prohibition + red-flag). Gate B.

**Cycle 2 — new `recall-retrieve/SKILL.md`:**
- [ ] **RED (falsifiable).** Give a fresh haiku agent the *draft sub-skill minus the contract binding*;
  observe the wrong-shape failure: it **editorializes / pre-digests / plan-synthesizes** instead of
  returning the raw YAML contract. Document.
- [ ] **GREEN.** Write the sub-skill (0.5/2/read/2.5-A/2.5-B + the YAML return). Re-run: returns
  verbatim material, no synthesis, no relations written.
- [ ] **REFACTOR + pressure.** Pressure: sub-skill told to "be helpful" → must still return raw, not a
  summary (recipe/contract form, not prohibition — per Match-the-Form). Gate B.

- [ ] **Reconcile existing baselines.** Re-run the existing `skills/recall/tests/` baselines
  (recency-conflict, empty-vault-skip, multi-query, bootstrap-create) against the split — coverage now
  crosses the model boundary; they must still pass or be consciously updated.

## Measure (the real L1 test) — concrete verdict

Deploy (`engram update`); run C3/C4/C5 warm on **opus host** (auto-delegates retrieval to haiku), n=5
→ n=10 if it holds. **Pass = ALL of:**
- C4 warm-XXp supersession ≥ **4/5**, **0 wrong-direction mis-rankings** (baseline opus 5/5).
- C5 honored ≥ **4/5** (baseline 5/5).
- C3 applied ≥ **23/25** (baseline 25/25).
- Cost (host op) < **0.6× the whole-opus baseline** (retrieval moved to haiku; conservative — recall is
  a sub-share of the op, so a full 5× recall cut ≠ 5× op cut).
- **Note-quality spot-check:** read **n=10** haiku-written 2.5-B notes from the runs; FAIL if ≥3 have a
  semantic error in subject/predicate/object vs their cluster members, or straw-man the principle. On
  fail → move 2.5-A/B to the host (v2), haiku still retrieves + reads.

Report **both axes**; do not crown on cost alone if any quality gate slips (metric-sensitivity).

## Caveats

- The cost win is real ONLY if the host stays off the 49K payload — gate B must *confirm* this, not
  assume it (the single point of failure for the lever).
- Coverage + crystallization on haiku is the measured bet; the spot-check + C3-applied gate it.
- v1 keeps ALL relations + 2.6/3/4 on opus (the conservative reasoning placement).
