import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { attribution, backgroundWork, capabilityFor, CAPABILITIES, failure, longWork, presence, rowKindDelta } from '../h1/capabilities.js'
import { counts, scan } from '../h1/count.js'
import { buildReport } from '../h1/report.js'
import { compact } from '../h1/results.js'
import { H1_DIR, installHooks, installMod, MOD_DIRS, STAND_IN } from '../h1/surfaces.js'
import { PRESENCE_TOKEN, TOKENS } from '../h1/constants.js'
import { verdictOf } from '../h1/verdict.js'

const tempHome = () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'h1-test-'))
  fs.mkdirSync(path.join(home, '.claude'), { recursive: true })
  fs.writeFileSync(path.join(home, '.claude', 'settings.json'), JSON.stringify({ permissions: { allow: ['Read'] } }))
  return { home }
}

const attachment = (i, token, hookEvent = 'UserPromptSubmit') => ({ i, type: 'attachment', attachment: { type: 'hook_additional_context', hookEvent, hookName: hookEvent, content: token }, text: '' })
const user = (i, text) => ({ i, type: 'user', role: 'user', text })

// ---- the surfaces are installed the way engram update installs the System ---------------------------------------------

test('the mod is staged in the engram-owned root and linked into the skills directory, with no settings edit', () => {
  const scenario = tempHome()
  const before = fs.readFileSync(path.join(scenario.home, '.claude', 'settings.json'), 'utf8')
  const { links } = installMod(scenario)
  assert.equal(links.length, Object.keys(MOD_DIRS).length)
  for (const link of links) {
    assert.ok(fs.lstatSync(link).isSymbolicLink())
    assert.ok(fs.readlinkSync(link).includes(`${path.sep}.claude${path.sep}engram${path.sep}skills${path.sep}`))
    assert.ok(fs.existsSync(path.join(link, 'hooks', 'hooks.json')))
  }
  assert.equal(fs.readFileSync(path.join(scenario.home, '.claude', 'settings.json'), 'utf8'), before)
  fs.rmSync(scenario.home, { recursive: true, force: true })
})

test('the settings hooks are merged into the user settings and keep what the file held', () => {
  const scenario = tempHome()
  const { dir } = installHooks(scenario, { timeoutSeconds: 10, asyncStop: true })
  const settings = JSON.parse(fs.readFileSync(path.join(scenario.home, '.claude', 'settings.json'), 'utf8'))
  assert.deepEqual(settings.permissions.allow, ['Read'])
  const extra = (event) => settings.hooks[event][0].hooks.find((hook) => hook.command.endsWith('extra.js'))
  assert.equal(extra('UserPromptSubmit').timeout, 10)
  assert.equal(extra('Stop').async, true)
  assert.ok(settings.hooks.SessionStart[0].hooks[0].command.includes(dir))
  assert.ok(fs.existsSync(path.join(dir, 'recorder.js')))
  fs.rmSync(scenario.home, { recursive: true, force: true })
})

test('the stand-in command prints its delivery after the delay, or something that is not JSON', () => {
  assert.deepEqual(JSON.parse(execFileSync('node', [STAND_IN, '0', 'TEXT'], { encoding: 'utf8' })), { delivery: 'TEXT' })
  assert.throws(() => JSON.parse(execFileSync('node', [STAND_IN, '0', 'TEXT', 'unparsable'], { encoding: 'utf8' })))
})

// ---- the counts ----------------------------------------------------------------------------------------------------------------

