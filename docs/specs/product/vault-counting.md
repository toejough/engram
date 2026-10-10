# Vault Counting - Product Spec

`engram count` answers "how many?" over the whole vault: how many notes carry each value of a
frontmatter attribute, and how many notes link to a given note. It reads only the Markdown notes at
the vault's root and changes nothing. It runs against the vault located by [792a2a Vault location](./vault-identity.md#792a2a-vault-location), and a vault that
does not exist is created first ([792a2e First-use vault creation](./vault-identity.md#792a2e-first-use-vault-creation)). `engram query` answers a different question, "what is
relevant to this phrase", and returns a ranked, bounded sample of the vault ([memory-query](./memory-query.md)). The `route`
skill uses `count` to audit the evidence tallies it reports.

## 792a8a Count notes by attribute value

Requirement: [792a8a Count notes by attribute value](#792a8a-count-notes-by-attribute-value) `engram count --group-by <attr>` MUST print, for each distinct value of the
frontmatter attribute `<attr>` among the notes at the vault root, one line `<value>`, a tab and the
number of notes that carry the value, ordered by that number descending and then by value
ascending. A list attribute MUST contribute each of its distinct elements once, and a note that
repeats an element MUST count once for it.

Rationale: a question such as "how many notes carry this tag" needs an exact count over the whole
vault. A query returns a similarity-ranked, truncated sample with no sign that it is not the full
set, so counting does not ride on it.

### 792a8a1 UAT Scenarios

```gherkin
Scenario: 792a8a1a A scalar attribute is counted per value
  Given a vault of 349 feedback notes, 192 fact notes and 8 qa-answer notes
  When the user runs engram count --group-by type
  Then stdout is "feedback", a tab and 349, then "fact", a tab and 192, then "qa-answer", a tab and 8, then the line "total: 549"

Scenario: 792a8a1b A list attribute counts each distinct element once per note
  Given the only note with vocab values is one whose frontmatter has vocab: [x, y, z]
  When the user runs engram count --group-by vocab
  Then each of x, y and z is printed with the count 1, followed by "total: 1"

Scenario: 792a8a1c A repeated element counts once
  Given a note whose frontmatter has vocab: [dup, dup, other]
  When the user runs engram count --group-by vocab
  Then dup and other are each printed with the count 1, not dup with 2, followed by "total: 1"

Scenario: 792a8a1d Equal counts are ordered by value
  Given two notes with type: fact and two notes with type: feedback
  When the user runs engram count --group-by type
  Then fact is printed before feedback

Scenario: 792a8a1e Only notes at the vault root are counted
  Given a vault with one note at its root and another note in a subdirectory, both with type: fact
  When the user runs engram count --group-by type
  Then fact is printed with the count 1
```

## 792a8b Absent attributes and the total

Requirement: [792a8b Absent attributes and the total](#792a8b-absent-attributes-and-the-total) `engram count --group-by <attr>` MUST print, after the value lines and before
the total, the line `(<attr> absent): <N>` when `<N>` counted notes have no `<attr>` key. It MUST
end with the line `total: <T>`, where `<T>` is the number of notes counted, not the number of
values. A note whose `<attr>` is present but empty MUST count toward the total and appear in no
value line and not in the absent line. When no note is counted it MUST print nothing, not even a
total.

### 792a8b1 UAT Scenarios

```gherkin
Scenario: 792a8b1a Notes without the attribute are reported
  Given one note with vocab: [x] and one note with no vocab key
  When the user runs engram count --group-by vocab
  Then stdout is "x", a tab and 1, then "(vocab absent): 1", then "total: 2"

Scenario: 792a8b1b No note has the attribute
  Given 557 notes and none has the attribute attr
  When the user runs engram count --group-by attr
  Then stdout is "(attr absent): 557" then "total: 557", with no value line

Scenario: 792a8b1c An empty vault prints nothing
  Given a vault with no notes
  When the user runs engram count --group-by type
  Then nothing is printed to stdout or stderr

Scenario: 792a8b1d An empty attribute counts only toward the total
  Given a note whose frontmatter has the key foo with no value
  When the user runs engram count --group-by foo
  Then stdout is "total: 1"
```

## 792a8c Filter the counted notes

Requirement: [792a8c Filter the counted notes](#792a8c-filter-the-counted-notes) `engram count --group-by <attr>` MUST count only the notes that satisfy every
`--filter <attr>=<value>` it is given; the flag MAY be repeated. A note satisfies a filter when its
attribute equals the value (a scalar) or lists the value among its elements (a list). The filter
text is split at its first `=`. The command MUST refuse a filter that has no `=`, failing with
`count: --filter must be attr=value: "<filter>"` and a non-zero exit.

### 792a8c1 UAT Scenarios

```gherkin
Scenario: 792a8c1a A scalar filter matches by equality
  Given notes of type feedback and notes of type fact
  When the user runs engram count --group-by type --filter type=feedback
  Then only the feedback notes are counted, and no other type is printed

Scenario: 792a8c1b A list filter matches by membership
  Given a note with tags: [tier/haiku, vocab/x]
  When the user runs engram count --group-by tags --filter tags=tier/haiku
  Then the note is counted

Scenario: 792a8c1c Every filter must match
  Given a fact note with outcome: pass, a fact note with outcome: fail, and a feedback note with outcome: pass
  When the user runs engram count --group-by outcome --filter type=fact --filter outcome=pass
  Then only the first note is counted, and the output is "pass", a tab and 1, then "total: 1"

Scenario: 792a8c1d A note without the filter's attribute is excluded
  Given a note that has no outcome key
  When the user runs engram count --group-by type --filter outcome=pass
  Then the note is not counted

Scenario: 792a8c1e No note matches
  Given a vault with no feedback note
  When the user runs engram count --group-by type --filter type=feedback
  Then nothing is printed

Scenario: 792a8c1f A filter without = is refused
  Given a vault
  When the user runs engram count --group-by type --filter bad
  Then it fails with an error saying --filter must be attr=value and quoting "bad", and prints nothing to stdout
```

## 792a8d Notes that cannot be counted

Requirement: [792a8d Notes that cannot be counted](#792a8d-notes-that-cannot-be-counted) `engram count --group-by` MUST skip a note that it cannot read, a note with no
frontmatter, and a note whose frontmatter is not valid YAML, counting it neither as a value, nor as
absent, nor in the total. It MUST count a note whose lines end in `\r\n` exactly as it counts the
same note with LF line endings.

Rationale: a note that cannot be parsed has no attribute to count, and one stray file must not
stop a count of the rest. Notes written on a CRLF platform are valid notes.

### 792a8d1 UAT Scenarios

```gherkin
Scenario: 792a8d1a A note without usable frontmatter is skipped
  Given one well-formed note with type: feedback, one note with no frontmatter, and one note whose frontmatter is malformed YAML
  When the user runs engram count --group-by type
  Then stdout is "feedback", a tab and 1, then "total: 1"

Scenario: 792a8d1b An unreadable note is skipped
  Given a vault with a note with type: fact that the user cannot read, and a readable note with type: fact
  When the user runs engram count --group-by type
  Then fact is printed with the count 1, followed by "total: 1"

Scenario: 792a8d1c A CRLF note is counted
  Given a note with type: feedback whose lines end in \r\n
  When the user runs engram count --group-by type
  Then feedback is printed with the count 1, followed by "total: 1"
```

## 792a8e Count the notes that link to a note

Requirement: [792a8e Count the notes that link to a note](#792a8e-count-the-notes-that-link-to-a-note) `engram count --backlinks-of <basename>` MUST print `in-degree: <N>`, where
`<N>` is the number of other notes at the vault root whose text links to the note named
`<basename>`, followed by those notes' basenames one per line in ascending order. A note that links
more than once MUST count once. A note's link to itself, a link inside a fenced code block, a
link written in a note below the vault root, and a link to a note that is not at the vault root
MUST NOT count. A link written with a trailing `.md`
MUST count as the link without it. `<basename>` is matched exactly as given. The command MUST fail
with a non-zero exit, naming the file, when any note at the vault root cannot be read.

Rationale: the wikilinks in a note's text are the vault's links, and the count agrees with the
note's backlinks panel in Obsidian. A link inside a fenced code block is not a link there, and a
link to a name with no note is not a backlink of any note.

### 792a8e1 UAT Scenarios

```gherkin
Scenario: 792a8e1a A note with incoming links
  Given three notes that contain [[foo.alpha]]
  When the user runs engram count --backlinks-of foo.alpha
  Then stdout starts with "in-degree: 3", followed by the three linkers' basenames, one per line, in ascending order

Scenario: 792a8e1b A note with no incoming links
  Given no note links to unknown.note
  When the user runs engram count --backlinks-of unknown.note
  Then stdout is "in-degree: 0" and no linker lines

Scenario: 792a8e1c A basename that is not a note
  Given a vault in which the basename nosuchfile never appears
  When the user runs engram count --backlinks-of nosuchfile
  Then stdout is "in-degree: 0" and no linker lines

Scenario: 792a8e1d A repeated link counts once
  Given a note x that contains [[y]] twice
  When the user runs engram count --backlinks-of y
  Then stdout is "in-degree: 1" then "x"

Scenario: 792a8e1e A note's link to itself does not count
  Given a note x whose only link is [[x]]
  When the user runs engram count --backlinks-of x
  Then stdout is "in-degree: 0"

Scenario: 792a8e1f A link inside a fenced code block does not count
  Given a note x that contains [[z]] only inside a fenced code block
  When the user runs engram count --backlinks-of z
  Then stdout is "in-degree: 0"

Scenario: 792a8e1g A link ending in .md counts as the bare link
  Given a note x that contains [[y.md]]
  When the user runs engram count --backlinks-of y
  Then stdout is "in-degree: 1" then "x"

Scenario: 792a8e1h A link to a missing note counts for no note
  Given a note x that contains [[nowhere]], and no note named nowhere
  When the user runs engram count --backlinks-of nowhere
  Then stdout is "in-degree: 0"

Scenario: 792a8e1i Only notes at the vault root link
  Given a note in a subdirectory that contains [[x]], and a note x at the vault root
  When the user runs engram count --backlinks-of x
  Then stdout is "in-degree: 0"

Scenario: 792a8e1j An unreadable note fails the report
  Given a vault with a note the user cannot read
  When the user runs engram count --backlinks-of x
  Then it fails with an error naming the unreadable file, and prints nothing to stdout
```

## 792a8f One mode per invocation

Requirement: [792a8f One mode per invocation](#792a8f-one-mode-per-invocation) `engram count` MUST fail with a non-zero exit, printing nothing to stdout, when
it is given both `--group-by` and `--backlinks-of` (`count: --group-by and --backlinks-of are
mutually exclusive`), and when it is given neither (`count: specify --group-by <attr> or
--backlinks-of <basename>`).

Rationale: the two modes measure different things ([792a8g The two modes measure different things](#792a8g-the-two-modes-measure-different-things)), so one invocation does not mix them.

### 792a8f1 UAT Scenarios

```gherkin
Scenario: 792a8f1a Both flags are refused
  Given a vault
  When the user runs engram count --group-by type --backlinks-of note.name
  Then it fails with "count: --group-by and --backlinks-of are mutually exclusive"

Scenario: 792a8f1b Neither flag is refused
  Given a vault
  When the user runs engram count with no --group-by and no --backlinks-of
  Then it fails with "count: specify --group-by <attr> or --backlinks-of <basename>"

Scenario: 792a8f1c A filter alone is not a mode
  Given a vault
  When the user runs engram count --filter type=fact
  Then it fails with "count: specify --group-by <attr> or --backlinks-of <basename>"
```

## 792a8g The two modes measure different things

Requirement: [792a8g The two modes measure different things](#792a8g-the-two-modes-measure-different-things) `engram count --backlinks-of` MUST count every note that links to the named
note, whether or not that note carries a given attribute value, and `engram count --group-by` MUST
count only the notes that carry the value, whether or not they link. For a note named for an
attribute value, the in-degree therefore equals the `--group-by` count of that value plus the
number of linking notes that do not carry the value, provided every note that carries the value
links to the note. A note that carries the value without linking to it counts toward `--group-by`
and not toward `--backlinks-of`.

Rationale: each mode can be checked by hand against its own Obsidian view (`--group-by` against a
property or tag filter, `--backlinks-of` against the backlinks panel), and the two are not equal to
each other. An index or map-of-content page links every note on a topic without carrying the topic
in its own frontmatter, so it counts toward `--backlinks-of` and not toward `--group-by`. Reporting
the two as "the same count" would make the modes look redundant when they are complementary. The relationship was checked on the live vault in July 2026: 33
members of `retrieval-design` and an in-degree of 34, the extra linker being a generated index page
that was retired later.

### 792a8g1 UAT Scenarios

```gherkin
Scenario: 792a8g1a The modes agree when every linker is a member
  Given notes n1 and n2 with foo: [alpha] that each contain [[foo.alpha]], and the note foo.alpha
  When the user runs engram count --group-by foo and engram count --backlinks-of foo.alpha
  Then --group-by prints alpha with the count 2 and --backlinks-of prints "in-degree: 2"

Scenario: 792a8g1b The in-degree exceeds the count by the non-member linkers
  Given n1 with foo: [alpha] that contains [[foo.alpha]], idx with no foo attribute that contains [[foo.alpha]], and the note foo.alpha
  When the user runs engram count --group-by foo and engram count --backlinks-of foo.alpha
  Then --group-by prints alpha with the count 1 and --backlinks-of prints "in-degree: 2" then n1 and idx
```

## 792a8h A count changes nothing and is exact

Requirement: [792a8h A count changes nothing and is exact](#792a8h-a-count-changes-nothing-and-is-exact) `engram count` MUST NOT create, modify or delete any note or sidecar in the
vault, and MUST NOT need a note's sidecar or an embedding to count it. It MUST count every note at
the vault root, not a ranked or truncated subset.

Rationale: counting is a read-only aggregation, deliberately off the recall and similarity path. It
never embeds, scores cosine similarity or clusters.

### 792a8h1 UAT Scenarios

```gherkin
Scenario: 792a8h1a A count leaves the vault as it was
  Given an existing vault with notes
  When the user runs engram count --group-by type and then engram count --backlinks-of x
  Then every file in the vault is byte-identical afterwards, and no file was created

Scenario: 792a8h1b Every note is counted without sidecars
  Given a vault of 600 notes with type: fact and no sidecars
  When the user runs engram count --group-by type
  Then stdout is "fact", a tab and 600, then "total: 600"
```
