import { rowsContaining } from './lib.js'
import { MS_PER_SECOND, tenthsOfSeconds } from '../lib/units.js'

const STDERR_LINES = 3

export const claim = 'Handler time inside $.process.run does not count against the 10 s hook budget for a 25 s job'
export const documentation =
  'reference.md, Limits: "A hook\'s own execution time for one event, not counting ' +
  'time inside `next` or a mods API call other than `$.clock.sleep`: 10 seconds"'

const TOKEN = 'CTX-LONG-3b9a'

export async function run(harness) {
  const scenario = harness.scenario()
  harness.recorder(scenario, { sleepSeconds: 25, context: { promptSubmit: TOKEN } })
  const session = scenario.startSession()
  const started = Date.now()
  const turn = await session.send('Reply with exactly the word: ok')
  const wallMs = Date.now() - started
  await harness.settle(MS_PER_SECOND)
  const events = harness.events(scenario)
  const processRun = events.find((event) => event.kind === 'process.run')
  return {
    documentation,
    turnWallS: tenthsOfSeconds(wallMs),
    replyOk: !turn.isError && /ok/i.test(turn.text),
    processRun: processRun?.data,
    hookErrors: events.filter((event) => event.kind.endsWith('.error')).map((event) => event.data),
    contextAddedAfterTheWait: rowsContaining(harness.rows(session), TOKEN).filter(
      (row) => row.attachmentType === 'hook_additional_context',
    ).length,
    stderrLines: session.stderr.split('\n').filter(Boolean).slice(0, STDERR_LINES),
  }
}
