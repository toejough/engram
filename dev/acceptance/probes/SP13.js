export const claim = 'With streaming input the module keeps running its work after turn.complete until the session closes'

export async function run(h) {
  const evidence = {}

  const open = h.scenario()
  h.recorder(open, { background: { afterMs: 2000, sleepSeconds: 6 } })
  const a = open.startSession()
  await a.send('Reply with exactly the word: ok')
  const resultAt = Date.now()
  await h.settle(10000)
  const events = h.events(open)
  const after = (kind) => {
    const event = events.find((e) => e.kind === kind)
    const complete = events.find((e) => e.kind === 'turn.complete')
    return event && complete ? Math.round((event.t - complete.t) / 100) / 10 : null
  }
  evidence.sessionKeptOpen = { timerFiredSecondsAfterTurnComplete: after('background.timer'), processDoneSecondsAfterTurnComplete: after('background.process.done'), kinds: events.map((e) => e.kind), waitedMs: Date.now() - resultAt }
  await a.close()

  const closing = h.scenario()
  h.recorder(closing, { background: { afterMs: 2000, sleepSeconds: 6 } })
  const b = closing.startSession()
  await b.send('Reply with exactly the word: ok')
  await h.settle(300)
  const closing_started = Date.now()
  await b.close()
  const closeMs = Date.now() - closing_started
  await h.settle(9000)
  evidence.closedRightAfterTheResult = { closeMs, kinds: h.events(closing).map((e) => e.kind) }
  return evidence
}
