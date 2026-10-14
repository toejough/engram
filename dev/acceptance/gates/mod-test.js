// The mod's contract tests (793e8o, 793e8p): `claude plugin test` of the mod in a temporary home. The green tests are
// the `tests/*.test.ts` files. The red gates are `tests/*.red.ts` files, which `claude plugin test` does not run (it
// reads `*.test.ts` only): a gate that has not turned green yet stays under that name, and is staged here, as a
// `.test.ts` in a copy of the mod, so that the plain run of the mod never fails on a gate. When a gate is green its
// file is renamed to `.test.ts`.
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const REPO = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..', '..')
export const MOD = path.join(REPO, 'agent-instructions', 'mods', 'engram-memory')
const RED_SUFFIX = '.red.ts'
const TEST_SUFFIX = '.test.ts'

// sessionEnv is the environment of the run: nothing of the caller's but the program lookup, and a home of its own.
export function sessionEnv(home) {
  return {
    PATH: process.env.PATH ?? '',
    HOME: home,
    CLAUDE_CONFIG_DIR: path.join(home, '.claude'),
    TMPDIR: home,
    LANG: 'C.UTF-8',
  }
}

// stage copies the mod into dir with the files named `pick` of its tests, the red ones renamed to `.test.ts`.
function stage(mod, dir, { red, filter }) {
  for (const entry of ['.claude-plugin', 'hooks'])
    fs.cpSync(path.join(mod, entry), path.join(dir, entry), { recursive: true })
  fs.mkdirSync(path.join(dir, 'tests'))
  const suffix = red ? RED_SUFFIX : TEST_SUFFIX
  const entries = fs.readdirSync(path.join(mod, 'tests'))
  // the helpers of the tests (files that are neither tests nor gates) go along
  for (const name of entries.filter((entry) => !entry.endsWith(TEST_SUFFIX) && !entry.endsWith(RED_SUFFIX))) {
    fs.copyFileSync(path.join(mod, 'tests', name), path.join(dir, 'tests', name))
  }
  const picked = entries.filter((name) => name.endsWith(suffix) && name.startsWith(filter))
  for (const name of picked) {
    fs.copyFileSync(
      path.join(mod, 'tests', name),
      path.join(dir, 'tests', `${name.slice(0, -suffix.length)}${TEST_SUFFIX}`),
    )
  }
  return picked
}

// results reads the `(pass)` and `(fail)` lines of a run, under the `tests/<file>:` line that heads each file's lines:
// { pass: [...names], fail: [...names], tests: [{ file, name, ok }] }; a name is "describe > test".
export function results(output) {
  const tests = []
  let file = ''
  for (const line of output.split('\n')) {
    const head = /^(?:tests\/)?([^\s/]+\.test\.ts):$/.exec(line)
    if (head) file = head[1]
    const match = /^\((pass|fail)\) (.*?)(?: \[[\d.]+m?s\])?$/.exec(line)
    if (match) tests.push({ file, name: match[2], ok: match[1] === 'pass' })
  }
  return {
    pass: tests.filter((test) => test.ok).map((test) => test.name),
    fail: tests.filter((test) => !test.ok).map((test) => test.name),
    tests,
  }
}

// run stages and runs the mod's tests; which is 'green' (the test files) or 'red' (the gates); filter is a file name
// prefix. It returns the exit status of the run, the lines of output and the test results.
export function run({ which, filter = '' }) {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'mod-test-home-'))
  const dir = path.join(home, 'mod')
  try {
    fs.mkdirSync(dir)
    const picked = stage(MOD, dir, { red: which === 'red', filter })
    if (picked.length === 0) throw new Error(`no ${which} test file starts with "${filter}"`)
    const child = spawnSync('claude', ['plugin', 'test', dir], { env: sessionEnv(home), encoding: 'utf8' })
    if (child.error) throw new Error(`claude plugin test could not run: ${child.error.message}`)
    const output = `${child.stdout}${child.stderr}`
    return { status: child.status, output, files: picked, ...results(output) }
  } finally {
    fs.rmSync(home, { recursive: true, force: true })
  }
}
