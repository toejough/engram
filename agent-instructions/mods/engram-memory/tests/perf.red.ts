// The performance budgets the mod exceeds (793e8o), each a contract test in the time of the mock clock or, for the
// redactor, in real time. Each fails today for the reason in its first comment; when the mod meets all of them this
// file is renamed to perf.test.ts. The baselines are those of perf.test.ts: haiku at `low` effort, 793b351; the
// commands measured on this machine. Stubbed, not live: the latencies are stubs; the redactor is the real code on real
// text.
import { describe, expect, test } from 'claude-code/testing'
import type { Driver, On, Reply } from 'claude-code/testing'

import { redact } from '../hooks/capture.js'
import { COMPLETE, PROMPT, STEP_MESSAGES, TOOL, advanceUntil, drive, noteOutput, runStep, world } from './world.js'

const QUERY_COLD_MS = 590
const RELEVANCE_P95_MS = 3440
const CLASSIFY_P95_MS = 2980
const HOLD_CEILING_MS = 8000
const TOOL_HOLD_BUDGET_MS = 7100
const NEVER_MS = 600_000

describe('the 95th percentile of the judgement is not cut', () => {
  // The relevance judgement is cut at 3 s (judge.js) and its 95th percentile is 3.44 s: one lookup in twenty that found
  // candidates is cut, delivers nothing, and warns the user that the System timed out.
  test('793d22b3 a prompt at the 95th percentile still gets its note within 8 s', async ($: Driver, on: On) => {
    const { clock } = world(on, {
      latency: { query: QUERY_COLD_MS, relevance: RELEVANCE_P95_MS },
      query: noteOutput('2.perf-red-p95.md'),
    })
    const { value, elapsedMs } = await drive<Reply>(clock, $.prompt.submit(PROMPT))
    expect(value.context?.length ?? 0, 'the note was not delivered').toBe(1)
    expect(elapsedMs <= HOLD_CEILING_MS, `held ${elapsedMs} ms, ceiling ${HOLD_CEILING_MS} ms`).toBe(true)
  })

  test('793d23b2 the in-turn cue at its 95th percentile still delivers within 7 s', async ($: Driver, on: On) => {
    const { clock, shown } = world(on, {
      latency: { classify: CLASSIFY_P95_MS, query: QUERY_COLD_MS, relevance: RELEVANCE_P95_MS },
      query: noteOutput('6.perf-red-cue.md'),
      messages: STEP_MESSAGES,
    })
    await runStep($)
    await clock.advance(1000)
    const { value, elapsedMs } = await drive<Reply>(clock, $.classic.PostToolUse(TOOL))
    await $.turn.complete(COMPLETE)
    await clock.advance(NEVER_MS)
    expect(
      shown.map((line) => line.text).filter((text) => /timed out/.test(text ?? '')),
      'the user was warned',
    ).toEqual([])
    expect(value.additionalContext?.length ?? 0, 'the note was not delivered').toBe(1)
    expect(elapsedMs <= TOOL_HOLD_BUDGET_MS, `held ${elapsedMs} ms, budget ${TOOL_HOLD_BUDGET_MS} ms`).toBe(true)
  })
})

describe('the capture of a subagent starts when its transcript is complete', () => {
  const FILE = '/home/.claude/projects/p/s/subagents/a1.jsonl'
  const LAG_MS = 100
  const POLL_ALLOWANCE_MS = 100
  const user = { type: 'user', message: { content: 'I ran go test. No, run targ test, not go test. Acknowledge.' } }
  const agent = { type: 'assistant', message: { content: [{ type: 'text', text: 'Noted: targ test.' }] } }
  const rows = (...each: object[]) => each.map((row) => JSON.stringify(row)).join('\n')

  // The transcript of a subagent lags its stop event (SP5); the mod polls it every 500 ms, so a file that completes
  // 100 ms after the stop is read 500 ms after it. The lag here is an assumption, not a measurement.
  test('793d27b1 a transcript complete 100 ms after the stop is captured at once', async ($: Driver, on: On) => {
    let stopAt = 0
    const { clock, calls } = world(on, {
      files: { [FILE]: (atMs: number) => (atMs - stopAt >= LAG_MS ? rows(user, agent) : rows(user)) },
    })
    stopAt = await clock.now()
    await $.classic.SubagentStop({
      agent_id: 'a1',
      agent_type: 'general-purpose',
      agent_transcript_path: FILE,
      last_assistant_message: 'Noted: targ test.',
      stop_hook_active: false,
    })
    const waited = await advanceUntil(clock, () => calls.some((call) => call.what === 'query'), 10)
    expect(
      waited <= LAG_MS + POLL_ALLOWANCE_MS,
      `the capture began ${waited} ms after the stop, budget ${LAG_MS + POLL_ALLOWANCE_MS} ms`,
    ).toBe(true)
    await clock.advance(NEVER_MS)
  })
})

describe('redaction costs time in proportion to the text', () => {
  // A message is redacted whole before it is cut (capture.js render), and the host runs the mod on one thread: a long
  // unbroken run of letters and a block of wrapped lines each make the redactor take time that grows with the square of
  // the text, so one pasted blob of a few tens of kilobytes holds every hook of the session for seconds.
  const BUDGET_MS = 200

  const cost = (text: string) => {
    const start = Date.now()
    redact(text)
    return Date.now() - start
  }

  test('793d26b1 a 12,000-character unbroken run is redacted within 200 ms', () => {
    const took = cost('a'.repeat(12_000))
    expect(took <= BUDGET_MS, `took ${took} ms, budget ${BUDGET_MS} ms`).toBe(true)
  })

  test('793d26b1 a 15,000-character block of wrapped lines is redacted within 200 ms', () => {
    const took = cost('abcdefghijklmnopqrstuvwxyz0123\n'.repeat(500))
    expect(took <= BUDGET_MS, `took ${took} ms, budget ${BUDGET_MS} ms`).toBe(true)
  })
})
