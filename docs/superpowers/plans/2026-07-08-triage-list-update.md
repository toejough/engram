# Plan — Update the ROADMAP triage list (2026-07-08)

> Rev 2 — incorporates Gate A findings (clarity: exact old→new edits + size tags + FEATURES voice;
> docs-alignment: drop the "count-consumer cluster" umbrella, describe each new issue by its real
> repo-vocabulary home, fix the #676 mischaracterization; ask-alignment: make the L16–41 vs L270
> scope boundary explicit).

## Ask (verbatim intent)

"We had a fresh triage of our issues and roadmap, and then we just added several new issues.
Update our triage list and present findings."

## Interpretation

- **"Our triage list"** = the **"Current priorities (2026-07-07 triage)"** snapshot in `docs/ROADMAP.md`
  (L16–41). Confirmed by grep (`triage` lives only in ROADMAP + a historical GLOSSARY back-reference to a
  deleted `triage.md`) and by commit `0a95bbbb docs(roadmap): current-priorities triage + correct #657 gate`.
- **"Several new issues just added"** = **#674, #675, #676**, all created 2026-07-08 19:32Z, all
  follow-ons to the just-shipped `engram count` feature (verified as the only issues created after the
  2026-07-07 baseline).

## Verified state deltas since the 2026-07-07 snapshot (RED baseline)

Every `#NNN` referenced in ROADMAP.md was cross-checked against `gh issue view`. Findings:

| Item | Triage currently says | Verified reality | Required fix |
| --- | --- | --- | --- |
| **#647** | "Actionable now … verify + close" (L32) | **CLOSED/COMPLETED** 2026-07-08T02:36Z | Remove from Actionable |
| **`engram count`** | absent; "Just shipped: #659" (L23) | **SHIPPED** ADR-0018, commit 56eb2617, 2026-07-08; documented in FEATURES/c1/c2/GLOSSARY/adr | Lead "Just shipped" with count |
| **#674** | absent | OPEN (new) — route dispatch fact-notes; **reconcile #669↔count first** | Add to Actionable; annotate #669 |
| **#675** | absent | OPEN (new) — `usage report` on `count --backlinks-of`; **Track C round-3, gated on P3′** | Add to Gated; cross-ref round-3 scope (L270) |
| **#676** | absent | OPEN (new) — count **write-side** dual-write generalization | Add to Actionable |

Accurate entries (verification bookkeeping — NOT edits): #643 (closed, correctly noted L34), #659
(shipped, L23 — demote below count), #665 (closed not-planned, correctly noted L62 in Track A residuals).
No OTHER open issue is unreferenced (19 open = 16 already in ROADMAP + the 3 new).

## Scope (grep-derived, note 186 — enumerate the surface mechanically)

