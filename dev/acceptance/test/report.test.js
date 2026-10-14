// The report keeps the activity line of a scenario that did not pass (793e8p): a lookup the System cut shows its
// warning on the status line, which is what tells it from a judgement that missed.
import assert from 'node:assert/strict'
import test from 'node:test'

import { formatReport } from '../lib/report.js'
import { entryFor } from '../lib/results.js'

const CUT = 'Automatic memory was skipped because the System timed out (the route lookup)'
const status = (text) => ({ type: 'system', subtype: 'ui_status', text })
const world = (sessions) => ({ id: '793a2n1j', title: 't', scenario: { sessions, usage: () => null } })

test('the entry of a scenario holds the status lines its sessions showed, in order', () => {
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
