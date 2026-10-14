// A shared stub of the engine under the mod, with latencies, for the performance and hardening gates (793e8o, 793e8p):
// the process, the model, the file system, the session and the status line are stubbed, and every call is recorded with
// the time of the mock clock at which it started. A call that takes longer than its deadline is cut at the deadline, as
// the platform cuts it. The latencies are the baselines of the tests that use them; nothing here is live.
import { mock } from 'claude-code/testing'
import type { Driver, MockClock, On } from 'claude-code/testing'

export const NOTE_OUTPUT = `version: 1
phrases:
  - declaring work done
items:
  - path: 1.2026-10-12.note-n.md
    kind: fact
    score: 0.49
    provenances:
      - direct
    content: |+
      ---
      type: fact
      situation: declaring a Go package rename done
      object: run go build first (SENTINEL-N)
      ---

      Information learned: run go build first (SENTINEL-N).

clusters:
`
// noteOutput is the `engram query` output holding one note of its own name, so that tests in one file (the mod keeps
// its delivery ledger for the whole process) never see each other's deliveries.
export function noteOutput(name: string): string {
  return NOTE_OUTPUT.replace('1.2026-10-12.note-n.md', name).replace('SENTINEL-N', `SENTINEL-${name}`)
}
export const EMPTY_OUTPUT = 'version: 1\nphrases: []\nitems: []\nclusters: []\n'

export const LESSON = {
  kind: 'feedback',
  slug: 'use-targ-not-go-test',
  situation: 'running Go tests in this repo',
  covered: null,
  behavior: 'ran go test directly',
  impact: 'skips the repo gates',
  action: 'use targ test',
}
export const MESSAGES = [
  { role: 'assistant', text: 'I will run go test ./...', toolUses: [] },
  { role: 'user', text: 'No, use targ, not go test directly.', toolUses: [] },
  { role: 'assistant', text: 'Understood, I will use targ.', toolUses: [] },
]
export const COMPLETE = {
  answer: 'Understood, I will use targ.',
  durationMs: 5,
  isAborted: false,
  turnId: 't1',
  reason: 'answer',
}
export const PROMPT = {
  text: 'I just finished renaming the package. Is it done?',
  wait: false,
  origin: { kind: 'sdk' },
}

export type Latency = Record<string, number>
export type Kind = 'classify' | 'relevance' | 'capture' | 'curation'

export interface Call {
  atMs: number
  what: string
  argv?: string[]
  prompt?: string
}
export interface Write {
  atMs: number
  path: string
  text: string
}
export interface Shown {
  atMs: number
  text: string | undefined
}
export interface ProcessResult {
  exitCode: number
  stdout: string
  stderr: string
}
export interface WorldOptions {
  // milliseconds a process (by subcommand, or program) or a model call (by kind) takes; absent means instantaneous
  latency?: Latency
  query?: string
  // answers a process by subcommand (or program) in place of the default
  process?: Record<string, ProcessResult | ((event: { argv: string[] }) => ProcessResult)>
  replies?: Partial<Record<Kind, string>>
  files?: Record<string, string | ((atMs: number) => string | null)>
  env?: Record<string, string>
  messages?: unknown[]
  // a platform that does not cut a call at its deadline: the stubs wait their whole latency
  ignoreDeadlines?: boolean
}
export interface World {
  clock: MockClock
  calls: Call[]
  writes: Write[]
  shown: Shown[]
}

const DEFAULT_REPLY: Record<Kind, string> = {
  classify: JSON.stringify([{ moment: 'declaring work done', subject: 'the rename of package util to helpers' }]),
  relevance: '[1]',
  capture: JSON.stringify([LESSON]),
  curation: '{"outcome":"absent","into":null,"fields":null}',
}

function kindOf(prompt: string): Kind {
  if (prompt.includes('You watch an AI coding agent')) return 'classify'
  if (prompt.includes('which vault notes bear')) return 'relevance'
  if (prompt.includes('keep the long-term memory')) return 'capture'
  return 'curation'
}

