import assert from 'node:assert/strict'
import test from 'node:test'

import { parseArgs } from '../lib/args.js'

test('a filter and the dry-run flag are read', () => {
  assert.deepEqual(parseArgs([]), { filter: '', dryRun: false, runs: 1 })
  assert.deepEqual(parseArgs(['793a5k1e']), { filter: '793a5k1e', dryRun: false, runs: 1 })
  assert.deepEqual(parseArgs(['--dry-run', '793d']), { filter: '793d', dryRun: true, runs: 1 })
})

test('an unknown flag or a second filter is an error, never a live run', () => {
  assert.throws(() => parseArgs(['--check']), /unknown flag --check/)
  assert.throws(() => parseArgs(['793a', '793b']), /at most one scenario ID prefix/)
})

test('a number of runs is read as a count, and anything else is an error, never a live run', () => {
  assert.equal(parseArgs(['--runs', '10', '793a2n1a']).runs, 10)
  assert.equal(parseArgs(['--runs=3']).runs, 3)
  const bad = [['--runs'], ['--runs', 'x'], ['--runs', '0'], ['--runs', '-2'], ['--runs', '2.5'], ['--runs', '101']]
  for (const argv of bad) assert.throws(() => parseArgs(argv), /--runs/, argv.join(' '))
})
