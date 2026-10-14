// Helpers for the spike probes of 793e8a: a probe is a function of this helper that runs real sessions in a
// scenario World and returns the evidence it observed (plain data, printed as JSON by probes/run.js).
import fs from 'node:fs'
import path from 'node:path'
import { setTimeout as sleep } from 'node:timers/promises'
import { fileURLToPath } from 'node:url'

import { redactDeep, resolveCredentials } from '../lib/creds.js'
import { preflight } from '../lib/preflight.js'
import { createRunContext, disposeRunContext } from '../lib/run-context.js'
import { readRows } from '../lib/transcript.js'
import { Scenario } from '../lib/world.js'

const HERE = path.dirname(fileURLToPath(import.meta.url))
const ROW_KEYS = 12
export const RECORDER = path.join(HERE, 'recorder')

export async function withProbeContext(body, { withCredentials = true } = {}) {
  const claude = preflight({ needClaude: true })
  const creds = withCredentials
    ? resolveCredentials(process.env, { readFile: (file) => fs.readFileSync(file, 'utf8') })
    : { env: {}, values: [] }
  const context = await createRunContext({ claudePath: claude.path, creds })
  const scenarios = []
  const helpers = {
    context,
    scenario() {
      const scenario = new Scenario(context)
      scenarios.push(scenario)
      return scenario
    },
    // linkMod puts a mod where Claude Code loads it from, the skills directory of the scenario's home (793b251).
    linkMod(scenario, modDir, name) {
      const skills = path.join(scenario.home, '.claude', 'skills')
      fs.mkdirSync(skills, { recursive: true })
      fs.symlinkSync(modDir, path.join(skills, name))
    },
    // recorder installs the shared recorder mod and points it at an events directory and a configuration.
    recorder(scenario, config = {}) {
      const dir = path.join(scenario.dir, 'probe-events')
      fs.mkdirSync(dir, { recursive: true })
      helpers.linkMod(scenario, RECORDER, 'probe-recorder')
      scenario.extraEnv.PROBE_DIR = dir
      scenario.extraEnv.PROBE_CONFIG = JSON.stringify(config)
      return dir
    },
    events(scenario) {
      const dir = scenario.extraEnv.PROBE_DIR
      if (!dir || !fs.existsSync(dir)) return []
      return fs
        .readdirSync(dir)
        .sort()
        .map((file) => JSON.parse(fs.readFileSync(path.join(dir, file), 'utf8')))
    },
    rows(session) {
      const file = session.transcriptPath()
      return file ? readRows(file) : []
    },
    async settle(milliseconds) {
      await sleep(milliseconds)
    },
  }
  try {
    return redactDeep(await body(helpers), creds)
  } finally {
    await Promise.all(scenarios.map((scenario) => scenario.dispose()))
    disposeRunContext(context)
  }
}

// rowsContaining lists, for each transcript row that holds the text, where it sits.
export function rowsContaining(rows, needle) {
  return rows
    .map((row, index) => ({ row, index }))
    .filter(({ row }) => JSON.stringify(row).includes(needle))
    .map(({ row, index }) => ({
      index,
      type: row.type,
      attachmentType: row.attachment?.type,
      role: row.message?.role,
      keys: Object.keys(row).slice(0, ROW_KEYS),
    }))
}
