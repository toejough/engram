#!/usr/bin/env node
// node probes/run.js SP3 [SP4 ...]: runs the spike probes and prints their evidence as JSON (793e8a).
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { withProbeContext } from './lib.js'
import { ARGS_FROM, JSON_INDENT, tenthsOfSeconds } from '../lib/units.js'

const HERE = path.dirname(fileURLToPath(import.meta.url))

async function main(names) {
  for (const name of names) {
    const probe = await import(path.join(HERE, `${name}.js`))
    if (probe.sameSessionAs) {
      console.log(
        JSON.stringify({ spike: name, claim: probe.claim, evidence: `see ${probe.sameSessionAs}: the same session` }),
      )
      continue
    }
    const started = Date.now()
    let evidence
    try {
      evidence = await withProbeContext((harness) => probe.run(harness), probe.options ?? {})
    } catch (error) {
      evidence = { error: error.message }
    }
    console.log(
      JSON.stringify(
        { spike: name, claim: probe.claim, wallS: tenthsOfSeconds(Date.now() - started), evidence },
        null,
        JSON_INDENT,
      ),
    )
  }
}

main(process.argv.slice(ARGS_FROM)).catch((error) => {
  console.error(error.message)
  process.exit(1)
})
