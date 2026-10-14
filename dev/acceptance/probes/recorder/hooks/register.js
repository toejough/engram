// The spike probe of 793e8a: observes events and writes what it saw to files outside the transcript (PROBE_DIR);
// PROBE_CONFIG (JSON) says which context to add, which model to call, how long to work. It never denies or drops.
const TIME_DIGITS = 15
const SEQ_DIGITS = 4
const TEXT_CHARS = 160
const MESSAGE_CHARS = 40
let seq = 0

async function config($) {
  const raw = await $.env.get('PROBE_CONFIG')
  return raw ? JSON.parse(raw) : {}
}

async function record($, kind, data) {
  const dir = await $.env.get('PROBE_DIR')
  if (!dir) return
  seq += 1
  const time = await $.clock.now()
  const entry = { kind, time: time, data }
  await $.fs.write(
    dir +
      '/' +
      String(time).padStart(TIME_DIGITS, '0') +
      '-' +
      String(seq).padStart(SEQ_DIGITS, '0') +
      '-' +
      kind +
      '.json',
    JSON.stringify(entry),
  )
}

function short(text) {
  return typeof text === 'string' && text.length > TEXT_CHARS ? text.slice(0, TEXT_CHARS) + '…' : text
}

async function guarded($, kind, body) {
  try {
    return await body()
  } catch (error) {
    await record($, kind + '.error', { message: String(error && error.message ? error.message : error) })
    return undefined
  }
}

export function register(on) {
  on('session.start', async ($, event, next) => {
    await record($, 'session.start', event)
    return next(event)
  })

  on('classic.SessionStart', async ($, event, next) => {
    await record($, 'classic.SessionStart', event)
    return next(event)
  })

  on('prompt.submit', async ($, event, next) => {
    const cfg = await config($)
    const started = await $.clock.now()
    const messages = await guarded($, 'session.messages', () => $.session.messages()) // not-h1
    await record($, 'prompt.submit', {
      text: short(event.text),
      origin: event.origin,
      turnId: event.turnId,
      context: event.context,
      messages: Array.isArray(messages) // not-h1
        ? messages.map((message) => ({
            role: message.role,
            text: short(message.text),
            tools: (message.toolUses || []).length,
            hasWatch: Boolean(cfg.watch && typeof message.text === 'string' && message.text.includes(cfg.watch)),
          })) // not-h1
        : null, // not-h1
    })
    // model-call:begin
    if (cfg.model) {
      const result = await guarded($, 'model.complete', () =>
        $.model.complete({ model: cfg.model.alias, prompt: cfg.model.prompt, maxTokens: 20, timeoutMs: 20000 }),
      )
      await record($, 'model.complete', result)
    }
    // model-call:end
    if (cfg.sleepSeconds) {
      const result = await guarded($, 'process.run', () => $.process.run(['sleep', String(cfg.sleepSeconds)]))
      await record($, 'process.run', {
        exitCode: result && result.exitCode,
        elapsedMs: (await $.clock.now()) - started,
      })
    }
    if (cfg.sendTo) {
      // not-h1
      const sent = await guarded($, 'session.send', () =>
        $.session.send({ to: { sessionId: cfg.sendTo }, text: cfg.sendText }),
      ) // not-h1
      await record($, 'session.send', sent) // not-h1
    } // not-h1
    if (cfg.status) {
      await guarded($, 'ui.status', () => $.ui.status(cfg.status))
      await record($, 'ui.status', { text: cfg.status })
    }
    if (cfg.context && cfg.context.promptSubmit) {
      return next({ ...event, context: [...(event.context || []), cfg.context.promptSubmit] })
    }
    return next(event)
  })

  on('classic.PostToolUse', async ($, event, next) => {
    const cfg = await config($)
    await record($, 'classic.PostToolUse', {
      tool_name: event.tool_name,
      agent_id: event.agent_id,
      tool_use_id: event.tool_use_id,
    })
    const result = await next(event)
    if (cfg.context && cfg.context.postToolUse && event.tool_name === 'Bash') {
      return {
        ...result,
        additionalContext: [...((result && result.additionalContext) || []), cfg.context.postToolUse],
      }
    }
    return result
  })

  on('agent.spawn', async ($, event, next) => {
    await record($, 'agent.spawn', {
      prompt: short(event.prompt),
      description: event.description,
      subagentType: event.subagentType,
    })
    return next(event)
  })

  on('classic.SubagentStart', async ($, event, next) => {
    const cfg = await config($)
    await record($, 'classic.SubagentStart', event)
    const result = await next(event)
    if (cfg.context && cfg.context.subagentStart) {
      return {
        ...result,
        additionalContext: [...((result && result.additionalContext) || []), cfg.context.subagentStart],
      }
    }
    return result
  })

  on('classic.SubagentStop', async ($, event, next) => {
    const info = { ...event }
    const text = await guarded($, 'fs.read', () => $.fs.read(event.agent_transcript_path))
    if (typeof text === 'string') {
      info.transcriptLinesAtStop = text.split('\n').filter((line) => line.trim() !== '').length
      info.transcriptHasLastMessageAtStop = event.last_assistant_message
        ? text.includes(event.last_assistant_message.slice(0, MESSAGE_CHARS))
        : null
      const assistantTexts = text
        .split('\n')
        .filter((line) => line.includes('"type":"assistant"'))
        .map((line) => {
          try {
            return (JSON.parse(line).message.content || [])
              .filter((block) => block.type === 'text')
              .map((block) => block.text)
              .join('')
          } catch {
            return ''
          }
        })
      info.assistantRowsAtStop = assistantTexts.length
      info.assistantRowHoldsLastMessageAtStop = event.last_assistant_message
        ? assistantTexts.some((candidate) => candidate.includes(event.last_assistant_message))
        : null
    }
    await record($, 'classic.SubagentStop', info)
    return next(event)
  })

  on('session.compact', async ($, event, next) => {
    await record($, 'session.compact', event)
    return next(event)
  })

  on('session.receive', async ($, event, next) => {
    await record($, 'session.receive', { origin: event.origin, text: short(event.text) })
    return next(event)
  })

  on('turn.complete', async ($, event, next) => {
    const cfg = await config($)
    await record($, 'turn.complete', {
      answer: short(event.answer),
      durationMs: event.durationMs,
      isAborted: event.isAborted,
      agentId: event.agentId,
    })
    if (cfg.background && !event.agentId) {
      $.clock.after(cfg.background.afterMs, async () => {
        await record($, 'background.timer', { afterMs: cfg.background.afterMs })
      })
      ;(async () => {
        const result = await guarded($, 'background.process', () =>
          $.process.run(['sleep', String(cfg.background.sleepSeconds)]),
        )
        await record($, 'background.process.done', { exitCode: result && result.exitCode })
      })()
    }
    return next(event)
  })
}
