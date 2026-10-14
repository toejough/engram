import { tenthsOfSeconds } from '../lib/units.js'

const LONG_SETTLE_MS = 10000
const SHORT_SETTLE_MS = 300
const CLOSE_SETTLE_MS = 9000

export const claim =
  'With streaming input the module keeps running its work after turn.complete until the session closes'

export async function run(harness) {
  const evidence = {}

  const open = harness.scenario()
  harness.recorder(open, { background: { afterMs: 2000, sleepSeconds: 6 } })
  const openSession = open.startSession()
  await openSession.send('Reply with exactly the word: ok')
  const resultAt = Date.now()
  await harness.settle(LONG_SETTLE_MS)
  const events = harness.events(open)
  const after = (kind) => {
    const event = events.find((entry) => entry.kind === kind)
    const complete = events.find((entry) => entry.kind === 'turn.complete')
    return event && complete ? tenthsOfSeconds(event.t - complete.t) : null
  }
  evidence.sessionKeptOpen = {
    timerFiredSecondsAfterTurnComplete: after('background.timer'),
    processDoneSecondsAfterTurnComplete: after('background.process.done'),
    kinds: events.map((entry) => entry.kind),
    waitedMs: Date.now() - resultAt,
  }
  await openSession.close()

  const closing = harness.scenario()
  harness.recorder(closing, { background: { afterMs: 2000, sleepSeconds: 6 } })
  const closingSession = closing.startSession()
  await closingSession.send('Reply with exactly the word: ok')
  await harness.settle(SHORT_SETTLE_MS)
  const closing_started = Date.now()
  await closingSession.close()
  const closeMs = Date.now() - closing_started
  await harness.settle(CLOSE_SETTLE_MS)
  evidence.closedRightAfterTheResult = { closeMs, kinds: harness.events(closing).map((entry) => entry.kind) }
  return evidence
}
