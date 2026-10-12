// Runs cucumber-js over feature files with the harness's support code (793b276). Strict is the default of
// cucumber-js 13 and is stated: pending, undefined and ambiguous steps fail the run.
import { loadConfiguration, runCucumber } from '@cucumber/cucumber/api'
import { Writable } from 'node:stream'

import { allResults, resetResults } from './results.js'

export async function runFeatures({ features, support, dryRun = false, names = [], cwd, worldParameters = {} }) {
  resetResults()
  const { runConfiguration } = await loadConfiguration(
    { provided: { paths: features, import: support, dryRun, strict: true, name: names, format: [], worldParameters } },
    { cwd, env: process.env },
  )
  const sink = new Writable({ write: (chunk, encoding, done) => done() })
  const { success } = await runCucumber(runConfiguration, { cwd, stdout: sink, stderr: sink, env: process.env })
  return { success, results: allResults() }
}
