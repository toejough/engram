import os from 'node:os'
import path from 'node:path'

// The one-time experiments (H1, H2: 793b343) are not repository fixtures: their scripts, tours, gold and results live
// under ENGRAM_EVAL_DIR, in the same layout as the repository (dev/acceptance/h1, dev/acceptance/h2, dev/acceptance/steps/h1.js).
const ONE_TIME_STEPS = [path.join('dev', 'acceptance', 'steps', 'h1.js')]

// evalDir is ENGRAM_EVAL_DIR, or a directory under the system temp dir when it is unset or empty
export function evalDir(env = process.env) {
  return env.ENGRAM_EVAL_DIR || path.join(os.tmpdir(), 'engram-eval')
}

// evalStepFiles are the step files of the one-time experiments that exist under the eval directory
export function evalStepFiles(env, exists) {
  return ONE_TIME_STEPS.map((file) => path.join(evalDir(env), file)).filter((file) => exists(file))
}
