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

// Every request the judgement prints opens with this (793d4g); anything else is not a request.
const REQUEST = '{"model":'

// The origin of the last prompt the session raised, and the origin of each turn the main loop has started and not
// completed, by the turn's ID (793d2b): the prompt names its origin, the end of the turn names only the turn.
let lastOrigin = ''
const originsOfTurns = new Map()

export function register(on) {
  on('prompt.submit', async ($, e, next) => {
    lastOrigin = originOf(e)

    const block = await notesFor($, e)

    return next(block ? { ...e, context: [...(e.context ?? []), block] } : e)
  })

  on('turn.start', async (_$, e, next) => {
    if (e.agentId === undefined) {
      originsOfTurns.set(e.turnId, lastOrigin)
    }

    return next(e)
  })

  on('turn.complete', async ($, e, next) => {
    const result = await next(e)
    const origin = originsOfTurns.get(e.turnId) ?? ''

    originsOfTurns.delete(e.turnId)

    // the lessons are captured apart from the turn: the event's result does not wait for them; the turn of a subagent
    // is left out
    if (e.agentId === undefined) {
      void captureLessons($, e, origin)
    }

    return result
  })
}

// ask is the text of the model's answer to a request, and fails when the model gives none (793d2b).
async function ask($, request) {
  let reply

  try {
    reply = await $.model.complete(JSON.parse(request))
  } catch (error) {
    throw new Error(`the model call failed: ${describe(error)}`)
  }

  if (!reply.isAnswered) {
    throw new Error(`the model call was not answered: ${reply.reason}`)
  }

  return reply.text
}

// captureLessons judges a turn that ended, asks the model for the judgement and has the lessons written (793d2b). It
// ends quietly when the judgement has nothing to judge, and has the writing record the failure of a step that fails:
// the turn went on as if automatic memory were not installed.
async function captureLessons($, e, origin) {
  let session = ''

  try {
    session = await $.session.id()

    const request = await run($, 'judge', { session, turn: e.turnId, answer: e.answer, origin })

    if (request === '') {
      return
    }

    if (!request.startsWith(REQUEST)) {
      throw new Error('engram auto judge printed what is not a request')
    }

    await run($, 'write', { session, turn: e.turnId, reply: await ask($, request) })
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

// originOf is the kind of origin that Claude Code stamped on a prompt, or nothing when it stamped none (793d2a).
function originOf(e) {
  return e.origin?.kind ?? ''
}

// report has the writing record the failure of a step of the capture, when the session is known (793d4k). A report that
// fails is the end of it.
async function report($, session, turn, error) {
  if (session === '') {
    return
  }

  try {
    await run($, 'write', { session, turn, failure: describe(error).slice(0, FAILURE_MAX) })
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
