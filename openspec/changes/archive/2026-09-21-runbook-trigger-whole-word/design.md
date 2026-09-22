## Context

`matchTriggers` (internal/cli/query_triggers.go) normalizes text and trigger (lowercase, whitespace
runs to one space) and tests `strings.Contains`. Design D3 of the archived change
`runbook-lexical-triggers` chose substring and deferred word-boundary matching "until a measured
false-positive appears". Joe wants bare-word triggers (`curate`, `please`), which makes false hits
("accurate", "pleased") certain, so the deferral condition is met by construction.

## Goals / Non-Goals

- Goals: bare-word triggers hit only as whole words; slash-form and multi-word triggers keep working.
- Non-Goals: stemming ("curates" does not match `curate`; author lists both if wanted), regex or
  wildcard triggers, changing normalization, changing surfacing/ranking.

## Decisions

**D1. Boundary rule.** A trigger occurrence in the normalized text is a hit when the rune immediately
before it (if any) and the rune immediately after it (if any) are not letters or digits
(`unicode.IsLetter` / `unicode.IsDigit`, on runes, so multi-byte neighbors such as "é" block).
Underscore, hyphen, slash, punctuation, and whitespace are boundaries. Text start/end counts as a
boundary. All occurrences are tried; a later one may satisfy the boundary when the first does not.

**D2. Edge-conditional boundaries.** The boundary is enforced on a side only if the trigger's own
edge rune on that side is a letter or digit. `/please` starts with `/`, so it needs no left boundary
("do/please" still hits) but still needs a right boundary ("/pleased" misses). A trigger ending in
punctuation needs no right boundary. Rationale: a boundary test against a non-word edge would
wrongly reject "x/please" or "e.g.x"; the author who wrote punctuation at the edge already
supplied the delimiter.

**D3. Why not regex.** A compiled regex per trigger per query would need escaping of author text,
and the rule is two rune lookups. A hand-written scan (`strings.Index` loop plus
`utf8.DecodeRune*`) is smaller, allocation-free, and has no injection surface. The property test
uses a regexp oracle, so the two implementations cross-check each other.

**D4. Effect on existing triggers.** Audited via `engram show
1045.2026-09-20.please-drive-ask-end-to-end` (read-only): triggers are `/please`,
`take this end-to-end`, `please`. `/please` and `take this end-to-end`: unchanged on every input
where they were meant to hit ("/please fix it", "do /please", "Please take this end-to-end: X").
`please`: stops matching "pleased"/"displease" (intended); still matches "please", "Please,", "(please)",
"/please". Trigger notes in eval fixtures were checked with grep for `triggers:`; the fixture
runbooks carry slash/phrase triggers only, so no fixture verdict changes. No vault write needed.

**D5. Not a migration.** Triggers are stored as authored; only the match function changes, so no
note is rewritten and no sidecar goes stale.

## Risks / Trade-offs

- [Inflected forms no longer hit: `curate` vs "curated"] -> intended; authors list forms explicitly.
- [Very common bare words still over-fire as whole words: `fix`, `run`] -> authoring guidance keeps
  "never a very common word"; a hit is still only a candidate the agent judges.
- [Unicode: combining marks (category M) are not letters] -> a base letter followed by a combining
  mark counts the mark as a boundary; accepted, matches `unicode.IsLetter` semantics and is rare.

## Migration Plan

None. Ship the binary; behavior changes on next `engram query --text`.

## Open Questions

- RESOLVED: the write-memory SKILL.md authoring rule and the spec requirement "Trigger cues SHALL be specific, author-chosen strings" now admit distinctive bare words (whole-word matched) and deliberate, recorded over-fire; verified with 15 control (HEAD) and 25 GREEN fresh-context headless runs (task 4.2).
