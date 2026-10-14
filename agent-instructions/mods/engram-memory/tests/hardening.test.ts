// The persisted queue of turn captures (793e8p, 793b203, 793b398): queued work survives the end of the session. The
// engine owns the file `capture-queue.json` (its format, the tries, the removal when a capture ends: Go tests in
// internal/auto, queue_test.go); the mod only hands it the jobs: `enqueue` before a turn's hook returns, `resume` when
// a session starts, and a `capture` of the entry it was given. The other gap of 793d35c, a subagent's compaction that
// empties the ledger of the main agent, is a Go test with the ledger (compaction_test.go) and a wiring test.
import { describe, expect, test } from 'claude-code/testing'
import type { Driver, On } from 'claude-code/testing'

import { COMPLETE, MESSAGES, world } from './world.js'
import type { Answer, Job } from './world.js'

const LEFT_BEHIND = {
  window: 'Agent: I will run go test ./...\nUser: No, use targ, not go test directly.\nAgent: Understood.',
  source: 'captured from the conversation; written automatically (session left-behind, turn 9)',
  transcript: '/home/.claude/projects/p/old.jsonl',
  tries: 1,
}

describe('queued work survives the end of the session (793b203)', () => {
  // Before the queue a capture was held in memory from the moment the turn ended until its job ran; a session that
  // ended in between (the user exits, the process is killed) lost the lesson of its last turn.
  test('793d22b4 a turn is in the queue file before its hook returns', async ($: Driver, on: On) => {
    const order: string[] = []
    const engine = (job: Job): Answer => {
      order.push(job.kind)
      return job.kind === 'enqueue' ? { done: { queued: LEFT_BEHIND } } : { done: {} }
    }
    const { clock } = world(on, { engine })
    await $.turn.complete(COMPLETE)
    order.push('returned')
    expect(order).toEqual(['enqueue', 'returned'])
    await clock.settle()
    expect(order).toEqual(['enqueue', 'returned', 'capture', 'housekeeping', 'summary'])
  })

  test('793d22b4 an entry a session left behind is captured at the next start', async ($: Driver, on: On) => {
    const engine = (job: Job): Answer => (job.kind === 'resume' ? { done: { resumed: [LEFT_BEHIND] } } : { done: {} })
    const { clock, jobs } = world(on, { engine })
    await $.classic.SessionStart({ source: 'startup', transcript_path: '/home/.claude/projects/p/next.jsonl' })
    await clock.settle()
    expect(jobs.map((job) => job.kind)).toEqual(['resume', 'capture', 'housekeeping'])
    expect(jobs[0].start).toBe('startup')
    expect(jobs[1].entry).toEqual(LEFT_BEHIND)
    expect(jobs[1].messages).toBeUndefined()
  })

  test(
    '793d22b4 a start that is not a new session is still handed to the engine, which decides',
    async ($: Driver, on: On) => {
      const { clock, jobs } = world(on)
      await $.classic.SessionStart({ source: 'clear', transcript_path: '/home/.claude/projects/p/next.jsonl' })
      await clock.settle()
      expect(jobs.map((job) => [job.kind, job.start])).toEqual([['resume', 'clear']])
    },
  )

  test(
    '793d22b4 a turn the file could not take is captured from memory and nobody is warned',
    async ($: Driver, on: On) => {
      const engine = (job: Job): Answer => (job.kind === 'enqueue' ? { exit: 1, stderr: 'disk full' } : { done: {} })
      const { clock, jobs, shown } = world(on, { engine })
      await $.turn.complete(COMPLETE)
      await clock.settle()
      expect(jobs.map((job) => job.kind)).toEqual(['enqueue', 'capture', 'housekeeping', 'summary'])
      expect(jobs[1].entry).toBeUndefined()
      expect(jobs[1].messages).toEqual(MESSAGES)
      expect(shown).toEqual([])
    },
  )
})
