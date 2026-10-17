// Automatic memory (793a2b): the notes of the vault that bear on a prompt arrive beside it. The logic is in the engram
// binary (`engram auto deliver`); this module routes the prompt to it and puts what it prints beside the prompt.
export function register(on) {
  on('prompt.submit', async ($, e, next) => {
    const request = { text: e.text, session: await $.session.id() }
    const { stdout } = await $.process.run(['engram', 'auto', 'deliver'], { stdin: JSON.stringify(request) })

    return next({ ...e, context: [...(e.context ?? []), stdout.trim()] })
  })
}
