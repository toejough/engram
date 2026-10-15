// The per-scenario World (793e2b): its own HOME, workspace, vault and TMPDIR, an explicit environment, notes seeded
// through the real write path, and the session. Nothing under the real ~/.claude, ~/.pi or the real vault is touched.
import { execFile } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { promisify } from 'node:util'

import { REPO_ROOT } from './extract.js'
import { seedOffer } from './offers.js'
import { baseEnv } from './run-context.js'
import { Session } from './session.js'
import { faultCalls, faultEnv } from './fault.js'
import { snapshotTree } from './tree.js'
import { JSON_INDENT } from './units.js'
import { sentinelFor } from './vault.js'

const DIR_MODE = 0o755
const SWEEP_SPEC = { non_persistent_prefixes: [], non_persistent_path_prefixes: [], ancestor_claude_dirs: false }
const run = promisify(execFile)
// The agent may run engram and sleep, and echo only the words the scenarios give it: `Bash(echo:*)` would let it echo
// the environment of the session, and the environment holds the credentials (793b316).
const ECHOES = [
  'ok',
  'tagging a release',
  'tagging a release again',
  "'TestSync failed: FIXTURE_DIR is unset'",
  '"TestSync failed: FIXTURE_DIR is unset"',
]
const ALLOWED_TOOLS = [
  'Bash(engram:*)',
  ...ECHOES.map((words) => `Bash(echo ${words})`),
  'Bash(sleep:*)',
  'Read',
  'Glob',
  'Grep',
  'Agent',
  'Skill',
  'CronCreate',
]
// a turn bound for the sessions of the scenarios: a skill such as please can run long, and no scenario needs more
const MAX_TURNS = 8

export class Scenario {
  constructor(context) {
    this.context = context
    this.log = []
    this.init()
  }

  init() {
    this.dir = fs.mkdtempSync(path.join(this.context.dir, 'scenario-'))
    this.home = path.join(this.dir, 'home')
    // the workspace sits under the home, as a real project does, so the ancestor .claude and .pi directories of
    // `engram ingest --auto` are the scenario's own
    this.workspace = path.join(this.home, 'work')
    this.notes = new Map()
    this.offers = new Map()
    // the ended sessions written as inputs (793e8g): the transcript, its copy and its time
    this.ended = []
    this.faultSeconds = 0
    this.vaultHidden = false
    // the seam of the System's work (793b281): the mod records when its queue is empty, so that a test can wait for it
    this.seamFile = path.join(this.dir, 'seam.log')
    this.extraEnv = { ENGRAM_AUTO_SEAM_LOG: this.seamFile }
    this.sessions = []
    this.session = null
    for (const dir of [this.home, path.join(this.home, 'tmp'), this.workspace, path.join(this.home, '.claude')]) {
      fs.mkdirSync(dir, { recursive: true })
    }
    this.vault = path.join(this.home, '.local', 'share', 'engram', 'vault')
    // the sweep spec of the workspace: a scenario lives under a throwaway root, which `--auto` skips by default (SP15).
    // The ancestor .claude directory holds the installed skills, about sixty seconds of embedding that every scenario's
    // fresh home would pay at the System's first indexing (793e8d); the sessions, which the scenarios look for, stay.
    fs.mkdirSync(path.join(this.workspace, '.engram'), { recursive: true })
    fs.writeFileSync(path.join(this.workspace, '.engram', 'sweep.json'), `${JSON.stringify(SWEEP_SPEC)}\n`)
    // the settings the World starts from: the update must add none (793a5j)
    this.baselineSettings = { permissions: { allow: ALLOWED_TOOLS } }
    fs.writeFileSync(
      path.join(this.home, '.claude', 'settings.json'),
      `${JSON.stringify(this.baselineSettings, null, JSON_INDENT)}\n`,
    )
    this.treeBefore = null
    this.updateOutput = ''
  }

  // setup runs a Given action and records it, so that a retried precondition can rebuild this World (793b266).
  async setup(action) {
    this.log.push(action)
    await action(this)
  }

  // rebuild discards the home, the vault and the sessions and replays the recorded setup into a fresh World.
  async rebuild() {
    const usage = this.usageSoFar ?? []
    usage.push(...this.sessions.flatMap((session) => session.turns))
    this.usageSoFar = usage
    await this.#discard()
    this.init()
    for (const action of this.log) await action(this)
  }

  // env is the whole environment of a process started for this scenario, credentials included for sessions only.
  env({ withCredentials = false } = {}) {
    const env = {
      ...baseEnv({ home: this.home, cacheDir: this.context.cacheDir, binDir: this.context.binDir }),
      ...this.extraEnv,
    }
    return withCredentials ? { ...env, ...this.context.creds.env } : env
  }

  async engram(args, { cwd = this.workspace, extraEnv = {} } = {}) {
    const { stdout } = await run(this.context.engramPath, args, { cwd, env: { ...this.env(), ...extraEnv } })
    return stdout
  }

  // seedNote writes a fact note through `engram learn` with a unique sentinel in its body; the sentinel is what a
  // delivery or a retrieval is recognised by.
  async seedNote(label, situation, says) {
    const sentinel = sentinelFor(label)
    const object = says ? `${says} (${sentinel})` : sentinel
    await this.engram([
      'learn',
      'fact',
      '--slug',
      `note-${label.toLowerCase()}`,
      '--position',
      'top',
      '--source',
      'acceptance harness',
      '--situation',
      situation,
      '--subject',
      situation,
      '--predicate',
      'is recorded as',
      '--object',
      object,
    ])
    this.notes.set(label, { label, situation, sentinel, object })
    return this.notes.get(label)
  }

