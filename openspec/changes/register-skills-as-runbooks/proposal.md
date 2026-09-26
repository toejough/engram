## Why

Four skill-to-runbook conversions shipped this month (`route`, `please`, `curate`, `write-memory`) each deleted the `SKILL.md` from `agent-instructions/skills/` and left the procedure only as a note in Joe's personal vault. That treated *runbook* and *skill* as mutually exclusive carriers — and it introduced a live regression nobody flagged: `engram update` syncs `agent-instructions/skills/` and `guidance/` into harness directories, but it never distributes vault notes. A fresh engram install today ships only `recall` and `learn`; `please`, `curate`, `route`, and `write-memory` are gone for anyone but Joe (verified: `~/.claude/engram/skills/` holds exactly `learn`, `recall`). Vault note 1054 records the reversal; it refutes note 1031's stated end goal ("every skill becomes a runbook; the shim is the only custom CLAUDE.md text").

The two original motivations for runbooks decouple cleanly from deleting the skill:

1. **Memory surfaces the right procedure at the right moment.** This needs engram to hold an indexed representation with `situation`, `triggers`, `red_flags`, `done_when` — everything built this month (lexical triggers, `--text`, whole-word matching, the shim's follow-frame and standing rule, the retrieval evals) works on that representation regardless of whether the body originates in a note or a file.
2. **Sessions share procedures across the vault graph** (server/client, personal vault → an AppDev environment). Runbook *notes* already flow over the graph's edges (verified: a parent-vault runbook reaches a client's `engram query` results via `ENGRAM_PARENT`, triggered items placed first). Skill *files* don't, because there is no edge type for files.

So the fix is not "undo runbooks" — it is to keep skills as the shipping form of every procedure and give each one a **runbook note in the vault** that mirrors its text and carries the runbook fields memory needs. Engram offers to create and refresh these notes; the user decides.

## What Changes

- **Skill files stay exactly as harnesses expect them.** `SKILL.md` keeps only `name` and `description`; engram adds no frontmatter and no metadata files to skill directories (that schema belongs to Claude Code, Pi, and other skill loaders). The runbook fields — `situation`, `triggers`, `done_when`, `red_flags` — are authored on the vault note, where engram controls the schema.
- **One runbook note per skill, named for it.** The note's slug is `skill-<name>` (basename `<luhmann>.<date>.skill-<name>.md`, normal id and date). Its body is a copy of the skill's `SKILL.md`, and a new `skill_hash` field records the SHA-256 of the text it was copied from.
- **`engram update` offers; the user decides.** After deploying skills, update offers to register any shipped skill that has no note, to refresh a note whose `skill_hash` no longer matches its skill, and to remove the note of a skill no longer shipped. A declined offer is recorded by hash in a vault-root `skill-registrations.json`, so the same skill version is never offered twice; a later skill change offers once more. Without a terminal (an agent running update through a shell), update writes nothing and prints one line naming the skills awaiting an answer. `engram register-skills` runs the same comparison standalone, with `--accept`/`--decline <name>` so an agent can relay the user's answer, `--dry-run`, and `--adopt <name>=<note>` for existing notes.
- **New notes are completed through curation.** Accepting registration creates the note with the skill's text, `skill_hash`, and `pending: true`, but no runbook fields. The pending-offer marker is extended from fact/feedback notes to runbook notes, so the incomplete note stays out of retrieval and raises the existing pending-offers notices; the curate runbook gains a branch that authors the fields and clears the marker. Accepting a refresh also sets `pending: true`, so curation re-checks the fields against the new text.
- **Restore the four deleted skills** from git history (route `87e9a006`, please `b79f390e`, curate `2c4c2637`, write-memory `a81f7edc`; only route also lost `price-table.md` and `tests/`), byte-identical. The existing promoted notes (1036, 1045, 1049, 1053) are **adopted** — renamed to the `skill-<name>` slug with inbound links rewritten, body replaced by the restored skill, fields kept. please's sub-notes (1042/1043/1044) are superseded by the full skill text on the please note and removed after their links are repointed.
- **Revert call-site wording that only existed because the skill was deleted.** `recall`/`learn` currently say "run `engram show 1053…` in Bash (write-memory is not a Skill tool)" at nine sites; write-memory's last `red_flag` literally says the skill "no longer exists". With the skill restored, both revert to native skill invocation — the write-memory eval's own skill-arm baseline was 3/3 on both metrics when invoked as a skill by recall/learn.
- **`engram show` gains a parent-vault fallback.** Today `engram show <basename>` is local-only unless `--parent` is passed explicitly, so the shim's follow-frame (`engram show <basename>`) cannot follow a runbook a client session received from a parent vault. With `ENGRAM_PARENT` configured and a local miss, `show` falls back to the parent. Sequenced last so the change can ship registration + restoration first and split here if it grows too large.
- **Abandon the two in-flight delete-conversions** (`learn-skill-to-runbook`, `recall-glance-skill-to-runbook`) rather than complete them. Their validated situation lines, triggers, red_flags, and eval fixtures become the runbook fields authored on `recall`'s and `learn`'s notes. (The unrelated open change `learn-rate-skill-only` is untouched.)
- **Reverse the now-wrong documentation and spec language** that describes skills as "retired" or states the old end goal, via the doc-surface enumeration gate every conversion used.

