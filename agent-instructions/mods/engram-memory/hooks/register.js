// The System's mod (793e2c): maps the events of the delivery moments to the moment job and adds what it returns
// through the channel each event has: prompt context, tool-result context, subagent-start context (793b192).
import {
  amendArgv, learnArgv, neighbours, parseLessons, queryText, recentNote, redact, sourceFor, stripOwnBlocks, transcriptHolds,
  windowFromMessages, windowFromTranscript,
} from './capture.js'
import { captureRequest, relevanceRequest, parseSelection } from './judge.js'
import { candidates, buildBlock, momentText, parseItems, present, queryArgv, sinceCompaction } from './moment.js'

const MAIN = 'main'
// a subagent's transcript file lags its stop event (SP5): it is polled this often, this many times
const POLL_MS = 500
const POLL_TRIES = 20
// the capture queue (793b203): one job at a time, in order, so that the capture of a later event sees the notes of an
// earlier one; the tail never rejects
// the dispatch prompts of agent.spawn, in order, for the SubagentStart events that follow (SP5: start carries no prompt)
const spawns = []
const ledgers = new Map()
const queue = { tail: Promise.resolve(), pending: 0, seam: [] }
// the notes this session wrote (basename, situation and text), kept to compare later windows with
const written = []
const state = {
  transcript: null,
  lastPrompt: '',
  ledger(agent) {
    if (!ledgers.has(agent)) ledgers.set(agent, new Set())
    return ledgers.get(agent)
  },
}

// attempt contains a failure of the job: the event goes on as if the mod were not there (793a4b).
async function attempt($, job) {
  try {
    return await job()
  } catch (error) {
    $.ui.log(`engram-memory: ${error.message}`, { to: 'debug' })
    return null
  }
}

// readTranscript reads the session's transcript file; the file does not exist before the first row is written.
async function readTranscript($, file) {
  try {
    return await $.fs.read(file)
  } catch {
    return ''
  }
}

// moment runs one lookup (793b192): query, presence, relevance. `agent` keys the ledger; a subagent's context is its
// task, so it reads no transcript. Nothing relevant remaining is nothing delivered (793a2n).
async function moment($, { agent, kind, text, readContext }) {
  const result = await $.process.run(queryArgv(kind, text))
  if (result.exitCode !== 0) throw new Error(`engram query exited ${result.exitCode}: ${result.stderr.slice(0, 200)}`)
  const ledger = state.ledger(agent)
  const context = readContext && state.transcript ? sinceCompaction(await readTranscript($, state.transcript)) : ''
  const fresh = candidates(parseItems(result.stdout)).filter((note) => !present(note, ledger, context))
  if (fresh.length === 0) return null
  const reply = await $.model.complete(relevanceRequest(kind, text, fresh))
  const chosen = reply.isAnswered ? parseSelection(reply.text, fresh.length).map((index) => fresh[index]) : []
  if (chosen.length === 0) return null
  for (const note of chosen) ledger.add(note.path)
  return buildBlock(chosen)
}

// run executes one engram command; a failing command fails the job that issued it.
async function run($, argv) {
  const result = await $.process.run(argv)
  if (result.exitCode !== 0) throw new Error(`${argv.slice(0, 3).join(' ')} exited ${result.exitCode}: ${result.stderr.slice(0, 200)}`)
  return result
}

// capture is one capture job (793b279): the nearest notes of the window by `engram query`, one `capture` judgement of
// extraction and reconciliation, then the writes: `engram learn` for a lesson no note states, `engram amend
// --activate` for one a note already states. Secrets are redacted before anything reaches the model or a note.
async function capture($, { window, sessionId, turn }) {
  const text = redact(stripOwnBlocks(window))
  if (!text.includes('User:')) return
  const found = await run($, queryArgv('capture', queryText(text)))
  const near = neighbours(parseItems(found.stdout), written)
  const reply = await $.model.complete(captureRequest(text, near))
  if (!reply.isAnswered) throw new Error(`capture judgement unanswered: ${reply.reason}`)
  const source = sourceFor(sessionId, turn)
  const lessons = parseLessons(reply.text, near)
  $.ui.log(`engram-memory: capture judged ${lessons.length} lesson(s) against ${near.length} nearest note(s)`, { to: 'debug' })
  for (const lesson of lessons) {
    if (lesson.covered) {
      await run($, amendArgv(lesson.covered))
      continue
    }
    const note = recentNote(lesson, (await run($, learnArgv(lesson, source))).stdout)
    if (note) written.push(note)
  }
}

