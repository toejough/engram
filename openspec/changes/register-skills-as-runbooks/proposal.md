## Why

Four skill-to-runbook conversions shipped this month (`route`, `please`, `curate`, `write-memory`) each deleted the `SKILL.md` from `agent-instructions/skills/` and left the procedure only as a note in Joe's personal vault. That treated *runbook* and *skill* as mutually exclusive carriers — and it introduced a live regression nobody flagged: `engram update` syncs `agent-instructions/skills/` and `guidance/` into harness directories, but it never distributes vault notes. A fresh engram install today ships only `recall` and `learn`; `please`, `curate`, `route`, and `write-memory` are gone for anyone but Joe (verified: `~/.claude/engram/skills/` holds exactly `learn`, `recall`). Vault note 1054 records the reversal; it refutes note 1031's stated end goal ("every skill becomes a runbook; the shim is the only custom CLAUDE.md text").

The two original motivations for runbooks decouple cleanly from deleting the skill:

1. **Memory surfaces the right procedure at the right moment.** This needs engram to hold an indexed representation with `situation`, `triggers`, `red_flags`, `done_when` — everything built this month (lexical triggers, `--text`, whole-word matching, the shim's follow-frame and standing rule, the retrieval evals) works on that representation regardless of whether the body originates in a note or a file.
2. **Sessions share procedures across the vault graph** (server/client, personal vault → an AppDev environment). Runbook *notes* already flow over the graph's edges (verified: a parent-vault runbook reaches a client's `engram query` results via `ENGRAM_PARENT`, triggered items placed first). Skill *files* don't, because there is no edge type for files.

So the fix is not "undo runbooks" — it is to make the runbook a **registered, mechanically-derived view of a skill that remains the shipping and authoring source of truth**, not a replacement for it.

## What Changes

