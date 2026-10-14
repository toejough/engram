#!/usr/bin/env node
// Entry point of the refactor gates (793e8l): node dev/acceptance/gates/run.js <gate> [arguments]
//   lint [standards|complexity|adapter|types]   eslint over the mod and the harness, the limits, the adapter rules, tsc
//   budget                              owned lines of each JavaScript file of the mod against budgets.json
//   mod-test [file-prefix]              the mod's contract tests: tests/*.test.ts
//   spec                                the technical spec against the mod's and Go tests, the product spec, the code
// Every gate exits non-zero while it is red, and every gate is in check-full. Each gate's module is loaded when it
// runs.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = path.dirname(fileURLToPath(import.meta.url))
const MOD = path.resolve(HERE, '..', '..', '..', 'agent-instructions', 'mods', 'engram-memory')
const GO_DIR = path.resolve(HERE, '..', '..', '..', 'internal', 'auto')
const ARGS_FROM = 2
const USAGE_EXIT = 2


async function lintGate(part) {
  const { lint } = await import('./lint.js')
  return lint(part ?? '') === 0 ? 0 : 1
}

async function budget() {
  const { measure, measureGo, goTable, goal, compare, externalImports, formatTable, jsFilesOf, passes } = await import(
    './owned-lines.js'
  )
  const table = JSON.parse(fs.readFileSync(path.join(HERE, 'budgets.json'), 'utf8'))
  const goResult = compare(measureGo(GO_DIR, table.go), goTable(table.go))
  const goals = Object.entries(table.go.components).map(([name, row]) => `${name} ${goal(row, table.go.ratio)}`)
  console.log(
    `Go components of ${table.go.dir}: owned lines against their ceilings; goal at ${table.go.ratio} ` +
      `times the owned JavaScript lines replaced: ${goals.join(', ')}`,
  )
  console.log(formatTable(goResult))
  for (const row of goResult.unbudgeted) console.log(`${row.file} has no budget`)
  console.log(
    `Go minimality: ${goResult.totalExcess > 0 || goResult.over.length > 0 ? 'over budget' : 'within budget'}\n`,
  )
  const result = compare(measure(MOD), table)
  const imports = jsFilesOf(MOD).flatMap((name) =>
    externalImports(fs.readFileSync(path.join(MOD, name), 'utf8')).map((module) => `${name} imports ${module}`),
  )
  const over = result.totalExcess > 0 || result.over.length > 0
  console.log(formatTable(result))
  for (const row of result.unbudgeted) console.log(`${row.file} has no budget`)
  for (const line of imports) console.log(`dependency: ${line}`)
  console.log(`minimality: ${over ? 'over budget' : 'within budget'}, ${imports.length} external imports`)
  return passes(result, imports) && passes(goResult, []) ? 0 : 1
}

async function modTest(filter = '') {
  const { run } = await import('./mod-test.js')
  const result = run({ filter })
  process.stdout.write(result.output)
  return result.status === 0 && result.fail.length === 0 ? 0 : 1
}

async function spec() {
  const specGate = await import('./spec.js')
  const { run } = await import('./mod-test.js')
  const { productScenarios, technicalText, REPO_ROOT } = await import('../lib/extract.js')
  const { jsFilesOf } = await import('./owned-lines.js')
  const text = technicalText()
  const requirements = specGate.parseSpec(text)
  const { tests } = run()
  const hooks = jsFilesOf(MOD).map((name) => fs.readFileSync(path.join(MOD, name), 'utf8'))
  const read = (file) =>
    fs.existsSync(path.join(REPO_ROOT, file)) ? fs.readFileSync(path.join(REPO_ROOT, file), 'utf8') : null
  const problems = [
    ...specGate.check({
      requirements,
      productIds: productScenarios().map((scenario) => scenario.id),
      tests,
      goFiles: specGate.loadGoFiles(REPO_ROOT, requirements),
    }),
    ...specGate.checkRealizations(specGate.realizations(text), { read, hooks }),
    ...specGate.checkSecurity(specGate.securityRows(text), { requirements }),
  ]
  console.log(
    `spec conformance: ${requirements.length} requirements, ${tests.length} mod tests, ${problems.length} problems`,
  )
  for (const problem of problems) console.log(`  ${problem}`)
  return problems.length === 0 ? 0 : 1
}

const [gate, ...rest] = process.argv.slice(ARGS_FROM)
const commands = {
  lint: () => lintGate(rest[0]),
  budget,
  'mod-test': () => modTest(rest[0]),
  spec,
}
if (!(gate in commands)) {
  console.error(`usage: node gates/run.js ${Object.keys(commands).join('|')} [arguments]`)
  process.exit(USAGE_EXIT)
}
process.exitCode = await commands[gate]()
