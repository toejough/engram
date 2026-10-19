// Automatic memory: the notes and passages of the vault that apply to a prompt arrive beside it (855a3b), and after the
// turn the lessons it holds become notes (793a2c) and its conversation is indexed (793a2f). The logic is in the engram
// binary (`engram auto moment`, `phrases`, `recall`, `deliver`, `judge`, `place`, `write` and `sweep`); this module
// routes to it, asks the model for the requests the steps print, puts the block the delivery prints beside the prompt,
// and shows the user the lines it prints for the user, through the host's log (855d2a).

// Each wait of the flow of a prompt, a step or a model call, has 2 minutes of its own (855a3c): a wait that is only
// slow is waited for, and one that has not answered by then has stalled, and the flow steps aside.
const STEP_BOUND_MS = 120000

// The steps of the flow of a prompt that print a request, in order; the delivery follows them (855d2a).
const CHAIN = ['moment', 'phrases', 'recall']

// Every request a step of the flow of a prompt prints begins with this (855d1b); anything else is not a request.
const STEP = '{"request":'

// Every block the command prints opens with this (793d4c); anything else is not a block.
const MARKER = '[engram memory]'

// The bounds of the stages of the capture of a turn (855d2b): the call for the session's ID, the steps `engram auto
// moment`, `judge` and `place`, `engram auto write` and the run of it that reports a failure. The three model calls
// hold the limits of their requests, 10, 30 and 15 seconds; together they are 115 seconds, under the 2 minutes of the
// product (793a2c).
const ID_BOUND_MS = 5000
const MOMENT_BOUND_MS = 5000
const JUDGE_BOUND_MS = 10000
const PLACE_BOUND_MS = 5000
const WRITE_BOUND_MS = 20000
const REPORT_BOUND_MS = 15000

// The write that follows a deep delivery has a bound of its own (855d2a): it holds neither the prompt nor the agent.
const DEEP_WRITE_BOUND_MS = 60000

// The sweep of the conversation has a bound of its own (793d2c): the call for the ID, the sweep and the report of its
// failure are 110 seconds, under the 2 minutes of the product (793a2f), and the capture's bounds are not extended.
const SWEEP_BOUND_MS = 90000

// A sweep that is ended is run once more, as ended, so that the command gives up for the session (793d2c): the call
// for the ID (5), the sweep (90), this run (5) and the report of its failure (15) are 115 seconds, under the 2 minutes.
const GIVE_UP_BOUND_MS = 5000

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

// The writes after the deliveries of a session, each chained after the one before it, for the same reason.
const writes = new Map()

// The text and the origin of the turn the main loop is in, which the events of its tools carry none of (855d2c): the
// last turn started.
let current = { text: '', origin: '' }

// The IDs of the calls whose failure was judged and whose batch has not come (855d2c): a batch of calls all judged as
// failures is not judged again.
const failed = new Set()

// The keys a tool.call event carries beside the arguments of the tool (855d2c).
const RESERVED = ['tool', 'tool_use_id', 'agentId', 'requestMeta', 'consent']

// So many prompts are kept at most: one that is dropped, or delivered into a running turn, starts no turn.
const RAISED_MAX = 20

