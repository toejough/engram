## Context

Between 2026-09-19 and 2026-09-22, four skills were converted to vault runbooks by deleting their `SKILL.md` and promoting a note to Joe's personal vault: route (`87e9a006`), please (`b79f390e`), curate (`2c4c2637`), write-memory (`a81f7edc`). Each conversion's proposal recorded "engram update stops shipping the X directory" under Impact as if neutral. It was not: `engram update` (`internal/update/update.go`, `planAndApply` :1161 → `planSkillCopies` :2380) syncs `agent-instructions/skills/` and `guidance/` into harness dirs and **never touches the vault** (stated at :300–302, :318–319, :341–343). Vault notes are not distributed. A fresh install today ships only `learn` and `recall` (verified: `~/.claude/engram/skills/` holds exactly those two). Vault note 1054 records the reversal; it refutes note 1031's "every skill becomes a runbook; the shim is the only custom CLAUDE.md text."

Two in-flight changes (`learn-skill-to-runbook`, `recall-glance-skill-to-runbook`) were about to repeat the pattern; neither reached a paid run. Their validated metadata (situation lines, triggers scored against over-fire probes, red_flags under the 1200-byte cap, eval fixtures with mutant-tested checkers) is authored input this change reuses, not discards.

Facts that constrain the design (all verified in code):
- **Skill frontmatter belongs to the harness, not to engram.** `SKILL.md` frontmatter today is exactly `name:` and `description:`; no Go code parses it (`planSkillCopies` copies skills as opaque bytes). Claude Code, Pi, and other skill loaders own that schema, so engram cannot add arbitrary fields there. Engram's own notes can carry any frontmatter it defines.
- **A note's slug is free text, so a basename can name its skill.** Basenames are `<luhmann>.<date>.<slug>.md` with a caller-supplied `[a-z0-9-]+` slug (`learnPath`, `learn.go:592`). A slug of `skill-<name>` keeps the Luhmann id and date convention intact while making the note findable by basename suffix.
- **No create-or-update mechanism exists for notes.** `deps.WriteNew` is `WriteFileExcl` (exclusive create, `deps_compose.go:180`); `engram amend` resolves `--target` only by exact Luhmann id or basename (`findNote`, `resituate.go:144`).
- **Renaming a note and rewriting links to it already exists.** `RenameAndRewriteReferences` (`luhmann_reparent.go:32`) renames a note plus its `.vec.json` sidecar and rewrites every `[[old-basename]]` wikilink in the vault in one pass. Adopting the existing promoted notes reuses it.
- **Pending offers exist, for fact/feedback only.** A note with frontmatter `pending: true` is excluded from normal query results and flagged via `pending_offers` for curation (`noteHasPendingMarker`, `offer.go:84`; spec `vault-offer-curation`). The marker is only recognized on `fact`/`feedback` notes today.
- **The vault-mutating step of update already exists one layer up.** `internal/cli/update.go:runPostUpdateChecks` (:458) resolves the vault path and runs vocab regen, identity backfill, and Luhmann reparenting with `cli.Deps` (lock, FS, embedder). Registration hooks in there. Vault-level state files live at the vault root (e.g. `vocab.centroids.json`).
- **Sharing already works for surfacing, not for following.** A parent-vault runbook reaches a client's `engram query` under `ENGRAM_PARENT` (`merged_query.go:100,177`; items tagged `from_parent`). But `engram show <basename>` is local-only unless `--parent` is passed explicitly (`show.go:19-23`; spec `vault-merged-recall` :135). The shim's follow-frame emits `engram show <basename>` with no `--parent`, so a shared runbook surfaces and then cannot be followed.
- **Nine call sites in `recall`/`learn` and write-memory's last red_flag** say the write-memory skill "no longer exists" and instruct `engram show 1053…` instead — true only while the skill is deleted. The write-memory eval's skill-arm baseline (invoked natively as a skill by recall/learn) was 3/3 followed_all, 3/3 end_state (`results/1.4_write_memory_skill_baseline_sonnet5.md`).

