// SP17 (793e8a, 793d1a): which user-facing surface of a mod is drawn on an interactive screen that also has a
// configured statusLine, and which of them stays out of the transcript and the model's context. The session runs under
// tmux, the screen is captured, and the transcript file and the model's own answer are the ground truth for "in
// context".
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

import { MARKERS, RECORDER_DIR_OF, readTranscriptFiles, screenHas } from './sp17-lib.js'
import { JSON_INDENT, MS_PER_SECOND } from '../lib/units.js'

export const claim =
  'With a statusLine command configured, which of $.ui.status, $.ui.toast, the AbovePrompt band and a turn.complete ' +
  '{ text } is drawn on an interactive screen, and which of them stays out of the transcript and the model context'

const SURFACES = path.join(RECORDER_DIR_OF, 'surfaces')
const STATUSLINE_TEXT = 'SP17-OWNER-STATUSLINE'
const READY_SECONDS = 90
const TURN_SECONDS = 120
const SETTLE_SECONDS = 2
const READY_MS = READY_SECONDS * MS_PER_SECOND
const TURN_MS = TURN_SECONDS * MS_PER_SECOND
const POLL_MS = 500
const SETTLE_MS = SETTLE_SECONDS * MS_PER_SECOND
const SOCKET_RANGE = 1000000
const SESSION = 'sp17'
const QUESTION =
  'List verbatim every line or text that starts with PROBE- anywhere in this conversation, including system ' +
  'reminders, hook output and your own earlier replies. If there is none, say exactly NONE-SEEN.'

// the statusLine commands and screens tried: a plain line, and a multi-line one with colours, as a HUD prints it
const VARIANTS = [
  { name: 'one-line statusLine, 220x60', lines: 1, columns: 220, rows: 60 },
  { name: 'three-line coloured statusLine, 160x40', lines: 3, columns: 160, rows: 40, colour: true },
  { name: 'three-line coloured statusLine, 120x24', lines: 3, columns: 120, rows: 24, colour: true },
  { name: 'one-line statusLine, turn text returned, 220x60', lines: 1, columns: 220, rows: 60, turnText: true },
]

function sleep(milliseconds) {
  return new Promise((resolve) => {
    setTimeout(resolve, milliseconds)
  })
}

// tmux runs the tmux client on the probe's own socket, so that the server it starts carries the scenario's environment
// (the credentials among it) and nothing of the calling session.
function tmux(scenario, socket, args) {
  return execFileSync('tmux', ['-L', socket, ...args], {
    env: scenario.env({ withCredentials: true }),
    encoding: 'utf8',
  })
}

const screen = (scenario, socket) => tmux(scenario, socket, ['capture-pane', '-p', '-t', SESSION, '-S', '-200'])

async function until(label, milliseconds, check) {
  const deadline = Date.now() + milliseconds
  while (Date.now() < deadline) {
    const value = check()
    if (value) return value
    await sleep(POLL_MS)
  }
  throw new Error(`timed out waiting for ${label}`)
}

function statusLineScript(variant) {
  const paint = (text, code) => (variant.colour ? `\\033[${code}m${text}\\033[0m` : text)
  const rows = [
    STATUSLINE_TEXT,
    paint('Tools: Read x3 | Bash x1 | Agents: none | Todos: 2/5 done', '36'),
    paint('Context: [####------] 41%% | 5h: 23%% | 7d: 8%% | git:(main*)', '33'),
  ].slice(0, variant.lines)
  return `#!/bin/sh\ncat >/dev/null\n${rows.map((row) => `printf '${row}\\n'`).join('\n')}\n`
}

function configure(scenario, variant) {
  const script = path.join(scenario.dir, 'statusline.sh')
  fs.writeFileSync(script, statusLineScript(variant), { mode: 0o755 })
  const settingsFile = path.join(scenario.home, '.claude', 'settings.json')
  const settings = JSON.parse(fs.readFileSync(settingsFile, 'utf8'))
  settings.statusLine = { type: 'command', command: script }
  fs.writeFileSync(settingsFile, JSON.stringify(settings, null, JSON_INDENT))
  // no first-run screens: the onboarding done, the workspace trusted (under both spellings of its path)
  const trusted = { hasTrustDialogAccepted: true, hasCompletedProjectOnboarding: true }
  fs.writeFileSync(
    path.join(scenario.home, '.claude.json'),
    JSON.stringify({
      hasCompletedOnboarding: true,
      theme: 'dark',
      projects: Object.fromEntries(
        [scenario.workspace, fs.realpathSync(scenario.workspace)].map((dir) => [dir, trusted]),
      ),
    }),
  )
  if (variant.turnText) scenario.extraEnv.SP17_TURN_TEXT = '1'
  scenario.extraEnv.SP17_OUT = path.join(scenario.dir, 'sp17-api.json')
}

