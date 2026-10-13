// A scripted turn (793b246): the prompt that elicits an event, sent to the scenario's session, and the observation
// that confirms the agent did what the event says. Only this precondition is retried, up to three attempts, each in a
// rebuilt World; the Then of a scenario is never retried.
import fs from 'node:fs'

import { agentCalls, memoryCalls, readRows, systemBlocks, toolCalls } from './transcript.js'

export const MAX_ATTEMPTS = 3
const TRANSCRIPT_WAIT_MS = 5000
const POLL_MS = 100

function matches(expected, actual) {
  if (expected === undefined) return true
  return expected instanceof RegExp ? expected.test(actual) : expected === actual
}

// observed: the agent did what the event says. `tool` and `command` (and `skill`) are matched against a tool call,
// `reply` against the agent's last reply, `delivered` that the System delivered a block in the turn, `calls` is how many matching tool calls the turn needs (default 1); a script
// with no observation is confirmed by the turn itself.
export function observed(rows, observe, reply = '', subagents = []) {
  if (!observe) return true
  if (observe.delivered && systemBlocks(rows).length === 0) return false
  if (observe.reply !== undefined && !matches(observe.reply, reply)) return false
  if (observe.tool === undefined) return true
  const found = toolCalls(rows).filter(
    (call) =>
      matches(observe.tool, call.name) &&
      matches(observe.command, call.input.command ?? '') &&
      matches(observe.skill, call.input.skill ?? ''),
  )
  if (found.length < (observe.calls ?? 1)) return false
  // `subagentsRead`: each dispatched subagent's transcript holds the text, so the lesson reached the subagent's session
  if (observe.subagentsRead === undefined) return true
  return subagents.length >= found.length && subagents.filter((text) => observe.subagentsRead.test(text)).length >= found.length
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

const IDLE_WAIT_MS = 180000
const NO_WORK_GRACE_MS = 3000
const IDLE_STABLE_MS = 1500

// awaitIdle waits for the System's work to finish: the mod's seam file ends with `idle` once its queue is empty. A
// session that queued nothing leaves no file, which a short grace period tells from work not yet queued.
export async function awaitIdle(scenario) {
  const start = Date.now()
  for (;;) {
    let lines = []
    try {
      lines = fs.readFileSync(scenario.seamFile, 'utf8').split('\n').filter((line) => line !== '')
    } catch {
      lines = []
    }
    if (lines.length > 0 && lines.at(-1) === 'idle') {
      // idle stays idle: no job joined the queue in the next moment
      await new Promise((resolve) => setTimeout(resolve, IDLE_STABLE_MS))
      if (fs.readFileSync(scenario.seamFile, 'utf8').split('\n').filter((line) => line !== '').length === lines.length) return
      continue
    }
    if (lines.length === 0 && Date.now() - start > NO_WORK_GRACE_MS) return
    if (Date.now() - start > IDLE_WAIT_MS) throw new Error('the System\'s work did not finish: no idle line in the seam file')
    await new Promise((resolve) => setTimeout(resolve, POLL_MS))
  }
}

export function currentSession(scenario) {
  return scenario.session ?? scenario.startSession()
}

const DISPATCH_WAIT_MS = 180000
const QUIET_MS = 3000

// reported: the subagents the main transcript says finished, by task id (a notification may repeat for one task).
function reported(rows) {
  const ids = new Set()
  for (const row of rows) {
    if (row.type !== 'user') continue
    const text = typeof row.message?.content === 'string' ? row.message.content : JSON.stringify(row.message?.content ?? '')
    if (!text.includes('<status>completed</status>')) continue
    for (const match of text.matchAll(/<task-id>([^<]+)<\/task-id>/g)) ids.add(match[1])
  }
  return ids.size
}

// afterDispatches: a turn that dispatched subagents is not over at its first result. The subagents run in the
// background and each one's report starts a turn of the session of its own, so the turn ends when every dispatched
// subagent has reported and the session has been quiet for a moment.
async function afterDispatches(session, rows) {
  const dispatched = agentCalls(rows).length
  if (dispatched === 0) return rows
  const deadline = Date.now() + DISPATCH_WAIT_MS
  let current = rows
  while (Date.now() < deadline) {
    current = readRows(session.transcriptPath())
    if (reported(current) >= dispatched) break
    await new Promise((resolve) => setTimeout(resolve, POLL_MS))
  }
  for (let seen = -1; seen !== session.turns.length && Date.now() < deadline; ) {
    seen = session.turns.length
    await new Promise((resolve) => setTimeout(resolve, QUIET_MS))
  }
  await settle()
  return readRows(session.transcriptPath())
}

// sendPrompt sends one prompt to the scenario's session and returns what the turn left in the transcript.
export async function sendPrompt(scenario, prompt) {
  const session = currentSession(scenario)
  const turn = await session.send(prompt)
  await settle()
  return { turn, rows: await afterDispatches(session, await readTranscript(session)) }
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

// subagentTexts are the transcripts of the subagents the scenario's session dispatched, as text.
function subagentTexts(scenario) {
  return scenario.session ? scenario.session.subagentTranscripts().map((file) => fs.readFileSync(file, 'utf8')) : []
}

// A script's `files` are input files of the workspace the prompt tells the agent to read: the situation of the turn
// reaches the agent through a tool result, so the prompt itself does not state it.
export async function runTurn(world, script) {
  world.typed = script.prompt
  const result = await attempt(
    world,
    (scenario) => {
      for (const [name, text] of Object.entries(script.files ?? {})) scenario.writeWorkspaceFile(name, text)
      return sendPrompt(scenario, script.prompt)
    },
    ({ turn, rows }) =>
      observed(rows, script.observe, turn.text, subagentTexts(world.scenario)) && constraintsHold(world, rows, script),
  )
  return result
}
