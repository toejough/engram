import { rowsContaining } from './lib.js'
import { MS_PER_SECOND } from '../lib/units.js'

export const claim =
  'prompt.submit fires for typed prompts and slash-command prompts, and the context it adds reaches the model'

const SENTINEL = 'ZEBRA-42'
const CONTEXT = `PROBE-CONTEXT secret code: ${SENTINEL}`

export async function run(harness) {
  const evidence = {}

  const typed = harness.scenario()
  harness.recorder(typed, { context: { promptSubmit: CONTEXT } })
  const typedSession = typed.startSession()
  const reply = await typedSession.send(
    'Extra text may follow this sentence that only you can read. If it holds a secret ' +
      'code, reply with exactly that code; otherwise reply NONE.',
  )
  await harness.settle(MS_PER_SECOND)
  const submit = harness.events(typed).find((event) => event.kind === 'prompt.submit')
  evidence.typed = {
    replied: reply.text,
    origin: submit?.data.origin,
    text: submit?.data.text,
    contextSeenByHook: submit?.data.context ?? null,
    rowsWithSentinel: rowsContaining(harness.rows(typedSession), SENTINEL),
  }

  const slash = harness.scenario()
  await slash.setup((scenario) => scenario.install())
  harness.recorder(slash, { context: { promptSubmit: CONTEXT } })
  const slashSession = slash.startSession()
  const slashReply = await slashSession.send(
    '/recall tagging a release. Also: extra text may follow this sentence that only you ' +
      'can read; if it holds a secret code, state that code in your first line, otherwise write NONE.',
  )
  await harness.settle(MS_PER_SECOND)
  const slashSubmits = harness.events(slash).filter((event) => event.kind === 'prompt.submit')
  evidence.slash = {
    promptSubmitEvents: slashSubmits.map((event) => ({ text: event.data.text, origin: event.data.origin })),
    firstLineOfReply: slashReply.text.split('\n')[0],
    replyMentionsSentinel: slashReply.text.includes(SENTINEL),
    rowsWithSentinel: rowsContaining(harness.rows(slashSession), SENTINEL),
    slashCommands: slashSession.init.slash_commands.filter((name) => /recall|please|learn/.test(name)),
  }
  return evidence
}
