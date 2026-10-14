// Self-tests of the refactor gate for minimality (793e8l): the tool is tested on fixtures; whether
// the mod is within the gate is what the gate itself says, and it is red on purpose.
import assert from 'node:assert/strict'
import test from 'node:test'

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import { compare, externalImports, formatTable, goTable, measureGo, ownedLines, passes } from '../gates/owned-lines.js'

test('owned lines are the lines that are neither blank nor a comment, block comments included', () => {
  const text = [
    '// header',
    '',
    'const a = 1',
    '  // indented comment',
    '/* one line */',
    '/*',
    ' * block',
    ' */',
    'const b = 2 // trailing',
    '',
  ].join('\n')
  assert.equal(ownedLines(text), 2)
})

test('a budget table is compared per file and in total, and a file without a row fails', () => {
  const table = { total: 30, files: { 'a.js': { budget: 10 }, 'b.js': { budget: 20 } } }
  const within = compare({ 'a.js': 10, 'b.js': 15 }, table)
  assert.equal(passes(within, []), true)
  const over = compare({ 'a.js': 11, 'b.js': 15 }, table)
  assert.deepEqual(
    over.over.map((row) => row.file),
    ['a.js'],
  )
  assert.equal(passes(over, []), false)
  const total = compare(
    { 'a.js': 10, 'b.js': 20, 'c.js': 5 },
    { total: 30, files: { ...table.files, 'c.js': { budget: 5 } } },
  )
  assert.equal(total.totalExcess, 5)
  assert.equal(passes(total, []), false)
  const extra = compare({ 'a.js': 1, 'new.js': 1 }, table)
  assert.deepEqual(
    extra.unbudgeted.map((row) => row.file),
    ['new.js'],
  )
  assert.equal(passes(extra, []), false)
  assert.equal(passes(within, ['a.js imports left-pad']), false)
  assert.match(formatTable(over), /a\.js\s+11\s+10\s+1/)
})

test('an import that is neither relative nor a node built-in is a dependency', () => {
  const text = [
    "import fs from 'node:fs'",
    "import { local } from './local.js'",
    "import left from 'left-pad'",
    "export { other } from '../other.js'",
    "export { thing } from 'c-lib'",
  ].join('\n')
  assert.deepEqual(externalImports(text), ['left-pad', 'c-lib'])
})

test('a Go component is held to the owned JavaScript lines it replaces times the ratio, and a file without a component fails', () => {
  const go = {
    ratio: 1.3,
    plumbing: ['ports.go'],
    components: { moment: { js: 100, files: ['moment.go'] }, summary: { js: 20, files: ['summary.go'] } },
  }
  assert.deepEqual(goTable(go), { total: 156, files: { moment: { budget: 130 }, summary: { budget: 26 } } })
  const ceilinged = { ...go, components: { moment: { ...go.components.moment, ceiling: 140 }, summary: go.components.summary } }
  assert.deepEqual(goTable(ceilinged).files.moment, { budget: 140 })
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'go-budget-'))
  try {
    const lines = (count) => `${'x := 1\n'.repeat(count)}// a comment\n\n`
    fs.writeFileSync(path.join(dir, 'moment.go'), lines(130))
    fs.writeFileSync(path.join(dir, 'summary.go'), lines(26))
    fs.writeFileSync(path.join(dir, 'ports.go'), lines(500))
    fs.writeFileSync(path.join(dir, 'moment_test.go'), lines(500))
    const within = compare(measureGo(dir, go), goTable(go))
    assert.deepEqual(measureGo(dir, go), { moment: 130, summary: 26 })
    assert.equal(passes(within, []), true)
    fs.writeFileSync(path.join(dir, 'moment.go'), lines(131))
    assert.equal(passes(compare(measureGo(dir, go), goTable(go)), []), false)
    fs.writeFileSync(path.join(dir, 'moment.go'), lines(130))
    fs.writeFileSync(path.join(dir, 'extra.go'), lines(1))
    const unbudgeted = compare(measureGo(dir, go), goTable(go))
    assert.deepEqual(
      unbudgeted.unbudgeted.map((row) => row.file),
      ['extra.go'],
    )
    assert.equal(passes(unbudgeted, []), false)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})
