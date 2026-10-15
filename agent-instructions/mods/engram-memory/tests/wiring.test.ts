// Contract tests of the mod as an adapter (793e8m): the events it maps to jobs of `engram auto run`, the relay of the
// model requests a job needs, what it does with the effects of a round, the order of the queue and the summary of a
// burst. The logic behind each job is tested in Go (internal/auto); here the engine is a script.
import { describe, expect, test } from 'claude-code/testing'
import type { Driver, On } from 'claude-code/testing'

import { BLOCK, COMPLETE, MESSAGES, NOTE, PROMPT, STEP_INPUT, STEP_RESULT, TOOL, expectHas, world } from './world.js'
import type { Answer, Job } from './world.js'

const REQUEST = { model: 'haiku', effort: 'low', prompt: 'which vault notes bear', maxTokens: 200, timeoutMs: 3000 }
const REPLY = { isAnswered: true, text: '[1]', usage: {} }

// asks answers a job's first round with a need for the request and its second with the block.
const asks = (_job: Job, call: number): Answer =>
  call % 2 === 1 ? { request: REQUEST, state: ['s'] } : { done: { block: BLOCK } }

async function runStep($: Driver): Promise<void> {
  const stream = $.turn.step(STEP_INPUT)
  let next = await stream.next()
  while (!next.done) next = await stream.next()
}

describe('the relay of a job (793b197)', () => {
  test(
    'a request is relayed as it came, its reply and the ' + 'state go back, and the end is the answer',
    async ($: Driver, on: On) => {
      const { jobs, requests } = world(on, { engine: asks, replies: () => REPLY })
      const result = await $.prompt.submit(PROMPT)
      expect(requests).toEqual([REQUEST])
      expect(jobs.length).toBe(2)
      expectHas(jobs[0], {
        kind: 'lookup',
        moment: 'prompt',
        text: PROMPT.text,
        readContext: true,
        step: 'the prompt lookup',
      })
      expectHas(jobs[1], { kind: 'lookup', state: ['s'], reply: REPLY })
      expect(result.context).toEqual([BLOCK])
    },
  )

  test('a wait is slept on the clock before the next round', async ($: Driver, on: On) => {
    const { clock, jobs } = world(on, {
      engine: (_job, call) => (call === 1 ? { waitMs: 500, state: [] } : { done: {} }),
    })
    const submitted = $.prompt.submit(PROMPT)
    await clock.advance(499)
    expect(jobs.length).toBe(1)
    await clock.advance(1)
    await submitted
    expect(jobs.length).toBe(2)
  })

  test(
    'the effects of a round are applied: debug lines, warnings ' + 'on the status line, counts in the summary',
    async ($: Driver, on: On) => {
      const engine = (job: Job): Answer => {
        if (job.kind === 'summary') return { done: { show: true, line: 'the line', lineShown: true } }
        if (job.kind !== 'lookup') return { done: {} }
        return { log: ['engram-memory: a line'], warnings: ['a warning'], counts: { delivered: 2 }, done: {} }
      }
      const { clock, jobs, logs, shown } = world(on, { engine })
      await $.prompt.submit(PROMPT)
      expect(logs).toEqual(['engram-memory: a line'])
      expect(shown).toEqual(['a warning'])
      await $.turn.complete(COMPLETE)
      await clock.settle()
      const summary = jobs.find((job) => job.kind === 'summary') ?? { kind: '' }
      expectHas(summary, { tally: [{ delivered: 2 }], warnings: ['a warning'], lineShown: true })
      expect(shown.at(-1)).toBe('the line')
    },
  )

  const FAILURES: [string, Answer, string][] = [
    ['the program exits with an error', { exit: 70, stderr: 'injected failure' }, 'of an error'],
    [
      'the program cannot start',
      { reject: '$.process.run(engram) failed to start: ENOENT' },
      'the vault is unavailable',
    ],
    [
      'the program is cut at its deadline',
      { reject: '$.process.run(engram) aborted: still running after 600000ms' },
      'the System timed out',
    ],
  ]
  for (const [name, answer, reason] of FAILURES) {
    test(`793d22b3 when ${name} the prompt goes on as typed and the user is told why`, async ($: Driver, on: On) => {
      const { shown } = world(on, { engine: () => answer })
      const result = await $.prompt.submit(PROMPT)
      expect(result.text).toBe(PROMPT.text)
      expect(result.context ?? []).toEqual([])
      expect(shown).toEqual([`Automatic memory was skipped because ${reason} (the prompt lookup)`])
    })
  }
})

