import { rowsContaining } from './lib.js'

export const claim = 'Handler time inside $.process.run does not count against the 10 s hook budget for a 25 s job'
export const documentation = 'reference.md, Limits: "A hook\'s own execution time for one event, not counting time inside `next` or a mods API call other than `$.clock.sleep`: 10 seconds"'

const TOKEN = 'CTX-LONG-3b9a'

export async function run(h) {
  const scenario = h.scenario()
  h.recorder(scenario, { sleepSeconds: 25, context: { promptSubmit: TOKEN } })
  const session = scenario.startSession()
  const started = Date.now()
  const turn = await session.send('Reply with exactly the word: ok')
  const wallS = (Date.now() - started) / 1000
  await h.settle(1000)
  const events = h.events(scenario)
  const run = events.find((event) => event.kind === 'process.run')
  return {
    documentation,
    turnWallS: Math.round(wallS * 10) / 10,
    replyOk: !turn.isError && /ok/i.test(turn.text),
    processRun: run?.data,
    hookErrors: events.filter((event) => event.kind.endsWith('.error')).map((event) => event.data),
    contextAddedAfterTheWait: rowsContaining(h.rows(session), TOKEN).filter((row) => row.attachmentType === 'hook_additional_context').length,
    stderrLines: session.stderr.split('\n').filter(Boolean).slice(0, 3),
  }
}
