// The steps of node 793e8g (backlog capture): the ended sessions a later turn captures, and the Thens about the
// transcripts of other harnesses (they are inputs here: no headless Pi runs on this machine, SP14).
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'

import { Given } from '@cucumber/cucumber'

import { turnsOf, writeEnded } from '../lib/ended.js'
import { parseNote } from '../lib/vault.js'
import { endedSourcesMarked } from './shared.js'
import { MARKER } from '../lib/transcript.js'
import { awaitIdle, runTurn } from '../lib/turn.js'

// notesStating: the notes in the vault whose text matches the words of the lesson.
function notesStating(scenario, pattern) {
  if (!fs.existsSync(scenario.vault)) return []
  return fs
    .readdirSync(scenario.vault)
    .filter((name) => name.endsWith('.md'))
    .filter((name) => pattern.test(parseNote(fs.readFileSync(path.join(scenario.vault, name), 'utf8')).content))
}

function lessonPattern(world, input) {
  const pattern = world.data.lessons?.[input.lesson]
  if (!pattern) throw new Error(`scenarios.js has no words for the lesson "${input.lesson}" of ${world.id}`)
  return pattern
}

// The note the agent of an ended session wrote by running the learn skill (793a3a1j): a real `engram learn`.
async function agentSavedTheLesson(scenario, note) {
  await scenario.engram([
    'learn',
    'feedback',
    '--slug',
    note.slug,
    '--position',
    'top',
    '--source',
    note.source,
    `--situation=${note.situation}`,
    `--behavior=${note.behavior}`,
    `--impact=${note.impact}`,
    `--action=${note.action}`,
  ])
}

const SAVED = new RegExp(
  'its lesson was not saved|its agent saved the lesson by running the learn skill|' +
    'its lesson was captured by an earlier Claude Code turn',
)

Given(
  new RegExp(`^an ended (Pi|Claude Code) session in which the user said "([^"]*)" and (${SAVED.source})$`),
  async function (harness, said, saved) {
    const input = this.data.ended?.[said]
    if (!input) throw new Error(`scenarios.js has no ended session for ${this.id}: "${said}"`)
    const pattern = lessonPattern(this, input)
    assert.deepEqual(notesStating(this.scenario, pattern), [], `a note already states the lesson of "${said}"`)
    if (saved === 'its lesson was captured by an earlier Claude Code turn') {
      // the earlier turn is a real one, run first and settled, so that the System has written its note
      await runTurn(this, this.data.earlierTurn)
      await awaitIdle(this.scenario)
      assert.ok(
        notesStating(this.scenario, pattern).length > 0,
        'the earlier Claude Code turn captured no note: a precondition of the scenario did not hold',
      )
    }
    await this.scenario.setup(async (scenario) => {
      writeEnded(scenario, { harness, turns: turnsOf(input, said) })
      if (saved === 'its agent saved the lesson by running the learn skill')
        await agentSavedTheLesson(scenario, this.data.agentNote)
    })
  },
)

function piFixture(world) {
  const ended = world.scenario.ended.find((item) => item.harness === 'Pi')
  if (!ended) throw new Error('no ended Pi session was written before this step: a harness error, a Given is missing')
  return ended
}

// The System only reads a harness's transcript: the bytes and the modification time are what the Given left.
export async function piTranscriptUnchanged(world) {
  await awaitIdle(world.scenario)
  const ended = piFixture(world)
  assert.ok(
    Buffer.compare(fs.readFileSync(ended.file), fs.readFileSync(ended.copy)) === 0,
    'the Pi transcript differs from its copy',
  )
  assert.equal(fs.statSync(ended.file).mtimeMs, ended.mtimeMs, 'the Pi transcript was modified')
  endedSourcesMarked(world)
}

export function piTranscriptHoldsNoDelivery(world) {
  const text = fs.readFileSync(piFixture(world).file, 'utf8')
  assert.ok(!text.includes(MARKER), 'the Pi transcript holds content the System delivered')
}
