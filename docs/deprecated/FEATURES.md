# Engram — Implemented Capabilities

**Behavior specifications (the primary record):** One capability specification per shipped capability lives in `openspec/specs/<id>/spec.md`, backfilled 2026-07-27 from this document's surface. Each spec's Purpose carries its own validation anchors and links to supporting docs: why in `docs/architecture/adr.md`, measurements in `dev/eval/LEDGER.md`. This file has been slimmed to a pointer while retaining the mission rollup below.

## Validated goals (mission rollup — not a capability)

This closing section is a cross-cutting summary, not a shipped capability: it records which
founding-mission goals an adversarial review found fully achieved, drawing on the behavior specs in
`openspec/specs/` and their `dev/eval/LEDGER.md` rows (the review's source document is retired; the
still-open goals live in `docs/ROADMAP.md`). Engram's mission: a correction given once should be
applied thereafter, without the user repeating it or the agent re-deriving it.

- **Say-once, capability half** — memory carries conventions and facts a cold model
  cannot derive on its own, and this holds even under a large, realistic mix of unrelated
  notes.
- **Persistent substrate + retrieval** — the vault reliably surfaces the right note for a
  query, at the practical ceiling of what the embedder can distinguish.
- **Retrieval ranking** — crystallized notes are not lost among raw transcript noise, and
  ranking quality holds across every subsequent recall change under a standing regression
  gate.
- **Tier democratization** — a cheaper model applying recalled memory matches a pricier
  model on the same memory-backed work, on most tested axes.

validation: `dev/eval/LEDGER.md` (draws on the matched-note-floor, tier-routing, and
capability-trap rows there)
