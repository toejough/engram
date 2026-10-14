// The JavaScript standards and complexity gates (793e8n, 793e8m): eslint over the mod and the harness, and `tsc
// --noEmit` over the mod's test files. Each part exits non-zero while it reports a problem, and prints the numbers the
// gate is judged by: problems per rule and per file.
import { spawnSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const PACKAGE = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
export const REPO = path.resolve(PACKAGE, '..', '..')
const MOD = 'agent-instructions/mods/engram-memory'
const ESLINT = path.join(PACKAGE, 'node_modules', 'eslint', 'bin', 'eslint.js')
const TSC = path.join(PACKAGE, 'node_modules', 'typescript', 'bin', 'tsc')
const KIBIBYTE = 1024
const BUFFER_MEBIBYTES = 256
const MAX_BUFFER = BUFFER_MEBIBYTES * KIBIBYTE * KIBIBYTE
const PARTS = ['standards', 'complexity', 'adapter', 'types']
const COUNT_WIDTH = 5
const FILES_SHOWN = 8

// tally counts the messages of eslint's JSON report by rule and by file (paths relative to the repository).
export function tally(report) {
  const rules = new Map()
  const files = new Map()
  let problems = 0
  for (const file of report) {
    for (const message of file.messages) {
      problems += 1
      const rule = message.ruleId ?? 'parse error'
      rules.set(rule, (rules.get(rule) ?? 0) + 1)
      const name = path.relative(REPO, file.filePath)
      files.set(name, (files.get(name) ?? 0) + 1)
    }
  }
  const byCount = (left, right) => right[1] - left[1] || left[0].localeCompare(right[0])
  const sorted = (map) => [...map.entries()].sort(byCount)
  return { problems, rules: sorted(rules), files: sorted(files) }
}

function eslint(config, targets) {
  const run = spawnSync(process.execPath, [ESLINT, '-c', path.join(PACKAGE, config), '-f', 'json', ...targets], {
    cwd: REPO,
    encoding: 'utf8',
    maxBuffer: MAX_BUFFER,
  })
  if (run.status !== 0 && run.status !== 1) throw new Error(`eslint failed to run (exit ${run.status}): ${run.stderr}`)
  return tally(JSON.parse(run.stdout))
}

function printTally(name, result, verbose) {
  console.log(`${name}: ${result.problems} problems`)
  for (const [rule, count] of result.rules) console.log(`  ${String(count).padStart(COUNT_WIDTH)}  ${rule}`)
  const shown = verbose ? result.files : result.files.slice(0, FILES_SHOWN)
  console.log('  by file:')
  for (const [file, count] of shown) console.log(`  ${String(count).padStart(COUNT_WIDTH)}  ${file}`)
  if (shown.length < result.files.length) console.log(`  (${result.files.length - shown.length} more files)`)
}

function standards(verbose) {
  const result = eslint('eslint.config.js', ['dev/acceptance', MOD])
  printTally('standards (eslint.config.js over dev/acceptance and the mod)', result, verbose)
  return result.problems
}

function complexity(verbose) {
  const result = eslint('eslint.complexity.config.js', [MOD])
  printTally('complexity and size (eslint.complexity.config.js over the mod hooks)', result, verbose)
  return result.problems
}

function adapter(verbose) {
  const result = eslint('eslint.adapter.config.js', [MOD])
  printTally('adapter (eslint.adapter.config.js over the mod hooks)', result, verbose)
  return result.problems
}

function types() {
  const run = spawnSync(
    process.execPath,
    [TSC, '--noEmit', '-p', path.join(PACKAGE, 'tsconfig.json'), '--pretty', 'false'],
    {
      cwd: REPO,
      encoding: 'utf8',
      maxBuffer: MAX_BUFFER,
    },
  )
  const errors = run.stdout.split('\n').filter((line) => /error TS\d+/.test(line))
  console.log(`types (tsc --noEmit over the mod's test files): ${errors.length} errors`)
  for (const line of errors) console.log(`  ${line}`)
  return errors.length
}

// lint runs the named part, or all of them, and returns the number of problems found.
export function lint(part, { verbose = false } = {}) {
  const parts = part === '' || part === undefined ? PARTS : [part]
  if (!parts.every((name) => PARTS.includes(name))) throw new Error(`unknown lint part "${part}": ${PARTS.join(', ')}`)
  const runs = {
    standards: () => standards(verbose),
    complexity: () => complexity(verbose),
    adapter: () => adapter(verbose),
    types,
  }
  return parts.reduce((sum, name) => sum + runs[name](), 0)
}
