import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'

import { Given } from '@cucumber/cucumber'

// SP14 (793e2e): `pi` runs headless but needs its own credentials, which the harness is not given.
const NO_HEADLESS_PI = 'no headless Pi: SP14, pi needs its own credentials and the subscription token is refused'

Given(/^the System is (installed|not installed)$/, async function (state) {
  await this.scenario.setup(async (scenario) => {
    if (state === 'installed') await scenario.install()
    else scenario.treeBefore = scenario.snapshot()
  })
})

Given(/^(Claude Code is installed|Pi is installed with the memory skills, guidance and shim|Pi is not installed)$/, function (form) {
  if (form === 'Pi is installed with the memory skills, guidance and shim') {
    this.skipReason = NO_HEADLESS_PI
    return 'skipped'
  }
  if (form === 'Pi is not installed') {
    assert.ok(!fs.existsSync(path.join(this.scenario.home, '.pi')), 'the World has a Pi directory')
    return undefined
  }
  if (!fs.existsSync(this.scenario.context.claudePath)) throw new Error('claude is not installed')
  return undefined
})

Given(/^an engram install made before the System existed$/, async function () {
  await this.scenario.setup((scenario) => scenario.installPreSystem())
})

Given(/^the vault holds (?:a note "([^"]*)" whose situation is "([^"]*)"(?: and which says "([^"]*)")?|no matching note)$/, async function (label, situation, says) {
  if (label == null) return undefined
  await this.scenario.setup((scenario) => scenario.seedNote(label, situation, says))
  return undefined
})

Given(/^the vault holds a pending offer "([^"]*)" that (?:note "([^"]*)" covers|no note covers)$/, async function (label, covered) {
  await this.scenario.setup((scenario) => scenario.seedOffer(label, covered))
})

// Fault injection by a real mechanism (793e2i step 6): an unreadable vault directory, or an engram shim that fails.
const DELAY_SECONDS = '(?:takes (\\d+) seconds)'
Given(
  new RegExp(`^(?:(the vault cannot be read)|the System's work for a turn (?:(raises an error)|(times out)|${DELAY_SECONDS}))$`),
  async function (unreadable, raises, times, seconds) {
    await this.scenario.setup((scenario) => {
      if (unreadable) scenario.hideVault()
      else if (raises) scenario.injectFault('error')
      else if (times) scenario.injectFault('hang')
      else scenario.injectFault('delay', Number(seconds))
    })
  },
)

// A constraint the turn steps enforce (793e2i): a turn in which the agent ran a memory command fails its precondition.
Given(/^(?:the user has not asked for any memory command|the agent runs no memory command itself)$/, function () {
  this.noAgentMemoryCommands = true
})
