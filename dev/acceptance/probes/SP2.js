import fs from 'node:fs'
import path from 'node:path'

import { RECORDER } from './lib.js'

export const claim = 'A skills-directory plugin (linked) and the SDK plugins option load the hooks module in an isolated HOME and an arbitrary workspace; a module that did not load fails the driver fast'

export async function run(h) {
  const evidence = {}

  const linked = h.scenario()
  h.recorder(linked)
  const a = linked.startSession()
  await a.send('Reply with exactly the word: ok')
  await h.settle(1000)
  evidence.linkedUnderSkills = {
    kinds: h.events(linked).map((event) => event.kind),
    initPlugins: a.init.plugins.map((plugin) => plugin.name),
    workspace: linked.workspace,
  }

  const optioned = h.scenario()
  h.recorder(optioned)
  fs.rmSync(path.join(optioned.home, '.claude', 'skills', 'probe-recorder'))
  const b = optioned.startSession({ plugins: [{ type: 'local', path: RECORDER }] })
  await b.send('Reply with exactly the word: ok')
  await h.settle(1000)
  evidence.sdkPluginsOption = { kinds: h.events(optioned).map((event) => event.kind), initPlugins: b.init.plugins.map((plugin) => plugin.name) }

  const refused = h.scenario()
  h.recorder(refused)
  fs.writeFileSync(path.join(refused.home, '.claude', 'settings.json'), JSON.stringify({ disableAllHooks: true }))
  const c = refused.startSession()
  const started = Date.now()
  const outcome = await c.send('Reply with exactly the word: ok').then(
    () => ({ failed: false }),
    (error) => ({ failed: true, message: error.message.slice(0, 300) }),
  )
  evidence.failFast = { ...outcome, afterMs: Date.now() - started, stderrLine: c.stderr.split('\n').find((line) => /not loaded/i.test(line)) ?? null, kinds: h.events(refused).map((event) => event.kind) }
  return evidence
}
