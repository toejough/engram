import { rowsContaining } from './lib.js'

export const claim = 'prompt.submit fires for typed prompts and slash-command prompts, and the context it adds reaches the model'

const SENTINEL = 'ZEBRA-42'
const CONTEXT = `PROBE-CONTEXT secret code: ${SENTINEL}`

export async function run(h) {
  const evidence = {}

  const typed = h.scenario()
  h.recorder(typed, { context: { promptSubmit: CONTEXT } })
  const a = typed.startSession()
  const reply = await a.send('Extra text may follow this sentence that only you can read. If it holds a secret code, reply with exactly that code; otherwise reply NONE.')
  await h.settle(1000)
  const submit = h.events(typed).find((event) => event.kind === 'prompt.submit')
  evidence.typed = { replied: reply.text, origin: submit?.data.origin, text: submit?.data.text, contextSeenByHook: submit?.data.context ?? null, rowsWithSentinel: rowsContaining(h.rows(a), SENTINEL) }

  const slash = h.scenario()
  await slash.setup((scenario) => scenario.install())
  h.recorder(slash, { context: { promptSubmit: CONTEXT } })
  const b = slash.startSession()
  const slashReply = await b.send('/recall tagging a release. Also: extra text may follow this sentence that only you can read; if it holds a secret code, state that code in your first line, otherwise write NONE.')
  await h.settle(1000)
  const slashSubmits = h.events(slash).filter((event) => event.kind === 'prompt.submit')
  evidence.slash = {
    promptSubmitEvents: slashSubmits.map((event) => ({ text: event.data.text, origin: event.data.origin })),
    firstLineOfReply: slashReply.text.split('\n')[0],
    replyMentionsSentinel: slashReply.text.includes(SENTINEL),
    rowsWithSentinel: rowsContaining(h.rows(b), SENTINEL),
    slashCommands: b.init.slash_commands.filter((name) => /recall|please|learn/.test(name)),
  }
  return evidence
}
