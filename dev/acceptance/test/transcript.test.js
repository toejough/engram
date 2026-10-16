import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import {
  ACTIVITY_HEADER,
  contextBlocks,
  finalReply,
  memoryCalls,
  noResponseBetween,
  readRows,
  resultIndex,
  splitActivity,
  toolCalls,
} from '../lib/transcript.js'

const FIXTURE = path.join(path.dirname(fileURLToPath(import.meta.url)), 'fixtures', 'engram-query-turn.jsonl')
const SENTINEL = 'SENTINEL-N-71c03d'

test('rows are parsed from the JSONL in order', () => {
  const rows = readRows(FIXTURE)
  assert.equal(rows.length, 5)
  assert.deepEqual(
    rows.map((row) => row.type),
    ['user', 'attachment', 'assistant', 'user', 'assistant'],
  )
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
  const other = [
    {
      type: 'assistant',
      message: { content: [{ type: 'tool_use', id: 't1', name: 'Bash', input: { command: 'ls -la' } }] },
    },
  ]
  assert.equal(memoryCalls(other).length, 0)
  const skill = [
    {
      type: 'assistant',
      message: { content: [{ type: 'tool_use', id: 't2', name: 'Skill', input: { skill: 'recall' } }] },
    },
  ]
  assert.equal(memoryCalls(skill).length, 1)
})

test(
  'the row of a tool result is found by the call, and what sits after it before any ' + 'response is read with it',
  () => {
    const rows = [
      {
        type: 'assistant',
        message: { content: [{ type: 'tool_use', id: 't1', name: 'Skill', input: { skill: 'route' } }] },
      },
      { type: 'user', message: { content: [{ type: 'tool_result', tool_use_id: 't1', content: 'ok' }] } },
      { type: 'user', message: { content: [{ type: 'text', text: 'Base directory for this skill' }] } },
      { type: 'attachment', attachment: { type: 'hook_additional_context', content: ['[engram memory]'] } },
      { type: 'assistant', message: { content: [{ type: 'text', text: 'ok' }] } },
      { type: 'attachment', attachment: { type: 'hook_additional_context', content: ['[engram memory]'] } },
    ]
    assert.equal(resultIndex(rows, 't1'), 1)
    assert.equal(resultIndex(rows, 'nope'), -1)
    assert.equal(noResponseBetween(rows, 1, 3), true)
    assert.equal(noResponseBetween(rows, 1, 5), false)
  },
)

test('the final reply is the last assistant text', () => {
  assert.match(finalReply(readRows(FIXTURE)), /releases are tagged from main/)
})

import { MARKER, agentCalls, deliveries, precedesResponse, retrievals, systemBlocks } from '../lib/transcript.js'

function block(text) {
  return { type: 'attachment', attachment: { type: 'hook_additional_context', content: [text] } }
}

const PROMPT = { type: 'user', message: { role: 'user', content: 'hi' } }
const REPLY = { type: 'assistant', message: { content: [{ type: 'text', text: 'ok' }] } }

test('system blocks are the hook context rows that open with the marker', () => {
  const rows = [PROMPT, block(`${MARKER} notes\n- n1 SENTINEL-A`), block('someone else'), REPLY]
  assert.deepEqual(
    systemBlocks(rows).map((entry) => entry.index),
    [1],
  )
  assert.equal(deliveries(rows, 'SENTINEL-A').length, 1)
  assert.equal(deliveries(rows, 'SENTINEL-B').length, 0)
})

test('a block precedes the response when no assistant row sits between its prompt and itself', () => {
  assert.equal(precedesResponse([PROMPT, block(`${MARKER} x`), REPLY], 1), true)
  assert.equal(precedesResponse([PROMPT, REPLY, block(`${MARKER} x`)], 2), false)
})

test('retrievals are the agent memory commands whose results show the text', () => {
  const rows = [
    {
      type: 'assistant',
      message: {
        content: [{ type: 'tool_use', id: 't1', name: 'Bash', input: { command: 'engram query --phrase x' } }],
      },
    },
    { type: 'user', message: { content: [{ type: 'tool_result', tool_use_id: 't1', content: 'SENTINEL-A' }] } },
    {
      type: 'assistant',
      message: { content: [{ type: 'tool_use', id: 't2', name: 'Bash', input: { command: 'echo SENTINEL-A' } }] },
    },
    { type: 'user', message: { content: [{ type: 'tool_result', tool_use_id: 't2', content: 'SENTINEL-A' }] } },
  ]
  assert.equal(retrievals(rows, 'SENTINEL-A').length, 1)
})