- **Skills gain runbook metadata in their own frontmatter.** `SKILL.md` frontmatter (today: `name`, `description` only; no Go code parses it) gains the authored runbook fields — `situation`, `triggers`, `done_when`, `red_flags` — so the hard, judgment-heavy, eval-validated parts of every conversion live in ONE place: the skill file, in git, deployed by `engram update`. A skill without these fields is deployed as today and not registered (registration is opt-in by authoring the fields).
- **New `engram register-skills` derivation** (exact command name a design detail): parses each registered skill's frontmatter + body and upserts an idempotent runbook note into the vault, keyed on a stable skill identity recorded in a new note frontmatter field (e.g. `registered_from: please/SKILL`). Re-derived on every `engram update` (hooked into `internal/cli/update.go`'s post-update vault step, where vocab regen and identity backfill already run), so the note can never drift from the skill and nothing is authored twice. Registration is a **sync**: a derived note whose skill is no longer registered is removed (vault note 442's deploys-are-syncs rule; a derived mirror is not memory).
- **Multi-runbook skills** (`please` = 1 top + 3 subs; `recall` planned as 2 entry points + shared subs) are expressed as one file per runbook inside the skill directory (`SKILL.md` is the top/entry; `runbooks/<slug>.md` are subs), each with its own runbook frontmatter. Cross-references between them are authored as stable skill-local slugs (`[[please/adversarial-review-gates]]`) and resolved to the real derived-note basenames at registration time, since vault basenames embed date + Luhmann id and cannot be known when authoring.
- **Restore the four deleted skills** from git history (route `87e9a006`, please `b79f390e`, curate `2c4c2637`, write-memory `a81f7edc`; only route also lost `price-table.md` and `tests/`), with each one's validated metadata lifted from its promoted vault note (1036, 1045+1042/1043/1044, 1049, 1053) into frontmatter. The existing promoted notes become the registered derived form (adopted via the identity key on first registration, not re-created).
- **Revert call-site wording that only existed because the skill was deleted.** `recall`/`learn` currently say "run `engram show 1053…` in Bash (write-memory is not a Skill tool)" at nine sites; write-memory's last `red_flag` literally says the skill "no longer exists". With the skill restored, both revert to native skill invocation — the write-memory eval's own skill-arm baseline was 3/3 on both metrics when invoked as a skill by recall/learn, so the native path is the proven one for a worker.
- **`engram show` gains a parent-vault fallback.** Today `engram show <basename>` is local-only unless `--parent` is passed explicitly, so the shim's follow-frame (`engram show <basename>`) cannot follow a runbook a client session received from a parent vault. With `ENGRAM_PARENT` configured and a local miss, `show` falls back to the parent. Without this, motivation 2 is only half-real: shared runbooks would surface but be unfollowable. Sequenced last so the change can ship registration + restoration first and split here if it grows too large.
- **Abandon the two in-flight delete-conversions** (`learn-skill-to-runbook`, `recall-glance-skill-to-runbook`) rather than complete them. Their validated outputs are not discarded: the situation lines, triggers, red_flags, sub-runbook splits, and eval fixtures become `recall`/`learn`'s frontmatter and registration inputs. (The unrelated open change `learn-rate-skill-only` is untouched.)
- **Reverse the now-wrong documentation and spec language** that describes skills as "retired" or states the old end goal, via the doc-surface enumeration gate every conversion used.

## Deferred (named follow-ons, not built here)

- Registering **arbitrary user/third-party skills** from `~/.claude/skills/` (29 entries today, 27 of them not engram's). This change registers engram's own shipped skills only; extending registration to user-selected external skills (Joe's "let the user decide what goes to the vault") is its own change once the mechanism exists.
- Retiring `guidance/{recall,learn,delegate}.md` — the redundancy with the shim's re-entry moments remains a real, separate cleanup.
- `recall`'s glance/deep split as two entry-point runbooks is preserved as the authored shape for its metadata, but its eval-validated promotion (the paid runs never happened) is not in scope here; registering `recall` and `learn` uses their already-built fixture metadata as the authored frontmatter, with the same retrieval checks any registered skill gets.

## Capabilities

### New Capabilities

- `skill-runbook-registration`: runbook fields in skill frontmatter; mechanical, idempotent, sync-semantics derivation of vault runbook notes from registered skills; stable identity key; multi-runbook skills with slug cross-references resolved at registration.

### Modified Capabilities

- `update-deploy-sync`: `engram update` gains a post-deploy registration step; removal of a registered skill removes its derived note.
- `vault-note-identity`: a new frontmatter field (`registered_from`) marks derived notes and carries the stable skill identity.
- `learn-runbook-capture`: registered runbooks are derived, not captured — they participate in Luhmann/embed like other runbooks but their content is owned by the skill file; `engram amend` on a registered note's derived fields is rejected or overwritten on next registration (design decides which).
- `guidance-runbook-follow-frame`: Purpose reversal ("every procedure is a runbook note" → "every registered skill is mirrored as a runbook note; skills remain the shipping form"); follow-frame's `engram show` works on parent-shared runbooks.
- `vault-merged-recall`: `engram show` falls back to the parent vault on a local miss when `ENGRAM_PARENT` is set.
- `write-memory-worker`: write-memory is a skill again, invoked natively by `recall`/`learn`; its runbook is the registered mirror.

## Impact

- **Go**: new SKILL.md frontmatter parser (none exists — `planSkillCopies` treats skills as opaque bytes); new registration command + upsert-by-identity (no create-or-update mechanism exists today — `WriteNew` is exclusive-create, `amend` resolves only by exact basename/Luhmann id); hook in `internal/cli/update.go:runPostUpdateChecks`; `show` parent fallback in `internal/cli/show.go`.
- **Restored**: `agent-instructions/skills/{route,please,curate,write-memory}/` with frontmatter metadata; `please` and (later) `recall` gain `runbooks/` subdirectories.
- **Edited skills**: `recall`, `learn` (call-site wording reverted; `learn`'s Step 2.5 duplicate-guard fix, already live, is kept).
- **Vault**: existing promoted notes 1036/1042–1045/1049/1053 adopted as registered; `recall`/`learn` registered fresh.
- **Docs/specs**: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{adr,c1,c2,c3}.md`, `docs/ROADMAP.md` (#758/#759 "Shipped" records amended), `guidance-runbook-follow-frame` Purpose, the archived conversions' "retired" language where it is normative.
- **Abandoned changes**: `learn-skill-to-runbook`, `recall-glance-skill-to-runbook` (recorded, not silently deleted — git keeps them).
- **Eval**: no new paid eval is required for the mechanism itself (registration is mechanical and unit-testable); retrieval checks for restored skills reuse the already-validated metadata. A paid re-validation is warranted only if metadata changes.
- **Issues**: #760 is re-scoped, not closed, by this change; the four "Shipped" retirements (#757/#758/#759 + write-memory) get an amendment comment.
