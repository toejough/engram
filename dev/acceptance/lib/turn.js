// A scripted turn (793b246): the prompt that elicits an event, sent to the scenario's session, and the observation
// that confirms the agent did what the event says. Only this precondition is retried, up to three attempts, each in a
// rebuilt World; the Then of a scenario is never retried.
import assert from 'node:assert/strict'
import fs from 'node:fs'
import { setTimeout as sleep } from 'node:timers/promises'

import { agentCalls, memoryCalls, readRows, systemBlocks, toolCalls } from './transcript.js'

export const MAX_ATTEMPTS = 3
const TRANSCRIPT_WAIT_MS = 5000
const POLL_MS = 100
const SETTLE_MS = 600

function matches(expected, actual) {
  if (expected === undefined) return true
  return expected instanceof RegExp ? expected.test(actual) : expected === actual
}

// preconditionFailure is null when the turn left what the script's event needs, else why it did not. `tool`, `command`,
// `skill` and `args` are matched against a tool call, `reply` against the agent's last reply, `calls` is how many
// matching tool calls the turn needs (default 1), `subagentsRead` that each dispatched subagent's transcript holds the
// text. All of these are about what the agent did; `delivered` (the turn left a block of the System) is about the
// System itself, and a precondition the System did not meet is a failed assertion and not a harness error (793b316).
export function preconditionFailure(rows, observe, reply = '', subagents = []) {
  if (!observe) return null
  const agent = { system: false, message: 'the agent did not do what the event says' }
  if (observe.reply !== undefined && !matches(observe.reply, reply)) return agent
  if (observe.tool !== undefined) {
    const found = toolCalls(rows).filter(
      (call) =>
        matches(observe.tool, call.name) &&
        matches(observe.command, call.input.command ?? '') &&
        matches(observe.skill, call.input.skill ?? '') &&
        matches(observe.args, call.input.args ?? ''),
    )
    if (found.length < (observe.calls ?? 1)) return agent
    // each dispatched subagent's transcript holds the text, so the lesson reached the subagent's session
    if (
      observe.subagentsRead !== undefined &&
      !(
        subagents.length >= found.length &&
        subagents.filter((text) => observe.subagentsRead.test(text)).length >= found.length
      )
    )
      return agent
  }
  if (observe.delivered && systemBlocks(rows).length === 0)
    return { system: true, message: 'the System delivered nothing in a turn that needed a delivery' }
  return null
}

// constraintsHold: the Given constraints of the scenario ("the user has not asked for any memory command", "the agent
// runs no memory command itself") are enforced by the turn: a turn in which the agent ran one is a failed precondition.
// A script says the same of its own turn with `noMemoryCommands` (the capture scenarios: the System writes, not the
// agent).
export function constraintsHold(world, rows, script = {}) {
  return !(world.noAgentMemoryCommands || script.noMemoryCommands) || memoryCalls(rows).length === 0
}

// failedPrecondition: why a try of a scripted turn is not the event, or null.
function failedPrecondition(world, script, { turn, rows }) {
  const failure = preconditionFailure(rows, script.observe, turn.text, subagentTexts(world.scenario))
  if (failure) return failure
  return constraintsHold(world, rows, script)
    ? null
    : { system: false, message: 'the agent ran a memory command itself' }
}

export async function readTranscript(session) {
  const deadline = Date.now() + TRANSCRIPT_WAIT_MS
  for (;;) {
    const file = session.transcriptPath()
    if (file) return readRows(file)
    if (Date.now() > deadline) throw new Error('the session transcript was not written')
    await sleep(POLL_MS)
  }
}

// settle gives the transcript writer a moment to flush the rows of a finished turn before they are read.
export async function settle(milliseconds = SETTLE_MS) {
  await sleep(milliseconds)
}

const IDLE_WAIT_MS = 180000
const NO_WORK_GRACE_MS = 3000
const IDLE_STABLE_MS = 1500

const seamLines = (file) => {
  try {
    return fs
      .readFileSync(file, 'utf8')
      .split('\n')
      .filter((line) => line !== '')
  } catch {
    return []
  }
}