test('subagent dispatches are the Agent and Task tool uses', () => {
  const rows = [
    {
      type: 'assistant',
      message: {
        content: [
          { type: 'tool_use', id: 'a', name: 'Agent', input: { prompt: 'p' } },
          { type: 'tool_use', id: 'b', name: 'Bash', input: {} },
        ],
      },
    },
  ]
  assert.equal(agentCalls(rows).length, 1)
})

import { declarationIndex, dispatchIndexAfter, precedesFirstResponse } from '../lib/transcript.js'

const SAY = (text) => ({ type: 'assistant', message: { content: [{ type: 'text', text }] } })
const CALL = (id, name, input = {}) => ({
  type: 'assistant',
  message: { content: [{ type: 'tool_use', id, name, input }] },
})
const RESULT = (id) => ({
  type: 'user',
  message: { content: [{ type: 'tool_result', tool_use_id: id, content: 'ok' }] },
})

test('the declaration is the last assistant row that says anything; a tool call alone is not one', () => {
  const rows = [PROMPT, SAY('I will search.'), CALL('t1', 'Grep'), RESULT('t1'), SAY('The rename is done.')]
  assert.equal(declarationIndex(rows), 4)
  assert.equal(declarationIndex([PROMPT, CALL('t1', 'Grep'), RESULT('t1')]), -1)
})

test('a block precedes the first response only when no assistant row sits before it at all', () => {
  assert.equal(precedesFirstResponse([PROMPT, block(`${MARKER} x`), SAY('ok')], 1), true)
  assert.equal(precedesFirstResponse([PROMPT, CALL('t1', 'Grep'), RESULT('t1'), block(`${MARKER} x`)], 3), false)
})

test(
  'the dispatch after a call is the next Agent or Task use after its result, or the ' +
    'declaration when there is none',
  () => {
    const dispatched = [
      PROMPT,
      CALL('r', 'Skill'),
      RESULT('r'),
      block(`${MARKER} x`),
      CALL('d', 'Agent'),
      RESULT('d'),
      SAY('done'),
    ]
    assert.equal(dispatchIndexAfter(dispatched, 'r'), 4)
    assert.equal(dispatchIndexAfter([PROMPT, CALL('r', 'Skill'), RESULT('r'), SAY('done')], 'r'), 3)
    assert.equal(dispatchIndexAfter([PROMPT, CALL('r', 'Skill'), RESULT('r')], 'r'), Infinity)
  },
)

const ACTIVITY = 'Memory activity: 2 notes added'
const withActivity = (notes) => `${notes}\n${ACTIVITY_HEADER}\n- ${ACTIVITY}`
const contextRow = (text) => ({ type: 'attachment', attachment: { type: 'hook_additional_context', content: [text] } })

test('the System\'s lines of the previous turn are split from what a block delivers', () => {
  assert.deepEqual(splitActivity('[engram memory] notes\n- n.md (x): y'), {
    notes: '[engram memory] notes\n- n.md (x): y',
    activity: [],
  })
  assert.deepEqual(splitActivity(withActivity('[engram memory] notes\n- n.md (x): y')), {
    notes: '[engram memory] notes\n- n.md (x): y',
    activity: [ACTIVITY],
  })
})

test('a block of the System\'s lines alone delivers nothing, and notes are delivered without the lines', () => {
  const rows = [
    contextRow(`[engram memory] ${ACTIVITY_HEADER}\n- ${ACTIVITY}`),
    contextRow(`[engram memory] Delivered by the memory system, not typed by you.\n${ACTIVITY_HEADER}\n- ${ACTIVITY}`),
    contextRow(withActivity('[engram memory] notes\n- n.md (x): y')),
  ]
  assert.deepEqual(
    contextBlocks(rows).map((found) => [found.index, found.activityOnly, found.activity]),
    [
      [0, true, [ACTIVITY]],
      [1, true, [ACTIVITY]],
      [2, false, [ACTIVITY]],
    ],
  )
  assert.deepEqual(systemBlocks(rows), [{ index: 2, text: '[engram memory] notes\n- n.md (x): y' }])
})
