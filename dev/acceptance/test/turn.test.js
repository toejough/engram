// The scripted turn's preconditions (793b246, 793b316): what a retried precondition is, what it is reported as, and
// when the System's work is over.
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { MARKER } from '../lib/transcript.js'
import { MAX_ATTEMPTS, attempt, awaitIdle, preconditionFailure } from '../lib/turn.js'

const block = { type: 'attachment', attachment: { type: 'hook_additional_context', content: [`${MARKER} notes`] } }
const call = (name, input) => [
  { type: 'assistant', message: { content: [{ type: 'tool_use', id: 't1', name, input }] } },
  { type: 'user', message: { content: [{ type: 'tool_result', tool_use_id: 't1', content: 'ok' }] } },
]

test('a precondition about the agent and one about the System are told apart', () => {
  assert.equal(preconditionFailure(call('Bash', { command: 'ls' }), { tool: 'Bash' }), null)
  assert.deepEqual(preconditionFailure([], { tool: 'Bash' }), {
    system: false,
    message: 'the agent did not do what the event says',
  })
  assert.equal(preconditionFailure([], { delivered: true }).system, true)
  assert.equal(preconditionFailure([block], { delivered: true }), null)
})

test('a tool call is matched on its skill and on its args', () => {
  const rows = call('Skill', { skill: 'route', args: 'write Go tests for internal/cli' })
  assert.equal(preconditionFailure(rows, { tool: 'Skill', skill: 'route', args: /Go tests/ }), null)
  assert.equal(preconditionFailure(rows, { tool: 'Skill', skill: 'route', args: /database/ }).system, false)
})

function fakeWorld() {
  return { retries: 0, scenario: { log: [], rebuild: async () => undefined } }
}

test('a precondition about the agent that fails every attempt is a harness error, not a failed assertion', async () => {
  let tries = 0
  await assert.rejects(
    attempt(
      fakeWorld(),
      async () => (tries += 1),
      () => ({ system: false, message: 'the agent did not' }),
    ),
    (error) => error.name === 'Error' && !(error instanceof assert.AssertionError),
  )
  assert.equal(tries, MAX_ATTEMPTS)
})

test('a precondition about the System that fails every attempt is a failed assertion, so the row is Red', async () => {
  await assert.rejects(
    attempt(
      fakeWorld(),
      async () => 1,
      () => ({ system: true, message: 'the System delivered nothing' }),
    ),
    (error) => error instanceof assert.AssertionError && /delivered nothing/.test(error.message),
  )
})

test('a precondition that holds on a later attempt is a pass', async () => {
  let tries = 0
  const world = fakeWorld()
  assert.equal(
    await attempt(
      world,
      async () => (tries += 1),
      (result) => (result === 2 ? true : { system: false, message: 'x' }),
    ),
    2,
  )
  assert.equal(world.retries, 1)
})

function seamDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'turn-test-'))
}

const FAST = { graceMs: 120, pollMs: 10, stableMs: 20 }

test(
  'a session whose turn has not ended is not idle: the grace period runs from the result, ' + 'not from the wait',
  async () => {
    const dir = seamDir()
    const session = { waiting: {}, lastResultAt: Date.now() }
    const scenario = { seamFile: path.join(dir, 'seam.log'), sessions: [session] }
    const started = Date.now()
    setTimeout(() => {
      session.waiting = null
      session.lastResultAt = Date.now()
    }, 200)
    await awaitIdle(scenario, FAST)
    assert.ok(Date.now() - started >= 200 + FAST.graceMs - 15, `returned after ${Date.now() - started} ms`)
    fs.rmSync(dir, { recursive: true, force: true })
  },
)

test('work queued after the grace period began is waited for to its idle line', async () => {
  const dir = seamDir()
  const seam = path.join(dir, 'seam.log')
  const scenario = { seamFile: seam, sessions: [{ waiting: null, lastResultAt: Date.now() }] }
  setTimeout(() => fs.writeFileSync(seam, 'queued\n'), 60)
  setTimeout(() => fs.writeFileSync(seam, 'queued\nidle\n'), 220)
  const started = Date.now()
  await awaitIdle(scenario, FAST)
  assert.ok(Date.now() - started >= 220, `returned after ${Date.now() - started} ms, before the idle line`)
  fs.rmSync(dir, { recursive: true, force: true })
})

test(
  'a home without the System queues nothing and is idle once the grace period after ' + 'the result has passed',
  async () => {
    const dir = seamDir()
    const scenario = { seamFile: path.join(dir, 'seam.log'), sessions: [{ waiting: null, lastResultAt: Date.now() }] }
    const started = Date.now()
    await awaitIdle(scenario, FAST)
    assert.ok(Date.now() - started >= FAST.graceMs - 15)
    fs.rmSync(dir, { recursive: true, force: true })
  },
)
