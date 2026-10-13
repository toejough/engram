// The spike probe of 793e8a: observes events and writes what it saw to files outside the transcript (PROBE_DIR);
// PROBE_CONFIG (JSON) says which context to add, which model to call, how long to work. It never denies or drops.
let seq = 0

async function config($) {
  const raw = await $.env.get('PROBE_CONFIG')
  return raw ? JSON.parse(raw) : {}
}

async function record($, kind, data) {
  const dir = await $.env.get('PROBE_DIR')
  if (!dir) return
  seq += 1
  const t = await $.clock.now()
  const entry = { kind, t, data }
  await $.fs.write(dir + '/' + String(t).padStart(15, '0') + '-' + String(seq).padStart(4, '0') + '-' + kind + '.json', JSON.stringify(entry))
}

function short(text) {
  return typeof text === 'string' && text.length > 160 ? text.slice(0, 160) + '…' : text
}

async function guarded($, kind, fn) {
  try {
    return await fn()
  } catch (error) {
    await record($, kind + '.error', { message: String(error && error.message ? error.message : error) })
    return undefined
  }
}

export function register(on) {
  on('session.start', async ($, e, next) => {
    await record($, 'session.start', e)
    return next(e)
  })

  on('classic.SessionStart', async ($, e, next) => {
    await record($, 'classic.SessionStart', e)
    return next(e)
  })

  on('prompt.submit', async ($, e, next) => {
    const cfg = await config($)
    const started = await $.clock.now()
    const messages = await guarded($, 'session.messages', () => $.session.messages()) // not-h1
    await record($, 'prompt.submit', {
      text: short(e.text),
      origin: e.origin,
      turnId: e.turnId,
      context: e.context,
      messages: Array.isArray(messages) // not-h1
        ? messages.map((m) => ({ role: m.role, text: short(m.text), tools: (m.toolUses || []).length, hasWatch: Boolean(cfg.watch && typeof m.text === 'string' && m.text.includes(cfg.watch)) })) // not-h1
        : null, // not-h1
    })
    // model-call:begin
    if (cfg.model) {
      const r = await guarded($, 'model.complete', () =>
        $.model.complete({ model: cfg.model.alias, prompt: cfg.model.prompt, maxTokens: 20, timeoutMs: 20000 }),
      )
      await record($, 'model.complete', r)
    }
    // model-call:end
    if (cfg.sleepSeconds) {
      const r = await guarded($, 'process.run', () => $.process.run(['sleep', String(cfg.sleepSeconds)]))
      await record($, 'process.run', { exitCode: r && r.exitCode, ms: (await $.clock.now()) - started })
    }
    if (cfg.sendTo) { // not-h1
      const sent = await guarded($, 'session.send', () => $.session.send({ to: { sessionId: cfg.sendTo }, text: cfg.sendText })) // not-h1
      await record($, 'session.send', sent) // not-h1
    } // not-h1
    if (cfg.status) {
      await guarded($, 'ui.status', () => $.ui.status(cfg.status))
      await record($, 'ui.status', { text: cfg.status })
    }
    if (cfg.context && cfg.context.promptSubmit) {
      return next({ ...e, context: [...(e.context || []), cfg.context.promptSubmit] })
    }
    return next(e)
  })

  on('classic.PostToolUse', async ($, e, next) => {
    const cfg = await config($)
    await record($, 'classic.PostToolUse', { tool_name: e.tool_name, agent_id: e.agent_id, tool_use_id: e.tool_use_id })
    const result = await next(e)
    if (cfg.context && cfg.context.postToolUse && e.tool_name === 'Bash') {
      return { ...result, additionalContext: [...((result && result.additionalContext) || []), cfg.context.postToolUse] }
    }
    return result
  })

  on('agent.spawn', async ($, e, next) => {
    await record($, 'agent.spawn', { prompt: short(e.prompt), description: e.description, subagentType: e.subagentType })
    return next(e)
  })

  on('classic.SubagentStart', async ($, e, next) => {
    const cfg = await config($)
    await record($, 'classic.SubagentStart', e)
    const result = await next(e)
    if (cfg.context && cfg.context.subagentStart) {
      return { ...result, additionalContext: [...((result && result.additionalContext) || []), cfg.context.subagentStart] }
    }
    return result
  })

  on('classic.SubagentStop', async ($, e, next) => {
    const info = { ...e }
    const text = await guarded($, 'fs.read', () => $.fs.read(e.agent_transcript_path))
    if (typeof text === 'string') {
      info.transcriptLinesAtStop = text.split('\n').filter((l) => l.trim() !== '').length
      info.transcriptHasLastMessageAtStop = e.last_assistant_message ? text.includes(e.last_assistant_message.slice(0, 40)) : null
      const assistantTexts = text
        .split('\n')
        .filter((l) => l.includes('"type":"assistant"'))
        .map((l) => {
          try {
            return (JSON.parse(l).message.content || []).filter((b) => b.type === 'text').map((b) => b.text).join('')
          } catch {
            return ''
          }
        })
      info.assistantRowsAtStop = assistantTexts.length
      info.assistantRowHoldsLastMessageAtStop = e.last_assistant_message ? assistantTexts.some((t) => t.includes(e.last_assistant_message)) : null
    }
    await record($, 'classic.SubagentStop', info)
    return next(e)
  })

  on('session.compact', async ($, e, next) => {
    await record($, 'session.compact', e)
    return next(e)
  })

  on('session.receive', async ($, e, next) => {
    await record($, 'session.receive', { origin: e.origin, text: short(e.text) })
    return next(e)
  })

  on('turn.complete', async ($, e, next) => {
    const cfg = await config($)
    await record($, 'turn.complete', { answer: short(e.answer), durationMs: e.durationMs, isAborted: e.isAborted, agentId: e.agentId })
    if (cfg.background && !e.agentId) {
      $.clock.after(cfg.background.afterMs, async () => {
        await record($, 'background.timer', { afterMs: cfg.background.afterMs })
      })
      ;(async () => {
        const r = await guarded($, 'background.process', () => $.process.run(['sleep', String(cfg.background.sleepSeconds)]))
        await record($, 'background.process.done', { exitCode: r && r.exitCode })
      })()
    }
    return next(e)
  })
}
