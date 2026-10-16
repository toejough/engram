// The readers and the fault seam of node 793e8d (housekeeping, purview, failure isolation).
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { faultCalls, faultEnv } from '../lib/fault.js'
import { preconditionFailure } from '../lib/turn.js'
import { MARKER, SKIPPED, rowsMentioning, shownTexts, typedRows } from '../lib/transcript.js'

function temp(body) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'housekeeping-test-'))
  try {
    return body(dir)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
}

test('the fault seam is set by the environment of the scenario: error, hang and a delay of N seconds', () => {
  assert.deepEqual(faultEnv({ dir: '/s', mode: 'error' }), {
    ENGRAM_AUTO_FAULT: 'error',
    ENGRAM_AUTO_FAULT_LOG: '/s/fault.log',
  })
  assert.equal(faultEnv({ dir: '/s', mode: 'hang' }).ENGRAM_AUTO_FAULT, 'hang')
  assert.equal(faultEnv({ dir: '/s', mode: 'delay', seconds: 20 }).ENGRAM_AUTO_FAULT, 'delay:20')
})

test('the lines the engine logged are read as seconds and what began, and no log has no calls', () => {
  temp((dir) => {
    const env = faultEnv({ dir, mode: 'delay', seconds: 1 })
    assert.deepEqual(faultCalls(env), [])
    fs.writeFileSync(env.ENGRAM_AUTO_FAULT_LOG, '100 slept-start ingest\n101 slept-start learn\n')
    assert.deepEqual(faultCalls(env), [
      { seconds: 100, what: 'slept-start ingest' },
      { seconds: 101, what: 'slept-start learn' },
    ])
  })
  assert.deepEqual(faultCalls({}), [])
})

test('an unknown fault mode is refused', () => {
  assert.throws(() => faultEnv({ dir: '/s', mode: 'slow' }), /no fault mode/)
})

test('shown texts are the ui_log, ui_status and ui_toast system messages, in order, and no debug line', () => {
  const messages = [
    { type: 'system', subtype: 'init' },
    { type: 'system', subtype: 'ui_log', plugin: 'engram-memory', text: 'one' },
    { type: 'assistant', text: 'two' },
    { type: 'system', subtype: 'ui_log', plugin: 'engram-memory', text: 'a debug line', to: 'debug' },
    { type: 'system', subtype: 'ui_status', plugin: 'engram-memory', text: 'three' },
    {
      type: 'system',
      subtype: 'ui_toast',
      plugin: 'engram-memory',
      text: 'Automatic memory skipped: indexing failed: x; nothing lost, retried next turn (details: y)',
    },
  ]
  assert.deepEqual(shownTexts(messages), [
    'one',
    'three',
    'Automatic memory skipped: indexing failed: x; nothing lost, retried next turn (details: y)',
  ])
  assert.equal(shownTexts(messages).filter((text) => SKIPPED.test(text)).length, 1)
})

test('typed rows are the user rows whose text is exactly the prompt, as a string or as joined text blocks', () => {
  const rows = [
    { type: 'user', message: { content: 'What is two plus two?' } },
    { type: 'user', message: { content: [{ type: 'text', text: 'What is two plus two?' }] } },
    { type: 'user', message: { content: 'What is two plus two? Also be brief.' } },
    { type: 'attachment', attachment: { content: ['What is two plus two?'] } },
  ]
  assert.equal(typedRows(rows, 'What is two plus two?').length, 2)
})

test('rows mentioning a pattern are found whatever their kind', () => {
  const rows = [
    { type: 'assistant', message: { content: 'Automatic memory skipped: x' } },
    { type: 'user', message: { content: 'hi' } },
  ]
  assert.equal(rowsMentioning(rows, SKIPPED).length, 1)
})

test('a script that needs a delivery is not observed in a turn without one', () => {
  const block = { type: 'attachment', attachment: { type: 'hook_additional_context', content: [`${MARKER} notes`] } }
  assert.notEqual(preconditionFailure([], { delivered: true }), null)
  assert.equal(preconditionFailure([block], { delivered: true }), null)
  assert.equal(preconditionFailure([], {}), null)
})
