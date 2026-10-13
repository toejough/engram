// Delivering a prompt from each origin (793a2n1h, SP8). The human types it. A task notification, a scheduled trigger
// and a peer's message are produced by real mechanisms: a background command, a one-shot CronCreate, another
// session's mod. The setup prompt of those three never states the situation, which sits in a file the agent reads, so
// that the setup turn is not itself a moment for the note: the delivery under test is the one at the prompt of the origin.
import fs from 'node:fs'
import path from 'node:path'

import { RECORDER } from '../probes/lib.js'
import { attempt, constraintsHold, currentSession, readTranscript, sendPrompt, settle } from './turn.js'
import { promptRows } from './transcript.js'

const NOTIFICATION_WAIT_MS = 120000
const SCHEDULE_WAIT_MS = 150000
const NOT_ARRIVED = { system: false, message: 'the prompt of the origin did not arrive' }

export function promptFor(note, correction) {
  const text = `Next I am working on: ${note.situation}. Answer with one short sentence.`
  return correction ? `${text} ${correction}` : text
}

const BY_ORIGIN = {
  'the human': human,
  'an automated task notification': taskNotification,
  'a scheduled trigger': scheduledTrigger,
  'a message from another session': peerMessage,
}

// deliverPrompt delivers a prompt matching the note from the origin; the origins other than the human retry their
// precondition (the agent must set the mechanism up and the prompt must arrive, 793b246).
export async function deliverPrompt(world, origin, label, correction) {
  const note = world.scenario.notes.get(label)
  if (!note) throw new Error(`no note "${label}" was seeded before this step`)
  const deliver = BY_ORIGIN[origin]
  if (!deliver) throw new Error(`no driver for the origin "${origin}"`)
  world.origin = origin
  return deliver(world, note, correction)
}

// the human's prompt retries its precondition: a Given may forbid the agent its own memory commands (793e2i step 7)
async function human(world, note, correction) {
  const prompt = promptFor(note, correction)
  world.typed = prompt
  return attempt(
    world,
    (scenario) => sendPrompt(scenario, prompt),
    ({ rows }) => (constraintsHold(world, rows, world.data) ? null : { system: false, message: 'the agent ran a memory command itself' }),
  )
}

async function taskNotification(world, note) {
  return attempt(
    world,
    async (current) => {
      current.writeWorkspaceFile('task.txt', `command: sleep 8\ndescription: Background task: ${note.situation}\n`)
      const session = currentSession(current)
      const before = session.turns.length
      await session.send(
        'Read task.txt. Run its command as a background Bash task (run_in_background true) with exactly its description line as the Bash description. Then reply started.',
      )
      const turn = await session.awaitTurn({ after: before + 1, timeoutMs: NOTIFICATION_WAIT_MS }).catch(() => null)
      await settle()
      return { turn, rows: await readTranscript(session) }
    },
    ({ turn, rows }) => (turn !== null && promptRows(rows, 'task').length > 0 ? null : NOT_ARRIVED),
  )
}

async function scheduledTrigger(world, note) {
  return attempt(
    world,
    async (current) => {
      current.writeWorkspaceFile('scheduled.txt', `${promptFor(note)}\n`)
      const session = currentSession(current)
      const before = session.turns.length
      await session.send(
        'Read scheduled.txt. Use the CronCreate tool to schedule its contents, exactly, as a prompt to run once, one minute from now. Then reply scheduled.',
      )
      const turn = await session.awaitTurn({ after: before + 1, timeoutMs: SCHEDULE_WAIT_MS }).catch(() => null)
      await settle()
      return { turn, rows: await readTranscript(session) }
    },
    ({ turn, rows }) => (turn !== null && promptRows(rows, 'schedul').length + promptRows(rows, 'cron').length > 0 ? null : NOT_ARRIVED),
  )
}

async function peerMessage(world, note) {
  return attempt(
    world,
    async (current) => {
      current.linkMod(RECORDER, 'probe-recorder')
      const receiver = currentSession(current)
      const before = receiver.turns.length
      await receiver.send('Reply with the word ready.')
      const events = path.join(current.dir, 'peer-events')
      fs.mkdirSync(events, { recursive: true })
      const sender = current.startSession({
        current: false,
        env: {
          ...current.env({ withCredentials: true }),
          PROBE_DIR: events,
          PROBE_CONFIG: JSON.stringify({ sendTo: receiver.sessionId, sendText: promptFor(note) }),
        },
      })
      await sender.send('Reply with the word sent.')
      const turn = await receiver.awaitTurn({ after: before + 1, timeoutMs: NOTIFICATION_WAIT_MS }).catch(() => null)
      await settle()
      return { turn, rows: await readTranscript(receiver) }
    },
    ({ turn, rows }) => (turn !== null && promptRows(rows, 'peer').length > 0 ? null : NOT_ARRIVED),
  )
}