## Goals / Non-Goals

**Goals:**
- Restore distribution: every engram install ships all six skills again, via the existing `engram update` mechanism, with no new distribution machinery.
- Keep everything the runbook work bought (memory-driven surfacing by situation/trigger, the follow-frame, cross-vault sharing) by giving each skill a runbook note in the vault.
- Leave skill files exactly as harnesses expect them: no engram-specific frontmatter, no extra metadata files.
- Never change a vault note without the user agreeing: engram offers, the user decides, and a declined offer is not repeated.
- Carry forward every validated conversion artifact.

**Non-Goals:**
- Registering skills engram doesn't ship (the 27 user/plugin skills in `~/.claude/skills/`). Named follow-on.
- Retiring `guidance/{recall,learn,delegate}.md`. Named follow-on.
- Any new distribution mechanism for vault notes. The point is that skills already have one.
- Changing how any skill's procedure actually works.
- Re-running the paid recall/learn evals.

## Decisions

**D1. The skill file owns the procedure; the vault note owns the runbook fields.** The skill (`SKILL.md`, shipped by `engram update`) is the source of the procedure text. Its runbook note carries a copy of that text as its body, plus the runbook fields (`situation`, `triggers`, `done_when`, `red_flags`) that the shim's follow-frame drives. Alternatives: (a) keep runbooks canonical and invent a note-distribution mechanism — rejected, it duplicates a mechanism that exists; (b) "search returns skills" with no runbook representation — rejected, a returned skill body is prose the agent interprets fresh, while a runbook carries `done_when`/`red_flags`/`triggers` the follow-frame drives (the please eval: skill row 0/3 followed_all vs triggered runbook 3/3).

