// The pure parts of the memory-activity summary (793a5h, 793b204): the counts the System keeps and the one line the
// user is shown. The line is built from constants and counts only, never from a note, an offer or a lesson (793b297):
// a count is a non-negative whole number the mod itself incremented. Every call on `$` is in register.js.

export const SUMMARY_PREFIX = 'Memory activity: '

// the counts in the order the line says them: the singular noun and the participle of each
const PARTS = [
  ['delivered', 'item', 'delivered'],
  ['added', 'note', 'added'],
  ['amended', 'note', 'amended'],
  ['curated', 'offer', 'curated'],
]

export const emptyCounts = () => ({ delivered: 0, added: 0, amended: 0, curated: 0 })

// summaryLine is "3 items delivered, 1 note added and 2 offers curated": only the counts above zero, null for none.
export function summaryLine(counts) {
  const phrases = PARTS.filter(([name]) => counts[name] > 0).map(([name, noun, verb]) => `${counts[name]} ${noun}${counts[name] === 1 ? '' : 's'} ${verb}`)
  if (phrases.length === 0) return null
  const text = phrases.length === 1 ? phrases[0] : `${phrases.slice(0, -1).join(', ')} and ${phrases.at(-1)}`
  return `${SUMMARY_PREFIX}${text}`
}

// statusLine is the whole of the pinned line after a turn (793b302: it holds one line, so the warnings of the turn
// and the summary share it): each warning once, then the summary. Null when there is nothing to show.
export function statusLine(counts, warnings) {
  const summary = summaryLine(counts)
  const parts = summary === null ? [...warnings] : [...warnings, summary]
  return parts.length === 0 ? null : parts.join('. ')
}
