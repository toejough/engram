// Per-scenario hooks (793e2b): the ID of the scenario is the first word of its name; a scenario with no entry in
// scenarios.js is not built yet and ends Pending before any session starts.
import fs from 'node:fs'
import path from 'node:path'

import { After, Before, setDefaultTimeout, setWorldConstructor, World as CucumberWorld } from '@cucumber/cucumber'

import { redact } from '../lib/creds.js'
import { entryFor, recordResult } from '../lib/results.js'
import { STEP_TIMEOUT_MS } from '../lib/timeouts.js'
import { Scenario } from '../lib/world.js'
import { getRunContext } from '../lib/run-state.js'
import scenarios from '../scenarios.js'

setWorldConstructor(CucumberWorld)
setDefaultTimeout(STEP_TIMEOUT_MS)

Before(async function ({ pickle }) {
  this.id = pickle.name.split(' ')[0]
  this.title = pickle.name.slice(this.id.length + 1)
  this.data = scenarios[this.id]
  if (!this.data) return 'pending'
  this.scenario = new Scenario(await getRunContext())
  this.retries = 0
  // a note the vault holds that bears on nothing the scenario does (793b316)
  const unrelated = this.data.unrelatedNote
  if (unrelated) await this.scenario.setup((scenario) => scenario.seedNote('UNRELATED', unrelated.situation, unrelated.says))
  this.skipReason = null
})

After(async function ({ result }) {
  const entry = entryFor(this, result)
  // an assertion message can quote tool output, and the agent's tools inherit the session environment
  entry.message = redact(entry.message, (await getRunContext()).creds)
  recordResult(entry)
  // ACCEPTANCE_KEEP=<dir> keeps the World of a scenario that did not pass (transcripts, vault, debug logs) for a look
  const keep = process.env.ACCEPTANCE_KEEP
  if (keep && this.scenario && result.status !== 'PASSED') fs.cpSync(this.scenario.dir, path.join(keep, `${this.id}-${Date.now()}`), { recursive: true })
  await this.scenario?.dispose()
})
