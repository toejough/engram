// The System's mod (793e2c, 793b368): an adapter. It maps the events of Claude Code to jobs of `engram auto run`, where
// every decision is made, relays the model requests a job needs, and adds what it returns through the channel each
// event has (prompt context, tool-result context, subagent-start context, the status line). The state it keeps is the
// engine's own, handed back and never read: the delivery ledger of the hook jobs and the work state of the queued ones.
const ROUND_MS = 600000
const MAX_ROUNDS = 200
// a tool result waits for the in-turn cue this long at most (the hook's own budget is about 10 s, measured), and a cue
// older than the deadline is dropped
const HOLD_MS = 7000
const STEP_DEADLINE_MS = 25000
// the steps a failure is reported against (793a4b)
const STEP = { prompt: 'the prompt lookup', route: 'the route lookup', subagent: 'the subagent lookup', step: 'the step lookup' }
const CAPTURE_STEP = 'the capture'
const HOUSEKEEPING_STEP = 'the housekeeping'
// what a failure of the engram program itself says (the engine words the failures it contains)
const REASONS = [['still running after', 'the System timed out'], ['failed to start', 'the vault is unavailable']]

const mod = {
  ledger: undefined, work: undefined, transcript: '', lastPrompt: '', lineShown: false, step: null, hooks: Promise.resolve(),
  tally: [], warnings: [], spawns: [], queue: { tail: Promise.resolve(), pending: 0, seam: [], turnEnded: false },
}
const each = (list, visit) => (list ?? []).forEach(visit)

async function attempt($, job, step) {
  try {
    return await job()
  } catch (error) {
    $.ui.log(`engram-memory: ${step ?? 'internal'}: ${error.message}`, { to: 'debug' })
    if (step) await warn($, `Automatic memory was skipped because ${reasonOf(error.message)} (${step})`)
    return null
  }
}

const reasonOf = (message) => (REASONS.find(([text]) => message.includes(text)) ?? [0, 'of an error'])[1]

// warn shows the user, on the status line and nowhere else, that automatic memory was skipped and why. It never fails.
async function warn($, text) {
  mod.warnings.push(text)
  mod.lineShown = true
  try {
    await $.ui.status(text)
  } catch (error) {
    $.ui.log(`engram-memory: the warning could not be shown: ${error.message}`, { to: 'debug' })
  }
}

function absorb($, out) {
  each(out.log, (line) => $.ui.log(line, { to: 'debug' }))
  each(out.warnings, (text) => warn($, text))
  if (out.counts) mod.tally.push(out.counts)
}

// relay runs a job to its end (793b197): one round of `engram auto run` after another, each with the replies of the
// model to the requests of the one before, which are relayed as they came and the replies as they were.
async function relay($, job) {
  let input = job
  for (let round = 0; round < MAX_ROUNDS; round += 1) {
    const result = await $.process.run(['engram', 'auto', 'run'], { stdin: JSON.stringify(input), timeoutMs: ROUND_MS })
    if (result.exitCode !== 0) throw new Error(`engram auto run exited ${result.exitCode}: ${result.stderr.slice(0, 200)}`)
    const out = JSON.parse(result.stdout)
    absorb($, out)
    if (out.done) return out.done
    input = { ...job, state: out.state, answers: await answers($, out) }
  }
  throw new Error('the job did not end')
}

async function answers($, out) {
  if (out.waitMs) await $.clock.sleep(out.waitMs)
  const replies = []
  for (const request of out.requests ?? []) replies.push(await $.model.complete(request))
  return replies
}

// hooked runs a job that reads and writes the ledger, one at a time, in the order the events came.
function hooked($, job) {
  const done = mod.hooks.then(() => relay($, { ...job, ledger: mod.ledger, lastPrompt: mod.lastPrompt, transcript: mod.transcript }))
  mod.hooks = done.then((end) => { mod.ledger = end.ledger }, () => undefined)
  return done
}

const blockOf = async ($, job, step) => (await attempt($, () => hooked($, { ...job, step }), step))?.block
const withBlock = (target, field, block) => (block ? { ...target, [field]: [...(target[field] ?? []), block] } : target)

