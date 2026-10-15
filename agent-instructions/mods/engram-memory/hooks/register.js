// The System's mod (793e2c, 793b368): an adapter. It maps the events of Claude Code to jobs of `engram auto run`, where
// every decision is made, relays the model requests a job needs, and adds what it returns through the channel each
// event has (prompt context, tool-result context, subagent-start context, the status line). The state it keeps is the
// engine's own, handed back and never read: the delivery ledger of the hook jobs and the work state of the queued ones.
const ROUND_MS = 600000
const MAX_ROUNDS = 200
const STDERR_CHARS = 200
// a tool result waits for the in-turn cue this long at most (the hook's own budget is about 10 s, measured), and a cue
// older than the deadline is dropped
const HOLD_MS = 7000
const STEP_DEADLINE_MS = 25000
// an event waits for a job that holds it (a prompt, a tool result of route, a subagent's start) this long at most; the
// declared deadlines of the engine's commands are within it (793d36b3)
const HOOK_DEADLINE_MS = 8000
// the in-turn cue reads the last six texts of the session (the engine's recentCount), so a step hands it the last
// messages and not the whole session, with the text of each and no tool input or result (793d23b2)
const STEP_WINDOW = 50
// the steps a failure is reported against (793a4b)
const STEP = {
  prompt: 'the prompt lookup',
  route: 'the route lookup',
  subagent: 'the subagent lookup',
  step: 'the step lookup',
}
const CAPTURE_STEP = 'the capture'
const HOUSEKEEPING_STEP = 'the housekeeping'
// what a failure is worded as when the engine cannot run at all and so cannot word it (it words every other)
const ENGINE_UNAVAILABLE = 'the vault is unavailable'

/** @type {ModState} */
const mod = {
  ledger: undefined,
  work: undefined,
  transcript: '',
  lastPrompt: '',
  lineShown: false,
  step: null,
  hooks: Promise.resolve(),
  tally: [],
  warnings: [],
  spawns: [],
  queue: { tail: Promise.resolve(), pending: 0, seam: [], turnEnded: false },
}
// remember merges state in one synchronous step, so that an update made after an await cannot interleave with another;
// the fields it is used for are each read and written by jobs that already run one at a time.
/** @type {(patch: Record<string, unknown>) => unknown} */
const remember = (patch) => Object.assign(mod, patch)
/** @type {(list: unknown[] | undefined, visit: (item: any) => unknown) => void} */
const each = (list, visit) => (list ?? []).forEach(visit)

/**
 * @param {Ctx} $
 * @param {() => Promise<any>} job
 * @param {string | null} [step]
 */
async function attempt($, job, step) {
  try {
    return await job()
  } catch (/** @type {any} */ error) {
    $.ui.log(`engram-memory: ${step ?? 'internal'}: ${error.message}`, { to: 'debug' })
    if (step) await told($, step, error.message)
    return null
  }
}

// told has the engine word and class a failure of a step, as it does the failures it contains (793d36b3); an engine
// that cannot run says only that.
/**
 * @param {Ctx} $
 * @param {string} step
 * @param {string} message
 */
async function told($, step, message) {
  try {
    await relay($, { kind: 'failure', step, message })
  } catch {
    await warn($, `Automatic memory was skipped because ${ENGINE_UNAVAILABLE} (${step})`)
  }
}

// warn shows the user, on the status line and nowhere else, that automatic memory was skipped and why. It never fails.
/**
 * @param {Ctx} $
 * @param {string} text
 */
async function warn($, text) {
  mod.warnings.push(text)
  mod.lineShown = true
  try {
    await $.ui.status(text)
  } catch (/** @type {any} */ error) {
    $.ui.log(`engram-memory: the warning could not be shown: ${error.message}`, { to: 'debug' })
  }
}

/**
 * @param {Ctx} $
 * @param {Round} out
 */
function absorb($, out) {
  each(out.log, (line) => $.ui.log(line, { to: 'debug' }))
  each(out.warnings, (text) => warn($, text))
  if (out.counts) mod.tally.push(out.counts)
}

