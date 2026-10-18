// Automatic memory: the notes and passages of the vault that apply to a prompt arrive beside it (855a3b), and after the
// turn the lessons it holds become notes (793a2c) and its conversation is indexed (793a2f). The logic is in the engram
// binary (`engram auto moment`, `phrases`, `recall`, `deliver`, `judge`, `write` and `sweep`); this module routes to
// it, asks the model for the requests the steps print, puts the block the delivery prints beside the prompt, and shows
// the user the lines it prints for the user, through the host's log (855d2a).

// Each wait of the flow of a prompt, a step or a model call, has 2 minutes of its own (855a3c): a wait that is only slow
// is waited for, and one that has not answered by then has stalled, and the flow steps aside.
const STEP_BOUND_MS = 120000

// The steps of the flow of a prompt that print a request, in order; the delivery follows them (855d2a).
const CHAIN = ['moment', 'phrases', 'recall']

// Every request a step of the flow of a prompt prints begins with this (855d1b); anything else is not a request.
const STEP = '{"request":'

// Every block the command prints opens with this (793d4c); anything else is not a block.
const MARKER = '[engram memory]'

// The bounds of the stages of the capture of a turn (793d2b): the call for the session's ID, `engram auto judge`,
// `engram auto write` and the run of it that reports a failure. With the 60 seconds that the request of the model holds
// they are 115 seconds, under the 2 minutes of the product (793a2c).
const ID_BOUND_MS = 5000
const JUDGE_BOUND_MS = 15000
const WRITE_BOUND_MS = 20000
const REPORT_BOUND_MS = 15000

// The sweep of the conversation has a bound of its own (793d2c): the call for the ID, the sweep and the report of its
// failure are 110 seconds, under the 2 minutes of the product (793a2f), and the capture's bounds are not extended.
const SWEEP_BOUND_MS = 90000

// A sweep that is ended is run once more, as ended, so that the command gives up for the session (793d2c): the call
// for the ID (5), the sweep (90), this run (5) and the report of its failure (15) are 115 seconds, under the 2 minutes.
const GIVE_UP_BOUND_MS = 5000

// Every request the judgement prints opens with this (793d4g); anything else is not a request.
const REQUEST = '{"model":'

// The prompts the session raised that have started no turn, oldest first, each with its text and its origin; and the
// text and the origin of each turn the main loop has started and not completed, by the turn's ID (793d2b): the prompt
// names its origin, the start of the turn names its text, the end of the turn names only the turn.
const raised = []
const turns = new Map()

// The IDs of the notes and passages the module delivered in each session, by the session's ID (855d2c): the file name
// of a note and the chunk ID of a passage, each given to the delivery as held, so that a note is delivered once.
const delivered = new Map()

// The last delivery of each session, as a promise that settles when it is over: the deliveries of one session run one
// after another, so that each reads the set after the one before added to it, and a note is marked as used once
// (855d2a).
const deliveries = new Map()

// So many prompts are kept at most: one that is dropped, or delivered into a running turn, starts no turn.
const RAISED_MAX = 20

export function register(on) {
  on('prompt.submit', async ($, e, next) => {
    raised.push({ text: e.text, origin: originOf(e) })

    if (raised.length > RAISED_MAX) {
      raised.shift()
    }

    const delivery = await deliveryFor($, e)

    // the lines go to the user and the model reads none of them: they are logged, not added to the context (793d2a)
    for (const line of delivery.shown) {
      $.ui.log(line)
    }

    return next(delivery.block ? { ...e, context: [...(e.context ?? []), delivery.block] } : e)
  })

  on('turn.start', async (_$, e, next) => {
    if (e.agentId === undefined) {
      turns.set(e.turnId, { text: e.text, origin: originOfStart(e.text) })
    }

    return next(e)
  })

  on('turn.complete', async ($, e, next) => {
    const result = await next(e)
    const turn = turns.get(e.turnId) ?? { text: '', origin: '' }

    turns.delete(e.turnId)

    // the lessons are captured and the conversation is swept (793d2c) apart from the turn, and apart from each other,
    // from a timer: the host holds the next event of the mod, a prompt among them, until the calls an event left in
    // flight settle, and a timer's calls are no event's; the turn of a subagent is left out
    if (e.agentId === undefined) {
      $.clock.after(0, () => {
        void afterTurn($, e, (session) => captureLessons($, e, turn, session))
        void afterTurn($, e, (session) => sweep($, session, turn.origin))
      })
    }

    return result
  })
}

