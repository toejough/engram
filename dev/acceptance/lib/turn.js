// A scripted turn (793b246): the prompt that elicits an event, sent to the scenario's session, and the observation
// that confirms the agent did what the event says. Only this precondition is retried, up to three attempts, each in a
// rebuilt World; the Then of a scenario is never retried.
import { memoryCalls, readRows, toolCalls } from './transcript.js'

export const MAX_ATTEMPTS = 3
const TRANSCRIPT_WAIT_MS = 5000
const POLL_MS = 100

function matches(expected, actual) {
  if (expected === undefined) return true
  return expected instanceof RegExp ? expected.test(actual) : expected === actual
}

// observed: the agent did what the event says. `tool` and `command` (and `skill`) are matched against a tool call,
// `reply` against the agent's last reply, `calls` is how many matching tool calls the turn needs (default 1); a script
// with no observation is confirmed by the turn itself.
export function observed(rows, observe, reply = '') {
  if (!observe) return true
  if (observe.reply !== undefined && !matches(observe.reply, reply)) return false
  if (observe.tool === undefined) return true
  const found = toolCalls(rows).filter(
    (call) =>
      matches(observe.tool, call.name) &&
      matches(observe.command, call.input.command ?? '') &&
      matches(observe.skill, call.input.skill ?? ''),
  )
  return found.length >= (observe.calls ?? 1)
}

// constraintsHold: the Given constraints of the scenario ("the user has not asked for any memory command", "the agent
// runs no memory command itself") are enforced by the turn: a turn in which the agent ran one is a failed precondition.
// A script says the same of its own turn with `noMemoryCommands` (the capture scenarios: the System writes, not the agent).
export function constraintsHold(world, rows, script = {}) {
  return !(world.noAgentMemoryCommands || script.noMemoryCommands) || memoryCalls(rows).length === 0
}

export async function readTranscript(session) {
  const deadline = Date.now() + TRANSCRIPT_WAIT_MS
  for (;;) {
    const file = session.transcriptPath()
    if (file) return readRows(file)
    if (Date.now() > deadline) throw new Error('the session transcript was not written')
    await new Promise((resolve) => setTimeout(resolve, POLL_MS))
  }
}

// settle gives the transcript writer a moment to flush the rows of a finished turn before they are read.
export async function settle(ms = 600) {
  await new Promise((resolve) => setTimeout(resolve, ms))
}

export function currentSession(scenario) {
  return scenario.session ?? scenario.startSession()
}

// sendPrompt sends one prompt to the scenario's session and returns what the turn left in the transcript.
export async function sendPrompt(scenario, prompt) {
  const session = currentSession(scenario)
  const turn = await session.send(prompt)
  await settle()
  return { turn, rows: await readTranscript(session) }
}

// attempt runs one try of a precondition and returns its rows, or null when the precondition was not observed. A
// successful try is appended to the setup log, so that a later retry rebuilds the World with this turn replayed.
export async function attempt(world, run, check) {
  for (let tries = 1; ; tries += 1) {
    const result = await run(world.scenario)
    if (check(result)) {
      world.scenario.log.push(async (scenario) => { await run(scenario) })
      return result
    }
    if (tries === MAX_ATTEMPTS) throw new Error(`the precondition of the turn was not observed in ${MAX_ATTEMPTS} attempts`)
    world.retries += 1
    await world.scenario.rebuild()
  }
}

// A script's `files` are input files of the workspace the prompt tells the agent to read: the situation of the turn
// reaches the agent through a tool result, so the prompt itself does not state it.
export async function runTurn(world, script) {
  const result = await attempt(
    world,
    (scenario) => {
      for (const [name, text] of Object.entries(script.files ?? {})) scenario.writeWorkspaceFile(name, text)
      return sendPrompt(scenario, script.prompt)
    },
    ({ turn, rows }) => observed(rows, script.observe, turn.text) && constraintsHold(world, rows, script),
  )
  return result
}