describe('the hold of an event (793d22b3)', () => {
  const LOOKUP_SILENT = (job: Job, call: number): Promise<Answer> | Answer =>
    job.kind === 'lookup' && call === 1 ? new Promise<Answer>(() => {}) : { done: { block: BLOCK } }

  test(
    '793d22b3 a job that holds the prompt past 8 s is cut: the prompt goes on as typed, the user is told, ' +
      'and the next prompt is served',
    async ($: Driver, on: On) => {
      const { clock, jobs, shown } = world(on, { engine: LOOKUP_SILENT })
      const submitted = $.prompt.submit(PROMPT)
      await clock.advance(7999)
      expect(shown).toEqual([])
      await clock.advance(1)
      const result = await submitted
      expect(result.text).toBe(PROMPT.text)
      expect(result.context ?? []).toEqual([])
      expect(shown).toEqual(['Automatic memory was skipped because the System timed out (the prompt lookup)'])
      const next = await $.prompt.submit(PROMPT)
      expect(next.context).toEqual([BLOCK])
      expect(jobs.filter((job) => job.kind === 'lookup').length).toBe(2)
    },
  )

  test('793d22b3 a job that ends within 8 s is not cut, and no timer keeps it waiting', async ($: Driver, on: On) => {
    const { clock, shown } = world(on, { engine: () => ({ done: { block: BLOCK } }) })
    const result = await $.prompt.submit(PROMPT)
    expect(result.context).toEqual([BLOCK])
    await clock.advance(8000)
    expect(shown).toEqual([])
  })

  test(
    '793d22b3 a round that would start after the cut is not started',
    async ($: Driver, on: On) => {
      let rounds = 0
      const slow = async (): Promise<Answer> => {
        rounds += 1
        return { waitMs: 5000, state: ['s'] }
      }
      const { clock, shown } = world(on, { engine: slow })
      const submitted = $.prompt.submit(PROMPT)
      await clock.advance(8000)
      await submitted
      await clock.advance(20000)
      expect(shown.length).toBe(1)
      expect(rounds).toBe(2)
    },
  )
})

describe('the moments (793d22b1)', () => {
  test(
    'the ledger the engine hands back goes with the next ' + 'job, and a compaction is a job of its own',
    async ($: Driver, on: On) => {
      const { jobs } = world(on, { engine: () => ({ done: { ledger: { main: ['a.md'] } } }) })
      await $.classic.SessionStart({ source: 'startup', transcript_path: '/t/s.jsonl' })
      await $.prompt.submit(PROMPT)
      await $.session.compact({ trigger: 'auto', messages: MESSAGES })
      await $.prompt.submit(PROMPT)
      expect(jobs.map((job) => job.kind)).toEqual(['lookup', 'compact', 'lookup'])
      expectHas(jobs[0], { transcript: '/t/s.jsonl', lastPrompt: PROMPT.text })
      expect(jobs[1].ledger).toEqual({ main: ['a.md'] })
      expect(jobs[2].ledger).toEqual({ main: ['a.md'] })
    },
  )

  test('the jobs that touch the ledger run one at a time, in the order the events came', async ($: Driver, on: On) => {
    const { clock, jobs } = world(on, {
      engine: async (job) => {
        await Promise.resolve()
        return { done: { ledger: { main: [String(job.text)] } } }
      },
    })
    await Promise.all([$.prompt.submit({ ...PROMPT, text: 'one' }), $.prompt.submit({ ...PROMPT, text: 'two' })])
    await clock.settle()
    expect(jobs.map((job) => job.ledger)).toEqual([undefined, { main: ['one'] }])
  })

  test(
    'the result of a tool carries the block of the route lookup, and a ' +
      'subagent task is given to the subagent that starts',
    async ($: Driver, on: On) => {
      const { jobs } = world(on, { engine: () => ({ done: { block: BLOCK } }) })
      const route = await $.classic.PostToolUse({
        tool_name: 'Skill',
        tool_input: { skill: 'route', args: 'write tests' },
        tool_response: '',
        tool_use_id: 'r',
      })
      expect(route.additionalContext).toEqual([BLOCK])
      expectHas(jobs[0], { kind: 'tool', route: true, args: 'write tests' })
      await $.agent.spawn({
        tool_use_id: 't',
        prompt: 'add tests for the cli',
        description: 'tests',
        subagentType: 'general-purpose',
      })
      const start = await $.classic.SubagentStart({ agent_id: 'a1', agent_type: 'general-purpose' })
      expect(start.additionalContext).toEqual([BLOCK])
      expectHas(jobs[1], {
        kind: 'lookup',
        moment: 'subagent',
        agent: 'a1',
        text: 'add tests for the cli',
        readContext: false,
      })
      expect(
        (await $.classic.SubagentStart({ agent_id: 'a2', agent_type: 'general-purpose' })).additionalContext ?? [],
      ).toEqual([])
      expect(jobs.length).toBe(2)
    },
  )

  test(
    'a subagent gets nothing of the main agent, and a dispatch ' + 'that started no subagent gives its task to none',
    async ($: Driver, on: On) => {
      const { clock, jobs } = world(on, { engine: () => ({ done: { block: BLOCK } }) })
      expect((await $.classic.PostToolUse({ ...TOOL, agent_id: 'a1' })).additionalContext ?? []).toEqual([])
      await $.agent.spawn({
        tool_use_id: 't',
        prompt: 'never started',
        description: 'x',
        subagentType: 'general-purpose',
      })
      await $.turn.complete(COMPLETE)
      await clock.settle()
      await $.classic.SubagentStart({ agent_id: 'a1', agent_type: 'general-purpose' })
      expect(jobs.some((job) => job.moment === 'subagent')).toBe(false)
    },
  )
})

