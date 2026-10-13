// The steps of the H1 scenarios (793d17, 793e8i): live experiments that build each probe once as a mod and once as a
// settings hook and compare them. The tours (h1/tours.js) are the scripted sessions; a step reads what a tour returned
// (h1/capabilities.js). What the Confirmed-if of 793d17 requires of the mod is asserted; what a settings hook does is
// recorded in the observations (h1/results/), because a hook that cannot do something is a finding and not a failed
// experiment (793b334). `targ h1-report` reads the observations.
import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

import { After, AfterAll, Given, Then, When } from '@cucumber/cucumber'

import { getRunContext } from '../lib/run-state.js'
import { attribution, backgroundWork, capabilityFor, failure, load, longWork, modelCall, presence, screen } from '../h1/capabilities.js'
import { counts } from '../h1/count.js'
import { CAUSES } from '../h1/tours.js'
import { failureKey, prefetch, tour } from '../h1/runs.js'
import { compact, flush, observationsOf, readObservations, recordObservation } from '../h1/results.js'
import { FILES } from '../h1/count.js'
import { MOD_DIRS, spend } from '../h1/surfaces.js'

const SURFACES = ['mod', 'hook']

// merge records the observation of one key of the scenario, adding to what an earlier example of an outline recorded
function merge(world, key, part) {
  recordObservation(world.id, key, { ...(observationsOf(world.id)[key] ?? {}), ...part })
}

// keep stores the compacted data of a tour with the scenario's observations, once per tour
function keep(world, name, data) {
  merge(world, 'tours', { [name]: compact(data) })
}

After(async function () {
  if (this.data?.h1) flush(this.id, (await getRunContext()).creds)
})

// the tours' sessions are not the scenarios' own World, so their cost is told here
AfterAll(function () {
  if (spend.sessions > 0) console.log(`H1 tours: ${spend.sessions} sessions, ${spend.turns} turns, ${spend.costUsd.toFixed(3)} USD (the SDK's client-side estimate), ${(spend.wallMs / 1000).toFixed(0)} s of turns`)
})

// ---- 793d17a1a: the nine capabilities ------------------------------------------------------------------------------

Given(/^a probe for (.+) built once as a mod module and once as a settings hook, both writing what they observed to a log outside the transcript$/, function (text) {
  const cap = capabilityFor(text)
  if (!cap) throw new Error(`H1 has no probe for the capability "${text}"`)
  for (const file of [...FILES.mod.code, ...FILES.hook.code]) assert.ok(fs.existsSync(file), `the probe ${file} is not built`)
  this.h1 = { cap }
})

Given(/^a scripted session that triggers (.+)$/, function (text) {
  const cap = capabilityFor(text)
  assert.ok(cap && cap === this.h1?.cap, `the scripted session is for "${text}"`)
  prefetch(cap.tour === 'main' ? ['main:mod', 'main:hook'] : [`${cap.tour}:mod`, `${cap.tour}:hook`])
})

When('the session runs on the verified Claude Code version', async function () {
  const { claudePath } = await getRunContext()
  this.h1.version = execFileSync(claudePath, ['--version'], { encoding: 'utf8' }).trim()
  merge(this, 'environment', { claude: this.h1.version })
})

Then(/^each probe's log records whether (.+) was achieved$/, async function (text) {
  const cap = capabilityFor(text)
  const found = {}
  for (const surface of SURFACES) {
    const data = await tour(`${cap.tour}:${surface}`)
    found[surface] = cap.evaluate(data, surface)
    keep(this, `${cap.tour}:${surface}`, data)
    assert.equal(typeof found[surface].achieved, 'boolean', `the ${surface} probe's log does not say whether "${text}" was achieved`)
  }
  this.h1.found = found
  merge(this, 'capabilities', { [cap.key]: { text, mod: found.mod, hook: found.hook } })
})

Then('the experiment table gets a mod column and a hook column for it', function () {
  assert.ok(this.h1.found.mod && this.h1.found.hook, 'a column is missing')
})

// ---- 793d17a1b: attributable context --------------------------------------------------------------------------------

