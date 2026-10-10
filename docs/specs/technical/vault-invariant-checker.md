# Vault Invariant Checker - Technical Spec

## 792b26a Orientation

- Ownership: Repository component.
- Responsibility: reads the notes and sidecars at a vault's root and reports which of the vault's
  invariants they violate. It classifies each violation as a correctness failure or a corpus-health
  warning, lists the offending items, and says whether any failure was found. It writes nothing.
- Current realization (informational): `internal/cli/check.go` (`RunCheck`, `CheckArgs`, `CheckDeps`,
  `checkGraphResolution`, `checkSituationPresence`, `checkSidecars`, `printLinkExamples`,
  `printNoteExamples`).
- Qualification: responsibility, the invariants and their severities are stated without naming any
  function; ownership, the [command-line-surface](./command-line-surface.md) decides how the failure reaches stderr and the exit
  status, while this component decides what is judged and how; independent verification, it is
  tested directly over a fake scan, note reader and sidecar reader and over a real directory;
  stability, a vault with invariants that something checks is a lasting need; necessity, `engram
  check` is this contract.
- Supported capabilities: [vault-health](../product/vault-health.md).

Rationale: the memory system once shipped structural breaks that went uncaught for a long time, and
its testing validated downstream results, never its own internals. A link written in one form and
resolved in another dropped most of the vault's edges, and nothing reported it. The cause was
trusting `grep` and a report of success over reading the frontmatter and running the binary, with no
invariant defined to trip over. So the invariants are executable: a check over the real vault that
parses it, rather than searching its text, and reports a violation where an agent or a person will
see it. The original plan wired the check as a `targ` gate so that a violation fails CI; no such gate
exists today. Severity separates a break in correctness from corpus health: only a `FAIL`
breaks correctness and fails the run, while a `WARN` reports without blocking, because a
legitimately standalone note or a deleted note's name is not an error.

## 792b26c Behavior Requirements

### 792b26c1 Running the checks

