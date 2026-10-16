// The rate bar (793b421): a row that depends on a model's judgement passes at 9 of 10 runs, every Red attributed from
// the activity line the System showed.
import assert from 'node:assert/strict'
import test from 'node:test'

import { attribution, formatRates, ratePassed, summarize } from '../lib/rate.js'

const WARNING = 'Automatic memory skipped: prompt lookup timed out: engram aborted; no notes delivered for this prompt (details: ~/.local/share/engram/auto.log)'
const green = (id) => ({ id, result: 'Green', retries: 0, usage: null, message: '' })
const red = (id, activity) => ({
  id,
  result: 'Red',
  retries: 0,
  usage: null,
  message: 'note N was not delivered',
  activity,
})

test('a Red is a System failure when its activity line holds a warning, a System miss when it holds none', () => {
  assert.deepEqual(attribution(red('a', [WARNING])), { kind: 'System failure', detail: WARNING })
  assert.equal(attribution(red('a', [])).kind, 'System miss')
  assert.equal(attribution(red('a', ['Memory activity: 1 item delivered'])).kind, 'System miss')
  assert.equal(attribution(red('a', undefined)).kind, 'unattributed')
})

test('the rate of a row is its Green runs over its runs, and 9 of 10 meets the bar, 8 of 10 does not', () => {
  const nine = Array.from({ length: 10 }, (_, run) => (run === 3 ? red('793a2n1a', []) : green('793a2n1a')))
  const eight = Array.from({ length: 10 }, (_, run) => (run < 2 ? red('793a2n1j', []) : green('793a2n1j')))
  const rows = summarize([nine, eight])

  assert.deepEqual(
    rows.map((row) => [row.id, row.runs, row.green, row.red, row.rate, row.met]),
    [
      ['793a2n1a', 10, 9, 1, 0.9, true],
      ['793a2n1j', 10, 8, 2, 0.8, false],
    ],
  )
  assert.equal(ratePassed(rows), false)
  assert.equal(ratePassed(rows.slice(0, 1)), true)
})

test('a Red without an activity line, or a harness error, keeps a row from meeting the bar', () => {
  const unattributed = Array.from({ length: 10 }, (_, run) => (run === 0 ? red('x', undefined) : green('x')))
  const boom = { ...green('y'), result: 'Harness error (boom)' }
  const errored = Array.from({ length: 10 }, (_, run) => (run === 0 ? boom : green('y')))
  const rows = summarize([unattributed, errored])

  assert.deepEqual(rows.map((row) => row.met), [false, false])
})

test('a row that was not run has no rate and does not fail the run', () => {
  const rows = summarize([[{ id: 'z', result: 'Not run (no headless Pi)', retries: 0, usage: null, message: '' }]])

  assert.equal(rows[0].rate, null)
  assert.equal(rows[0].met, null)
  assert.equal(ratePassed(rows), true)
})

test('the report holds the rate of each row and the attribution of each Red, by run', () => {
  const rounds = [[red('793a2n1j', [WARNING])], [green('793a2n1j')], [red('793a2n1j', [])]]
  const report = formatRates(summarize(rounds), rounds)

  assert.match(report, /793a2n1j\s+3\s+1\s+2\s+0\s+0\s+0\.33\s+no/)
  assert.match(report, /793a2n1j run 1: System failure: Automatic memory was skipped because the System timed out/)
  assert.match(report, /793a2n1j run 3: System miss: no warning on the activity line/)
  assert.match(report, /3 runs of 1 rows/)
})
