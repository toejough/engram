import assert from 'node:assert/strict'
import test from 'node:test'

import { AUTOMATIC, parseNote, subagentOf } from '../lib/vault.js'

const NOTE = `---
type: feedback
situation: running Go tests
action: use targ
source: 'captured from the conversation; written automatically (session s1, turn 3, subagent a1b2c3)'
---

Lesson learned: use targ, not go test.
`

test('a note has a front matter, a body and a source', () => {
  const note = parseNote(NOTE)
  assert.match(note.body, /^Lesson learned: use targ/)
  assert.ok(!note.body.includes('source'))
  assert.match(note.source, AUTOMATIC)
  assert.equal(subagentOf(note.source), 'a1b2c3')
})

test('a source that names no subagent names none', () => {
  assert.equal(subagentOf('captured from the conversation; written automatically (session s1, turn 3)'), null)
  assert.equal(subagentOf('Pi session; saved by the Pi agent running the learn skill'), null)
})

test('a lesson worded only in the source is not in what the note says; one worded in its fields is', () => {
  const note = parseNote('---\nsource: use targ not go test\naction: use targ test\n---\n\nsomething else\n')
  assert.ok(!/go test/.test(note.content))
  assert.ok(/targ test/.test(note.content))
  assert.ok(!/targ/.test(note.body))
})
