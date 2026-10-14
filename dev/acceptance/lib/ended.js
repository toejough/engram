// Fixtures for the ended sessions of node 793e8g (backlog capture): the transcript file a harness left behind when its
// session ended before the lesson was saved. They are inputs, written in the format of the harness that wrote them
// (Pi: SP14, no headless Pi on this machine; Claude Code: a session killed mid-turn), set back in time so that the
// session is quiet, and kept with a byte-identical copy for the scenarios that say the transcript is left unchanged.
import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'
import { MS_PER_SECOND } from './units.js'

const HOUR_MS = 3600000
const DEFAULT_AGE_HOURS = 2
const DEFAULT_AGE_MS = DEFAULT_AGE_HOURS * HOUR_MS
const PI_VERSION = 3

// slugOf is how Claude Code names the project directory of a working directory.
export function slugOf(directory) {
  return directory.replace(/[^A-Za-z0-9]/g, '-')
}

function stamp(base, index) {
  return new Date(base + index * MS_PER_SECOND).toISOString()
}

// piDirName is how Pi names the session directory of a working directory (read from ~/.pi/agent/sessions, 793b325):
// two dashes, the path without its first slash with a dash for each slash, two dashes. Not Claude Code's slug.
export function piDirName(directory) {
  return `--${directory.replace(/^[/\\]/, '').replace(/[/\\:]/g, '-')}--`
}

const PI_MODEL = { api: 'anthropic-messages', provider: 'anthropic', model: 'claude-sonnet-5-5' }
const PI_USAGE = {
  input: 1200,
  output: 80,
  cacheRead: 0,
  cacheWrite: 0,
  reasoning: 0,
  totalTokens: 1280,
  cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0, total: 0 },
}

// piRows is the file Pi (version 3) writes, in the shape of a real one: a session row, a model change, a thinking-level
// change, then one message row per message, each with a numeric timestamp inside the message; an assistant message
// holds thinking, text and tool-call blocks with its model, usage and stop reason, and a tool call is answered by a
// message whose role is toolResult. The first assistant turn runs one shell command, so that the file holds the roles
// the backlog must not read as the conversation. `turns` are the user's and the agent's words, in order.
function piRows({ id, cwd, startedAt, turns }) {
  const rows = [
    { type: 'session', version: PI_VERSION, id, timestamp: stamp(startedAt, 0), cwd },
    {
      type: 'model_change',
      id: 'mc0',
      parentId: null,
      timestamp: stamp(startedAt, 0),
      provider: PI_MODEL.provider,
      modelId: PI_MODEL.model,
    },
    { type: 'thinking_level_change', id: 'tl0', parentId: 'mc0', timestamp: stamp(startedAt, 0), thinkingLevel: 'off' },
  ]
  let parent = 'tl0'
  let tick = 0
  const add = (rowId, message) => {
    tick += 1
    rows.push({
      type: 'message',
      id: rowId,
      parentId: parent,
      timestamp: stamp(startedAt, tick),
      message: { ...message, timestamp: startedAt + tick * MS_PER_SECOND },
    })
    parent = rowId
  }
  let ran = false
  turns.forEach(([role, text], index) => {
    if (role === 'user') return add(`u${index}`, { role, content: [{ type: 'text', text }] })
    const assistant = (content, stopReason) => ({
      role,
      content: [{ type: 'thinking', thinking: '', thinkingSignature: 'sig' }, ...content],
      ...PI_MODEL,
      usage: PI_USAGE,
      stopReason,
      responseId: `resp-${index}`,
    })
    if (ran) return add(`a${index}`, assistant([{ type: 'text', text }], 'stop'))
    ran = true
    add(
      `a${index}`,
      assistant(
        [
          { type: 'text', text },
          { type: 'toolCall', id: `call-${index}`, name: 'bash', arguments: { command: 'go test ./...' } },
        ],
        'toolUse',
      ),
    )
    add(`r${index}`, {
      role: 'toolResult',
      toolCallId: `call-${index}`,
      toolName: 'bash',
      content: [{ type: 'text', text: 'ok  \texample.com/work\t0.4s' }],
      isError: false,
    })
    return undefined
  })
  return rows
}

function claudeRows({ id, cwd, startedAt, turns }) {
  let parent = null
  return turns.map(([role, text], index) => {
    const uuid = crypto.randomUUID()
    const row = {
      parentUuid: parent,
      isSidechain: false,
      type: role,
      message: { role, content: role === 'user' ? text : [{ type: 'text', text }] },
      uuid,
      timestamp: stamp(startedAt, index + 1),
      sessionId: id,
      cwd,
      version: '2.1.296',
    }
    parent = uuid
    return row
  })
}

// turnsOf: the conversation of the ended session, from the scenario's inputs: an opening, what the user said, and (a
// session that ended normally) the agent's last reply. A session killed mid-turn ends with what the user said.
export function turnsOf({ opening, closing }, said) {
  const turns = [...opening, ['user', said]]
  if (closing) turns.push(['assistant', closing])
  return turns
}

// writeEnded writes the transcript of an ended session into the directory its harness sweeps, back-dates it, and
// keeps a copy beside the scenario. Returns what the Thens compare against.
export function writeEnded(scenario, { harness, turns, ageMs = DEFAULT_AGE_MS }) {
  const id = crypto.randomUUID()
  const startedAt = Date.now() - ageMs - HOUR_MS
  const cwd = fs.realpathSync(scenario.workspace)
  let file
  let rows
  if (harness === 'Pi') {
    const dir = path.join(scenario.home, '.pi', 'agent', 'sessions', piDirName(cwd))
    fs.mkdirSync(dir, { recursive: true })
    file = path.join(fs.realpathSync(dir), `${stamp(startedAt, 0).replace(/[:.]/g, '-')}_${id}.jsonl`)
    rows = piRows({ id, cwd, startedAt, turns })
  } else {
    const dir = path.join(scenario.home, '.claude', 'projects', slugOf(cwd))
    fs.mkdirSync(dir, { recursive: true })
    file = path.join(fs.realpathSync(dir), `${id}.jsonl`)
    rows = claudeRows({ id, cwd, startedAt, turns })
  }
  fs.writeFileSync(file, `${rows.map((row) => JSON.stringify(row)).join('\n')}\n`)
  const when = new Date(Date.now() - ageMs)
  fs.utimesSync(file, when, when)
  const copy = path.join(scenario.dir, `ended-${harness === 'Pi' ? 'pi' : 'claude'}-${id}.copy`)
  fs.copyFileSync(file, copy)
  fs.utimesSync(copy, when, when)
  const ended = { harness, id, file, copy, mtimeMs: fs.statSync(file).mtimeMs }
  scenario.ended.push(ended)
  return ended
}
