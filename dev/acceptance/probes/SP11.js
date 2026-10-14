import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { MS_PER_SECOND } from '../lib/units.js'

const DEFAULT_DIGITS = 3
const COARSE_DIGITS = 2
const BASE = 10
const TRIALS = 3

export const claim = 'A fresh session costs how long and how many tokens; the effect of the pre-warmed embedder cache'

const mean = (values) => values.reduce((sum, value) => sum + value, 0) / values.length
const round = (value, digits = DEFAULT_DIGITS) => Math.round(value * BASE ** digits) / BASE ** digits

function tokens(turn) {
  const usage = turn.usage ?? {}
  return {
    input: usage.input_tokens,
    cacheCreate: usage.cache_creation_input_tokens,
    cacheRead: usage.cache_read_input_tokens,
    output: usage.output_tokens,
  }
}

export async function run(harness) {
  const evidence = {}
  const patterns = {
    trivial: 'Reply with exactly the word: pong',
    oneToolCall: 'Run the shell command `echo probe-ran` with the Bash tool, then reply with the single word DONE.',
    twoToolCalls:
      'Run `echo one` and then `echo two` with the Bash tool (two separate calls), then ' +
      'reply with the single word DONE.',
  }
  const runs = []
  for (const [name, prompt] of Object.entries(patterns)) {
    const samples = []
    for (let index = 0; index < TRIALS; index += 1) {
      const scenario = harness.scenario()
      const session = scenario.startSession()
      const turn = await session.send(prompt)
      samples.push({ costUsd: turn.costUsd, wallS: turn.wallMs / MS_PER_SECOND, ...tokens(turn) })
      await session.close()
    }
    runs.push({
      pattern: name,
      count: samples.length,
      meanWallS: round(mean(samples.map((sample) => sample.wallS)), 1),
      meanCostUsd: round(mean(samples.map((sample) => sample.costUsd))),
      minMaxCostUsd: [
        round(Math.min(...samples.map((sample) => sample.costUsd))),
        round(Math.max(...samples.map((sample) => sample.costUsd))),
      ],
      meanTokens: Object.fromEntries(
        ['input', 'cacheCreate', 'cacheRead', 'output'].map((field) => [
          field,
          Math.round(mean(samples.map((sample) => sample[field] ?? 0))),
        ]),
      ),
    })
  }
  evidence.sessionTurns = runs

  // the embedder cache: the same engram query with an empty cache directory, then the pre-warmed one
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'sp11-'))
  const timed = async (cacheDir) => {
    const scenario = harness.scenario()
    const started = Date.now()
    await scenario.engram(['query', '--phrase', 'x', '--limit', '1'], { extraEnv: { XDG_CACHE_HOME: cacheDir } })
    return (Date.now() - started) / MS_PER_SECOND
  }
  const cold = await timed(path.join(home, 'cold-cache'))
  const warm = await timed(harness.context.cacheDir)
  const warmAgain = await timed(harness.context.cacheDir)
  fs.rmSync(home, { recursive: true, force: true })
  evidence.embedderCache = {
    coldQueryS: round(cold, COARSE_DIGITS),
    warmQueryS: round(warm, COARSE_DIGITS),
    warmAgainS: round(warmAgain, COARSE_DIGITS),
  }

  // the install of a scenario and a seeded note
  const scenario = harness.scenario()
  let started = Date.now()
  await scenario.install()
  evidence.installS = round((Date.now() - started) / MS_PER_SECOND, COARSE_DIGITS)
  started = Date.now()
  await scenario.seedNote('N', 'tagging a release', 'releases are tagged from main')
  evidence.seedNoteS = round((Date.now() - started) / MS_PER_SECOND, COARSE_DIGITS)
  return evidence
}
