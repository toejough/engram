// Per-scenario hooks (793e2b): the ID of the scenario is the first word of its name; a scenario with no entry in
// scenarios.js is not built yet and ends Pending before any session starts.
import { After, Before, setWorldConstructor, World as CucumberWorld } from '@cucumber/cucumber'

import { entryFor, recordResult } from '../lib/results.js'
import { Scenario } from '../lib/world.js'
import { getRunContext } from '../lib/run-state.js'
import scenarios from '../scenarios.js'

setWorldConstructor(CucumberWorld)

Before(async function ({ pickle }) {
  this.id = pickle.name.split(' ')[0]
  this.title = pickle.name.slice(this.id.length + 1)
  this.data = scenarios[this.id]
  if (!this.data) return 'pending'
  this.scenario = new Scenario(await getRunContext())
  this.retries = 0
  this.substituted = false
  this.skipReason = null
})

After(async function ({ result }) {
  recordResult(entryFor(this, result))
  await this.scenario?.dispose()
})