function type(scenario, socket, text) {
  tmux(scenario, socket, ['send-keys', '-t', SESSION, '-l', text])
  tmux(scenario, socket, ['send-keys', '-t', SESSION, 'Enter'])
}

function visible(shot) {
  return Object.fromEntries(Object.entries(MARKERS).map(([name, marker]) => [name, screenHas(shot, marker)]))
}

// rowsOf lists, per marker, the transcript rows that hold it (the row of the question does not count).
function rowsOf(transcript) {
  const rows = transcript
    .split('\n')
    .filter(Boolean)
    .map((line) => JSON.parse(line))
    .filter((row) => !JSON.stringify(row).includes('List verbatim every line'))
  return Object.fromEntries(
    Object.entries(MARKERS).map(([name, marker]) => [
      name,
      rows
        .filter((row) => JSON.stringify(row).includes(marker))
        .map((row) => [row.type, row.subtype, row.attachment?.type].filter(Boolean).join('/')),
    ]),
  )
}

function lastAnswer(transcript) {
  const rows = transcript
    .split('\n')
    .filter(Boolean)
    .map((line) => JSON.parse(line))
    .filter((row) => row.type === 'assistant')
  return (rows.at(-1)?.message?.content ?? []).filter((block) => block.type === 'text').map((block) => block.text)
}

async function drive(harness, variant) {
  const scenario = harness.scenario()
  configure(scenario, variant)
  const socket = `sp17-${process.pid}-${Math.floor(Math.random() * SOCKET_RANGE)}`
  const evidence = { variant: variant.name, shots: {} }
  const command = `${harness.context.claudePath} --model sonnet --plugin-dir ${SURFACES}`
  tmux(scenario, socket, [
    'new-session',
    '-d',
    '-s',
    SESSION,
    '-x',
    String(variant.columns),
    '-y',
    String(variant.rows),
    '-c',
    scenario.workspace,
    command,
  ])
  try {
    await until('the statusLine on the first screen', READY_MS, () =>
      screenHas(screen(scenario, socket), STATUSLINE_TEXT),
    )
    type(scenario, socket, 'Reply with exactly the word: ok')
    await until('the first reply', TURN_MS, () => readTranscriptFiles(scenario.home).includes('"text":"ok'))
    await sleep(SETTLE_MS)
    evidence.shots.afterTurn1 = screen(scenario, socket)
    evidence.visibleAfterTurn1 = visible(evidence.shots.afterTurn1)
    type(scenario, socket, QUESTION)
    await until('the second reply', TURN_MS, () => {
      const transcript = readTranscriptFiles(scenario.home)
      const answer = lastAnswer(transcript).join(' ')
      return transcript.includes('List verbatim every line') && /NONE-SEEN|PROBE-/.test(answer)
    })
    await sleep(SETTLE_MS)
    evidence.shots.afterTurn2 = screen(scenario, socket)
    evidence.visibleAfterTurn2 = visible(evidence.shots.afterTurn2)
  } catch (error) {
    evidence.error = error.message
    evidence.shots.lastScreen = screen(scenario, socket)
  } finally {
    try {
      tmux(scenario, socket, ['kill-server'])
    } catch {
      // the server is gone already
    }
  }
  await sleep(SETTLE_MS)
  const transcript = readTranscriptFiles(scenario.home)
  evidence.transcriptRowsHoldingMarker = rowsOf(transcript)
  evidence.modelAnswerToTheQuestion = lastAnswer(transcript)
  const api = path.join(scenario.dir, 'sp17-api.json')
  evidence.apiViewAtLastTurnEnd = fs.existsSync(api) ? JSON.parse(fs.readFileSync(api, 'utf8')) : null
  return evidence
}

export async function run(harness) {
  const variants = []
  const only = process.env.SP17_ONLY?.split(',').map(Number)
  for (const [index, variant] of VARIANTS.entries()) {
    if (!only || only.includes(index)) variants.push(await drive(harness, variant))
  }
  return { variants }
}
