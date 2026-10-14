import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { pruneGoSources } from '../prune-go-sources.js'

test('the Go sources of an npm package are removed and its JavaScript is left', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'prune-test-'))
  try {
    fs.mkdirSync(path.join(dir, 'flatted', 'golang', 'pkg'), { recursive: true })
    fs.writeFileSync(path.join(dir, 'flatted', 'golang', 'pkg', 'flatted.go'), 'package flatted')
    fs.writeFileSync(path.join(dir, 'flatted', 'index.js'), 'export {}')
    const removed = pruneGoSources(dir)
    assert.deepEqual(
      removed.map((file) => path.relative(dir, file)),
      [path.join('flatted', 'golang', 'pkg', 'flatted.go')],
    )
    assert.equal(fs.existsSync(path.join(dir, 'flatted', 'index.js')), true)
    assert.deepEqual(pruneGoSources(dir), [])
    assert.deepEqual(pruneGoSources(path.join(dir, 'absent')), [])
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})