describe('a compaction (793d24b2)', () => {
  test('names the subagent that was compacted, and no agent when it was the main agent', async ($: Driver, on: On) => {
    const { jobs } = world(on)
    await $.session.compact({ trigger: 'auto', messages: MESSAGES })
    await $.session.compact({ trigger: 'auto', messages: MESSAGES, agentId: 'a1' })
    expect(jobs.map((job) => job.kind)).toEqual(['compact', 'compact'])
    expect(jobs[0].agent).toBeUndefined()
    expect(jobs[1].agent).toBe('a1')
  })
})

describe('the in-turn cue (793d23b2)', () => {
  const cue = (job: Job): Answer => (job.kind === 'step' ? { done: { notes: [NOTE] } } : { done: { block: BLOCK } })

  test(
    'a step that called a tool starts the job, and the ' + 'notes ride the next tool result',
    async ($: Driver, on: On) => {
      const { clock, jobs } = world(on, { engine: cue })
      await runStep($)
      await clock.settle()
      expectHas(jobs[0], { kind: 'step', step: 'the step lookup', toolUses: STEP_RESULT.toolUses })
      const tool = await $.classic.PostToolUse(TOOL)
      expect(tool.additionalContext).toEqual([BLOCK])
      expectHas(jobs[1], { kind: 'tool', route: false, held: [NOTE] })
    },
  )

  test(
    '793d23b2 a step hands the job the last 50 messages with their role and text, not the whole session',
    async ($: Driver, on: On) => {
      const HUGE = 'x'.repeat(100000)
      const session = Array.from({ length: 5000 }, (_, index) => ({
        role: index % 2 === 0 ? 'user' : 'assistant',
        text: `message ${index}`,
        toolUses: [{ tool: 'Read' }],
        toolResults: [HUGE],
      }))
      const { clock, jobs } = world(on, { engine: cue, messages: session })
      await runStep($)
      await clock.settle()
      const sent = jobs[0].messages as { role: string; text: string }[]
      expect(sent.length).toBe(50)
      expect(sent.at(-1)).toEqual({ role: 'assistant', text: 'message 4999' })
      expect(Object.keys(sent[0]).sort()).toEqual(['role', 'text'])
      expect(JSON.stringify(jobs[0]).length).toBeLessThanOrEqual(20000)
    },
  )

  test(
    'no job is started for a step of a subagent or while another ' + 'is running; the cue is dropped with its turn',
    async ($: Driver, on: On) => {
      const { clock, jobs } = world(on, { engine: () => new Promise<Answer>(() => {}) })
      const stream = $.turn.step({ ...STEP_INPUT, agentId: 'a1' })
      while (!(await stream.next()).done) continue
      await runStep($)
      await clock.advance(1)
      await runStep($)
      await clock.advance(1)
      expect(jobs.filter((job) => job.kind === 'step').length).toBe(1)
    },
  )

  test(
    'the next tool result waits for the job 7 s at most, and a job ' + 'older than 25 s is dropped for the next step',
    async ($: Driver, on: On) => {
      const { clock, jobs } = world(on, {
        engine: (job) => (job.kind === 'step' ? new Promise<Answer>(() => {}) : { done: {} }),
      })
      await runStep($)
      await clock.advance(1)
      const held = $.classic.PostToolUse(TOOL)
      await clock.advance(6999)
      expect(jobs.filter((job) => job.kind === 'tool').length).toBe(0)
      await clock.advance(1)
      await held
      expect(jobs.filter((job) => job.kind === 'tool')[0].held).toEqual([])
      await clock.advance(20000)
      const dropped = $.classic.PostToolUse(TOOL)
      await clock.advance(7000)
      await dropped
      await runStep($)
      await clock.advance(1)
      expect(jobs.filter((job) => job.kind === 'step').length).toBe(2)
    },
  )
})

