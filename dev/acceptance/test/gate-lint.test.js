// Self-tests of the refactor gate for JS standards and complexity (793e8l): the tool is tested on fixtures; whether
// the mod is within the gate is what the gate itself says, and it is red on purpose.
import assert from 'node:assert/strict'
import test from 'node:test'

import path from 'node:path'

import { ESLint } from 'eslint'

import { REPO, tally, tscProblems } from '../gates/lint.js'

test('the problems of an eslint report are counted by rule and by file, the largest first', () => {
  const report = [
    { filePath: '/repo/a.js', messages: [{ ruleId: 'x' }, { ruleId: 'y' }, { ruleId: 'x' }] },
    { filePath: '/repo/b.js', messages: [{ ruleId: 'x' }, { ruleId: null }] },
    { filePath: '/repo/c.js', messages: [] },
  ]
  const counted = tally(report)
  assert.equal(counted.problems, 5)
  assert.deepEqual(counted.rules[0], ['x', 3])
  assert.deepEqual(counted.rules.map(([rule]) => rule).sort(), ['parse error', 'x', 'y'])
  assert.equal(counted.files.length, 2)
})

const PACKAGE = path.resolve(import.meta.dirname, '..')
const DISABLED = '// eslint-disable-next-line no-var\nvar thing = 1\nexport { thing }\n'

// A file cannot switch a rule off for itself: the comment is not honoured, and the rule still reports.
for (const [config, file, rule] of [
  ['eslint.config.js', 'dev/acceptance/zz-fixture.js', 'no-var'],
  ['eslint.complexity.config.js', 'agent-instructions/mods/engram-memory/hooks/zz-fixture.js', 'max-statements'],
  ['eslint.adapter.config.js', 'agent-instructions/mods/engram-memory/hooks/zz-fixture.js', 'max-params'],
]) {
  test(`an inline directive does not silence a rule of ${config}`, async () => {
    const eslint = new ESLint({ cwd: REPO, overrideConfigFile: path.join(PACKAGE, config) })
    const body = `export function f(a, b, c, d, e, f) {\n${'  a()\n'.repeat(60)}}\n`
    const directive = rule === 'no-var' ? DISABLED : `// eslint-disable-next-line ${rule}\n${body}`
    const [result] = await eslint.lintText(directive, { filePath: path.join(REPO, file) })
    assert.ok(
      result.messages.some((message) => message.ruleId === rule),
      JSON.stringify(result.messages),
    )
  })
}

test('tsc is red by its exit status as well as by the errors it prints', () => {
  assert.deepEqual(tscProblems({ status: 0, stdout: '' }), [])
  assert.deepEqual(tscProblems({ status: 2, stdout: 'a.ts(1,1): error TS2304: x\nnoise\n' }), [
    'a.ts(1,1): error TS2304: x',
  ])
  assert.equal(tscProblems({ status: 1, stdout: '', stderr: 'error: cannot read tsconfig' }).length, 1)
  assert.equal(tscProblems({ status: null, signal: 'SIGKILL', stdout: '' }).length, 1)
})
