# Luhmann Addressing - Technical Spec

## 792b7a Orientation

- Ownership: Repository component.
- Responsibility: the Luhmann ID scheme. It parses an ID into its alternating number and letter
  segments, reads the ID that leads a note's filename, orders letter segments, and allocates the
  next unused ID for a new note from the existing IDs, a target and a position.
- Current realization (informational): `ParseID`, `FromBasename` and `LetterLess` in
  `internal/luhmann/luhmann.go`; `nextLuhmannID` and its helpers in `internal/cli/luhmann.go`;
  `extractLuhmannFromFilename` in `internal/cli/learn.go`. The parse and order functions are a
  shared kernel: [note-authoring](./note-authoring.md), the vault scanner, Luhmann reparenting and the exchange code all
  use them.
- Qualification: responsibility, the scheme is stated without naming any caller; ownership, callers
  choose the target and position, while this component decides what ID they yield; independent
  verification, parsing, ordering and allocation are pure functions tested directly; stability, the
  scheme is the vault's addressing convention and outlives any refactoring; necessity, every fact,
  feedback and runbook note is placed by it.
- Supported capabilities: [note-writing](../product/note-writing.md). Vault-migration ([vault-migration](../product/vault-migration.md)) adds reparenting's needs.

Rationale: Luhmann IDs must be unique across the vault and well formed, a digit-led run of digits
and letters. The allocator only ever returns an ID that no existing note holds; two concurrent
writers are kept from allocating the same one by the [vault-write-lock](./vault-write-lock.md).

## 792b7c Behavior Requirements

### 792b7c1 ID parsing

