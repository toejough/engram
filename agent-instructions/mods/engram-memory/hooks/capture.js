// The pure parts of the capture job (793b199, 793b200, 793b279): the window of a turn or a subagent, the redaction, the
// reading of the judgement and the arguments of the writes. Every call on `$` is in register.js.
import { MARKER } from './moment.js'

const PART_LIMIT = 1200
const WINDOW_LIMIT = 8000
const TOOL_INPUT_LIMIT = 200
const NEAR_LIMIT = 8
const RESULT_LIMIT = 300
const SOURCE_NOTE = 'captured from the conversation'
const REDACTED = '[redacted]'

// Secrets never reach a model request or a note (793b199): the patterns are credentials with a known shape, private
// keys, and a value that follows a name such as token, secret, password or api key.
const SECRETS = [
  /-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?(-----END [A-Z ]*PRIVATE KEY-----|$)/g,
  /\b(sk|pk|rk)-[A-Za-z0-9_-]{16,}/g,
  /\b(ghp|gho|ghs|ghu|github_pat)_[A-Za-z0-9_]{20,}/g,
  /\bAKIA[0-9A-Z]{16}\b/g,
  /\bxox[abprs]-[A-Za-z0-9-]{10,}/g,
  /\bBearer\s+[A-Za-z0-9._~+/=-]{16,}/gi,
  /\b([A-Za-z_]*(token|secret|password|passwd|api[_-]?key)[A-Za-z_]*)\s*[=:]\s*\S+/gi,
]

export function redact(text) {
  let result = text
  for (const pattern of SECRETS) result = result.replace(pattern, (match, name) => (name && /[=:]/.test(match) ? `${name}=${REDACTED}` : REDACTED))
  return result
}

// stripOwnBlocks removes what the System delivered itself: a block opens with the fixed marker line and runs to the
// next blank line, so that capture never records its own deliveries.
export function stripOwnBlocks(text) {
  const kept = []
  let inBlock = false
  for (const line of text.split('\n')) {
    if (line.startsWith(MARKER)) inBlock = true
    else if (line.trim() === '') inBlock = false
    if (!inBlock) kept.push(line)
  }
  return kept.join('\n')
}

function cut(text, limit = PART_LIMIT) {
  return text.length > limit ? `${text.slice(0, limit)}...` : text
}

// userText: what a user message says. A task notification is not the user speaking: only its summary is kept, because
// its result is a subagent's reply, which that subagent's own capture has judged (and the notification is markup).
function userText(text) {
  const summary = /<summary>([\s\S]*?)<\/summary>/.exec(text)?.[1]
  return text.trimStart().startsWith('<task-notification>') ? `Task notification: ${(summary ?? 'a task finished').trim()}` : text.trim()
}

// render turns a message (the shape of `$.session.messages()`) into the lines the judge reads. A tool's result is
// read from the message that carries it, so a call's text is not repeated.
function render(message) {
  const lines = []
  if (message.text?.trim()) lines.push(`${message.role === 'user' ? 'User' : 'Agent'}: ${cut(message.role === 'user' ? userText(message.text) : message.text.trim())}`)
  for (const use of message.toolUses ?? []) lines.push(`Agent ran ${use.tool}: ${cut(JSON.stringify(use.input ?? {}).slice(0, TOOL_INPUT_LIMIT))}`)
  for (const result of message.toolResults ?? []) if (result.text?.trim()) lines.push(`Result: ${cut(result.text.trim(), RESULT_LIMIT)}`)
  return lines
}

function join(lines) {
  return lines.join('\n').slice(0, WINDOW_LIMIT)
}

// isPrompt: a message the user typed, not a tool result.
function isPrompt(message) {
  return message.role === 'user' && (message.text ?? '').trim() !== '' && !(message.toolResults ?? []).length
}

// windowFromMessages is the turn: the assistant message before the turn's last human prompt (a correction needs the
// message it corrects), the prompt, and everything after it. The answer is appended when the messages lag behind it.
export function windowFromMessages(messages, answer = '') {
  let start = -1
  messages.forEach((message, index) => {
    if (isPrompt(message)) start = index
  })
  if (start < 0) return ''
  const from = start > 0 && messages[start - 1].role === 'assistant' ? start - 1 : start
  const lines = messages.slice(from).flatMap(render)
  if (answer.trim() && !lines.some((line) => line.includes(answer.trim().slice(0, 80)))) lines.push(`Agent: ${cut(answer.trim())}`)
  return join(lines)
}

function blocksOf(content) {
  if (typeof content === 'string') return [{ type: 'text', text: content }]
  return Array.isArray(content) ? content : []
}

// messagesFromTranscript reads the rows of a transcript file in the shape of `$.session.messages()`.
export function messagesFromTranscript(fileText) {
  const messages = []
  for (const line of fileText.split('\n')) {
    let row
    try {
      row = JSON.parse(line)
    } catch {
      continue
    }
    if (row?.type !== 'user' && row?.type !== 'assistant') continue
    const blocks = blocksOf(row.message?.content)
    const text = blocks.filter((block) => block.type === 'text').map((block) => block.text).join('\n')
    const toolUses = blocks.filter((block) => block.type === 'tool_use').map((block) => ({ tool: block.name, input: block.input }))
    const toolResults = blocks
      .filter((block) => block.type === 'tool_result')
      .map((block) => ({ text: typeof block.content === 'string' ? block.content : (block.content ?? []).map((part) => part.text ?? '').join('\n') }))
    messages.push({ role: row.type, text, toolUses, toolResults })
  }
  return messages
}

