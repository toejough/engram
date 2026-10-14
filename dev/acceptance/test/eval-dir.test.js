import assert from 'node:assert/strict'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { evalDir, evalStepFiles } from '../lib/eval-dir.js'

test('the eval directory is ENGRAM_EVAL_DIR when set, and a directory under the system temp dir when not', () => {
  assert.equal(evalDir({ ENGRAM_EVAL_DIR: '/scratch/eval' }), '/scratch/eval')
  assert.equal(evalDir({}), path.join(os.tmpdir(), 'engram-eval'))
  assert.equal(evalDir({ ENGRAM_EVAL_DIR: '' }), path.join(os.tmpdir(), 'engram-eval'))
})

test('the one-time experiment steps are loaded from the eval directory only where they exist', () => {
  const present = new Set(['/scratch/eval/dev/acceptance/steps/h1.js'])
  const exists = (file) => present.has(file)
  assert.deepEqual(evalStepFiles({ ENGRAM_EVAL_DIR: '/scratch/eval' }, exists), ['/scratch/eval/dev/acceptance/steps/h1.js'])
  assert.deepEqual(evalStepFiles({ ENGRAM_EVAL_DIR: '/elsewhere' }, exists), [])
})
