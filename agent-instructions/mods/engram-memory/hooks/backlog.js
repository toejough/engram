// The pure parts of the backlog capture (793e8g, 793b278, SP15): which transcript sources still hold content nobody has
// captured. It reads what `engram ingest --auto` leaves in the chunks directory (the manifest) and a mark file of its
// own; the conversation itself is read from the source file, by its rows' roles (the index text cannot tell a speaker
// from a line that imitates one). Every call on `$` is in register.js.

// A source is left alone while it may still be running (quiet), and history older than the age limit is not mined:
// only the leftovers of the last days are the backlog.
const QUIET_MS = 15 * 60 * 1000
const MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000
// The transcript is finished a moment after the turn that the System captured ends.
const TURN_SLACK_MS = 2 * 60 * 1000
const NANOS_PER_MS = 1e6
export const MARKS_NAME = 'backlog-marks.json'

// chunksDirectory is where the chunk index lives (792a5a): the same precedence as the engram commands.
export function chunksDirectory({ chunks, xdg, home }) {
  if (chunks) return chunks
  if (xdg) return `${xdg}/engram/chunks`
  return `${home}/.local/share/engram/chunks`
}

export const baseName = (path) => path.split('/').at(-1)

// parseManifest lists the session transcripts the manifest knows: a `.jsonl` source that is not a subagent's (its own
// stop captured it) and not a duplicate of another source (the canonical one is the one indexed). An absent manifest
// is an empty one; one that is not a JSON object is an error.
export function parseManifest(text) {
  if (text.trim() === '') return []
  const manifest = JSON.parse(text)
  if (manifest === null || typeof manifest !== 'object' || Array.isArray(manifest)) throw new Error('ingest manifest is not an object')
  return Object.entries(manifest)
    .filter(([path, entry]) => path.endsWith('.jsonl') && !path.includes('/subagents/') && !entry.duplicate_of)
    .map(([path, entry]) => ({ path, mtimeMs: entry.mtime_unix_nano / NANOS_PER_MS, hash: entry.file_hash }))
}

// parseMarks reads the mark file: `sources` maps a source path to the hash of the file when its backlog capture
// finished; `turns` maps a session's file name to the time its turns were last captured by the System itself.
export function parseMarks(text) {
  if (text.trim() === '') return { sources: {}, turns: {} }
  const marks = JSON.parse(text)
  if (marks === null || typeof marks !== 'object' || typeof marks.sources !== 'object' || typeof marks.turns !== 'object') throw new Error('backlog marks are not {sources, turns}')
  return { sources: marks.sources, turns: marks.turns }
}

// pending are the sources to capture, the most recently changed first: not the live session (by its file name), quiet
// for a while, within the age limit, and neither marked with their present hash nor covered by the System's own turn
// captures.
export function pending(sources, marks, { now, live }) {
  const liveName = live ? baseName(live) : null
  return sources
    .filter((source) => baseName(source.path) !== liveName)
    .filter((source) => now - source.mtimeMs >= QUIET_MS && now - source.mtimeMs <= MAX_AGE_MS)
    .filter((source) => marks.sources[source.path] !== source.hash)
    .filter((source) => !(baseName(source.path) in marks.turns && source.mtimeMs <= marks.turns[baseName(source.path)] + TURN_SLACK_MS))
    .sort((a, b) => b.mtimeMs - a.mtimeMs)
}

export const withSource = (marks, source) => ({ ...marks, sources: { ...marks.sources, [source.path]: source.hash } })
export const withTurns = (marks, session, now) => ({ ...marks, turns: { ...marks.turns, [session]: now } })
