// What a run reports (793e2h): one entry per scenario run, filled by the hooks and read by the report.
import { denials, readRows, shownTexts, systemLines } from './transcript.js'

const results = []

export function resetResults() {
  results.length = 0
}

export function recordResult(entry) {
  results.push(entry)
}

export function allResults() {
  return [...results]
}

// classify maps a cucumber scenario status and its failure into a result of 793e2h.
// An assertion that did not hold is Red; any other exception is a harness error and is never Red evidence.
export function classify({ status, exceptionType, skipReason }) {
  switch (status) {
    case 'PASSED':
      return 'Green'
    case 'PENDING':
      return 'Pending (not built)'
    case 'SKIPPED':
      return skipReason ? `Not run (${skipReason})` : 'Not run'
    case 'FAILED':
      return exceptionType === 'AssertionError' ? 'Red' : 'Harness error'
    case 'UNDEFINED':
      return 'Harness error (undefined step)'
    case 'AMBIGUOUS':
      return 'Harness error (ambiguous step)'
    default:
      return `Harness error (${status})`
  }
}

// sessionRows are the transcript rows of a session, none when it has no transcript file.
function sessionRows(session) {
  try {
    const file = session.transcriptPath?.()
    return file ? readRows(file) : []
  } catch {
    return []
  }
}

// entryFor builds the report entry of a finished scenario from its World and the cucumber result.
export function entryFor(world, result) {
  const status = result.status
  const sessions = world.scenario?.sessions ?? []
  const rows = sessions.map(sessionRows)
  const denied = rows.flatMap((sessionRowsOf) => denials(sessionRowsOf))
  let verdict = classify({
    status,
    exceptionType: result.exception?.type,
    skipReason: world.skipReason,
  })
  // a refused tool call is a harness condition: the agent could not do what the scenario needs, whatever it then did
  if (verdict === 'Red' && denied.length > 0) verdict = `Harness error (permission denied: ${denied[0].name})`
  return {
    id: world.id,
    title: world.title,
    result: verdict,
    retries: world.retries ?? 0,
    usage: world.scenario?.usage() ?? null,
    message: status === 'PASSED' || status === 'PENDING' ? '' : (result.message ?? '').split('\n')[0],
    // what the System showed the user (its lines): a lookup it cut is a warning here, a judgement that missed is not
    // (stream messages of a mod, and the transcript lines `$.ui.log` writes, 793e7c)
    activity: sessions.flatMap((session, index) => [...shownTexts(session.messages), ...systemLines(rows[index])]),
  }
}
