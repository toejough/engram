// The Then steps of node 793e8d: housekeeping after the turn, purview, and failure isolation. Each reads what the
// System left after its background work settled (the seam's idle line) and the sessions were closed.
import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

import { Then } from '@cucumber/cucumber'

import {
  MARKER,
  SKIPPED,
  deliveries,
  finalReply,
  memoryCalls,
  readRows,
  rowsMentioning,
  statusTexts,
  systemBlocks,
  toolCalls,
  typedRows,
} from '../lib/transcript.js'
import { APPLY, Q, REQUEST, mainRows, quiesce, sentinelOf, subagentRows, vaultHolds } from './shared.js'

const MOD_NAME = 'engram-memory'
// the fixed prefix of the System's summary line (mod: summary.js)
const SUMMARY_PREFIX = 'Memory activity: '
// An item line of a delivery: a path, the situation in brackets, then the text.
const ITEM_LINE = /^- \S+\.md \(.*\): /
// Events a mod could use to change a tool call's arguments or its permission, the model, or the prompt (793a4c).
const PURVIEW_EVENTS = [
  'tool.call', 'tool.check', 'tool.register', 'classic.PreToolUse', 'classic.PermissionRequest', 'session.authorize',
  'session.model', 'prompt.compose', 'prompt.edit', 'prompt.section', 'model.classify', 'model.complete', 'model.fork',
  'turn.start', 'turn.step', 'agent.register', 'skill.prompt',
]

function statuses(world) {
  return world.scenario.sessions.flatMap((session) => statusTexts(session.messages))
}

function vaultFiles(world) {
  return fs.readdirSync(world.scenario.vault).filter((name) => name.endsWith('.md') && name !== 'README.md')
}

function readNote(world, name) {
  return fs.readFileSync(path.join(world.scenario.vault, name), 'utf8')
}

function offerOf(world, label) {
  const offer = world.scenario.offers.get(label)
  if (!offer) throw new Error(`no offer "${label}" was seeded before this step`)
  return offer
}

// ---- 793a4j: housekeeping after the turn -------------------------------------------------------------------------

// Row 16 of the catalog: what the System left in the vault and the index.
Then(
  new RegExp(
    `^(?:the vault holds (a|exactly one) note stating ${Q}` +
      `|the offer ${Q} was (?:(folded) into note ${Q}|(accepted)) and is no longer pending` +
      '|(a later engram query returns a chunk from that turn)' +
      "|every note written by the agent's own memory command is still in the vault)$",
  ),
  async function (quantity, lesson, offerLabel, folded, noteLabel, accepted, chunk) {
    if (quantity != null) return vaultHolds(this, quantity, lesson)
    if (offerLabel != null) {
      await quiesce(this)
      if (folded) return foldedInto(this, offerLabel, noteLabel)
      return acceptedOffer(this, offerLabel)
    }
    if (chunk) return chunkReturned(this)
    return agentNotesRemain(this)
  },
)