Given('a mod that adds one block through each channel: after a prompt, next to a tool result and at a subagent\'s start', function () {
  prefetch(['main:mod', 'main:hook', 'main:none'])
})

When('the session runs and its transcript file is read', async function () {
  const control = await tour('main:none')
  this.h1 = { found: {} }
  for (const surface of SURFACES) {
    const data = await tour(`main:${surface}`)
    this.h1.found[surface] = attribution(data, control)
    keep(this, `main:${surface}`, data)
  }
  keep(this, 'main:none', control)
  merge(this, 'attribution', this.h1.found)
})

Then(/^each block appears as its own attachment row of type hook_additional_context, in the transcript of the agent that received it .*$/, function () {
  const { mod } = this.h1.found
  assert.ok(mod.ownRows, `the mod's blocks are not each an attachment row: ${JSON.stringify(mod)}`)
  assert.ok(mod.channelTold, `a block does not name the hook event that added it: ${JSON.stringify(mod)}`)
  assert.equal(mod.subagentInMain.length, 0, "the subagent's start block is in the main transcript")
})

Then('the prompt text and the tool result are exactly as they were without the mod', function () {
  const { mod } = this.h1.found
  assert.ok(mod.promptsUnchanged, 'the prompt as typed differs from the control run')
  assert.ok(mod.toolResultUnchanged, 'the tool result differs from the control run')
})

// ---- 793d17a1c: what the surface can read of the context -------------------------------------------------------------

Given(/^a mod that delivered block X to the main agent, and a probe that lists the delivered blocks .*$/, function () {
  prefetch(['presence:mod', 'presence:hook'])
})

When('the session is compacted so that X no longer reaches the model, X is delivered again, and the session is then ended and resumed', async function () {
  this.h1 = { found: {} }
  for (const surface of SURFACES) {
    const data = await tour(`presence:${surface}`)
    this.h1.found[surface] = presence(data)
    keep(this, `presence:${surface}`, data)
  }
})

Then('the probe lists X after its delivery, does not list X after the compaction removed it, and lists X again after the resume restored it', function () {
  merge(this, 'presence', this.h1.found)
  assert.ok(this.h1.found.mod.holds, `the mod's probe lists ${JSON.stringify(this.h1.found.mod.listed)} at the five prompts`)
})

// ---- 793d17b1a: the screen --------------------------------------------------------------------------------------------

Given(/^a mod that shows the user one warning line, and a settings hook that returns the equivalent user-facing message, in an interactive session under tmux.*$/, function () {
  prefetch(['screen:mod', 'screen:hook'])
})

When('the session runs a turn and then the next prompt', async function () {
  this.h1 = { found: {} }
  for (const surface of SURFACES) {
    const data = await tour(`screen:${surface}`)
    this.h1.found[surface] = screen(data)
    merge(this, 'screenShots', { [surface]: data.screenLines })
  }
  merge(this, 'screen', this.h1.found)
})

Then('the user sees each message on screen', function () {
  for (const surface of SURFACES) assert.ok(this.h1.found[surface].onScreen, `the ${surface} message is not on the screen`)
})

Then("neither message appears in the transcript file or in the messages the model's next request carries", function () {
  const { mod, hook } = this.h1.found
  assert.ok(!mod.inTranscriptFile && !mod.modelQuotedIt, `the mod's message is in the transcript file or was quoted by the model: ${JSON.stringify(mod)}`)
  // the settings hook's message is recorded: it sits in the transcript file as a row of its own (hook_system_message)
  merge(this, 'screenHookNote', { inTranscriptFile: hook.inTranscriptFile, rowKinds: hook.transcriptRowKinds, modelQuotedIt: hook.modelQuotedIt })
})

// ---- 793d17c1a: the model call ------------------------------------------------------------------------------------------

Given(/^a mod that relays one short judgement request naming the model "haiku", and a settings hook script that must make the same call$/, function () {
  prefetch(['model:mod', 'model:hook'])
})

When('both run', async function () {
  this.h1 = { found: {} }
  for (const surface of SURFACES) {
    const data = await tour(`model:${surface}`)
    this.h1.found[surface] = modelCall(data, surface)
    keep(this, `model:${surface}`, data)
  }
  merge(this, 'modelCall', this.h1.found)
})

