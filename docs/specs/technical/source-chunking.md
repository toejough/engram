# Source Chunking - Technical Spec

## 792b16a Orientation

- Ownership: Repository component.
- Responsibility: cuts one source's text into embedding-sized chunks, each with an anchor that
  locates it in the source. A Markdown document is cut by heading section, and a transcript's
  conversation text by user turn; oversized pieces are split on line boundaries and pieces too short
  to carry meaning are dropped. It does no input or output.
- Current realization (informational): `Markdown` and `Transcript` in `internal/chunk/chunk.go`,
  called by `chunkSource` in `internal/cli/ingest.go` with a 500-character transcript target and a
  1500-character maximum.
- Qualification: responsibility, cutting text into anchored chunks is stated without naming any
  function; ownership, the caller decides which text to cut and what to do with the chunks, while
  this component owns the boundaries and anchors; independent verification, it is tested directly on
  strings; stability, the contract survives any change to the index format; necessity, every chunk
  in the index is one of its pieces.
- Supported capabilities: [chunk-indexing](../product/chunk-indexing.md).

Rationale: a chunk is the unit that is embedded and retrieved, so it must fit the embedder's
window ([792b9c2 Input window](./text-embedder.md#792b9c2-input-window)) and still read as a coherent piece of evidence. Heading sections are the natural
units of a document. Single transcript turns are often a few words, so consecutive turns are merged
until the chunk is worth embedding on its own.

## 792b16c Behavior Requirements

### 792b16c1 Markdown sections

Requirement: [792b16c1 Markdown sections](#792b16c1-markdown-sections) The source-chunking MUST cut Markdown into one piece per section, where a
section starts at each ATX heading line (one to six `#` followed by a space) outside fenced code and
includes that heading line, a line whose trimmed text starts with three backticks opens or closes a
fence, the anchor is the heading's text without its `#` marks and surrounding spaces, and text
before the first heading is anchored `preamble`.

Supports: [792a5h1a Markdown is chunked by heading](../product/chunk-indexing.md#792a5h1-uat-scenarios), [792a5h1d A long section is split under one anchor](../product/chunk-indexing.md#792a5h1-uat-scenarios).

#### 792b16c1a Scenarios

```gherkin
Scenario: 792b16c1a1 Each heading starts a section
  Given "# Title", "## Section A" and "## Section B", each followed by a sentence
  When the Markdown is chunked
  Then there are three chunks, anchored Title, Section A and Section B

Scenario: 792b16c1a2 A heading inside a code fence does not split
  Given a "## Real section" holding a fenced block whose line is "# not a heading"
  When the Markdown is chunked
  Then there is one chunk, and it holds the fenced line

Scenario: 792b16c1a3 Text before the first heading is the preamble
  Given a sentence followed by "## Setup" and another sentence
  When the Markdown is chunked
  Then the first chunk is anchored preamble and the second Setup
```

### 792b16c2 Transcript turns

Requirement: [792b16c2 Transcript turns](#792b16c2-transcript-turns) The source-chunking MUST group a transcript's lines into turns, each starting
at a line that begins `USER: ` and running to the next, with any lines before the first such line
as one leading unit, MUST append whole turns to a chunk until its text reaches the 500-character
target and then start the next chunk, and MUST anchor each chunk `turn-<N>`, where `<N>` counts the
first turn it holds among the user turns from 1, or `preamble` when it starts with the leading unit.

Supports: [792a5h1b A transcript is chunked by turns](../product/chunk-indexing.md#792a5h1-uat-scenarios).

#### 792b16c2a Scenarios

```gherkin
Scenario: 792b16c2a1 Short turns merge into one chunk
  Given three short user turns with their replies, together under 500 characters
  When the transcript is chunked
  Then there is one chunk holding all three, anchored turn-1

Scenario: 792b16c2a2 A chunk past the target closes
  Given a first user turn of about 600 characters, then a second turn
  When the transcript is chunked
  Then there are two chunks, the second anchored turn-2

Scenario: 792b16c2a3 Text before the first user turn is the preamble
  Given an ASSISTANT: line followed by the first USER: line
  When the transcript is chunked
  Then the first chunk is anchored preamble
```

### 792b16c3 Oversized pieces

Requirement: [792b16c3 Oversized pieces](#792b16c3-oversized-pieces) The source-chunking MUST split a section or chunk whose trimmed text is longer
than 1500 characters by packing its non-blank lines, in order, into pieces, starting a new piece
whenever adding the next line would take the piece past 1500 characters, and MUST give every piece
the anchor of the text it came from.

Supports: [792a5h1d A long section is split under one anchor](../product/chunk-indexing.md#792a5h1-uat-scenarios).

Rationale: 1500 characters fits the embedder's window. A single line longer than that is not cut
here; it stays one piece, and the embedder embeds its first 1500 characters ([792b9c2 Input window](./text-embedder.md#792b9c2-input-window)).

#### 792b16c3a Scenarios

```gherkin
Scenario: 792b16c3a1 An oversized section is split under its anchor
  Given a "## Big" section of three paragraphs of about 940 characters each
  When the Markdown is chunked
  Then there is more than one chunk, each at most 1500 characters and anchored Big

Scenario: 792b16c3a2 An oversized turn is split
  Given one user turn followed by 80 assistant lines of 50 characters each
  When the transcript is chunked
  Then there is more than one chunk, each at most 1500 characters
```

### 792b16c4 Noise floor

Requirement: [792b16c4 Noise floor](#792b16c4-noise-floor) The source-chunking MUST drop every section, chunk and split piece whose
trimmed text is shorter than 40 characters.

Supports: [792a5h1c Short sections are not indexed](../product/chunk-indexing.md#792a5h1-uat-scenarios).

#### 792b16c4a Scenarios

```gherkin
Scenario: 792b16c4a1 A tiny transcript yields nothing
  Given the transcript text "USER: ok"
  When the transcript is chunked
  Then there are no chunks

Scenario: 792b16c4a2 A tiny section yields nothing
  Given the Markdown "## A" followed by "tiny"
  When the Markdown is chunked
  Then there are no chunks
```
