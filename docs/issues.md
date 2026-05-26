# engram issues

Active design / implementation items that don't yet warrant a
spec or roadmap. Each item names *what* the issue is, *why* it
matters, and the minimum that would close it. Move resolved items
to the bottom under "Closed" with the closing commit SHA.

## Open

### 1. Project/issue metadata on vault notes

**What.** Add a `project` field (and optionally `issue`) to the
frontmatter of facts, feedback, and episodes. Relax the strict
"no project names in `situation`" rule so a future query like
"what did we learn on the engram project last month" is
answerable.

**Why.** Today's vault rule treats all notes as if they were
universal principles. In practice many notes are project-bound;
they retain value, but the projectness is implicit in the
content and not queryable as metadata. Joe surfaced this in the
2026-05-26 /please inline-notes pass on `/recall` SKILL.md
(criterion 3 of the synthesis gate: "Principle is generalizable,
not project-specific").

**What "closed" looks like.**

- `engram learn fact|feedback|episode --project <slug> [--issue <id>]`
  flags added. Slug pattern: kebab-case, repository-name shape.
- Frontmatter renders `project: <slug>` (and optionally
  `issue: <id>`) below `source`.
- `/learn` SKILL guidance updated: project name is welcome in
  `--project`; situation phrasing stays retrieval-shaped (still
  no `engram` in the situation, but `project: engram` makes
  cross-project filtering possible).
- `engram query` gains optional `--project <slug>` filter that
  restricts items to notes with matching `project:` field.
- README + GLOSSARY surface the new field.

**Why not yet.** v2 closed without this; capturing as an issue
to revisit when the cross-project query is something we
actually want to do.

## Closed

(none yet)