// startStep starts the in-turn cue (793b320) in the background, one at a time: the step goes on while it runs.
async function startStep($, result) {
  if (mod.step && !mod.step.done) return
  const cue = { done: false, notes: [], startedAt: await $.clock.now() }
  const job = { kind: 'step', step: STEP.step, answer: result.answer, toolUses: result.toolUses, ledger: mod.ledger, lastPrompt: mod.lastPrompt, transcript: mod.transcript }
  job.messages = await $.session.messages()
  cue.promise = new Promise((resolve) => {
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
async function heldStep($) {
  const cue = mod.step
  if (cue && !cue.done) await Promise.race([cue.promise, $.clock.sleep(HOLD_MS)])
  mod.step = await kept($, cue)
  return cue?.done ? cue.notes : []
}

const kept = async ($, cue) => (cue && !cue.done && !(await expired($, cue)) ? cue : null)
const expired = async ($, cue) => (await $.clock.now()) - cue.startedAt > STEP_DEADLINE_MS
const isRoute = (e) => e.tool_name === 'Skill' && e.tool_input?.skill === 'route'

// toolBlock is the context a tool result of the working agent carries: the result of route has the lookup of its own,
// any other result the notes the cue held.
async function toolBlock($, e) {
  const route = isRoute(e)
  const job = { kind: 'tool', route, held: await (route ? [] : heldStep($)), args: e.tool_input?.args, toolResponse: e.tool_response }
  return blockOf($, job, route ? STEP.route : null)
}

// queued runs a job of the queue: the work state goes in and comes back.
async function queued($, job) {
  const done = await relay($, { ...job, work: mod.work, transcript: mod.transcript })
  mod.work = done.work
}

// report shows the user the one line of the turn (793a5h), or nothing; it runs once the System's work has finished.
async function report($) {
  const done = await relay($, { kind: 'summary', tally: mod.tally.splice(0), warnings: mod.warnings.splice(0), lineShown: mod.lineShown })
  mod.lineShown = done.lineShown === true
  if (done.show) await $.ui.status(done.line || undefined)
}

// seam is the harness-only line (793b281): when ENGRAM_AUTO_SEAM_LOG names a file, the mod records `queued` when a job
// joins the queue and `idle` when the queue is empty, so that a test waits for the System's work before it closes the
// session (a close does not wait for a job that still has a model call to make). Without the variable it is nothing.
async function seam($, line) {
  const file = await $.env.get('ENGRAM_AUTO_SEAM_LOG')
  if (!file) return
  mod.queue.seam.push(line)
  await $.fs.write(file, `${mod.queue.seam.join('\n')}\n`)
}

// finish ends a burst of jobs: the housekeeping follows the last job of the queue, and the report a turn's burst.
async function finish($) {
  const turnEnded = mod.queue.turnEnded
  mod.queue.turnEnded = false
  await attempt($, () => queued($, { kind: 'housekeeping', turnEnded }), HOUSEKEEPING_STEP)
  if (turnEnded) await attempt($, () => report($))
}

// schedule returns at once: the job joins the queue from a timer, after the event has been answered (793b203).
async function schedule($, job, turnEnded) {
  mod.queue.pending += 1
  mod.queue.turnEnded = mod.queue.turnEnded || turnEnded
  await attempt($, () => seam($, 'queued'))
  $.clock.after(0, () => {
    mod.queue.tail = mod.queue.tail.then(async () => {
      await attempt($, () => queued($, job), CAPTURE_STEP)
      if (mod.queue.pending === 1) await finish($)
      mod.queue.pending -= 1
      if (mod.queue.pending === 0) await attempt($, () => seam($, 'idle'))
    }).catch(() => undefined)
  })
}

const facts = async ($) => (await attempt($, async () => ({ messages: await $.session.messages(), sessionId: await $.session.id(), turn: await $.session.turns() }), CAPTURE_STEP)) ?? {}

export function register(on) {
  on('classic.SessionStart', async ($, e, next) => {
    mod.transcript = e.transcript_path
    return next(e)
  })

  on('session.compact', async ($, e, next) => {
    await attempt($, () => hooked($, { kind: 'compact' }))
    return next(e)
  })

  on('turn.complete', async ($, e, next) => {
    const result = await next(e)
    if (e.agentId) return result
    // a dispatch that started no subagent must not give its task to the next one that starts (793b317), and a cue is
    // dropped with the turn it belongs to (793b320)
    mod.spawns.length = 0
    mod.step = null
    await schedule($, { kind: 'capture', answer: e.answer ?? '', ...(await facts($)) }, true)
    return result
  })

  on('classic.SubagentStop', async ($, e, next) => {
    const result = await next(e)
    await schedule($, { kind: 'subagent', path: e.agent_transcript_path, last: e.last_assistant_message ?? '', agent: e.agent_id, ...(await facts($)) }, false)
    return result
  }).catch(($, e, next) => next(e))

  on('prompt.submit', async ($, e, next) => {
    mod.lastPrompt = e.text
    return next(withBlock(e, 'context', await blockOf($, { kind: 'lookup', moment: 'prompt', text: e.text, readContext: true }, STEP.prompt)))
  }).catch(($, e, next) => next(e))

  // The step hook only reads: the chunks pass as they come and the result is the one beneath (purview, 793a4c).
  on('turn.step', async function* ($, e, next) {
    const result = yield* next(e)
    if (!e.agentId && result?.toolUses?.length) await attempt($, () => startStep($, result))
    return result
  })

  on('classic.PostToolUse', async ($, e, next) => {
    const result = await next(e)
    return e.agent_id ? result : withBlock(result, 'additionalContext', await toolBlock($, e))
  })

  on('agent.spawn', async ($, e, next) => {
    mod.spawns.push(e.prompt)
    return next(e)
  })

  on('classic.SubagentStart', async ($, e, next) => {
    const result = await next(e)
    const task = mod.spawns.shift() ?? ''
    const job = { kind: 'lookup', moment: 'subagent', agent: e.agent_id, text: task, readContext: false }
    return task === '' ? result : withBlock(result, 'additionalContext', await blockOf($, job, STEP.subagent))
  })
}