// relay runs a job to its end (793b197): one round of `engram auto run` after another, each with the replies of the
// model to the requests of the one before, which are relayed as they came and the replies as they were. A job whose
// hold has passed (its `cut` is set) starts no further round.
/**
 * @param {Ctx} $
 * @param {Job} job
 * @param {Hold} [hold]
 */
async function relay($, job, hold = { cut: false }) {
  let input = job
  for (let round = 0; round < MAX_ROUNDS; round += 1) {
    const out = await roundOf($, input, hold)
    absorb($, out)
    if (out.done) return out.done
    input = { ...job, state: out.state, reply: await replyOf($, out) }
  }
  throw new Error('the job did not end')
}

/**
 * @param {Ctx} $
 * @param {Job} input
 * @param {Hold} hold
 * @returns {Promise<Round>}
 */
async function roundOf($, input, hold) {
  if (hold.cut) throw new Error(`engram auto run still running after ${HOOK_DEADLINE_MS} ms`)
  const result = await $.process.run(['engram', 'auto', 'run'], { stdin: JSON.stringify(input), timeoutMs: ROUND_MS })
  if (result.exitCode !== 0)
    throw new Error(`engram auto run exited ${result.exitCode}: ${result.stderr.slice(0, STDERR_CHARS)}`)
  return JSON.parse(result.stdout)
}

/**
 * @param {Ctx} $
 * @param {Round} out
 */
async function replyOf($, out) {
  if (out.waitMs) await $.clock.sleep(out.waitMs)
  return out.request ? $.model.complete(out.request) : undefined
}

// within gives up on a job that has not ended within the hold of the event (793d22b3): the event goes on as received.
/**
 * @param {Ctx} $
 * @param {Promise<any>} work
 * @param {Hold} hold
 */
async function within($, work, hold) {
  work.catch(() => undefined)
  const expired = Symbol('expired')
  const outcome = await Promise.race([work, $.clock.sleep(HOOK_DEADLINE_MS).then(() => expired)])
  if (outcome !== expired) return outcome
  hold.cut = true
  throw new Error(`engram auto run still running after ${HOOK_DEADLINE_MS} ms`)
}

// hooked runs a job that reads and writes the ledger, one at a time, in the order the events came, and holds the
// event for it HOOK_DEADLINE_MS at most.
/**
 * @param {Ctx} $
 * @param {Job} job
 */
function hooked($, job) {
  const hold = { cut: false }
  const waiting = mod.hooks.then(() =>
    relay($, { ...job, ledger: mod.ledger, lastPrompt: mod.lastPrompt, transcript: mod.transcript }, hold),
  )
  const done = within($, waiting, hold)
  mod.hooks = done.then(
    (end) => {
      mod.ledger = end.ledger
    },
    () => undefined,
  )
  return done
}

/** @type {($: Ctx, job: Job, step: string | null) => Promise<string | undefined>} */
const blockOf = async ($, job, step) => (await attempt($, () => hooked($, { ...job, step }), step))?.block
/** @type {(target: any, field: string, block: string | undefined) => any} */
const withBlock = (target, field, block) => (block ? { ...target, [field]: [...(target[field] ?? []), block] } : target)

// startStep starts the in-turn cue (793b320) in the background, one at a time: the step goes on while it runs.
/**
 * @param {Ctx} $
 * @param {any} result
 */
async function startStep($, result) {
  const startedAt = await $.clock.now()
  const messages = (await $.session.messages()).slice(-STEP_WINDOW).map(({ role, text }) => ({ role, text }))
  if (mod.step && !mod.step.done) return
  /** @type {Cue} */
  const cue = { done: false, notes: [], startedAt }
  const job = {
    kind: 'step',
    step: STEP.step,
    answer: result.answer,
    toolUses: result.toolUses,
    ledger: mod.ledger,
    lastPrompt: mod.lastPrompt,
    transcript: mod.transcript,
    messages,
  }
  cue.promise = new Promise(/** @param {() => void} resolve */ (resolve) => {
    $.clock.after(0, async () => {
      cue.notes = (await attempt($, () => relay($, job), STEP.step))?.notes ?? []
      cue.done = true
      resolve()
    })
  })
  mod.step = cue
}

