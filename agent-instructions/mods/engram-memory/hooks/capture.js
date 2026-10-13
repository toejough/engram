// The pure parts of the capture job (793b199, 793b200, 793b279): the window of a turn or a subagent, the redaction, the
// reading of the judgement and the arguments of the writes. Every call on `$` is in register.js.
import { parseDocument } from './judge.js'
import { MARKER } from './moment.js'

const PART_LIMIT = 1200
const WINDOW_LIMIT = 8000
const NEAR_LIMIT = 8
const BACKLOG_WINDOWS = 6
const SOURCE_NOTE = 'captured from the conversation'
const REDACTED = '[redacted]'

// Secrets never reach a model request or a note (793b199, 793b298). Each part of the window is redacted before it is
// cut or joined, and the window and the judge's fields again. The patterns: credentials with a known shape (private
// keys, JWTs, provider keys, bearer and basic headers, passwords in URLs), a value that follows a name such as token,
// secret, password or api key (quoted with spaces, after `is`, or in JSON), a long unbroken run of letters and digits
// (an encoded secret), a run of wrapped lines of such characters (a key split across lines), a percent-encoded token
// whose decoding is one of these. What no pattern describes (a short secret in prose, a base64 value that is not long)
// is the residual risk: the judge is also told never to write a credential.
const NAMED = /([\w.-]*(?:token|secret|passw(?:or)?d|pwd|api[-_ ]?key|apikey|credential|private[-_ ]?key|authorization)[\w.-]*["']?\s*(?::|=|\bis\b)\s*)("[^"\n]*"|'[^'\n]*'|\S+)/gi
const SECRETS = [
  /-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?(-----END [A-Z ]*PRIVATE KEY-----|$)/g,
  /\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}/g,
  /\b(?:sk|pk|rk)[-_][A-Za-z0-9_-]{16,}/g,
  /\b(?:ghp|gho|ghs|ghu|ghr|github_pat|npm|glpat)[-_][A-Za-z0-9_-]{16,}/g,
  /\b(?:AKIA|ASIA)[0-9A-Z]{16}\b/g,
  /\bAIza[0-9A-Za-z_-]{30,}/g,
  /\bxox[abprs]-[A-Za-z0-9-]{10,}/g,
  /\b(?:Bearer|Basic)\s+[A-Za-z0-9._~+/=-]{12,}/gi,
]
const URL_USERINFO = /\b([a-z][a-z0-9+.-]*:\/\/)[^\s:@/]+:[^\s@/]+@/gi
const LONG_TOKEN = /[A-Za-z0-9+_=]{32,}/g
const WRAPPED_LINES = /(?:^[ \t]*[A-Za-z0-9+/=_-]{20,}[ \t]*\r?\n){2,}/gm
const WRAPPED_TOKEN = /([A-Za-z0-9_+/=-]{12,})[ \t]*\r?\n[ \t]*(?=[A-Za-z0-9_+/=-]{4,})/g
const PERCENT_TOKEN = /\S*%[0-9A-Fa-f]{2}\S*/g

const mixed = (text) => /\d/.test(text) && /[A-Za-z]/.test(text)

function redactOnce(text) {
  let result = text
  for (const pattern of SECRETS) result = result.replace(pattern, REDACTED)
  result = result.replace(URL_USERINFO, `$1${REDACTED}@`)
  result = result.replace(NAMED, (_match, name) => `${name}${REDACTED}`)
  result = result.replace(WRAPPED_LINES, (run) => (mixed(run) ? `${REDACTED}\n` : run))
  return result.replace(LONG_TOKEN, (token) => (mixed(token) ? REDACTED : token))
}

function decoded(token) {
  try {
    return decodeURIComponent(token)
  } catch {
    return token
  }
}

export function redact(text) {
  // a token wrapped over lines is judged joined; the joined text is kept only when it held a secret
  const joined = text.replace(WRAPPED_TOKEN, '$1')
  const base = redactOnce(joined) === joined ? text : joined
  const result = redactOnce(base)
  return result.replace(PERCENT_TOKEN, (token) => (redactOnce(decoded(token)) === decoded(token) ? token : REDACTED))
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

// oneLine keeps a message on one line of the window, so that every line begins with a label this module wrote: a
// label or a tag inside a message's own words is part of that message and cannot pose as another speaker.
const oneLine = (text) => text.replace(/\s*[\r\n]+\s*/g, ' \u23CE ')

// render turns a message (the shape of `$.session.messages()`) into the lines the judge reads. The speaker is the
// message's role, a structured field, never anything in its text. Only what the user and the agent said is read
// (793b297): a tool's input and result are untrusted content (a file, a web page, another program's output), so they
// never enter the window, only the name of the tool that was used. Every part is redacted before it is cut.
function render(message) {
  const lines = []
  if (message.text?.trim()) {
    const said = message.role === 'user' ? userText(message.text) : message.text.trim()
    lines.push(`${message.role === 'user' ? 'User' : 'Agent'}: ${cut(oneLine(redact(said)))}`)
  }
  for (const use of message.toolUses ?? []) lines.push(`Agent used ${oneLine(String(use.tool))}`)
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

// What a harness puts in a user turn that the user did not say: injected reminders, a skill's body, the output of a
// local command. It is not the user's words.
const HARNESS_TEXT = /^\s*(?:<system-reminder|<local-command-|Base directory for this skill:)/

const TOOL_USE = new Set(['tool_use', 'toolCall'])

// rowMessage is the speaker and content of a transcript row: a Claude Code row (`type` user or assistant, the content
// in `message`) or a Pi row (`type` message, its role in `message.role`). Any other row, and any other role (a tool
// result, program output a harness records as a message), is nobody's words.
function rowMessage(row) {
  if (row?.type === 'user' || row?.type === 'assistant') return { role: row.type, content: row.message?.content }
  if (row?.type === 'message' && (row.message?.role === 'user' || row.message?.role === 'assistant')) return { role: row.message.role, content: row.message.content }
  return null
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
    const found = rowMessage(row)
    if (!found) continue
    const blocks = blocksOf(found.content)
    const text = blocks.filter((block) => block.type === 'text' && !(found.role === 'user' && HARNESS_TEXT.test(block.text))).map((block) => block.text).join('\n')
    const toolUses = blocks.filter((block) => TOOL_USE.has(block.type)).map((block) => ({ tool: block.name }))
    const toolResults = blocks
      .filter((block) => block.type === 'tool_result')
      .map((block) => ({ text: typeof block.content === 'string' ? block.content : (block.content ?? []).map((part) => part.text ?? '').join('\n') }))
    messages.push({ role: found.role, text, toolUses, toolResults })
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

// windowsFromMessages is a whole conversation as capture windows: each holds whole messages up to the window limit and
// starts with the last message of the one before (a correction needs the message it corrects), at most `max` of them;
// what lies beyond is left, a known limit.
export function windowsFromMessages(messages, max = BACKLOG_WINDOWS) {
  const groups = messages.map(render).filter((lines) => lines.length > 0)
  const windows = []
  let current = []
  let size = 0
  for (const group of groups) {
    const length = group.join('\n').length + 1
    if (current.length > 0 && size + length > WINDOW_LIMIT) {
      if (windows.length + 1 === max) return [...windows, current.flat().join('\n')]
      windows.push(current.flat().join('\n'))
      current = [current.at(-1)]
      size = current[0].join('\n').length + 1
    }
    current.push(group)
    size += length
  }
  return current.length > 0 ? [...windows, current.flat().join('\n')] : windows
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

const STRING_LIMIT = 600
const LESSON_LIMIT = 10
const COMMON_KEYS = ['kind', 'slug', 'situation', 'covered']

function text(value, name) {
  if (typeof value !== 'string' || value.trim() === '' || value.length > STRING_LIMIT) throw new Error(`capture reply: ${name} is not a short non-empty string`)
  return value.trim()
}

// parseLessons reads the judge's reply strictly (793b299): exactly one JSON document, an array of at most ten lessons,
// each an object with exactly the keys of its kind (`feedback`: behavior, impact, action; `fact`: subject, predicate,
// object, beside kind, slug, situation and covered), every string non-empty and bounded, `covered` null or the number
// of a nearest note. Anything else is an error for the whole reply and nothing is written.
export function parseLessons(reply, near) {
  const list = parseDocument(reply)
  if (!Array.isArray(list) || list.length > LESSON_LIMIT) throw new Error('capture reply is not an array of at most ten lessons')
  const seen = new Set()
  const lessons = []
  for (const item of list) {
    if (item === null || typeof item !== 'object' || Array.isArray(item)) throw new Error('capture reply: a lesson is not an object')
    const fields = KINDS[item.kind]
    if (!fields) throw new Error('capture reply: a lesson has a kind that is neither feedback nor fact')
    const allowed = new Set([...COMMON_KEYS, ...fields])
    if (Object.keys(item).some((key) => !allowed.has(key)) || [...allowed].some((key) => !(key in item))) throw new Error(`capture reply: a ${item.kind} lesson does not have exactly its keys`)
    if (item.covered !== null && !(Number.isInteger(item.covered) && item.covered >= 1 && item.covered <= near.length)) throw new Error('capture reply: covered is no nearest note')
    const slug = slugOf(text(item.slug, 'slug'))
    if (seen.has(slug)) continue
    seen.add(slug)
    const values = Object.fromEntries(fields.map((name) => [name, redact(text(item[name], name))]))
    lessons.push({ kind: item.kind, slug, situation: redact(text(item.situation, 'situation')), covered: item.covered === null ? null : near[item.covered - 1].path, values })
  }
  return lessons
}

export function sourceFor(sessionId, turn) {
  return `${SOURCE_NOTE}; written automatically (session ${sessionId}, turn ${turn})`
}

// backlogSourceFor names a note written from a transcript read after the fact: the source and the position in it.
export function backlogSourceFor(path, position) {
  return `${SOURCE_NOTE}; written automatically (source ${path}, position ${position})`
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
