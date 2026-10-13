// The System's mod (793e2c): maps the events of the delivery moments to the moment job and adds what it returns
// through the channel each event has: prompt context, tool-result context, subagent-start context (793b192).
import {
  MARKS_NAME, SOURCES_PER_TURN, baseName, chunksDirectory, nextWindows, parseManifest, parseMarks, pending, windowsDone, withProgress, withSource,
  withTurns,
} from './backlog.js'
import {
  amendArgv, backlogSourceFor, learnArgv, messagesFromTranscript, neighbours, parseLessons, queryText, recentNote, recentTexts,
  redact, sourceFor, transcriptHolds, windowFromMessages, windowFromTranscript, windowsFromMessages,
} from './capture.js'
import { DEADLINE_MS, StepError, failureKind, kindOf, warning } from './failure.js'
import {
  MAX_OFFERS, NEAR_COUNT, ingestArgv, nearestArgv, parseOffers, parseShow, pendingArgv, settleArgvs, showArgv, vaultFrom,
} from './housekeeping.js'
import { captureRequest, classifyRequest, curationRequest, parseCuration, parseMoments, parseSelection, relevanceRequest } from './judge.js'
import { candidates, buildBlock, momentText, parseQuery, present, queryArgv, sinceCompaction } from './moment.js'
import { emptyCounts, statusLine } from './summary.js'

const MAIN = 'main'
// a subagent's transcript file lags its stop event (SP5): it is polled this often, this many times
const POLL_MS = 500
const POLL_TRIES = 20
// the capture queue (793b203): one job at a time, in order, so that the capture of a later event sees the notes of an
// earlier one; the tail never rejects
// the dispatch prompts of agent.spawn, in order, for the SubagentStart events that follow (SP5: start carries no prompt)
const spawns = []
const ledgers = new Map()
// the pending offers this process will not look at again (793b318): those curate leaves alone (a skill registration, a
// runbook, a note no longer pending) and those that failed, so that none of them uses up the run's cap or is retried
const skipped = new Set()
// `turnDone`: a top-level turn finished in the burst of jobs now in the queue, so its summary is due at the burst's end
const queue = { tail: Promise.resolve(), pending: 0, seam: [], turnDone: false }
// the in-turn cue (793b320): the job started at the last step of the working agent that wrote text and called a tool
// ({ promise, done, notes, startedAt }), whose notes ride the result of a later tool of the same turn
const step = { job: null }
// a tool result waits for the job this long at most (the hook's own budget is about 10 s, measured), and a job older
// than the deadline is dropped
const HOLD_MS = 7000
const STEP_DEADLINE_MS = 25000
// the notes this session wrote (basename, situation and text), kept to compare later windows with
const written = []
const state = {
  transcript: null,
  lastPrompt: '',
  // what the System did since the last summary (793a5h), the warnings not yet reported with one, and whether the
  // status line holds anything the System put there
  counts: emptyCounts(),
  warnings: new Set(),
  lineShown: false,
  ledger(agent) {
    if (!ledgers.has(agent)) ledgers.set(agent, new Set())
    return ledgers.get(agent)
  },
}

// The steps a failure is reported against (793a4b): the user is told which part of automatic memory was skipped.
const LOOKUP_STEP = { prompt: 'the prompt lookup', route: 'the route lookup', subagent: 'the subagent lookup', step: 'the step lookup' }
const CAPTURE_STEP = 'the capture'
const BACKLOG_STEP = 'the backlog capture'

// warn shows the user, on the status line and nowhere else, that automatic memory was skipped and why. It never fails.
async function warn($, step, kind) {
  const text = warning(step, kind)
  state.warnings.add(text)
  state.lineShown = true
  try {
    await $.ui.status(text)
  } catch (error) {
    $.ui.log(`engram-memory: the warning could not be shown: ${error.message}`, { to: 'debug' })
  }
}

// attempt contains a failure of the job: the event goes on as if the mod were not there, and a job with a `step` warns
// the user (793a4b). A job with no step is bookkeeping of the mod's own and fails quietly, into the debug log.
async function attempt($, job, step) {
  try {
    return await job()
  } catch (error) {
    $.ui.log(`engram-memory: ${step ?? 'internal'}: ${error.message}`, { to: 'debug' })
    if (step) await warn($, step, kindOf(error))
    return null
  }
}