// ask is the text of the model's answer to a request, and fails when the model gives none (793d2b). The call ends at
// the time limit that the request holds.
async function ask($, request) {
  let reply

  try {
    reply = await $.model.complete(request)
  } catch (error) {
    throw new Error(`the model call failed: ${describe(error)}`)
  }

  if (!reply.isAnswered) {
    throw new Error(`the model call was not answered: ${reply.reason}`)
  }

  return reply.text
}

// askModel is the text of the model's answer to a request of the flow of a prompt, and fails when the model gives none.
// The request is asked exactly as the step printed it; the call is aborted through its signal when it has not answered
// within 2 minutes (855d2a).
async function askModel($, request) {
  const stop = new AbortController()
  const timer = setTimeout(() => stop.abort(), STEP_BOUND_MS)
  let reply

  try {
    reply = await $.model.complete(request, { signal: stop.signal })
  } catch (error) {
    // a call that is aborted may reject, and it was the bound that ended it
    throw new Error(stop.signal.aborted ? `the model call gave nothing within ${limitText(STEP_BOUND_MS)}`
      : `the model call failed: ${describe(error)}`)
  } finally {
    clearTimeout(timer)
  }

  if (!reply.isAnswered) {
    const why = stop.signal.aborted ? `gave nothing within ${limitText(STEP_BOUND_MS)}` : `was not answered: ${reply.reason}`

    throw new Error(`the model call ${why}`)
  }

  return reply.text
}

// bounded is what a promise gives, or a failure that names it when it gives nothing within the bound: the host gives
// some calls no time limit of their own (793d2b).
async function bounded(promise, bound, what) {
  let timer
  const late = new Promise((_resolve, reject) => {
    timer = setTimeout(() => reject(new Error(`${what} gave nothing within ${bound} ms`)), bound)
  })

  try {
    return await Promise.race([promise, late])
  } finally {
    clearTimeout(timer)
  }
}

// afterTurn does a job of the capture of a turn that ended, with the session's ID, and has the writing record the
// failure of a step that fails: the turn went on as if automatic memory were not installed. A session whose ID does not
// come is left alone, since the writing cannot record anything without it (793d2b, 793d2c).
async function afterTurn($, e, job) {
  let session = ''

  try {
    session = await bounded($.session.id(), ID_BOUND_MS, "the call for the session's ID")

    await job(session)
  } catch (error) {
    await report($, session, e.turnId, error)
  }
}

// captureLessons judges a turn that ended, asks the model for the judgement and has the lessons written (793d2b). It
// ends quietly when the judgement has nothing to judge. Each stage has its bound.
async function captureLessons($, e, turn, session) {
  const { stdout: printed } = await run($, 'judge', {
    session,
    turn: e.turnId,
    text: turn.text,
    answer: e.answer,
    origin: turn.origin,
  }, JUDGE_BOUND_MS)

  if (printed === '') {
    return
  }

  const reply = await ask($, requestOf(printed))

  await run($, 'write', { session, turn: e.turnId, reply }, WRITE_BOUND_MS)
}

// sweep indexes the conversation of the session (793d2c).
// A sweep that is ended at its bound is run once more, as ended:
// the command gives up for the session and fails once, naming what to run (793d4n), and that failure is reported.
async function sweep($, session, origin) {
  try {
    await run($, 'sweep', { session, origin }, SWEEP_BOUND_MS)
  } catch (error) {
    if (!error.ended) {
      throw error
    }

    await run($, 'sweep', { session, origin, ended: true }, GIVE_UP_BOUND_MS)
  }
}

// limitText is a time limit written for a person: whole minutes, else milliseconds.
function limitText(ms) {
  return ms >= 60000 && ms % 60000 === 0 ? `${ms / 60000} minutes` : `${ms} ms`
}

