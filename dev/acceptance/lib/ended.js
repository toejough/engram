// Fixtures for the ended sessions of node 793e8g (backlog capture): the transcript file a harness left behind when its
// session ended before the lesson was saved. They are inputs, written in the format of the harness that wrote them
// (Pi: SP14, no headless Pi on this machine; Claude Code: a session killed mid-turn), set back in time so that the
// session is quiet, and kept with a byte-identical copy for the scenarios that say the transcript is left unchanged.
import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'

const HOUR_MS = 3600 * 1000
const DEFAULT_AGE_MS = 2 * HOUR_MS
const PI_VERSION = 3

// slugOf is how Claude Code names the project directory of a working directory.
export function slugOf(directory) {
  return directory.replace(/[^A-Za-z0-9]/g, '-')
}

function stamp(base, index) {
  return new Date(base + index * 1000).toISOString()
}

function piRows({ id, cwd, startedAt, turns }) {
  const rows = [{ type: 'session', version: PI_VERSION, id, timestamp: stamp(startedAt, 0), cwd }]
  let parent = null
  turns.forEach(([role, text], index) => {
    const rowId = `${role[0]}${index}`
    rows.push({
      type: 'message',
      id: rowId,
      parentId: parent,
      timestamp: stamp(startedAt, index + 1),
      message: { role, content: [{ type: 'text', text }] },
    })
    parent = rowId
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
    const dir = path.join(scenario.home, '.pi', 'agent', 'sessions', slugOf(cwd))
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
