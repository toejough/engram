import { MS_PER_SECOND, excerpt } from '../lib/units.js'

export const claim =
  'An isolated-HOME session authenticates from the credentials in its environment; ' +
  'a mod model.complete then uses it; without credentials the session answers Not logged in'

export async function run(harness) {
  const evidence = {}

  const authenticated = harness.scenario()
  const authenticatedSession = authenticated.startSession()
  const turn = await authenticatedSession.send('Reply with exactly the word: pong')
  evidence.withCredentials = {
    reply: turn.text,
    isError: turn.isError,
    apiKeySource: authenticatedSession.init.apiKeySource,
    version: authenticatedSession.init.claude_code_version,
    model: authenticatedSession.init.model,
  }

  const anonymous = harness.scenario()
  const anonymousSession = anonymous.startSession({ env: anonymous.env() })
  const noAuth = await anonymousSession
    .send('Reply with exactly the word: pong')
    .catch((error) => ({ rejected: excerpt(error.message) }))
  evidence.withoutCredentials = {
    reply: noAuth.text ?? null,
    isError: noAuth.isError ?? null,
    rejected: noAuth.rejected ?? null,
  }

  const modded = harness.scenario()
  harness.recorder(modded, { model: { alias: 'haiku', prompt: 'Reply with the single word PONG' } })
  const moddedSession = modded.startSession()
  await moddedSession.send('Reply with exactly the word: ok')
  await harness.settle(MS_PER_SECOND)
  const calls = harness.events(modded).filter((event) => event.kind === 'model.complete')
  evidence.modModelComplete = calls.map((event) => ({
    isAnswered: event.data?.isAnswered,
    text: event.data?.text,
    reason: event.data?.reason,
    usage: event.data?.usage,
  }))
  return evidence
}
