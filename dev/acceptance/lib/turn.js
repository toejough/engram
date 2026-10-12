// A scripted turn (793b246): the prompt that elicits an event, sent to the scenario's session, and the observation
// that confirms the agent did what the event says. Only this precondition is retried, up to three attempts, each in a
// rebuilt World; the Then of a scenario is never retried.
import { readRows, toolCalls } from './transcript.js'

export const MAX_ATTEMPTS = 3
const TRANSCRIPT_WAIT_MS = 5000
const POLL_MS = 100

export function observed(rows, observe) {
  if (!observe) return true
  return toolCalls(rows).some(
    (call) => call.name === observe.tool && (observe.command === undefined || observe.command.test(call.input.command ?? '')),
  )
}

async function readTranscript(session) {
  const deadline = Date.now() + TRANSCRIPT_WAIT_MS
  for (;;) {
    const file = session.transcriptPath()
    if (file) return readRows(file)
    if (Date.now() > deadline) throw new Error('the session transcript was not written')
    await new Promise((resolve) => setTimeout(resolve, POLL_MS))
  }
}

export async function runTurn(world, script) {
  for (let attempt = 1; ; attempt += 1) {
    const scenario = world.scenario
    const session = scenario.session ?? scenario.startSession()
    const turn = await session.send(script.prompt)
    const rows = await readTranscript(session)
    if (observed(rows, script.observe)) return { turn, rows }
    if (attempt === MAX_ATTEMPTS) {
      throw new Error(`the precondition of the turn was not observed in ${MAX_ATTEMPTS} attempts (${JSON.stringify(script.observe)})`)
    }
    world.retries += 1
    await scenario.rebuild()
  }
}