## Deferred (named follow-ons, not built here)

- Registering **arbitrary user/third-party skills** from `~/.claude/skills/` (29 entries today, 27 of them not engram's). This change registers engram's own shipped skills only; extending registration to user-selected external skills (Joe's "let the user decide what goes to the vault") is its own change once the mechanism exists.
- Retiring `guidance/{recall,learn,delegate}.md` — the redundancy with the shim's re-entry moments remains a real, separate cleanup.
- `recall`'s glance/deep split as two entry-point runbooks is not carried forward: each skill has one note. Its fixture metadata informs the fields authored on `recall`'s single note; the eval-validated promotion (the paid runs never happened) is not in scope.

## Capabilities

### New Capabilities

- `skill-runbook-registration`: one runbook note per shipped skill (slug `skill-<name>`, `skill_hash`); update offers to register, refresh, and remove, with declines remembered by hash; new and refreshed notes are pending until curated; adoption of existing notes.

### Modified Capabilities

- `update-deploy-sync`: `engram update` runs the registration offers after deploying; without a terminal it writes nothing and reports the waiting offers.
- `vault-note-identity`: a new frontmatter field (`skill_hash`) marks skill notes and records the skill text they mirror; `engram amend` preserves it.
- `learn-runbook-capture`: a skill's runbook note participates like a captured runbook; its body mirrors the skill, its runbook fields are authored on the note.
- `vault-offer-curation`: the pending-offer marker extends to runbook notes; curation authors or re-checks a pending skill note's runbook fields; curate is a skill again.
- `guidance-runbook-follow-frame`: Purpose reversal ("every procedure is a runbook note" → "every registered skill has a runbook note; skills remain the shipping form"); follow-frame's `engram show` works on parent-shared runbooks.
- `vault-merged-recall`: `engram show` falls back to the parent vault on a local miss when `ENGRAM_PARENT` is set.
- `write-memory-worker`: write-memory is a skill again, invoked natively by `recall`/`learn`; its runbook note mirrors it.
- `please-doc-enumeration-gate`: the doc-surface enumeration gate is carried by the please skill again, not a separate runbook; the skill's Step 3 procedure runs the enumeration grep.
- `learn-branching-disposition`: the placement-fields handoff contract belongs to the write-memory skill again, reached by native invocation rather than by basename/wikilink.

## Impact

- **Go**: `skill_hash` field and slug lookup; pending marker recognized on runbook notes; the offer comparison, interactive prompts, non-interactive summary, and `skill-registrations.json` decline state; `engram register-skills` (`--accept`, `--decline`, `--dry-run`, `--adopt`, the last reusing `RenameAndRewriteReferences`); hook in `internal/cli/update.go:runPostUpdateChecks`; `show` parent fallback in `internal/cli/show.go`. No SKILL.md parser — skills stay opaque bytes except for hashing.
- **Restored**: `agent-instructions/skills/{route,please,curate,write-memory}/`, byte-identical to their last intact commits.
- **Edited skills**: `recall`, `learn` (call-site wording reverted; `learn`'s Step 2.5 duplicate-guard fix, already live, is kept); `curate` (branch for pending skill notes).
- **Vault**: notes 1036/1045/1049/1053 adopted (renamed to `skill-<name>` slugs, links rewritten); 1042–1044 removed after links are repointed; `recall`/`learn` registered through the offer flow; new vault-root `skill-registrations.json`.
- **Docs/specs**: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{adr,c1,c2,c3}.md`, `docs/ROADMAP.md` (#758/#759 "Shipped" records amended), `guidance-runbook-follow-frame` Purpose, the archived conversions' "retired" language where it is normative.
- **Abandoned changes**: `learn-skill-to-runbook`, `recall-glance-skill-to-runbook` (recorded, not silently deleted — git keeps them).
- **Eval**: no new paid eval is required for the mechanism itself (registration is mechanical and unit-testable); retrieval checks for restored skills reuse the already-validated metadata. A paid re-validation is warranted only if metadata changes.
- **Issues**: #760 is re-scoped, not closed, by this change; the four "Shipped" retirements (#757/#758/#759 + write-memory) get an amendment comment.
