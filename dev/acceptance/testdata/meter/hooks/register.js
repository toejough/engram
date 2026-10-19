// The meter of the measurements of change 855 (855d7a): a hooks module that changes nothing and appends one line to
// meter.jsonl, beside the machine's home, for each model call of any mod (its time, its request's size and the usage
// the host reports), for each tool call of the main loop (its time) and for each turn of the main loop (the usage of the
// agent's own requests). A fixture of the acceptance tests, shipped by nothing and installed by none of `engram update`.

const SHELL = 'cat >> "$HOME/../meter.jsonl"'
const HEAD = 80
const REPLY = 400

async function append($, line) {
  await $.process.run(['sh', '-c', SHELL], { stdin: `${JSON.stringify(line)}\n` })
}

export function register(on) {
  on('model.complete', async ($, e, next) => {
    const at = Date.now()
    const result = await next(e)
    const took = Date.now() - at
    const value = result.value ?? {}
    const bytes = (text) => new TextEncoder().encode(text ?? '').length

    await append($, {
      k: 'model', at, took, model: e.model, system: bytes(e.system), prompt: bytes(e.prompt),
      head: (e.system ?? '').slice(0, HEAD), answered: value.isAnswered ?? false, usage: value.usage ?? null,
      reply: (value.text ?? '').slice(0, REPLY),
    })

    return result
  })

  on('tool.call', async ($, e, next) => {
    const at = Date.now()
    const result = await next(e)

    if (e.agentId === undefined) {
      await append($, { k: 'tool', at, took: Date.now() - at, tool: e.tool, command: e.command ?? '' })
    }

    return result
  })

  on('turn.complete', async ($, e, next) => {
    const result = await next(e)

    if (e.agentId === undefined) {
      await append($, { k: 'turn', at: Date.now(), usage: e.usage ?? null })
    }

    return result
  })
}
