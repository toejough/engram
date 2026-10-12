// The per-scenario World (793e2b): its own HOME, workspace, vault and TMPDIR, an explicit environment, notes seeded
// through the real write path, and the session. Nothing under the real ~/.claude, ~/.pi or the real vault is touched.
import { execFile } from 'node:child_process'
import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'
import { promisify } from 'node:util'

import { REPO_ROOT } from './extract.js'
import { baseEnv } from './run-context.js'
import { Session } from './session.js'
import { snapshotTree } from './tree.js'

const run = promisify(execFile)
const ALLOWED_TOOLS = ['Bash(engram:*)', 'Bash(echo:*)', 'Bash(sleep:*)', 'Read', 'Glob', 'Grep', 'Agent', 'Skill', 'CronCreate']
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
    this.extraEnv = {}
    this.sessions = []
    this.session = null
    this.installed = false
    for (const dir of [this.home, path.join(this.home, 'tmp'), this.workspace, path.join(this.home, '.claude')]) {
      fs.mkdirSync(dir, { recursive: true })
    }
    this.vault = path.join(this.home, '.local', 'share', 'engram', 'vault')
    // the sweep spec of the workspace: a scenario lives under a throwaway root, which `--auto` skips by default (SP15)
    fs.mkdirSync(path.join(this.workspace, '.engram'), { recursive: true })
    fs.writeFileSync(
      path.join(this.workspace, '.engram', 'sweep.json'),
      `${JSON.stringify({ non_persistent_prefixes: [], non_persistent_path_prefixes: [] })}\n`,
    )
    // the settings the World starts from: the update must add none (793a5j)
    this.baselineSettings = { permissions: { allow: ALLOWED_TOOLS } }
    fs.writeFileSync(path.join(this.home, '.claude', 'settings.json'), `${JSON.stringify(this.baselineSettings, null, 2)}\n`)
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
    const env = { ...baseEnv({ home: this.home, cacheDir: this.context.cacheDir, binDir: this.context.binDir }), ...this.extraEnv }
    return withCredentials ? { ...env, ...this.context.creds.env } : env
  }

  async engram(args, { cwd = this.workspace, extraEnv = {} } = {}) {
    const { stdout } = await run(this.context.engramPath, args, { cwd, env: { ...this.env(), ...extraEnv } })
    return stdout
  }

  // seedNote writes a fact note through `engram learn` with a unique sentinel in its body; the sentinel is what a
  // delivery or a retrieval is recognised by.
  async seedNote(label, situation, says) {
    const sentinel = `SENTINEL-${label}-${crypto.randomBytes(3).toString('hex')}`
    const object = says ? `${says} (${sentinel})` : sentinel
    await this.engram([
      'learn', 'fact', '--slug', `note-${label.toLowerCase()}`, '--position', 'top', '--source', 'acceptance harness',
      '--situation', situation, '--subject', situation, '--predicate', 'is recorded as', '--object', object,
    ])
    this.notes.set(label, { label, situation, sentinel })
    return this.notes.get(label)
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
    this.installed = true
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
      fs.cpSync(path.join(REPO_ROOT, 'agent-instructions', name), path.join(agentInstructions, name), { recursive: true })
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
    return { sessions: this.sessions.length + (this.discardedSessions ?? 0), turns: turns.length, costUsd: turns.reduce((sum, turn) => sum + turn.costUsd, 0), wallMs: turns.reduce((sum, turn) => sum + turn.wallMs, 0) }
  }

  async #discard() {
    this.discardedSessions = (this.discardedSessions ?? 0) + this.sessions.length
    await Promise.all(this.sessions.map((session) => session.close()))
    fs.rmSync(this.dir, { recursive: true, force: true })
  }

  async dispose() {
    await this.#discard()
  }
}
