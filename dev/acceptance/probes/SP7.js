export const claim = '$.session.messages() contains a delivered block, drops it after compaction and shows it again after a resume; /compact in the stream and the SDK resume option drive both'

const TOKEN = 'CTX-MSG-5d1e'

function snapshot(events, nth) {
  const submit = events.filter((event) => event.kind === 'prompt.submit')[nth]
  if (!submit) return null
  const messages = submit.data.messages
  return messages === null ? null : { count: messages.length, withToken: messages.filter((m) => m.hasWatch).length, roles: messages.map((m) => `${m.role}${m.tools ? `+${m.tools}tools` : ''}`) }
}

export async function run(h) {
  const evidence = {}

  const compacting = h.scenario()
  h.recorder(compacting, { context: { promptSubmit: TOKEN }, watch: TOKEN })
  const a = compacting.startSession()
  await a.send('Reply with the word one.')
  await a.send('Reply with the word two.')
  const compactTurn = await a.send('/compact').then((turn) => ({ ok: !turn.isError, text: turn.text.slice(0, 120) }), (error) => ({ error: error.message.slice(0, 160) }))
  await a.send('Reply with the word three.')
  await h.settle(1000)
  const events = h.events(compacting)
  evidence.compaction = {
    compactTurn,
    compactEvents: events.filter((event) => event.kind === 'session.compact').length,
    atTurn1: snapshot(events, 0),
    atTurn2: snapshot(events, 1),
    promptSubmitEventsSeen: events.filter((event) => event.kind === 'prompt.submit').map((event) => event.data.text),
    atTurn3AfterCompact: snapshot(events, events.filter((event) => event.kind === 'prompt.submit').length - 1),
  }

  const resuming = h.scenario()
  h.recorder(resuming, { context: { promptSubmit: TOKEN }, watch: TOKEN })
  const first = resuming.startSession()
  await first.send('Reply with the word one.')
  await h.settle(500)
  const sessionId = first.sessionId
  await first.close()
  const second = resuming.startSession({ resume: sessionId })
  await second.send('Reply with the word two.')
  await h.settle(1000)
  const resumedEvents = h.events(resuming)
  evidence.resume = { sameSessionId: second.sessionId === sessionId, beforeEnd: snapshot(resumedEvents, 0), afterResume: snapshot(resumedEvents, 1), sessionStarts: resumedEvents.filter((event) => event.kind === 'session.start' || event.kind === 'classic.SessionStart').map((event) => event.kind + ':' + (event.data.source ?? '')) }
  return evidence
}
