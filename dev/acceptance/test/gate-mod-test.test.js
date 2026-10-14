// Self-tests of the refactor gate for the mod tests (793e8l): the tool is tested on fixtures; whether
// the mod is within the gate is what the gate itself says, and it is red on purpose.
import assert from 'node:assert/strict'
import test from 'node:test'

import { results, sessionEnv } from '../gates/mod-test.js'

test('the pass and fail lines of a plugin test run are read under the file that heads them', () => {
  const output = [
    '',
    'tests/a.test.ts:',
    '(pass) group > one [1.20ms]',
    '(fail) group > two [0.30ms]',
    '  boom',
    '',
    'tests/b.test.ts:',
    '(pass) three',
    ' 2 pass',
  ].join('\n')
  const read = results(output)
  assert.deepEqual(read.tests, [
    { file: 'a.test.ts', name: 'group > one', ok: true },
    { file: 'a.test.ts', name: 'group > two', ok: false },
    { file: 'b.test.ts', name: 'three', ok: true },
  ])
  assert.deepEqual(read.fail, ['group > two'])
})

test('a session environment holds a home of its own and nothing of the caller but the program lookup', () => {
  const environment = sessionEnv('/tmp/h')
  assert.equal(environment.HOME, '/tmp/h')
  assert.equal(environment.CLAUDE_CONFIG_DIR, '/tmp/h/.claude')
  assert.deepEqual(Object.keys(environment).sort(), ['CLAUDE_CONFIG_DIR', 'HOME', 'LANG', 'PATH', 'TMPDIR'])
})
