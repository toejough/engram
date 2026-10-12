import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

import { rowsContaining } from './lib.js'

export const claim = '$.ui.status, $.env.get and $.fs.write work in handlers of a stream with no interface; a gating hook needs a .catch handler (claude plugin validate warns)'

const STATUS = 'PROBE-STATUS-line-1'

function makeMod(dir, source) {
  fs.mkdirSync(path.join(dir, '.claude-plugin'), { recursive: true })
  fs.mkdirSync(path.join(dir, 'hooks'), { recursive: true })
  fs.writeFileSync(path.join(dir, '.claude-plugin', 'plugin.json'), JSON.stringify({ name: path.basename(dir), version: '0.0.1', author: { name: 'probe' } }))
  fs.writeFileSync(path.join(dir, 'hooks', 'hooks.json'), JSON.stringify({ modules: ['./register.js'] }))
  fs.writeFileSync(path.join(dir, 'hooks', 'register.js'), source)
}

function validate(h, scenario, dir) {
  try {
    return execFileSync(h.context.claudePath, ['plugin', 'validate', dir], { encoding: 'utf8', env: scenario.env(), cwd: scenario.workspace })
  } catch (error) {
    return `${error.stdout ?? ''}${error.stderr ?? ''}`
  }
}

export async function run(h) {
  const evidence = {}
  const scenario = h.scenario()
  h.recorder(scenario, { status: STATUS })
  const session = scenario.startSession()
  const turn = await session.send('Reply with exactly the word: ok')
  await h.settle(1000)
  const events = h.events(scenario)
  evidence.handlers = {
    envGetAndFsWriteWorked: events.length > 0,
    statusRecorded: events.some((event) => event.kind === 'ui.status'),
    statusErrors: events.filter((event) => event.kind === 'ui.status.error').map((event) => event.data),
    statusMessage: session.messages.filter((message) => JSON.stringify(message).includes(STATUS)).map((message) => ({ type: message.type, subtype: message.subtype, plugin: message.plugin, text: message.text })),
    statusInTranscript: rowsContaining(h.rows(session), STATUS).length,
    replyOk: !turn.isError,
  }

  const dir = path.join(scenario.dir, 'validate')
  makeMod(path.join(dir, 'gate-no-catch'), "export function register(on) {\n  on('tool.call', { tool: 'Bash' }, async ($, e, next) => next(e))\n}\n")
  makeMod(path.join(dir, 'gate-with-catch'), "export function register(on) {\n  on('tool.call', { tool: 'Bash' }, async ($, e, next) => next(e)).catch(async ($, e, next) => (next.called ? next(e) : { deny: 'guard failed' }))\n}\n")
  const lines = (name) => validate(h, scenario, path.join(dir, name)).split('\n').filter((line) => /gating|passed|failed|error/i.test(line)).map((line) => line.trim())
  evidence.validate = { withoutCatch: lines('gate-no-catch'), withCatch: lines('gate-with-catch') }
  return evidence
}
