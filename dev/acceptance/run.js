#!/usr/bin/env node
// Entry point of `targ acceptance` (793e2a): node dev/acceptance/run.js [--dry-run] [filter]
//   filter    scenario ID prefix; 793d selects the H1 scenarios of the technical spec, anything else the product spec
//   --dry-run the vocabulary check only: no session, no credentials
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  extractScenarios,
  productScenarios,
  selectScenarios,
  technicalSpecFiles,
  writeFeatures,
} from './lib/extract.js'
import { parseArgs } from './lib/args.js'
import { secretValues } from './lib/creds.js'
import { formatRates, ratePassed, summarize } from './lib/rate.js'
import { formatReport, passed } from './lib/report.js'
import { runFeatures } from './lib/runner.js'
import { disposeRun } from './lib/run-state.js'
import { checkVocabulary } from './lib/vocabulary.js'
import { preflight } from './lib/preflight.js'
import { ARGS_FROM, EXIT_FAILURE } from './lib/units.js'

const EXIT_INTERRUPTED = 130
// the file a run of a rate run leaves its results in, for the process that started it to read
const RESULTS_FILE = 'ENGRAM_ACCEPTANCE_RESULTS'
const ROOT = path.dirname(fileURLToPath(import.meta.url))
const SUPPORT = [path.join(ROOT, 'steps', 'index.js'), path.join(ROOT, 'support', 'hooks.js')]

async function main(argv) {
  const { dryRun, filter, runs } = parseArgs(argv)
  const technical = filter.startsWith('793d')
  const all = technical ? extractScenarios(technicalSpecFiles(), { idPrefix: '793d17' }) : productScenarios()
  const selected = selectScenarios(all, filter)
  if (selected.length === 0) {
    console.error(`no scenario ID starts with "${filter}" (${all.length} scenarios in the spec)`)
    return EXIT_FAILURE
  }
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'engram-features-'))
  try {
    const features = writeFeatures(selected, dir)
    if (dryRun) {
      const report = await checkVocabulary({ features, support: SUPPORT.slice(0, 1), cwd: ROOT })
      const broken = [...report.undefinedSteps, ...report.ambiguousSteps]
      console.log(
        `vocabulary: ${selected.length} scenarios (${report.scenarios} runs), ${report.definitions} ` +
          `definitions, ${report.undefinedSteps.length} undefined, ${report.ambiguousSteps.length} ambiguous`,
      )
      for (const text of broken) console.error(`  ${text}`)
      return broken.length === 0 ? 0 : 1
    }
    // The vocabulary check is cheap and runs first, in a process of its own (cucumber registers step definitions once
    // per process), so that an undefined or ambiguous step is reported before any session is paid for.
    const vocabulary = spawnSync(process.execPath, [fileURLToPath(import.meta.url), '--dry-run', filter], {
      stdio: 'inherit',
    })
    if (vocabulary.status !== 0) return 1
    preflight({ needClaude: true })
    if (runs > 1) return await rateRun({ filter, runs })
    const { results } = await runFeatures({ features, support: SUPPORT, cwd: ROOT })
    if (process.env[RESULTS_FILE]) fs.writeFileSync(process.env[RESULTS_FILE], JSON.stringify(results))
    console.log(formatReport(results))
    const pending = results.filter((entry) => entry.result.startsWith('Pending')).length
    if (pending > 0)
      console.log(
        `INCOMPLETE: ${pending} scenario runs are Pending (not built); the run found no Red ` +
          `and no harness error: ${passed(results)}`,
      )
    return passed(results) ? 0 : 1
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
    await disposeRun()
  }
}

// rateRun runs the selected rows `runs` times and prints the rate of each and the attribution of each Red (793b421): a
// row that depends on a model's judgement passes at 9 of 10 runs with every Red attributed from the activity line.
async function rateRun({ filter, runs }) {
  const rounds = []
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'engram-rates-'))
  try {
    for (let run = 1; run <= runs; run += 1) {
      // a run is a process of its own: cucumber registers the step definitions once per process
      const file = path.join(dir, `run-${run}.json`)
      spawnSync(process.execPath, [fileURLToPath(import.meta.url), ...(filter ? [filter] : [])], {
        env: { ...process.env, [RESULTS_FILE]: file },
        stdio: ['ignore', 'ignore', 'inherit'],
      })
      if (!fs.existsSync(file)) throw new Error(`run ${run} of ${runs} left no results: it did not reach a session`)
      const results = JSON.parse(fs.readFileSync(file, 'utf8'))
      rounds.push(results)
      console.log(`run ${run} of ${runs}: ${results.map((entry) => `${entry.id} ${entry.result}`).join(', ')}`)
    }
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
  const rows = summarize(rounds)
  console.log(`\n${formatRates(rows, rounds)}`)
  return ratePassed(rows) ? 0 : 1
}

process.on('SIGINT', () => {
  disposeRun().finally(() => process.exit(EXIT_INTERRUPTED))
})

main(process.argv.slice(ARGS_FROM)).then(
  (code) => process.exit(code),
  (error) => {
    // an error can quote what a command printed, and the commands run with the credentials in their environment
    const secrets = secretValues(process.env, { readFile: (file) => fs.readFileSync(file, 'utf8') })
    console.error(`acceptance: ${secrets.reduce((text, value) => text.split(value).join('[redacted]'), error.message)}`)
    process.exit(EXIT_FAILURE)
  },
)
