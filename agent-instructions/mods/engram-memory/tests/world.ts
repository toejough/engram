// A stub of the engine under the mod, for the mod's contract tests (793e8m): the mod only wires events to jobs of
// `engram auto run`, so the engine is a script that answers each job the mod hands over, and everything else the mod
// touches (the model, the file system, the session, the status line, the clock) is a recorder. Nothing here is live.
import { expect, mock } from 'claude-code/testing'
import type { MockClock, On } from 'claude-code/testing'

export interface Job {
  kind: string
  [field: string]: unknown
}
// what the script answers a job with: a response of the engine, or how the program itself fails
export type Answer = Record<string, unknown> | { exit: number; stderr: string } | { reject: string }
export interface WorldOptions {
  engine?: (job: Job, call: number) => Answer | Promise<Answer>
  messages?: unknown[]
  files?: Record<string, string>
  env?: Record<string, string>
  replies?: (request: Record<string, unknown>) => Record<string, unknown>
}
export interface World {
  clock: MockClock
  jobs: Job[]
  requests: Record<string, unknown>[]
  shown: (string | undefined)[]
  logs: string[]
  writes: { path: string; text: string }[]
}

export const COMPLETE = { answer: 'Understood.', durationMs: 5, isAborted: false, turnId: 't1', reason: 'answer' }
export const PROMPT = { text: 'I just finished renaming the package. Is it done?', wait: false, origin: { kind: 'sdk' } }
export const MESSAGES = [
  { role: 'user', text: 'No, use targ, not go test directly.', toolUses: [] },
  { role: 'assistant', text: 'Understood.', toolUses: [] },
]
export const STEP_INPUT = { turnId: 't1', index: 1, model: 'm', messageCount: 5 }
export const STEP_RESULT = {
  turnId: 't1',
  index: 1,
  answer: 'The rename looks complete; I will grep for leftovers.',
  toolUses: [{ name: 'Grep', input: { pattern: 'util' } }],
  stopReason: 'tool_use',
  usage: {},
}
export const TOOL = { tool_name: 'Grep', tool_input: { pattern: 'util' }, tool_response: 'no matches', tool_use_id: 'u1' }
export const NOTE = { Path: '1.2026-10-12.note-n.md', Kind: 'fact', Situation: 'renaming a package', Body: 'run go build first' }
export const BLOCK = '[engram memory] a block'

export function world(on: On, options: WorldOptions = {}): World {
  const clock = mock.clock(on)
  const jobs: Job[] = []
  const requests: Record<string, unknown>[] = []
  const shown: (string | undefined)[] = []
  const logs: string[] = []
  const writes: { path: string; text: string }[] = []
  mock.env(on, { HOME: '/home/t', ...options.env })
  on('process.run', async (_ctx, event) => {
    const job = JSON.parse(event.init.stdin) as Job
    jobs.push(job)
    const answer = (await (options.engine?.(job, jobs.length) ?? { done: {} })) as Record<string, unknown>
    // the engine rejects a call it cannot complete; in a test a hook beneath the mod stands for it with a denial
    if ('reject' in answer) return { deny: String(answer.reject) }
    if ('exit' in answer) return { value: { exitCode: answer.exit, stdout: '', stderr: answer.stderr } }
    return { value: { exitCode: 0, stdout: JSON.stringify(answer), stderr: '' } }
  })
  on('model.complete', async (_ctx, event) => {
    requests.push(event)
    const reply = options.replies?.(event) ?? { isAnswered: true, text: '[1]', usage: {} }
    return { value: reply }
  })
  on('fs.read', async (_ctx, event) => {
    const text = options.files?.[event.path]
    return text === undefined ? { deny: 'ENOENT' } : { value: text }
  })
  on('fs.write', async (_ctx, event) => {
    writes.push({ path: event.path, text: event.text ?? event.data ?? '' })
    return { value: undefined }
  })
  on('ui.log', async (_ctx, event) => {
    logs.push(event.text ?? event.line ?? JSON.stringify(event))
    return { value: undefined }
  })
  on('ui.status', async (_ctx, event) => {
    shown.push(event.text)
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
  on('session.compact', async (_ctx, event) => ({ messages: event.messages }))
  on('turn.step', async function* (_ctx: unknown, event: { turnId: string; index: number }) {
    yield { kind: 'text', index: 0, text: 'chunk' }
    return { ...STEP_RESULT, turnId: event.turnId, index: event.index }
  })
  return { clock, jobs, requests, shown, logs, writes }
}

// expectHas asserts that a job holds the fields given, with these values.
export function expectHas(job: Job, fields: Record<string, unknown>): void {
  const picked = Object.fromEntries(Object.keys(fields).map((key) => [key, job[key]]))
  expect(picked).toEqual(fields)
}
