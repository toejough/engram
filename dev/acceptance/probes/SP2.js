import fs from 'node:fs'
import path from 'node:path'

import { RECORDER } from './lib.js'
import { LONG_EXCERPT, MS_PER_SECOND, excerpt } from '../lib/units.js'

export const claim =
  'A skills-directory plugin (linked) and the SDK plugins option load the hooks module ' +
  'in an isolated HOME and an arbitrary workspace; a module that did not load fails the driver fast'

export async function run(harness) {
  const evidence = {}

  const linked = harness.scenario()
  harness.recorder(linked)
  const linkedSession = linked.startSession()
  await linkedSession.send('Reply with exactly the word: ok')
  await harness.settle(MS_PER_SECOND)
  evidence.linkedUnderSkills = {
    kinds: harness.events(linked).map((event) => event.kind),
    initPlugins: linkedSession.init.plugins.map((plugin) => plugin.name),
    workspace: linked.workspace,
  }

  const optioned = harness.scenario()
  harness.recorder(optioned)
  fs.rmSync(path.join(optioned.home, '.claude', 'skills', 'probe-recorder'))
  const optionedSession = optioned.startSession({ plugins: [{ type: 'local', path: RECORDER }] })
  await optionedSession.send('Reply with exactly the word: ok')
  await harness.settle(MS_PER_SECOND)
  evidence.sdkPluginsOption = {
    kinds: harness.events(optioned).map((event) => event.kind),
    initPlugins: optionedSession.init.plugins.map((plugin) => plugin.name),
  }

  const refused = harness.scenario()
  harness.recorder(refused)
  fs.writeFileSync(path.join(refused.home, '.claude', 'settings.json'), JSON.stringify({ disableAllHooks: true }))
  const refusedSession = refused.startSession()
  const started = Date.now()
  const outcome = await refusedSession.send('Reply with exactly the word: ok').then(
    () => ({ failed: false }),
    (error) => ({ failed: true, message: excerpt(error.message, LONG_EXCERPT) }),
  )
  evidence.failFast = {
    ...outcome,
    afterMs: Date.now() - started,
    stderrLine: refusedSession.stderr.split('\n').find((line) => /not loaded/i.test(line)) ?? null,
    kinds: harness.events(refused).map((event) => event.kind),
  }
  return evidence
}