Then("the mod's call returns an answered reply with its text and token usage, on the session's credentials, and the debug log of the session names a model of the haiku family as the one that answered", function () {
  const { mod } = this.h1.found
  assert.ok(mod.answered && mod.text, `the mod's call was not answered: ${JSON.stringify(mod)}`)
  assert.ok(mod.tokens > 0, 'the reply holds no token usage')
  assert.ok(mod.haikuFamily, `the debug log does not name a haiku model: ${JSON.stringify(mod.models)}`)
})

Then(/^the report counts what the hook script needed for the same call \(credentials, subprocess and parsing\)$/, function () {
  const code = counts()
  merge(this, 'modelCallCost', {
    modLines: code.mod.modelCallLines,
    hookLines: code.hook.modelCallLines,
    hookNeeded: ['a credential passed under another name (Claude Code removes the session\'s from a hook\'s environment)', 'a `claude -p` subprocess with eight flags that leave out the tools, settings and default prompt', 'parsing of its JSON result', 'the model id read from modelUsage (the mod reads it from the debug log)'],
  })
  assert.ok(this.h1.found.hook.answered, `the hook script's call was not answered: ${JSON.stringify(this.h1.found.hook)}`)
})

// ---- 793d17c1b: work longer than the hook budget ---------------------------------------------------------------------------

Given("a stand-in command that returns a delivery after 25 seconds, and a handler that awaits it through the mods API", function () {
  prefetch(['long:mod', 'long:hook', 'long:hook-timeout10'])
})

When('a prompt is submitted', async function () {
  this.h1 = { found: {} }
  for (const [label, name, surface] of [['mod', 'long:mod', 'mod'], ['hook', 'long:hook', 'hook'], ['hookTimeout10', 'long:hook-timeout10', 'hook']]) {
    const data = await tour(name)
    this.h1.found[label] = longWork(data, surface)
    keep(this, name, data)
  }
  merge(this, 'longWork', this.h1.found)
})

Then('the prompt continues with the delivery after about 25 seconds and the handler is not cut off', function () {
  const { mod } = this.h1.found
  assert.ok(mod.delivered, 'the delivery did not reach the prompt')
  assert.ok(mod.turnWallS >= 24 && mod.turnWallS < 40, `the turn took ${mod.turnWallS} s, not about 25`)
})

Then("the handler's own time stays under the hook budget of 10 seconds", function () {
  assert.ok(this.h1.found.mod.ownMs < 10000, `the handler's own time was ${this.h1.found.mod.ownMs} ms`)
})

Then('the report records, for a settings hook running the same stand-in command, whether it was cut off and what setting avoided that', function () {
  const { hook, hookTimeout10 } = this.h1.found
  merge(this, 'longWorkHook', {
    cutOffAtDefault: !hook.delivered,
    defaultTimeoutSeconds: 30,
    cutOffWithTimeout10: hookTimeout10.cancelled || !hookTimeout10.delivered,
    avoidedBy: 'the timeout field of the hook entry (the default of a UserPromptSubmit command hook is 30 s, so 25 s needs no setting here; a longer job needs a larger value)',
  })
})

// ---- 793d17c1c: background work ----------------------------------------------------------------------------------------------

Given('a handler that starts a 20 second job in the background and continues its event at once, and a settings hook that starts the same job detached', function () {
  prefetch(['background:mod', 'background:mod-late', 'background:hook', 'background:hook-async'])
})

When('a turn ends and the next prompt is submitted immediately', async function () {
  this.h1 = { found: {} }
  for (const [label, name] of [['mod', 'background:mod'], ['modLate', 'background:mod-late'], ['hook', 'background:hook'], ['hookAsync', 'background:hook-async']]) {
    const data = await tour(name)
    this.h1.found[label] = backgroundWork(data)
    keep(this, name, data)
  }
  merge(this, 'backgroundWork', this.h1.found)
})

Then('the next prompt is processed without waiting for the job, under both surfaces', function () {
  assert.ok(this.h1.found.mod.notDelayed, `the next prompt took ${this.h1.found.mod.secondPromptS} s under the mod`)
  assert.ok(this.h1.found.hook.notDelayed, `the next prompt took ${this.h1.found.hook.secondPromptS} s under the settings hook`)
})

