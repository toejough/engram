# MOC migration procedure (F4 resolution)

Date: 2026-05-24. Companion to the tiered-memory research log.

This document specifies the procedure for migrating the 25
existing MOCs into the v2 fact/feedback schema. Executing it is
a separate work item from spec'ing it.

## Goal

Convert each existing MOC's framing-prose synthesis into one or
more fact/feedback notes that fit the v2 schema, with proper
linkage and (where applicable) a higher-level abstraction
binding multi-principle MOCs together. Original MOC files move
to `_legacy/MOCs/` as an audit trail.

## Source of truth

`/Users/joe/.local/share/engram/vault/MOCs/` — 25 files.

## Per-MOC procedure

### 1. Read the framing prose

Read the H1 title, the date line, and the multi-paragraph
framing prose. Ignore the bulleted constituent list during this
step — that's denormalized backlink material, not new content.

### 2. Identify principles in the framing

The framing prose typically contains 1–5 distinct principles.
For each, classify as **fact** or **feedback**:

- **Fact** — statement of how things ARE, how a thing behaves,
  what's true about a domain. *"LLM defaults bias toward
  caution and helpfulness."*
- **Feedback** — statement of what to DO (or NOT do) differently
  next time. *"Don't substitute a softer action for an
  explicit user directive."*

Some principles split: *"LLM defaults bias toward caution,
which override clear directives, so the counter-discipline is
to read literally."* That's two notes — a fact about the bias
and a feedback about the discipline.

### 3. Extract each principle as a note

For each identified principle:

- Use `engram learn fact …` or `engram learn feedback …`.
- `--source`: `"migrated from MOCs/<original-id>.md, 2026-05-24"`.
- `--situation`: lifted from the MOC's framing, normalized to
  the activity + domain (recall-mirror discipline applies).
- `--position top` unless the principle clearly continues an
  existing top-level fact in the vault.

### 4. Examine for higher-level abstraction (meta-analysis)

If the MOC's framing prose produced multiple principles, ask:
**is there a higher-level abstraction that binds them together
beyond what the individual principles state?**

The MOC's framing often *names* this abstraction explicitly in
the first paragraph (e.g., MOC/66's "dilution moves" frames
three specific dilution failures as instances of one pattern).

If yes:

- Write the meta-abstraction as its own fact or feedback note.
- The constituent notes get `--relation "<meta-id>|instance of
  the [pattern name]"`.
- The meta-abstraction gets `--relation` entries pointing at
  each constituent: `<constituent-id>|specific instance: …`.

If no — the principles are independent observations that
happened to share a MOC — write them without a meta-abstraction
and cross-link them with relations: `<sibling-id>|adjacent
principle from the same migration cluster`.

### 5. Wire the related-to bullets

For every new note, populate `--relation` flags to:

- The MOC's original constituent permanents (the wikilinks the
  MOC listed). Rationale: `"original MOC constituent — [what
  it specifically contributes]"`.
- Sibling notes from the same migration (cross-linking).
- Any other MOCs' new abstractions if the original MOC
  cross-linked to them.

### 6. Move the original MOC to `_legacy/`

After all derived notes are written, move the original MOC
file from `MOCs/` to `_legacy/MOCs/`. Don't delete — preserve
the audit trail for one release cycle, then prune.

## Migration order

MOCs reference other MOCs (e.g., MOC/65 references MOC/7 and
MOC/11 in its framing prose). Migration order matters for
wikilink targeting:

1. Process MOCs in dependency order — leaf MOCs first, then
   MOCs that reference others.
2. When a MOC's framing prose references another MOC, the
   target should already be migrated. The new note's relation
   points at the *new* fact/feedback note, not the legacy MOC.
3. If circular references exist (MOC A → MOC B → MOC A in
   prose), break the cycle by referencing the original
   constituent permanents instead of the unmigrated MOC.

Mechanical first step: build the MOC-to-MOC dependency graph
by scanning each MOC's framing for `[[<id>.<date>...]]`-shape
references where the target ID matches another MOC file.

## Example migration: MOC/66 (user-directive-compliance)

Original framing prose extracts to:

**One meta-abstraction (feedback):**

- Situation: *"When executing an explicit user directive,
  especially a destructive or specific one"*
- Behavior: *"LLM defaults dilute the directive — substituting
  a softer action, inferring a different artifact, or treating
  general acknowledgment as authorization"*
- Impact: *"The user's contract is silently broken; literal
  intent is overridden by agreeable interpretation"*
- Action: *"Read the directive literally and execute literally,
  especially when the literal reading is narrower or scarier
  than the agreeable interpretation"*

**Plus one supporting fact:**

- Situation: *"When an LLM is given an explicit user directive"*
- Subject: *"LLM default behavior"*
- Predicate: *"biases toward caution, helpfulness, and
  conversational responsiveness"*
- Object: *"all of which can quietly override a clear literal
  directive without explicit reasoning"*

The three original constituents (Permanent/35, 35a, 35b) are
already principle-stated atomic notes; they remain as-is. The
new meta-abstraction relates to them as the binding pattern.

## Effort estimate

- ~25 MOCs to process
- Small MOCs (~1.7-3KB, 3-5 constituents) → 1-3 notes each;
  ~10 minutes per migration
- Medium MOCs (~3-5KB) → 2-4 notes; ~20 minutes
- Large MOCs (~10-12KB, 50+ constituents) → 3-8 notes;
  ~45-60 minutes

Total: probably 4-8 hours of focused LLM-judgment work, ideally
spread across multiple sessions to avoid fatigue. ~50-100 new
notes generated.

## What the resulting vault looks like

- `MOCs/` directory empty (or deleted).
- `_legacy/MOCs/` contains the 25 original files for audit.
- `Permanent/` has ~50-100 new fact/feedback notes derived
  from MOC migration, indistinguishable in schema from
  organically-written facts/feedback.
- Inbound wikilinks (from Permanents that linked back to MOCs
  via `Related to:`) need updating in a separate pass — search
  for `[[<moc-luhmann-id>.` patterns and rewrite to point at
  the most relevant migrated note (or its meta-abstraction).

## When to execute

After the v2 spike passes (Arctic-xs verified) but before
shipping v2 publicly. Migration is upstream of:

- Embedding the vault (sidecars for migrated notes get
  generated as part of the v2 backfill).
- Cutting the `engram learn moc` command from the binary
  (the writable kind goes away after migration; before would
  break partial migration runs).

## What this exercise teaches us about F9

The MOC migration is **manual F9** — extracting cross-cutting
abstractions from a cluster of related notes, with LLM judgment
deciding whether the synthesis binds them into a higher
principle. Each MOC was an early cluster; each migration is a
mini synthesis pass.

If the migration produces valuable meta-abstractions (likely),
F9 — which automates this exact operation across embedding-
detected clusters — has demonstrated value. The migration is a
proof-of-concept for the F9 pattern.