Requirement: [792b7c1 ID parsing](#792b7c1-id-parsing) The luhmann-addressing MUST split an ID into its alternating runs of digits and
letters, MUST refuse an empty ID, and MUST refuse an ID that does not start with a digit.

Supports: [792a3j1a Top takes the next top-level number](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1b A continuation of a number takes the next letter](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1c A continuation of a letter takes the next number](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1d A sibling takes the next child of the target's parent](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1e Letters roll over from z to aa](../product/note-writing.md#792a3j1-uat-scenarios), [792a3k1d A malformed target is refused](../product/note-writing.md#792a3k1-uat-scenarios).

#### 792b7c1a Scenarios

```gherkin
Scenario: 792b7c1a1 Segments alternate
  Given the IDs 1a3b and 12ab3
  When they are parsed
  Then they split into 1, a, 3, b and into 12, ab, 3

Scenario: 792b7c1a2 A top-level ID is one segment
  Given the ID 1
  When it is parsed
  Then it is the single segment 1

Scenario: 792b7c1a3 An empty ID is refused
  Given an empty ID
  When it is parsed
  Then the empty-ID error is returned

Scenario: 792b7c1a4 A leading letter is refused
  Given the ID a1
  When it is parsed
  Then the leading-letter error is returned
```

### 792b7c2 ID from a filename

Requirement: [792b7c2 ID from a filename](#792b7c2-id-from-a-filename) The luhmann-addressing MUST read a note's ID as the part of its `.md`
filename before the first dot, when that part starts with a digit and holds only digits and
letters, and MUST report no ID for any other filename.

Supports: [792a3j1a Top takes the next top-level number](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1b A continuation of a number takes the next letter](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1c A continuation of a letter takes the next number](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1d A sibling takes the next child of the target's parent](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1e Letters roll over from z to aa](../product/note-writing.md#792a3j1-uat-scenarios).

#### 792b7c2a Scenarios

```gherkin
Scenario: 792b7c2a1 The leading ID is read
  Given the filenames 1a3.2026-05-09.subagent-recovery.md and 9o1.2026-05-10.slug
  When their IDs are read
  Then they are 1a3 and 9o1

Scenario: 792b7c2a2 A filename without a leading ID has none
  Given README.md, a name starting with a dot, a name with no dot, or a name not starting with a digit
  When its ID is read
  Then no ID is reported

Scenario: 792b7c2a3 A file that is not Markdown has no ID
  Given 1a3.2026-05-09.subagent-recovery.txt
  When its ID is read
  Then no ID is reported
```

### 792b7c3 Letter order

Requirement: [792b7c3 Letter order](#792b7c3-letter-order) The luhmann-addressing MUST order letter segments shorter first, then
alphabetically, so that `z` comes before `aa`, `az` before `ba`, and `zz` before `aaa`.

Supports: [792a3j1e Letters roll over from z to aa](../product/note-writing.md#792a3j1-uat-scenarios).

Rationale: an allocator once compared letter segments lexically, so with children `a` to `z`, `aa`
and `ab`, it took `z` as the largest and allocated `aa` again.

#### 792b7c3a Scenarios

```gherkin
Scenario: 792b7c3a1 Shorter letter segments come first
  Given the letter segments z, aa, ab, az, ba, zz and aaa
  When they are compared
  Then z < aa < ab, az < ba, zz < aaa, and no segment is less than itself
```

### 792b7c4 Next-ID allocation

Requirement: [792b7c4 Next-ID allocation](#792b7c4-next-id-allocation) Given the existing IDs, the luhmann-addressing MUST allocate, for `top`, one
more than the largest top-level number, or `1` when there is none; for `continuation`, the next
child of the target, counting only the target's direct children: after an ID ending in a number,
the letter after its largest letter child in Luhmann order, or `a`, and after an ID ending in a
letter, the number after its largest numeric child, or `1`; and for `sibling`, the next child of
the target's parent.

Supports: [792a3j1a Top takes the next top-level number](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1b A continuation of a number takes the next letter](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1c A continuation of a letter takes the next number](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1d A sibling takes the next child of the target's parent](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1e Letters roll over from z to aa](../product/note-writing.md#792a3j1-uat-scenarios), [792a10e1a A continuation answer renames the note and rewrites links](../product/vault-migration.md#792a10e1-uat-scenarios), [792a10e1b Renames are ordered by original ID](../product/vault-migration.md#792a10e1-uat-scenarios).

#### 792b7c4a Scenarios

```gherkin
Scenario: 792b7c4a1 Top takes the next top-level number
  Given the IDs 1, 1a, 2 and 2a, or no IDs at all
  When an ID is allocated at top
  Then it is 3, or 1 for no IDs

Scenario: 792b7c4a2 A continuation of a number takes the next letter
  Given the IDs 1 and 2, the IDs 1, 1a and 1b, or the IDs 1, 2, 2a and 1a3
  When an ID is allocated as a continuation of 1
  Then it is 1a, 1c and 1a respectively, ignoring IDs that are not direct children of 1

Scenario: 792b7c4a3 A continuation of a letter takes the next number
  Given the IDs 1 and 1a, or the IDs 1, 1a, 1a1 and 1a2
  When an ID is allocated as a continuation of 1a
  Then it is 1a1 or 1a3 respectively

Scenario: 792b7c4a4 Letters roll over
  Given the IDs 1 and 1a through 1z, optionally with 1aa and 1ab
  When an ID is allocated as a continuation of 1
  Then it is 1aa, or 1ac when 1aa and 1ab exist

Scenario: 792b7c4a5 A sibling takes the next child of the parent
  Given the IDs 1 and 1a
  When an ID is allocated as a sibling of 1a
  Then it is 1b
```

### 792b7c5 Allocation refusals

Requirement: [792b7c5 Allocation refusals](#792b7c5-allocation-refusals) The luhmann-addressing MUST refuse to allocate for a position other than `top`,
`continuation` or `sibling`, for a `continuation` or `sibling` with no target or with a target that
is not a valid ID, and for a `sibling` of a top-level ID.

Supports: [792a3k1a A missing position is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3k1b A continuation without a target is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3k1c A sibling of a top-level note is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3k1d A malformed target is refused](../product/note-writing.md#792a3k1-uat-scenarios).

#### 792b7c5a Scenarios

```gherkin
Scenario: 792b7c5a1 An unknown position is refused
  Given the position bogus
  When an ID is allocated
  Then an error is returned

Scenario: 792b7c5a2 A missing target is refused
  Given a continuation or a sibling with no target
  When an ID is allocated
  Then an error is returned

Scenario: 792b7c5a3 A malformed target is refused
  Given a continuation or a sibling whose target is abc
  When an ID is allocated
  Then an error is returned

Scenario: 792b7c5a4 A sibling of a top-level ID is refused
  Given the IDs 1 and 2
  When an ID is allocated as a sibling of 1
  Then an error is returned
```

### 792b7c6 ID and date from a filename

Requirement: [792b7c6 ID and date from a filename](#792b7c6-id-and-date-from-a-filename) The luhmann-addressing MUST read from a note filename of the form `<id>.<date>.<slug>.md`
its ID, as in [792b7c2 ID from a filename](#792b7c2-id-from-a-filename), and its date, the second dot-separated segment, and MUST refuse a filename that has no
valid leading ID or fewer than three dot-separated segments.

Supports: [792a10e1a A continuation answer renames the note and rewrites links](../product/vault-migration.md#792a10e1-uat-scenarios).

Rationale: a re-parented note keeps its date and slug and takes a new ID, so the new basename is built from
these parts.

#### 792b7c6a Scenarios

```gherkin
Scenario: 792b7c6a1 A note filename yields its ID and date
  Given the filename 211.2026-07-10.vocab-retrieval-design-definition.md
  When its ID and date are read
  Then they are 211 and 2026-07-10

Scenario: 792b7c6a2 A filename without a leading ID is refused
  Given the filename not-an-id.2026-07-10.vocab-x-definition.md
  When its ID and date are read
  Then it is refused

Scenario: 792b7c6a3 A filename with too few segments is refused
  Given the filename 211.md
  When its ID and date are read
  Then it is refused
```