// heldStep answers the notes of the cue for a tool result of the working agent: it waits for the cue up to HOLD_MS; a
// cue still running is kept for a later result until its deadline, a finished one is taken.
/** @param {Ctx} $ */
async function heldStep($) {
  const cue = mod.step
  if (cue && !cue.done) await Promise.race([cue.promise, $.clock.sleep(HOLD_MS)])
  retain(cue, await kept($, cue))
  return cue?.done ? cue.notes : []
}

// retain keeps the cue for a later result unless the turn dropped it, or another cue replaced it, while it waited.
/** @type {(cue: Cue | null, still: Cue | null) => unknown} */
const retain = (cue, still) => remember(mod.step === cue ? { step: still } : {})
/** @type {($: Ctx, cue: Cue | null) => Promise<Cue | null>} */
const kept = async ($, cue) => (cue && !cue.done && !(await expired($, cue)) ? cue : null)
/** @type {($: Ctx, cue: Cue) => Promise<boolean>} */
const expired = async ($, cue) => (await $.clock.now()) - cue.startedAt > STEP_DEADLINE_MS
/** @type {(event: any) => boolean} */
const isRoute = (event) => event.tool_name === 'Skill' && event.tool_input?.skill === 'route'

// toolBlock is the context a tool result of the working agent carries: the result of route has the lookup of its own,
// any other result the notes the cue held.
/**
 * @param {Ctx} $
 * @param {any} event
 */
async function toolBlock($, event) {
  const route = isRoute(event)
  const job = {
    kind: 'tool',
    route,
    held: await (route ? [] : heldStep($)),
    args: event.tool_input?.args,
    toolResponse: event.tool_response,
  }
  return blockOf($, job, route ? STEP.route : null)
}

// queued runs a job of the queue: the work state goes in and comes back.
/**
 * @param {Ctx} $
 * @param {Job} job
 */
async function queued($, job) {
  const done = await relay($, { ...job, work: mod.work, transcript: mod.transcript })
  remember({ work: done.work })
}

// report shows the user the one line of the turn (793a5h), or nothing; it runs once the System's work has finished.
/** @param {Ctx} $ */
async function report($) {
  const done = await relay($, {
    kind: 'summary',
    tally: mod.tally.splice(0),
    warnings: mod.warnings.splice(0),
    lineShown: mod.lineShown,
  })
  remember({ lineShown: done.lineShown === true })
  if (done.show) await $.ui.status(done.line || undefined)
}

// seam is the harness-only line (793b281): when ENGRAM_AUTO_SEAM_LOG names a file, the mod records `queued` when a job
// joins the queue and `idle` when the queue is empty, so that a test waits for the System's work before it closes the
// session (a close does not wait for a job that still has a model call to make). Without the variable it is nothing.
/**
 * @param {Ctx} $
 * @param {string} line
 */
async function seam($, line) {
  const file = await $.env.get('ENGRAM_AUTO_SEAM_LOG')
  if (!file) return
  mod.queue.seam.push(line)
  await $.fs.write(file, `${mod.queue.seam.join('\n')}\n`)
}

// finish ends a burst of jobs: the housekeeping follows the last job of the queue, and the report a turn's burst.
/** @param {Ctx} $ */
async function finish($) {
  const turnEnded = mod.queue.turnEnded
  mod.queue.turnEnded = false
  await attempt($, () => queued($, { kind: 'housekeeping', turnEnded }), HOUSEKEEPING_STEP)
  if (turnEnded) await attempt($, () => report($))
}

// schedule returns at once: the job joins the queue from a timer, after the event has been answered (793b203).
/**
 * @param {Ctx} $
 * @param {Job} job
 * @param {boolean} turnEnded
 */
async function schedule($, job, turnEnded) {
  mod.queue.pending += 1
  mod.queue.turnEnded = mod.queue.turnEnded || turnEnded
  await attempt($, () => seam($, 'queued'))
  $.clock.after(0, () => {
    mod.queue.tail = mod.queue.tail
      .then(async () => {
        await attempt($, () => queued($, job), CAPTURE_STEP)
        if (mod.queue.pending === 1) await finish($)
        mod.queue.pending -= 1
        if (mod.queue.pending === 0) await attempt($, () => seam($, 'idle'))
      })
      .catch(() => undefined)
  })
}