// exec runs one engram (or system) command with its deadline. A command that cannot start, that passes its deadline or
// that exits with a code other than the `ok` ones is a StepError with its class (793b205).
async function exec($, argv, deadlineMs, ok = [0]) {
  const label = argv.slice(0, 3).join(' ')
  let result
  try {
    result = await $.process.run(argv, { timeoutMs: deadlineMs })
  } catch (error) {
    throw new StepError(`${label}: ${error.message}`, failureKind(error.message))
  }
  if (!ok.includes(result.exitCode)) throw new StepError(`${label} exited ${result.exitCode}: ${result.stderr.slice(0, 200)}`, failureKind(result.stderr))
  return result
}

// answered is the text of a model reply, or a StepError of the class the reply's absence shows.
function answered(reply, what) {
  if (!reply.isAnswered) throw new StepError(`${what} judgement unanswered: ${reply.reason}`, failureKind(reply.reason ?? ''))
  return reply.text
}

// readTranscript reads the session's transcript file; the file does not exist before the first row is written.
async function readTranscript($, file) {
  try {
    return await $.fs.read(file)
  } catch {
    return ''
  }
}

// lookup runs one lookup (793b192): query, presence, relevance, and answers the notes that bear on the moment. `agent`
// keys the ledger; a subagent's context is its task, so it reads no transcript. `moments` are the phrases the moment
// names beside the text. Nothing relevant remaining is nothing (793a2n).
async function lookup($, { agent, kind, text, readContext, moments = [] }) {
  const result = await exec($, queryArgv(kind, text, moments), DEADLINE_MS.moment)
  const ledger = state.ledger(agent)
  const context = readContext && state.transcript ? sinceCompaction(await readTranscript($, state.transcript)) : ''
  const fresh = candidates(parseQuery(result.stdout)).filter((note) => !present(note, ledger, context))
  const said = (delivered) => $.ui.log(`engram-memory: ${kind} lookup: ${fresh.length} candidate(s), ${delivered} delivered`, { to: 'debug' })
  if (fresh.length === 0) {
    await said(0)
    return []
  }
  const reply = await $.model.complete(relevanceRequest(moments.length > 0 ? moments.join(', ') : kind, text, fresh))
  const chosen = parseSelection(answered(reply, 'relevance'), fresh.length).map((index) => fresh[index])
  await said(chosen.length)
  return chosen
}

// deliver makes the block of notes the agent does not hold yet, and counts them: a note that entered the ledger since
// the lookup is not delivered again (793a2e).
function deliver(agent, notes) {
  const ledger = state.ledger(agent)
  const fresh = notes.filter((note) => !ledger.has(note.path))
  if (fresh.length === 0) return null
  for (const note of fresh) ledger.add(note.path)
  state.counts.delivered += fresh.length
  return buildBlock(fresh)
}

const moment = async ($, args) => deliver(args.agent, await lookup($, args))

// stepJob is the in-turn cue (793b320): one classification of the recent texts, then, when it names a moment, the lookup
// of those moments and subjects.
async function stepJob($, { messages, answer }) {
  const reply = await $.model.complete(classifyRequest(recentTexts(messages, answer)))
  const moments = parseMoments(answered(reply, 'classification'), redact)
  if (moments.length === 0) return []
  const text = moments.map((each) => each.subject).join('; ')
  return lookup($, { agent: MAIN, kind: 'step', text, readContext: true, moments: moments.map((each) => each.moment) })
}

// The agent is running memory itself (793a2e): a prompt that is a memory command or asks for one, and a step whose tool
// is a memory skill or an engram command. The System adds nothing to what the agent retrieves on purpose.
const MEMORY_ASK = /^\s*\/(?:recall|learn|curate|write-memory)\b|\bengram\s+(?:query|learn|amend|ingest|show|activate)\b/i
const MEMORY_SKILLS = new Set(['recall', 'learn', 'curate', 'write-memory'])
const runsMemory = (use) => (use.name === 'Skill' && MEMORY_SKILLS.has(use.input?.skill)) || (use.name === 'Bash' && /\bengram\s/.test(use.input?.command ?? ''))
// the path of a note, as engram names it: what a tool result shows of it is in the agent's context
const NOTE_PATH = /\b\d+\.\d{4}-\d{2}-\d{2}\.[A-Za-z0-9_.-]+\.md\b/g

// startStep starts the job of a step in the background, one at a time: the step goes on while it runs.
async function startStep($, result) {
  if (MEMORY_ASK.test(state.lastPrompt) || result.toolUses.some(runsMemory)) return
  if (step.job && !step.job.done) return
  const messages = await $.session.messages()
  const job = { done: false, notes: [], startedAt: await $.clock.now() }
  job.promise = new Promise((resolve) => {
    $.clock.after(0, async () => {
      job.notes = (await attempt($, () => stepJob($, { messages, answer: result.answer }), LOOKUP_STEP.step)) ?? []
      job.done = true
      resolve()
    })
  })
  step.job = job
}

