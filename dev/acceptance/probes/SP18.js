import { MS_PER_SECOND } from '../lib/units.js'

const KIB = 1024
// the sizes asked for double from 64 KiB to 64 MiB
const FIRST_KIB = 64
const DOUBLINGS = 11
const SIZES = Array.from({ length: DOUBLINGS }, (_, step) => (FIRST_KIB * KIB) << step)

export const claim = '$.process.run resolves the whole standard output of a program, whatever its size'
export const documentation =
  'api.md: "`$.process.run` ... resolves to `{ exitCode, stdout, stderr }` whatever the exit code": no limit on the ' +
  'size of stdout is documented (793b467)'

export async function run(harness) {
  const scenario = harness.scenario()
  harness.recorder(scenario, { stdoutBytes: SIZES })
  const session = scenario.startSession()
  const turn = await session.send('Reply with exactly the word: ok')
  await harness.settle(MS_PER_SECOND)
  const events = harness.events(scenario)
  return {
    documentation,
    replyOk: !turn.isError && /ok/i.test(turn.text),
    stdout: events.filter((event) => event.kind === 'process.stdout').map((event) => event.data),
    errors: events.filter((event) => event.kind.endsWith('.error')).map((event) => event.data),
  }
}