// world registers the stubs and returns the records. `HOME` is set so that the mod's directories resolve.
export function world(on: On, options: WorldOptions = {}): World {
  const latency = options.latency ?? {}
  const clock = mock.clock(on)
  const calls: Call[] = []
  const writes: Write[] = []
  const shown: Shown[] = []
  mock.env(on, { HOME: '/home/t', ENGRAM_VAULT_PATH: '/vault', ...options.env })
  on('process.run', async (_ctx, event) => {
    const argv: string[] = event.argv
    const key = argv[0] === 'engram' ? argv[1] : argv[0]
    calls.push({ atMs: await clock.now(), what: key, argv })
    const wait = latency[key] ?? 0
    const deadline: number = options.ignoreDeadlines ? Infinity : (event.init?.timeoutMs ?? Infinity)
    if (wait > deadline) {
      await clock.sleep(deadline)
      throw new Error(`$.process.run(${argv[0]}) aborted: still running after ${deadline}ms`)
    }
    if (wait > 0) await clock.sleep(wait)
    const answer = options.process?.[key]
    if (answer !== undefined) return { value: typeof answer === 'function' ? answer({ argv }) : answer }
    const defaults: Record<string, string> = {
      query: options.query ?? EMPTY_OUTPUT,
      learn: '/vault/9.2026-10-14.written.md\n',
    }
    const stdout = defaults[key] ?? ''
    return { value: { exitCode: key === 'grep' ? 1 : 0, stdout, stderr: '' } }
  })
  on('model.complete', async (_ctx, event) => {
    const kind = kindOf(event.prompt)
    calls.push({ atMs: await clock.now(), what: `model:${kind}`, prompt: event.prompt })
    const wait = latency[kind] ?? 0
    const deadline: number = options.ignoreDeadlines ? Infinity : (event.timeoutMs ?? Infinity)
    if (wait > deadline) {
      await clock.sleep(deadline)
      return { value: { isAnswered: false, reason: 'aborted', usage: {} } }
    }
    if (wait > 0) await clock.sleep(wait)
    return {
      value: {
        isAnswered: true,
        text: options.replies?.[kind] ?? DEFAULT_REPLY[kind],
        usage: { input_tokens: 1, output_tokens: 1 },
      },
    }
  })
  on('fs.read', async (_ctx, event) => {
    const file = options.files?.[event.path]
    const text = typeof file === 'function' ? file(await clock.now()) : file
    return text === undefined || text === null ? { deny: 'ENOENT' } : { value: text }
  })
  on('fs.write', async (_ctx, event) => {
    writes.push({ atMs: await clock.now(), path: event.path, text: event.text ?? event.data ?? '' })
    return { value: undefined }
  })
  on('fs.exists', async () => ({ value: true }))
  on('ui.log', async () => ({ value: undefined }))
  on('ui.status', async (_ctx, event) => {
    shown.push({ atMs: await clock.now(), text: event.text })
    return { value: undefined }
  })
  on('session.id', async () => ({ value: 'sess-1' }))
  on('session.turns', async () => ({ value: 3 }))
  on('session.messages', async () => ({ value: options.messages ?? MESSAGES }))
  on('turn.complete', async (_ctx, event) => ({ text: event.answer }))
  on('prompt.submit', async (_ctx, event) => ({ text: event.text, context: event.context }))
  on('agent.spawn', async () => ({ model: 'sonnet' }))
  on('classic.SubagentStop', async () => ({}))
  on('classic.SubagentStart', async () => ({}))
  on('classic.SessionStart', async () => ({}))
  on('classic.PostToolUse', async () => ({}))
  on('turn.step', async function* (_ctx: unknown, event: { turnId: string; index: number }) {
    yield { kind: 'text', index: 0, text: 'chunk' }
    return { ...STEP_RESULT, turnId: event.turnId, index: event.index }
  })
  on('session.compact', async (_ctx, event) => ({ messages: event.messages }))
  return { clock, calls, writes, shown }
}

export const STEP_MS = 50
export const LIMIT_MS = 120_000

// drive advances the mock clock in steps until the work has finished, and returns its value with the time it took on
// that clock; the resolution is one step.
export async function drive<T>(
  clock: MockClock,
  work: Promise<T>,
  step = STEP_MS,
): Promise<{ value: T; elapsedMs: number }> {
  let finished = false
  const tracked = work.then((value) => {
    finished = true
    return value
  })
  const start = await clock.now()
  while (!finished) {
    if ((await clock.now()) - start > LIMIT_MS)
      throw new Error(`the work did not finish within ${LIMIT_MS} ms of the mock clock`)
    await clock.advance(step)
  }
  return { value: await tracked, elapsedMs: (await clock.now()) - start }
}

// advanceUntil advances the mock clock in steps until the condition holds, and returns the time it took.
export async function advanceUntil(clock: MockClock, condition: () => boolean, step = STEP_MS): Promise<number> {
  const start = await clock.now()
  while (!condition()) {
    if ((await clock.now()) - start > LIMIT_MS)
      throw new Error(`the condition did not hold within ${LIMIT_MS} ms of the mock clock`)
    await clock.advance(step)
  }
  return (await clock.now()) - start
}

// The in-turn cue: a step of the working agent that wrote text and called a tool, and the tool result that follows it.
export const STEP_INPUT = { turnId: 't1', index: 1, model: 'm', messageCount: 5 }
const STEP_ANSWER = 'The rename looks complete; I will grep for leftovers.'
export const STEP_RESULT = {
  turnId: 't1',
  index: 1,
  answer: STEP_ANSWER,
  toolUses: [{ name: 'Grep', input: { pattern: 'util' } }],
  stopReason: 'tool_use',
  usage: {},
}
export const STEP_MESSAGES = [
  { role: 'user', text: 'Follow the instructions in task.md.', toolUses: [] },
  { role: 'assistant', text: STEP_ANSWER, toolUses: [{ tool: 'Grep' }] },
]
export const TOOL = {
  tool_name: 'Grep',
  tool_input: { pattern: 'util' },
  tool_response: 'no matches',
  tool_use_id: 'u1',
}

// runStep drives one step through the chain: its chunks, then its result (the stub of the step is registered by world).
export async function runStep($: Driver): Promise<void> {
  const stream = $.turn.step(STEP_INPUT)
  let next = await stream.next()
  while (!next.done) next = await stream.next()
}

// An offer as `engram show` prints it.
export function show(situation: string, hash = 'xh1:abc'): string {
  const front = `type: fact\nsituation: ${situation}\nsubject: ${situation}\npending: true`
  const links = '# outbound links (fetch with: engram show <basename>)\n(none)\n'
  return `# exchange_hash: ${hash}\n---\n${front}\n---\n\nInformation learned: ${situation} claim.\n\n\n${links}`
}