// heldStep answers the notes of the step job for a tool result of the working agent: it waits for the job up to HOLD_MS;
// a job still running is kept for a later result until its deadline, a finished one is taken.
async function heldStep($) {
  const job = step.job
  if (!job) return []
  if (!job.done) await Promise.race([job.promise, $.clock.sleep(HOLD_MS)])
  if (job.done) {
    step.job = null
    return job.notes
  }
  if ((await $.clock.now()) - job.startedAt > STEP_DEADLINE_MS) step.job = null
  return []
}

// capture is one capture job (793b279): the nearest notes of the window by `engram query`, one `capture` judgement of
// extraction and reconciliation, then the writes: `engram learn` for a lesson no note states, `engram amend
// --activate` for one a note already states. Secrets are redacted before anything reaches the model or a note.
async function capture($, { window, source }) {
  const text = redact(window)
  if (!text.includes('User:')) return
  const found = await exec($, queryArgv('capture', queryText(text)), DEADLINE_MS.lookup)
  const near = neighbours(parseQuery(found.stdout), written)
  const reply = await $.model.complete(captureRequest(text, near))
  const lessons = parseLessons(answered(reply, 'capture'), near)
  $.ui.log(`engram-memory: capture judged ${lessons.length} lesson(s) against ${near.length} nearest note(s)`, { to: 'debug' })
  for (const lesson of lessons) {
    if (lesson.covered) {
      await exec($, amendArgv(lesson.covered), DEADLINE_MS.command)
      continue
    }
    const note = recentNote(lesson, (await exec($, learnArgv(lesson, source), DEADLINE_MS.command)).stdout)
    state.counts.added += 1
    if (note) written.push(note)
  }
}

// chunksDir is where `engram ingest` keeps the chunk index; the mark file of the backlog lies beside the manifest.
async function chunksDir($) {
  return chunksDirectory({ chunks: await $.env.get('ENGRAM_CHUNKS_DIR'), xdg: await $.env.get('XDG_DATA_HOME'), home: await $.env.get('HOME') })
}

async function readMarks($, dir) {
  return parseMarks(await readTranscript($, `${dir}/${MARKS_NAME}`))
}

// markTurns records that the System captured the turns of this session up to now, so that the backlog does not read
// the session again once it is quiet (793e8g).
async function markTurns($) {
  if (!state.transcript) return
  const dir = await chunksDir($)
  const marks = await readMarks($, dir)
  await $.fs.write(`${dir}/${MARKS_NAME}`, JSON.stringify(withTurns(marks, baseName(state.transcript), await $.clock.now())))
}

// A file the platform will not read whole (over 4 MiB) is read from its tail by `tail -c`; the first line is then cut
// and is dropped.
const READ_LIMIT = /over the \d+-byte limit/
const TAIL_ARGV = (path) => ['tail', '-c', '3000000', path]

async function readSource($, path) {
  try {
    return await $.fs.read(path)
  } catch (error) {
    if (!READ_LIMIT.test(error.message)) throw error
    const tail = await exec($, TAIL_ARGV(path), DEADLINE_MS.command)
    return tail.stdout.slice(tail.stdout.indexOf('\n') + 1)
  }
}

// sourceMessages reads a source's conversation from the file itself, by the roles of its rows (the index text cannot
// tell a speaker from a line that imitates one, 793e8g). A source that cannot be read is null.
async function sourceMessages($, source) {
  try {
    return messagesFromTranscript(await readSource($, source.path))
  } catch (error) {
    $.ui.log(`engram-memory: backlog: ${source.path}: ${error.message}`, { to: 'debug' })
    return null
  }
}

const writeMarks = ($, dir, marks) => $.fs.write(`${dir}/${MARKS_NAME}`, JSON.stringify(marks))

// the sources that could not be read, which this process does not try again (a later process does)
const unreadable = new Set()

