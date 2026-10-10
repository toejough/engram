# Vault Counter - Technical Spec

## 792b25a Orientation

- Ownership: Repository component.
- Responsibility: turns the notes at a vault's root into exact counts. It counts the notes that
  carry each value of a frontmatter attribute, optionally restricted by attribute filters, and it
  reports how many notes link to a named note and which ones. It reads notes and never writes.
- Current realization (informational): `internal/cli/count.go` (`RunCount`, `CountArgs`, `CountDeps`).
- Qualification: responsibility, counting by attribute value and by incoming link is stated without
  naming any function; ownership, the [command-line-surface](./command-line-surface.md) decides when to run it and how to report
  its error, while this component decides what is counted, in what order, and what is skipped;
  independent verification, it is tested directly over a fake vault listing and fake note contents;
  stability, an exact count over the whole vault is a lasting need that a change to recall does not
  touch; necessity, `engram count` is this contract.
- Supported capabilities: [vault-counting](../product/vault-counting.md). The [route](../product/route.md) capability extends it by using it as its
  audit surface.

Rationale: a count and a recall answer different questions. Recall returns a similarity-ranked,
truncated sample of the vault, so it cannot say how many notes carry a tag or link to a note.
Counting is therefore a separate, read-only surface that never embeds, scores cosine similarity or
clusters. Its design goal is that each mode can be checked by hand against its own
Obsidian view: `--group-by` against a property or tag filter, `--backlinks-of` against the
backlinks panel.

## 792b25c Behavior Requirements

### 792b25c1 Mode selection

