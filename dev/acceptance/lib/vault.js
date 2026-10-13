// The parts of a note file engram writes: the front matter between the two `---` lines (source and situation among
// its fields), then the body. A lesson is stated by the body of a note; how it was written is told by its source (793b316).
export function parseNote(text) {
  const parts = text.split(/^---$/m)
  const front = parts[1] ?? ''
  // `content` is everything a note says but where it came from: a lesson is stated by the note's fields and body, not by its source line
  const content = text.replace(/^source:.*$/m, '')
  return { front, content, body: parts.slice(2).join('---').trim(), source: /^source:[ \t]*(.*)$/m.exec(front)?.[1]?.trim().replace(/^["']|["']$/g, '') ?? '' }
}

// automaticSource: the note says in its source that the System wrote it ("written automatically", mod: capture.js).
export const AUTOMATIC = /written automatically/

// subagentOf is the id of the subagent a note's source names, or null.
export const subagentOf = (source) => /\(session [^,]+, turn \d+, subagent ([A-Za-z0-9_-]+)\)/.exec(source)?.[1] ?? null