Then(/^the job's result appears after about 20 seconds and, in the mod, reaches the agent at the first tool result that follows it .*$/, function () {
  const { mod } = this.h1.found
  assert.ok(mod.jobDoneAfterS >= 19 && mod.jobDoneAfterS < 25, `the job finished after ${mod.jobDoneAfterS} s`)
  assert.ok(mod.reachedAgent && mod.deliveredAt, 'the result did not reach the agent at a tool result')
})

Then("the report records how a settings hook's detached result would reach the agent", function () {
  const { hook, hookAsync, modLate } = this.h1.found
  merge(this, 'backgroundWorkNotes', {
    hookDetached: `a detached process writes a file; the next PostToolUse hook reads it and returns additionalContext (reached the agent: ${hook.reachedAgent})`,
    hookAsync: `an async hook returns its additionalContext when it exits; delivered with the next conversation turn (reached the agent: ${hookAsync.reachedAgent}; next prompt took ${hookAsync.secondPromptS} s)`,
    modTimerInsideOpenHandler: `a mod whose handler made an API call after setting the timer held the next prompt for ${modLate.secondPromptS} s: the timer's callback ran inside the open handler's scope`,
  })
})

// ---- 793d17d1a: load -------------------------------------------------------------------------------------------------------------

Given('the mod directory in the engram-owned root, a link to it in the skills directory, and no flag, install step or settings edit', function () {
  prefetch(['load:mod', 'load:hook', 'load:hook-plugin'])
})

When(/^a new session starts, is cleared .*, is resumed .* and is branched .*$/, async function () {
  this.h1 = { found: {} }
  for (const surface of SURFACES) {
    const data = await tour(`load:${surface}`)
    this.h1.found[surface] = load(data, surface)
    keep(this, `load:${surface}`, data)
  }
  const plugin = await tour('load:hook-plugin')
  keep(this, 'load:hook-plugin', plugin)
  this.h1.hookPlugin = { settingsHasHooks: plugin.settingsHasHooks, fired: plugin.events.map((event) => event.kind), validated: plugin.validated }
  merge(this, 'load', { ...this.h1.found, hookPlugin: this.h1.hookPlugin })
})

const SESSION_SOURCES = ['startup', 'clear', 'resume', 'fork']

Then("the mod's probe log shows its register ran and a handler fired in each, and the debug log of each session says that the module loaded and lists its events", function () {
  const { mod } = this.h1.found
  assert.deepEqual(mod.sessionStarts, SESSION_SOURCES, `the mod's session starts: ${JSON.stringify(mod.sessionStarts)}`)
  assert.equal(mod.promptsHandled, 4, 'a handler did not fire in every session')
  assert.ok(mod.registerRan >= 3, `register ran ${mod.registerRan} times`)
  for (const name of Object.keys(MOD_DIRS)) assert.ok(mod.moduleLoadedLines.some((line) => line.includes(`hooks module ${name}@skills-dir loaded`) && line.includes('events:')), `the debug log does not say that ${name} loaded and lists its events`)
})

Then('a settings hook probe built to the same events fires in the same sessions', function () {
  const { hook } = this.h1.found
  assert.deepEqual(hook.sessionStarts, SESSION_SOURCES, `the hook's session starts: ${JSON.stringify(hook.sessionStarts)}`)
  assert.equal(hook.promptsHandled, 4)
})

Then('claude plugin validate accepts the directory', function () {
  for (const entry of this.h1.found.mod.validated) assert.ok(entry.ok, `claude plugin validate refused ${entry.dir}: ${entry.output}`)
})

Then('the directory in the skills directory is a link whose target is in the engram-owned root', function () {
  for (const entry of this.h1.found.mod.inspect) {
    assert.ok(entry.isLink, `${entry.link} is not a link`)
    assert.ok(entry.target.includes(`${path.sep}.claude${path.sep}engram${path.sep}`), `the target ${entry.target} is not in the engram-owned root`)
  }
})