describe('the queue (793d22b2)', () => {
  const work = (job: Job): Answer => ({ done: { work: { name: job.kind } } })

  test(
    'a turn is answered before any job runs, then the capture, ' +
      'the housekeeping and the summary follow in order',
    async ($: Driver, on: On) => {
      const { clock, jobs } = world(on, { engine: work })
      await $.classic.SessionStart({ source: 'clear', transcript_path: '/t/s.jsonl' })
      jobs.length = 0
      const result = await $.turn.complete(COMPLETE)
      expect(result.text).toBe(COMPLETE.answer)
      expect(jobs).toEqual([])
      await clock.settle()
      expect(jobs.map((job) => job.kind)).toEqual(['capture', 'housekeeping', 'summary'])
      expectHas(jobs[0], { answer: COMPLETE.answer, sessionId: 'sess-1', turn: 3, messages: MESSAGES })
      expectHas(jobs[0], { transcript: '/t/s.jsonl' })
      expectHas(jobs[1], { turnEnded: true, work: { name: 'capture' } })
    },
  )

  test(
    'the jobs of a burst run one at a time, one housekeeping ' + 'follows the last, and the summary only a turn',
    async ($: Driver, on: On) => {
      const { clock, jobs } = world(on, { engine: work })
      await $.classic.SubagentStop({
        agent_id: 'a1',
        agent_type: 'general-purpose',
        agent_transcript_path: '/t/a1.jsonl',
        last_assistant_message: 'Noted.',
        stop_hook_active: false,
      })
      await $.classic.SubagentStop({
        agent_id: 'a2',
        agent_type: 'general-purpose',
        agent_transcript_path: '/t/a2.jsonl',
        last_assistant_message: 'Done.',
        stop_hook_active: false,
      })
      await clock.settle()
      expect(jobs.map((job) => job.kind)).toEqual(['subagent', 'subagent', 'housekeeping'])
      expectHas(jobs[0], { path: '/t/a1.jsonl', last: 'Noted.', agent: 'a1', sessionId: 'sess-1', turn: 3 })
      expectHas(jobs[1], { agent: 'a2', work: { name: 'subagent' } })
      expectHas(jobs[2], { turnEnded: false })
    },
  )

  test(
    'a job that fails warns, and the next one still runs; a ' + 'subagent turn is not captured at its turn.complete',
    async ($: Driver, on: On) => {
      const { clock, jobs, shown } = world(on, {
        engine: (job) => (job.kind === 'capture' ? { exit: 1, stderr: 'boom' } : { done: {} }),
      })
      await $.turn.complete({ ...COMPLETE, agentId: 'a1' })
      await clock.settle()
      expect(jobs).toEqual([])
      await $.turn.complete(COMPLETE)
      await clock.settle()
      expect(jobs.map((job) => job.kind)).toEqual(['capture', 'housekeeping', 'summary'])
      expect(shown).toContain('Automatic memory was skipped because of an error (the capture)')
    },
  )

  test('a quiet turn shows nothing and clears nothing; a turn after a line clears it', async ($: Driver, on: On) => {
    const lines = [{ show: false }, { show: true, line: 'a line', lineShown: true }, { show: true, lineShown: false }]
    let turn = 0
    const { clock, jobs, shown } = world(on, {
      engine: (job) => (job.kind === 'summary' ? { done: lines[turn++] } : { done: {} }),
    })
    for (let each = 0; each < lines.length; each += 1) {
      await $.turn.complete(COMPLETE)
      await clock.settle()
    }
    expect(shown).toEqual(['a line', undefined])
    expect(jobs.filter((job) => job.kind === 'summary').map((job) => job.lineShown)).toEqual([false, false, true])
  })

  test(
    'the seam file records a job joining the queue before the ' + 'hook returns, and the queue being empty',
    async ($: Driver, on: On) => {
      const { clock, writes } = world(on, { engine: work, env: { ENGRAM_AUTO_SEAM_LOG: '/seam.log' } })
      await $.turn.complete(COMPLETE)
      expect(writes.at(-1)).toEqual({ path: '/seam.log', text: 'queued\n' })
      await clock.settle()
      expect(writes.at(-1)).toEqual({ path: '/seam.log', text: 'queued\nidle\n' })
    },
  )
})
