import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import { baseEnv } from '../lib/run-context.js'

export const claim = 'A fresh session costs how long and how many tokens; the effect of the pre-warmed embedder cache'

const mean = (xs) => xs.reduce((a, b) => a + b, 0) / xs.length
const round = (x, d = 3) => Math.round(x * 10 ** d) / 10 ** d

function tokens(turn) {
  const u = turn.usage ?? {}
  return { input: u.input_tokens, cacheCreate: u.cache_creation_input_tokens, cacheRead: u.cache_read_input_tokens, output: u.output_tokens }
}

export async function run(h) {
  const evidence = {}
  const patterns = {
    trivial: 'Reply with exactly the word: pong',
    oneToolCall: 'Run the shell command `echo probe-ran` with the Bash tool, then reply with the single word DONE.',
    twoToolCalls: 'Run `echo one` and then `echo two` with the Bash tool (two separate calls), then reply with the single word DONE.',
  }
  const runs = []
  for (const [name, prompt] of Object.entries(patterns)) {
    const samples = []
    for (let i = 0; i < 3; i += 1) {
      const scenario = h.scenario()
      const session = scenario.startSession()
      const turn = await session.send(prompt)
      samples.push({ costUsd: turn.costUsd, wallS: turn.wallMs / 1000, ...tokens(turn) })
      await session.close()
    }
    runs.push({
      pattern: name,
      n: samples.length,
      meanWallS: round(mean(samples.map((s) => s.wallS)), 1),
      meanCostUsd: round(mean(samples.map((s) => s.costUsd))),
      minMaxCostUsd: [round(Math.min(...samples.map((s) => s.costUsd))), round(Math.max(...samples.map((s) => s.costUsd)))],
      meanTokens: Object.fromEntries(['input', 'cacheCreate', 'cacheRead', 'output'].map((k) => [k, Math.round(mean(samples.map((s) => s[k] ?? 0)))])),
    })
  }
  evidence.sessionTurns = runs

  // the embedder cache: the same engram query with an empty cache directory, then the pre-warmed one
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'sp11-'))
  const timed = async (cacheDir) => {
    const scenario = h.scenario()
    const started = Date.now()
    await scenario.engram(['query', '--phrase', 'x', '--limit', '1'], { extraEnv: { XDG_CACHE_HOME: cacheDir } })
    return (Date.now() - started) / 1000
  }
  const cold = await timed(path.join(home, 'cold-cache'))
  const warm = await timed(h.context.cacheDir)
  const warmAgain = await timed(h.context.cacheDir)
  fs.rmSync(home, { recursive: true, force: true })
  evidence.embedderCache = { coldQueryS: round(cold, 2), warmQueryS: round(warm, 2), warmAgainS: round(warmAgain, 2) }

  // the install of a scenario and a seeded note
  const scenario = h.scenario()
  let started = Date.now()
  await scenario.install()
  evidence.installS = round((Date.now() - started) / 1000, 2)
  started = Date.now()
  await scenario.seedNote('N', 'tagging a release', 'releases are tagged from main')
  evidence.seedNoteS = round((Date.now() - started) / 1000, 2)
  return evidence
}
