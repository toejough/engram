import { MS_PER_SECOND, excerpt } from '../lib/units.js'

const SHORT_SETTLE_MS = 500
const REPLY_CHARS = 120
const ERROR_CHARS = 160

export const claim =
  '$.session.messages() contains a delivered block, drops it after compaction and shows ' +
  'it again after a resume; /compact in the stream and the SDK resume option drive both'

const TOKEN = 'CTX-MSG-5d1e'

function snapshot(events, nth) {
  const submit = events.filter((event) => event.kind === 'prompt.submit')[nth]
  if (!submit) return null
  const messages = submit.data.messages
  return messages === null
    ? null
    : {
        count: messages.length,
        withToken: messages.filter((message) => message.hasWatch).length,
        roles: messages.map((message) => `${message.role}${message.tools ? `+${message.tools}tools` : ''}`),
      }
}

export async function run(harness) {
  const evidence = {}

  const compacting = harness.scenario()
  harness.recorder(compacting, { context: { promptSubmit: TOKEN }, watch: TOKEN })
  const compactingSession = compacting.startSession()
  await compactingSession.send('Reply with the word one.')
  await compactingSession.send('Reply with the word two.')
  const compactTurn = await compactingSession.send('/compact').then(
    (turn) => ({ ok: !turn.isError, text: excerpt(turn.text, REPLY_CHARS) }),
    (error) => ({ error: excerpt(error.message, ERROR_CHARS) }),
  )
  await compactingSession.send('Reply with the word three.')
  await harness.settle(MS_PER_SECOND)
  const events = harness.events(compacting)
  evidence.compaction = {
    compactTurn,
    compactEvents: events.filter((event) => event.kind === 'session.compact').length,
    atTurn1: snapshot(events, 0),
    atTurn2: snapshot(events, 1),
    promptSubmitEventsSeen: events.filter((event) => event.kind === 'prompt.submit').map((event) => event.data.text),
    atTurn3AfterCompact: snapshot(events, events.filter((event) => event.kind === 'prompt.submit').length - 1),
  }

  const resuming = harness.scenario()
  harness.recorder(resuming, { context: { promptSubmit: TOKEN }, watch: TOKEN })
  const first = resuming.startSession()
  await first.send('Reply with the word one.')
  await harness.settle(SHORT_SETTLE_MS)
  const sessionId = first.sessionId
  await first.close()
  const second = resuming.startSession({ resume: sessionId })
  await second.send('Reply with the word two.')
  await harness.settle(MS_PER_SECOND)
  const resumedEvents = harness.events(resuming)
  evidence.resume = {
    sameSessionId: second.sessionId === sessionId,
    beforeEnd: snapshot(resumedEvents, 0),
    afterResume: snapshot(resumedEvents, 1),
    sessionStarts: resumedEvents
      .filter((event) => event.kind === 'session.start' || event.kind === 'classic.SessionStart')
      .map((event) => event.kind + ':' + (event.data.source ?? '')),
  }
  return evidence
}
