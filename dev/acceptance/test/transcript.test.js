import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import { readRows, userPrompts, toolCalls, memoryCalls, countContent, finalReply } from '../lib/transcript.js'

const FIXTURE = path.join(path.dirname(fileURLToPath(import.meta.url)), 'fixtures', 'engram-query-turn.jsonl')
const SENTINEL = 'SENTINEL-N-71c03d'

test('rows are parsed from the JSONL in order', () => {
  const rows = readRows(FIXTURE)
  assert.equal(rows.length, 5)
  assert.deepEqual(rows.map((row) => row.type), ['user', 'attachment', 'assistant', 'user', 'assistant'])
})

test('a malformed last line is skipped, not an error', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'transcript-test-'))
  try {
    const file = path.join(dir, 't.jsonl')
    fs.writeFileSync(file, '{"type":"user"}\n{"type":"assis')
    assert.deepEqual(readRows(file), [{ type: 'user' }])
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('user prompts are the typed texts, not tool results', () => {
  const prompts = userPrompts(readRows(FIXTURE))
  assert.equal(prompts.length, 1)
  assert.match(prompts[0], /^In the middle of this task, run `engram query/)
})

test('a tool call is paired with its result', () => {
  const calls = toolCalls(readRows(FIXTURE))
  assert.equal(calls.length, 1)
  assert.equal(calls[0].name, 'Bash')
  assert.match(calls[0].input.command, /^engram query --phrase/)
  assert.equal(calls[0].result.isError, false)
  assert.ok(calls[0].result.text.includes(SENTINEL))
})

test('memory calls are the runs of an engram command and of the memory skills', () => {
  const rows = readRows(FIXTURE)
  assert.equal(memoryCalls(rows).length, 1)
  const other = [{ type: 'assistant', message: { content: [{ type: 'tool_use', id: 't1', name: 'Bash', input: { command: 'ls -la' } }] } }]
  assert.equal(memoryCalls(other).length, 0)
  const skill = [{ type: 'assistant', message: { content: [{ type: 'tool_use', id: 't2', name: 'Skill', input: { skill: 'recall' } }] } }]
  assert.equal(memoryCalls(skill).length, 1)
})

test('content occurrences count the rows that carry a text, whatever their kind', () => {
  const rows = readRows(FIXTURE)
  assert.equal(countContent(rows, SENTINEL), 2)
  assert.equal(countContent(rows, 'no-such-text'), 0)
})

test('the final reply is the last assistant text', () => {
  assert.match(finalReply(readRows(FIXTURE)), /releases are tagged from main/)
})
