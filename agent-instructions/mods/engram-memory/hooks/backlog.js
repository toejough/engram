// The pure parts of the backlog capture (793e8g, 793b278, SP15): which transcript sources still hold content nobody has
// captured. It reads what `engram ingest --auto` leaves in the chunks directory (the manifest) and a mark file of its
// own; the conversation itself is read from the source file, by its rows' roles (the index text cannot tell a speaker
// from a line that imitates one). Every call on `$` is in register.js.

// A source is left alone while it may still be running (quiet), and history older than the age limit is not mined:
// only the leftovers of the last days are the backlog.
const QUIET_MS = 15 * 60 * 1000
const MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000
// The rows a harness writes just after the last turn the System captured (a closing row, a flush) are not a turn: a
// source changed by no more than this after that turn is covered by it. A turn that ends later is captured (793b317).
const TURN_SLACK_MS = 10 * 1000
// What one turn's backlog work does: the windows judged, and the sources looked at (a source with nothing to capture is
// marked, but each looked at costs a read).
export const WINDOWS_PER_TURN = 6
export const SOURCES_PER_TURN = 20
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
// finished; `turns` maps a session's file name to the time its turns were last captured by the System itself;
// `partial` maps a source path to the hash it had and the number of its windows (counted from the end) judged so far.
export function parseMarks(text) {
  if (text.trim() === '') return { sources: {}, turns: {}, partial: {} }
  const marks = JSON.parse(text)
  if (marks === null || typeof marks !== 'object' || typeof marks.sources !== 'object' || typeof marks.turns !== 'object') throw new Error('backlog marks are not {sources, turns}')
  return { sources: marks.sources, turns: marks.turns, partial: marks.partial ?? {} }
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

// withSource marks a source captured at its present hash and drops its progress.
export function withSource(marks, source) {
  const { [source.path]: _progress, ...partial } = marks.partial
  return { ...marks, sources: { ...marks.sources, [source.path]: source.hash }, partial }
}

// withProgress records that the last `done` windows of a source, at its present hash, are judged.
export const withProgress = (marks, source, done) => ({ ...marks, partial: { ...marks.partial, [source.path]: { hash: source.hash, done } } })

// windowsDone: the windows of the source judged already; progress at another hash is progress on another text.
export const windowsDone = (marks, source) => (marks.partial[source.path]?.hash === source.hash ? marks.partial[source.path].done : 0)

// nextWindows are the windows to judge in a turn, the end of the conversation first: from the windows (oldest first),
// leave the last `done` and take up to `budget` before them. `finished` says whether they are all the windows left.
export function nextWindows(windows, done, budget = WINDOWS_PER_TURN) {
  const end = windows.length - done
  const batch = windows.slice(Math.max(end - budget, 0), end).reverse()
  return { batch, finished: end - batch.length <= 0 }
}
export const withTurns = (marks, session, now) => ({ ...marks, turns: { ...marks.turns, [session]: now } })
