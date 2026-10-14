// Performance budgets that the mod meets and must keep meeting (793e8o): the hold on a prompt, the hold on a tool
// result and the time from the end of a turn to its summary, in the time of the mock clock with the latencies of the
// measurements below. They are the guard half of the performance gate; the budgets the mod exceeds are in perf.red.ts.
//
// Baselines. Live, measured on this machine: `engram query` 0.17 s with a warm embedder cache and 0.59 s with an empty
// one, `engram ingest --auto` 0.06 s, the scan for pending offers 0.01 s, `engram learn` 0.33 s (SP11, 793e2e). Model
// calls: haiku at `low` effort, 793b351 (the effort-floor control of H2): median and 95th percentile of classify 1.67
// and 2.98 s, relevance 2.39 and 3.44 s, capture 2.40 and 9.12 s. Stubbed, not live: every call below is a stub that
// waits that long on the mock clock; what is measured is the sequencing the mod adds around the calls, not the calls.
import { describe, expect, test } from 'claude-code/testing'
import type { Driver, On, Reply } from 'claude-code/testing'

import { COMPLETE, PROMPT, STEP_MESSAGES, TOOL, advanceUntil, drive, noteOutput, runStep, world } from './world.js'

const QUERY_WARM_MS = 170
const RELEVANCE_P50_MS = 2390
const CLASSIFY_P50_MS = 1670
const CAPTURE_P50_MS = 2400
const LEARN_MS = 330
const INGEST_MS = 60
const SCAN_MS = 10
const NEVER_MS = 600_000

// budgets: the sum of the baselines on the path, with one step of the mock clock (50 ms) of resolution
const PROMPT_HOLD_BUDGET_MS = QUERY_WARM_MS + RELEVANCE_P50_MS + 80
const SUMMARY_BUDGET_MS = QUERY_WARM_MS + CAPTURE_P50_MS + LEARN_MS + INGEST_MS + SCAN_MS + 80
// 793d22b3: the lookup of a moment (5 s) and its relevance judgement (3 s) hold the event 8 s at most
const HOLD_CEILING_MS = 8000
// the hold of a tool result for the in-turn cue is 7 s (the platform cuts a hook at about 10 s)
const TOOL_HOLD_BUDGET_MS = 7000 + 100

describe('the hold on a prompt', () => {
  test('793d22b3 a prompt at the median is held for the lookup and the judgement only', async ($: Driver, on: On) => {
    const { clock } = world(on, {
      latency: { query: QUERY_WARM_MS, relevance: RELEVANCE_P50_MS },
      query: noteOutput('2.perf-p50.md'),
    })
    const { value, elapsedMs } = await drive<Reply>(clock, $.prompt.submit(PROMPT))
    expect(value.context.length).toBe(1)
    expect(elapsedMs <= PROMPT_HOLD_BUDGET_MS, `held ${elapsedMs} ms, budget ${PROMPT_HOLD_BUDGET_MS} ms`).toBe(true)
  })

  test('793d22b3 a lookup that never answers holds the prompt for its deadline only', async ($: Driver, on: On) => {
    const { clock, calls } = world(on, { latency: { query: NEVER_MS }, query: noteOutput('3.perf-hang.md') })
    const { value, elapsedMs } = await drive<Reply>(clock, $.prompt.submit(PROMPT))
    expect(value.text).toBe(PROMPT.text)
    expect(calls.some((call) => call.what.startsWith('model'))).toBe(false)
    expect(elapsedMs <= HOLD_CEILING_MS, `held ${elapsedMs} ms, ceiling ${HOLD_CEILING_MS} ms`).toBe(true)
  })

  test('793d22b3 a judgement that never answers keeps the hold under 8 s', async ($: Driver, on: On) => {
    const { clock } = world(on, { latency: { query: 4900, relevance: NEVER_MS }, query: noteOutput('4.perf-slow.md') })
    const { value, elapsedMs } = await drive<Reply>(clock, $.prompt.submit(PROMPT))
    expect(value.context ?? []).toEqual([])
    expect(elapsedMs <= HOLD_CEILING_MS, `held ${elapsedMs} ms, ceiling ${HOLD_CEILING_MS} ms`).toBe(true)
  })
})

describe('the hold on a tool result', () => {
  test('793d23b2 a tool result waits for the in-turn cue for 7 s at most', async ($: Driver, on: On) => {
    const { clock } = world(on, { latency: { classify: NEVER_MS }, messages: STEP_MESSAGES, ignoreDeadlines: true })
    await runStep($)
    await clock.advance(1)
    const { value, elapsedMs } = await drive<Reply>(clock, $.classic.PostToolUse(TOOL))
    expect(value.additionalContext ?? []).toEqual([])
    expect(elapsedMs <= TOOL_HOLD_BUDGET_MS, `held ${elapsedMs} ms, budget ${TOOL_HOLD_BUDGET_MS} ms`).toBe(true)
    // the job that outlived the hold is still dropped with its turn
    await $.turn.complete(COMPLETE)
    await clock.advance(NEVER_MS)
  })

  test('793d23b2 the cue at its median is held for what a 1 s tool leaves of it', async ($: Driver, on: On) => {
    const jobMs = CLASSIFY_P50_MS + QUERY_WARM_MS + RELEVANCE_P50_MS
    const { clock } = world(on, {
      latency: { classify: CLASSIFY_P50_MS, query: QUERY_WARM_MS, relevance: RELEVANCE_P50_MS },
      query: noteOutput('5.perf-cue.md'),
      messages: STEP_MESSAGES,
    })
    await runStep($)
    // the tool runs for a second while the cue works, then its result is held for what the cue still needs
    await clock.advance(1000)
    const { value, elapsedMs } = await drive<Reply>(clock, $.classic.PostToolUse(TOOL))
    expect(value.additionalContext?.length).toBe(1)
    expect(elapsedMs <= jobMs - 1000 + 100, `held ${elapsedMs} ms, budget ${jobMs - 1000 + 100} ms`).toBe(true)
    await $.turn.complete(COMPLETE)
    await clock.advance(NEVER_MS)
  })
})

describe('the time from the end of a turn to its summary', () => {
  test('793d22b2 a median turn is summarised after the capture, the index and the scan', async ($: Driver, on: On) => {
    const { clock, shown } = world(on, {
      latency: { query: QUERY_WARM_MS, capture: CAPTURE_P50_MS, learn: LEARN_MS, ingest: INGEST_MS, grep: SCAN_MS },
    })
    await $.turn.complete(COMPLETE)
    const elapsedMs = await advanceUntil(clock, () => shown.some((line) => line.text?.startsWith('Memory activity')))
    expect(shown.at(-1)?.text).toBe('Memory activity: 1 note added')
    expect(elapsedMs <= SUMMARY_BUDGET_MS, `summarised after ${elapsedMs} ms, budget ${SUMMARY_BUDGET_MS} ms`).toBe(
      true,
    )
  })
})