// settled reads a subagent's transcript file until it holds the last message (SP5), or the tries are spent.
async function settled($, file, last) {
  let text = ''
  for (let tries = 0; tries < POLL_TRIES; tries += 1) {
    text = await readTranscript($, file)
    if (text !== '' && transcriptHolds(text, last)) break
    await $.clock.sleep(POLL_MS)
  }
  return text
}

// seam is the harness-only line (793b281): when ENGRAM_AUTO_SEAM_LOG names a file, the mod records `queued` when a
// job joins the queue and `idle` when the queue is empty, so that a test waits for the System's work before it closes
// the session (a close does not wait for a job that still has a model call to make). Without the variable it is nothing.
async function seam($, line) {
  const file = await $.env.get('ENGRAM_AUTO_SEAM_LOG')
  if (!file) return
  queue.seam.push(line)
  await $.fs.write(file, `${queue.seam.join('\n')}\n`)
}

// schedule returns at once: the job joins the queue from a timer, after the event has been answered.
async function schedule($, job) {
  queue.pending += 1
  await attempt($, () => seam($, 'queued'))
  $.clock.after(0, () => {
    queue.tail = queue.tail.then(async () => {
      await attempt($, job)
      queue.pending -= 1
      if (queue.pending === 0) await attempt($, () => seam($, 'idle'))
    })
  })
}

export function register(on) {
  on('classic.SessionStart', async ($, e, next) => {
    state.transcript = e.transcript_path
    return next(e)
  })

  on('session.compact', async ($, e, next) => {
    state.ledger(MAIN).clear()
    return next(e)
  })

  on('turn.complete', async ($, e, next) => {
    const result = await next(e)
    if (e.agentId) return result
    const messages = await attempt($, () => $.session.messages())
    const sessionId = await attempt($, () => $.session.id())
    const turn = await attempt($, () => $.session.turns())
    const window = messages ? windowFromMessages(messages, e.answer ?? '') : ''
    if (window !== '') await schedule($, () => capture($, { window, sessionId, turn }))
    return result
  })

  on('classic.SubagentStop', async ($, e, next) => {
    const result = await next(e)
    const sessionId = await attempt($, () => $.session.id())
    const turn = await attempt($, () => $.session.turns())
    const last = e.last_assistant_message ?? ''
    await schedule($, async () => {
      const file = await settled($, e.agent_transcript_path, last)
      const window = windowFromTranscript(file, last)
      if (window !== '') await capture($, { window, sessionId, turn })
    })
    return result
  }).catch(($, e, next) => next(e))

  on('prompt.submit', async ($, e, next) => {
    state.lastPrompt = e.text
    const block = await attempt($, () => moment($, { agent: MAIN, kind: 'prompt', text: momentText(e.text), readContext: true }))
    return next(block ? { ...e, context: [...(e.context ?? []), block] } : e)
  }).catch(($, e, next) => next(e))

  on('classic.PostToolUse', async ($, e, next) => {
    const result = await next(e)
    if (e.tool_name !== 'Skill' || e.tool_input?.skill !== 'route' || e.agent_id) return result
    const text = `${e.tool_input.args ?? ''} ${state.lastPrompt}`.trim()
    const block = await attempt($, () => moment($, { agent: MAIN, kind: 'route', text, readContext: true }))
    return block ? { ...result, additionalContext: [...(result?.additionalContext ?? []), block] } : result
  })

  on('agent.spawn', async ($, e, next) => {
    spawns.push(e.prompt)
    return next(e)
  })

  on('classic.SubagentStart', async ($, e, next) => {
    const result = await next(e)
    const task = spawns.shift() ?? ''
    if (task === '') return result
    const block = await attempt($, () => moment($, { agent: e.agent_id, kind: 'subagent', text: task, readContext: false }))
    return block ? { ...result, additionalContext: [...(result?.additionalContext ?? []), block] } : result
  })
}
