# Doc-surface enumeration: fix-show-amend-reparent-frontmatter

This change alters four behaviors that are documented:

- `engram show` returns every red_flag, and the 1200 bytes becomes a query *preview budget*.
- amend keeps the prior `user:` when detection comes back empty.
- The reparent payload defers to learn's batch mode and its hand-back and `--dry-run` flow.
- Adopt and refresh keep unknown keys, and rename refuses unsafe frontmatter.

**Search** (2026-10-01, worktree `runbook-vs-skill`, branch `defect-batch-show-amend-reparent-frontmatter` @ `b48d6557`):

```
grep -rn -E "1200|rendered cap|OMITTED|engram show|reparent-luhmann|preserves every|user: \"\"|CRLF" \
  --include='*.md' agent-instructions docs README.md openspec/specs
```

**Excluded:**

- `openspec/changes/archive/`, which is history;
- `dev/eval/`, which holds frozen fixtures and results;
- `docs/research/`, which holds dated notes.

The C4 docs (`docs/architecture/c1-system-context.md` and `c2-containers.md`) mention only `engram show-chunk`, which is unaffected. Every row was read at its cited line.

**Disposition key:** *rewrite*, *append*, *no change* (with the reason), and *spec delta* (handled by this change's `specs/`, applied at archive).

| # | Surface | Line | Disposition | New-present grep | Old-absent grep |
|---|---|---|---|---|---|
| 1 | `agent-instructions/skills/curate/SKILL.md` | 37 | rewrite: "total rendered under 1200 bytes" → "aim for the 1200-byte `engram query` preview budget (rendered YAML bytes); never drop a valid red flag to fit" | `grep -n "preview budget" agent-instructions/skills/curate/SKILL.md` | `grep -n "total rendered under 1200 bytes" agent-instructions/skills/curate/SKILL.md` → none |
| 2 | `agent-instructions/skills/curate/SKILL.md` | 43 | rewrite: "keeping `red_flags` under the 1200-byte rendered cap" → the preview-budget wording from row 1 | `grep -n "rendered YAML bytes" agent-instructions/skills/curate/SKILL.md` | `grep -n "rendered cap" agent-instructions/skills/curate/SKILL.md` → none |
| 3 | `openspec/specs/vault-offer-curation/spec.md` | 31 | spec delta | `grep -n "preview budget" openspec/specs/vault-offer-curation/spec.md` | `grep -n "1200-byte rendered cap" openspec/specs/vault-offer-curation/spec.md` → none |
| 4 | `openspec/specs/recall-runbook-surfacing/spec.md` | 42-54 | spec delta | `grep -n "Show never truncates red flags" openspec/specs/recall-runbook-surfacing/spec.md` | `grep -n "is not exempt from this guarantee" openspec/specs/recall-runbook-surfacing/spec.md` → none |
| 5 | `agent-instructions/guidance/shim.md` | 98-102 (rules 6, 7) | no change: both rules already name `engram show` as the full-body source, which D1 makes true | — | — |
| 6 | `openspec/specs/guidance-runbook-follow-frame/spec.md` | 111 | no change: same as row 5 | — | — |
| 7 | `README.md` | 93 | no change: "Print a note (frontmatter + body)" is now accurate | — | — |
| 8 | `docs/GLOSSARY.md` | 866-869 (Derive bullet) | append: the payload's `instruction` names learn's batch mode, and the answers file is handed back for a `--dry-run` preview and then the apply | `grep -n "instruction.*batch mode" docs/GLOSSARY.md` | — |
| 9 | `docs/GLOSSARY.md` | 893-895 | rewrite: "so an agent can drive the whole derive→judge→apply loop … without any further manual step" → state that the user or orchestrating agent runs each apply after the learn batch-mode pass hands back the answers | `grep -n "hands back the answers" docs/GLOSSARY.md` | `grep -n "further manual step" docs/GLOSSARY.md` → none |
| 10 | `agent-instructions/skills/learn/SKILL.md` | 256-299 | no change: canonical under D4 (Option 1) | — | — |
| 11 | `openspec/specs/update-reparent-luhmann-batch/spec.md` | 20-37 and the new requirement | spec delta | `grep -n "Rename and rewrite SHALL refuse unsafe frontmatter" openspec/specs/update-reparent-luhmann-batch/spec.md` | — |
| 12 | `openspec/specs/update-flat-vault-luhmann-notice/spec.md` | 30-40 | spec delta | `grep -n "Notice names the answering procedure" openspec/specs/update-flat-vault-luhmann-notice/spec.md` | — |
| 13 | `openspec/specs/vault-note-identity/spec.md` | 46-67 | spec delta | `grep -n "Empty user detection keeps the prior user" openspec/specs/vault-note-identity/spec.md` | — |
| 14 | `openspec/specs/skill-runbook-registration/spec.md` | 95-111, 146-158 | spec delta | `grep -n "keeps an unmodeled key" openspec/specs/skill-runbook-registration/spec.md` (2 hits) | — |
| 15 | `docs/architecture/adr.md` | 1232 | no change: "`resituate` preserves every field it doesn't change" is about resituate, which is out of scope (design Open Question 5) | — | — |
| 16 | `docs/ROADMAP.md` | 92, 96, 97, 100 (#780, #772, #776, #770) | rewrite at close-out: mark done, citing this change; record that the #776 `vault:` part was already fixed | `grep -n "fix-show-amend-reparent-frontmatter" docs/ROADMAP.md` | — |
| 17 | `internal/cli/skillreg_accept.go` docstrings | 92, 172, 317-322 | rewrite (code comments, task 5.3): "preserves every frontmatter key it doesn't set; comments best-effort" | `grep -n "comments best-effort" internal/cli/skillreg_accept.go` | `grep -n "preserves every other frontmatter field" internal/cli/skillreg_accept.go` → none |
| 18 | Vault note `skill-claude-curate`, which mirrors curate SKILL.md | — | no direct edit: `engram update` raises a refresh offer after rows 1-2 deploy, and the offer is left for Joe | — | — |