Requirement: [792b26c1 Running the checks](#792b26c1-running-the-checks) The vault-invariant-checker MUST scan the vault ([792b13c1 Vault scan](./wikilink-graph.md#792b13c1-vault-scan)), run the link check
([792b26c2 Link resolution](#792b26c2-link-resolution)), then the situation check ([792b26c3 Situation presence](#792b26c3-situation-presence)), then the sidecar check ([792b26c4 Sidecar schema](#792b26c4-sidecar-schema)), and print each
check's lines in that order. It MUST return the error `check: FAIL-class invariant violations found`
when any check reported a failure, and MUST return no error when its worst line is a `WARN`. It MUST
return a scan failure as `check: scan: <error>` before printing any line. It MUST judge a note by
parsing its bytes, so that a note whose text is not valid UTF-8 is checked like any other.

Supports: [792a9a1a A clean vault passes](../product/vault-health.md#792a9a1-uat-scenarios), [792a9a1b A FAIL line fails the command](../product/vault-health.md#792a9a1-uat-scenarios), [792a9a1c A WARN line does not fail the command](../product/vault-health.md#792a9a1-uat-scenarios), [792a9a1e An unreadable note stops the check](../product/vault-health.md#792a9a1-uat-scenarios), [792a9a1f An empty vault passes](../product/vault-health.md#792a9a1-uat-scenarios), [792a9a1g A note that is not valid UTF-8 is still checked](../product/vault-health.md#792a9a1-uat-scenarios).

Rationale: a text search that stops at an invalid multibyte sequence silently skipped notes in an early
census and gave a false count of untagged notes, so a checker must not be built on one.

#### 792b26c1a Scenarios

```gherkin
Scenario: 792b26c1a1 A clean vault returns no error
  Given notes whose links resolve, whose fact notes all have a situation, and whose sidecars are current
  When the vault is checked
  Then no error is returned and the S1 line begins PASS

Scenario: 792b26c1a2 A failure is returned as an error
  Given a note whose sidecar is on the old schema
  When the vault is checked
  Then the error "check: FAIL-class invariant violations found" is returned and an S1 FAIL line is printed

Scenario: 792b26c1a3 A warning is not an error
  Given a note whose only link targets no note
  When the vault is checked
  Then a WARN line is printed, the G0 line begins PASS, and no error is returned

Scenario: 792b26c1a4 A scan failure is returned before any line
  Given a vault whose scan fails
  When the vault is checked
  Then it fails with an error beginning "check: scan:" and nothing is printed

Scenario: 792b26c1a5 A note that is not valid UTF-8 is checked
  Given a note with invalid UTF-8 bytes that links a missing note
  When the vault is checked
  Then the dangling link is reported
```

### 792b26c2 Link resolution

Requirement: [792b26c2 Link resolution](#792b26c2-link-resolution) The vault-invariant-checker MUST take the links that resolve to no note
([792b13c6 Unresolved links](./wikilink-graph.md#792b13c6-unresolved-links)) and treat a link as a G0 violation when its target is a bare Luhmann ID, one or more digits
followed by letters or digits, equal to the part of a scanned note's basename before its first `.`; every
other unresolved link MUST be a G3 finding. It MUST print `FAIL  G0 graph-resolution: <N> bare-id links
that should resolve` and list the G0 links, or `PASS  G0 graph-resolution: every authored link resolves
by form` when there are none, and MUST report a failure for G0. It MUST print `WARN  G3 dangling: <N>
authored links target no note` and list the G3 links when there are any, and MUST NOT report a failure
for them.

Supports: [792a9a1a A clean vault passes](../product/vault-health.md#792a9a1-uat-scenarios), [792a9a1b A FAIL line fails the command](../product/vault-health.md#792a9a1-uat-scenarios), [792a9a1c A WARN line does not fail the command](../product/vault-health.md#792a9a1-uat-scenarios), [792a9b1a A bare-ID link fails](../product/vault-health.md#792a9b1-uat-scenarios), [792a9b1b A basename link passes](../product/vault-health.md#792a9b1-uat-scenarios), [792a9b1c A legacy suffixed link resolves](../product/vault-health.md#792a9b1-uat-scenarios), [792a9c1a A link to a missing note warns](../product/vault-health.md#792a9c1-uat-scenarios), [792a9c1b A missing note is reported once however it is spelled](../product/vault-health.md#792a9c1-uat-scenarios), [792a9c1c A note's link to itself is not reported](../product/vault-health.md#792a9c1-uat-scenarios),
[792a9c1d A link inside a fenced code block is not reported](../product/vault-health.md#792a9c1-uat-scenarios), [792a9c1e A vault with no dangling link prints no G3 line](../product/vault-health.md#792a9c1-uat-scenarios).

Rationale: a link `[[105]]` for a note `105.2026-05-30.foo` could resolve if written in the form the
graph keys on, so it is a defect to fix and fails the run. A link to a name that is no note at all
cannot be fixed by rewriting it, so it is only reported.

#### 792b26c2a Scenarios

```gherkin
Scenario: 792b26c2a1 A bare-ID link fails
  Given a note 105.2026-05-30.foo that links 105
  When the vault is checked
  Then a FAIL line names G0, the link 105.2026-05-30.foo → [[105]] is listed, and a failure is reported

Scenario: 792b26c2a2 Resolving links pass
  Given a note A that links B, and a note B
  When the vault is checked
  Then a PASS line names G0 and no failure is reported

Scenario: 792b26c2a3 A link to a missing note warns
  Given a note whose link [[999]] matches no note
  When the vault is checked
  Then a WARN line names G3, the G0 line begins PASS, and no failure is reported

Scenario: 792b26c2a4 A bare ID that matches no note is dangling
  Given a note that links 999, and no note whose basename begins "999."
  When the vault is checked
  Then the link is counted under G3 and not under G0

Scenario: 792b26c2a5 No G3 line without a dangling link
  Given notes whose links all resolve
  When the vault is checked
  Then no G3 line is printed
```

### 792b26c3 Situation presence

Requirement: [792b26c3 Situation presence](#792b26c3-situation-presence) The vault-invariant-checker MUST read each scanned note, as its LF form, and
split its frontmatter ([792b6c6 Frontmatter splitting](./note-document-format.md#792b6c6-frontmatter-splitting)), and MUST count a note of type `fact` or `feedback` whose `situation`
is absent, empty or only whitespace. It MUST print `FAIL  M5 situation-presence: <N> note(s) missing
a situation` and list those notes, or `PASS  M5 situation-presence: every fact/feedback names a
situation` when there are none, and MUST report a failure for M5. It MUST skip a note it cannot read,
a note with no frontmatter, a note whose frontmatter does not parse, and a note of any other type.

Supports: [792a9a1a A clean vault passes](../product/vault-health.md#792a9a1-uat-scenarios), [792a9d1a A fact without a situation fails](../product/vault-health.md#792a9d1-uat-scenarios), [792a9d1b A blank situation fails](../product/vault-health.md#792a9d1-uat-scenarios), [792a9d1c Every situation present passes](../product/vault-health.md#792a9d1-uat-scenarios), [792a9d1d Other note types are not judged](../product/vault-health.md#792a9d1-uat-scenarios), [792a9d1e A note without usable frontmatter is not judged](../product/vault-health.md#792a9d1-uat-scenarios), [792a9d1f A CRLF note is judged](../product/vault-health.md#792a9d1-uat-scenarios).

Rationale: recall matches a query against each note's situation ([792a7d1h A note is found by its own situation](../product/memory-query.md#792a7d1-uat-scenarios)), so a fact or feedback note
without one cannot be found by the situation it was written for. The write paths refuse a blank one
([792a3f Required content](../product/note-writing.md#792a3f-required-content)), so this finds a note that reached the vault another way. A note that cannot be read or
parsed is left for the checks that report it.

#### 792b26c3a Scenarios

```gherkin
Scenario: 792b26c3a1 A fact without a situation fails
  Given a fact note with no situation
  When the vault is checked
  Then a FAIL line names M5 and the note, and a failure is reported

Scenario: 792b26c3a2 A blank situation fails
  Given a fact note whose situation is only whitespace
  When the vault is checked
  Then M5 fails naming the note

Scenario: 792b26c3a3 Every situation present passes
  Given a fact note with a situation
  When the vault is checked
  Then a PASS line names M5

Scenario: 792b26c3a4 Other types are skipped
  Given a note of type moc with no situation
  When the vault is checked
  Then M5 passes

Scenario: 792b26c3a5 Unparseable notes are skipped
  Given a note with malformed YAML frontmatter and a note with no frontmatter
  When the vault is checked
  Then M5 passes

Scenario: 792b26c3a6 A CRLF note is judged
  Given a fact note whose lines end in \r\n and whose situation is empty
  When the vault is checked
  Then M5 fails naming the note

Scenario: 792b26c3a7 An unreadable note is skipped
  Given a scanned note whose file cannot be read
  When the vault is checked
  Then it is not counted under M5
```

### 792b26c4 Sidecar schema

Requirement: [792b26c4 Sidecar schema](#792b26c4-sidecar-schema) The vault-invariant-checker MUST read the sidecar of each scanned note and
decode it under the current schema ([792b10c1 Sidecar format](./embedding-sidecar.md#792b10c1-sidecar-format)). It MUST count a note with no readable sidecar file as
missing, and a sidecar that does not decode as invalid. It MUST print `WARN  S1 sidecar-schema: <N>
note(s) missing a sidecar (run `engram embed apply`)` when notes are missing a sidecar and MUST NOT
report a failure for them. It MUST print `FAIL  S1 sidecar-schema: <N> sidecar(s) on an old/invalid
schema (run `engram embed apply --force`)` and list the notes when a sidecar is invalid, and MUST
report a failure. It MUST print `PASS  S1 sidecar-schema: every sidecar parses under the current
schema` when no sidecar is invalid, after any missing-sidecar line.

Supports: [792a9a1a A clean vault passes](../product/vault-health.md#792a9a1-uat-scenarios), [792a9e1a An old-schema sidecar fails](../product/vault-health.md#792a9e1-uat-scenarios), [792a9e1b A malformed sidecar fails](../product/vault-health.md#792a9e1-uat-scenarios), [792a9e1c A current sidecar passes](../product/vault-health.md#792a9e1-uat-scenarios), [792a9e1d A missing sidecar warns](../product/vault-health.md#792a9e1-uat-scenarios).

#### 792b26c4a Scenarios

```gherkin
Scenario: 792b26c4a1 An old-schema sidecar fails
  Given a sidecar with the old single-vector schema
  When the vault is checked
  Then a FAIL line names S1 and the note, and a failure is reported

Scenario: 792b26c4a2 A current sidecar passes
  Given a current sidecar
  When the vault is checked
  Then a PASS line names S1

Scenario: 792b26c4a3 A missing sidecar warns
  Given a note whose sidecar cannot be read
  When the vault is checked
  Then a WARN line counts it, the PASS line follows, and no failure is reported

Scenario: 792b26c4a4 A malformed sidecar fails
  Given a sidecar that is not valid JSON
  When the vault is checked
  Then a FAIL line names S1 and the note
```

### 792b26c5 Example lists

Requirement: [792b26c5 Example lists](#792b26c5-example-lists) The vault-invariant-checker MUST list at most ten offending items under a
violated invariant, one per line, a link as `<source> → [[<target>]]` and a note by its basename, and
MUST follow ten with the line `… and <N> more` when more were found.

Supports: [792a9a1d Offenders are capped at ten](../product/vault-health.md#792a9a1-uat-scenarios).

#### 792b26c5a Scenarios

```gherkin
Scenario: 792b26c5a1 Links are capped
  Given twelve offending links
  When they are listed
  Then ten are listed and the next line says "and 2 more"

Scenario: 792b26c5a2 Notes are capped
  Given twelve offending notes
  When they are listed
  Then ten are listed and the next line says "and 2 more"
```

### 792b26c6 No writes

Requirement: [792b26c6 No writes](#792b26c6-no-writes) The vault-invariant-checker MUST NOT create, modify or delete any file.

Supports: [792a9f1a A check leaves the vault as it was](../product/vault-health.md#792a9f1-uat-scenarios).

Rationale: the checker is given listing and reading capabilities only, so it cannot write.

#### 792b26c6a Scenarios

```gherkin
Scenario: 792b26c6a1 A check writes nothing
  Given a vault that violates every invariant
  When the vault is checked
  Then no file is created, changed or deleted
```