// describe is the text of a failure.
function describe(error) {
  return error instanceof Error ? error.message : String(error)
}

// delivery is the block and the lines for the user for a prompt, or nothing when the flow does not finish or fails: the
// prompt goes on as if automatic memory were not installed, and the failure is reported (855d2a).
async function deliveryFor($, e) {
  const state = { session: '' }

  try {
    return await promptChain($, e, state)
  } catch (error) {
    reportDelivery($, state.session, error)

    return { block: '', shown: [] }
  }
}

// promptChain runs the steps of the flow of a prompt in order (855d2a): the moment, the phrases and the recall, each
// given the reply to the request before it and its carry and asking the model for the request it prints, and then the
// delivery with the last reply and its carry. A step that prints nothing ends the chain there, and the delivery is run
// with the prompt's text and the cue alone. The session's ID is left in the state as soon as it is known, for the
// report of a failure.
async function promptChain($, e, state) {
  state.session = await bounded($.session.id(), ID_BOUND_MS, "the call for the session's ID")

  const base = { session: state.session, origin: originOf(e), cue: 'prompt', held: [...(delivered.get(state.session) ?? [])] }
  let answer = {}

  for (const step of CHAIN) {
    const { stdout } = await run($, step, { ...base, ...(step === 'moment' ? { text: e.text } : answer) }, STEP_BOUND_MS)

    if (stdout === '') {
      answer = {}

      break
    }

    const printed = stepOf(stdout, step)

    answer = { reply: await askModel($, printed.request), carry: printed.carry }
  }

  return inOrder(state.session, async () => {
    const held = [...(delivered.get(state.session) ?? [])]
    const { stdout, stderr } = await run($, 'deliver', { ...base, held, text: e.text, ...answer }, STEP_BOUND_MS)
    const lines = stderr.split('\n').map((text) => text.trim()).filter((text) => text !== '')

    // the lines go to the user and the model reads none of them
    return stdout.startsWith(MARKER) ? keepNew(state.session, stdout, lines) : { block: '', shown: lines }
  })
}

// inOrder runs a delivery of a session after the one before it is over, whatever became of that one (855d2a).
async function inOrder(session, job) {
  const before = deliveries.get(session) ?? Promise.resolve()
  let over
  const mine = new Promise((resolve) => { over = resolve })

  deliveries.set(session, before.then(() => mine))
  await before

  try {
    return await job()
  } finally {
    over()
  }
}

// reportDelivery has the writing record that the delivery for a prompt could not be done, from a timer, so that the
// prompt is held no longer and no call is left in flight when the hook settles (793d2a). A delivery whose session's ID
// did not come is left alone, since the writing cannot record anything without it.
function reportDelivery($, session, error) {
  if (session === '') {
    return
  }

  $.clock.after(0, () => {
    void report($, session, '', new Error(`the delivery for a prompt could not be done: ${describe(error)}`))
  })
}

// stepOf is what a step of the flow of a prompt printed: the request for the model and the carry, or a failure that
// names the step when what it printed is not one (855d2a).
function stepOf(printed, step) {
  if (!printed.startsWith(STEP)) {
    throw new Error(`engram auto ${step} printed what is not a request`)
  }

  try {
    return JSON.parse(printed)
  } catch (error) {
    throw new Error(`engram auto ${step} printed a request that is not JSON: ${describe(error)}`)
  }
}

// requestOf is the request for the model that the judgement printed, or a failure that names the judgement when what it
// printed is not one, or is not a JSON object (793d2b).
function requestOf(printed) {
  if (!printed.startsWith(REQUEST)) {
    throw new Error('engram auto judge printed what is not a request')
  }

  try {
    return JSON.parse(printed)
  } catch (error) {
    throw new Error(`engram auto judge printed a request that is not JSON: ${describe(error)}`)
  }
}

