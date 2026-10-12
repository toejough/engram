// Every judgement the mod asks of a model, in one place: the tier table, the prompts and the reading of the replies
// (793b279). The relay itself, `$.model.complete`, is called in register.js.
// A request names the model of its kind; a kind without a row is a programming error, never a default.
export const JUDGE_TIERS = { relevance: 'haiku' }
const JUDGE_TIMEOUT_MS = 20000
const REPLY_TOKENS = 200
const TEXT_LIMIT = 1500

// RELEVANCE is the whole of the delivery judgement: which candidate notes bear on the moment.
export const RELEVANCE = [
  'You decide which vault notes bear on what an AI coding agent is about to do.',
  'Moment: {moment}',
  'Text at the moment:\n{text}',
  'Candidate notes:\n{notes}',
  'A note bears on the moment when its situation describes the work in the text. If the text itself asks to run a',
  'memory command (recall, learn, an engram query), the user is retrieving on purpose: none bears.',
  'Reply with only a JSON array of the numbers of the notes that bear, [] if none.',
].join('\n')

export function relevanceRequest(moment, text, candidates) {
  const notes = candidates.map((note, index) => `${index + 1}. situation: ${note.situation}. ${note.body}`).join('\n')
  return request('relevance', RELEVANCE.replace('{moment}', moment).replace('{text}', text.slice(0, TEXT_LIMIT)).replace('{notes}', notes))
}

export function request(kind, prompt) {
  const model = JUDGE_TIERS[kind]
  if (!model) throw new Error(`no judgement tier for kind "${kind}"`)
  return { model, prompt, maxTokens: REPLY_TOKENS, effort: 'low', timeoutMs: JUDGE_TIMEOUT_MS }
}

// parseSelection reads the numbers of the reply; anything that is not an array of valid numbers selects nothing.
export function parseSelection(reply, count) {
  const match = /\[[^\]]*\]/.exec(reply)
  if (!match) return []
  try {
    const numbers = JSON.parse(match[0])
    return numbers.filter((number) => Number.isInteger(number) && number >= 1 && number <= count).map((number) => number - 1)
  } catch {
    return []
  }
}
