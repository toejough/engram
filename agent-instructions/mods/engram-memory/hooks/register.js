// The System's mod (793e2c): maps the events of the delivery moments to the moment job and adds what it returns
// through the channel each event has: prompt context, tool-result context, subagent-start context (793b192).
import { relevanceRequest, parseSelection } from './judge.js'
import { candidates, buildBlock, momentText, parseItems, present, queryArgv, sinceCompaction } from './moment.js'

const MAIN = 'main'
// the dispatch prompts of agent.spawn, in order, for the SubagentStart events that follow (SP5: start carries no prompt)
const spawns = []
const ledgers = new Map()
const state = {
  transcript: null,
  lastPrompt: '',
  ledger(agent) {
    if (!ledgers.has(agent)) ledgers.set(agent, new Set())
    return ledgers.get(agent)
  },
}

// attempt contains a failure of the job: the event goes on as if the mod were not there (793a4b).
async function attempt($, job) {
  try {
    return await job()
  } catch (error) {
    $.ui.log(`engram-memory: ${error.message}`, { to: 'debug' })
    return null
  }
}

// readTranscript reads the session's transcript file; the file does not exist before the first row is written.
async function readTranscript($, file) {
  try {
    return await $.fs.read(file)
  } catch {
    return ''
  }
}

// moment runs one lookup (793b192): query, presence, relevance. `agent` keys the ledger; a subagent's context is its
// task, so it reads no transcript. Nothing relevant remaining is nothing delivered (793a2n).
async function moment($, { agent, kind, text, readContext }) {
  const result = await $.process.run(queryArgv(kind, text))
  if (result.exitCode !== 0) throw new Error(`engram query exited ${result.exitCode}: ${result.stderr.slice(0, 200)}`)
  const ledger = state.ledger(agent)
  const context = readContext && state.transcript ? sinceCompaction(await readTranscript($, state.transcript)) : ''
  const fresh = candidates(parseItems(result.stdout)).filter((note) => !present(note, ledger, context))
  if (fresh.length === 0) return null
  const reply = await $.model.complete(relevanceRequest(kind, text, fresh))
  const chosen = reply.isAnswered ? parseSelection(reply.text, fresh.length).map((index) => fresh[index]) : []
  if (chosen.length === 0) return null
  for (const note of chosen) ledger.add(note.path)
  return buildBlock(chosen)
}

export function register(on) {
  on('classic.SessionStart', async ($, e, next) => {
    state.transcript = e.transcript_path
    return next(e)
  })

  on('session.compact', async ($, e, next) => {
    state.ledger(MAIN).clear()
    return next(e)
  })

  on('prompt.submit', async ($, e, next) => {
    state.lastPrompt = e.text
    const block = await attempt($, () => moment($, { agent: MAIN, kind: 'prompt', text: momentText(e.text), readContext: true }))
    return next(block ? { ...e, context: [...(e.context ?? []), block] } : e)
  }).catch(($, e, next) => next(e))

  on('classic.PostToolUse', async ($, e, next) => {
    const result = await next(e)
    if (e.tool_name !== 'Skill' || e.tool_input?.skill !== 'route' || e.agent_id) return result
    const text = `${e.tool_input.args ?? ''} ${state.lastPrompt}`.trim()
    const block = await attempt($, () => moment($, { agent: MAIN, kind: 'route', text, readContext: true }))
    return block ? { ...result, additionalContext: [...(result?.additionalContext ?? []), block] } : result
  })

  on('agent.spawn', async ($, e, next) => {
    spawns.push(e.prompt)
    return next(e)
  })

  on('classic.SubagentStart', async ($, e, next) => {
    const result = await next(e)
    const task = spawns.shift() ?? ''
    if (task === '') return result
    const block = await attempt($, () => moment($, { agent: e.agent_id, kind: 'subagent', text: task, readContext: false }))
    return block ? { ...result, additionalContext: [...(result?.additionalContext ?? []), block] } : result
  })
}