  // seedOffer puts a pending offer in the vault (793e2i step 5): one that states the claim of note `covers` (and, with
  // `extend`, one more), or a claim no note states.
  async seedOffer(label, coversLabel, extend) {
    const covers = coversLabel ? this.notes.get(coversLabel) : null
    if (coversLabel && !covers) throw new Error(`no note "${coversLabel}" was seeded before the offer "${label}"`)
    // a near offer: the note's claim and one more it does not state
    const claim = covers && extend ? { ...covers, object: `${covers.object}; ${extend}` } : covers
    this.offers.set(label, await seedOffer(this, label, claim))
    return this.offers.get(label)
  }

  // injectFault sets the fault seam of the System's work for every process of the World (793b418); `delay` takes the
  // seconds the background work waits.
  injectFault(mode, seconds = 0) {
    this.extraEnv = { ...this.extraEnv, ...faultEnv({ dir: this.dir, mode, seconds }) }
    this.faultSeconds = seconds
  }

  faultCalls() {
    return faultCalls(this.extraEnv)
  }

  // hideVault makes the vault directory unreadable, a real way for it to be unavailable (793e8d).
  hideVault() {
    fs.mkdirSync(this.vault, { recursive: true })
    fs.chmodSync(this.vault, 0o000)
    this.vaultHidden = true
  }

  showVault() {
    if (this.vaultHidden && fs.existsSync(this.vault)) fs.chmodSync(this.vault, DIR_MODE)
    this.vaultHidden = false
  }

  // runUpdate runs the real `engram update` (local mode, `source` as working directory, ENGRAM_UPDATE_REEXEC=1 so
  // that the go install leg is skipped, 793b261) into this scenario's home. `claude` is put on PATH for the update's
  // own `claude --version`; nothing else is added to the update's environment.
  async runUpdate({ dryRun = false, source = REPO_ROOT, args = [] } = {}) {
    const claudeDir = path.dirname(this.context.claudePath)
    const PATH = `${this.env().PATH}${path.delimiter}${claudeDir}`
    this.updateOutput = await this.engram(['update', ...(dryRun ? ['--dry-run'] : []), ...args], {
      cwd: source,
      extraEnv: { ENGRAM_UPDATE_REEXEC: '1', PATH },
    })
    return this.updateOutput
  }

  // install: the System as `engram update` deploys it, and the tree it left, to compare later steps against.
  async install() {
    await this.runUpdate()
    this.treeBefore = this.snapshot()
  }

  // installPreSystem: an update of a source tree that has the skills and the guidance (with the shim) and no mod,
  // which is what an install made before the System existed holds (793e2i step 3).
  async installPreSystem() {
    const source = path.join(this.dir, 'pre-system-source')
    const agentInstructions = path.join(source, 'agent-instructions')
    fs.mkdirSync(agentInstructions, { recursive: true })
    fs.copyFileSync(path.join(REPO_ROOT, 'go.mod'), path.join(source, 'go.mod'))
    for (const name of ['skills', 'guidance']) {
      fs.cpSync(path.join(REPO_ROOT, 'agent-instructions', name), path.join(agentInstructions, name), {
        recursive: true,
      })
    }
    await this.runUpdate({ source, args: ['--with-guidance'] })
    this.treeBefore = this.snapshot()
  }

  snapshot() {
    return snapshotTree(this.home)
  }

  // linkMod puts a mod that is not the System where Claude Code loads it from: a link in the skills directory of this
  // home (793b251). Only the recorder mod of a peer's sender uses it; the System itself is deployed by `engram update`.
  linkMod(modDir, name) {
    const skills = path.join(this.home, '.claude', 'skills')
    fs.mkdirSync(skills, { recursive: true })
    fs.symlinkSync(modDir, path.join(skills, name))
  }

  // writeWorkspaceFile writes an input file the agent is told to read (a command, a prompt to schedule).
  writeWorkspaceFile(name, text) {
    fs.writeFileSync(path.join(this.workspace, name), text)
  }

  // startSession starts a session in this World; current: false leaves `this.session` alone (a peer's sender).
  startSession({ current = true, ...options } = {}) {
    const session = new Session({
      env: this.env({ withCredentials: true }),
      cwd: this.workspace,
      home: this.home,
      claudePath: this.context.claudePath,
      creds: this.context.creds,
      maxTurns: MAX_TURNS,
      ...options,
    })
    this.sessions.push(session)
    if (current) this.session = session
    return session
  }

  usage() {
    const turns = [...(this.usageSoFar ?? []), ...this.sessions.flatMap((session) => session.turns)]
    return {
      sessions: this.sessions.length + (this.discardedSessions ?? 0),
      turns: turns.length,
      costUsd: turns.reduce((sum, turn) => sum + turn.costUsd, 0),
      wallMs: turns.reduce((sum, turn) => sum + turn.wallMs, 0),
    }
  }

  async #discard() {
    this.discardedSessions = (this.discardedSessions ?? 0) + this.sessions.length
    await Promise.all(this.sessions.map((session) => session.close()))
    this.showVault()
    fs.rmSync(this.dir, { recursive: true, force: true })
  }

  async dispose() {
    await this.#discard()
  }
}
