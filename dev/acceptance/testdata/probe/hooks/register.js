// The probe of the gates (855b46): the smallest hooks module that adds a block within a turn. At the first `tool.call`
// of the session it adds, as the context that the model reads after the tool's result, a block that begins
// `[engram memory]` and names one note, and it adds none at any later call. At its second call it reads the session's
// transcript and reports how many `hook_additional_context` attachments the transcript already holds, before the
// turn ends. It holds no model call.
// A fixture of the acceptance tests, shipped by nothing and installed by none of `engram update`.

const BLOCK = '[engram memory]\n- probe-note.md (2026-10-18; probing the host): the probe note states one thing.'

let calls = 0

export function register(on) {
  on('tool.call', async ($, e, next) => {
    const result = await next(e)

    calls += 1

    if (calls === 2) {
      const id = await $.session.id()
      const found = await $.process.run(['sh', '-c', `cat "$CLAUDE_CONFIG_DIR"/projects/*/${id}.jsonl | grep -c '"hook_additional_context"'`])
      $.ui.log(`probe: at the second call the transcript holds ${found.stdout.trim()} hook_additional_context`)
    }

    if (calls > 1) {
      return result
    }

    $.ui.log(`probe: put the block at ${Date.now()}`)

    return { ...result, context: [...(result.context ?? []), BLOCK] }
  })
}
