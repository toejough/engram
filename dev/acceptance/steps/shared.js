// Helpers the Then steps share (793e2b): the rows of the sessions after the System's work has settled.
import fs from 'node:fs'
import path from 'node:path'

import assert from 'node:assert/strict'

import { awaitIdle, settle } from '../lib/turn.js'
import { readRows } from '../lib/transcript.js'

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
}

// "stating L": scenarios.js gives the words that state the lesson (a pattern per lesson), because a note is worded by
// a model; the check is that a note written to the vault says it.
export async function vaultHolds(world, quantity, lesson) {
  const pattern = world.data.lessons?.[lesson]
  if (!pattern) throw new Error(`scenarios.js has no words for the lesson "${lesson}" of ${world.id}`)
  const stating = (await writtenNotes(world)).filter((note) => pattern.test(note.text))
  const names = stating.map((note) => note.name).join(', ')
  if (quantity === 'a') assert.ok(stating.length > 0, `no note in the vault states "${lesson}"`)
  else assert.equal(stating.length, 1, `${stating.length} notes state "${lesson}", expected exactly one: ${names}`)
}

