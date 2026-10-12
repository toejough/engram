// Needs the installed dependencies: run by `targ acceptance --check`, not by check-full.
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import { runFeatures } from '../../lib/runner.js'

const HERE = path.dirname(fileURLToPath(import.meta.url))

const FEATURE = `Feature: red path

  Scenario: 999z1a Holds
    Given a world
    When something happens
    Then it holds

  Scenario: 999z1b A Then that fails
    Given a world
    When something happens
    Then it does not hold

  Scenario: 999z1c A harness bug
    Given a world
    When something happens
    Then the harness breaks

  Scenario: 999z1d A step not built
    Given a world
    Then it is not built

  Scenario: 999z1e A step nobody defined
    Given a world
    Then nothing defines this
`

test('a failing Then is Red with its ID; a harness bug, a pending and an undefined step are never Red', async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'redpath-test-'))
  try {
    const feature = path.join(dir, 'red.feature')
    fs.writeFileSync(feature, FEATURE)
    const { success, results } = await runFeatures({ features: [feature], support: [path.join(HERE, 'fixture', 'steps.js')], cwd: HERE })
    const byId = Object.fromEntries(results.map((entry) => [entry.id, entry.result]))
    assert.deepEqual(byId, {
      '999z1a': 'Green',
      '999z1b': 'Red',
      '999z1c': 'Harness error',
      '999z1d': 'Pending (not built)',
      '999z1e': 'Harness error (undefined step)',
    })
    assert.equal(success, false)
    assert.match(results.find((entry) => entry.id === '999z1b').message, /the System does not do the thing/)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})
