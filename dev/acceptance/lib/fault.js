// The fault seam of the System's work (793b418): the harness sets ENGRAM_AUTO_FAULT for the processes of a scenario,
// and `engram auto run` makes the operations of a job fail (`error`), the lookups never answer (`hang`) or the work
// other than lookups take N seconds (`delay:N`), noting in the file named by ENGRAM_AUTO_FAULT_LOG when each slow
// operation began. It replaced the programs named engram that stood before the real one on the PATH, which cannot
// see the operations the engine performs inside its own process.
import fs from 'node:fs'
import path from 'node:path'

const MODES = ['error', 'hang', 'delay']

// faultEnv is the environment that sets the fault of a scenario; the log is a file beside the scenario.
export function faultEnv({ dir, mode, seconds = 0 }) {
  if (!MODES.includes(mode)) throw new Error(`no fault mode "${mode}"`)
  return {
    ENGRAM_AUTO_FAULT: mode === 'delay' ? `delay:${seconds}` : mode,
    ENGRAM_AUTO_FAULT_LOG: path.join(dir, 'fault.log'),
  }
}

// faultCalls are the lines the engine logged: { seconds, what }.
export function faultCalls(env) {
  const file = env?.ENGRAM_AUTO_FAULT_LOG
  if (!file) return []
  try {
    return fs
      .readFileSync(file, 'utf8')
      .split('\n')
      .filter((line) => line !== '')
      .map((line) => {
        const [seconds, ...rest] = line.split(' ')
        return { seconds: Number(seconds), what: rest.join(' ') }
      })
  } catch {
    return []
  }
}
