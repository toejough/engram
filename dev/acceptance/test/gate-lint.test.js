// Self-tests of the refactor gate for JS standards and complexity (793e8l): the tool is tested on fixtures; whether
// the mod is within the gate is what the gate itself says, and it is red on purpose.
import assert from 'node:assert/strict'
import test from 'node:test'

import { tally } from '../gates/lint.js'

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