**D2. Runbook fields are authored on the note, never in the skill.** `SKILL.md` keeps only `name` and `description`. Alternatives: (a) add the runbook fields to `SKILL.md` frontmatter — rejected, that schema belongs to the harnesses that load skills, and other harnesses or skill tooling may use or validate those fields; (b) a per-skill metadata file (e.g. `runbook.yaml`) — rejected, it adds a second authoring surface inside the skill directory for fields that already have a natural home on the note, where engram controls the schema. The runbook fields were the judgment-heavy, eval-validated part of every conversion (please's situation line took three rewrites; triggers needed over-fire scoring), so they are authored by an agent or a person, never inferred mechanically (D5).

**D3. One note per skill, found by its slug; a hash records what it was synced from.** Each registered skill has exactly one runbook note, with slug `skill-<name>` (basename `<luhmann>.<date>.skill-<name>.md`, normal Luhmann id and date). Registration finds it by basename suffix `.skill-<name>.md` on a runbook note carrying a `skill_hash` field. `skill_hash` is the SHA-256 of the `SKILL.md` bytes the note's body was last copied from. Alternatives: (a) a `registered_from:` frontmatter key scanned across the vault — rejected, the slug already identifies the note, and a basename is visible to people reading the vault; (b) several notes per skill (please's former top + 3 subs, recall's glance/deep entry points) — rejected, there is no body source for a sub-note other than a second file in the skill directory, and a rule for which skills get several notes was the confusion that sank the earlier draft. A skill is one procedure; it gets one note.

**D4. `engram update` offers; the user decides; a declined offer is remembered by hash.** After deploying skills, update compares each shipped skill with its note:

| State | Offer | Yes | No |
| --- | --- | --- | --- |
| No `skill-<name>` note | "Register skill `<name>` as a vault runbook?" | Create the note (D5) | Record the skill's current hash as declined |
| Note exists, `skill_hash` differs from the skill | "Skill `<name>` changed since its note was last synced. Update the note?" | Replace the body with the current `SKILL.md`, set `skill_hash`, keep the runbook fields, and set `pending: true` so curation re-checks the fields against the new procedure (D5) | Record the new hash as declined |
| Note exists, skill no longer shipped | "Skill `<name>` is no longer shipped. Remove its runbook note?" | Remove the note and its sidecar | Record the decline |
| Hashes match, or the current hash is already declined | none | — | — |

Declines live in one vault-root state file, `skill-registrations.json`, mapping skill name to the declined hash. One file, rather than a field on the note, because a declined *new* skill has no note to hold it. A later change to the skill produces a new hash, so the offer returns once. When stdin is not a terminal (an agent running `engram update` through a shell), update prompts for nothing, writes nothing, and records nothing; it prints one line naming the skills awaiting an answer and the command to answer them. `engram register-skills` runs the same comparison standalone, with `--accept <name>` and `--decline <name>` (repeatable) so an agent can pass along the user's answer, and `--dry-run` to list the offers without prompting. Alternative: re-derive silently on every update (the earlier draft) — rejected, the note now holds authored fields, so overwriting or deleting it without asking loses work.

**D5. A newly registered note starts as a pending offer that curation completes.** Accepting registration creates the note with the `SKILL.md` body, `skill_hash`, and `pending: true`, and no runbook fields. The pending-offer marker is extended from `fact`/`feedback` to `runbook` notes, so the incomplete note is kept out of normal retrieval (it has no situation to match yet) and raises the existing `pending_offers` flag and notices. The curate runbook gains a branch for pending skill notes: author (or, after a refresh, re-check) `situation`, `done_when`, `triggers`, and `red_flags` from the body, then clear the marker with `engram amend --clear-pending`. Alternatives: (a) update drafts the fields through the LLM API — rejected, it skips the review those fields needed in every conversion; (b) update only reports unregistered skills and leaves creation to a separate command — rejected, it asks the user twice for one decision.

**D6. Adopt the existing promoted notes instead of re-creating them.** Notes 1036 (route), 1045 (please), 1049 (curate), and 1053 (write-memory) already carry eval-validated runbook fields. `engram register-skills --adopt <name>=<note-ref>` renames the note to the `skill-<name>` slug through `RenameAndRewriteReferences` (same Luhmann id and date; inbound wikilinks rewritten), replaces its body with the restored `SKILL.md`, and stamps `skill_hash`; the fields are kept and the note is not marked pending. please's sub-notes 1042/1043/1044 are superseded by the full `SKILL.md` body now on the please note; they are removed at migration time after inbound links to them are repointed at the please note (D3: one note per skill). `--adopt` is general-purpose: it also serves a user who hand-wrote a runbook for a skill before registration existed.

**D7. Restore the four skills from their last intact commits and revert deletion-only wording.** route from `f5b44504` (incl. `price-table.md`, `tests/`), please from `ec5f3735`, curate from `14049280`, write-memory from `cbda7f6e`, byte-identical and with no frontmatter additions. The nine `recall`/`learn` call sites revert to native skill invocation of write-memory (the proven 3/3 path); write-memory's "no longer exists as a skill" red_flag is dropped from note 1053 at adoption; curate gains the D5 branch. `learn`'s Step 2.5 duplicate-guard fix (already live in the real skill) is kept. Skill edits go through `superpowers:writing-skills`.

**D8. `engram show` falls back to the parent on a local miss when `ENGRAM_PARENT` is set.** Today only `--parent` reaches the parent. The shim's follow-frame cannot be told to add `--parent` selectively (its instruction is a bare `engram show <basename>`). Fallback-on-miss is transparent, needs no shim change, and mirrors merged query's existing local-then-parent behavior. `ENGRAM_SERVER` precedence and the explicit `--parent` flag are unchanged. Sequenced last (see Migration Plan) so registration + restoration can ship first and this can be split off if the change grows too large.

**D9. Abandon, don't archive, the two delete-conversion changes.** OpenSpec has no "abandoned" state; the change directories are removed with a commit whose message records why and where their artifacts were carried (the fixture vaults and fidelity reports under `dev/eval/` stay; recall/learn's runbook fields are authored from them). Alternative: archive them as complete — rejected, they weren't, and archiving would sync their delta specs (which assert the skills are retired) into the main specs.

