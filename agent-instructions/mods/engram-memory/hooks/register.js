// Automatic memory: the notes of the vault that bear on a prompt arrive beside it (793a2b), and the lessons a turn holds
// become notes after it (793a2c). The logic is in the engram binary (`engram auto deliver`, `judge` and `write`); this
// module routes to it and puts what the delivery prints beside the prompt.

// The prompt is held for 8 seconds at most (793a2d): everything the module does for it, the call for the session's ID
// and the command, is bounded at 7, and the second left is for starting the command and running the prompt's hooks.
const BOUND_MS = 7000

// Every block the command prints opens with this (793d4c); anything else is not a block.
const MARKER = '[engram memory]'

// A failure the writing records is cut to this many characters (793d4k).
const FAILURE_MAX = 300

// The bounds of the stages of the capture of a turn (793d2b): the call for the session's ID, `engram auto judge`,
// `engram auto write` and the run of it that reports a failure. With the 60 seconds that the request of the model holds
// they are 115 seconds, under the 2 minutes of the product (793a2c).
const ID_BOUND_MS = 5000
const JUDGE_BOUND_MS = 15000
const WRITE_BOUND_MS = 20000
const REPORT_BOUND_MS = 15000

// Every request the judgement prints opens with this (793d4g); anything else is not a request.
const REQUEST = '{"model":'

// The prompts the session raised that have started no turn, oldest first, each with its text and its origin; and the
// text and the origin of each turn the main loop has started and not completed, by the turn's ID (793d2b): the prompt
// names its origin, the start of the turn names its text, the end of the turn names only the turn.
const raised = []
const turns = new Map()

// So many prompts are kept at most: one that is dropped, or delivered into a running turn, starts no turn.
const RAISED_MAX = 20

export function register(on) {
  on('prompt.submit', async ($, e, next) => {
    raised.push({ text: e.text, origin: originOf(e) })

    if (raised.length > RAISED_MAX) {
      raised.shift()
    }

    const block = await notesFor($, e)

    return next(block ? { ...e, context: [...(e.context ?? []), block] } : e)
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

    // the lessons are captured apart from the turn: the event's result does not wait for them; the turn of a subagent
    // is left out
    if (e.agentId === undefined) {
      void captureLessons($, e, turn)
    }

    return result
  })
}

// ask is the text of the model's answer to a request, and fails when the model gives none (793d2b). The call ends at the
// time limit that the request holds.
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

// captureLessons judges a turn that ended, asks the model for the judgement and has the lessons written (793d2b). It
// ends quietly when the judgement has nothing to judge, and has the writing record the failure of a step that fails:
// the turn went on as if automatic memory were not installed. Each stage has its bound, and a session whose ID does
// not come is left alone, since the writing cannot record anything without it.
async function captureLessons($, e, turn) {
  let session = ''

  try {
    session = await bounded($.session.id(), ID_BOUND_MS, "the call for the session's ID")

    const printed = await run($, 'judge', {
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
  } catch (error) {
    await report($, session, e.turnId, error)
  }
}

// describe is the text of a failure.
function describe(error) {
  return error instanceof Error ? error.message : String(error)
}

// deliver is the block the command prints for the prompt, or nothing when it prints something that is not a block. The
// command is ended at the bound by the time limit of the process call, and a command that fails fails the delivery.
async function deliver($, e) {
  const input = { text: e.text, session: await $.session.id(), origin: originOf(e) }
  const block = await run($, 'deliver', input, BOUND_MS)

  return block.startsWith(MARKER) ? block : ''
}

// notesFor is the block for the prompt, or nothing when the whole of deliver does not finish within the bound or fails:
// the prompt goes on as if automatic memory were not installed.
async function notesFor($, e) {
  let timer
  const late = new Promise((resolve) => {
    timer = setTimeout(() => resolve(''), BOUND_MS)
  })

  try {
    return await Promise.race([deliver($, e), late])
  } catch {
    return ''
  } finally {
    clearTimeout(timer)
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

// report has the writing record the failure of a step of the capture, when the session is known (793d4k). A report that
// fails is the end of it.
async function report($, session, turn, error) {
  if (session === '') {
    return
  }

  try {
    await run($, 'write', { session, turn, failure: describe(error).slice(0, FAILURE_MAX) }, REPORT_BOUND_MS)
  } catch {
    // nothing more can be done
  }
}

// run is what a step of `engram auto` printed for an input, and fails, naming the command, when it exits with other
// than 0 or cannot be run. The command is ended after timeoutMs, and after the default of the process call when there
// is none.
async function run($, step, input, timeoutMs) {
  const command = `engram auto ${step}`
  let result

  try {
    result = await $.process.run(['engram', 'auto', step], { stdin: JSON.stringify(input), timeoutMs })
  } catch (error) {
    throw new Error(`${command} was ended or could not run: ${describe(error)}`)
  }

  if (result.exitCode !== 0) {
    throw new Error(`${command} exited with ${result.exitCode}: ${String(result.stderr ?? '').trim().split('\n')[0]}`)
  }

  return result.stdout.trim()
}
