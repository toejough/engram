import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { Scenario } from '../lib/world.js'

test('the agent of a scenario cannot echo the environment: only the words of the scenarios are allowed to echo', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'world-test-'))
  const scenario = new Scenario({ dir })
  try {
    const allow = scenario.baselineSettings.permissions.allow
    const echoes = allow.filter((rule) => rule.startsWith('Bash(echo'))
    assert.ok(echoes.length > 0)
    for (const rule of echoes) assert.ok(!/[*$`]/.test(rule), `${rule} lets the agent echo more than a fixed text`)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('a World configures no statusLine: the suite does not depend on a user\'s status line (793b443)', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'world-test-'))
  const scenario = new Scenario({ dir })
  try {
    const settings = JSON.parse(fs.readFileSync(path.join(scenario.home, '.claude', 'settings.json'), 'utf8'))
    assert.equal(settings.statusLine, undefined)
    assert.deepEqual(settings, scenario.baselineSettings)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})
