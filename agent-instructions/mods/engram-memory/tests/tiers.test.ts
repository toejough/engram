// Contract tests of the tier table (793d18c): every request names the model of its kind, and raising one kind changes only
// that kind (793e8j). The request builders are pure, so no engine stub is needed.
import { describe, expect, test } from 'claude-code/testing'

import { JUDGE_TIERS, captureRequest, classifyRequest, curationRequest, relevanceRequest } from '../hooks/judge.js'

const note = { path: '1.2026-10-12.note-n.md', kind: 'fact', situation: 'renaming a Go package', body: 'run go build first' }
const builders: Record<string, () => { model: string; effort: string }> = {
  classify: () => classifyRequest('User: is it done?'),
  relevance: () => relevanceRequest('declaring work done', 'I just finished', [note]),
  capture: () => captureRequest('User: no, use the formatter', [note]),
  curation: () => curationRequest({ situation: 'renaming a package', body: 'run go build first' }, [note]),
}
const KINDS = Object.keys(builders)

describe('the tier table', () => {
  test('every request carries the model the table gives for its kind, and none the session model', () => {
    for (const kind of KINDS) {
      const request = builders[kind]()
      expect(request.model).toBe(JUDGE_TIERS[kind])
      expect(request.model).toBe('haiku')
      expect(request.model).not.toBe('sonnet')
    }
  })

  test('every request runs at the lowest accepted effort (793c79)', () => {
    for (const kind of KINDS) {
      expect(builders[kind]().effort).toBe('low')
    }
  })

  test('a kind without a row is an error, never a default model', () => {
    const saved = JUDGE_TIERS.capture
    delete JUDGE_TIERS.capture
    try {
      expect(() => builders.capture()).toThrow('no judgement tier for kind "capture"')
    } finally {
      JUDGE_TIERS.capture = saved
    }
  })

  test('raising one kind changes only the requests of that kind', () => {
    for (const raised of KINDS) {
      const before = Object.fromEntries(KINDS.map((kind) => [kind, builders[kind]()]))
      const saved = JUDGE_TIERS[raised]
      JUDGE_TIERS[raised] = 'sonnet'
      try {
        for (const kind of KINDS) {
          const after = builders[kind]()
          if (kind === raised) {
            expect(after.model).toBe('sonnet')
            expect({ ...after, model: 'haiku' }).toEqual(before[kind])
          } else {
            expect(after).toEqual(before[kind])
          }
        }
      } finally {
        JUDGE_TIERS[raised] = saved
      }
    }
  })
})
