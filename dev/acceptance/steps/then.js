import assert from 'node:assert/strict'

import { Then } from '@cucumber/cucumber'

import { finalReply, memoryCalls, readRows } from '../lib/transcript.js'

const NOT_BUILT = 'pending'
const Q = '"([^"]*)"'

Then(
  new RegExp(
    `^(?:the (?:subagent's )?transcript shows note ${Q} delivered with the instruction to apply it before the agent's next response or tool call` +
      "|the (?:subagent's |Pi )?transcript shows no content delivered, the instruction to apply it included" +
      `|the transcript shows note ${Q} delivered (\\d+) times? and retrieved by the agent (\\d+) times?` +
      `|the working agent's dispatch text does not mention note ${Q})$`,
  ),
  () => NOT_BUILT,
)

Then(
  /^the (?:subagent's )?transcript shows no request from the System to the agent to run a memory command, or to evaluate, approve, write or curate a note$/,
  () => NOT_BUILT,
)

Then(
  new RegExp(
    `^(?:the vault holds (?:a|exactly one) note stating ${Q}` +
      `|the offer ${Q} was (?:folded into note ${Q}|accepted) and is no longer pending` +
      '|a later engram query returns a chunk from that turn' +
      "|every note written by the agent's own memory command is still in the vault)$",
  ),
  () => NOT_BUILT,
)

Then(
  new RegExp(
    '^(?:the user is shown a warning that automatic memory was skipped because (?:the vault is unavailable|of an error|the System timed out)' +
      '|the user is shown a one-line summary with (?:3 items delivered, 1 note added, 1 note amended and 2 offers curated|1 note added and no other count)' +
      '|the user is shown no (?:warning|memory-activity summary)' +
      "|the (?:warning|summary) appears nowhere in the transcript(?: or in the working agent's next response)?)$",
  ),
  () => NOT_BUILT,
)

Then(/^the session proceeds normally, with no error from the System and no user input missing$/, () => NOT_BUILT)

Then(/^the System changed none of the session's tool calls, permission decisions or model$/, () => NOT_BUILT)

Then(new RegExp(`^the user's prompt text is exactly as typed, and note ${Q} is separate content attributed to its source$`), () => NOT_BUILT)

Then(
  /^the only content the System added to the working agent's context is the delivered items and the instruction to apply them$/,
  () => NOT_BUILT,
)

Then(/^(?:the next prompt was processed|the response to the completed turn appeared) without waiting for the System's work$/, () => NOT_BUILT)

Then(/^a Claude Code session runs with the System active and no other configuration$/, () => NOT_BUILT)

Then(
  /^the install tree (?:is unchanged|has no Pi configuration|still holds the memory skills, guidance and shim it held before)$/,
  () => NOT_BUILT,
)

Then(
  /^the (?:Claude Code|Pi) session behaves identically to (?:its behavior before the System existed|a session run after the first update)$/,
  () => NOT_BUILT,
)

Then(
  /^the Pi (?:agent runs recall before acting|agent's own ingest and learn run and write as they would without the System|transcript is identical to its copy)$/,
  () => NOT_BUILT,
)

Then(/^the transcript shows the results of the agent's own memory command(, the agent's reply to the user)? and no error$/, function (withReply) {
  const session = this.scenario.session
  if (!session) throw new Error('no session ran: a harness error, the scenario has no turn before this step')
  const rows = readRows(session.transcriptPath())
  const calls = memoryCalls(rows)
  assert.ok(calls.length > 0, "the agent ran no memory command of its own: the transcript has no engram command or memory skill call")
  for (const call of calls) {
    assert.ok(call.result !== null, `the memory command has no result: ${JSON.stringify(call.input)}`)
    assert.equal(call.result.isError, false, `the memory command returned an error: ${call.result.text.slice(0, 200)}`)
  }
  const sentinels = [...this.scenario.notes.values()].map((note) => note.sentinel)
  if (sentinels.length > 0) {
    assert.ok(
      calls.some((call) => sentinels.some((sentinel) => call.result.text.includes(sentinel))),
      'the results of the memory command do not show the seeded note',
    )
  }
  // an errored turn (an API error, the budget guard, no login) is not evidence about the System
  if (session.turns.at(-1).isError) throw new Error(`the last turn ended in an error: ${session.turns.at(-1).text.slice(0, 200)}`)
  if (withReply) assert.notEqual(finalReply(rows), '', 'the transcript has no reply of the agent')
})
