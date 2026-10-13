// Helpers the Then steps share (793e2b): the rows of the sessions after the System's work has settled.
import fs from 'node:fs'
import path from 'node:path'

import assert from 'node:assert/strict'

import { awaitIdle, settle } from '../lib/turn.js'
import { readRows } from '../lib/transcript.js'
import { AUTOMATIC, parseNote, subagentOf } from '../lib/vault.js'

export const NOT_BUILT = 'pending'
export const Q = '"([^"]*)"'

export async function mainRows(world) {
  const session = world.scenario.session
  if (!session) throw new Error('no session ran: a harness error, the scenario has no turn before this step')
  await settle()
  return readRows(session.transcriptPath())
}

export async function subagentRows(world) {
  await settle()
  return world.scenario.session.subagentTranscripts().map((file) => readRows(file))
}

export function sentinelOf(world, label) {
  const note = world.scenario.notes.get(label)
  if (!note) throw new Error(`no note "${label}" was seeded before this step`)
  return note.sentinel
}

// quiesce waits for the System's background work (the seam's idle line) and closes the sessions, so that what the
// System did is read, not raced.
export async function quiesce(world) {
  const scenario = world.scenario
  if (!scenario.session) throw new Error('no session ran: a harness error, the scenario has no turn before this step')
  await awaitIdle(scenario)
  await Promise.all(scenario.sessions.map((session) => session.close()))
  await settle()
}


export const APPLY = /\bapply\b/i

// A request is an instruction to run a memory command or to evaluate, approve, write or curate a note; the framing
// of a block (everything but its item lines) must hold none.
export const REQUEST = /\b(run|execute|invoke|call)\b[^.\n]*\b(engram|recall|learn|curate|write-memory|memory command)\b|\b(evaluate|approve|curate)\b|\bwrite (a|the) note\b/i


// The notes the System or the agent wrote: the vault's notes less the seeded ones, read once the System's work is idle
// and the sessions are closed.
export async function writtenNotes(world) {
  await quiesce(world)
  const scenario = world.scenario
  const seeded = [...scenario.notes.values()].map((note) => note.sentinel)
  return fs
    .readdirSync(scenario.vault)
    .filter((name) => name.endsWith('.md'))
    .map((name) => ({ name, text: fs.readFileSync(path.join(scenario.vault, name), 'utf8') }))
    .filter((note) => !seeded.some((sentinel) => note.text.includes(sentinel)))
    .map((note) => ({ ...note, ...parseNote(note.text) }))
}

// endedSourcesMarked: the backlog read every ended session the scenario wrote (the mod's mark file lists its source),
// so that "a lesson is not written again" is a judgement of the backlog and not a backlog that never ran (793b316).
export function endedSourcesMarked(world) {
  const scenario = world.scenario
  if (scenario.ended.length === 0) return
  const file = path.join(scenario.home, '.local', 'share', 'engram', 'chunks', 'backlog-marks.json')
  assert.ok(fs.existsSync(file), 'the backlog wrote no mark file: it never captured an ended session')
  const marked = Object.keys(JSON.parse(fs.readFileSync(file, 'utf8')).sources ?? {})
  for (const ended of scenario.ended) assert.ok(marked.includes(ended.file), `the mark file does not list the ${ended.harness} source ${ended.file}; it lists ${JSON.stringify(marked)}`)
}

// "stating L": scenarios.js gives the words that state each lesson (a pattern per lesson, matched against what a note says,
// never its source line), because a note is worded by a model. The check is that a note written to the vault says it;
// where the scenario says the System wrote it (`automatic`) the source says so, and where the lesson was only in a
// subagent's own session (`fromSubagent`) the source names that subagent.
export async function vaultHolds(world, quantity, lesson) {
  const pattern = world.data.lessons?.[lesson]
  if (!pattern) throw new Error(`scenarios.js has no words for the lesson "${lesson}" of ${world.id}`)
  const stating = (await writtenNotes(world)).filter((note) => pattern.test(note.content))
  const names = stating.map((note) => note.name).join(', ')
  const written = (await writtenNotes(world)).map((note) => `${note.name}: ${note.content.replace(/\s+/g, ' ').slice(0, 260)}`).join(' | ')
  if (quantity === 'a') assert.ok(stating.length > 0, `no note in the vault states "${lesson}"; the notes written were: ${written || 'none'}`)
  else assert.equal(stating.length, 1, `${stating.length} notes state "${lesson}", expected exactly one: ${names}; the notes written were: ${written || 'none'}`)
  if (world.data.automatic) {
    for (const note of stating) assert.match(note.source, AUTOMATIC, `the note ${note.name} was not written automatically (its source is "${note.source}")`)
  }
  if (world.data.fromSubagent) {
    const files = world.scenario.session.subagentTranscripts().map((file) => path.basename(file))
    for (const note of stating) {
      const agent = subagentOf(note.source)
      assert.ok(agent !== null, `the source of ${note.name} does not name a subagent: "${note.source}"`)
      assert.ok(files.some((file) => file.includes(agent)), `the subagent ${agent} named by ${note.name} is none of ${JSON.stringify(files)}`)
    }
  }
  endedSourcesMarked(world)
}

// lookupDeclined: where the scenario put a note in the vault that bears on nothing (`unrelatedNote`), the lookup of a
// prompt found it and the judgement of relevance declined it, which the mod says in the debug log of the session: "N
// candidate(s), 0 delivered" with N above zero. The path "matched but irrelevant" ran, not only "nothing matched".
export function lookupDeclined(world) {
  if (!world.data.unrelatedNote) return
  const lines = world.scenario.sessions.flatMap((session) => {
    try {
      return fs.readFileSync(session.debugFile, 'utf8').split('\n')
    } catch {
      return []
    }
  })
  const lookups = lines.map((line) => /engram-memory: \w+ lookup: (\d+) candidate\(s\), (\d+) delivered/.exec(line)).filter(Boolean)
  assert.ok(lookups.length > 0, 'the debug log of the session holds no lookup of the mod')
  assert.ok(lookups.some((found) => Number(found[1]) > 0 && Number(found[2]) === 0), `no lookup found the unrelated note and declined it: ${lookups.map((found) => found[0]).join('; ')}`)
}
