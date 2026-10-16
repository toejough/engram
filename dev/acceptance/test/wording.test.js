// The words a failure is shown in are the engine's (internal/auto/failure.go, `warning` and the patterns of a class);
// the mod words a failure of the engram program itself, which the engine cannot, with the same words (793b413). This
// test holds the two to each other, so that they cannot drift.
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

import { REPO_ROOT } from '../lib/extract.js'

const ENGINE_SOURCE = fs.readFileSync(path.join(REPO_ROOT, 'internal', 'auto', 'failure.go'), 'utf8')
const MOD_SOURCE = fs.readFileSync(
  path.join(REPO_ROOT, 'agent-instructions', 'mods', 'engram-memory', 'hooks', 'register.js'),
  'utf8',
)

// the engine's words per class (what happened, after the operation), and the start of every warning
const engineReasons = () => {
  const body = /func predicateOf\(kind string\) string \{([\s\S]*?)\n\}/.exec(ENGINE_SOURCE)?.[1] ?? ''
  const unavailable = /case kindUnavailable:\s*return "([^"]+)"/.exec(body)
  const timeout = /case kindTimeout:\s*return "([^"]+)"/.exec(body)
  const error = /\n\treturn "([^"]+)"\s*$/.exec(body)
  assert.ok(unavailable && timeout && error, 'failure.go no longer words a failure the way this test reads it')
  return { unavailable: unavailable[1], timeout: timeout[1], error: error[1] }
}
const enginePrefix = () => {
  const found = /skippedPrefix\s*=\s*"([^"]+)"/.exec(ENGINE_SOURCE)
  assert.ok(found, 'failure.go has no skippedPrefix')
  return found[1]
}

// the engine's patterns per class
const engineCue = (variable) => {
  const found = new RegExp(`${variable}\\s*=\\s*regexp\\.MustCompile\\(\`\\(\\?i\\)([^\`|]+)`).exec(ENGINE_SOURCE)
  assert.ok(found, `failure.go has no pattern ${variable}`)
  return found[1]
}

// the mod's table: [text of a failure, words]
const modReasons = () => {
  const table = /const REASONS = \[((?:\s*\['[^']+', '[^']+'\],?)+)\s*\]/.exec(MOD_SOURCE)
  const unclassed = /const UNCLASSED_REASON = '([^']+)'/.exec(MOD_SOURCE)
  const prefix = /const SKIPPED_PREFIX = '([^']+)'/.exec(MOD_SOURCE)
  assert.ok(table && unclassed && prefix, 'register.js no longer words a failure the way this test reads it')
  return {
    pairs: [...table[1].matchAll(/\['([^']+)', '([^']+)'\]/g)].map((match) => [match[1], match[2]]),
    unclassed: unclassed[1],
    prefix: prefix[1],
  }
}

test('the mod words the failures of the engram program as the engine words the failures of its steps', () => {
  const engine = engineReasons()
  const mod = modReasons()
  const wordsOf = Object.fromEntries(mod.pairs)
  assert.equal(wordsOf[engineCue('timedOut')], engine.timeout)
  assert.equal(wordsOf[engineCue('unavailable')], engine.unavailable)
  assert.equal(mod.unclassed, engine.error)
  assert.equal(mod.pairs.length, 2)
  assert.equal(mod.prefix, enginePrefix())
})

// the housekeeping is never named in a warning, by the engine or by the mod (793b450): the user does not know the word
test('no warning names the housekeeping', () => {
  assert.ok(!/HOUSEKEEPING_STEP = '[^']*housekeeping/i.test(MOD_SOURCE), 'the mod names a step housekeeping')
  assert.ok(!/attempt\([^)]*"housekeeping"/.test(ENGINE_SOURCE), 'the engine names a step housekeeping')
})
