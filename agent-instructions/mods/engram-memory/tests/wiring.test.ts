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
        step: 'prompt lookup',
      })
      expectHas(jobs[1], { kind: 'lookup', state: ['s'], reply: REPLY })
      expect(result.context).toEqual([BLOCK])
    },
  )

  test('a wait is slept on the clock before the next round', async ($: Driver, on: On) => {
    let waited = false
    const { clock, jobs } = world(on, {
      engine: (job) => {
        if (job.kind !== 'lookup' || waited) return { done: {} }
        waited = true
        return { waitMs: 500, state: [] }
      },
    })
    const submitted = $.prompt.submit(PROMPT)
    await clock.advance(499)
    expect(jobs.filter((job) => job.kind === 'lookup').length).toBe(1)
    await clock.advance(1)
    await submitted
    expect(jobs.filter((job) => job.kind === 'lookup').length).toBe(2)
  })

  test(
    'the effects of a round are applied: debug lines, warnings ' + 'as transcript lines, counts in the summary',
    async ($: Driver, on: On) => {
      const engine = (job: Job): Answer => {
        if (job.kind === 'summary') return { done: { show: true, line: 'the line' } }
        if (job.kind !== 'lookup') return { done: {} }
        return { log: ['engram-memory: a line'], warnings: ['a warning'], counts: { delivered: 2 }, done: {} }
      }
      const { clock, jobs, logs, shown, pinned } = world(on, { engine })
      await $.prompt.submit(PROMPT)
      expect(logs).toEqual(['engram-memory: a line'])
      expect(shown).toEqual(['a warning'])
      await $.turn.complete(COMPLETE)
      await clock.settle()
      const summary = jobs.find((job) => job.kind === 'summary') ?? { kind: '' }
      expectHas(summary, { tally: [{ delivered: 2 }] })
      // the warning was its own line; the summary job is not handed it to repeat (793b442)
      expect('warnings' in summary).toBe(false)
      expect(shown.at(-1)).toBe('the line')
      expect(shown).toEqual(['a warning', 'the line'])
      // no status line and no toast: the lines are transcript lines the model is not sent (793b442)
      expect(pinned).toEqual([])
    },
  )

  // the engine words and logs a failure it is handed (793d36b4); the mod relays its warning
  const ENGINE_WARNING = 'Automatic memory skipped: prompt lookup failed: x; no notes delivered for this prompt (d)'
  const FAILURES: [string, Answer, string][] = [
    ['the program exits with an error', { exit: 70, stderr: 'injected failure' }, 'exited 70: injected failure'],
    ['the program cannot start', { reject: '$.process.run(engram) failed to start: ENOENT' }, 'failed to start'],
    [
      'the program is cut at its deadline',
      { reject: '$.process.run(engram) aborted: still running after 600000ms' },
      'still running after 600000ms',
    ],
  ]
  for (const [name, answer, cue] of FAILURES) {
    test(`793d22b3 when ${name} the prompt goes on as typed and the user is told why`, async ($: Driver, on: On) => {
      const { jobs, shown } = world(on, {
        engine: (job) => (job.kind === 'failure' ? { warnings: [ENGINE_WARNING], done: {} } : answer),
      })
      const result = await $.prompt.submit(PROMPT)
      expect(result.text).toBe(PROMPT.text)
      expect(result.context ?? []).toEqual([])
      expect(shown).toEqual([ENGINE_WARNING])
      const handed = jobs.find((job) => job.kind === 'failure') ?? { kind: '' }
      expectHas(handed, { step: 'prompt lookup', sessionId: 'sess-1' })
      expect(String(handed.text)).toContain(cue)
    })
  }

  test(
    '793d22b3 a failure of the program is handed to the engine, which words and logs it: the housekeeping is ' +
      'named in the terms of the user',
    async ($: Driver, on: On) => {
      const { clock, jobs, shown } = world(on, {
        engine: (job) => {
          if (job.kind === 'housekeeping') return { exit: 70, stderr: 'boom' }
          return job.kind === 'failure' ? { warnings: ['the engine words it'], done: {} } : { done: {} }
        },
      })
      await $.turn.complete(COMPLETE)
      await clock.settle()
      const handed = jobs.filter((job) => job.kind === 'failure')
      expect(handed.map((job) => job.step)).toEqual(['indexing', 'curation', 'backlog capture'])
      expectHas(handed[0], { sessionId: 'sess-1' })
      expect(String(handed[0].text)).toContain('exited 70: boom')
      expect(handed.map((job) => String(job.step)).join(' ')).not.toContain('housekeeping')
      expect(shown).toContain('the engine words it')
    },
  )

  // the process resolves no more than 4 MiB of output and cuts the rest without an error (SP18): a reply of tens of
  // megabytes is a document cut short, and the failure says how large it was and how it began and ended (793b469)
  test(
    '793d22b3 a reply that is not JSON is handed to the engine as it came, to say how large it was and how it ended',
    async ($: Driver, on: On) => {
      const cut = `{"log":["engram-memory: x"],"state":[{"out":"${'m'.repeat(5000)}`
      const { jobs, shown } = world(on, {
        engine: (job) => (job.kind === 'failure' ? { warnings: ['the engine words it'], done: {} } : { raw: cut }),
      })
      const result = await $.prompt.submit(PROMPT)
      expect(result.text).toBe(PROMPT.text)
      const handed = jobs.find((job) => job.kind === 'failure') ?? { kind: '' }
      expectHas(handed, { step: 'prompt lookup', unreadable: cut })
      expect(String(handed.text)).not.toBe('')
      expect(shown).toEqual(['the engine words it'])
    },
  )

  test('793d36b3 a part of the housekeeping whose reply cannot be read is blamed by name, and the others run', async (
    $: Driver,
    on: On,
  ) => {
    const { clock, jobs, shown } = world(on, {
      engine: (job) => {
        if (job.kind === 'failure') return { warnings: [`${job.step} failed`], done: {} }
        return job.kind === 'housekeeping' && job.part === 'curation' ? { raw: '{"log":["cut' } : { done: {} }
      },
    })
    await $.turn.complete(COMPLETE)
    await clock.settle()
    const parts = jobs.filter((job) => job.kind === 'housekeeping')
    expect(parts.map((job) => job.part)).toEqual(['indexing', 'curation', 'backlog capture'])
    expect(jobs.filter((job) => job.kind === 'failure').map((job) => job.step)).toEqual(['curation'])
    expect(shown).toEqual(['curation failed'])
    expect(jobs.at(-1)?.kind).toBe('summary')
  })

  test('793d36b3 a burst that is not a turn runs no backlog part', async ($: Driver, on: On) => {
    const { clock, jobs } = world(on)
    await $.classic.SubagentStop({
      agent_id: 'a1',
      agent_type: 'general-purpose',
      agent_transcript_path: '/t/a1.jsonl',
      last_assistant_message: 'Noted.',
      stop_hook_active: false,
    })
    await clock.settle()
    expect(jobs.filter((job) => job.kind === 'housekeeping').map((job) => job.part)).toEqual(['indexing', 'curation'])
  })

  const FALLBACKS: [string, Answer, string][] = [
    ['exits with an error', { exit: 70, stderr: 'injected failure' }, 'prompt lookup failed'],
    [
      'cannot start',
      { reject: '$.process.run(engram) failed to start: ENOENT' },
      'prompt lookup could not reach the vault',
    ],
    ['is cut', { reject: '$.process.run(engram) aborted: still running after 600000ms' }, 'prompt lookup timed out'],
  ]
  for (const [name, answer, what] of FALLBACKS) {
    test(`793d22b3 when the engine cannot be run for the failure either (it ${name}), the mod words it`, async (
      $: Driver,
      on: On,
    ) => {
      const { shown, pinned } = world(on, { engine: () => answer })
      const result = await $.prompt.submit(PROMPT)
      expect(result.text).toBe(PROMPT.text)
      expect(shown).toEqual([
        `Automatic memory skipped: ${what}; nothing was done this time (details: the debug log)`,
      ])
      expect(pinned).toEqual([])
    })
  }
})

