// The fixtures of the ended sessions (793e8g).
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { slugOf, turnsOf, writeEnded } from '../lib/ended.js'

function world(fn) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ended-test-'))
  const home = path.join(dir, 'home')
  const workspace = path.join(home, 'work')
  fs.mkdirSync(workspace, { recursive: true })
  try {
    return fn({ dir, home, workspace, ended: [] })
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
}

const INPUT = { opening: [['user', 'Run the tests.'], ['assistant', 'Running go test.']], closing: 'Understood.' }

test('a session killed mid-turn ends with what the user said, one that ended normally with the last reply', () => {
  assert.deepEqual(turnsOf({ opening: INPUT.opening }, 'no').at(-1), ['user', 'no'])
  assert.deepEqual(turnsOf(INPUT, 'no').at(-1), ['assistant', 'Understood.'])
})

test('a Pi session is written in the Pi session directory, in the Pi row format, back-dated, with an identical copy', () => {
  world((scenario) => {
    const ended = writeEnded(scenario, { harness: 'Pi', turns: turnsOf(INPUT, 'no, use targ') })
    assert.ok(ended.file.includes(`${path.sep}.pi${path.sep}agent${path.sep}sessions${path.sep}`))
    const rows = fs.readFileSync(ended.file, 'utf8').trim().split('\n').map((line) => JSON.parse(line))
    assert.equal(rows[0].type, 'session')
    assert.deepEqual(rows.slice(1).map((row) => row.message.role), ['user', 'assistant', 'user', 'assistant'])
    assert.ok(Date.now() - fs.statSync(ended.file).mtimeMs > 3600 * 1000, 'the session is not quiet')
    assert.ok(Buffer.compare(fs.readFileSync(ended.file), fs.readFileSync(ended.copy)) === 0)
    assert.equal(scenario.ended.length, 1)
  })
})

test('a Claude Code session is written in the project directory of the workspace, one row per turn', () => {
  world((scenario) => {
    const ended = writeEnded(scenario, { harness: 'Claude Code', turns: turnsOf({ opening: INPUT.opening }, 'remember x') })
    assert.ok(ended.file.includes(`${path.sep}.claude${path.sep}projects${path.sep}${slugOf(fs.realpathSync(scenario.workspace))}${path.sep}`))
    const rows = fs.readFileSync(ended.file, 'utf8').trim().split('\n').map((line) => JSON.parse(line))
    assert.deepEqual(rows.map((row) => row.type), ['user', 'assistant', 'user'])
    assert.equal(rows.at(-1).message.content, 'remember x')
  })
})
