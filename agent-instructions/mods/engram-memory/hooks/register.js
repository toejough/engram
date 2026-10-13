// The System's mod (793e2c): maps the events of the delivery moments to the moment job and adds what it returns
// through the channel each event has: prompt context, tool-result context, subagent-start context (793b192).
import { MARKS_NAME, baseName, chunksDirectory, parseManifest, parseMarks, pending, withSource, withTurns } from './backlog.js'
import {
  amendArgv, backlogSourceFor, learnArgv, messagesFromTranscript, neighbours, parseLessons, queryText, recentNote, redact,
  sourceFor, stripOwnBlocks, transcriptHolds, windowFromMessages, windowFromTranscript, windowsFromMessages,
} from './capture.js'
import { DEADLINE_MS, StepError, exitKind, kindOf, replyKind, spawnKind, warning } from './failure.js'
import {
  NEAR_COUNT, ingestArgv, nearestArgv, parseOffers, parseShow, pendingArgv, settleArgvs, showArgv, vaultFrom,
} from './housekeeping.js'
import { captureRequest, curationRequest, parseCuration, parseSelection, relevanceRequest } from './judge.js'
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
// `turnDone`: a top-level turn finished in the burst of jobs now in the queue, so its summary is due at the burst's end
const queue = { tail: Promise.resolve(), pending: 0, seam: [], turnDone: false }
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
const LOOKUP_STEP = { prompt: 'the prompt lookup', route: 'the route lookup', subagent: 'the subagent lookup' }
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
    throw new StepError(`${label}: ${error.message}`, spawnKind(error.message))
  }
  if (!ok.includes(result.exitCode)) throw new StepError(`${label} exited ${result.exitCode}: ${result.stderr.slice(0, 200)}`, exitKind(result.stderr))
  return result
}

// answered is the text of a model reply, or a StepError of the class the reply's absence shows.
function answered(reply, what) {
  if (!reply.isAnswered) throw new StepError(`${what} judgement unanswered: ${reply.reason}`, replyKind(reply))
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

// moment runs one lookup (793b192): query, presence, relevance. `agent` keys the ledger; a subagent's context is its
// task, so it reads no transcript. Nothing relevant remaining is nothing delivered (793a2n).
async function moment($, { agent, kind, text, readContext }) {
  const result = await exec($, queryArgv(kind, text), DEADLINE_MS.lookup)
  const ledger = state.ledger(agent)
  const context = readContext && state.transcript ? sinceCompaction(await readTranscript($, state.transcript)) : ''
  const fresh = candidates(parseQuery(result.stdout)).filter((note) => !present(note, ledger, context))
  if (fresh.length === 0) return null
  const reply = await $.model.complete(relevanceRequest(kind, text, fresh))
  const chosen = parseSelection(answered(reply, 'relevance'), fresh.length).map((index) => fresh[index])
  if (chosen.length === 0) return null
  for (const note of chosen) ledger.add(note.path)
  state.counts.delivered += chosen.length
  return buildBlock(chosen)
}

// capture is one capture job (793b279): the nearest notes of the window by `engram query`, one `capture` judgement of
// extraction and reconciliation, then the writes: `engram learn` for a lesson no note states, `engram amend
// --activate` for one a note already states. Secrets are redacted before anything reaches the model or a note.
async function capture($, { window, source }) {
  const text = redact(stripOwnBlocks(window))
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
// tell a speaker from a line that imitates one, 793e8g). A source that cannot be read is left for a later turn.
async function sourceMessages($, source) {
  try {
    return messagesFromTranscript(await readSource($, source.path))
  } catch (error) {
    $.ui.log(`engram-memory: backlog: ${source.path}: ${error.message}`, { to: 'debug' })
    return []
  }
}

// backlog is the job after a turn's housekeeping (793e8g), whose ingest has brought the manifest up to date: take the
// one most recently changed source nobody has captured yet, capture it in windows with the same judgement as a turn,
// and mark it only when every window finished without a failure.
async function backlog($) {
  const dir = await chunksDir($)
  const sources = parseManifest(await readTranscript($, `${dir}/manifest.json`))
  const marks = await readMarks($, dir)
  for (const source of pending(sources, marks, { now: await $.clock.now(), live: state.transcript })) {
    const messages = await sourceMessages($, source)
    const windows = windowsFromMessages(messages)
    if (windows.length === 0) continue
    for (const window of windows) await capture($, { window, source: backlogSourceFor(source.path, `message ${messages.length}`) })
    await $.fs.write(`${dir}/${MARKS_NAME}`, JSON.stringify(withSource(await readMarks($, dir), source)))
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
  if (!shown.pending || shown.skill || shown.kind === 'runbook') return
  const offer = { situation: redact(shown.situation), body: redact(shown.body) }
  const near = await nearest($, shown, offer)
  const reply = await $.model.complete(curationRequest(offer, near))
  const decision = parseCuration(answered(reply, 'curation'), near, redact)
  $.ui.log(`engram-memory: curation judged ${name} ${decision.outcome}${decision.target ? ` ${decision.target}` : ''} against ${near.length} nearest note(s)`, { to: 'debug' })
  for (const argv of settleArgvs(decision, name, shown.hash)) await exec($, argv, DEADLINE_MS.command)
  // a reinforcing activation (covered) is not an amendment: only a near offer changes a note's claim (793b204)
  state.counts.curated += 1
  if (decision.outcome === 'near') state.counts.amended += 1
}

// curate scans the vault for pending offers the way the curate skill does (the query excludes them) and settles each.
async function curate($) {
  const vault = vaultFrom({ vault: await $.env.get('ENGRAM_VAULT_PATH'), xdg: await $.env.get('XDG_DATA_HOME'), home: await $.env.get('HOME') })
  if (!(await $.fs.exists(vault))) return
  const found = await exec($, pendingArgv(vault), DEADLINE_MS.scan, [0, 1])
  for (const name of parseOffers(found.stdout)) await curateOne($, name)
}

// housekeeping is the work that follows the last job of a queue (793b202, 793b201): the chunk index catches up with the
// turn, then the pending offers are settled, then, after a top-level turn, one source of the backlog is captured
// (793e8g). A failure of one step is reported and the others still run.
async function housekeeping($) {
  const turn = queue.turnDone
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
  queue.turnDone = false
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
        await housekeeping($)
        if (queue.turnDone) await attempt($, () => report($))
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
      if (window !== '') await capture($, { window, source: sourceFor(sessionId, turn) })
    }, CAPTURE_STEP)
    return result
  }).catch(($, e, next) => next(e))

  on('prompt.submit', async ($, e, next) => {
    state.lastPrompt = e.text
    const block = await attempt($, () => moment($, { agent: MAIN, kind: 'prompt', text: momentText(e.text), readContext: true }), LOOKUP_STEP.prompt)
    return next(block ? { ...e, context: [...(e.context ?? []), block] } : e)
  }).catch(($, e, next) => next(e))

  on('classic.PostToolUse', async ($, e, next) => {
    const result = await next(e)
    if (e.tool_name !== 'Skill' || e.tool_input?.skill !== 'route' || e.agent_id) return result
    const text = `${e.tool_input.args ?? ''} ${state.lastPrompt}`.trim()
    const block = await attempt($, () => moment($, { agent: MAIN, kind: 'route', text, readContext: true }), LOOKUP_STEP.route)
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
