// The readers and fault shims of node 793e8d (housekeeping, purview, failure isolation).
import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { installShim, shimCalls } from '../lib/shim.js'
import { preconditionFailure } from '../lib/turn.js'
import { MARKER, SKIPPED, rowsMentioning, statusTexts, typedRows } from '../lib/transcript.js'

function temp(fn) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'housekeeping-test-'))
  try {
    return fn(dir)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
}

function fakeEngram(dir) {
  const file = path.join(dir, 'real-engram')
  fs.writeFileSync(file, '#!/bin/sh\necho "real $*"\n', { mode: 0o755 })
  return file
}

test('the error shim exits with a message and logs the call', () => {
  temp((dir) => {
    const shim = installShim({ dir, realEngram: fakeEngram(dir), mode: 'error' })
    assert.throws(() => execFileSync(path.join(shim, 'engram'), ['query', 'x'], { stdio: 'pipe' }), (error) => error.status === 70 && /injected failure/.test(String(error.stderr)))
    assert.equal(shimCalls(shim).length, 1)
  })
})

test('the delay shim passes a query and the mod\'s jobs through at once and waits before any other command', () => {
  temp((dir) => {
    const shim = installShim({ dir, realEngram: fakeEngram(dir), mode: 'delay', seconds: 1 })
    // the evidence is the shim's own log, not the clock: a loaded machine can make a passed-through query slow
    assert.equal(execFileSync(path.join(shim, 'engram'), ['query', 'x'], { encoding: 'utf8' }), 'real query x\n')
    assert.equal(execFileSync(path.join(shim, 'engram'), ['auto', 'run'], { encoding: 'utf8' }), 'real auto run\n')
    assert.equal(shimCalls(shim).filter((call) => call.what.includes('slept-start')).length, 0)
    const ingest = Date.now()
    assert.equal(execFileSync(path.join(shim, 'engram'), ['ingest', '--auto'], { encoding: 'utf8' }), 'real ingest --auto\n')
    assert.ok(Date.now() - ingest >= 900)
    assert.equal(shimCalls(shim).filter((call) => call.what.includes('slept-start')).length, 1)
  })
})

test('the hang shim lets a lookup run on and passes other commands through', () => {
  temp((dir) => {
    const shim = installShim({ dir, realEngram: fakeEngram(dir), mode: 'hang' })
    assert.equal(execFileSync(path.join(shim, 'engram'), ['learn', 'x'], { encoding: 'utf8' }), 'real learn x\n')
    assert.throws(() => execFileSync(path.join(shim, 'engram'), ['query', 'x'], { timeout: 300, stdio: 'pipe' }), (error) => error.code === 'ETIMEDOUT')
  })
})

test('an unknown shim mode is refused, and no shim has no calls', () => {
  temp((dir) => {
    assert.throws(() => installShim({ dir, realEngram: 'x', mode: 'slow' }), /no shim mode/)
  })
  assert.deepEqual(shimCalls(null), [])
})

test('status texts are the ui_status system messages, in order, and nothing else', () => {
  const messages = [
    { type: 'system', subtype: 'init' },
    { type: 'system', subtype: 'ui_status', plugin: 'engram-memory', text: 'one' },
    { type: 'assistant', text: 'one' },
    { type: 'system', subtype: 'ui_status', plugin: 'engram-memory', text: 'Automatic memory was skipped because of an error (indexing)' },
  ]
  assert.deepEqual(statusTexts(messages), ['one', 'Automatic memory was skipped because of an error (indexing)'])
  assert.equal(statusTexts(messages).filter((text) => SKIPPED.test(text)).length, 1)
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
  const rows = [{ type: 'assistant', message: { content: 'Automatic memory was skipped' } }, { type: 'user', message: { content: 'hi' } }]
  assert.equal(rowsMentioning(rows, SKIPPED).length, 1)
})

test('a script that needs a delivery is not observed in a turn without one', () => {
  const block = { type: 'attachment', attachment: { type: 'hook_additional_context', content: [`${MARKER} notes`] } }
  assert.notEqual(preconditionFailure([], { delivered: true }), null)
  assert.equal(preconditionFailure([block], { delivered: true }), null)
  assert.equal(preconditionFailure([], {}), null)
})