// transcriptHolds: the subagent's file is not complete when the stop event fires (SP5); it is complete when an
// assistant row holds the last message.
export function transcriptHolds(fileText, lastMessage) {
  if (!lastMessage) return true
  const needle = lastMessage.trim().slice(0, 40)
  return messagesFromTranscript(fileText).some((message) => message.role === 'assistant' && message.text.includes(needle))
}

// windowFromTranscript is a subagent's whole transcript, with the last message appended when the file lags.
export function windowFromTranscript(fileText, lastMessage = '') {
  const lines = messagesFromTranscript(fileText).flatMap(render)
  if (lastMessage.trim() && !transcriptHolds(fileText, lastMessage)) lines.push(`Agent: ${cut(lastMessage.trim())}`)
  return join(lines)
}

// queryText is what the nearest notes are looked up with: what the user said first, then the rest.
export function queryText(window) {
  const lines = window.split('\n')
  return [...lines.filter((line) => line.startsWith('User:')), ...lines.filter((line) => !line.startsWith('User:'))].join(' ')
}

// neighbours: the notes the judge compares the window with: the vault's nearest by `engram query`, then the notes this
// session already wrote, so that a lesson it recorded is never recorded twice whatever the query ranked.
export function neighbours(items, recent = []) {
  const near = items.filter((item) => item.kind !== 'chunk' && item.situation !== '').slice(0, NEAR_LIMIT)
  const paths = new Set(near.map((item) => item.path))
  return [...near, ...recent.filter((note) => !paths.has(note.path))].slice(0, NEAR_LIMIT)
}

// recentNote is what a write leaves for the notes the session wrote: `engram learn` prints the path of the note.
export function recentNote(lesson, learnOutput) {
  const path = learnOutput.trim().split('\n').at(-1)?.split('/').at(-1) ?? ''
  if (!path.endsWith('.md')) return null
  return { path, situation: lesson.situation, body: Object.values(lesson.values).join('. ') }
}

const KINDS = {
  feedback: ['behavior', 'impact', 'action'],
  fact: ['subject', 'predicate', 'object'],
}

function slugOf(text) {
  return text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 60) || 'captured-lesson'
}

// parseLessons reads the judge's reply: a JSON array, each lesson with its kind, its fields and the number of the
// nearest note that already states it (`covered`) or null. A lesson that is incomplete, or covered by a number that is
// no note, is dropped or written as absent; a reply that is not an array is no lesson.
export function parseLessons(reply, near) {
  const start = reply.indexOf('[')
  const end = reply.lastIndexOf(']')
  if (start < 0 || end < start) return []
  let list
  try {
    list = JSON.parse(reply.slice(start, end + 1))
  } catch {
    return []
  }
  const seen = new Set()
  const lessons = []
  for (const item of Array.isArray(list) ? list : []) {
    const fields = KINDS[item?.kind]
    if (!fields || typeof item.situation !== 'string' || item.situation.trim() === '') continue
    if (item.kind === 'fact' && fields.some((name) => typeof item[name] !== 'string' || item[name].trim() === '')) continue
    if (item.kind === 'feedback' && (typeof item.behavior !== 'string' || typeof item.action !== 'string' || !item.behavior.trim() || !item.action.trim())) continue
    const slug = slugOf(typeof item.slug === 'string' && item.slug !== '' ? item.slug : item.situation)
    if (seen.has(slug)) continue
    seen.add(slug)
    const covered = Number.isInteger(item.covered) && item.covered >= 1 && item.covered <= near.length ? near[item.covered - 1].path : null
    const values = Object.fromEntries(fields.map((name) => [name, redact(String(item[name] ?? '').trim())]))
    if (item.kind === 'feedback' && values.impact === '') values.impact = 'the lesson had to be given again'
    lessons.push({ kind: item.kind, slug, situation: redact(item.situation.trim()), covered, values })
  }
  return lessons
}

export function sourceFor(sessionId, turn) {
  return `${SOURCE_NOTE}; written automatically (session ${sessionId}, turn ${turn})`
}

// learnArgv is the existing `engram learn` for an absent lesson; a free-text value is `--flag=value` so that one
// that starts with a dash is not read as a flag.
export function learnArgv(lesson, source) {
  const argv = ['engram', 'learn', lesson.kind, '--slug', lesson.slug, '--position', 'top', `--source=${source}`, `--situation=${lesson.situation}`]
  for (const [name, value] of Object.entries(lesson.values)) argv.push(`--${name}=${value}`)
  return argv
}

// amendArgv reinforces the note that already states a lesson.
export function amendArgv(path) {
  return ['engram', 'amend', '--target', path, '--activate']
}
