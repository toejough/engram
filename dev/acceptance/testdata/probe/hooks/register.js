// The probe of the gates (855b46): the smallest hooks module that adds a block within a turn. At the first `tool.call`
// of the session it adds, as the context that the model reads after the tool's result, a block that begins
// `[engram memory]` and names one note, and it adds none at any later call. It holds no model call and no command.
// A fixture of the acceptance tests, shipped by nothing and installed by none of `engram update`.

const BLOCK = '[engram memory]\n- probe-note.md (2026-10-18; probing the host): the probe note states one thing.'

let put = false

export function register(on) {
  on('tool.call', async ($, e, next) => {
    const result = await next(e)

    if (put) {
      return result
    }

    put = true
    $.ui.log(`probe: put the block at ${Date.now()}`)

    return { ...result, context: [...(result.context ?? []), BLOCK] }
  })
}
