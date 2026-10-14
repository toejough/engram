import assert from 'node:assert/strict'
import test from 'node:test'

import { MIN_CLAUDE, parseClaudeVersion, checkPrograms } from '../lib/preflight.js'

test('the Claude Code version is read from its --version line', () => {
  assert.deepEqual(parseClaudeVersion('2.1.296 (Claude Code)'), [2, 1, 296])
  assert.equal(parseClaudeVersion('nonsense'), null)
})

test('a missing program is reported by name with how to get it', () => {
  const which = (name) => (name === 'claude' ? null : `/bin/${name}`)
  assert.throws(() => checkPrograms({ which, run: () => '' }, { needClaude: true }), /claude.*not found on PATH/s)
  const none = () => null
  assert.throws(() => checkPrograms({ which: none, run: () => '' }, { needClaude: false }), /go.*not found on PATH/s)
})

test('a Claude Code older than the minimum fails and names both versions', () => {
  const which = (name) => `/bin/${name}`
  const run = () => '2.1.286 (Claude Code)'
  assert.throws(
    () => checkPrograms({ which, run }, { needClaude: true }),
    (error) => error.message.includes('2.1.286') && error.message.includes(MIN_CLAUDE.join('.')),
  )
})

test('with every program present the claude path and version are returned', () => {
  const which = (name) => `/bin/${name}`
  const found = checkPrograms({ which, run: () => '2.1.296 (Claude Code)' }, { needClaude: true })
  assert.equal(found.path, '/bin/claude')
  assert.deepEqual(found.version, [2, 1, 296])
})