// backlog is the job after a turn's housekeeping (793e8g), whose ingest has brought the manifest up to date: take the
// one most recently changed source nobody has captured yet and judge up to WINDOWS_PER_TURN of its windows, the end of
// the conversation first, with the same judgement as a turn. The source is marked only when every window has been
// judged without a failure; a source with nothing to judge is marked at once, one that cannot be read is left alone.
async function backlog($) {
  const dir = await chunksDir($)
  const sources = parseManifest(await readTranscript($, `${dir}/manifest.json`))
  const marks = await readMarks($, dir)
  let looked = 0
  for (const source of pending(sources, marks, { now: await $.clock.now(), live: state.transcript })) {
    if (unreadable.has(source.path)) continue
    if (looked >= SOURCES_PER_TURN) return
    looked += 1
    const messages = await sourceMessages($, source)
    if (messages === null) {
      unreadable.add(source.path)
      continue
    }
    const windows = windowsFromMessages(messages)
    if (windows.length === 0) {
      await writeMarks($, dir, withSource(await readMarks($, dir), source))
      continue
    }
    const done = windowsDone(marks, source)
    const { batch, finished } = nextWindows(windows, done)
    for (const [index, window] of batch.entries()) {
      await capture($, { window, source: backlogSourceFor(source.path, `message ${messages.length}`) })
      if (index < batch.length - 1) await writeMarks($, dir, withProgress(await readMarks($, dir), source, done + index + 1))
    }
    const latest = await readMarks($, dir)
    await writeMarks($, dir, finished ? withSource(latest, source) : withProgress(latest, source, done + batch.length))
    return
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

// nearest is the existing notes an offer is judged against: for an offer that amends a note, that note; otherwise the
// vault's nearest notes by `engram query`, which leaves pending offers out. Everything the judge reads is redacted.
async function nearest($, shown, offer) {
  const cleaned = (note) => ({ ...note, situation: redact(note.situation), body: redact(note.body) })
  if (shown.forNote) {
    const note = parseShow((await exec($, showArgv(shown.forNote), DEADLINE_MS.command)).stdout)
    return [cleaned({ path: shown.forNote, kind: note.kind, situation: note.situation, body: note.body })]
  }
  const found = await exec($, nearestArgv(offer), DEADLINE_MS.lookup)
  return parseQuery(found.stdout).filter((item) => item.kind !== 'chunk' && item.situation !== '').slice(0, NEAR_COUNT).map(cleaned)
}

// curateOne settles one pending offer as curate does (793b201): read it, judge it against the nearest notes, run the
// commands of the outcome. A skill registration (a runbook marked by its skill hash) is authored, never judged, so it
// is left pending for curate; a note that is not pending (settled meanwhile) is left alone.
async function curateOne($, name) {
  const shown = parseShow((await exec($, showArgv(name), DEADLINE_MS.command)).stdout)
  if (!shown.pending || shown.skill || shown.kind === 'runbook') return 'skipped'
  const offer = { situation: redact(shown.situation), body: redact(shown.body) }
  const near = await nearest($, shown, offer)
  const reply = await $.model.complete(curationRequest(offer, near))
  const decision = parseCuration(answered(reply, 'curation'), near, redact)
  $.ui.log(`engram-memory: curation judged ${name} ${decision.outcome}${decision.target ? ` ${decision.target}` : ''} against ${near.length} nearest note(s)`, { to: 'debug' })
  for (const argv of settleArgvs(decision, name, shown.hash)) await exec($, argv, DEADLINE_MS.command)
  // a reinforcing activation (covered) is not an amendment: only a near offer changes a note's claim (793b204)
  state.counts.curated += 1
  if (decision.outcome === 'near') state.counts.amended += 1
  return 'settled'
}

// curate scans the vault for pending offers the way the curate skill does (the query excludes them) and settles each,
// at most MAX_OFFERS a run; an offer that is skipped or fails is not counted again and not looked at again.
async function curate($) {
  const vault = vaultFrom({ vault: await $.env.get('ENGRAM_VAULT_PATH'), xdg: await $.env.get('XDG_DATA_HOME'), home: await $.env.get('HOME') })
  if (!(await $.fs.exists(vault))) return
  const found = await exec($, pendingArgv(vault), DEADLINE_MS.scan, [0, 1])
  let handled = 0
  for (const name of parseOffers(found.stdout)) {
    if (handled >= MAX_OFFERS) break
    if (skipped.has(name)) continue
    // one offer that fails is warned about once and does not stop the offers after it
    const outcome = await attempt($, () => curateOne($, name), 'curation')
    if (outcome !== 'skipped') handled += 1
    if (outcome !== 'settled') skipped.add(name)
  }
}

// housekeeping is the work that follows the last job of a queue (793b202, 793b201): the chunk index catches up with the
// turn, then the pending offers are settled, then, after a top-level turn, one source of the backlog is captured
// (793e8g). A failure of one step is reported and the others still run.
async function housekeeping($, turn) {
  await attempt($, () => exec($, ingestArgv(), DEADLINE_MS.ingest), 'indexing')
  await attempt($, () => curate($), 'curation')
  if (turn) await attempt($, () => backlog($), BACKLOG_STEP)
}

// report shows the user the one line of the turn (793a5h): the summary of what the System did since the previous one,
// with the warnings of the turn on the same line (793b302), or nothing; a turn with nothing to say clears the line a
// previous turn left, so that a stale count is not read as this turn's. It runs once the System's work has finished.
async function report($) {
  const line = statusLine(state.counts, [...state.warnings])
  state.counts = emptyCounts()
  state.warnings.clear()
  if (line === null && !state.lineShown) return
  state.lineShown = line !== null
  await $.ui.status(line ?? undefined)
}

// schedule returns at once: the job joins the queue from a timer, after the event has been answered. The last job of
// the queue is followed by the housekeeping, so that the index is brought up to date once per burst of jobs.
async function schedule($, job, step, { turn = false } = {}) {
  queue.pending += 1
  if (turn) queue.turnDone = true
  await attempt($, () => seam($, 'queued'))
  $.clock.after(0, () => {
    queue.tail = queue.tail.then(async () => {
      await attempt($, job, step)
      if (queue.pending === 1) {
        // the flag is taken as the burst ends: a turn that completes during the housekeeping sets it again and is
        // reported by the burst that holds its own work (793b317)
        const turn = queue.turnDone
        queue.turnDone = false
        await housekeeping($, turn)
        if (turn) await attempt($, () => report($))
      }
      queue.pending -= 1
      if (queue.pending === 0) await attempt($, () => seam($, 'idle'))
    }).catch(() => undefined)
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
    // a dispatch that started no subagent must not give its task to the next one that starts (793b317), and a step job
    // is dropped with the turn it belongs to (793b320)
    spawns.length = 0
    step.job = null
    const messages = await attempt($, () => $.session.messages(), CAPTURE_STEP)
    const sessionId = await attempt($, () => $.session.id(), CAPTURE_STEP)
    const turn = await attempt($, () => $.session.turns(), CAPTURE_STEP)
    const window = messages ? windowFromMessages(messages, e.answer ?? '') : ''
    // every completed turn is followed by the housekeeping, whether or not it has a window to capture
    await schedule(
      $,
      async () => {
        if (window === '') return
        await capture($, { window, source: sourceFor(sessionId, turn) })
        await attempt($, () => markTurns($))
      },
      CAPTURE_STEP,
      { turn: true },
    )
    return result
  })

  on('classic.SubagentStop', async ($, e, next) => {
    const result = await next(e)
    const sessionId = await attempt($, () => $.session.id(), CAPTURE_STEP)
    const turn = await attempt($, () => $.session.turns(), CAPTURE_STEP)
    const last = e.last_assistant_message ?? ''
    await schedule($, async () => {
      const file = await settled($, e.agent_transcript_path, last)
      const window = windowFromTranscript(file, last)
      if (window !== '') await capture($, { window, source: sourceFor(sessionId, turn, e.agent_id) })
    }, CAPTURE_STEP)
    return result
  }).catch(($, e, next) => next(e))

  on('prompt.submit', async ($, e, next) => {
    state.lastPrompt = e.text
    const block = await attempt($, () => moment($, { agent: MAIN, kind: 'prompt', text: momentText(e.text), readContext: true }), LOOKUP_STEP.prompt)
    return next(block ? { ...e, context: [...(e.context ?? []), block] } : e)
  }).catch(($, e, next) => next(e))

  // The step hook only reads: the chunks pass as they come and the result is the one beneath (purview, 793a4c). A step
  // of the working agent that wrote text and called a tool starts the in-turn cue.
  on('turn.step', async function* ($, e, next) {
    const result = yield* next(e)
    if (!e.agentId && result?.answer?.trim() && result.toolUses?.length > 0) await attempt($, () => startStep($, result))
    return result
  })

  on('classic.PostToolUse', async ($, e, next) => {
    const result = await next(e)
    if (e.agent_id) return result
    // what a tool result shows of a note, the agent holds: it is not delivered again
    for (const path of JSON.stringify(e.tool_response ?? '').match(NOTE_PATH) ?? []) state.ledger(MAIN).add(path)
    const isRoute = e.tool_name === 'Skill' && e.tool_input?.skill === 'route'
    // the result of route has the lookup of its own; the notes of a step wait for a later result (793b239)
    const block = isRoute
      ? await attempt($, () => moment($, { agent: MAIN, kind: 'route', text: `${e.tool_input.args ?? ''} ${state.lastPrompt}`.trim(), readContext: true }), LOOKUP_STEP.route)
      : deliver(MAIN, (await attempt($, () => heldStep($))) ?? [])
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
    const block = await attempt($, () => moment($, { agent: e.agent_id, kind: 'subagent', text: task, readContext: false }), LOOKUP_STEP.subagent)
    return block ? { ...result, additionalContext: [...(result?.additionalContext ?? []), block] } : result
  })
}
