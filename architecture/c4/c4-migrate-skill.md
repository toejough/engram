---
level: 4
name: migrate-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: cd55eab2
---

# C4 — migrate-skill (Property/Invariant Ledger)

> Component in focus: **E14 · migrate skill** (refines L3 c3-skills.md).
> Source files in scope:
> - [skills/migrate/SKILL.md](skills/migrate/SKILL.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E14. The migrate skill is a one-shot upgrade tool for pre-`cfd5fb5` (2026-04-17) flat-format memory files. Its body text drives the agent through a 10-step flow: locate legacy data, read each file fully, classify as feedback or fact, rewrite hindsight-biased situations to task shape, build the new TOML using a feedback or fact template, derive `source` from legacy `confidence`/quotation, drop removed legacy fields unconditionally, write to the correct split destination, verify each migrated file via `engram show`, and finally archive (never delete) the legacy directory under a dated rename. R6 is the only outbound edge: the skill instructs the agent to invoke the engram CLI binary (E9) for verification of each rewritten file. R-edges cite the P-list each edge backs.

![C4 migrate-skill context diagram](svg/c4-migrate-skill.svg)

> Diagram source: [svg/c4-migrate-skill.mmd](svg/c4-migrate-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-migrate-skill.mmd -o architecture/c4/svg/c4-migrate-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

**Legend:**
- **Focus** — yellow (E14 migrate skill).
- **Component** — light blue (sibling skills E10–E13, E15).
- **Container** — darker blue (E9 engram CLI binary).
- **External** — grey (E3 Claude Code).
- **R-edges** — solid.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-trigger-surface"></a>P1 | Trigger surface | For all invocations matching `/migrate`, the phrase "migrate engram memories", "upgrade engram memories", or detection of legacy-shaped TOML files under `~/.local/share/engram/`, the skill body is loaded into context. | [skills/migrate/SKILL.md:3](../../skills/migrate/SKILL.md#L3) | **⚠ UNTESTED** | Trigger phrase enumerated in front-matter description; behavioral test absent. |
| <a id="p2-skip-current-format-files"></a>P2 | Skip current-format files | For all candidate files where the file has `schema_version = 2` and a `[content]` sub-table, the skill instructs the agent to skip the file (do not migrate). | [skills/migrate/SKILL.md:20](../../skills/migrate/SKILL.md#L20), [:42](../../skills/migrate/SKILL.md#L42) | **⚠ UNTESTED** | Prevents double-migration. |
| <a id="p3-legacy-field-detection-set"></a>P3 | Legacy field detection set | For all candidate files, the skill instructs the agent to treat the file as legacy if any top-level field is one of: `confidence`, `surfaced_count`, `followed_count`, `not_followed_count`, `irrelevant_count`, `keywords`, `concepts`, `principle`, `anti_pattern`, `project_scoped`, or `title`. | [skills/migrate/SKILL.md:15](../../skills/migrate/SKILL.md#L15), [:39](../../skills/migrate/SKILL.md#L39) | **⚠ UNTESTED** | Enumerated in both "When to use" and Step 1; the two lists must stay in sync. |
| <a id="p4-read-full-file-before-classifying"></a>P4 | Read full file before classifying | For all legacy files, the skill instructs the agent to read the file's full contents before deciding feedback-vs-fact; classification by filename alone is forbidden. | [skills/migrate/SKILL.md:22](../../skills/migrate/SKILL.md#L22), [:45](../../skills/migrate/SKILL.md#L45), [:193](../../skills/migrate/SKILL.md#L193) | **⚠ UNTESTED** | Reinforced under "When NOT to use", Step 2, and Red flags. |
| <a id="p5-feedback-classification-rule"></a>P5 | Feedback classification rule | For all legacy files exhibiting `behavior` + `impact` + `action`, or `principle` + `anti_pattern`, or content reading as "do X when Y" / "don't do X", the skill instructs the agent to classify the file as feedback. | [skills/migrate/SKILL.md:49](../../skills/migrate/SKILL.md#L49) | **⚠ UNTESTED** | Quick-reference table at line 165 mirrors the same rule. |
| <a id="p6-fact-classification-rule"></a>P6 | Fact classification rule | For all legacy files with subject–predicate–object shape or content reading as "X is Y" / "project Z uses tool W" / configuration-or-convention statements, the skill instructs the agent to classify the file as fact. | [skills/migrate/SKILL.md:54](../../skills/migrate/SKILL.md#L54) | **⚠ UNTESTED** |   |
| <a id="p7-stop-on-ambiguous-classification"></a>P7 | Stop on ambiguous classification | For all legacy files where feedback-vs-fact classification is unclear, the skill instructs the agent to STOP and ask the user rather than batch-classifying. | [skills/migrate/SKILL.md:59](../../skills/migrate/SKILL.md#L59), [:199](../../skills/migrate/SKILL.md#L199) | **⚠ UNTESTED** | Prevents misfiled memories that fail to surface for the right query. |
| <a id="p8-task-shaped-situation-rewrite"></a>P8 | Task-shaped situation rewrite | For all legacy `situation` fields that are hindsight-biased ("after I forgot…", "when we discovered…"), the skill instructs the agent to rewrite the situation as the task a future agent would be starting before writing the new file. | [skills/migrate/SKILL.md:26](../../skills/migrate/SKILL.md#L26), [:62](../../skills/migrate/SKILL.md#L62) | **⚠ UNTESTED** | Core principle of the skill — same hindsight-bias rule used by `/learn` and `/remember`. |
| <a id="p9-feedback-template-shape"></a>P9 | Feedback template shape | For all rewritten feedback files, the new file contains `schema_version = 2`, `type = "feedback"`, a `source`, a task-shaped `situation`, the legacy `created_at` preserved verbatim, an `updated_at` set to now (RFC3339 UTC), and a `[content]` sub-table with `behavior`, `impact`, and `action`. | [skills/migrate/SKILL.md:76](../../skills/migrate/SKILL.md#L76) | **⚠ UNTESTED** | behavior/impact/action may be derived from legacy `anti_pattern` / `principle` when explicit fields are absent. |
| <a id="p10-fact-template-shape"></a>P10 | Fact template shape | For all rewritten fact files, the new file contains `schema_version = 2`, `type = "fact"`, a `source`, a task-shaped `situation`, the legacy `created_at` preserved verbatim, an `updated_at` set to now (RFC3339 UTC), and a `[content]` sub-table with `subject`, `predicate`, and `object`. | [skills/migrate/SKILL.md:91](../../skills/migrate/SKILL.md#L91) | **⚠ UNTESTED** |   |
| <a id="p11-source-derivation-rules"></a>P11 | Source derivation rules | For all rewritten files, `source = "human"` iff the legacy content contains a direct quotation of the user OR legacy `confidence = "A"`; otherwise `source = "agent"`. Reported speech does not qualify as a direct quotation. | [skills/migrate/SKILL.md:107](../../skills/migrate/SKILL.md#L107) | **⚠ UNTESTED** | Default-to-agent bias is intentional: misclassified "agent" memories are corrected by users far more often than misclassified "human" ones. |
| <a id="p12-unconditional-drop-list"></a>P12 | Unconditional drop list | For all rewritten files, the legacy fields `surfaced_count`, `followed_count`, `not_followed_count`, `irrelevant_count`, `project_scoped`, `confidence`, `keywords`, `concepts`, and `title` are dropped — never carried forward, never re-encoded, and the user is not asked whether to keep them. | [skills/migrate/SKILL.md:117](../../skills/migrate/SKILL.md#L117), [:125](../../skills/migrate/SKILL.md#L125) | **⚠ UNTESTED** | Reinforced in Quick-reference and Common-mistakes tables. |
| <a id="p13-split-destination-by-type"></a>P13 | Split destination by type | For all rewritten files, feedback memories are written to `~/.local/share/engram/memory/feedback/<slug>.toml` and fact memories to `~/.local/share/engram/memory/facts/<slug>.toml`. | [skills/migrate/SKILL.md:129](../../skills/migrate/SKILL.md#L129) | **⚠ UNTESTED** | Matches the post-`cfd5fb5` split layout. |
| <a id="p14-slug-filename-rules"></a>P14 | Slug filename rules | For all rewritten files, the filename slug is the legacy filename if it is already kebab-case (lowercase letters, digits, hyphens only; ≤60 chars); otherwise it is regenerated from the new situation by lowercasing, replacing non-alphanumerics with `-`, collapsing repeats, and trimming to 60 chars. | [skills/migrate/SKILL.md:132](../../skills/migrate/SKILL.md#L132) | **⚠ UNTESTED** |   |
| <a id="p15-verify-each-migrated-file"></a>P15 | Verify each migrated file | For all migrated files, the skill instructs the agent to run `engram show --name <slug>` and fix any parse error before moving to the next file. | [skills/migrate/SKILL.md:138](../../skills/migrate/SKILL.md#L138), [:198](../../skills/migrate/SKILL.md#L198) | **⚠ UNTESTED** | Backs L3 R6: the skill's outbound call to E9 (engram CLI binary). |
| <a id="p16-build-engram-before-verifying-if-missing"></a>P16 | Build engram before verifying if missing | For all sessions in which the `engram` binary is not on `$PATH`, the skill instructs the agent to build it (`cd ~/repos/personal/engram && targ build`) and re-run verification rather than skipping verification. | [skills/migrate/SKILL.md:144](../../skills/migrate/SKILL.md#L144) | **⚠ UNTESTED** | Verification is non-negotiable. |
| <a id="p17-archive-never-delete-legacy-source"></a>P17 | Archive (never delete) legacy source | For all completed migrations where every file has been verified, the skill instructs the agent to rename the legacy directory to `memories.legacy-migrated-<YYYYMMDD>` rather than deleting it, and to retain the archive until at least one real session has surfaced the migrated memories via `/recall` and `/prepare`. | [skills/migrate/SKILL.md:147](../../skills/migrate/SKILL.md#L147), [:157](../../skills/migrate/SKILL.md#L157) | **⚠ UNTESTED** | Defense against silent loss if a classification was wrong. |
| <a id="p18-fully-qualified-paths"></a>P18 | Fully qualified paths | For all path references in the skill flow, paths are absolute (rooted at `~/.local/share/engram/`); bare `memory/feedback/` or `memory/facts/` are never interpreted as relative to the current working directory. | [skills/migrate/SKILL.md:161](../../skills/migrate/SKILL.md#L161) | **⚠ UNTESTED** | Caller-cwd hazard called out explicitly. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **E14 · migrate skill**)
- Siblings:
  - [c4-c4-skill.md](c4-c4-skill.md)
  - [c4-learn-skill.md](c4-learn-skill.md)
  - [c4-prepare-skill.md](c4-prepare-skill.md)
  - [c4-recall-skill.md](c4-recall-skill.md)
  - [c4-remember-skill.md](c4-remember-skill.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

