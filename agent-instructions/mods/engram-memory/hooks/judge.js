// Every judgement the mod asks of a model, in one place: the tier table, the prompts and the reading of the replies
// (793b279). The relay itself, `$.model.complete`, is called in register.js.
// A request names the model of its kind; a kind without a row is a programming error, never a default.
export const JUDGE_TIERS = { relevance: 'haiku', capture: 'haiku' }
const JUDGE_TIMEOUT_MS = { relevance: 20000, capture: 40000 }
const REPLY_TOKENS = { relevance: 200, capture: 1500 }
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

// CAPTURE is the whole of the capture judgement (793b279): extraction and reconciliation in one reply. The notes are
// the vault's nearest to the window; a lesson one of them already states is marked with its number, so nothing is written.
export const CAPTURE = [
  'You keep the long-term memory of an AI coding agent. Below is a conversation window and the vault notes nearest to it.',
  'Window:\n{window}',
  'Nearest notes:\n{notes}',
  'List each lesson the window contains: a correction by the user ("no, do X"), an explicit request to remember',
  'something, a mistake of the agent that later evidence overturned and the agent corrected, or an approach the user',
  'confirmed worked. Plain tasks, questions and chatter are not lessons.',
  'For each lesson give a JSON object with: kind ("feedback" for a rule about what to do or not to do, "fact" for',
  'something that is true), slug (kebab-case), situation (when it applies, as a retrieval cue), covered (the number',
  'of a nearest note that already states the same lesson, else null), and for feedback behavior, impact and action, for',
  'fact subject, predicate and object.',
  'Reply with only a JSON array, [] if the window has no lesson.',
].join('\n')

export function captureRequest(window, near) {
  const notes = near.length === 0 ? 'none' : near.map((note, index) => `${index + 1}. situation: ${note.situation}. ${note.body}`).join('\n')
  return request('capture', CAPTURE.replace('{window}', () => window).replace('{notes}', () => notes))
}

export function relevanceRequest(moment, text, candidates) {
  const notes = candidates.map((note, index) => `${index + 1}. situation: ${note.situation}. ${note.body}`).join('\n')
  return request('relevance', RELEVANCE.replace('{moment}', () => moment).replace('{text}', () => text.slice(0, TEXT_LIMIT)).replace('{notes}', () => notes))
}

export function request(kind, prompt) {
  const model = JUDGE_TIERS[kind]
  if (!model) throw new Error(`no judgement tier for kind "${kind}"`)
  return { model, prompt, maxTokens: REPLY_TOKENS[kind], effort: 'low', timeoutMs: JUDGE_TIMEOUT_MS[kind] }
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
