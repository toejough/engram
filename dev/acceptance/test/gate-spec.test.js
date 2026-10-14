// Self-tests of the refactor gate for spec conformance (793e8l): the tool is tested on fixtures; whether
// the mod is within the gate is what the gate itself says, and it is red on purpose.
import assert from 'node:assert/strict'
import test from 'node:test'

import {
  check,
  checkRealizations,
  checkSecurity,
  matchesName,
  parseSpec,
  realizations,
  references,
  securityRows,
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

test('the contract tests a Verified by line names are listed, a Go test by its exact function name', () => {
  const refs = references(
    [
      '`delivery.test.ts` "a relevant note…"',
      '`internal/update/mods_test.go`: Test793d31b1_A, Test793d31b1_B (Pi absent)',
      '`internal/cli/update_mods_test.go`: Test793d34b1_C',
    ].join('; '),
  )
  assert.deepEqual(refs, [
    { kind: 'mod', file: 'delivery.test.ts', name: 'a relevant note…' },
    { kind: 'go', file: 'internal/update/mods_test.go', name: 'Test793d31b1_A' },
    { kind: 'go', file: 'internal/update/mods_test.go', name: 'Test793d31b1_B' },
    { kind: 'go', file: 'internal/cli/update_mods_test.go', name: 'Test793d34b1_C' },
  ])
})

test('a Go test cited by the start of its name is not found: the name must be the function', () => {
  const requirement = {
    id: '793d22b1',
    supports: ['793a1a1a'],
    verifiedBy: '`internal/a_test.go`: Test793d22b1',
    security: null,
  }
  const goFiles = { 'internal/a_test.go': 'func Test793d22b1_Longer(t *testing.T) {\n}\n' }
  assert.deepEqual(check({ requirements: [requirement], productIds: ['793a1a1a'], tests: [], goFiles }), [
    '793d22b1: internal/a_test.go has no test Test793d22b1',
  ])
  const exact = { ...requirement, verifiedBy: '`internal/a_test.go`: Test793d22b1_Longer' }
  assert.deepEqual(check({ requirements: [exact], productIds: ['793a1a1a'], tests: [], goFiles }), [])
})

const SPEC = [
  '## 793d21 Intro',
  'Requirement: 793d21x1 in a section below 22.',
  '',
  '## 793d22 Mod',
  'Requirement: 793d22b1 The mod MUST.',
  'Supports: [793a1a1a One](x), [793a1a1b Two](x)',
  'Verified by: `a.test.ts` "does a thing…"',
  'Requirement: 793d22b2 The mod MUST also.',
  'Supports: [793a1a1a One](x)',
].join('\n')

test('every requirement is read, in any section, with its supports and verification', () => {
  const requirements = parseSpec(SPEC)
  assert.deepEqual(
    requirements.map((requirement) => requirement.id),
    ['793d21x1', '793d22b1', '793d22b2'],
  )
  requirements.shift()
  assert.deepEqual(requirements[0].supports, ['793a1a1a', '793a1a1b'])
  assert.equal(requirements[1].verifiedBy, null)
})

test('the conformance check names a missing test, line, unknown scenario and unsupported scenario', () => {
  const requirements = parseSpec(SPEC).slice(1)
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

const MAP = [
  '| Mitigation | Requirements |',
  '| --- | --- |',
  '| Data fenced | [793d25b2 Data and replies](#793d25b2-data-and-replies), [793d26b1 Shapes](#793d26b1-shapes) |',
  '| Nothing named | none |',
].join('\n')

test('the security map names requirements that exist and carry a Security line', () => {
  const rows = securityRows(`text\n\n${MAP}\n\nafter`)
  assert.deepEqual(rows, [
    { mitigation: 'Data fenced', requirements: ['793d25b2', '793d26b1'] },
    { mitigation: 'Nothing named', requirements: [] },
  ])
  const requirements = [{ id: '793d25b2', security: null }]
  assert.deepEqual(checkSecurity(rows, { requirements }), [
    'security map: 793d25b2 has no Security line',
    'security map: 793d26b1 is no requirement',
    'security map: "Nothing named" names no requirement',
  ])
})

test('a Security line links requirements that exist', () => {
  const requirements = [
    { id: '793d25b2', security: 'risk (1): [793d26b1 Credential shapes](#793d26b1-credential-shapes)' },
  ]
  assert.deepEqual(checkSecurity([], { requirements }), [
    '793d25b2: its Security line links 793d26b1, which is no requirement',
  ])
})

test('a requirement ends at the next heading: a later section does not lend it its lines', () => {
  const text = [
    'Requirement: 793d22b1 The mod MUST.',
    'Supports: [793a1a1a One](x)',
    '',
    '### 793d22c Orientation',
    'Verified by: `a.test.ts` "belongs to nothing"',
  ].join('\n')
  assert.equal(parseSpec(text)[0].verifiedBy, null)
})