// ---- 793d17e1a: a failure inside the surface -----------------------------------------------------------------------------------------

Given(/^(.+) while the mod's handler, and the settings hook script, handle a prompt$/, function (cause) {
  assert.ok(CAUSES[cause], `H1 has no cause "${cause}"`)
  this.h1 = { cause, found: {} }
  prefetch(SURFACES.flatMap((surface) => [true, false].map((contained) => failureKey(surface, cause, contained))))
})

When('the handler finishes', async function () {
  for (const surface of SURFACES) {
    for (const contained of [true, false]) {
      const data = await tour(failureKey(surface, this.h1.cause, contained))
      this.h1.found[`${surface}-${contained ? 'contained' : 'naive'}`] = failure(data)
      keep(this, failureKey(surface, this.h1.cause, contained), data)
    }
  }
  merge(this, 'failures', { [this.h1.cause]: this.h1.found })
})

Then('the prompt continues with its text and no added context', function () {
  for (const key of ['mod-contained', 'mod-naive']) {
    const found = this.h1.found[key]
    assert.ok(found.continued && found.typedPromptKept && found.addedContext === 0, `${key}: ${JSON.stringify(found)}`)
  }
})

Then("no error text and no warning reach the agent's context, and no user input is lost", function () {
  for (const key of ['mod-contained', 'mod-naive']) assert.ok(!this.h1.found[key].agentQuotedAnError, `${key}: the agent quoted an error: ${this.h1.found[key].asked}`)
})

Then(/^the user sees a warning outside the transcript saying what was skipped and why \(in a stream, a ui_status message\)$/, function () {
  assert.ok(this.h1.found['mod-contained'].warningToUser.length > 0, 'the contained mod showed the user no warning')
})

Then('the report counts the code each surface needed to contain this failure', function () {
  const code = counts()
  merge(this, 'containment', { modLines: code.mod.containmentLines, hookLines: code.hook.containmentLines })
  assert.ok(code.mod.containmentLines > 0 && code.hook.containmentLines > 0, 'no containment code is tagged')
})

// ---- 793d17e1b: the System's mod changes nothing it must not ----------------------------------------------------------------------------

const EVENT_LIST = /events: (.+)$/

Given('the loaded mod and an agent that runs engram query and the learn skill in a turn', function () {
  prefetch(['system:with', 'system:without'])
})

When(/^the debug log lists the events its handlers are registered for .* and Claude Code fires the tool events of those commands$/, async function () {
  const withMod = await tour('system:with')
  const without = await tour('system:without')
  const context = await getRunContext()
  const modDir = path.join(path.dirname(FILES.mod.code[0]), '..', '..', '..', '..', '..', 'agent-instructions', 'mods', 'engram-memory')
  const test = execFileSync(context.claudePath, ['plugin', 'test', modDir], { encoding: 'utf8', env: { PATH: process.env.PATH, HOME: fs.mkdtempSync(path.join(context.dir, 'plugin-test-')) } })
  this.h1 = { withMod, without, test, modDir }
  keep(this, 'system:with', withMod)
  keep(this, 'system:without', without)
})

