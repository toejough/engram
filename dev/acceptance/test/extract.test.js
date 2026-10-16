import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import {
  changeDirectory,
  extractScenarios,
  productScenarios,
  productSpecFiles,
  selectScenarios,
  specRoot,
  technicalSpecFiles,
  technicalText,
  writeFeatures,
} from '../lib/extract.js'

const product = productScenarios()
const hypothesisOne = extractScenarios(technicalSpecFiles(), { idPrefix: '793d17' })
const hypothesisTwo = extractScenarios(technicalSpecFiles(), { idPrefix: '793d18' })

test('the product spec yields exactly its 50 scenario IDs, once each', () => {
  const ids = product.map((scenario) => scenario.id)
  assert.equal(ids.length, 50)
  assert.equal(new Set(ids).size, 50)
  assert.ok(
    ids.every((id) => /^793a\d+[a-z][a-z0-9]*$/.test(id)),
    ids.join(' '),
  )
})

test(
  'the hypothesis sections of the technical spec carry no scenarios: 793b355 retired ' + 'the experiment scenarios',
  () => {
    assert.deepEqual(hypothesisOne, [])
    assert.deepEqual(hypothesisTwo, [])
  },
)

test('an outline is one scenario and reports its example count', () => {
  const byId = Object.fromEntries(product.map((scenario) => [scenario.id, scenario]))
  assert.equal(byId['793a2n1h'].outline, true)
  assert.equal(byId['793a2n1h'].examples, 4)
  assert.equal(byId['793a3a1j'].examples, 3)
  assert.equal(byId['793a3a1a'].examples, 3)
  assert.equal(byId['793a3a1k'].examples, 2)
  assert.equal(byId['793a5k1e'].outline, false)
  assert.equal(
    product.reduce((total, scenario) => total + (scenario.outline ? scenario.examples : 1), 0),
    58,
  )
})

test('the ID filter selects by prefix and by exact ID, in spec order', () => {
  assert.deepEqual(
    selectScenarios(product, '793a5k1e').map((scenario) => scenario.id),
    ['793a5k1e'],
  )
  assert.deepEqual(
    selectScenarios(product, '793a5k').map((scenario) => scenario.id),
    ['793a5k1c', '793a5k1d', '793a5k1e'],
  )
  assert.equal(selectScenarios(product, '').length, 50)
  assert.deepEqual(selectScenarios(product, '793a9'), [])
})

test('written features carry one Feature line per heading and the scenario text as written', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'extract-test-'))
  try {
    const files = writeFeatures(selectScenarios(product, '793a5k'), dir)
    assert.equal(files.length, 1)
    const text = fs.readFileSync(files[0], 'utf8')
    assert.match(text, /^Feature: 793a5k1 UAT Scenarios\n/)
    assert.match(text, /Scenario: 793a5k1e An intentional engram query by the agent is allowed\n/)
    assert.equal((text.match(/^Feature:/gm) ?? []).length, 1)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('a gherkin block under a heading that is not a scenario heading is not extracted', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'extract-heading-'))
  try {
    const file = path.join(dir, 'spec.md')
    fs.writeFileSync(
      file,
      [
        '#### 793z1 UAT Scenarios',
        '',
        '```gherkin',
        'Scenario: 793z1a Counted',
        '  Given a world',
        '```',
        '',
        '### 793z2 Some other section',
        '',
        '```gherkin',
        'Scenario: 793z2a Not a scenario of the spec',
        '  Given a world',
        '```',
        '',
      ].join('\n'),
    )
    assert.deepEqual(
      extractScenarios(file).map((scenario) => scenario.id),
      ['793z1a'],
    )
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

// The gates read the specs where the change leaves them: the change directory now, the canonical files after promotion.
function promotedRoot() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'promoted-root-'))
  const productDir = path.join(root, 'docs', 'specs', 'product')
  const technical = path.join(root, 'docs', 'specs', 'technical')
  fs.mkdirSync(productDir, { recursive: true })
  fs.mkdirSync(technical, { recursive: true })
  const scenario = (id) =>
    ['### x UAT Scenarios', '', '```gherkin', `Scenario: ${id} A title`, '  Given a thing', '```', ''].join('\n')
  fs.writeFileSync(path.join(productDir, 'automatic-memory.md'), `# Automatic Memory\n\n${scenario('793a1a1a')}`)
  fs.writeFileSync(path.join(productDir, 'recall.md'), `# Recall\n\n${scenario('792a1a1a')}`)
  fs.writeFileSync(path.join(technical, 'judgement-relay.md'), '# Judgement Relay\n\n## 793d25 Judgement Relay\n')
  fs.writeFileSync(path.join(technical, 'note-amendment.md'), '# Note Amendment\n\n## 792b14a Orientation\n')
  return root
}

test('after promotion the product scenarios of the change are read from the canonical files, and only those', () => {
  const root = promotedRoot()
  try {
    assert.deepEqual(
      productScenarios(root).map((scenario) => scenario.id),
      ['793a1a1a'],
    )
    assert.equal(changeDirectory(root), null)
  } finally {
    fs.rmSync(root, { recursive: true, force: true })
  }
})

test('after promotion the technical text is that of the canonical files that hold sections of the change', () => {
  const root = promotedRoot()
  try {
    assert.deepEqual(
      technicalSpecFiles(root).map((file) => path.basename(file)),
      ['judgement-relay.md'],
    )
    assert.match(technicalText(root), /## 793d25 Judgement Relay/)
    assert.equal(specRoot({ ENGRAM_SPEC_ROOT: root }), root)
  } finally {
    fs.rmSync(root, { recursive: true, force: true })
  }
})

test('while the change exists, its specs are read as well as the canonical files', () => {
  assert.ok(changeDirectory() !== null)
  assert.ok(productSpecFiles().some((file) => file.endsWith('793a_product-spec.md')))
  assert.ok(technicalSpecFiles().some((file) => file.endsWith('793d_technical-spec.md')))
})

test('a scenario that the change and a canonical file both hold is read once', () => {
  const root = promotedRoot()
  try {
    const change = path.join(root, 'docs', 'changes', '793_parallel-memory-agent')
    fs.mkdirSync(change, { recursive: true })
    fs.copyFileSync(
      path.join(root, 'docs', 'specs', 'product', 'automatic-memory.md'),
      path.join(change, '793a_product-spec.md'),
    )
    assert.deepEqual(
      productScenarios(root).map((scenario) => scenario.id),
      ['793a1a1a'],
    )
  } finally {
    fs.rmSync(root, { recursive: true, force: true })
  }
})
