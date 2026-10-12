import { When } from '@cucumber/cucumber'

import { runTurn } from '../lib/turn.js'

const NOT_BUILT = 'pending'

When(/^a turn completes in which "([^"]*)"$/, async function (event) {
  const script = this.data.turns?.[event]
  if (!script) throw new Error(`scenarios.js has no turn script for ${this.id}: "${event}"`)
  this.lastTurn = await runTurn(this, script)
})

When(
  /^(?:the human|an automated task notification|a scheduled trigger|a message from another session|a Pi user) delivers a prompt matching note "([^"]*)"(?: that says "([^"]*)")?$/,
  () => NOT_BUILT,
)

When(/^the user submits the prompt "([^"]*)" immediately$/, () => NOT_BUILT)

When(/^the session is (?:compacted|ended and resumed)$/, () => NOT_BUILT)

When(/^the user runs engram update(?: --dry-run)?$/, () => NOT_BUILT)