/** @type {($: Ctx) => Promise<Record<string, any>>} */
const facts = async ($) =>
  (await attempt(
    $,
    async () => ({
      messages: await $.session.messages(),
      sessionId: await $.session.id(),
      turn: await $.session.turns(),
    }),
    CAPTURE_STEP,
  )) ?? {}

// resume takes up the turn captures an earlier session left in the queue file (793b398): the engine reads and rewrites
// the file and hands the entries back, and each joins the queue as a capture of its own.
/**
 * @param {Ctx} $
 * @param {string} start
 */
async function resume($, start) {
  const done = await attempt($, () => relay($, { kind: 'resume', start }))
  for (const entry of done?.resumed ?? []) await schedule($, { kind: 'capture', entry }, false)
}

/** @type {Hook} */
const onSessionStart = async ($, event, next) => {
  mod.transcript = event.transcript_path
  await resume($, event.source)
  return next(event)
}

/** @type {Hook} */
const onCompact = async ($, event, next) => {
  await attempt($, () => hooked($, { kind: 'compact', agent: event.agentId }))
  return next(event)
}

// A dispatch that started no subagent must not give its task to the next one that starts (793b317), and a cue is
// dropped with the turn it belongs to (793b320).
/** @type {Hook} */
const onTurnComplete = async ($, event, next) => {
  const result = await next(event)
  if (event.agentId) return result
  mod.spawns.length = 0
  mod.step = null
  const turn = { kind: 'capture', answer: event.answer ?? '', ...(await facts($)) }
  // the turn is in the queue file before the hook returns (793b398); a turn the file could not take is captured from
  // memory, as it was before the file
  const stored = await attempt($, () => relay($, { ...turn, kind: 'enqueue', transcript: mod.transcript }))
  await schedule($, stored?.queued ? { kind: 'capture', entry: stored.queued } : turn, true)
  return result
}

/** @type {Hook} */
const onSubagentStop = async ($, event, next) => {
  const result = await next(event)
  const job = {
    kind: 'subagent',
    path: event.agent_transcript_path,
    last: event.last_assistant_message ?? '',
    agent: event.agent_id,
    ...(await facts($)),
  }
  await schedule($, job, false)
  return result
}

/** @type {Hook} */
const onPrompt = async ($, event, next) => {
  mod.lastPrompt = event.text
  const job = { kind: 'lookup', moment: 'prompt', text: event.text, readContext: true }
  return next(withBlock(event, 'context', await blockOf($, job, STEP.prompt)))
}

// The step hook only reads: the chunks pass as they come and the result is the one beneath (purview, 793a4c).
/** @type {StreamHook} */
async function* onStep($, event, next) {
  const result = yield* next(event)
  if (!event.agentId && result?.toolUses?.length) await attempt($, () => startStep($, result))
  return result
}

/** @type {Hook} */
const onToolResult = async ($, event, next) => {
  const result = await next(event)
  return event.agent_id ? result : withBlock(result, 'additionalContext', await toolBlock($, event))
}

/** @type {Hook} */
const onSpawn = async (_, event, next) => {
  mod.spawns.push(event.prompt)
  return next(event)
}

/** @type {Hook} */
const onSubagentStart = async ($, event, next) => {
  const result = await next(event)
  const task = mod.spawns.shift() ?? ''
  const job = { kind: 'lookup', moment: 'subagent', agent: event.agent_id, text: task, readContext: false }
  return task === '' ? result : withBlock(result, 'additionalContext', await blockOf($, job, STEP.subagent))
}

/** @type {PassOn} */
const passOn = (_, event, next) => next(event)

/** @param {(event: string, hook: Hook | StreamHook) => any} on */
export function register(on) {
  on('classic.SessionStart', onSessionStart)
  on('session.compact', onCompact)
  on('turn.complete', onTurnComplete)
  on('classic.SubagentStop', onSubagentStop).catch(passOn)
  on('prompt.submit', onPrompt).catch(passOn)
  on('turn.step', onStep)
  on('classic.PostToolUse', onToolResult)
  on('agent.spawn', onSpawn)
  on('classic.SubagentStart', onSubagentStart)
}
