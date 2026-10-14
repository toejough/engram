// Self-tests of the refactor gate for spec conformance (793e8l): the tool is tested on fixtures; whether
// the mod is within the gate is what the gate itself says, and it is red on purpose.
import assert from 'node:assert/strict'
import test from 'node:test'

import {
  check,
  checkRealizations,
  matchesName,
  parseSpec,
  realizations,
  references,
  statedTestCount,
} from '../gates/spec.js'

test('a cited test name matches the start of a test name, with … standing for any text', () => {
  assert.equal(
    matchesName(
      'a relevant note rides the prompt context',
      'prompt delivery > a relevant note rides the prompt context with the apply instruction',
    ),
    true,
  )
  assert.equal(matchesName('rides the prompt', 'prompt delivery > a relevant note rides the prompt context'), false)
  assert.equal(
    matchesName(
      'a reply is read strictly: … a covered…',
      'x > a reply is read strictly: prose around, a covered mark that is no note',
    ),
    true,
  )
  assert.equal(
    matchesName('a reply is read strictly: … a covered…', 'x > a reply is read strictly: prose around, an unknown key'),
    false,
  )
})

test('the contract tests a Verified by line names are listed, Go test suffixes expanded', () => {
  const refs = references(
    [
      '`delivery.test.ts` "a relevant note…"',
      '`internal/update/mods_test.go`: Test793e8e1a, 1b and 1f (Pi absent)',
      '`internal/cli/update_mods_test.go`: Test793e8e3a',
    ].join('; '),
  )
  assert.deepEqual(refs, [
    { kind: 'mod', file: 'delivery.test.ts', name: 'a relevant note…' },
    { kind: 'go', file: 'internal/update/mods_test.go', name: 'Test793e8e1a' },
    { kind: 'go', file: 'internal/update/mods_test.go', name: 'Test793e8e1b' },
    { kind: 'go', file: 'internal/update/mods_test.go', name: 'Test793e8e1f' },
    { kind: 'go', file: 'internal/cli/update_mods_test.go', name: 'Test793e8e3a' },
  ])
})

const SPEC = [
  '## 793d21 Intro',
  'Requirement: 793d21x1 outside the range',
  '',
  '## 793d22 Mod',
  'Requirement: 793d22b1 The mod MUST.',
  'Supports: [793a1a1a One](x), [793a1a1b Two](x)',
  'Verified by: `a.test.ts` "does a thing…"',
  'Requirement: 793d22b2 The mod MUST also.',
  'Supports: [793a1a1a One](x)',
].join('\n')

test('the requirements of the component sections are read with their supports and verification', () => {
  const requirements = parseSpec(SPEC)
  assert.deepEqual(
    requirements.map((requirement) => requirement.id),
    ['793d22b1', '793d22b2'],
  )
  assert.deepEqual(requirements[0].supports, ['793a1a1a', '793a1a1b'])
  assert.equal(requirements[1].verifiedBy, null)
})

test('the conformance check names a missing test, line, unknown scenario and unsupported scenario', () => {
  const requirements = parseSpec(SPEC)
  const tests = [{ file: 'a.test.ts', name: 'g > does a thing today', ok: true }]
  const problems = check({ requirements, productIds: ['793a1a1a', '793a1a1c'], tests, goFiles: {} })
  assert.deepEqual(problems, [
    '793d22b1: supports 793a1a1b, which is no product scenario',
    '793d22b2: no Verified by line',
    '793a1a1c: no requirement supports this product scenario',
  ])
  const failing = check({
    requirements: [requirements[0]],
    productIds: ['793a1a1a', '793a1a1b'],
    tests: [{ ...tests[0], ok: false }],
    goFiles: {},
  })
  assert.deepEqual(failing, ['793d22b1: a.test.ts "g > does a thing today" fails'])
  const absent = check({
    requirements: [requirements[0]],
    productIds: ['793a1a1a', '793a1a1b'],
    tests: [],
    goFiles: {},
  })
  assert.deepEqual(absent, ['793d22b1: a.test.ts has no test named "does a thing…"'])
})

test('a Realization line names files and names that must exist in the code', () => {
  const text = [
    '### 793d24a Orientation',
    'Ownership: x. Realization: `lookup`, `deliver` of `hooks/register.js`, `hooks/moment.js`.' +
      ' Next sentence `ignored`.',
  ].join('\n')
  const list = realizations(text)
  assert.deepEqual(list, [
    { section: '793d24a', files: ['hooks/register.js', 'hooks/moment.js'], names: ['lookup', 'deliver'] },
  ])
  const files = {
    'agent-instructions/mods/engram-memory/hooks/register.js': 'async function lookup() {}',
    'agent-instructions/mods/engram-memory/hooks/moment.js': 'export const x = 1',
  }
  const read = (path) => files[path] ?? null
  assert.deepEqual(checkRealizations(list, { read, hooks: [] }), [
    '793d24a: Realization names deliver, which no file it names declares',
  ])
  assert.deepEqual(checkRealizations(list, { read: () => null, hooks: [] }).slice(0, 2), [
    '793d24a: Realization names hooks/register.js, which does not exist',
    '793d24a: Realization names hooks/moment.js, which does not exist',
  ])
})

test('the test count the reading guide states is read', () => {
  assert.equal(statedTestCount('a file under tests/ (run by `claude plugin test`; 142 tests) with'), 142)
  assert.equal(statedTestCount('no count here'), null)
})
