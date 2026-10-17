// Automatic memory: the notes of the vault that bear on a prompt arrive beside it (793a2b), and the lessons a turn holds
// become notes after it (793a2c). The logic is in the engram binary (`engram auto deliver`, `judge` and `write`); this
// module routes to it and puts what the delivery prints beside the prompt.

// The prompt is held for 8 seconds at most (793a2d): everything the module does for it, the call for the session's ID
// and the command, is bounded at 7, and the second left is for starting the command and running the prompt's hooks.
const BOUND_MS = 7000

// Every block the command prints opens with this (793d4c); anything else is not a block.
const MARKER = '[engram memory]'

// Every request the judgement prints opens with this (793d4g); anything else is not a request.
const REQUEST = '{"model":'

export function register(on) {
  on('prompt.submit', async ($, e, next) => {
    const block = await notesFor($, e)

    return next(block ? { ...e, context: [...(e.context ?? []), block] } : e)
  })

  on('turn.complete', async ($, e, next) => {
    const result = await next(e)

    // the lessons are captured apart from the turn: the event's result does not wait for them
    void captureLessons($, e)

    return result
  })
}

// captureLessons judges a turn that ended, asks the model for the judgement and has the lessons written (793d2b). It
// ends quietly when a step fails or has nothing to give: the turn went on as if automatic memory were not installed.
async function captureLessons($, e) {
  try {
    const session = await $.session.id()
    const request = await run($, 'judge', { session, answer: e.answer })

    if (!request.startsWith(REQUEST)) {
      return
    }

    const reply = await $.model.complete(JSON.parse(request))

    if (reply.isAnswered) {
      await run($, 'write', { session, reply: reply.text })
    }
  } catch {
    // nothing to do: the capture of this turn is over
  }
}

// deliver is the block the command prints for the prompt, or nothing when the command fails or prints something that is
// not a block. The command is ended at the bound by the time limit of the process call.
async function deliver($, e) {
  const request = { text: e.text, session: await $.session.id() }
  const { exitCode, stdout } = await $.process.run(['engram', 'auto', 'deliver'], {
    stdin: JSON.stringify(request),
    timeoutMs: BOUND_MS,
  })
  const block = stdout.trim()

  return exitCode === 0 && block.startsWith(MARKER) ? block : ''
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

// run is what a step of `engram auto` printed for an input, or nothing when the command failed.
async function run($, step, input) {
  const { exitCode, stdout } = await $.process.run(['engram', 'auto', step], { stdin: JSON.stringify(input) })

  return exitCode === 0 ? stdout.trim() : ''
}