// awaitIdle waits for the System's work to finish: the mod's seam file ends with `idle` once its queue is empty. The
// mod writes `queued` before the hook of a completed turn returns, so a session that queued nothing leaves no file;
// that is told from work not yet queued by a grace period that begins when the last turn's result arrived, and never
// while a session is still in a turn: an absence guard cannot pass before the System had started (793b316).
export async function awaitIdle(
  scenario,
  { graceMs = NO_WORK_GRACE_MS, pollMs = POLL_MS, stableMs = IDLE_STABLE_MS } = {},
) {
  const start = Date.now()
  const sessions = () => scenario.sessions ?? []
  for (;;) {
    const lines = seamLines(scenario.seamFile)
    if (lines.at(-1) === 'idle') {
      // idle stays idle: no job joined the queue in the next moment
      await sleep(stableMs)
      if (seamLines(scenario.seamFile).length === lines.length) return
      continue
    }
    const inTurn = sessions().some((session) => session.waiting)
    const lastResult = Math.max(start, ...sessions().map((session) => session.lastResultAt ?? 0))
    if (lines.length === 0 && !inTurn && Date.now() - lastResult > graceMs) return
    if (Date.now() - start > IDLE_WAIT_MS)
      throw new Error("the System's work did not finish: no idle line in the seam file")
    await sleep(pollMs)
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
    const text =
      typeof row.message?.content === 'string' ? row.message.content : JSON.stringify(row.message?.content ?? '')
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
  while (Date.now() < deadline) {
    const current = readRows(session.transcriptPath())
    if (reported(current) >= dispatched) break
    await sleep(POLL_MS)
  }
  for (let seen = -1; seen !== session.turns.length && Date.now() < deadline; ) {
    seen = session.turns.length
    await sleep(QUIET_MS)
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

// attempt runs one try of a precondition and returns its result once `check` accepts it: check returns null (or true)
// when the try is what the event needs, else why not, as { system, message }. A successful try is appended to the setup
// log, so that a retry rebuilds the World with this turn replayed. A precondition the agent did not meet in three
// attempts is a harness error (the model did not do what the script asked); one the System did not meet is a failed
// assertion, which makes the row Red (793b316).
export async function attempt(world, run, check) {
  for (let tries = 1; ; tries += 1) {
    const result = await run(world.scenario)
    const failure = check(result)
    if (failure === null || failure === true) {
      world.scenario.log.push(async (scenario) => {
        await run(scenario)
      })
      return result
    }
    if (tries === MAX_ATTEMPTS) {
      const reason = failure.message ?? 'not observed'
      const message = `the precondition of the turn was not observed in ${MAX_ATTEMPTS} attempts: ${reason}`
      throw failure.system ? new assert.AssertionError({ message }) : new Error(message)
    }
    world.retries += 1
    await world.scenario.rebuild()
  }
}

// subagentTexts are the transcripts of the subagents the scenario's session dispatched, as text.
function subagentTexts(scenario) {
  return scenario.session ? scenario.session.subagentTranscripts().map((file) => fs.readFileSync(file, 'utf8')) : []
}

// seed puts a script's `seeds` in the vault before its prompt is sent (793e8f): the notes a delivery should find and
// the pending offers housekeeping should settle. A near offer (`extend`) states the claim of its note and one more.
async function seed(scenario, seeds = {}) {
  for (const note of seeds.notes ?? []) await scenario.seedNote(note.label, note.situation, note.says)
  for (const offer of seeds.offers ?? []) await scenario.seedOffer(offer.label, offer.covers, offer.extend)
}

// A script's `files` are input files of the workspace the prompt tells the agent to read: the situation of the turn
// reaches the agent through a tool result, so the prompt itself does not state it.
export async function runTurn(world, script) {
  world.typed = script.prompt
  const result = await attempt(
    world,
    async (scenario) => {
      for (const [name, text] of Object.entries(script.files ?? {})) scenario.writeWorkspaceFile(name, text)
      await seed(scenario, script.seeds)
      const sent = await sendPrompt(scenario, script.prompt)
      // `settle`: the user reads the System's line before the next prompt (a summary counts since the previous one)
      if (script.settle) await awaitIdle(scenario)
      return sent
    },
    (sent) => failedPrecondition(world, script, sent),
  )
  return result
}
