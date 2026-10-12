import { execFile } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { promisify } from 'node:util'

import { redact } from '../lib/creds.js'
import { whichOnPath } from '../lib/preflight.js'

const exec = promisify(execFile)

export const claim = 'pi runs headless in an isolated home and writes a session file that the Transcript Reader parses'

export async function run(h) {
  const pi = whichOnPath('pi')
  if (!pi) return { available: false }
  const evidence = { available: true, version: (await exec(pi, ['--version'], { env: { PATH: process.env.PATH } }).catch((e) => ({ stdout: '', stderr: e.message }))).stdout.trim() }
  const attempt = async (extraEnv) => {
    const scenario = h.scenario()
    const sessions = path.join(scenario.dir, 'pi-sessions')
    const env = { ...scenario.env(), PI_OFFLINE: '1', ...extraEnv }
    const pending = exec(pi, ['-p', 'Reply with the word pong', '--provider', 'anthropic', '--model', 'claude-haiku-4-5', '--session-dir', sessions], { cwd: scenario.workspace, env, timeout: 60000 })
    pending.child.stdin.end()
    const result = await pending.catch((error) => ({ stdout: error.stdout ?? '', stderr: error.stderr ?? '', failed: true, message: error.message }))
    const files = fs.existsSync(sessions) ? fs.readdirSync(sessions) : []
    const ingest = files.length ? await scenario.engram(['ingest', '--transcript', path.join(sessions, files[0])]).then((out) => out.trim().split('\n').pop(), (error) => error.message.slice(0, 100)) : null
    return {
      output: redact(`${result.stdout}${result.stderr}${result.message ?? ''}`, h.context.creds).trim().slice(0, 160),
      sessionFiles: files.length,
      firstRows: files.length ? fs.readFileSync(path.join(sessions, files[0]), 'utf8').split('\n').filter(Boolean).slice(0, 5).map((line) => JSON.parse(line).type) : [],
      ingestLine: ingest && ingest.replace(/^ingest \S+: /, ''),
    }
  }
  evidence.withoutCredentials = await attempt({})
  const token = h.context.creds.env.CLAUDE_CODE_OAUTH_TOKEN
  evidence.withTheSubscriptionToken = token ? await attempt({ ANTHROPIC_OAUTH_TOKEN: token }) : 'no oauth token in use'
  return evidence
}
