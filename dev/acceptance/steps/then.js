import assert from 'node:assert/strict'

import { Then } from '@cucumber/cucumber'

import { settle } from '../lib/turn.js'
import {
  agentCalls,
  deliveries,
  finalReply,
  memoryCalls,
  precedesResponse,
  readRows,
  retrievals,
  systemBlocks,
} from '../lib/transcript.js'

const NOT_BUILT = 'pending'
const Q = '"([^"]*)"'

// What a delivery looks like in the transcripts (SP4): the System's blocks are attachment rows of the agent that
// received them, the main transcript for the working agent and the subagent's own file for a subagent.
const APPLY = /\bapply\b/i
const ORIGIN_KIND = {
  'the human': 'sdk',
  'an automated task notification': 'task',
  'a scheduled trigger': 'schedul|cron',
  'a message from another session': 'peer',
}

async function mainRows(world) {
  const session = world.scenario.session
  if (!session) throw new Error('no session ran: a harness error, the scenario has no turn before this step')
  await settle()
  return readRows(session.transcriptPath())
}

async function subagentRows(world) {
  await settle()
  return world.scenario.session.subagentTranscripts().map((file) => readRows(file))
}

function sentinelOf(world, label) {
  const note = world.scenario.notes.get(label)
  if (!note) throw new Error(`no note "${label}" was seeded before this step`)
  return note.sentinel
}

// causedBy: the nearest earlier user row of the block is a prompt of the origin the scenario delivered from.
function causedBy(rows, index, origin) {
  const kind = ORIGIN_KIND[origin]
  if (!kind) return true
  for (let i = index - 1; i >= 0; i -= 1) {
    if (rows[i].type !== 'user') continue
    return new RegExp(kind).test(JSON.stringify([rows[i].origin, rows[i].turnOrigin, rows[i].promptSource]))
  }
  return false
}

Then(
  new RegExp(
    `^(?:()the (subagent's )?transcript shows note ${Q} delivered with the instruction to apply it before the agent's next response or tool call` +
      "|()the (subagent's |Pi )?transcript shows no content delivered, the instruction to apply it included" +
      `|()the transcript shows note ${Q} delivered (\\d+) times? and retrieved by the agent (\\d+) times?` +
      `|()the working agent's dispatch text does not mention note ${Q})$`,
  ),
  async function (isDelivered, subagent, delivered, isNone, noneOf, isCount, counted, times, fetched, isDispatch, dispatched) {
    if (isDelivered != null) {
      const groups = subagent ? await subagentRows(this) : [await mainRows(this)]
      const sentinel = sentinelOf(this, delivered)
      const found = groups.flatMap((rows) => deliveries(rows, sentinel).map((block) => ({ rows, ...block })))
      assert.ok(found.length > 0, `note "${delivered}" was not delivered to the ${subagent ? 'subagent' : 'working agent'}: no System block holds ${sentinel}`)
      const good = found.filter((block) => APPLY.test(block.text) && precedesResponse(block.rows, block.index) && (subagent || causedBy(block.rows, block.index, this.origin)))
      assert.ok(good.length > 0, `note "${delivered}" was delivered, but not with the instruction to apply it before the agent's next response${this.origin ? ` at the prompt from ${this.origin}` : ''}`)
    } else if (isNone != null) {
      if (noneOf === 'Pi ') return NOT_BUILT
      const groups = noneOf === "subagent's " ? await subagentRows(this) : [await mainRows(this), ...(await subagentRows(this))]
      const blocks = groups.flatMap((rows) => systemBlocks(rows))
      assert.equal(blocks.length, 0, `the System delivered content: ${blocks[0]?.text.slice(0, 200)}`)
    } else if (isCount != null) {
      const rows = await mainRows(this)
      const sentinel = sentinelOf(this, counted)
      assert.equal(deliveries(rows, sentinel).length, Number(times), `note "${counted}" delivered a different number of times`)
      assert.equal(retrievals(rows, sentinel).length, Number(fetched), `note "${counted}" retrieved by the agent a different number of times`)
    } else {
      const rows = await mainRows(this)
      const sentinel = sentinelOf(this, dispatched)
      const dispatches = agentCalls(rows)
      assert.ok(dispatches.length > 0, 'the working agent dispatched no subagent')
      for (const call of dispatches) {
        assert.ok(!JSON.stringify(call.input).includes(sentinel), `the dispatch text mentions note "${dispatched}"`)
      }
    }
    return undefined
  },
)

// A request is an instruction to run a memory command or to evaluate, approve, write or curate a note; the framing
// of a block (everything but its item lines) must hold none.
const REQUEST = /\b(run|execute|invoke|call)\b[^.\n]*\b(engram|recall|learn|curate|write-memory|memory command)\b|\b(evaluate|approve|curate)\b|\bwrite (a|the) note\b/i

Then(
  /^the (subagent's )?transcript shows no request from the System to the agent to run a memory command, or to evaluate, approve, write or curate a note$/,
  async function (subagent) {
    const groups = subagent ? await subagentRows(this) : [await mainRows(this)]
    for (const rows of groups) {
      for (const block of systemBlocks(rows)) {
        const framing = block.text.split('\n').filter((line) => !line.startsWith('- ')).join('\n')
        assert.ok(!REQUEST.test(framing), `a System block asks something of the agent: ${framing.slice(0, 300)}`)
      }
    }
    return undefined
  },
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