**The triage-list edit itself is `docs/ROADMAP.md` L16–41** (header date, Just-shipped, Actionable,
Gated, #669 annotation). **One incidental one-line consistency cross-reference** is added at **L270**
(Track C round-3 scope) — this is *not* part of "the triage list" as scoped above; it is included so the
newly-added #675 Gated-line does not point at a round-3 subsection that is silently unaware of the issue.
Flagging it as a deliberate, justified boundary extension, not silent creep.

**No other doc needs touching:** `engram count` is already fully documented in FEATURES.md,
c1-system-context, c2-containers, GLOSSARY, and adr.md (ADR-0018); #674/#675/#676 appear nowhere in docs
yet (correct — they are new). Verified by `grep -rn "#674\|#675\|#676"` (hits only this plan).

## Prioritization judgment (surface, do not act unilaterally — anti-displacement)

`engram count` shipping produced three follow-on issues that sit in **different existing homes**, not one
new bucket (note 188 — use the repo's own vocabulary; do not invent a taxonomy):

- **#674** — a **route-track consumer** of count (route records dispatches as countable fact-notes; first
  step reconciles #669↔count).
- **#675** — a **Track C round-3 consumer** of count (usage report on `--backlinks-of`; gated on P3′).
- **#676** — **count write-side infrastructure** (generalizes the attr-node dual-write); it *extends*
  count, it does not consume it.

Together they raise a real question of whether this work should displace the roadmap-flagged
**"Next: payload-prune production build"** (Track B). Decision: **keep payload-prune as Next** (the
roadmap flags it; count was an opportunistic primitive build), and **present the reprioritization as a
candidate for Joe to decide** — recorded in the findings, not baked into the edit (note 143: verify
shipped-status before reframing; note 35: present the full set, recommend within it).

## Exact edits (old → new)

**Edit 1 — L16 header date.**
`## Current priorities (2026-07-07 triage)` → `## Current priorities (2026-07-08 triage)`

**Edit 2 — L23 "Just shipped" (replace one line with two; count leads, #659 demoted).**
Old:
`**Just shipped:** **#659** — prune now *detaches* (preserves) chunks on source deletion instead of GC-ing them; the `~/restic-restore-claude/` reclaim is now safe (`docs/FEATURES.md` — Prune preserves memory).`
New (two lines):
`**Just shipped:** **`engram count`** (ADR-0018) — a read-only counting surface over the vault, separate from `query`'s similarity recall: `--group-by <attr>` (+ `--filter`) over frontmatter and `--backlinks-of <basename>` wikilink in-degree (`docs/FEATURES.md` — Count / backlinks aggregation).`
`**Also recently shipped:** **#659** — prune now *detaches* (preserves) chunks on source deletion instead of GC-ing them (`docs/FEATURES.md` — Prune preserves memory).`
(FEATURES section-name phrasing matches the doc's own voice, per the clarity finding.)

**Edit 3 — L28 #669 annotation (append reconcile note inside the bullet).**
Old:
`- **#669** (L) — structured routing-evidence ledger + `engram query` (foundation of the route track; #670 depends on it).`
New:
`- **#669** (L) — structured routing-evidence ledger + `engram query` (foundation of the route track; #670 depends on it). **Reconcile with #674 first** — `engram count` may subsume the bespoke store.`

**Edit 4 — L32 replace closed #647 with the two new Actionable issues.**
Old:
`- **#647** (S) — README/command-surface drift (core shipped `5fd24c9d`; verify + close).`
New (two bullets; #647 removed as closed/completed):
`- **#674** (M) — record route dispatches as countable fact-notes + reconcile the #669↔`count` overlap (decide count-based vs bespoke store first).`
`- **#676** (M) — generalize count's **write side**: mint `attr/<k>/<v>` nodes + dual-write so new record types are countable by both `--group-by` and `--backlinks-of` (enables #674's Obsidian-verifiable side).`

**Edit 5 — L36 "Gated" line (append #675 as a new `·`-separated item).**
Old (line ending):
`... · #652 recency centroid (gated on an over-surfacing eval).`
New (line ending):
`... · #652 recency centroid (gated on an over-surfacing eval) · #675 (Track C round-3 `usage report` — rank notes by citation in-degree on the shipped `count --backlinks-of` primitive; gated on P3′ spread PASS).`

**Edit 6 — L270 round-3 scope bullet (cross-ref #675 as the now-filed form).**
Old:
`- **`engram usage report`** (sorted per-note contribution in-degree, for retention/triage) builds only if`
`  P3′ shows spread (PASS above).`
New:
`- **`engram usage report`** (**#675** — sorted per-note contribution in-degree, for retention/triage;`
`  the `count --backlinks-of` primitive now ships the in-degree building block) builds only if`
`  P3′ shows spread (PASS above).`

## TDD-analogue (doc reconciliation)

- **RED:** the cross-check above (every triage `#NNN` == gh state; every OPEN issue placed-or-scoped-out;
  count present in shipped) — currently FAILS on #647, #674/675/676, and count. Captured as the
  verification checklist.
- **GREEN:** re-run the same cross-check after edits → all pass.
- **REFACTOR:** prose matches the snapshot's terse voice; engram's own vocabulary only (Track A/B/C,
  C1–C7, gated/parked buckets — note 188); no dangling refs. **This check covers both the ROADMAP.md
  prose AND the findings-report deliverable** (per the docs-alignment finding — the report must not
  reintroduce an invented umbrella label).

## Gates

- **Gate A** (this plan): ask-alignment (PASS); doc-state-alignment (PASS); docs/diagrams-alignment
  (finding addressed in rev 2); clarity/standards (REFUTE addressed in rev 2). Re-ACK the two
  reviewers-with-findings before execution.
- **Gate C** (touched doc): relevance + clarity/cohesion over ROADMAP.md.
- **Gate D**: commit message + findings prose.

## Deliverable

Updated `docs/ROADMAP.md` triage + a findings report (labeled criteria table) **attached as a file**
(note 175 — Joe reads from phone), repo path cited alongside. The report describes the three new issues
by their real repo-vocabulary homes (route-track / Track C round-3 / count write-side infra), not by an
invented umbrella.
