export const claim = 'An isolated-HOME session authenticates from the credentials in its environment; a mod model.complete then uses it; without credentials the session answers Not logged in'

export async function run(h) {
  const evidence = {}

  const authenticated = h.scenario()
  const a = authenticated.startSession()
  const turn = await a.send('Reply with exactly the word: pong')
  evidence.withCredentials = { reply: turn.text, isError: turn.isError, apiKeySource: a.init.apiKeySource, version: a.init.claude_code_version, model: a.init.model }

  const anonymous = h.scenario()
  const b = anonymous.startSession({ env: anonymous.env() })
  const noAuth = await b.send('Reply with exactly the word: pong').catch((error) => ({ rejected: error.message.slice(0, 200) }))
  evidence.withoutCredentials = { reply: noAuth.text ?? null, isError: noAuth.isError ?? null, rejected: noAuth.rejected ?? null }

  const modded = h.scenario()
  h.recorder(modded, { model: { alias: 'haiku', prompt: 'Reply with the single word PONG' } })
  const c = modded.startSession()
  await c.send('Reply with exactly the word: ok')
  await h.settle(1000)
  const calls = h.events(modded).filter((event) => event.kind === 'model.complete')
  evidence.modModelComplete = calls.map((event) => ({ isAnswered: event.data?.isAnswered, text: event.data?.text, reason: event.data?.reason, usage: event.data?.usage }))
  return evidence
}
