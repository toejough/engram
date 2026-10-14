// The hardening gate the mod does not meet yet (793e8p), from the gaps of 793d35c: queued work is not persisted.
// Each test fails today for the reason in its comment; when the mod meets all of them
// this file is renamed to hardening.test.ts.
// The other gap, a subagent's compaction that empties the top-level ledger and not its own, moved to Go with the ledger
// (internal/auto, refactorred_test.go).
//
// The contract of the persisted queue is chosen here, by the test (793b362): the file is `capture-queue.json` in the
// chunks directory beside the backlog marks, holding a JSON array of { window, source } for every turn capture that has
// been queued and has not finished; it is written before the turn's hook returns, and rewritten without a job when the
// job ends.
import { describe, expect, test } from 'claude-code/testing'
import type { Driver, On } from 'claude-code/testing'

import { COMPLETE, world } from './world.js'

const QUEUE_FILE = '/home/t/.local/share/engram/chunks/capture-queue.json'
const USER_LINE = 'No, use targ, not go test directly.'
const LEFT_BEHIND = {
  window: `Agent: I will run go test ./...\nUser: ${USER_LINE}\nAgent: Understood, I will use targ.`,
  source: 'captured from the conversation; written automatically (session left-behind, turn 9)',
}

describe('queued work survives the end of the session (793b203)', () => {
  // Today a capture is held in memory from the moment the turn ends until its job runs; a session that ends in between
  // (the user exits, the process is killed) loses the lesson of its last turn.
  test('793d22b2 a capture is queued in the file before the turn returns', async ($: Driver, on: On) => {
    const { clock, writes } = world(on)
    await $.turn.complete(COMPLETE)
    const queued = writes.filter((write) => write.path === QUEUE_FILE)
    expect(queued.length, 'nothing was written to the queue file before the turn returned').toBeGreaterThan(0)
    expect(queued[0].text).toContain(USER_LINE)
    await clock.settle()
    expect(
      writes
        .filter((write) => write.path === QUEUE_FILE)
        .at(-1)
        ?.text.trim(),
    ).toBe('[]')
  })

  test('793d22b2 a capture left by an ended session is captured at the next start', async ($: Driver, on: On) => {
    const { clock, jobs, writes } = world(on, { files: { [QUEUE_FILE]: JSON.stringify([LEFT_BEHIND]) } })
    await $.classic.SessionStart({ source: 'startup', transcript_path: '/home/.claude/projects/p/next.jsonl' })
    await clock.settle()
    const capture = jobs.find((job) => JSON.stringify(job).includes(USER_LINE))
    expect(capture !== undefined, 'the left-behind capture was not handed to the engine').toBe(true)
    expect(
      writes
        .filter((write) => write.path === QUEUE_FILE)
        .at(-1)
        ?.text.trim(),
    ).toBe('[]')
  })

  test('793d22b2 a capture that fails stays in the file for the next session', async ($: Driver, on: On) => {
    const { clock, writes } = world(on, {
      files: { [QUEUE_FILE]: JSON.stringify([LEFT_BEHIND]) },
      engine: () => ({ exit: 1, stderr: 'this is not a lesson document' }),
    })
    await $.classic.SessionStart({ source: 'startup', transcript_path: '/home/.claude/projects/p/again.jsonl' })
    await clock.settle()
    const last = writes.filter((write) => write.path === QUEUE_FILE).at(-1)
    expect(last?.text, 'the failed capture was dropped from the queue file').toContain(USER_LINE)
  })
})