test('a handler of a mod is counted as a handler and the code around it as glue; an excluded handler is counted as neither', () => {
  const source = [
    'import x from "x"',
    '',
    'export function register(on) {',
    "  on('kept', async ($, e, next) => {",
    '    const a = 1 // not-h1',
    '    return next(e)',
    '  })',
    "  on('left', async ($, e, next) => {",
    '    return next(e)',
    '  })',
    '}',
  ].join('\n')
  const result = scan(source, { open: /^ {2}on\('([^']+)'/, close: /^ {2}\}\)/, excluded: ['left'] })
  assert.deepEqual([...result.handlers], [['kept', 3]])
  assert.equal(result.glue, 3)
  assert.equal(result.excludedLines, 3)
})

test('the tagged lines are counted apart: containment, the model call; instrumentation and stimulus are not code of the surface', () => {
  const source = ['const handlers = {', '  Event() {', '    try {', '      work()', '    } catch (error) { // contain', '      throw error // contain', '    }', '    log() // instrumentation', '  },', '}'].join('\n')
  const result = scan(source, { open: /^ {2}([A-Za-z]+)\(\) \{$/, close: /^ {2}\},$/, excluded: [] })
  assert.equal(result.containment, 2)
  assert.equal(result.handlers.get('Event'), 7)
  assert.equal(result.glue, 2)
})

test('the counts over the probe files hold the four numbers of each surface and a total that is their sum', () => {
  const found = counts()
  for (const surface of ['mod', 'hook']) {
    const c = found[surface]
    assert.equal(c.total, c.filesDeployed + c.registrations + c.editsToUserFiles + c.glueLines)
    assert.ok(c.containmentLines > 0 && c.modelCallLines > 0, `${surface}: the tagged lines are found`)
  }
  assert.equal(found.hook.editsToUserFiles, 1)
  assert.equal(found.mod.editsToUserFiles, 0)
  // 11 hook entries are registered in settings.hooks.json and the recorder's Stop is left out
  assert.equal(found.hook.registrations, 10)
  assert.ok(!found.mod.perFile[0].handlers['turn.complete'], 'the recorder turn.complete is left out')
  assert.ok(found.mod.perFile[1].handlers['turn.complete'], 'the extra probe turn.complete is counted')
})

// ---- the capability readers: each can fail ---------------------------------------------------------------------------------

const mainTour = (overrides = {}) => ({
  surface: 'hook',
  replies: [TOKENS.prompt, TOKENS.tool, TOKENS.sub],
  compact: { ok: true },
  events: [
    { kind: 'SessionStart', data: { transcript_path: '/t.jsonl', source: 'startup' } },
    { kind: 'dispatch', data: { prompt: 'a SUB-CODE task' } },
    { kind: 'SubagentStop', data: { agent_transcript_path: '/s.jsonl', last_assistant_message: TOKENS.sub, assistantRowHoldsLastMessageAtStop: false } },
    { kind: 'PostCompact', data: { trigger: 'manual' } },
  ],
  main: { path: '/t.jsonl', rows: [user(2, 'x'), attachment(3, TOKENS.prompt), attachment(4, TOKENS.tool, 'PostToolUse')] },
  subagents: [{ rows: [attachment(1, TOKENS.sub, 'SubagentStart'), { i: 2, type: 'assistant', text: TOKENS.sub }] }],
  ...overrides,
})

test('the nine capabilities are achieved by a tour that shows them, and each is not achieved by one that does not', () => {
  assert.equal(CAPABILITIES.length, 9)
  for (const key of ['promptContext', 'toolContext', 'subagentContext', 'dispatch', 'subagentStop', 'transcriptPath', 'compaction']) {
    const capability = CAPABILITIES.find((c) => c.key === key)
    assert.equal(capability.evaluate(mainTour(), 'hook').achieved, true, key)
  }
  const bare = mainTour({ replies: ['NONE', 'NONE', 'NONE'], events: [], compact: { ok: false }, main: { path: '/t.jsonl', rows: [] }, subagents: [] })
  for (const key of ['promptContext', 'toolContext', 'subagentContext', 'dispatch', 'subagentStop', 'transcriptPath', 'compaction']) {
    assert.equal(CAPABILITIES.find((c) => c.key === key).evaluate(bare, 'hook').achieved, false, key)
  }
})

test('a block of the subagent that is also in the main transcript is not "for that subagent only"', () => {
  const leaking = mainTour({ main: { path: '/t.jsonl', rows: [attachment(3, TOKENS.sub, 'SubagentStart')] } })
  assert.equal(capabilityFor('text is added at a subagent\'s start, for that subagent only').evaluate(leaking, 'hook').achieved, false)
})

test('the capability of prompts from every origin needs all four, and notes whether the surface tells the origin', () => {
  const tour = (origin) => ({
    task: { prompts: [{ text: 'sleep 3', origin }, { text: '<task-notification>', origin }], blocks: 2 },
    cron: { prompts: [{ text: 'scheduled-ok: say', origin }], blocks: 1 },
    peer: { prompts: [{ text: '<cross-session-message>', origin }], blocks: 1 },
  })
  const origins = capabilityFor('a prompt from the composer, from a task notification, from a scheduled trigger and from another session is observed before the agent responds')
  assert.equal(origins.evaluate(tour(null)).achieved, true)
  assert.equal(origins.evaluate(tour(null)).detail.originTold, false)
  assert.equal(origins.evaluate({ ...tour('sdk'), cron: { prompts: [], blocks: 0 } }).achieved, false)
})

test('the delivered block is listed after the delivery, not after the compaction, and again after the resume', () => {
  const tour = (listed) => ({ events: listed.map((n) => ({ kind: 'presence', data: { listed: n } })), main: { rows: [attachment(1, PRESENCE_TOKEN)] } })
  assert.equal(presence(tour([0, 1, 0, 1, 1])).holds, true)
  assert.equal(presence(tour([0, 1, 1, 1, 1])).holds, false)
  assert.equal(presence(tour([0, 1, 0, 1, 0])).holds, false)
})

test('attribution tells a block by its own row and its hook event, and reports rows a surface adds beyond blocks', () => {
  const tour = mainTour()
  const control = mainTour({ main: { path: '/t.jsonl', rows: [user(2, 'x')] }, subagents: [] })
  const found = attribution(tour, control)
  assert.equal(found.ownRows, true)
  assert.equal(found.channelTold, true)
  const noisy = { ...tour, main: { rows: [...tour.main.rows, { i: 9, type: 'system', subtype: 'stop_hook_summary', text: '' }] } }
  assert.equal(attribution(noisy, control).extraRowKinds['system:stop_hook_summary'], 1)
  assert.deepEqual(rowKindDelta([user(1, 'a')], [user(1, 'a')]), {})
})

test('a long job is delivered or cancelled, and the next prompt is delayed or not', () => {
  const tour = (rows) => ({ events: [{ kind: 'standIn', data: { ownMs: 1, totalMs: 25000 } }], main: { rows }, turnWallS: 27 })
  assert.equal(longWork(tour([attachment(1, 'STAND-IN-DELIVERY')]), 'mod').delivered, true)
  assert.equal(longWork(tour([{ i: 1, type: 'attachment', attachment: { type: 'hook_cancelled', content: '' }, text: '' }]), 'hook').cancelled, true)
  const wall = (second) => [{ s: 2, reply: 'a' }, { s: second, reply: 'b' }, { s: 3, reply: 'RESULT' }]
  const run = (second) => ({ wall: wall(second), events: [{ kind: 'background.start', t: 0 }, { kind: 'background.done', t: 20000 }, { kind: 'delivered', t: 21000, data: { tool_use_id: 't' } }], main: { rows: [attachment(1, 'RESULT', 'PostToolUse')] } })
  assert.equal(backgroundWork(run(3)).notDelayed, true)
  assert.equal(backgroundWork(run(22)).notDelayed, false)
})

test('a failure is contained when the prompt continues, no block is added, the user is warned and the agent quotes no error', () => {
  const tour = (overrides) => ({ typed: 'Reply', reply: 'ok', isError: false, asked: 'NONE', statuses: ['Automatic memory was skipped (start)'], informational: [], main: { rows: [user(1, 'Reply')] }, ...overrides })
  const good = failure(tour({}))
  assert.equal(good.continued && good.typedPromptKept && good.addedContext === 0 && !good.agentQuotedAnError, true)
  assert.equal(failure(tour({ asked: 'Failed with non-blocking status code' })).agentQuotedAnError, true)
  assert.equal(failure(tour({ statuses: [] })).warningToUser.length, 0)
  assert.deepEqual(failure(tour({ main: { rows: [user(1, 'Reply'), { i: 2, type: 'attachment', attachment: { type: 'hook_non_blocking_error', content: '' }, text: '' }] } })).noticeRowsInTranscript, ['hook_non_blocking_error'])
})

// ---- the verdict rule ----------------------------------------------------------------------------------------------------------

function observations({ hookAchieves = 9, modAchieves = 9 } = {}) {
  const capabilities = Object.fromEntries(CAPABILITIES.map((c, index) => [c.key, { mod: { achieved: index < modAchieves, detail: {} }, hook: { achieved: index < hookAchieves, detail: {} } }]))
  capabilities.origins.mod.detail = { promptsSeen: { task: 2, cron: 2, peer: 2 }, blocksAdded: { task: 2, cron: 2, peer: 2 } }
  const failureOk = { continued: true, typedPromptKept: true, addedContext: 0, agentQuotedAnError: false }
  return {
    a1a: { capabilities },
    a1b: { attribution: { mod: { ownRows: true, channelTold: true, promptsUnchanged: true, toolResultUnchanged: true } } },
    b1a: { screen: { mod: { onScreen: true, inTranscriptFile: false, modelQuotedIt: false } } },
    c1a: { modelCall: { mod: { answered: true, haikuFamily: true } } },
    c1b: { longWork: { mod: { delivered: true, ownMs: 1 } } },
    c1c: { backgroundWork: { mod: { notDelayed: true, reachedAgent: true } } },
    d1a: { load: { mod: { sessionStarts: [1, 2, 3, 4], promptsHandled: 4, validated: [{ ok: true }] }, hook: { sessionStarts: [1, 2, 3, 4] } } },
    e1a: { failures: Object.fromEntries([1, 2, 3, 4].map((n) => [n, { 'mod-contained': failureOk, 'mod-naive': failureOk }])) },
    e1b: { restraint: { refusalsInSource: [], turnStepPassesThrough: true, contractTests: { fail: 0 } } },
  }
}

const surface = (numbers) => ({ filesDeployed: 1, registrations: 1, editsToUserFiles: 0, glueLines: 1, ...numbers })
const withTotals = (mod, hook) => ({ mod: { ...surface(mod), total: Object.values(surface(mod)).reduce((a, b) => a + b, 0) }, hook: { ...surface(hook), total: Object.values(surface(hook)).reduce((a, b) => a + b, 0) } })

test('H1 is confirmed only when the mod is smaller on the total and not higher on any count', () => {
  assert.equal(verdictOf(observations(), withTotals({ glueLines: 1 }, { glueLines: 5 })).verdict, 'confirmed')
  const higher = verdictOf(observations({ hookAchieves: 0 }), withTotals({ filesDeployed: 3, glueLines: 1 }, { filesDeployed: 1, glueLines: 9 }))
  assert.equal(higher.conditions.smaller.holds, false)
  assert.equal(higher.verdict, 'inconclusive')
})

test('H1 is disproved when the settings hooks reach the same capabilities with a lower total or with any lower count', () => {
  const total = verdictOf(observations(), withTotals({ glueLines: 9 }, { glueLines: 1 }))
  assert.equal(total.verdict, 'disproved')
  assert.equal(total.disproof.hooksSmaller.holds, true)
  const oneCount = verdictOf(observations(), withTotals({ glueLines: 1, editsToUserFiles: 0 }, { glueLines: 9, editsToUserFiles: 0, filesDeployed: 0 }))
  assert.equal(oneCount.verdict, 'disproved')
})

test('H1 is disproved when the hooks achieve a capability the mod does not, or when neither does', () => {
  assert.equal(verdictOf({ ...observations(), a1a: { capabilities: { ...observations().a1a.capabilities, step: { mod: { achieved: false }, hook: { achieved: true } } } } }, withTotals({ glueLines: 1 }, { glueLines: 5 })).verdict, 'disproved')
  assert.equal(verdictOf(observations({ modAchieves: 8, hookAchieves: 8 }), withTotals({ glueLines: 1 }, { glueLines: 5 })).disproof.hookOnlyOrNeither.holds, true)
})

test('the report holds the labelled tables and a verdict line, and does not need a result for every scenario', () => {
  const { text } = buildReport({}, counts())
  assert.match(text, /\| Capability \| Mod achieved \| Hook achieved \| Mod handler lines \(lines\)/)
  assert.match(text, /\| Surface \| Files deployed \(files \+ links\)/)
  assert.match(text, /Verdict: (CONFIRMED|DISPROVED|INCONCLUSIVE)/)
})

test('a compacted tour drops the harness bookkeeping and the paths', () => {
  const tour = { events: [{ kind: 'x', t: 1, data: { transcript_path: '/a', session_id: 's', cwd: '/c', keep: 1 } }], main: { path: '/a', rows: [{ type: 'queue-operation' }, { type: 'user' }] }, debug: ['[DEBUG] hooks module m loaded\nnoise'], subagents: [] }
  const small = compact(tour)
  assert.equal(small.main.rows.length, 1)
  assert.equal(small.events[0].data.keep, 1)
  assert.equal(small.events[0].data.session_id, undefined)
  assert.deepEqual(small.debug, ['[DEBUG] hooks module m loaded'])
})

test('the H1 directory holds the probes the counts read', () => {
  for (const file of ['stand-in.js', 'hook/recorder.js', 'hook/extra.js', 'hook/settings.hooks.json', 'mod/hooks/register.js', 'mod/hooks/hooks.json', 'mod/.claude-plugin/plugin.json']) assert.ok(fs.existsSync(path.join(H1_DIR, file)), file)
})
