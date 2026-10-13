// What a run reports (793e2h): one entry per scenario run, filled by the hooks and read by the report.
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

// entryFor builds the report entry of a finished scenario from its World and the cucumber result.
export function entryFor(world, result) {
  const status = result.status
  return {
    id: world.id,
    title: world.title,
    result: classify({
      status,
      exceptionType: result.exception?.type,
      skipReason: world.skipReason,
    }),
    retries: world.retries ?? 0,
    usage: world.scenario?.usage() ?? null,
    message: status === 'PASSED' || status === 'PENDING' ? '' : (result.message ?? '').split('\n')[0],
  }
}