// 793a5k: the notes the agent's own memory command wrote (the path `engram learn` prints, in the command's result)
// are still in the vault once the System's work is idle: the System neither undid nor discarded them.
async function agentNotesRemain(world) {
  await quiesce(world)
  const session = world.scenario.session
  const written = memoryCalls(readRows(session.transcriptPath()))
    .filter((call) => call.name === 'Bash' && /(^|[\s;&|(])engram\s+learn\b/.test(call.input.command ?? ''))
    .flatMap((call) => (call.result?.text ?? '').split('\n'))
    .map((line) => line.trim().split('/').at(-1))
    .filter((name) => name.endsWith('.md'))
  if (world.data.agentWrites) assert.ok(written.length > 0, "the agent's own memory command wrote no note: a precondition of the scenario did not hold")
  for (const name of written) assert.ok(fs.existsSync(path.join(world.scenario.vault, name)), `the note ${name}, written by the agent's own memory command, is no longer in the vault`)
}

async function chunkReturned(world) {
  await quiesce(world)
  const probe = world.data.chunk
  if (!probe) throw new Error(`scenarios.js has no chunk to look for for ${world.id}`)
  const output = await world.scenario.engram(['query', '--text', probe.query, '--phrase', probe.query, '--limit', '10'])
  const chunks = output.split(/^ {2}- path: /m).slice(1).filter((item) => /^ {4}kind: chunk$/m.test(item))
  assert.ok(chunks.some((chunk) => chunk.includes(probe.text)), `no chunk of the index holds "${probe.text}" (${chunks.length} chunks returned)`)
}

function foldedInto(world, label, noteLabel) {
  const offer = offerOf(world, label)
  const sentinel = sentinelOf(world, noteLabel)
  assert.ok(!fs.existsSync(path.join(world.scenario.vault, offer.basename)), `the offer "${label}" (${offer.basename}) is still in the vault`)
  const holders = vaultFiles(world).filter((name) => readNote(world, name).includes(sentinel))
  assert.equal(holders.length, 1, `${holders.length} notes hold the sentinel of "${noteLabel}", expected the one note it was folded into: ${holders.join(', ')}`)
  const note = readNote(world, holders[0])
  assert.ok(!/^pending: true$/m.test(note), `note "${noteLabel}" is pending`)
  assert.ok(note.includes(offer.basename.replace(/\.md$/, '')), `note "${noteLabel}" does not record the offer ${offer.basename} it absorbed (--into keeps its identity)`)
}

function acceptedOffer(world, label) {
  const offer = offerOf(world, label)
  const file = path.join(world.scenario.vault, offer.basename)
  assert.ok(fs.existsSync(file), `the offer "${label}" (${offer.basename}) is no longer in the vault: it was discarded, not accepted`)
  const text = fs.readFileSync(file, 'utf8')
  assert.ok(!/^pending: true$/m.test(text), `the offer "${label}" is still pending`)
  assert.ok(text.includes(offer.sentinel), `the accepted offer "${label}" lost its content`)
}

Then(
  /^the (next prompt was processed|response to the completed turn appeared) without waiting for the System's work$/,
  async function (which) {
    await quiesce(this)
    const scenario = this.scenario
    const delayMs = scenario.shimSeconds * 1000
    assert.ok(delayMs > 0, 'no slow work was injected: a harness error, a Given is missing')
    const slow = scenario.shimCalls().filter((call) => call.what.includes('slept-start'))
    assert.ok(slow.length > 0, 'the System did no slow work in this scenario: the delay was never exercised')
    const turns = scenario.session.turns
    const rows = readRows(scenario.session.transcriptPath())
    const first = turns[0]
    const second = turns[1]
    assert.ok(first && second, `the scenario ran ${turns.length} turns, expected two`)
    if (which === 'response to the completed turn appeared') {
      assert.ok(!first.isError, `the first turn ended in an error: ${first.text.slice(0, 200)}`)
      assert.ok(first.wallMs < delayMs, `the response took ${first.wallMs} ms, as long as the System's slow work (${delayMs} ms)`)
      return
    }
    assert.ok(!second.isError, `the next turn ended in an error: ${second.text.slice(0, 200)}`)
    assert.equal(typedRows(rows, this.typed).length, 1, 'the next prompt is not in the transcript as typed')
    assert.ok(second.wallMs < delayMs, `the next prompt took ${second.wallMs} ms, as long as the System's slow work (${delayMs} ms)`)
    const slowStart = Math.min(...slow.map((call) => call.at * 1000))
    assert.ok(second.endedAt < slowStart + delayMs, 'the next prompt was answered only after the System had finished its slow work')
  },
)

// ---- 793a4c: purview ---------------------------------------------------------------------------------------------

function hookedEvents(world) {
  const dir = path.join(world.scenario.home, '.claude', 'skills', MOD_NAME)
  let output
  try {
    output = execFileSync(world.scenario.context.claudePath, ['plugin', 'validate', dir], { encoding: 'utf8', env: world.scenario.env(), cwd: world.scenario.workspace })
  } catch (error) {
    output = `${error.stdout ?? ''}${error.stderr ?? ''}`
  }
  const line = output.split('\n').find((text) => /register\.js hooks:/.test(text))
  if (!line) throw new Error(`claude plugin validate did not list the hooks of ${dir}: ${output.slice(0, 300)}`)
  return line.split('hooks:')[1].split(',').map((name) => name.trim()).filter((name) => name !== '')
}

Then(/^the System changed none of the session's tool calls, permission decisions or model$/, async function () {
  await quiesce(this)
  const scenario = this.scenario
  // the System has no handler that could change them
  const hooked = hookedEvents(this)
  assert.deepEqual(hooked.filter((name) => PURVIEW_EVENTS.includes(name)), [], `the mod hooks events that change tool calls, permissions, the model or the prompt: ${hooked.join(', ')}`)
  const session = scenario.session
  const rows = readRows(session.transcriptPath())
  // the model: every reply of the session is the configured model's
  const configured = session.init?.model
  assert.ok(configured, 'the session reported no model')
  const models = [...new Set(rows.filter((row) => row.type === 'assistant').map((row) => row.message?.model).filter((model) => model && model !== '<synthetic>'))]
  assert.deepEqual(models.filter((model) => model !== configured), [], `a reply came from a model other than the configured ${configured}`)
  // the tool calls and permission decisions: each call is one the settings allow, with the arguments the agent gave,
  // and none was refused or answered by the System
  const allowed = scenario.baselineSettings.permissions.allow.map((rule) => rule.replace(/\(.*\)$/, ''))
  const calls = toolCalls(rows)
  assert.ok(calls.length > 0, 'the session made no tool call to compare')
  for (const call of calls) {
    assert.ok(allowed.includes(call.name), `tool ${call.name} is not one the settings allow`)
    assert.ok(call.result !== null, `tool call ${call.name} has no result`)
    assert.ok(!(call.result.isError && /permission|not allowed|denied/i.test(call.result.text)), `tool call ${call.name} was refused: ${call.result.text.slice(0, 200)}`)
    if (call.name === 'Bash' && this.data.commands) assert.match(call.input.command, this.data.commands, 'a command differs from the one the agent was told to run')
  }
})

