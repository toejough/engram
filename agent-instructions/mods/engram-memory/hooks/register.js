// Automatic memory (793a2b): the notes of the vault that bear on a prompt arrive beside it. The logic is in the engram
// binary (`engram auto deliver`); this module routes the prompt to it and puts what it prints beside the prompt.

// The prompt is held for 8 seconds at most (793a2d): the command is ended at 7, and the second left is for starting it.
const COMMAND_MS = 7000

// Every block the command prints opens with this (793d4c); anything else is not a block.
const MARKER = '[engram memory]'

export function register(on) {
  on('prompt.submit', async ($, e, next) => {
    const block = await notesFor($, e)

    return next(block ? { ...e, context: [...(e.context ?? []), block] } : e)
  })
}

// notesFor is the block the command prints for the prompt, or nothing when it does not finish in time, fails or prints
// something that is not a block: the prompt goes on as if automatic memory were not installed.
async function notesFor($, e) {
  try {
    const request = { text: e.text, session: await $.session.id() }
    const { exitCode, stdout } = await $.process.run(['engram', 'auto', 'deliver'], {
      stdin: JSON.stringify(request),
      timeoutMs: COMMAND_MS,
    })
    const block = stdout.trim()

    return exitCode === 0 && block.startsWith(MARKER) ? block : ''
  } catch {
    return ''
  }
}
