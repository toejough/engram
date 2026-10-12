import fs from 'node:fs'

import { Given } from '@cucumber/cucumber'

const NOT_BUILT = 'pending'

Given(/^the System is (installed|not installed)$/, async function (state) {
  await this.scenario.setup(async (scenario) => {
    if (state === 'installed') await scenario.install()
  })
})

Given(/^(Claude Code is installed|Pi is installed with the memory skills, guidance and shim|Pi is not installed)$/, function (form) {
  if (form !== 'Claude Code is installed') return NOT_BUILT
  if (!fs.existsSync(this.scenario.context.claudePath)) throw new Error('claude is not installed')
  return undefined
})

Given(/^an engram install made before the System existed$/, () => NOT_BUILT)

Given(/^the vault holds (?:a note "([^"]*)" whose situation is "([^"]*)"(?: and which says "([^"]*)")?|no matching note)$/, async function (label, situation, says) {
  if (label == null) return undefined
  await this.scenario.setup((scenario) => scenario.seedNote(label, situation, says))
  return undefined
})

Given(/^the vault holds a pending offer "([^"]*)" that (?:note "([^"]*)" covers|no note covers)$/, () => NOT_BUILT)

Given(/^(?:the vault cannot be read|the System's work for a turn (?:raises an error|times out|takes (\d+) seconds))$/, () => NOT_BUILT)

// A constraint the turn steps enforce (793e2i): a turn in which the agent ran a memory command fails its precondition.
Given(/^(?:the user has not asked for any memory command|the agent runs no memory command itself)$/, function () {
  this.noAgentMemoryCommands = true
})

Given(
  /^an ended (?:Pi|Claude Code) session in which the user said "([^"]*)" and (?:its lesson was not saved|its agent saved the lesson by running the learn skill|its lesson was captured by an earlier Claude Code turn)$/,
  () => NOT_BUILT,
)