Then(new RegExp(`^the user's prompt text is exactly as typed, and note ${Q} is separate content attributed to its source$`), async function (label) {
  await quiesce(this)
  const rows = await mainRows(this)
  const sentinel = sentinelOf(this, label)
  assert.equal(typedRows(rows, this.typed).length, 1, `the transcript does not hold the prompt exactly as typed (once): ${this.typed}`)
  assert.equal(rows.filter((row) => row.type === 'user' && JSON.stringify(row.message).includes(sentinel)).length, 0, 'the note is inside a user row')
  const blocks = deliveries(rows, sentinel)
  assert.ok(blocks.length > 0, `note "${label}" was not delivered`)
  for (const block of blocks) {
    assert.equal(rows[block.index].type, 'attachment', 'the note is not a row of its own')
    assert.ok(block.text.startsWith(MARKER), 'the note is not marked as the System\'s')
    assert.ok(block.text.split('\n').some((line) => ITEM_LINE.test(line) && line.includes(sentinel)), 'the note is not attributed to its source (its path)')
  }
})

Then(/^the only content the System added to the working agent's context is the delivered items and the instruction to apply them$/, async function () {
  await quiesce(this)
  const groups = [await mainRows(this), ...(await subagentRows(this))]
  let seen = 0
  for (const rows of groups) {
    const context = rows.filter((row) => row.type === 'attachment' && row.attachment?.type === 'hook_additional_context')
    const blocks = systemBlocks(rows)
    assert.equal(blocks.length, context.length, 'context was added that is not a block of the System (no marker line)')
    for (const block of blocks) {
      seen += 1
      assert.ok(APPLY.test(block.text), 'a block holds no instruction to apply it')
      const lines = block.text.split('\n').slice(1)
      const items = lines.filter((line) => line.startsWith('- '))
      assert.ok(items.length > 0, 'a block delivers no item')
      for (const item of items) assert.match(item, ITEM_LINE, 'a block line is no delivered item')
      const framing = lines.filter((line) => !line.startsWith('- ')).join('\n')
      assert.ok(!REQUEST.test(framing), `the instruction asks more than to apply: ${framing.slice(0, 300)}`)
    }
  }
  assert.ok(seen > 0, 'the System added nothing: a precondition of the scenario (a delivery) did not hold')
})

// ---- 793a4b: failure isolation -----------------------------------------------------------------------------------

// Row 17 of the catalog: what the user is shown on the status line, and that none of it is in the transcript.
Then(
  new RegExp(
    '^(?:the user is shown a warning that automatic memory was skipped because (the vault is unavailable|of an error|the System timed out)' +
      '|the user is shown a one-line summary with (3 items delivered, 1 note added, 1 note amended and 2 offers curated|1 note added and no other count)' +
      '|the user is shown no (warning|memory-activity summary)' +
      "|the (warning|summary) appears nowhere in the transcript( or in the working agent's next response)?)$",
  ),
  async function (reason, counts, noneOf, nowhereOf, inResponse) {
    if (counts != null) return summaryShown(this, counts)
    if (reason != null) return warningShown(this, reason)
    if (noneOf != null) return noneOf === 'warning' ? noWarning(this) : noSummary(this)
    return nowhereOf === 'warning' ? warningNotInTranscript(this) : summaryNotInContext(this, inResponse != null)
  },
)

