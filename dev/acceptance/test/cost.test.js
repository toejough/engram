import assert from 'node:assert/strict'
import test from 'node:test'

import { turnCost } from '../lib/cost.js'

test('the SDK reports a running total per session, so a turn costs the change since the previous result', () => {
  const totals = [0.0164239, 0.0318358, 0.0337758]
  let previous = 0
  const costs = totals.map((total) => {
    const cost = turnCost(previous, total)
    previous = total
    return cost
  })
  assert.ok(Math.abs(costs[0] - 0.0164239) < 1e-9)
  assert.ok(Math.abs(costs[1] - 0.0154119) < 1e-9)
  assert.ok(Math.abs(costs.reduce((a, b) => a + b, 0) - 0.0337758) < 1e-9)
})

test('a total that went down (a new process) counts whole, never negative', () => {
  assert.equal(turnCost(0.5, 0.02), 0.02)
  assert.equal(turnCost(0, undefined), 0)
})
