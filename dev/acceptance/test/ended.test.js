// The fixtures of the ended sessions (793e8g).
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { piDirName, slugOf, turnsOf, writeEnded } from '../lib/ended.js'

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
    assert.deepEqual(rows.filter((row) => row.type === 'message').map((row) => row.message.role).filter((role) => role !== 'toolResult'), ['user', 'assistant', 'user', 'assistant'])
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

// The shape of a real Pi session file (version 3), read from one file under ~/.pi/agent/sessions on 2026-10-13: structure
// only (793b325). The fixture must have it, so that the backlog reader is shown against what Pi writes.
test('a Pi session directory is named as Pi names it: two dashes, the working directory without its first slash with dashes for slashes, two dashes', () => {
  assert.equal(piDirName('/Users/joe/repos/personal/pi-skills'), '--Users-joe-repos-personal-pi-skills--')
  assert.equal(piDirName('/private/var/x'), '--private-var-x--')
})

test('a Pi session file has the rows Pi writes: session, model change, thinking level, then messages with a numeric timestamp, thinking blocks, tool calls and tool results', () => {
  world((scenario) => {
    const ended = writeEnded(scenario, { harness: 'Pi', turns: turnsOf(INPUT, 'no, use targ') })
    assert.equal(path.basename(path.dirname(ended.file)), piDirName(fs.realpathSync(scenario.workspace)))
    assert.match(path.basename(ended.file), /^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}-\d{3}Z_[0-9a-f-]{36}\.jsonl$/)
    const rows = fs.readFileSync(ended.file, 'utf8').trim().split('\n').map((line) => JSON.parse(line))
    assert.deepEqual(rows.slice(0, 3).map((row) => row.type), ['session', 'model_change', 'thinking_level_change'])
    assert.equal(rows[0].version, 3)
    assert.equal(rows[0].parentId, undefined)
    assert.equal(rows[1].parentId, null)
    for (const row of rows.slice(1)) assert.equal(typeof row.id, 'string')
    const messages = rows.filter((row) => row.type === 'message').map((row) => row.message)
    assert.ok(messages.every((message) => typeof message.timestamp === 'number'))
    const roles = messages.map((message) => message.role)
    assert.deepEqual([...new Set(roles)].sort(), ['assistant', 'toolResult', 'user'])
    const assistants = messages.filter((message) => message.role === 'assistant')
    for (const message of assistants) {
      assert.deepEqual(Object.keys(message.usage).sort(), ['cacheRead', 'cacheWrite', 'cost', 'input', 'output', 'reasoning', 'totalTokens'])
      assert.ok(['stop', 'toolUse'].includes(message.stopReason))
      assert.equal(message.content[0].type, 'thinking')
    }
    const call = assistants.flatMap((message) => message.content).find((block) => block.type === 'toolCall')
    assert.deepEqual(Object.keys(call).sort(), ['arguments', 'id', 'name', 'type'])
    const result = messages.find((message) => message.role === 'toolResult')
    assert.equal(result.toolCallId, call.id)
    assert.equal(result.isError, false)
    // the roles the lesson turns carry are unchanged: only the user's and the agent's text are the conversation
    const said = messages.filter((message) => message.role !== 'toolResult').flatMap((message) => message.content.filter((block) => block.type === 'text').map((block) => [message.role, block.text]))
    assert.deepEqual(said, turnsOf(INPUT, 'no, use targ'))
  })
})
