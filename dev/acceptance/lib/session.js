// One Claude Code session driven through the Agent SDK with streaming input (793b276): turns are sent one at a
// time, a turn ends at the SDK result message, and a mod that failed to load fails the session at once.
import fs from 'node:fs'
import path from 'node:path'

import { query } from '@anthropic-ai/claude-agent-sdk'

import { turnCost } from './cost.js'
import { redact } from './creds.js'

const NOT_LOADED = /hooks module (\S+) not loaded:? ?(.*)/i

export class Session {
  constructor({ env, cwd, home, claudePath, creds, model = 'sonnet', plugins = [], resume, maxBudgetUsd = 5, debugFile }) {
    this.home = home
    this.debugFile = debugFile ?? path.join(home, 'claude-debug.log')
    this.creds = creds
    this.messages = []
    this.turns = []
    this.stderr = ''
    this.sessionId = null
    this.init = null
    this.failure = null
    this.costTotal = 0
    this.waiting = null
    this.inputs = []
    this.inputWaiter = null
    this.closed = false
    this.query = query({
      prompt: this.#inputStream(),
      options: {
        cwd,
        env,
        model,
        plugins,
        resume,
        maxBudgetUsd,
        pathToClaudeCodeExecutable: claudePath,
        permissionMode: 'default',
        permissionPrompts: 'none',
        extraArgs: { 'debug-file': this.debugFile },
        stderr: (data) => this.#onStderr(data),
      },
    })
    this.reader = this.#read()
  }

  // send submits one user turn and resolves with its summary when the SDK result message arrives.
  send(text) {
    if (this.failure) return Promise.reject(this.failure)
    return new Promise((resolve, reject) => {
      this.waiting = { resolve, reject, startedAt: Date.now() }
      this.#push({ type: 'user', message: { role: 'user', content: text }, parent_tool_use_id: null })
    })
  }

  // close ends the input stream and waits for the SDK to finish.
  async close() {
    this.closed = true
    this.#push(null)
    await this.reader.catch(() => undefined)
  }

  transcriptPath() {
    const projects = path.join(this.home, '.claude', 'projects')
    if (!this.sessionId || !fs.existsSync(projects)) return null
    for (const dir of fs.readdirSync(projects)) {
      const file = path.join(projects, dir, `${this.sessionId}.jsonl`)
      if (fs.existsSync(file)) return file
    }
    return null
  }

  async *#inputStream() {
    for (;;) {
      while (this.inputs.length === 0) {
        await new Promise((resolve) => { this.inputWaiter = resolve })
      }
      const next = this.inputs.shift()
      if (next === null) return
      yield { ...next, session_id: this.sessionId ?? '' }
    }
  }

  #push(item) {
    this.inputs.push(item)
    this.inputWaiter?.()
  }

  #onStderr(data) {
    this.stderr += data
    this.#checkLoad()
  }

  // #checkLoad fails the session when a hooks module was refused. In a stream the line goes to the debug log, not to
  // stderr (SP2), so both are read.
  #checkLoad() {
    if (this.failure) return
    let log = ''
    try {
      log = fs.readFileSync(this.debugFile, 'utf8')
    } catch {
      log = ''
    }
    const match = NOT_LOADED.exec(`${this.stderr}\n${log}`)
    if (match) this.#fail(new Error(`hooks module ${match[1]} not loaded (fail fast): ${redact(match[2], this.creds)}`))
  }

  #fail(error) {
    this.failure = error
    this.waiting?.reject(error)
    this.waiting = null
    this.#push(null)
  }

  async #read() {
    try {
      for await (const message of this.query) {
        this.messages.push(message)
        if (message.type === 'system' && message.subtype === 'init') {
          this.sessionId = message.session_id
          this.init = message
          this.#checkLoad()
        } else if (message.type === 'result' && this.waiting) {
          this.#checkLoad()
          if (this.failure) return
          const turn = summarize(message, this.waiting.startedAt, turnCost(this.costTotal, message.total_cost_usd))
          this.costTotal = typeof message.total_cost_usd === 'number' ? message.total_cost_usd : this.costTotal
          this.turns.push(turn)
          const { resolve } = this.waiting
          this.waiting = null
          resolve(turn)
        }
      }
    } catch (error) {
      if (!this.closed) this.#fail(error)
    }
    if (this.waiting) this.#fail(new Error(`the session ended before the turn completed: ${redact(this.stderr.trim(), this.creds)}`))
  }
}

function summarize(result, startedAt, costUsd) {
  return {
    text: result.result ?? '',
    isError: Boolean(result.is_error),
    subtype: result.subtype,
    costUsd,
    durationMs: result.duration_ms ?? Date.now() - startedAt,
    wallMs: Date.now() - startedAt,
    usage: result.usage,
    modelUsage: result.modelUsage,
  }
}