describe('what the mod keeps and logs of a response (793b468, 793b471)', () => {
  test('793d36b1 a response that holds no session state leaves the mod\'s own in place', async ($: Driver, on: On) => {
    const { clock, jobs } = world(on, {
      engine: (job, call) => {
        if (job.kind === 'lookup') return call === 1 ? { done: { ledger: { main: ['n.md'] } } } : { done: {} }
        return { done: job.kind === 'capture' ? { work: { name: 'kept' } } : {} }
      },
    })
    await $.prompt.submit(PROMPT)
    await $.prompt.submit(PROMPT)
    await $.prompt.submit(PROMPT)
    expect(jobs.filter((job) => job.kind === 'lookup').map((job) => job.ledger)).toEqual([
      undefined,
      { main: ['n.md'] },
      { main: ['n.md'] },
    ])
    await $.turn.complete(COMPLETE)
    await $.turn.complete(COMPLETE)
    await clock.settle()
    const captures = jobs.filter((job) => job.kind === 'capture')
    expect(captures.length).toBe(2)
    expect(jobs.filter((job) => job.kind === 'housekeeping').at(-1)?.work).toEqual({ name: 'kept' })
  })

  test('793d36b1 a truncated response is written to the debug log', async ($: Driver, on: On) => {
    const { logs } = world(on, { engine: () => ({ truncated: true, done: {} }) })
    await $.prompt.submit(PROMPT)
    expect(logs.join('\n')).toContain('the reply was truncated')
  })

  test('793d36b4 the debug log of a failed step holds no text of what the program printed', async (
    $: Driver,
    on: On,
  ) => {
    const stderr = 'ghp_a1B2a1B2a1B2a1B2a1B2a1B2a1B2a1B2a1B2'
    const { logs } = world(on, {
      engine: (job) => (job.kind === 'failure' ? { done: {} } : { exit: 70, stderr }),
    })
    await $.prompt.submit(PROMPT)
    expect(logs.join('\n')).toContain('prompt lookup failed')
    expect(logs.join('\n')).not.toContain('ghp_')
  })
})

