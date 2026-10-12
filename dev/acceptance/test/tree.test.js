import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { changes, lost, snapshotTree } from '../lib/tree.js'

function home() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'tree-test-'))
  fs.mkdirSync(path.join(dir, '.claude', 'engram', 'skills', 'recall'), { recursive: true })
  fs.writeFileSync(path.join(dir, '.claude', 'engram', 'skills', 'recall', 'SKILL.md'), 'recall')
  fs.mkdirSync(path.join(dir, '.claude', 'skills'), { recursive: true })
  fs.symlinkSync(path.join(dir, '.claude', 'engram', 'skills', 'recall'), path.join(dir, '.claude', 'skills', 'recall'))
  return dir
}

test('a snapshot holds files by hash, links by target and directories', () => {
  const dir = home()
  try {
    const entries = snapshotTree(dir)
    assert.match(entries.get(path.join('.claude', 'engram', 'skills', 'recall', 'SKILL.md')), /^file [0-9a-f]{64}$/)
    assert.match(entries.get(path.join('.claude', 'skills', 'recall')), /^link .*recall$/)
    assert.equal(entries.get(path.join('.claude', 'engram')), 'dir')
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('an identical tree has no changes; an edit, a removal and an addition are each named', () => {
  const dir = home()
  try {
    const before = snapshotTree(dir)
    assert.deepEqual(changes(before, snapshotTree(dir)), [])
    fs.writeFileSync(path.join(dir, '.claude', 'engram', 'skills', 'recall', 'SKILL.md'), 'edited')
    fs.writeFileSync(path.join(dir, '.claude', 'skills', 'new'), 'x')
    fs.rmSync(path.join(dir, '.claude', 'skills', 'recall'))
    const after = snapshotTree(dir)
    assert.deepEqual(changes(before, after).sort(), [
      `added ${path.join('.claude', 'skills', 'new')}`,
      `changed ${path.join('.claude', 'engram', 'skills', 'recall', 'SKILL.md')}`,
      `removed ${path.join('.claude', 'skills', 'recall')}`,
    ])
    assert.equal(lost(before, after).length, 2)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('a home with neither root has an empty snapshot', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'tree-test-'))
  try {
    assert.equal(snapshotTree(dir).size, 0)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})
