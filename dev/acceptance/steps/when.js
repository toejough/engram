import assert from 'node:assert/strict'

import { When } from '@cucumber/cucumber'

import { deliverPrompt } from '../lib/origins.js'
import { changes } from '../lib/tree.js'
import { runTurn, sendPrompt, settle } from '../lib/turn.js'

const NOT_BUILT = 'pending'

When(/^a turn completes in which "([^"]*)"$/, async function (event) {
  const script = this.data.turns?.[event]
  if (!script) throw new Error(`scenarios.js has no turn script for ${this.id}: "${event}"`)
  this.origin = null
  this.lastTurn = await runTurn(this, script)
})

When(
  /^(the human|an automated task notification|a scheduled trigger|a message from another session|a Pi user) delivers a prompt matching note "([^"]*)"(?: that says "([^"]*)")?$/,
  async function (origin, label, correction) {
    if (origin === 'a Pi user') return NOT_BUILT
    this.lastTurn = await deliverPrompt(this, origin, label, correction)
    return undefined
  },
)

// At once: the prompt goes in as the previous turn's result arrives, without waiting for the System's work (793e2i step 11).
When(/^the user submits the prompt "([^"]*)" immediately$/, async function (prompt) {
  this.typed = prompt
  this.lastTurn = await sendPrompt(this.scenario, prompt)
})

// compact sends /compact in the stream (SP7); resume ends the session and starts one that resumes it by id.
async function compact(scenario) {
  await scenario.session.send('/compact')
  await settle()
}

async function resume(scenario) {
  const { sessionId } = scenario.session
  await scenario.session.close()
  scenario.startSession({ resume: sessionId })
}

When(/^the session is (compacted|ended and resumed)$/, async function (what) {
  const action = what === 'compacted' ? compact : resume
  await action(this.scenario)
  this.scenario.log.push(action)
})

// A preview writes nothing: the tree it ran against is the tree it leaves (793a5j).
When(/^the user runs engram update( --dry-run)?$/, async function (dryRun) {
  await this.scenario.setup(async (scenario) => {
    const before = scenario.snapshot()
    await scenario.runUpdate({ dryRun: Boolean(dryRun) })
    if (dryRun) assert.deepEqual(changes(before, scenario.snapshot()), [], 'the preview wrote to the install tree')
  })
})
