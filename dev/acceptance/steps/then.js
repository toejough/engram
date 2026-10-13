import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'

import { Then } from '@cucumber/cucumber'

import { changes, lost } from '../lib/tree.js'
import { awaitIdle, currentSession, sendPrompt, settle } from '../lib/turn.js'
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
      `|()the working agent's dispatch text does not mention note ${Q}` +
      `|()the transcript shows note ${Q} retrieved by the agent and not delivered by the System)$`,
  ),
  async function (isDelivered, subagent, delivered, isNone, noneOf, isCount, counted, times, fetched, isDispatch, dispatched, isFetched, fetchedOnly) {
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
    } else if (isFetched != null) {
      // the promise of 793a2e1c: what the agent retrieved itself the System does not deliver (793b293)
      const rows = await mainRows(this)
      const sentinel = sentinelOf(this, fetchedOnly)
      assert.ok(retrievals(rows, sentinel).length > 0, `the agent did not retrieve note "${fetchedOnly}" itself`)
      assert.equal(deliveries(rows, sentinel).length, 0, `the System delivered note "${fetchedOnly}" that the agent retrieved itself`)
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

// The notes the System or the agent wrote: the vault's notes less the seeded ones, read once the System's work is idle
// and the sessions are closed.
async function writtenNotes(world) {
  const scenario = world.scenario
  await awaitIdle(scenario)
  await Promise.all(scenario.sessions.map((session) => session.close()))
  const seeded = [...scenario.notes.values()].map((note) => note.sentinel)
  return fs
    .readdirSync(scenario.vault)
    .filter((name) => name.endsWith('.md'))
    .map((name) => ({ name, text: fs.readFileSync(path.join(scenario.vault, name), 'utf8') }))
    .filter((note) => !seeded.some((sentinel) => note.text.includes(sentinel)))
}

// "stating L": scenarios.js gives the words that state the lesson (a pattern per lesson), because a note is worded by
// a model; the check is that a note written to the vault says it.
async function vaultHolds(world, quantity, lesson) {
  const pattern = world.data.lessons?.[lesson]
  if (!pattern) throw new Error(`scenarios.js has no words for the lesson "${lesson}" of ${world.id}`)
  const stating = (await writtenNotes(world)).filter((note) => pattern.test(note.text))
  const names = stating.map((note) => note.name).join(', ')
  if (quantity === 'a') assert.ok(stating.length > 0, `no note in the vault states "${lesson}"`)
  else assert.equal(stating.length, 1, `${stating.length} notes state "${lesson}", expected exactly one: ${names}`)
}

Then(
  new RegExp(
    `^(?:the vault holds (a|exactly one) note stating ${Q}` +
      `|the offer ${Q} was (?:folded into note ${Q}|accepted) and is no longer pending` +
      '|a later engram query returns a chunk from that turn' +
      "|every note written by the agent's own memory command is still in the vault)$",
  ),
  async function (quantity, lesson, _offer, _folded) {
    if (quantity === undefined) return NOT_BUILT
    await vaultHolds(this, quantity, lesson)
    return undefined
  },
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

// The System is the mod `engram update` deployed: a session of the home lists it among its loaded plugins (SP2).
const MOD_NAME = 'engram-memory'

function modLoaded(session) {
  const plugins = session.init?.plugins ?? []
  return plugins.some((plugin) => `${plugin.name ?? ''} ${plugin.path ?? ''}`.includes(MOD_NAME))
}

function treeBefore(world) {
  const before = world.scenario.treeBefore
  if (!before) throw new Error('no install tree was recorded before this step: a harness error, a Given or When is missing')
  return before
}

// A session of the home, started if none ran yet, that has answered one turn.
async function answered(world, prompt) {
  const scenario = world.scenario
  const session = currentSession(scenario)
  if (session.turns.length === 0) await sendPrompt(scenario, prompt)
  return session
}

function settingsOf(world) {
  return JSON.parse(fs.readFileSync(path.join(world.scenario.home, '.claude', 'settings.json'), 'utf8'))
}

Then(/^a Claude Code session runs with the System active and no other configuration$/, async function () {
  const session = await answered(this, 'Reply with the word ready.')
  assert.ok(modLoaded(session), `the session did not load the ${MOD_NAME} mod: plugins ${JSON.stringify(session.init?.plugins)}`)
  assert.equal(session.turns.at(-1).isError, false, `the session's turn ended in an error: ${session.turns.at(-1).text.slice(0, 200)}`)
  assert.deepEqual(settingsOf(this), this.scenario.baselineSettings, 'the update added configuration to the settings of the home')
})

Then(/^the install tree (is unchanged|has no Pi configuration|still holds the memory skills, guidance and shim it held before)$/, function (what) {
  const scenario = this.scenario
  if (what === 'has no Pi configuration') {
    assert.ok(!fs.existsSync(path.join(scenario.home, '.pi')), 'the update created a Pi directory')
  } else if (what === 'is unchanged') {
    assert.deepEqual(changes(treeBefore(this), scenario.snapshot()), [], 'the install tree changed')
  } else {
    assert.deepEqual(lost(treeBefore(this), scenario.snapshot()), [], 'the install tree lost what it held before')
  }
})

// "Identical" is the same install tree, the mod loaded as it was, and no System-attributed row in the transcript
// (793e2h): a prompt that matches no note is answered with nothing delivered.
Then(
  /^the (Claude Code|Pi) session behaves identically to (its behavior before the System existed|a session run after the first update)$/,
  async function (harness, baseline) {
    if (harness !== 'Claude Code' || baseline !== 'a session run after the first update') return NOT_BUILT
    assert.deepEqual(changes(treeBefore(this), this.scenario.snapshot()), [], 'the install tree changed')
    const session = await answered(this, 'What is two plus two? Answer in one word.')
    assert.ok(modLoaded(session), `the mod is not loaded as it was after the first update: plugins ${JSON.stringify(session.init?.plugins)}`)
    assert.equal(session.turns.at(-1).isError, false, `the turn ended in an error: ${session.turns.at(-1).text.slice(0, 200)}`)
    assert.equal(systemBlocks(await mainRows(this)).length, 0, 'the System added content to a session that matches no note')
    return undefined
  },
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