// keepNew is the block less each line whose ID the session's set holds, and the lines for the user, less the notes and
// passages that line of theirs names which the block left out, with the IDs of the lines left added to the set in the
// same step, so that a note or a passage is delivered once however many chains find it; no block when no line is left
// (855d2c, 855d2a).
function keepNew(session, block, shown) {
  const ids = delivered.get(session) ?? new Set()
  const lines = block.split('\n')
  const left = []
  const kept = lines.filter((line) => {
    if (!line.startsWith('- ')) {
      return true
    }

    const entry = entryOf(line)

    if (ids.has(entry.id)) {
      return false
    }

    left.push(entry)

    return true
  })

  for (const entry of left) {
    ids.add(entry.id)
  }

  delivered.set(session, ids)

  const all = lines.filter((line) => line.startsWith('- ')).length

  return {
    block: left.length > 0 ? kept.join('\n') : '',
    shown: left.length === all ? shown : shown.flatMap((line) => deliveredLineOf(line, left)),
  }
}

// entryOf is the ID of a line of a block, whole, and whether the line is a passage's (855d2c): the line of a note is a
// dash, the file name, which holds no space, and the situation in parentheses; that of a passage is a dash, the chunk
// ID, which may hold spaces, and `(passage)`.
function entryOf(line) {
  const rest = line.slice(2)
  const note = /^(\S+\.md) \(/.exec(rest)

  if (note) {
    return { id: note[1], passage: rest.startsWith(`${note[1]} (passage): `) }
  }

  const at = rest.indexOf(' (passage): ')

  if (at >= 0) {
    return { id: rest.slice(0, at), passage: true }
  }

  const open = rest.indexOf(' (')

  return { id: open >= 0 ? rest.slice(0, open) : rest, passage: false }
}

// deliveredLineOf is the line for the user that names what was delivered, written again for the entries that are left,
// or nothing when none is left; any other line goes through (855d2a).
function deliveredLineOf(line, left) {
  if (!line.startsWith('delivered ')) {
    return [line]
  }

  const parts = []

  for (const [passage, word] of [[false, 'note'], [true, 'passage']]) {
    const ids = left.filter((entry) => entry.passage === passage).map((entry) => entry.id)

    if (ids.length > 0) {
      parts.push(`the ${word}${ids.length === 1 ? '' : 's'} ${ids.join(', ')}`)
    }
  }

  return parts.length > 0 ? [`delivered ${parts.join(' and ')}`] : []
}

// originOf is the kind of origin that Claude Code stamped on a prompt, or nothing when it stamped none (793d2a).
function originOf(e) {
  return e.origin?.kind ?? ''
}

// originOfStart is the origin of the prompt that a turn which starts with a text began with, and takes that prompt off
// the ones kept, with the prompts before it, which began no turn of their own. The prompt is the oldest kept with the
// text; when none has it, the newest one; a turn that starts with no text has no origin (793d2b).
function originOfStart(text) {
  const at = raised.findIndex((prompt) => prompt.text === text)
  const taken = at >= 0 ? raised.splice(0, at + 1) : text === '' ? [] : raised.splice(0)

  return taken.length > 0 ? taken[taken.length - 1].origin : ''
}

// report has the writing record the failure of a step of the capture, or of a delivery, when the session is known:
// the failure goes whole, for the writing redacts it and cuts it (793d4k). A report that fails is the end of it.
async function report($, session, turn, error) {
  if (session === '') {
    return
  }

  try {
    await run($, 'write', { session, turn, failure: describe(error) }, REPORT_BOUND_MS)
  } catch {
    // nothing more can be done
  }
}

// run is what a step of `engram auto` printed for an input, on standard output and on standard error, and fails, naming
// the command, when it exits with other than 0 or cannot be run. The command is ended after timeoutMs, and after the
// default of the process call when there is none.
async function run($, step, input, timeoutMs) {
  const command = `engram auto ${step}`
  let result

  try {
    result = await $.process.run(['engram', 'auto', step], { stdin: JSON.stringify(input), timeoutMs })
  } catch (error) {
    throw Object.assign(
      new Error(`${command} was ended at its limit of ${limitText(timeoutMs ?? 30000)} or could not run: ${describe(error)}`),
      { ended: true },
    )
  }

  if (result.exitCode !== 0) {
    throw new Error(`${command} exited with ${result.exitCode}: ${String(result.stderr ?? '').trim().split('\n')[0]}`)
  }

  return { stdout: result.stdout.trim(), stderr: String(result.stderr ?? '') }
}
