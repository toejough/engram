import assert from 'node:assert/strict'
import test from 'node:test'

import { parseArgs } from '../lib/args.js'

test('a filter and the dry-run flag are read', () => {
  assert.deepEqual(parseArgs([]), { filter: '', dryRun: false })
  assert.deepEqual(parseArgs(['793a5k1e']), { filter: '793a5k1e', dryRun: false })
  assert.deepEqual(parseArgs(['--dry-run', '793d']), { filter: '793d', dryRun: true })
})

test('an unknown flag or a second filter is an error, never a live run', () => {
  assert.throws(() => parseArgs(['--check']), /unknown flag --check/)
  assert.throws(() => parseArgs(['793a', '793b']), /at most one scenario ID prefix/)
})
