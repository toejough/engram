import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { extractScenarios, selectScenarios, writeFeatures, PRODUCT_SPEC, TECHNICAL_SPEC } from '../lib/extract.js'

const product = extractScenarios(PRODUCT_SPEC)
const h1 = extractScenarios(TECHNICAL_SPEC, { idPrefix: '793d17' })

test('the product spec yields exactly its 43 scenario IDs, once each', () => {
  const ids = product.map((s) => s.id)
  assert.equal(ids.length, 43)
  assert.equal(new Set(ids).size, 43)
  assert.ok(ids.every((id) => /^793a\d+[a-z][a-z0-9]*$/.test(id)), ids.join(' '))
})

test('the H1 section of the technical spec yields exactly its 11 scenario IDs', () => {
  assert.deepEqual(
    h1.map((s) => s.id),
    ['793d17a1a', '793d17a1b', '793d17a1c', '793d17b1a', '793d17c1a', '793d17c1b', '793d17c1c', '793d17d1a', '793d17e1a', '793d17e1b', '793d17f1a'],
  )
})

test('an outline is one scenario and reports its example count', () => {
  const byId = Object.fromEntries(product.map((s) => [s.id, s]))
  assert.equal(byId['793a2n1h'].outline, true)
  assert.equal(byId['793a2n1h'].examples, 4)
  assert.equal(byId['793a3a1j'].examples, 3)
  assert.equal(byId['793a3a1a'].examples, 3)
  assert.equal(byId['793a3a1k'].examples, 2)
  assert.equal(byId['793a5k1e'].outline, false)
  assert.equal(product.reduce((n, s) => n + (s.outline ? s.examples : 1), 0), 51)
})

test('the ID filter selects by prefix and by exact ID, in spec order', () => {
  assert.deepEqual(selectScenarios(product, '793a5k1e').map((s) => s.id), ['793a5k1e'])
  assert.deepEqual(selectScenarios(product, '793a5k').map((s) => s.id), ['793a5k1c', '793a5k1d', '793a5k1e'])
  assert.equal(selectScenarios(product, '').length, 43)
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
