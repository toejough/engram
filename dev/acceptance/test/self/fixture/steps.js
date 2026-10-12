import assert from 'node:assert/strict'

import { After, Before, Given, Then, When } from '@cucumber/cucumber'

import { entryFor, recordResult } from '../../../lib/results.js'

Before(function ({ pickle }) {
  this.id = pickle.name.split(' ')[0]
  this.title = pickle.name.slice(this.id.length + 1)
  this.retries = 0
})

After(function ({ result }) {
  recordResult(entryFor(this, result))
})

Given(/^a world$/, () => undefined)
When(/^something happens$/, () => undefined)
Then(/^it holds$/, () => undefined)
Then(/^it does not hold$/, () => assert.equal(1, 2, 'the System does not do the thing'))
Then(/^the harness breaks$/, () => {
  throw new TypeError('a harness bug, not an assertion')
})
Then(/^it is not built$/, () => 'pending')