describe('the System\'s lines reach the agent (793b470)', () => {
  const LINE = 'Memory activity: 2 notes added'
  const WARNING = 'Automatic memory skipped: curation failed: boom; nothing lost, retried next turn (details: ~/a.log)'

  test(
    '793d30b3 the lines shown since the last prompt go with the next prompt\'s lookup, once, and the user still sees ' +
      'them',
    async ($: Driver, on: On) => {
      const { clock, jobs, shown } = world(on, {
        engine: (job) => {
          if (job.kind === 'summary') return { done: { show: true, line: LINE } }
          const curation = job.kind === 'housekeeping' && job.part === 'curation'
          return curation ? { warnings: [WARNING], done: {} } : { done: {} }
        },
      })
      await $.prompt.submit(PROMPT)
      await $.turn.complete(COMPLETE)
      await clock.settle()
      expect(shown).toEqual([WARNING, LINE])
      await $.prompt.submit(PROMPT)
      await $.prompt.submit(PROMPT)
      const lookups = jobs.filter((job) => job.kind === 'lookup')
      expect(lookups.map((job) => job.activity)).toEqual([[], [WARNING, LINE], []])
    },
  )
})

describe('the hold of an event (793d22b3)', () => {
  const LOOKUP_SILENT = (job: Job, call: number): Promise<Answer> | Answer => {
    if (job.kind === 'failure') return { warnings: ['the engine words it'], done: {} }
    return job.kind === 'lookup' && call === 1 ? new Promise<Answer>(() => {}) : { done: { block: BLOCK } }
  }

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
      expect(shown).toEqual(['the engine words it'])
      expect(jobs.find((job) => job.kind === 'failure')?.text).toContain('still running after 8000 ms')
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
      const { clock, shown } = world(on, {
        engine: (job) => {
          if (job.kind === 'failure') return { warnings: ['the engine words it'], done: {} }
          return job.kind === 'lookup' ? slow() : { done: {} }
        },
      })
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
      expect(jobs.filter((job) => job.kind !== 'deep').map((job) => job.kind)).toEqual(['lookup', 'compact', 'lookup'])
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
    expect(jobs.filter((job) => job.kind === 'lookup').map((job) => job.ledger)).toEqual([undefined, { main: ['one'] }])
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
      expectHas(jobs[0], { kind: 'step', step: 'mid-turn lookup', toolUses: STEP_RESULT.toolUses })
      const tool = await $.classic.PostToolUse(TOOL)
      expect(tool.additionalContext).toEqual([BLOCK])
      expectHas(jobs[1], { kind: 'tool', route: false, held: [NOTE] })
    },
  )

  test(
    'a tool result that arrives while the step is still being started waits for its cue, as a later one does',
    async ($: Driver, on: On) => {
      let release = () => {}
      const messagesGate = new Promise<void>((resolve) => {
        release = resolve
      })
      const { clock, jobs } = world(on, { engine: cue, messagesGate })
      const stepping = runStep($)
      await clock.advance(1)
      const tool = $.classic.PostToolUse(TOOL)
      await clock.advance(1)
      release()
      await stepping
      await clock.settle()
      expect((await tool).additionalContext).toEqual([BLOCK])
      expectHas(jobs.filter((job) => job.kind === 'tool')[0], { held: [NOTE] })
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

describe('the lookup with the chunk index (793d24b1)', () => {
  const STUB = { path: '/s/chunk.md#Heading', kind: 'chunk', situation: '', body: '', score: 0.6 }
  const deep = (job: Job): Answer => (job.kind === 'deep' ? { done: { notes: [STUB] } } : { done: { block: BLOCK } })

  test(
    'a prompt is followed by a lookup with the chunk index in the background, and its stubs ride a later tool result',
    async ($: Driver, on: On) => {
      const { clock, jobs } = world(on, { engine: deep })
      const prompted = await $.prompt.submit(PROMPT)
      expect(prompted.context).toEqual([BLOCK])
      await clock.advance(1)
      expect(jobs.map((job) => job.kind)).toEqual(['lookup', 'deep'])
      expectHas(jobs[1], { step: 'history lookup', text: PROMPT.text })
      const tool = await $.classic.PostToolUse(TOOL)
      expect(tool.additionalContext).toEqual([BLOCK])
      expectHas(jobs[2], { kind: 'tool', route: false, held: [STUB] })
      await $.classic.PostToolUse(TOOL)
      expect(jobs[3].held).toEqual([])
    },
  )

  test(
    'no tool result waits for it, and one that is still running is kept for a later result until it is too old',
    async ($: Driver, on: On) => {
      const { clock, jobs } = world(on, {
        engine: (job) => (job.kind === 'deep' ? new Promise<Answer>(() => {}) : { done: {} }),
      })
      await $.prompt.submit(PROMPT)
      await clock.advance(1)
      await $.classic.PostToolUse(TOOL)
            expect(jobs.filter((job) => job.kind === 'tool').map((job) => job.held)).toEqual([[]])
      await clock.advance(26000)
      await $.classic.PostToolUse(TOOL)
      expect(jobs.filter((job) => job.kind === 'tool').length).toBe(2)
    },
  )

  test('it is dropped with the turn it belongs to', async ($: Driver, on: On) => {
    const { clock, jobs } = world(on, { engine: deep })
    await $.prompt.submit(PROMPT)
    await clock.advance(1)
    await $.turn.complete(COMPLETE)
    await clock.settle()
    await $.classic.PostToolUse(TOOL)
    expect(jobs.filter((job) => job.kind === 'tool')[0].held).toEqual([])
  })
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
      const kinds = ['capture', 'housekeeping', 'housekeeping', 'housekeeping', 'summary']
      expect(jobs.map((job) => job.kind)).toEqual(kinds)
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
      expect(jobs.map((job) => job.kind)).toEqual(['subagent', 'subagent', 'housekeeping', 'housekeeping'])
      expectHas(jobs[0], { path: '/t/a1.jsonl', last: 'Noted.', agent: 'a1', sessionId: 'sess-1', turn: 3 })
      expectHas(jobs[1], { agent: 'a2', work: { name: 'subagent' } })
      expectHas(jobs[2], { turnEnded: false })
    },
  )

  test(
    'a job that fails warns, and the next one still runs; a ' + 'subagent turn is not captured at its turn.complete',
    async ($: Driver, on: On) => {
      const { clock, jobs, shown } = world(on, {
        engine: (job) => {
          if (job.kind === 'capture') return { exit: 1, stderr: 'boom' }
          return job.kind === 'failure' ? { warnings: ['the engine words it'], done: {} } : { done: {} }
        },
      })
      await $.turn.complete({ ...COMPLETE, agentId: 'a1' })
      await clock.settle()
      expect(jobs).toEqual([])
      await $.turn.complete(COMPLETE)
      await clock.settle()
      expect(jobs.map((job) => job.kind)).toEqual([
        'capture',
        'failure',
        'housekeeping',
        'housekeeping',
        'housekeeping',
        'summary',
      ])
      expectHas(jobs[1], { step: 'capture' })
      expect(shown).toContain('the engine words it')
    },
  )

  test('a quiet turn shows nothing, and a turn after a line writes nothing', async ($: Driver, on: On) => {
    const lines = [{ show: false }, { show: true, line: 'a line' }, { show: false }]
    let turn = 0
    const { clock, jobs, shown, pinned } = world(on, {
      engine: (job) => (job.kind === 'summary' ? { done: lines[turn++] } : { done: {} }),
    })
    for (let each = 0; each < lines.length; each += 1) {
      await $.turn.complete(COMPLETE)
      await clock.settle()
    }
    // a transcript line stays where it was written: a later turn has nothing to clear
    expect(shown).toEqual(['a line'])
    expect(pinned).toEqual([])
    expect(jobs.filter((job) => job.kind === 'summary').length).toBe(lines.length)
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
