// Automatic memory (793a2b): the notes of the vault that bear on a prompt arrive beside it. The logic is in the engram
// binary (`engram auto deliver`); this module routes the prompt to it and puts what it prints beside the prompt.

// The prompt is held for 8 seconds at most (793a2d): everything the module does for it, the call for the session's ID
// and the command, is bounded at 7, and the second left is for starting the command and running the prompt's hooks.
const BOUND_MS = 7000

// Every block the command prints opens with this (793d4c); anything else is not a block.
const MARKER = '[engram memory]'

export function register(on) {
  on('prompt.submit', async ($, e, next) => {
    const block = await notesFor($, e)

    return next(block ? { ...e, context: [...(e.context ?? []), block] } : e)
  })
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
