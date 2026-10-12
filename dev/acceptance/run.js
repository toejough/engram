#!/usr/bin/env node
// Entry point of `targ acceptance` (793e2a): node dev/acceptance/run.js [--dry-run] [--check] [filter]
//   filter    scenario ID prefix; 793d selects the H1 scenarios of the technical spec, anything else the product spec
//   --dry-run the vocabulary check only: no session, no credentials
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { extractScenarios, selectScenarios, writeFeatures, PRODUCT_SPEC, TECHNICAL_SPEC } from './lib/extract.js'
import { formatReport, passed } from './lib/report.js'
import { runFeatures } from './lib/runner.js'
import { disposeRun } from './lib/run-state.js'
import { checkVocabulary } from './lib/vocabulary.js'
import { preflight } from './lib/preflight.js'

const ROOT = path.dirname(fileURLToPath(import.meta.url))
const SUPPORT = [path.join(ROOT, 'steps', 'index.js'), path.join(ROOT, 'support', 'hooks.js')]

async function main(argv) {
  const dryRun = argv.includes('--dry-run')
  const filter = argv.find((arg) => !arg.startsWith('--')) ?? ''
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
      console.log(`vocabulary: ${selected.length} scenarios (${report.scenarios} runs), ${report.definitions} definitions, ${report.undefinedSteps.length} undefined, ${report.ambiguousSteps.length} ambiguous`)
      for (const text of [...report.undefinedSteps, ...report.ambiguousSteps]) console.error(`  ${text}`)
      return report.undefinedSteps.length + report.ambiguousSteps.length === 0 ? 0 : 1
    }
    preflight({ needClaude: true })
    const { results } = await runFeatures({ features, support: SUPPORT, cwd: ROOT })
    console.log(formatReport(results))
    return passed(results) && !results.some((entry) => entry.result.startsWith('Pending')) ? 0 : 1
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