export function register(on) {
  on('prompt.submit', async ($, e, next) => {
    raised.push({ text: e.text, origin: originOf(e) })

    if (raised.length > RAISED_MAX) {
      raised.shift()
    }

    const delivery = await deliveryFor($, 'a prompt', (state) => promptChain($, e, state))

    // the lines go to the user and the model reads none of them: they are logged, not added to the context (793d2a)
    for (const line of delivery.shown) {
      $.ui.log(line)
    }

    return next(delivery.block ? { ...e, context: [...(e.context ?? []), delivery.block] } : e)
  })

  on('turn.start', async (_$, e, next) => {
    if (e.agentId === undefined) {
      current = { text: e.text, origin: originOfStart(e.text) }
      turns.set(e.turnId, current)
    }

    return next(e)
  })

  // the three cues of a turn (855d2c): the chain of the prompt runs for the event, and the block rides the answer
  on('tool.call', async ($, e, next) => {
    if (e.agentId !== undefined) {
      return next(e)
    }

    const input = Object.fromEntries(Object.entries(e).filter(([key]) => !RESERVED.includes(key)))
    const event = { tool: e.tool, input }
    // the call is let run while its chain runs (855b102); the block rides the result once both are over
    const [delivery, result] = await Promise.all([
      cueDelivery($, 'before-tool', () => bounded($.session.id(), ID_BOUND_MS, "the call for the session's ID"), event),
      next(e),
    ])

    // a call that was denied has no context to add to; the notes of the block are held no longer
    if (result.deny !== undefined) {
      release(delivery)

      return result
    }

    return withBlock(result, 'context', delivery.block)
  })

  on('classic.PostToolUseFailure', async ($, e, next) => {
    if (e.agent_id !== undefined || e.is_interrupt) {
      return next(e)
    }

    failed.add(e.tool_use_id)

    const event = { tool: e.tool_name, input: e.tool_input, error: e.error }
    const delivery = await cueDelivery($, 'failure', async () => e.session_id, event)

    return withBlock(await next(e), 'additionalContext', delivery.block)
  })

  on('classic.PostToolBatch', async ($, e, next) => {
    const calls = e.tool_calls ?? []
    const judged = calls.every((call) => failed.has(call.tool_use_id))

    for (const call of calls) {
      failed.delete(call.tool_use_id)
    }

    if (e.agent_id !== undefined || judged) {
      return next(e)
    }

    const event = {
      calls: calls.map((call) => ({ tool: call.tool_name, input: call.tool_input, response: call.tool_response })),
    }
    const delivery = await cueDelivery($, 'batch-done', async () => e.session_id, event)

    return withBlock(await next(e), 'additionalContext', delivery.block)
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
    const why = stop.signal.aborted
      ? `gave nothing within ${limitText(STEP_BOUND_MS)}`
      : `was not answered: ${reply.reason}`

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

// captureLessons judges a turn that ended and has the lessons written (855d2b): the moment, the judgement and the
// placement each print the request for the model, which is asked exactly as printed, and the writing is given the
// replies and the carry of the placement. It ends quietly when the moment or the judgement prints nothing, and writes
// the lessons at the top when the placement prints nothing. Each stage has its bound.
async function captureLessons($, e, turn, session) {
  const fields = { session, turn: e.turnId, text: turn.text, answer: e.answer, origin: turn.origin }
  const moment = await asked($, 'moment', { ...fields, cue: 'turn-end' }, MOMENT_BOUND_MS)

  if (moment === null) {
    return
  }

  const judged = await asked($, 'judge', { ...fields, reply: moment.reply, carry: moment.carry }, JUDGE_BOUND_MS)

  if (judged === null) {
    return
  }

  const written = { session, turn: e.turnId, reply: judged.reply }

  try {
    const placed = await asked($, 'place', { session, turn: e.turnId, reply: judged.reply, carry: judged.carry },
      PLACE_BOUND_MS)

    if (placed !== null) {
      written.carry = placed.carry
      written.placement = placed.reply
    }
  } catch (error) {
    // a placement that fails does not fail the capture: the lessons are written at the top and the writing records why
    written.placementFailure = describe(error)
  }

  await run($, 'write', written, WRITE_BOUND_MS)
}

// asked runs a step of the capture and asks the model for the request it prints (855d2b): the reply and the carry, or
// nothing when the step prints nothing. A request that is not an object is a failure of the step.
async function asked($, step, input, bound) {
  const { stdout } = await run($, step, input, bound)

  if (stdout === '') {
    return null
  }

  const printed = stepOf(stdout, step)

  try {
    return { reply: await ask($, printed.request), carry: printed.carry }
  } catch (error) {
    throw new Error(`${step}: ${describe(error)}`)
  }
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

// deliveryFor is the block and the lines for the user for a prompt or a tool event, or nothing when the flow does not
// finish or fails: the work goes on as if automatic memory were not installed, and the failure is reported (855d2a,
// 855d2c). The chain is given the state, in which the session's ID is left as soon as it is known, for the report.
async function deliveryFor($, what, chainOf) {
  const state = { session: '', write: null }

  try {
    const delivery = await chainOf(state)

    writeAfter($, state)

    return delivery
  } catch (error) {
    reportDelivery($, state.session, what, error)

    return { block: '', shown: [] }
  }
}

// cueDelivery is deliveryFor for the event of a tool: the chain of a prompt with the event and the text the turn began
// with in place of the prompt's text (855d2c). The lines for the user are logged here.
async function cueDelivery($, cue, sessionOf, event) {
  const delivery = await deliveryFor($, 'a tool call', async (state) => {
    state.session = await sessionOf()

    return chain($, state, { origin: current.origin, cue }, { text: current.text, event })
  })

  for (const line of delivery.shown) {
    $.ui.log(line)
  }

  return delivery
}

// withBlock is the answer of the hooks beneath, with the block added as one more entry of the list the key names, or
// the answer as it is when there is no block (855d2c).
function withBlock(result, key, block) {
  return block ? { ...result, [key]: [...(result[key] ?? []), block] } : result
}

// promptChain runs the chain of a prompt (855d2a).
async function promptChain($, e, state) {
  state.session = await bounded($.session.id(), ID_BOUND_MS, "the call for the session's ID")

  return chain($, state, { origin: originOf(e), cue: 'prompt' }, { text: e.text })
}

// chain runs the steps of the flow in order (855d2a): the moment, the phrases and the recall, each given the reply to
// the request before it and its carry and asking the model for the request it prints, and then the delivery with the
// last reply and its carry. A step that prints nothing ends the chain there, and the delivery is run with the fields
// of the moment alone: the prompt's text, or for a tool event the event and the text the turn began with.
async function chain($, state, who, fields) {
  const base = { session: state.session, ...who, held: [...(delivered.get(state.session) ?? [])] }
  let answer = {}

  for (const step of CHAIN) {
    const { stdout } = await run($, step, { ...base, ...(step === 'moment' ? fields : answer) }, STEP_BOUND_MS)

    if (stdout === '') {
      answer = {}

      break
    }

    const printed = stepOf(stdout, step)

    answer = { reply: await askModel($, printed.request), carry: printed.carry }

    // a deep recall says that a write follows the delivery (855d2a)
    state.write = printed.write === true ? { cue: base.cue, ...answer } : null
  }

  return inOrder(deliveries, state.session, async () => {
    const held = [...(delivered.get(state.session) ?? [])]
    const { stdout, stderr } = await run($, 'deliver', { ...base, held, ...fields, ...answer }, STEP_BOUND_MS)
    const lines = stderr.split('\n').map((text) => text.trim()).filter((text) => text !== '')

    // the lines go to the user and the model reads none of them
    return stdout.startsWith(MARKER) ? keepNew(state.session, stdout, lines) : { block: '', shown: lines }
  })
}

// writeAfter has the lessons and the notes of a deep delivery written from a timer once the block is given, so that the
// writing holds neither the prompt nor the agent and no call is in flight when the hook settles; a write that fails is
// reported (855d2a, 855b29). The writes of a session are run one after the other, so that two deep cues in a turn do
// not crystallize the same lesson twice.
function writeAfter($, state) {
  if (state.write === null) {
    return
  }

  const { session, write } = state

  $.clock.after(0, () => {
    void (async () => {
      await inOrder(writes, session, async () => {
        try {
          await run($, 'write', { session, ...write }, DEEP_WRITE_BOUND_MS)
        } catch (error) {
          await report($, session, '', new Error(`the write after a deep delivery failed: ${describe(error)}`))
        }
      })
    })()
  })
}

// inOrder runs a job of a session after the one before it in the chains is over, whatever became of that one (855d2a).
async function inOrder(chains, session, job) {
  const before = chains.get(session) ?? Promise.resolve()
  let over
  const mine = new Promise((resolve) => { over = resolve })

  chains.set(session, before.then(() => mine))
  await before

  try {
    return await job()
  } finally {
    over()
  }
}

// reportDelivery has the writing record that the delivery for a prompt or a tool call could not be done, from a timer,
// so that the prompt is held no longer and no call is left in flight when the hook settles (793d2a). A delivery whose
// session's ID did not come is left alone, since the writing cannot record anything without it.
function reportDelivery($, session, what, error) {
  if (session === '') {
    return
  }

  $.clock.after(0, () => {
    void report($, session, '', new Error(`the delivery for ${what} could not be done: ${describe(error)}`))
  })
}

// stepOf is what a step printed: the request for the model and the carry, or a failure that names the step when what it
// printed is not one (855d2a, 855d2b).
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
    session,
    added: left.map((entry) => entry.id),
  }
}

// release takes the IDs of a block that was not given off the session's set, so that its notes are delivered later
// (855d2c).
function release(delivery) {
  for (const id of delivery.added ?? []) {
    delivered.get(delivery.session)?.delete(id)
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
      new Error(
        `${command} was ended at its limit of ${limitText(timeoutMs ?? 30000)} or could not run: ${describe(error)}`,
      ),
      { ended: true },
    )
  }

  if (result.exitCode !== 0) {
    throw new Error(`${command} exited with ${result.exitCode}: ${String(result.stderr ?? '').trim().split('\n')[0]}`)
  }

  return { stdout: result.stdout.trim(), stderr: String(result.stderr ?? '') }
}
