#!/usr/bin/env node
// Entry point of the refactor gates (793e8l): node dev/acceptance/gates/run.js <gate> [arguments]
//   lint [standards|complexity|adapter|types]   eslint over the mod and the harness, the limits, the adapter rules, tsc
//   budget                              owned lines of each hook file against dev/acceptance/gates/budgets.json
//   mod-test [green|red] [file-prefix]  the mod's contract tests: tests/*.test.ts, or the gates tests/*.red.ts
//   spec                                the specs 793d22 onward against the mod's tests, the product spec and the code
//   status                              every gate above, in one table
// Every gate exits non-zero while it is red. The ones that are green are in check-full; the others are on the
// refactor-red list (793e8l). Each gate's module is loaded when it runs, so that a gate that has not landed does not
// stop the others.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = path.dirname(fileURLToPath(import.meta.url))
const MOD = path.resolve(HERE, '..', '..', '..', 'agent-instructions', 'mods', 'engram-memory')
const HOOKS = path.join(MOD, 'hooks')
const GO_DIR = path.resolve(HERE, '..', '..', '..', 'internal', 'auto')
const NAME_WIDTH = 12
const ARGS_FROM = 2
const USAGE_EXIT = 2

const hookFiles = () => fs.readdirSync(HOOKS).filter((name) => name.endsWith('.js'))

async function lintGate(part) {
  const { lint } = await import('./lint.js')
  return lint(part ?? '') === 0 ? 0 : 1
}

async function budget() {
  const { measure, measureGo, goTable, goal, compare, externalImports, formatTable, passes } = await import(
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
  const result = compare(measure(HOOKS), table)
  const imports = hookFiles().flatMap((name) =>
    externalImports(fs.readFileSync(path.join(HOOKS, name), 'utf8')).map((module) => `${name} imports ${module}`),
  )
  const over = result.totalExcess > 0 || result.over.length > 0
  console.log(formatTable(result))
  for (const row of result.unbudgeted) console.log(`${row.file} has no budget`)
  for (const line of imports) console.log(`dependency: ${line}`)
  console.log(`minimality: ${over ? 'over budget' : 'within budget'}, ${imports.length} external imports`)
  return passes(result, imports) && passes(goResult, []) ? 0 : 1
}

async function modTest(which = 'green', filter = '') {
  const { run } = await import('./mod-test.js')
  const result = run({ which, filter })
  if (which === 'red') {
    console.log(`${which} gates (${result.files.join(', ')}): ${result.pass.length} pass, ${result.fail.length} fail`)
    for (const test of result.tests) {
      console.log(`  ${test.ok ? 'pass' : 'FAIL'}  ${test.file.replace(/\.test\.ts$/, '.red.ts')}  ${test.name}`)
    }
  } else {
    process.stdout.write(result.output)
  }
  return result.status === 0 && result.fail.length === 0 ? 0 : 1
}

async function spec() {
  const specGate = await import('./spec.js')
  const { run } = await import('./mod-test.js')
  const { extractScenarios, PRODUCT_SPEC, TECHNICAL_SPEC, REPO_ROOT } = await import('../lib/extract.js')
  const text = fs.readFileSync(TECHNICAL_SPEC, 'utf8')
  const requirements = specGate.parseSpec(text)
  const { tests } = run({ which: 'green' })
  const hooks = hookFiles().map((name) => fs.readFileSync(path.join(HOOKS, name), 'utf8'))
  const read = (file) =>
    fs.existsSync(path.join(REPO_ROOT, file)) ? fs.readFileSync(path.join(REPO_ROOT, file), 'utf8') : null
  const problems = [
    ...specGate.check({
      requirements,
      productIds: extractScenarios(PRODUCT_SPEC).map((scenario) => scenario.id),
      tests,
      goFiles: specGate.loadGoFiles(REPO_ROOT, requirements),
    }),
    ...specGate.checkRealizations(specGate.realizations(text), { read, hooks }),
    ...specGate.checkTestCount(specGate.statedTestCount(text), tests.length),
  ]
  console.log(
    `spec conformance: ${requirements.length} requirements, ${tests.length} mod tests, ${problems.length} problems`,
  )
  for (const problem of problems) console.log(`  ${problem}`)
  return problems.length === 0 ? 0 : 1
}

// the gates that are still red: the five of nodes 793e8m and 793e8n (complexity, adapter, minimality, standards, types)
// and the mod's contract tests (793e8p) are green and in check-full
const GATES = [
  ['spec', 'targ check-spec-conformance', spec],
]

async function status() {
  const rows = []
  for (const [name, command, gate] of GATES) {
    console.log(`\n== ${name}: ${command}`)
    rows.push([name, command, await gate()])
  }
  console.log('\ngate         state   command')
  for (const [name, command, code] of rows) {
    console.log(`${name.padEnd(NAME_WIDTH)} ${code === 0 ? 'green  ' : 'RED    '} ${command}`)
  }
  return rows.every(([, , code]) => code === 0) ? 0 : 1
}

const [gate, ...rest] = process.argv.slice(ARGS_FROM)
const commands = {
  lint: () => lintGate(rest[0]),
  budget,
  'mod-test': () => modTest(rest[0], rest[1]),
  spec,
  status,
}
if (!(gate in commands)) {
  console.error(`usage: node gates/run.js ${Object.keys(commands).join('|')} [arguments]`)
  process.exit(USAGE_EXIT)
}
process.exitCode = await commands[gate]()