async function warningShown(world, reason) {
  await quiesce(world)
  const shown = statuses(world)
  assert.ok(
    shown.some((text) => SKIPPED.test(text) && text.toLowerCase().includes(`because ${reason}`.toLowerCase())),
    `no status message warns that automatic memory was skipped because ${reason}; the status messages were ${JSON.stringify(shown)}`,
  )
  return undefined
}

async function noWarning(world) {
  await quiesce(world)
  assert.deepEqual(statuses(world).filter((text) => SKIPPED.test(text)), [], 'the user was shown a warning')
  return undefined
}

async function warningNotInTranscript(world) {
  await quiesce(world)
  assert.ok(statuses(world).some((text) => SKIPPED.test(text)), 'no warning was shown, so there is nothing to look for in the transcript')
  const groups = [await mainRows(world), ...(await subagentRows(world))]
  for (const rows of groups) assert.equal(rowsMentioning(rows, SKIPPED).length, 0, 'the warning is in the transcript')
  return undefined
}

// ---- 793a5h: the summary --------------------------------------------------------------------------------------

// summaries are the status messages that are a memory-activity summary (the warning may share the line, 793b302)
const summaries = (world) => statuses(world).filter((text) => text.includes(SUMMARY_PREFIX))

// "with COUNTS": the line says these counts; "and no other count" is a line that says nothing more.
async function summaryShown(world, counts) {
  await quiesce(world)
  const shown = statuses(world)
  const exact = counts.endsWith(' and no other count')
  const expected = `${SUMMARY_PREFIX}${exact ? counts.slice(0, -' and no other count'.length) : counts}`
  const found = summaries(world).filter((text) => (exact ? text.endsWith(expected) : text.includes(expected)))
  assert.ok(found.length > 0, `no status message is a one-line summary with ${counts}; the status messages were ${JSON.stringify(shown)}`)
  for (const text of found) assert.ok(!/[\r\n]/.test(text), 'the summary is not one line')
  return undefined
}

async function noSummary(world) {
  await quiesce(world)
  assert.deepEqual(summaries(world), [], 'the user was shown a memory-activity summary')
  return undefined
}

// the summary is out of the agent's context: no row of any transcript holds a line the user was shown, and the
// agent's reply to the turn after it does not either
async function summaryNotInContext(world, inResponse) {
  await quiesce(world)
  const lines = summaries(world)
  assert.ok(lines.length > 0, 'no summary was shown, so there is nothing to look for in the transcript')
  const groups = [await mainRows(world), ...(await subagentRows(world))]
  for (const rows of groups) {
    assert.equal(rowsMentioning(rows, new RegExp(SUMMARY_PREFIX.trim(), 'i')).length, 0, 'the summary is in the transcript')
    for (const line of lines) assert.equal(rows.filter((row) => JSON.stringify(row).includes(line)).length, 0, `the summary line "${line}" is in the transcript`)
  }
  if (inResponse) {
    const reply = finalReply(await mainRows(world))
    for (const line of lines) assert.ok(!reply.includes(line), "the working agent's next response repeats the summary")
    assert.ok(!/memory activity/i.test(reply), "the working agent's next response mentions the memory activity")
  }
  return undefined
}

Then(/^the session proceeds normally, with no error from the System and no user input missing$/, async function () {
  await quiesce(this)
  const session = this.scenario.session
  const rows = readRows(session.transcriptPath())
  for (const turn of session.turns) assert.ok(!turn.isError, `a turn ended in an error: ${turn.text.slice(0, 200)}`)
  assert.equal(typedRows(rows, this.typed).length, 1, 'the typed prompt is not in the transcript exactly once')
  assert.notEqual(finalReply(rows), '', 'the agent did not answer')
  const errors = rowsMentioning(rows, /HooksError|engram-memory:|hook (?:failed|error)|automatic memory/i)
  assert.equal(errors.length, 0, `the transcript holds an error from the System: ${JSON.stringify(errors[0] ?? '').slice(0, 300)}`)
})
