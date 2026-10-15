// Needs the installed dependencies: run by `targ acceptance --check`, not by check-full.
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import { changeDirectory, productScenarios, writeFeatures } from '../../lib/extract.js'
import { checkVocabulary } from '../../lib/vocabulary.js'

const CHANGE = changeDirectory()

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')

test(
  'every step line of the 44 product scenarios matches exactly one of the 30 definitions, ' + 'and each is used',
  async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vocabulary-test-'))
    try {
      const features = writeFeatures(productScenarios(), dir)
      const report = await checkVocabulary({ features, support: [path.join(ROOT, 'steps', 'index.js')], cwd: ROOT })
      assert.deepEqual(report.undefinedSteps, [])
      assert.deepEqual(report.ambiguousSteps, [])
      assert.equal(report.scenarios, 52)
      assert.equal(report.definitions, 30)
      assert.equal(report.unusedDefinitions.length, 0, report.unusedDefinitions.join('\n'))
    } finally {
      fs.rmSync(dir, { recursive: true, force: true })
    }
  },
)

// the plan is a document of the change: it is deleted with the change directory at promotion
const SKIP_WITHOUT_CHANGE = { skip: CHANGE === null }
test('the step catalog of the plan lists the same 30 definitions, numbered 1 to 30', SKIP_WITHOUT_CHANGE, () => {
  const plan = fs.readFileSync(path.join(CHANGE, '793e_implementation-plan.md'), 'utf8')
  const section = plan.split('### 793e2i Step catalog')[1].split('## 793e3')[0]
  const numbers = [...section.matchAll(/^\| (\d+) \| (Given|When|Then) \|/gm)].map((match) => Number(match[1]))
  assert.deepEqual(
    numbers,
    Array.from({ length: 30 }, (_, index) => index + 1),
  )
})
