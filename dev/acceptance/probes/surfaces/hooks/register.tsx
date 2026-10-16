// The SP17 probe (793e8a): puts a marker on each user-facing surface a mod has, so that a run can see which are drawn
// on a screen that also has a configured statusLine, and which reach the transcript or the model. It never denies.
// The status and the toast are shown once the turn has finished, as the System's own summary is (793a5h).
import type { Register } from 'claude-code'

const TOAST_MS = 60000
const LOG_WORDS = 40

export const register: Register = (on) => {
  on('turn.complete', async ($, e, next) => {
    if (e.agentId !== undefined) return next(e)
    $.ui.status('PROBE-STATUS-A')
    $.ui.toast('PROBE-TOAST-B', { timeoutMs: TOAST_MS })
    // a transcript line: does it reach the model's input on the next turn (the api view of the session)?
    const out = await $.env.get('SP17_OUT')
    const api = JSON.stringify(await $.session.messages({ as: 'api' }))
    if (out) await $.fs.write(out, JSON.stringify({ apiHoldsLog: api.includes('PROBE-LOG-E'), apiChars: api.length }))
    $.ui.log(`PROBE-LOG-E ${'wrapword '.repeat(LOG_WORDS)}PROBE-LOG-END`)
    const result = await next(e)
    // the hook result: with SP17_TURN_TEXT set, the turn's text comes back with a marker line appended
    if (!(await $.env.get('SP17_TURN_TEXT'))) return result
    return { ...result, text: `${result.text}\nPROBE-TURN-C` }
  })

  on('ui.render', { component: 'AbovePrompt' }, async ($, e, next) => {
    const { Box, Text } = $.ui.resolve(e)
    return (
      <Box>
        <Text>PROBE-BAND-D</Text>
      </Box>
    )
  })
}
