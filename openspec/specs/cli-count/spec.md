# Count Specification

## Purpose

`engram count` provides read-only aggregation and counting over the vault, distinct from `query`'s similarity-based recall. Two mutually exclusive modes support structured enumeration: frontmatter-based membership counts and wikilink in-degree metrics. Both modes verify Obsidian-checkable against their own native views (property/tag filters and backlinks panels respectively), and the two modes intentionally diverge on purpose — they measure different things. Why: docs/architecture/adr.md ADR-0018. Validation: internal/cli/count_test.go (TestRunCount_GroupByBacklinksAgreement, TestRunCount_BacklinksExceedGroupByForNonMemberLinkers).

## Requirements

### Requirement: Count distinct note membership per frontmatter attribute value

The command SHALL count how many notes carry each value of a frontmatter attribute using `--group-by <attr>`, grouping membership across the entire vault.

#### Scenario: Scalar attribute counting
- **WHEN** invoked with `--group-by type` on a vault with 349 feedback, 192 fact, and 8 qa-answer notes
- **THEN** output lists each value with its count, sorted by count descending then value ascending: `feedback	349\nfact	192\nqa-answer	8\ntotal: 557`

#### Scenario: List attribute counting with distinct-element deduplication
- **WHEN** a note's frontmatter contains `vocab: [x, y, z]` and is the only note with those values, and invoked with `--group-by vocab`
- **THEN** output counts each distinct list element once per note, not once per element occurrence: `x	1\ny	1\nz	1\ntotal: 1`

#### Scenario: Within-note value deduplication
- **WHEN** a note's frontmatter contains a list attribute with duplicate values, e.g. `vocab: [dup, dup, other]`
- **THEN** duplicates within the same note count only once: `dup	1\nother	1\ntotal: 1`, not `dup	2`

#### Scenario: Absent attribute handling
- **WHEN** invoked with `--group-by attr` and some notes lack that attribute in frontmatter
- **THEN** output includes a line `(attr absent): N` showing how many notes lack the attribute, before the total line

#### Scenario: All notes lack the attribute
- **WHEN** no note in the vault carries the specified attribute
- **THEN** output contains only `(attr absent): 557` and `total: 557`, printing nothing for any value bucket

### Requirement: Restrict membership count to notes matching AND-ed filter predicates

The command SHALL restrict membership counts to notes matching every `--filter attr=value` predicate (repeatable and AND-ed), excluding notes that fail any predicate.

#### Scenario: Scalar equality filtering
- **WHEN** invoked with `--group-by type --filter type=feedback`
- **THEN** output counts only notes where `type: feedback`, other types are excluded entirely

#### Scenario: List-contains filtering
- **WHEN** invoked with `--group-by tags --filter tags=tier/haiku` on a note with `tags: [tier/haiku, vocab/x]`
- **THEN** the note matches the filter because `tier/haiku` is in its list; filtering uses list-membership, not equality

#### Scenario: Multiple filters apply AND logic
- **WHEN** invoked with `--group-by outcome --filter type=fact --filter outcome=pass`
- **THEN** output counts only notes that match BOTH predicates (type is fact AND outcome is pass), rejecting notes that fail either predicate

#### Scenario: No notes match all filters
- **WHEN** invoked with `--group-by type --filter type=feedback` on a vault where no notes have that exact `type` value
- **THEN** output prints nothing (an empty result set produces no output lines, not even total)

### Requirement: Print wikilink in-degree and sorted linkers for a given note

The command SHALL print the wikilink in-degree of a note and list each linker in ascending alphabetical order when invoked with `--backlinks-of <basename>`.

#### Scenario: Note with incoming wikilinks
- **WHEN** invoked with `--backlinks-of foo.alpha` and three notes contain `[[foo.alpha]]`
- **THEN** output prints `in-degree: 3` followed by one linker per line in ascending alphabetical order

#### Scenario: Note with no incoming wikilinks
- **WHEN** invoked with `--backlinks-of unknown.note` and no note wikilinks to it
- **THEN** output prints `in-degree: 0` with no linker lines

#### Scenario: Non-existent basename
- **WHEN** invoked with `--backlinks-of nosuchfile` on a vault where that basename never appears
- **THEN** output prints `in-degree: 0` with no linker lines (same behavior as an unlinked note)

### Requirement: The two modes are mutually exclusive

The command SHALL reject requests combining `--group-by` and `--backlinks-of` in the same invocation, as they measure fundamentally different quantities.

#### Scenario: Both flags provided
- **WHEN** invoked with `--group-by type --backlinks-of note.name`
- **THEN** command errors with message `count: --group-by and --backlinks-of are mutually exclusive`

#### Scenario: No mode specified
- **WHEN** invoked without `--group-by` or `--backlinks-of`
- **THEN** command errors with message `count: specify --group-by <attr> or --backlinks-of <basename>`

### Requirement: Group-by and backlinks-of modes intentionally diverge for non-member linkers

Group-by and backlinks-of modes SHALL diverge in count when a non-member linker (a note linking to a target without carrying that target's attribute in its own frontmatter) exists: backlinks-of counts the linker while group-by excludes it, and the relationship is `in-degree == group-by count + (# non-member linkers)`.

#### Scenario: Clean agreement when all linkers are members
- **WHEN** all notes linking to a node carry that node's attribute value in their frontmatter
- **THEN** `--group-by attr` count for that value equals `--backlinks-of node` in-degree

#### Scenario: Backlinks exceed group-by for non-member linkers
- **WHEN** `n1.md` has `foo: [alpha]` and links `[[foo.alpha]]`, and `idx.md` links `[[foo.alpha]]` but has NO `foo` attribute
- **THEN** `--group-by foo` counts 1 (only n1.md), `--backlinks-of foo.alpha` reports in-degree 2 (both n1.md and idx.md), and the relationship is `in-degree == group-by count + (# non-member linkers)`

