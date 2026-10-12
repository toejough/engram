// Needs the installed dependencies: run by `targ acceptance --check`, not by check-full.
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import { extractScenarios, writeFeatures, PRODUCT_SPEC } from '../../lib/extract.js'
import { checkVocabulary } from '../../lib/vocabulary.js'

const PLAN = path.join(path.dirname(PRODUCT_SPEC), '793e_implementation-plan.md')

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')

test('every step line of the 43 product scenarios matches exactly one of the 27 definitions, and each is used', async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vocabulary-test-'))
  try {
    const features = writeFeatures(extractScenarios(PRODUCT_SPEC), dir)
    const report = await checkVocabulary({ features, support: [path.join(ROOT, 'steps', 'index.js')], cwd: ROOT })
    assert.deepEqual(report.undefinedSteps, [])
    assert.deepEqual(report.ambiguousSteps, [])
    assert.equal(report.scenarios, 51)
    assert.equal(report.definitions, 27)
    assert.equal(report.unusedDefinitions.length, 0, report.unusedDefinitions.join('\n'))
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('the step catalog of the plan lists the same 27 definitions, numbered 1 to 27', () => {
  const section = fs.readFileSync(PLAN, 'utf8').split('### 793e2i Step catalog')[1].split('## 793e3')[0]
  const numbers = [...section.matchAll(/^\| (\d+) \| (Given|When|Then) \|/gm)].map((match) => Number(match[1]))
  assert.deepEqual(numbers, Array.from({ length: 27 }, (_, i) => i + 1))
})