Requirement: [792b25c1 Mode selection](#792b25c1-mode-selection) The vault-counter MUST fail with `count: --group-by and --backlinks-of are
mutually exclusive` when both an attribute and a basename are given, and MUST fail with `count:
specify --group-by <attr> or --backlinks-of <basename>` when neither is given, whatever filters
accompany it. Otherwise it MUST run the incoming-link report ([792b25c6 Incoming-link report](#792b25c6-incoming-link-report)) when a basename is given
and the attribute count ([792b25c2 Attribute membership counting](#792b25c2-attribute-membership-counting)) when an attribute is given.

Supports: [792a8f1a Both flags are refused](../product/vault-counting.md#792a8f1-uat-scenarios), [792a8f1b Neither flag is refused](../product/vault-counting.md#792a8f1-uat-scenarios), [792a8f1c A filter alone is not a mode](../product/vault-counting.md#792a8f1-uat-scenarios).

Rationale: the two modes measure different quantities ([792b25c7 The two measures are independent](#792b25c7-the-two-measures-are-independent)), so one run does not mix them.

#### 792b25c1a Scenarios

```gherkin
Scenario: 792b25c1a1 Both modes are refused
  Given the attribute type and the basename note.name
  When the vault is counted
  Then it fails with "count: --group-by and --backlinks-of are mutually exclusive" and prints nothing

Scenario: 792b25c1a2 No mode is refused
  Given neither an attribute nor a basename, with or without a filter
  When the vault is counted
  Then it fails with "count: specify --group-by <attr> or --backlinks-of <basename>" and prints nothing

Scenario: 792b25c1a3 A basename alone selects the incoming-link report
  Given only a basename
  When the vault is counted
  Then the incoming-link report is printed

Scenario: 792b25c1a4 An attribute alone selects the attribute count
  Given only an attribute
  When the vault is counted
  Then the attribute count is printed
```

### 792b25c2 Attribute membership counting

Requirement: [792b25c2 Attribute membership counting](#792b25c2-attribute-membership-counting) The vault-counter MUST, given an attribute, count each note that is readable
([792b25c4 Uncountable notes](#792b25c4-uncountable-notes)) and satisfies every filter ([792b25c3 Filters](#792b25c3-filters)) once in the total. A scalar value MUST contribute
its text as the note's one value, a list MUST contribute each distinct element once, and a null or
empty value MUST contribute none. It MUST count a note that has no such key as absent, and MUST
count the notes per value. It MUST return a failure to list the vault wrapped as `count: listing
vault: <error>`.

Supports: [792a8a1a A scalar attribute is counted per value](../product/vault-counting.md#792a8a1-uat-scenarios), [792a8a1b A list attribute counts each distinct element once per note](../product/vault-counting.md#792a8a1-uat-scenarios), [792a8a1c A repeated element counts once](../product/vault-counting.md#792a8a1-uat-scenarios), [792a8a1e Only notes at the vault root are counted](../product/vault-counting.md#792a8a1-uat-scenarios), [792a8b1a Notes without the attribute are reported](../product/vault-counting.md#792a8b1-uat-scenarios), [792a8b1b No note has the attribute](../product/vault-counting.md#792a8b1-uat-scenarios), [792a8b1d An empty attribute counts only toward the total](../product/vault-counting.md#792a8b1-uat-scenarios).

#### 792b25c2a Scenarios

```gherkin
Scenario: 792b25c2a1 A scalar value is counted per note
  Given notes with type feedback, fact and qa-answer
  When the vault is counted by type
  Then each value's count is the number of notes carrying it

Scenario: 792b25c2a2 List elements are counted per note
  Given a note with vocab [x, y, z] and no other note with vocab values
  When the vault is counted by vocab
  Then x, y and z each count 1 and the total is 1

Scenario: 792b25c2a3 A repeated element counts once
  Given a note with vocab [dup, dup, other]
  When the vault is counted by vocab
  Then dup counts 1, other counts 1 and the total is 1

Scenario: 792b25c2a4 An empty value counts only toward the total
  Given a note with the key foo and no value, a note with foo: [], and a note with foo: 3
  When the vault is counted by foo
  Then 3 counts 1, nothing is absent and the total is 3

Scenario: 792b25c2a5 A note without the key is absent
  Given a note with vocab [x] and a note with no vocab key
  When the vault is counted by vocab
  Then x counts 1, one note is absent and the total is 2

Scenario: 792b25c2a6 A listing failure is returned
  Given a vault whose listing fails
  When the vault is counted by type
  Then it fails with an error beginning "count: listing vault:"
```

### 792b25c3 Filters

Requirement: [792b25c3 Filters](#792b25c3-filters) The vault-counter MUST read each filter as an attribute and a value split at
the first `=`, MUST count a note only when it satisfies every filter, and MUST treat a note as
satisfying a filter when the attribute's distinct values include the value. A note without the
attribute MUST NOT satisfy it. It MUST fail with `count: --filter must be attr=value: "<filter>"`
for a filter that has no `=`, before it lists the vault.

Supports: [792a8c1a A scalar filter matches by equality](../product/vault-counting.md#792a8c1-uat-scenarios), [792a8c1b A list filter matches by membership](../product/vault-counting.md#792a8c1-uat-scenarios), [792a8c1c Every filter must match](../product/vault-counting.md#792a8c1-uat-scenarios), [792a8c1d A note without the filter's attribute is excluded](../product/vault-counting.md#792a8c1-uat-scenarios), [792a8c1f A filter without = is refused](../product/vault-counting.md#792a8c1-uat-scenarios).

#### 792b25c3a Scenarios

```gherkin
Scenario: 792b25c3a1 A scalar filter matches by equality
  Given notes of type feedback and notes of type fact
  When the vault is counted by type with the filter type=feedback
  Then only feedback is counted

Scenario: 792b25c3a2 A list filter matches by membership
  Given a note with tags [tier/haiku, vocab/x]
  When the vault is counted by tags with the filter tags=tier/haiku
  Then the note is counted

Scenario: 792b25c3a3 Every filter must match
  Given a fact with outcome pass, a fact with outcome fail, and a feedback with outcome pass
  When the vault is counted by outcome with the filters type=fact and outcome=pass
  Then only the first note is counted

Scenario: 792b25c3a4 A note without the attribute does not match
  Given a note with no outcome key
  When the vault is counted by type with the filter outcome=pass
  Then the note is not counted

Scenario: 792b25c3a5 A filter without = fails
  Given the filter "bad"
  When the vault is counted by type
  Then it fails with "count: --filter must be attr=value" quoting "bad"
```

### 792b25c4 Uncountable notes

Requirement: [792b25c4 Uncountable notes](#792b25c4-uncountable-notes) The vault-counter MUST skip a note whose file cannot be read, a note with no
frontmatter, and a note whose frontmatter is not a valid YAML mapping, without counting it as a
value, as absent or in the total. It MUST read a note with CRLF line endings as its LF form.

Supports: [792a8d1a A note without usable frontmatter is skipped](../product/vault-counting.md#792a8d1-uat-scenarios), [792a8d1b An unreadable note is skipped](../product/vault-counting.md#792a8d1-uat-scenarios), [792a8d1c A CRLF note is counted](../product/vault-counting.md#792a8d1-uat-scenarios).

Rationale: one stray file must not stop a count of the rest. A frontmatter split that is CRLF-aware
would hand writers mixed line endings, so notes are read as LF ([792b6c8 CRLF reading](./note-document-format.md#792b6c8-crlf-reading)).

#### 792b25c4a Scenarios

```gherkin
Scenario: 792b25c4a1 A note without usable frontmatter is skipped
  Given a well-formed note, a note with no frontmatter, and a note with malformed YAML frontmatter
  When the vault is counted by type
  Then only the well-formed note is counted

Scenario: 792b25c4a2 An unreadable note is skipped
  Given a listed note whose file cannot be read
  When the vault is counted by type
  Then nothing is counted and no error is returned

Scenario: 792b25c4a3 A CRLF note is read as LF
  Given a fact note whose lines end in \r\n
  When its frontmatter is read for counting
  Then its type is fact
```

### 792b25c5 Attribute count report

Requirement: [792b25c5 Attribute count report](#792b25c5-attribute-count-report) The vault-counter MUST report an attribute count as one line per value, the
value, a tab and its count, ordered by count descending and then value ascending, then the line
`(<attr> absent): <N>` when `<N>` is above zero, then the line `total: <T>`. It MUST report nothing
when no note was counted.

Supports: [792a8a1a A scalar attribute is counted per value](../product/vault-counting.md#792a8a1-uat-scenarios), [792a8a1b A list attribute counts each distinct element once per note](../product/vault-counting.md#792a8a1-uat-scenarios), [792a8a1d Equal counts are ordered by value](../product/vault-counting.md#792a8a1-uat-scenarios), [792a8b1a Notes without the attribute are reported](../product/vault-counting.md#792a8b1-uat-scenarios), [792a8b1b No note has the attribute](../product/vault-counting.md#792a8b1-uat-scenarios), [792a8b1c An empty vault prints nothing](../product/vault-counting.md#792a8b1-uat-scenarios), [792a8c1e No note matches](../product/vault-counting.md#792a8c1-uat-scenarios).

#### 792b25c5a Scenarios

```gherkin
Scenario: 792b25c5a1 Values are ordered by count, then value
  Given counts of 349 for feedback, 192 for fact, 8 for qa-answer, and 2 each for b and a
  When the counts are reported
  Then the lines are feedback, fact, qa-answer, a, b

Scenario: 792b25c5a2 The absent line precedes the total
  Given one note with vocab [x] and one without vocab
  When the counts are reported
  Then the output is "x", a tab and 1, then "(vocab absent): 1", then "total: 2"

Scenario: 792b25c5a3 Every note is absent
  Given notes none of which has the attribute
  When the counts are reported
  Then the output is only the absent line and the total

Scenario: 792b25c5a4 Nothing counted prints nothing
  Given no note passes the filters
  When the counts are reported
  Then nothing is printed, not even a total

Scenario: 792b25c5a5 The total counts notes, not values
  Given one note with three values
  When the counts are reported
  Then the total is 1
```

### 792b25c6 Incoming-link report

Requirement: [792b25c6 Incoming-link report](#792b25c6-incoming-link-report) The vault-counter MUST scan the vault ([792b13c1 Vault scan](./wikilink-graph.md#792b13c1-vault-scan)), build the link graph
([792b13c4 Link graph](./wikilink-graph.md#792b13c4-link-graph)) and print `in-degree: <N>` with the named note's in-degree ([792b13c5 In-degree](./wikilink-graph.md#792b13c5-in-degree)), then the basenames
of the notes that link to it, one per line, in ascending order. It MUST print an in-degree of 0 and
no linkers for a basename that no note links to or that is not a scanned note. It MUST return a
scan failure wrapped as `count: scan: <error>`, printing nothing.

Supports: [792a8e1a A note with incoming links](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1b A note with no incoming links](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1c A basename that is not a note](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1d A repeated link counts once](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1e A note's link to itself does not count](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1f A link inside a fenced code block does not count](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1g A link ending in .md counts as the bare link](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1h A link to a missing note counts for no note](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8e1i Only notes at the vault root link](../product/vault-counting.md#792a8e1-uat-scenarios),
[792a8e1j An unreadable note fails the report](../product/vault-counting.md#792a8e1-uat-scenarios).

#### 792b25c6a Scenarios

```gherkin
Scenario: 792b25c6a1 The linkers are listed in ascending order
  Given three notes that link to foo.alpha
  When the incoming links of foo.alpha are reported
  Then the output is "in-degree: 3" followed by the three basenames in ascending order

Scenario: 792b25c6a2 A note nothing links to
  Given no note links to unknown.note
  When the incoming links of unknown.note are reported
  Then the output is "in-degree: 0" and no linker line

Scenario: 792b25c6a3 A basename that is not a note
  Given no note named nosuchfile
  When the incoming links of nosuchfile are reported
  Then the output is "in-degree: 0" and no linker line

Scenario: 792b25c6a4 A scan failure is returned
  Given a scan that fails
  When the incoming links of C are reported
  Then it fails with an error beginning "count: scan:"
```

### 792b25c7 The two measures are independent

Requirement: [792b25c7 The two measures are independent](#792b25c7-the-two-measures-are-independent) The vault-counter MUST count by attribute only the notes that carry the value
([792b25c2 Attribute membership counting](#792b25c2-attribute-membership-counting)) and MUST count by incoming link every note that links ([792b25c6 Incoming-link report](#792b25c6-incoming-link-report)), so that a node's
in-degree equals the attribute count of its value plus the number of linkers that do not carry it,
when every note that carries the value links to the node.

Supports: [792a8g1a The modes agree when every linker is a member](../product/vault-counting.md#792a8g1-uat-scenarios), [792a8g1b The in-degree exceeds the count by the non-member linkers](../product/vault-counting.md#792a8g1-uat-scenarios).

#### 792b25c7a Scenarios

```gherkin
Scenario: 792b25c7a1 The measures agree when every linker is a member
  Given notes n1 and n2 with foo [alpha] that each link foo.alpha, and the note foo.alpha
  When the vault is counted by foo and the incoming links of foo.alpha are reported
  Then the attribute count of alpha and the in-degree are both 2

Scenario: 792b25c7a2 The in-degree exceeds the count by the non-member linkers
  Given n1 with foo [alpha] that links foo.alpha, idx with no foo that links foo.alpha, and the note foo.alpha
  When the vault is counted by foo and the incoming links of foo.alpha are reported
  Then the attribute count of alpha is 1, the in-degree is 2, and the linkers are idx and n1
```

### 792b25c8 No writes

Requirement: [792b25c8 No writes](#792b25c8-no-writes) The vault-counter MUST NOT create, modify or delete any file, and MUST count
from the notes alone, without a sidecar or an embedding.

Supports: [792a8h1a A count leaves the vault as it was](../product/vault-counting.md#792a8h1-uat-scenarios), [792a8h1b Every note is counted without sidecars](../product/vault-counting.md#792a8h1-uat-scenarios).

Rationale: counting is read-only and off the recall path. The component is given only
listing and reading capabilities, so it cannot write.

#### 792b25c8a Scenarios

```gherkin
Scenario: 792b25c8a1 A count writes nothing
  Given a vault of notes without sidecars
  When it is counted in each mode
  Then no file is created, changed or deleted

Scenario: 792b25c8a2 Notes are counted without sidecars
  Given 600 notes and no sidecar
  When the vault is counted by type
  Then all 600 are counted
```
