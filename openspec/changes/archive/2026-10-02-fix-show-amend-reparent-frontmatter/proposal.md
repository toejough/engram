## Why

Four defects (#772, #776, #770, #780) each make engram quietly lose or contradict information an agent relies on:

- `engram show` drops a runbook's red_flags, and its marker points back to `engram show`.
- `engram amend` can blank a note's `user:` field.
- The reparent derive payload sends agents away from the learn skill's batch mode.
- The shared frontmatter-rewrite code can corrupt a note, leave a note's Luhmann id stale, or drop frontmatter keys.

Joe approved fixing all four in one batch on 2026-10-01. Each fix is small and touches separate code. Before this proposal was written, each claim was checked against the code. Most of them hold. One part of #776 was already fixed: the CLI already resolves `vault:` (see Impact).

## What Changes

- **#772: `engram show` returns the full note.** show stops passing note content through `capRedFlagsForPreview`, so every `red_flags` entry reaches the agent. `engram query` keeps its newest-first red_flags preview. Its omission marker now gives the number of dropped entries, the total, and the note's actual basename, as a command that returns everything (`engram show <basename>`). The 1200-byte figure is renamed from a "cap" on what a runbook may carry to a *query preview budget measured in rendered YAML bytes*. The rename applies in the `vault-offer-curation` spec and in the curate skill.
- **#776: amend no longer blanks `user:`.** When a re-stamping amend detects an empty user, it keeps the note's existing non-empty `user:` value and prints a warning, instead of writing `user: ""`. Joe decided this on 2026-10-01. `vault:` needs no code change: the `amend` CLI target already calls `resolveVaultName`. This change adds a regression test that pins that behavior.
- **#770: the reparent derive payload defers to learn's batch mode.** The payload's `instruction` names the learn skill's "Batch mode — Luhmann re-eval answers" and keeps the JSON shape. It requires exactly one entry per distinct candidate `note`. It says the answers file goes back to the user or orchestrating agent, who previews with `--dry-run` and then applies. The learn pass does not apply. The flat-vault update notice names the same skill mode. `next_command`, the fingerprint, and apply gating stay the same.
- **#780: frontmatter rewrites become safe.**
  - (a) Before it renames or writes anything, the rename and rewrite step checks every note it would rename. The rewritten frontmatter must decode, and the decoded `luhmann:` must equal the new id. Otherwise the whole run is refused. This closes the dangling-anchor corruption on reparent.
  - (b) Joe decided on 2026-10-01 that the rename and rewrite step converts a CRLF note to LF. It does this only for notes it already writes, and inside that same atomic write. The sidecar is rebuilt. The renamed file's `luhmann:` is no longer left holding the old id, and a CRLF referrer's `supersedes:` is now rewritten.
  - (c) Adopt and refresh edit the frontmatter as a YAML node tree. Keys the typed struct does not model are kept with their values.
  - (#780 comment, folded in by Joe's ruling of 2026-10-01) `engram resituate` gets the same YAML-node edit, so it no longer drops keys the struct does not model.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `recall-runbook-surfacing`: `engram show` returns every `red_flags` entry. Only query previews truncate, and their marker gives a count and a working full-retrieval command.
- `vault-offer-curation`: the 1200 bytes becomes a query preview budget, not a cap on how much a runbook may carry.
- `vault-note-identity`: an amend that re-stamps identity keeps the prior non-empty `user:` when detection comes back empty. New scenario: an amend with no vault name configured stamps `personal`. `resituate` keeps every frontmatter key it does not change, including unmodeled keys.
- `update-reparent-luhmann-batch`: the derive payload's `instruction` names learn's batch mode and its hand-back and preview flow. New requirement: the rename and rewrite step refuses, before any write, a note whose rewrite would be undecodable or would leave a stale `luhmann:`. The step converts the CRLF notes it writes to LF and rebuilds their sidecars.
- `update-flat-vault-luhmann-notice`: the notice names learn's batch mode as how the answers are produced.
- `skill-runbook-registration`: refresh and adopt keep every frontmatter key they don't set, including keys the typed note struct does not model.

## Impact

- **Code (`internal/cli`):**
  - `show.go:60`, `redflags_truncation.go:11,30`, and the call sites `query.go:1652`, `query_triggers.go:61` and `merged_dedupe.go:233` (#772).
  - `amend.go:231-241` and `identity.go:28-34` (#776).
  - `luhmann_reparent_apply.go:537-545` and `update.go:58-59` (#770).
  - `luhmann_reparent.go:86-112,197-304` and `skillreg_accept.go:88-93,172,301-347` (#780).
  - `resituate.go:230-264` (#780 comment). These reuse the existing `yaml.Node` helpers in `pulldown.go` (`setMappingValue`, `deleteMappingKeys`, `encodeNode`).
- **Not changed:**
  - `embed.SplitFrontmatter`, which has 9+ callers and feeds hashing. CRLF is converted only on notes the rename or rewrite writes. For CRLF *frontmatter*, the exchange hash changes when the note is converted. This is harmless: before conversion the note's xid could not be read, and afterwards the note hashes the same as its LF original. See design D5.
  - The learn skill and note 1067.
- **Skills and docs:**
  - `agent-instructions/skills/curate/SKILL.md:37,43`: reworded to "preview budget", under writing-skills TDD.
  - `docs/GLOSSARY.md`: the reparent derive and answer bullets.
  - `docs/ROADMAP.md`: rows 8, 12, 13 and 16 are closed out.
  - `docs/architecture/adr.md:1232`: unchanged. Its claim that "`resituate` preserves every field it doesn't change" becomes true with D6b.
  - `agent-instructions/guidance/shim.md` rules 6 and 7 are already correct under this change.
- **Vault:** no migration. Note files are not touched. Notes whose red_flags fit the budget render byte-identically in query.
