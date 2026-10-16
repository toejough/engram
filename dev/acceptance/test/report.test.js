// The report keeps the activity line of a scenario that did not pass (793e8p): a lookup the System cut shows its
// warning as a transcript line, which is what tells it from a judgement that missed.
import assert from 'node:assert/strict'
import test from 'node:test'

import { formatReport } from '../lib/report.js'
import { entryFor } from '../lib/results.js'

const CUT =
  'Automatic memory skipped: route lookup timed out: engram aborted; no notes delivered this time ' +
  '(details: ~/.local/share/engram/auto.log)'
const status = (text) => ({ type: 'system', subtype: 'ui_log', text })
const world = (sessions) => ({ id: '793a2n1j', title: 't', scenario: { sessions, usage: () => null } })

test('the entry of a scenario holds the lines its sessions showed, in order', () => {
  const sessions = [{ messages: [status('one'), { type: 'assistant' }] }, { messages: [status(CUT)] }]
  const entry = entryFor(world(sessions), { status: 'FAILED', exception: { type: 'AssertionError' }, message: 'x' })

  assert.deepEqual(entry.activity, ['one', CUT])
})

test('a scenario that did not pass shows its activity under the table, and one that passed does not', () => {
  const failed = { id: '793a2n1j', result: 'Red', retries: 0, usage: null, message: 'no note', activity: [CUT] }
  const passed = { id: '793a2n1a', result: 'Green', retries: 0, usage: null, message: '', activity: [CUT] }
  const quiet = { id: '793a2n1h', result: 'Red', retries: 0, usage: null, message: 'no note', activity: [] }
  const report = formatReport([failed, passed, quiet])

  assert.match(report, new RegExp(`793a2n1j activity: ${CUT.replaceAll('(', '\\(').replaceAll(')', '\\)')}`))
  assert.doesNotMatch(report, /793a2n1a activity/)
  assert.match(report, /793a2n1h activity: none shown/)
})

// The System's lines are transcript lines now (system/informational rows), not stream messages (793e7c).
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

const informational = (content) => ({ type: 'system', subtype: 'informational', content, level: 'notice' })
function sessionWithRows(rows, messages = []) {
  const file = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'rows-')), 's.jsonl')
  fs.writeFileSync(file, rows.map((row) => JSON.stringify(row)).join('\n') + '\n')
  return { messages, transcriptPath: () => file }
}

test('the lines of the System are kept from the system/informational rows of the transcript', () => {
  const rows = [
    informational('[engram route] looked up 3 notes'),
    informational('Memory activity: 1 item delivered'),
    informational('Automatic memory skipped: x'),
    informational('some unrelated notice'),
    { type: 'user', message: { content: 'hi' } },
  ]
  const entry = entryFor(world([sessionWithRows(rows)]), { status: 'FAILED', exception: { type: 'AssertionError' } })

  assert.deepEqual(entry.activity, [
    '[engram route] looked up 3 notes',
    'Memory activity: 1 item delivered',
    'Automatic memory skipped: x',
  ])
})

test('a permission denial is a Harness error, never a Red', () => {
  const denied = {
    type: 'user',
    message: {
      content: [
        { type: 'tool_result', tool_use_id: 't', is_error: true, content: 'Permission for this tool use was denied' },
      ],
    },
  }
  const use = { type: 'assistant', message: { content: [{ type: 'tool_use', id: 't', name: 'Bash', input: {} }] } }
  const entry = entryFor(world([sessionWithRows([use, denied])]), {
    status: 'FAILED',
    exception: { type: 'AssertionError' },
    message: 'the precondition of the turn was not observed',
  })
  assert.match(entry.result, /^Harness error/)
  assert.match(entry.result, /permission/i)
})
