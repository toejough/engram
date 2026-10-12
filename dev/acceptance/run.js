#!/usr/bin/env node
// Entry point of `targ acceptance` (793e2a): node dev/acceptance/run.js [--dry-run] [filter]
//   filter    scenario ID prefix; 793d selects the H1 scenarios of the technical spec, anything else the product spec
//   --dry-run the vocabulary check only: no session, no credentials
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { extractScenarios, selectScenarios, writeFeatures, PRODUCT_SPEC, TECHNICAL_SPEC } from './lib/extract.js'
import { parseArgs } from './lib/args.js'
import { formatReport, passed } from './lib/report.js'
import { runFeatures } from './lib/runner.js'
import { disposeRun } from './lib/run-state.js'
import { checkVocabulary } from './lib/vocabulary.js'
import { preflight } from './lib/preflight.js'

const ROOT = path.dirname(fileURLToPath(import.meta.url))
const SUPPORT = [path.join(ROOT, 'steps', 'index.js'), path.join(ROOT, 'support', 'hooks.js')]

async function main(argv) {
  const { dryRun, filter } = parseArgs(argv)
  const technical = filter.startsWith('793d')
  const all = technical ? extractScenarios(TECHNICAL_SPEC, { idPrefix: '793d17' }) : extractScenarios(PRODUCT_SPEC)
  const selected = selectScenarios(all, filter)
  if (selected.length === 0) {
    console.error(`no scenario ID starts with "${filter}" (${all.length} scenarios in the spec)`)
    return 2
  }
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'engram-features-'))
  try {
    const features = writeFeatures(selected, dir)
    if (dryRun) {
      const report = await checkVocabulary({ features, support: SUPPORT.slice(0, 1), cwd: ROOT })
      const broken = [...report.undefinedSteps, ...report.ambiguousSteps]
      console.log(`vocabulary: ${selected.length} scenarios (${report.scenarios} runs), ${report.definitions} definitions, ${report.undefinedSteps.length} undefined, ${report.ambiguousSteps.length} ambiguous`)
      for (const text of broken) console.error(`  ${text}`)
      return broken.length === 0 ? 0 : 1
    }
    // The vocabulary check is cheap and runs first, in a process of its own (cucumber registers step definitions once
    // per process), so that an undefined or ambiguous step is reported before any session is paid for.
    const vocabulary = spawnSync(process.execPath, [fileURLToPath(import.meta.url), '--dry-run', filter], { stdio: 'inherit' })
    if (vocabulary.status !== 0) return 1
    preflight({ needClaude: true })
    const { results } = await runFeatures({ features, support: SUPPORT, cwd: ROOT })
    console.log(formatReport(results))
    const pending = results.filter((entry) => entry.result.startsWith('Pending')).length
    if (pending > 0) console.log(`INCOMPLETE: ${pending} scenario runs are Pending (not built); the run found no Red and no harness error: ${passed(results)}`)
    return passed(results) ? 0 : 1
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
    await disposeRun()
  }
}

process.on('SIGINT', () => {
  disposeRun().finally(() => process.exit(130))
})

main(process.argv.slice(2)).then(
  (code) => process.exit(code),
  (error) => {
    console.error(`acceptance: ${error.message}`)
    process.exit(2)
  },
)