Then("no handler denies, drops, or answers an event in place of continuing it, and the handler of turn.step passes the step's input, chunks and result on unchanged", function () {
  const loaded = this.h1.withMod.loaded.find((line) => line.includes('hooks module engram-memory'))
  assert.ok(loaded, 'the debug log does not say that the System\'s mod loaded')
  const events = EVENT_LIST.exec(loaded)[1].split(',')
  const source = fs.readFileSync(path.join(this.h1.modDir, 'hooks', 'register.js'), 'utf8')
  const refusals = source.match(/\{\s*(deny|drop|consumed|isOffered|skip|refuse|decision)\s*:/g) ?? []
  const passes = /on\('turn\.step', async function\* \(\$, e, next\) \{\n\s*const result = yield\* next\(e\)[\s\S]*?return result\n\s*\}\)/.test(source)
  const summary = /(\d+) pass\n\s*(\d+) fail/.exec(this.h1.test)
  merge(this, 'restraint', { events, refusalsInSource: refusals, turnStepPassesThrough: passes, contractTests: summary ? { pass: Number(summary[1]), fail: Number(summary[2]) } : null })
  assert.deepEqual(refusals, [], `a handler may deny, drop or answer: ${refusals.join(', ')}`)
  assert.ok(passes, 'the handler of turn.step does not pass the step on unchanged')
  assert.equal(summary?.[2], '0', 'the mod\'s contract tests fail')
})

Then("each memory command's input, permission decision and result are unchanged", function () {
  const byName = (data) => Object.fromEntries(data.calls.map((call) => [call.name, call]))
  const resultOf = (data, call) => data.results.find((result) => result.id === call.id)
  const withMod = byName(this.h1.withMod)
  const without = byName(this.h1.without)
  const compared = {}
  for (const name of ['Bash', 'Skill']) {
    assert.ok(withMod[name] && without[name], `a ${name} call is missing from one of the runs`)
    const a = resultOf(this.h1.withMod, withMod[name])
    const b = resultOf(this.h1.without, without[name])
    compared[name] = { sameInput: name === 'Skill' ? withMod[name].input === without[name].input : /engram query --phrase/.test(withMod[name].input) && /engram query --phrase/.test(without[name].input), isError: [a.isError, b.isError], sameResult: name === 'Skill' ? a.text === b.text : a.text.slice(0, 60) === b.text.slice(0, 60) }
    assert.equal(a.isError, b.isError, `the permission or the outcome of ${name} differs`)
  }
  merge(this, 'restraintCompared', compared)
})

Then('the only effect of the mod on the agent\'s context is the blocks it added', async function () {
  const { rowKindDelta } = await import('../h1/capabilities.js')
  const added = rowKindDelta(this.h1.withMod.rows, this.h1.without.rows)
  merge(this, 'restraintContext', { addedRowKinds: added })
  const other = Object.keys(added).filter((kind) => kind !== 'attachment:hook_additional_context')
  assert.deepEqual(other, [], `the mod added rows that are not blocks: ${other.join(', ')}`)
})

// ---- 793d17f1a: the counts --------------------------------------------------------------------------------------------------------------------

Given(/^the probe pairs of 793d17a1a, 793d17c1a to 793d17c1c and 793d17e1a, each built once as a mod and once as a settings hook, calling the same stand-in command$/, function () {
  for (const file of [...FILES.mod.code, ...FILES.hook.code, FILES.hook.settings]) assert.ok(fs.existsSync(file), `${file} is not built`)
})

When('the artifacts each surface needs are counted over the pairs', function () {
  this.h1 = { counts: counts() }
  merge(this, 'counts', this.h1.counts)
})

Then("the report gives, for each surface, the files it deploys outside the Go program, its registered handlers or hook entries, its edits to files the user owns, and its glue lines", function () {
  for (const surface of SURFACES) {
    const c = this.h1.counts[surface]
    for (const key of ['filesDeployed', 'registrations', 'editsToUserFiles', 'glueLines']) assert.ok(Number.isInteger(c[key]), `${surface} has no ${key}`)
  }
})

Then('the report gives how many capabilities of 793d17a1a each surface achieved', function () {
  let capabilities
  try {
    capabilities = readObservations('793d17a1a').capabilities
  } catch {
    throw new Error('793d17a1a has no recorded result: run it first (targ acceptance 793d17)')
  }
  const tally = Object.fromEntries(SURFACES.map((surface) => [surface, Object.values(capabilities).filter((entry) => entry[surface].achieved).length]))
  merge(this, 'achieved', { ...tally, of: Object.keys(capabilities).length })
  assert.equal(Object.keys(capabilities).length, 9, 'the result of 793d17a1a does not hold the nine capabilities')
})

Then("the mod's surface is smaller when its total of the four counts is lower and none of the four is higher than the settings hooks'", function () {
  const { mod, hook } = this.h1.counts
  const four = ['filesDeployed', 'registrations', 'editsToUserFiles', 'glueLines']
  const smaller = mod.total < hook.total && four.every((key) => mod[key] <= hook[key])
  merge(this, 'smaller', { smaller, modTotal: mod.total, hookTotal: hook.total, higherCounts: four.filter((key) => mod[key] > hook[key]) })
})