**D10. No paid eval for the mechanism.** Registration is mechanical (compare hash → prompt → create/refresh/remove) and fully unit-testable with DI fakes. Re-run a retrieval check (no spend) for each registered skill after adoption to confirm the notes surface as the promoted notes did. One structural change is unmeasured: please's runbook becomes one note instead of four (D3/D6). A paid re-validation of please's following fidelity is warranted only if the retrieval check or use shows a regression, and needs Joe's confirmation first.

## Risks / Trade-offs

- **[Risk] A declined refresh leaves the note's body behind the skill.** → Intended: the user chose it. The decline is per-hash, so the next skill change offers again. The note's body preamble names the skill it mirrors, so a reader can tell where the current text lives.
- **[Risk] Agents run `engram update` without a terminal, so offers may go unanswered indefinitely.** → The non-interactive summary line names the waiting skills and the `engram register-skills --accept/--decline` command; an agent relays it to the user. Nothing is written or recorded without an answer.
- **[Risk] A refreshed body can invalidate runbook fields** (a red_flag naming a removed step). → Accepting a refresh sets `pending: true`, so curation re-checks the fields before the note returns to normal retrieval.
- **[Trade-off] A pending note is invisible to retrieval until curated.** → Correct for a new note (no situation to match). After a refresh, the skill itself is still deployed and fires normally, so the gap costs only memory-driven surfacing, and only until curation.
- **[Trade-off] Collapsing please to one note gives up per-sub red_flags budgets.** → please's 9 top-level red_flags carry over; sub-specific red_flags are reviewed into the single note within the 1200-byte cap at adoption, and anything cut is recorded in the migration task (D10 covers the fidelity question).
- **[Risk] `show` parent fallback changes the meaning of a local miss (previously an error).** → Only when `ENGRAM_PARENT` is set; the fallback result is labeled as parent-sourced; explicit `--parent`/`ENGRAM_SERVER` semantics unchanged and tested.
- **[Trade-off] This change is large (offer flow, pending runbooks, adoption, four restorations, docs/spec reversals, two abandonments, show fallback).** → Sequenced so registration + restoration land first; D8 and the doc enumeration are the natural split points.
- **[Risk] Doc/spec surface is the widest yet** — every conversion's "retired" language plus the reversed end goal in `guidance-runbook-follow-frame`'s Purpose, ADR-0026's amendment, GLOSSARY, CLAUDE.md, README, ROADMAP "Shipped" rows. → The doc-surface enumeration gate with an independent fresh-context reviewer; Purpose-line edits become archive-time tasks.

## Migration Plan

1. Pending marker on runbook notes; `skill_hash` field; slug lookup; the offer comparison and decline state file; `engram register-skills` (`--accept`, `--decline`, `--dry-run`, `--adopt`); hook into `runPostUpdateChecks` with the non-interactive path. TDD with DI fakes.
2. Restore the four skills byte-identical; revert the nine call sites; add curate's D5 branch.
3. Adopt 1036, 1045, 1049, 1053 with `--adopt`; review please's sub-note red_flags into 1045 within budget; repoint links to 1042–1044 and remove them; drop write-memory's stale red_flag; `engram embed status` clean; retrieval checks (no spend).
4. Register `recall` and `learn` through the offer flow; author their runbook fields from the fixture runbooks during curation; retrieval checks.
5. Abandon the two delete-conversion changes (D9).
6. `show` parent fallback (D8) with tests, incl. a merged-vault smoke.
7. Doc-surface enumeration + independent review; perform every row; archive-time Purpose edits recorded as tasks.
8. `engram update --with-guidance` on this machine; verify all six skills deployed in both harnesses, every note present with a matching `skill_hash`, and no offers outstanding; `targ check-full`; commit; archive.

Rollback: revert the commit(s); skill notes are ordinary runbook notes and can be removed by hand (git holds the vault's history); restored skills are plain files; `skill-registrations.json` can be deleted.

## Open Questions

- None blocking. Exact prompt wording and the non-interactive summary line are settled during implementation.
